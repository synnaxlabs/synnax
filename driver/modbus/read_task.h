// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <set>

/// internal
#include "driver/modbus/channels.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/util/util.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/sample_clock.h"

namespace modbus {
/// @brief Reader interface that abstracts reading from different types of Modbus
/// registers/bits.
struct Reader {
    virtual ~Reader() = default;

    /// @brief read from the device and populate the frame with the response data.
    /// @returns xerrors::NIL if successful, any other error otherwise.
    virtual xerrors::Error read(
        const std::shared_ptr<device::Device> &dev,
        synnax::Frame &fr,
        size_t &offset
    ) = 0;

    [[nodiscard]] virtual std::vector<synnax::Channel> sy_channels() const = 0;
};

template<typename Channel>
struct BaseReader : Reader {
    std::vector<Channel> channels;

    explicit BaseReader(const std::vector<Channel> &channels): channels(channels) {}

    [[nodiscard]] std::vector<synnax::Channel> sy_channels() const override {
        std::vector<synnax::Channel> result;
        result.reserve(channels.size());
        for (const auto &channel: channels)
            result.push_back(channel.ch);
        return result;
    }
};

class RegisterReader final : public BaseReader<channel::InputRegister> {
    device::RegisterType register_type;
    std::vector<uint16_t> buffer;

public:
    explicit RegisterReader(
        const device::RegisterType register_type,
        const std::vector<channel::InputRegister> &chs
    ):
        BaseReader(chs), register_type(register_type) {
        auto first_addr = this->channels.front().address;
        auto last_addr = this->channels.back().address;
        last_addr += this->channels.back().value_type.density() / 2;
        this->buffer.resize(last_addr - first_addr);
    }

    xerrors::Error read(
        const std::shared_ptr<device::Device> &dev,
        synnax::Frame &fr,
        size_t &frame_offset
    ) override {
        if (channels.empty()) return xerrors::NIL;

        const int start_addr = this->channels[0].address;

        if (const auto err = dev->read_registers(
                this->register_type,
                start_addr,
                this->buffer.size(),
                this->buffer.data()
            ))
            return err;

        for (const auto &channel: channels) {
            int offset = channel.address - start_addr;
            auto [value, err] = util::parse_register(
                this->buffer.data(),
                offset,
                channel.value_type,
                channel.swap_bytes,
                channel.swap_words
            );
            if (err) return err;
            fr.series->at(frame_offset++).write(value);
        }
        return xerrors::NIL;
    }
};

class BitReader final : public BaseReader<channel::InputBit> {
    device::BitType bit_type;
    std::vector<uint8_t> buffer;

public:
    explicit BitReader(
        const device::BitType bit_type,
        const std::vector<channel::InputBit> &channels
    ):
        BaseReader(channels),
        bit_type(bit_type),
        buffer(this->channels.back().address - this->channels.front().address + 1) {}

    xerrors::Error read(
        const std::shared_ptr<device::Device> &dev,
        synnax::Frame &fr,
        size_t &frame_offset
    ) override {
        if (channels.empty()) return xerrors::NIL;

        const int start_addr = channels.front().address;

        if (const auto err = dev->read_bits(
                this->bit_type,
                start_addr,
                this->buffer.size(),
                this->buffer.data()
            ))
            return err;

        for (const auto &channel: channels)
            fr.series->at(frame_offset++)
                .write(this->buffer.data()[channel.address - start_addr]);
        return xerrors::NIL;
    }
};

struct ReadTaskConfig : common::BaseReadTaskConfig {
    size_t channel_count;
    std::set<synnax::ChannelKey> indexes;
    std::vector<std::unique_ptr<Reader>> ops;
    device::ConnectionConfig conn;
    std::string dev;
    std::size_t samples_per_chan;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        BaseReadTaskConfig(std::move(other)),
        channel_count(other.channel_count),
        indexes(std::move(other.indexes)),
        ops(std::move(other.ops)),
        conn(std::move(other.conn)),
        dev(std::move(other.dev)),
        samples_per_chan(other.samples_per_chan) {}

    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ):
        BaseReadTaskConfig(cfg),
        channel_count(0),
        dev(cfg.required<std::string>("device")),
        samples_per_chan(sample_rate / stream_rate) {
        std::vector<channel::InputRegister> holding_registers;
        std::vector<channel::InputRegister> input_registers;
        std::vector<channel::InputBit> coils;
        std::vector<channel::InputBit> discrete_inputs;

        auto [dev, dev_err] = client->hardware.retrieve_device(this->dev);
        if (dev_err) {
            cfg.field_err("device", dev_err.message());
            return;
        }

        auto conn_parser = xjson::Parser(dev.properties);
        this->conn = device::ConnectionConfig(conn_parser.child("connection"));
        if (conn_parser.error()) {
            cfg.field_err("device", conn_parser.error().message());
            return;
        }

        cfg.iter("channels", [&, this](xjson::Parser &ch) {
            const auto type = ch.required<std::string>("type");
            if (type == "holding_register_input")
                holding_registers.emplace_back(ch);
            else if (type == "register_input")
                input_registers.emplace_back(ch);
            else if (type == "coil_input")
                coils.emplace_back(ch);
            else if (type == "discrete_input")
                discrete_inputs.emplace_back(ch);
            else {
                cfg.field_err("channels", "invalid channel type: " + type);
                return;
            }
            this->channel_count++;
        });

        channel::sort_by_address(holding_registers);
        channel::sort_by_address(input_registers);
        channel::sort_by_address(coils);
        channel::sort_by_address(discrete_inputs);

        std::vector<synnax::ChannelKey> keys;
        for (const auto &ch: holding_registers)
            keys.push_back(ch.synnax_key);
        for (const auto &ch: input_registers)
            keys.push_back(ch.synnax_key);
        for (const auto &ch: coils)
            keys.push_back(ch.synnax_key);
        for (const auto &ch: discrete_inputs)
            keys.push_back(ch.synnax_key);
        auto [synnax_channels, err] = client->channels.retrieve(keys);
        if (err) {
            cfg.field_err("channels", err.message());
            return;
        }

        for (auto i = 0; i < holding_registers.size(); i++)
            holding_registers[i].ch = synnax_channels[i];
        for (auto i = 0; i < input_registers.size(); i++)
            input_registers[i].ch = synnax_channels[i + holding_registers.size()];
        for (auto i = 0; i < coils.size(); i++)
            coils[i].ch = synnax_channels
                [i + holding_registers.size() + input_registers.size()];
        for (auto i = 0; i < discrete_inputs.size(); i++)
            discrete_inputs[i].ch = synnax_channels
                [i + holding_registers.size() + input_registers.size() + coils.size()];

        if (!holding_registers.empty())
            ops.push_back(std::make_unique<RegisterReader>(
                device::HoldingRegister,
                std::move(holding_registers)
            ));
        if (!input_registers.empty())
            ops.push_back(std::make_unique<RegisterReader>(
                device::InputRegister,
                std::move(input_registers)
            ));
        if (!coils.empty())
            ops.push_back(std::make_unique<BitReader>(device::Coil, std::move(coils)));
        if (!discrete_inputs.empty())
            ops.push_back(std::make_unique<BitReader>(
                device::DiscreteInput,
                std::move(discrete_inputs)
            ));
        for (const auto &ch: synnax_channels)
            if (ch.index != 0) this->indexes.insert(ch.index);
    }

    static std::pair<ReadTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        return {ReadTaskConfig(client, parser), parser.error()};
    }


    [[nodiscard]] std::vector<synnax::Channel> channels() const {
        std::vector<synnax::Channel> result;
        result.reserve(this->channel_count);
        for (const auto &op: this->ops)
            for (const auto &ch: op->sy_channels())
                result.push_back(ch);
        return result;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        const auto channels = this->channels();
        keys.reserve(channels.size() + this->indexes.size());
        for (const auto &ch: this->channels())
            keys.push_back(ch.key);
        for (const auto &idx: this->indexes)
            keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .start = telem::TimeStamp::now(),
            .mode = synnax::data_saving_writer_mode(this->data_saving),
        };
    }
};

class ReadTaskSource final : public common::Source {
    const ReadTaskConfig config;
    std::shared_ptr<device::Device> dev;
    common::SoftwareTimedSampleClock sample_clock;

public:
    explicit ReadTaskSource(
        const std::shared_ptr<device::Device> &dev,
        ReadTaskConfig cfg
    ):
        config(std::move(cfg)), dev(dev), sample_clock(this->config.sample_rate) {}

    common::ReadResult read(breaker::Breaker &breaker, synnax::Frame &fr) override {
        common::ReadResult res;
        const auto n_channels = this->config.channel_count;
        const auto n_samples = this->config.samples_per_chan;
        auto total_channel_count = n_channels + this->config.indexes.size();
        if (fr.size() != total_channel_count) {
            fr.reserve(total_channel_count);
            for (const auto &ch: this->config.channels())
                fr.emplace(ch.key, telem::Series(ch.data_type, n_samples));
            for (const auto &idx: this->config.indexes)
                fr.emplace(idx, telem::Series(telem::TIMESTAMP_T, n_samples));
        }
        for (auto &ser: *fr.series)
            ser.clear();
        for (size_t i = 0; i < n_samples; ++i) {
            size_t offset = 0;
            const auto start = this->sample_clock.wait(breaker);
            for (const auto &op: this->config.ops)
                if (res.error = op->read(this->dev, fr, offset); res.error) return res;
            const auto end = this->sample_clock.end();
            for (size_t j = offset; j < this->config.indexes.size() + offset; ++j)
                fr.series->at(j).write(telem::TimeStamp(end - (end - start) / 2));
        }
        return res;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return this->config.writer_config();
    }

    [[nodiscard]] std::vector<synnax::Channel> channels() const override {
        return this->config.channels();
    }
};
}
