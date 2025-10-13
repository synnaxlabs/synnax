// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <set>

#include "driver/modbus/channels.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/util/util.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/sample_clock.h"

namespace modbus {
/// @brief interface for reading from different types of Modbus registers/bits.
struct Reader {
    virtual ~Reader() = default;

    /// @brief read from the device, populate the frame with the response data, and
    /// increment offset by the number of series modified.
    /// @param dev the device to read from.
    /// @param fr the frame to populate with the response data.
    /// @param offset the series offset into the frame to start writing at. This is
    /// incremented by the number of series modified.
    /// @returns xerrors::NIL if successful, any other error otherwise.
    virtual xerrors::Error read(
        const std::shared_ptr<device::Device> &dev,
        synnax::Frame &fr,
        size_t &offset
    ) = 0;

    /// @brief return the list of Synnax channels that this reader is responsible for.
    [[nodiscard]] virtual std::vector<synnax::Channel> sy_channels() const = 0;
};

/// @brief base reader class for all reader types.
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

/// @brief reads from holding and input registers.
class RegisterReader final : public BaseReader<channel::InputRegister> {
    /// @brief the register type to read from. either HoldingRegister or InputRegister.
    device::RegisterType register_type;
    /// @brief the buffer to read into.
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

        for (const auto &ch: channels) {
            int offset = ch.address - start_addr;
            auto [value, err] = util::parse_register_value(
                this->buffer.data() + offset,
                ch.value_type,
                ch.swap_bytes,
                ch.swap_words
            );
            if (err) return err;
            fr.series->at(frame_offset++).write(value);
        }
        return xerrors::NIL;
    }
};

/// @brief reads from coils and discrete inputs.
class BitReader final : public BaseReader<channel::InputDiscrete> {
    /// @brief the bit type to read from. either Coil or DiscreteInput.
    device::BitType bit_type;
    /// @brief the buffer to read into.
    std::vector<uint8_t> buffer;

public:
    explicit BitReader(
        const device::BitType bit_type,
        const std::vector<channel::InputDiscrete> &channels
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
                .write(this->buffer[channel.address - start_addr]);
        return xerrors::NIL;
    }
};

/// @brief configuration for a modbus read task.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// @brief the total number of data channels in the task.
    size_t data_channel_count;
    /// @brief the key of the device to read from.
    std::string device_key;
    /// @brief the indexes of all data channels in the task.
    std::set<synnax::ChannelKey> indexes;
    /// @brief the list of readers to use for reading data from the device.
    std::vector<std::unique_ptr<Reader>> readers;
    /// @brief the connection configuration for the device.
    device::ConnectionConfig conn;
    /// @brief the number of samples per channel to read on each read() call.
    std::size_t samples_per_chan;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        BaseReadTaskConfig(std::move(other)),
        data_channel_count(other.data_channel_count),
        device_key(std::move(other.device_key)),
        indexes(std::move(other.indexes)),
        readers(std::move(other.readers)),
        conn(std::move(other.conn)),
        samples_per_chan(other.samples_per_chan) {}

    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ):
        BaseReadTaskConfig(cfg),
        data_channel_count(0),
        device_key(cfg.required<std::string>("device")),
        samples_per_chan(sample_rate / stream_rate) {
        std::vector<channel::InputRegister> holding_registers;
        std::vector<channel::InputRegister> input_registers;
        std::vector<channel::InputDiscrete> coils;
        std::vector<channel::InputDiscrete> discrete_inputs;

        auto [dev, dev_err] = client->hardware.retrieve_device(this->device_key);
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
            this->data_channel_count++;
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
            readers.push_back(
                std::make_unique<RegisterReader>(
                    device::HoldingRegister,
                    std::move(holding_registers)
                )
            );
        if (!input_registers.empty())
            readers.push_back(
                std::make_unique<RegisterReader>(
                    device::InputRegister,
                    std::move(input_registers)
                )
            );
        if (!coils.empty())
            readers.push_back(
                std::make_unique<BitReader>(device::Coil, std::move(coils))
            );
        if (!discrete_inputs.empty())
            readers.push_back(
                std::make_unique<BitReader>(
                    device::DiscreteInput,
                    std::move(discrete_inputs)
                )
            );
        for (const auto &ch: synnax_channels)
            if (ch.index != 0) this->indexes.insert(ch.index);
    }

    /// @brief parses the configuration for the task from its JSON representation,
    /// using the provided Synnax client to retrieve the device and channel information.
    /// @param client the Synnax client to use to retrieve the device and channel
    /// information.
    /// @param task the task to parse.
    /// @returns a pair containing the parsed configuration and any error that occurred
    /// during parsing.
    static std::pair<ReadTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        return {ReadTaskConfig(client, parser), parser.error()};
    }

    /// @brief all synnax channels that the task will write to, excluding indexes.
    [[nodiscard]] std::vector<synnax::Channel> data_channels() const {
        std::vector<synnax::Channel> result;
        result.reserve(this->data_channel_count);
        for (const auto &op: this->readers)
            for (const auto &ch: op->sy_channels())
                result.push_back(ch);
        return result;
    }

    /// @brief configuration for opening a synnax writer for the task.
    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        const auto data_channels = this->data_channels();
        keys.reserve(data_channels.size() + this->indexes.size());
        for (const auto &ch: data_channels)
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

/// @brief implements common::Source to read from a modbus device.
class ReadTaskSource final : public common::Source {
    /// @brief the configuration for the task.
    const ReadTaskConfig config;
    /// @brief the device to read from.
    std::shared_ptr<device::Device> dev;
    /// @brief the sample clock to regulate the read rate.
    common::SoftwareTimedSampleClock sample_clock;

public:
    explicit ReadTaskSource(
        const std::shared_ptr<device::Device> &dev,
        ReadTaskConfig cfg
    ):
        config(std::move(cfg)), dev(dev), sample_clock(this->config.sample_rate) {}

    common::ReadResult read(breaker::Breaker &breaker, synnax::Frame &fr) override {
        common::ReadResult res;
        const auto n_channels = this->config.data_channel_count;
        const auto n_samples = this->config.samples_per_chan;
        auto total_channel_count = n_channels + this->config.indexes.size();
        if (fr.size() != total_channel_count) {
            fr.reserve(total_channel_count);
            for (const auto &ch: this->config.data_channels())
                fr.emplace(ch.key, telem::Series(ch.data_type, n_samples));
            for (const auto &idx: this->config.indexes)
                fr.emplace(idx, telem::Series(telem::TIMESTAMP_T, n_samples));
        }
        for (auto &ser: *fr.series)
            ser.clear();
        for (size_t i = 0; i < n_samples; ++i) {
            size_t offset = 0;
            const auto start = this->sample_clock.wait(breaker);
            for (const auto &op: this->config.readers)
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
        return this->config.data_channels();
    }
};
}
