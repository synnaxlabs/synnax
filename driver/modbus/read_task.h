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
#include <string>
#include <unordered_map>
#include <utility>
#include <variant>

#include "client/cpp/modbus/json.gen.h"

#include "driver/common/read_task.h"
#include "driver/common/sample_clock.h"
#include "driver/modbus/block.h"
#include "driver/modbus/channels.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/util/util.h"

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
    /// @brief the channels of the reader, sorted by address.
    std::vector<Channel> channels;

    explicit BaseReader(const std::vector<Channel> &channels): channels(channels) {
        channel::sort_by_address(this->channels);
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> sy_channels() const override {
        std::vector<synnax::channel::Channel> result;
        result.reserve(channels.size());
        for (const auto &channel: channels)
            result.push_back(channel.ch);
        return result;
    }

protected:
    /// @brief names the channels and addresses of a failed block read. A lost
    /// connection is about the device, not the block, so it passes through.
    [[nodiscard]] x::errors::Error wrap(
        const x::errors::Error &err,
        const std::string &table,
        const block::Block &b
    ) const {
        if (err.matches(device::UNREACHABLE)) return err;
        std::vector<std::string> names;
        for (std::size_t i = b.first; i < b.last; i++)
            names.push_back(this->channels[i].ch.name);
        return block::wrap(err, table, b, names);
    }
};

/// @returns the size of the buffer that holds the largest block.
inline std::size_t max_count(const std::vector<block::Block> &blocks) {
    std::size_t count = 0;
    for (const auto &b: blocks)
        count = std::max(count, b.count);
    return count;
}

/// @brief reads from holding and input registers.
class RegisterReader final : public BaseReader<channel::InputRegister> {
    /// @brief the register type to read from. either HoldingRegister or InputRegister.
    device::RegisterType register_type;
    /// @brief the requests that cover the channels.
    std::vector<block::Block> blocks;
    /// @brief the buffer to read each block into.
    std::vector<uint16_t> buffer;

public:
    explicit RegisterReader(
        const device::RegisterType register_type,
        const std::vector<channel::InputRegister> &chs,
        const device::RequestConfig &requests = {}
    ):
        BaseReader(chs),
        register_type(register_type),
        blocks(block::plan(this->channels, requests.max_registers, requests.max_gap)),
        buffer(max_count(this->blocks)) {}

    x::errors::Error read(
        const std::shared_ptr<device::Device> &dev,
        x::telem::Frame &fr,
        size_t &frame_offset
    ) override {
        for (const auto &b: this->blocks) {
            if (const auto err = dev->read_registers(
                    this->register_type,
                    b.start,
                    b.count,
                    this->buffer.data()
                ))
                return this->wrap(
                    err,
                    this->register_type == device::HoldingRegister ? "holding registers"
                                                                   : "input registers",
                    b
                );
            for (std::size_t i = b.first; i < b.last; i++) {
                const auto &ch = this->channels[i];
                auto [value, err] = ch.decode(
                    this->buffer.data() + (ch.address - b.start)
                );
                if (err) return err;
                auto &s = fr.series->at(frame_offset++);
                s.write(s.data_type().cast(value));
            }
        }
        return x::errors::NIL;
    }
};

/// @brief reads from coils and discrete inputs.
class BitReader final : public BaseReader<channel::InputDiscrete> {
    /// @brief the bit type to read from. either Coil or DiscreteInput.
    device::BitType bit_type;
    /// @brief the requests that cover the channels.
    std::vector<block::Block> blocks;
    /// @brief the buffer to read each block into.
    std::vector<uint8_t> buffer;

public:
    explicit BitReader(
        const device::BitType bit_type,
        const std::vector<channel::InputDiscrete> &channels,
        const device::RequestConfig &requests = {}
    ):
        BaseReader(channels),
        bit_type(bit_type),
        blocks(block::plan(this->channels, MODBUS_MAX_READ_BITS, requests.max_gap)),
        buffer(max_count(this->blocks)) {}

    x::errors::Error read(
        const std::shared_ptr<device::Device> &dev,
        x::telem::Frame &fr,
        size_t &frame_offset
    ) override {
        for (const auto &b: this->blocks) {
            if (const auto err = dev->read_bits(
                    this->bit_type,
                    b.start,
                    b.count,
                    this->buffer.data()
                ))
                return this->wrap(
                    err,
                    this->bit_type == device::Coil ? "coils" : "discrete inputs",
                    b
                );
            for (std::size_t i = b.first; i < b.last; i++) {
                auto &s = fr.series->at(frame_offset++);
                s.write(s.data_type().cast(
                    this->buffer[this->channels[i].address - b.start]
                ));
            }
        }
        return x::errors::NIL;
    }
};

/// @brief checks the data type and string length of a register value channel.
/// @returns false and records a field error on ch if the channel is invalid.
inline bool valid_value(
    x::json::Parser &ch,
    const ::synnax::modbus::RegisterValue &value,
    const std::int32_t string_length
) {
    if (value.data_type == x::telem::STRING_T) {
        if (string_length <= 0 ||
            std::cmp_greater(string_length, channel::MAX_STRING_LENGTH)) {
            ch.field_err(
                "string_length",
                "must be from 1 to " + std::to_string(channel::MAX_STRING_LENGTH)
            );
            return false;
        }
        return true;
    }
    if (!channel::is_register_type(value.data_type)) {
        ch.field_err("data_type", "unsupported data type: " + value.data_type.name());
        return false;
    }
    return true;
}

/// @brief checks the bit index of a register bit channel.
/// @returns false and records a field error on ch if the bit is past 15.
inline bool valid_bit(x::json::Parser &ch, const ::synnax::modbus::RegisterBit &bit) {
    if (bit.bit < 16) return true;
    ch.field_err("bit", "must be from 0 to 15");
    return false;
}

/// @brief configuration for a modbus read task.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// @brief the total number of data channels in the task.
    size_t data_channel_count{0};
    /// @brief the key of the device to read from.
    std::string device_key;
    /// @brief the indexes of all data channels in the task.
    /// Dynamically populated by querying the core.
    std::set<synnax::channel::Key> indexes;
    /// @brief the list of readers to use for reading data from the device.
    std::vector<std::unique_ptr<Reader>> readers;
    /// @brief the properties of the device. Dynamically populated at configure.
    device::Properties props;
    /// @brief the number of samples per channel to read on each read() call.
    std::size_t samples_per_chan;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        BaseReadTaskConfig(std::move(other)),
        data_channel_count(other.data_channel_count),
        device_key(std::move(other.device_key)),
        indexes(std::move(other.indexes)),
        readers(std::move(other.readers)),
        props(std::move(other.props)),
        samples_per_chan(other.samples_per_chan) {}

    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &cfg
    ):
        BaseReadTaskConfig(cfg),
        device_key(cfg.field<std::string>("device")),
        samples_per_chan(sample_rate / stream_rate) {
        std::vector<channel::InputRegister> holding_registers;
        std::vector<channel::InputRegister> input_registers;
        std::vector<channel::InputDiscrete> coils;
        std::vector<channel::InputDiscrete> discrete_inputs;

        auto [dev, dev_err] = client->devices.retrieve(this->device_key);
        if (dev_err) {
            cfg.field_err("device", dev_err.data);
            return;
        }

        auto props_parser = x::json::Parser(dev.properties);
        this->props = device::Properties(props_parser);
        if (props_parser.error()) {
            cfg.field_err(
                "device",
                common::config_err_description(props_parser.error())
            );
            return;
        }

        std::set<synnax::channel::Key> keys;
        std::vector<std::pair<x::json::Parser, synnax::channel::Key>> bindings;
        std::vector<std::pair<x::json::Parser, synnax::channel::Key>> strings;
        const auto max_registers = this->props.requests.max_registers;
        const auto fits =
            [max_registers](x::json::Parser &ch, const channel::InputRegister &reg) {
                if (reg.width() <= max_registers) return true;
                ch.field_err(
                    reg.value_type == x::telem::STRING_T ? "string_length"
                                                         : "data_type",
                    "needs " + std::to_string(reg.width()) +
                        " registers, but the device reads at most " +
                        std::to_string(max_registers) + " per request"
                );
                return false;
            };
        cfg.iter("channels", [&, this](x::json::Parser &ch) {
            const auto parsed = ::synnax::modbus::parse_read_channel(ch);
            const auto &base = std::visit(
                [](const auto &c) -> const ::synnax::modbus::BaseReadChannel & {
                    return c;
                },
                parsed
            );
            if (base.disabled) return;
            if (base.channel == 0) {
                ch.field_err("channel", "channel must be specified");
                return;
            }
            if (!keys.insert(base.channel).second) {
                ch.field_err("channel", "another channel in the task reads into it");
                return;
            }
            auto *bound = &bindings;
            if (const auto *c = std::get_if<
                    ::synnax::modbus::HoldingRegisterReadChannel>(&parsed)) {
                if (!valid_value(ch, *c, c->string_length)) return;
                channel::InputRegister reg(*c, this->props.connection);
                if (!fits(ch, reg)) return;
                if (c->data_type == x::telem::STRING_T) bound = &strings;
                holding_registers.push_back(std::move(reg));
            } else if (
                const auto *c = std::get_if<::synnax::modbus::InputRegisterReadChannel>(
                    &parsed
                )
            ) {
                if (!valid_value(ch, *c, c->string_length)) return;
                channel::InputRegister reg(*c, this->props.connection);
                if (!fits(ch, reg)) return;
                if (c->data_type == x::telem::STRING_T) bound = &strings;
                input_registers.push_back(std::move(reg));
            } else if (
                const auto *c = std::get_if<
                    ::synnax::modbus::HoldingRegisterBitReadChannel>(&parsed)
            ) {
                if (!valid_bit(ch, *c)) return;
                holding_registers.emplace_back(*c);
            } else if (
                const auto *c = std::get_if<
                    ::synnax::modbus::InputRegisterBitReadChannel>(&parsed)
            ) {
                if (!valid_bit(ch, *c)) return;
                input_registers.emplace_back(*c);
            } else if (
                const auto *c = std::get_if<::synnax::modbus::CoilReadChannel>(&parsed)
            )
                coils.emplace_back(*c);
            else
                discrete_inputs.emplace_back(
                    std::get<::synnax::modbus::DiscreteInputReadChannel>(parsed)
                );
            bound->emplace_back(ch, base.channel);
            this->data_channel_count++;
        });
        if (cfg.ok() && keys.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }

        auto [synnax_channels, err] = client->channels.retrieve(
            std::vector(keys.begin(), keys.end())
        );
        if (err) {
            cfg.field_err("channels", err.data);
            return;
        }
        std::unordered_map<synnax::channel::Key, synnax::channel::Channel> by_key;
        for (const auto &ch: synnax_channels) {
            by_key.emplace(ch.key, ch);
            if (ch.index != 0) this->indexes.insert(ch.index);
        }
        for (auto &[ch, key]: bindings)
            if (const auto &dt = by_key.at(key).data_type;
                !channel::is_register_type(dt))
                ch.field_err(
                    "channel",
                    "must be a numeric channel, not a " + dt.name() + " channel"
                );
        for (auto &[ch, key]: strings)
            if (by_key.at(key).data_type != x::telem::STRING_T)
                ch.field_err("channel", "a string must read into a string channel");
        for (auto &ch: holding_registers)
            ch.ch = by_key.at(ch.synnax_key);
        for (auto &ch: input_registers)
            ch.ch = by_key.at(ch.synnax_key);
        for (auto &ch: coils)
            ch.ch = by_key.at(ch.synnax_key);
        for (auto &ch: discrete_inputs)
            ch.ch = by_key.at(ch.synnax_key);

        const auto &requests = this->props.requests;
        if (!holding_registers.empty())
            readers.push_back(
                std::make_unique<RegisterReader>(
                    device::HoldingRegister,
                    std::move(holding_registers),
                    requests
                )
            );
        if (!input_registers.empty())
            readers.push_back(
                std::make_unique<RegisterReader>(
                    device::InputRegister,
                    std::move(input_registers),
                    requests
                )
            );
        if (!coils.empty())
            readers.push_back(
                std::make_unique<BitReader>(device::Coil, std::move(coils), requests)
            );
        if (!discrete_inputs.empty())
            readers.push_back(
                std::make_unique<BitReader>(
                    device::DiscreteInput,
                    std::move(discrete_inputs),
                    requests
                )
            );
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
    /// @brief shares the connection to the device with other tasks.
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

    x::errors::Error start() override {
        auto [d, err] = this->devs->acquire(
            this->config.props.connection,
            this->config.props.requests
        );
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
