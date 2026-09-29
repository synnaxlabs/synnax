// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <set>
#include <variant>

#include "client/cpp/modbus/json.gen.h"

#include "driver/codec/plan.h"
#include "driver/common/read_task.h"
#include "driver/common/sample_clock.h"
#include "driver/modbus/channels.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/registers/registers.h"

namespace driver::modbus {
/// @brief interface for reading from different types of Modbus registers/bits.
struct Reader {
    virtual ~Reader() = default;

    /// @brief read from the device, populate the frame with the response data, and
    /// increment offset by the number of series modified.
    /// @param dev the device to read from.
    /// @param fr the frame to populate with the response data.
    /// @param offset the series offset into the frame to start writing at. This is
    /// incremented by the number of series modified.
    /// @returns x::errors::NIL if successful, any other error otherwise.
    virtual x::errors::Error read(
        const std::shared_ptr<device::Device> &dev,
        x::telem::Frame &fr,
        size_t &offset
    ) = 0;

    /// @brief return the list of Synnax channels that this reader is responsible for.
    [[nodiscard]] virtual std::vector<synnax::channel::Channel> sy_channels() const = 0;
};

/// @brief base reader class for all reader types.
template<typename Channel>
struct BaseReader : Reader {
    std::vector<Channel> channels;
    /// @brief decodes each channel's value from the payload.
    codec::Plan plan;
    /// @brief holds the decoded values, reused across reads.
    codec::Values values;
    /// @brief the bytes read from the device.
    std::vector<uint8_t> payload;

    BaseReader(const std::vector<Channel> &channels, codec::Plan plan):
        channels(channels),
        plan(std::move(plan)),
        values(this->plan.values()),
        payload(this->plan.length()) {}

    [[nodiscard]] std::vector<synnax::channel::Channel> sy_channels() const override {
        std::vector<synnax::channel::Channel> result;
        result.reserve(channels.size());
        for (const auto &channel: channels)
            result.push_back(channel.ch);
        return result;
    }

protected:
    /// @brief decodes the payload and appends each channel's value to its series.
    x::errors::Error decode(x::telem::Frame &fr, size_t &frame_offset) {
        if (const auto err = this->plan.decode(this->payload, this->values)) return err;
        for (size_t i = 0; i < this->channels.size(); i++)
            this->values.write(i, fr.series->at(frame_offset++));
        return x::errors::NIL;
    }
};

/// @brief reads from holding and input registers.
class RegisterReader final : public BaseReader<channel::InputRegister> {
    /// @brief the register type to read from. either HoldingRegister or InputRegister.
    device::RegisterType register_type;
    /// @brief the buffer to read into.
    std::vector<uint16_t> buffer;

public:
    /// @param plan from channel::compile(chs).
    RegisterReader(
        const device::RegisterType register_type,
        const std::vector<channel::InputRegister> &chs,
        codec::Plan plan
    ):
        BaseReader(chs, std::move(plan)),
        register_type(register_type),
        buffer((this->payload.size() + 1) / 2) {
        this->payload.resize(this->buffer.size() * 2);
    }

    x::errors::Error read(
        const std::shared_ptr<device::Device> &dev,
        x::telem::Frame &fr,
        size_t &frame_offset
    ) override {
        if (channels.empty()) return x::errors::NIL;
        if (const auto err = dev->read_registers(
                this->register_type,
                this->channels.front().address,
                this->buffer.size(),
                this->buffer.data()
            ))
            return err;
        registers::to_bytes(this->buffer, this->payload);
        return this->decode(fr, frame_offset);
    }
};

/// @brief reads from coils and discrete inputs.
class BitReader final : public BaseReader<channel::InputDiscrete> {
    /// @brief the bit type to read from. either Coil or DiscreteInput.
    device::BitType bit_type;

public:
    /// @param plan from channel::compile(channels).
    BitReader(
        const device::BitType bit_type,
        const std::vector<channel::InputDiscrete> &channels,
        codec::Plan plan
    ):
        BaseReader(channels, std::move(plan)), bit_type(bit_type) {}

    x::errors::Error read(
        const std::shared_ptr<device::Device> &dev,
        x::telem::Frame &fr,
        size_t &frame_offset
    ) override {
        if (channels.empty()) return x::errors::NIL;
        if (const auto err = dev->read_bits(
                this->bit_type,
                this->channels.front().address,
                this->payload.size(),
                this->payload.data()
            ))
            return err;
        return this->decode(fr, frame_offset);
    }
};

/// @brief configuration for a modbus read task.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// @brief the total number of data channels in the task.
    size_t data_channel_count;
    /// @brief the key of the device to read from.
    std::string device_key;
    /// @brief the indexes of all data channels in the task.
    /// Dynamically populated by querying the core.
    std::set<synnax::channel::Key> indexes;
    /// @brief the list of readers to use for reading data from the device.
    std::vector<std::unique_ptr<Reader>> readers;
    /// @brief the connection configuration for the device.
    /// Dynamically populated from device properties.
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
        x::json::Parser &cfg
    ):
        BaseReadTaskConfig(cfg),
        data_channel_count(0),
        device_key(cfg.field<std::string>("device")),
        samples_per_chan(sample_rate / stream_rate) {
        std::vector<channel::InputRegister> holding_registers;
        std::vector<channel::InputRegister> input_registers;
        std::vector<channel::InputDiscrete> coils;
        std::vector<channel::InputDiscrete> discrete_inputs;

        auto [dev, dev_err] = client->devices.retrieve(this->device_key);
        if (dev_err) {
            cfg.field_err("device", dev_err.message());
            return;
        }

        auto conn_parser = x::json::Parser(dev.properties);
        this->conn = device::ConnectionConfig(conn_parser.child("connection"));
        if (conn_parser.error()) {
            cfg.field_err("device", conn_parser.error().message());
            return;
        }

        cfg.iter("channels", [&, this](x::json::Parser &ch) {
            const auto parsed = ::synnax::modbus::parse_read_channel(ch);
            const auto &base = std::visit(
                [](const auto &c) -> const ::synnax::modbus::BaseReadChannel & {
                    return c;
                },
                parsed
            );
            if (base.disabled) return;
            if (base.channel == 0)
                return ch.field_err("channel", "channel must be specified");
            if (const auto *c = std::get_if<
                    ::synnax::modbus::HoldingRegisterReadChannel>(&parsed))
                holding_registers.emplace_back(*c);
            else if (
                const auto *c = std::get_if<::synnax::modbus::InputRegisterReadChannel>(
                    &parsed
                )
            )
                input_registers.emplace_back(*c);
            else if (
                const auto *c = std::get_if<::synnax::modbus::CoilReadChannel>(&parsed)
            )
                coils.emplace_back(*c);
            else
                discrete_inputs.emplace_back(
                    std::get<::synnax::modbus::DiscreteInputReadChannel>(parsed)
                );
            this->data_channel_count++;
        });

        channel::sort_by_address(holding_registers);
        channel::sort_by_address(input_registers);
        channel::sort_by_address(coils);
        channel::sort_by_address(discrete_inputs);

        std::vector<synnax::channel::Key> keys;
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

        for (size_t i = 0; i < holding_registers.size(); i++)
            holding_registers[i].ch = synnax_channels[i];
        for (size_t i = 0; i < input_registers.size(); i++)
            input_registers[i].ch = synnax_channels[i + holding_registers.size()];
        for (size_t i = 0; i < coils.size(); i++)
            coils[i].ch = synnax_channels
                [i + holding_registers.size() + input_registers.size()];
        for (size_t i = 0; i < discrete_inputs.size(); i++)
            discrete_inputs[i].ch = synnax_channels
                [i + holding_registers.size() + input_registers.size() + coils.size()];

        if (!holding_registers.empty()) {
            auto [plan, err] = channel::compile(holding_registers);
            if (err) {
                cfg.field_err("channels", err);
                return;
            }
            readers.push_back(
                std::make_unique<RegisterReader>(
                    device::HoldingRegister,
                    std::move(holding_registers),
                    std::move(plan)
                )
            );
        }
        if (!input_registers.empty()) {
            auto [plan, err] = channel::compile(input_registers);
            if (err) {
                cfg.field_err("channels", err);
                return;
            }
            readers.push_back(
                std::make_unique<RegisterReader>(
                    device::InputRegister,
                    std::move(input_registers),
                    std::move(plan)
                )
            );
        }
        if (!coils.empty()) {
            auto [plan, err] = channel::compile(coils);
            if (err) {
                cfg.field_err("channels", err);
                return;
            }
            readers.push_back(
                std::make_unique<BitReader>(
                    device::Coil,
                    std::move(coils),
                    std::move(plan)
                )
            );
        }
        if (!discrete_inputs.empty()) {
            auto [plan, err] = channel::compile(discrete_inputs);
            if (err) {
                cfg.field_err("channels", err);
                return;
            }
            readers.push_back(
                std::make_unique<BitReader>(
                    device::DiscreteInput,
                    std::move(discrete_inputs),
                    std::move(plan)
                )
            );
        }
        for (const auto &ch: synnax_channels)
            if (ch.index != 0) this->indexes.insert(ch.index);
    }

    /// @brief parses the configuration for the task from its JSON representation,
    /// using the provided Synnax client to retrieve the device and channel information.
    /// @param client the Synnax client to use to retrieve the device and channel
    /// information.
    /// @param task the task to parse.
    /// @returns a pair containing the parsed configuration and any error that occurred.
    static std::pair<ReadTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        ReadTaskConfig cfg(client, parser);
        return {std::move(cfg), parser.error()};
    }

    /// @brief all synnax channels that the task will write to, excluding indexes.
    [[nodiscard]] std::vector<synnax::channel::Channel> data_channels() const {
        std::vector<synnax::channel::Channel> result;
        result.reserve(this->data_channel_count);
        for (const auto &op: this->readers)
            for (const auto &ch: op->sy_channels())
                result.push_back(ch);
        return result;
    }

    /// @brief configuration for opening a synnax writer for the task.
    [[nodiscard]] synnax::framer::WriterConfig writer_config() const {
        std::vector<synnax::channel::Key> keys;
        const auto data_channels = this->data_channels();
        keys.reserve(data_channels.size() + this->indexes.size());
        for (const auto &ch: data_channels)
            keys.push_back(ch.key);
        for (const auto &idx: this->indexes)
            keys.push_back(idx);
        return synnax::framer::WriterConfig{
            .channels = keys,
            .mode = common::data_saving_writer_mode(this->data_saving_disabled),
        };
    }
};

/// @brief implements common::Source to read from a Modbus server.
class ReadTaskSource final : public common::Source {
    /// @brief the configuration for the task.
    const ReadTaskConfig config;
    /// @brief creates the device connection on each start.
    const std::shared_ptr<device::Manager> devs;
    /// @brief the device to read from. Populated on start.
    std::shared_ptr<device::Device> dev;
    /// @brief the sample clock to regulate the read rate.
    common::SoftwareTimedSampleClock sample_clock;

public:
    explicit ReadTaskSource(
        const std::shared_ptr<device::Manager> &devs,
        ReadTaskConfig cfg
    ):
        config(std::move(cfg)), devs(devs), sample_clock(this->config.sample_rate) {}

    /// @brief connects on every start so a server restart between runs cannot
    /// leave the task reading from a dead socket.
    x::errors::Error start() override {
        auto [d, err] = this->devs->acquire(this->config.conn);
        if (err) return err;
        this->dev = std::move(d);
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        this->dev.reset();
        return x::errors::NIL;
    }

    common::ReadResult
    read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        common::ReadResult res;
        const auto n_channels = this->config.data_channel_count;
        const auto n_samples = this->config.samples_per_chan;
        auto total_channel_count = n_channels + this->config.indexes.size();
        if (fr.size() != total_channel_count) {
            fr.reserve(total_channel_count);
            for (const auto &ch: this->config.data_channels())
                fr.emplace(ch.key, x::telem::Series(ch.data_type, n_samples));
            for (const auto &idx: this->config.indexes)
                fr.emplace(idx, x::telem::Series(x::telem::TIMESTAMP_T, n_samples));
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
                fr.series->at(j).write(x::telem::TimeStamp(end - (end - start) / 2));
        }
        return res;
    }

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return this->config.writer_config();
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->config.data_channels();
    }
};
}
