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
#include "driver/modbus/device/device.h"
#include "driver/modbus/channels.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/sample_clock.h"
#include "driver/modbus/util/util.h"

namespace modbus {
/// @brief Reader interface that abstracts reading from different types of Modbus
/// registers/bits.
struct Reader {
    virtual ~Reader() = default;

    /// @brief read from the device and populate the frame with the response data.
    /// @returns xerrors:NIL if successful, any other error otherwise.
    virtual xerrors::Error read(
        const std::shared_ptr<device::Device> &dev,
        synnax::Frame &fr
    ) = 0;

    virtual std::vector<synnax::Channel> sy_channels() = 0;
};

class RegisterReader final : public Reader {
    std::vector<channel::InputRegister> channels;
    bool read_holding;
    std::vector<uint16_t> buffer;

public:
    explicit RegisterReader(
        const std::vector<channel::InputRegister> &chs,
        const bool read_holding
    ): channels(chs), read_holding(read_holding) {
    }

    std::vector<synnax::Channel> sy_channels() override {
        std::vector<synnax::Channel> result;
        result.reserve(channels.size());
        for (const auto &channel: channels) result.push_back(channel.ch);
        return result;
    }

    xerrors::Error
    read(const std::shared_ptr<device::Device> &dev, synnax::Frame &fr) override {
        if (channels.empty()) return xerrors::NIL;

        const int start_addr = this->channels[0].address;

        if (this->read_holding) {
            if (const auto err = dev->read_registers(
                start_addr,
                this->buffer.size(),
                this->buffer.data()
            ))
                return err;
        } else if (const auto err = dev->read_input_registers(
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
            fr.emplace(channel.synnax_key, telem::Series(value));
        }
        return xerrors::NIL;
    }
};

class BitReader final : public Reader {
    std::vector<channel::InputBit> channels;
    bool read_coils;
    std::vector<uint8_t> buffer;

public:
    explicit BitReader(std::vector<channel::InputBit> chs, const bool read_coils):
        channels(std::move(chs)), read_coils(read_coils) {
    }

    std::vector<synnax::Channel> sy_channels() override {
        std::vector<synnax::Channel> result;
        result.reserve(channels.size());
        for (const auto &channel: channels) result.push_back(channel.ch);
        return result;
    }

    xerrors::Error
    read(const std::shared_ptr<device::Device> &dev, synnax::Frame &fr) override {
        if (channels.empty()) return xerrors::NIL;

        const int start_addr = channels.front().address;

        if (this->read_coils) {
            if (const auto err = dev->read_bits(
                start_addr,
                this->buffer.size(),
                this->buffer.data()
            ))
                return err;
        } else if (const auto err = dev->read_input_bits(
            start_addr,
            this->buffer.size(),
            this->buffer.data()
        ))
            return err;

        for (const auto &channel: channels)
            fr.emplace(
                channel.synnax_key,
                telem::Series(this->buffer[channel.address - start_addr])
            );
        return xerrors::NIL;
    }
};

struct ReadTaskConfig {
    const bool data_saving;
    const telem::Rate sample_rate;
    const telem::Rate stream_rate;
    size_t channel_count;
    std::set<synnax::ChannelKey> indexes;
    std::vector<std::unique_ptr<Reader> > ops;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        data_saving(other.data_saving),
        sample_rate(other.sample_rate),
        stream_rate(other.stream_rate),
        channel_count(other.channel_count),
        indexes(std::move(other.indexes)),
        ops(std::move(other.ops)) {
    }

    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ): data_saving(cfg.optional<bool>("data_saving", false)),
       sample_rate(telem::Rate(cfg.required<float>("sample_rate"))),
       stream_rate(telem::Rate(cfg.required<float>("stream_rate"))),
       channel_count(0) {
        std::vector<channel::InputRegister> holding_registers;
        std::vector<channel::InputRegister> input_registers;
        std::vector<channel::InputBit> coils;
        std::vector<channel::InputBit> discrete_inputs;

        cfg.iter("channels", [&, this](xjson::Parser &ch) {
            const auto type = ch.required<std::string>("type");

            if (type == "holding_register_input")
                holding_registers.emplace_back(ch);
            else if (type == "input_register_input")
                input_registers.emplace_back(ch);
            else if (type == "coil_input")
                coils.emplace_back(ch);
            else if (type == "discrete_input_input")
                discrete_inputs.emplace_back(ch);
            this->channel_count++;
        });

        auto sort_by_address = [](const auto &a, const auto &b) {
            return a.address < b.address;
        };

        std::sort(holding_registers.begin(), holding_registers.end(), sort_by_address);
        std::sort(input_registers.begin(), input_registers.end(), sort_by_address);
        std::sort(coils.begin(), coils.end(), sort_by_address);
        std::sort(discrete_inputs.begin(), discrete_inputs.end(), sort_by_address);

        std::vector<synnax::ChannelKey> keys;
        for (const auto &ch: holding_registers) keys.push_back(ch.synnax_key);
        for (const auto &ch: input_registers) keys.push_back(ch.synnax_key);
        for (const auto &ch: coils) keys.push_back(ch.synnax_key);
        for (const auto &ch: discrete_inputs) keys.push_back(ch.synnax_key);
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
            coils[i].ch = synnax_channels[
                i + holding_registers.size() + input_registers.size()];
        for (auto i = 0; i < discrete_inputs.size(); i++)
            discrete_inputs[i].ch = synnax_channels[
                i + holding_registers.size() + input_registers.size() + coils.size()];

        if (!holding_registers.empty())
            ops.push_back(std::make_unique<RegisterReader>(
                std::move(holding_registers),
                true
            ));
        if (!input_registers.empty())
            ops.push_back(std::make_unique<RegisterReader>(
                std::move(input_registers),
                false
            ));
        if (!coils.empty())
            ops.push_back(std::make_unique<BitReader>(
                std::move(coils),
                true
            ));
        if (!discrete_inputs.empty())
            ops.push_back(std::make_unique<BitReader>(
                std::move(discrete_inputs),
                false
            ));
        for (const auto &ch: synnax_channels) this->indexes.insert(ch.key);
    }

    static std::pair<ReadTaskConfig, xerrors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        return {ReadTaskConfig(client, parser), parser.error()};
    }


    [[nodiscard]] std::vector<synnax::Channel> channels() const {
        std::vector<synnax::Channel> result;
        result.reserve(this->channel_count);
        for (const auto &op: this->ops)
            for (const auto &ch: op->sy_channels()) result.push_back(ch);
        return result;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        const auto channels = this->channels();
        keys.reserve(channels.size() + this->indexes.size());
        for (const auto &ch: this->channels()) keys.push_back(ch.key);
        for (const auto &idx: this->indexes) keys.push_back(idx);
        return synnax::WriterConfig{
            .start = telem::TimeStamp::now(),
            .channels = keys,
            .mode = synnax::data_saving_writer_mode(this->data_saving),
        };
    }
};

class ReadTaskSource final : common::Source {
    const ReadTaskConfig config;
    std::shared_ptr<device::Device> dev;
public:
    explicit ReadTaskSource(
        const std::shared_ptr<device::Device> &dev, ReadTaskConfig cfg
    ): config(std::move(cfg)), dev(dev) {
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        synnax::Frame fr;
        fr.reserve(this->config.channel_count + this->config.indexes.size());
        for (const auto &op: this->config.ops)
            if (const auto err = op->read(this->dev, fr))
                return {std::move(fr), err};
        const auto start = telem::TimeStamp::now();
        common::generate_index_data(fr, this->config.indexes, start, start, 1);
        return {std::move(fr), xerrors::NIL};
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return this->config.writer_config();
    }

    [[nodiscard]] std::vector<synnax::Channel> channels() const override {
        return this->config.channels();
    }
};
}
