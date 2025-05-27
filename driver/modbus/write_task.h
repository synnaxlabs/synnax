// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// internal
#include "driver/modbus/channels.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/util/util.h"
#include "driver/task/common/write_task.h"

namespace modbus {
/// @brief interface for writing to different types of modbus registers/bits.
class Writer {
public:
    virtual ~Writer() = default;

    /// @brief write to the device from the given frame.
    /// @param dev the device to write to.
    /// @param fr the frame to write from. The frame is not guaranteed to have values
    /// for all channels in the writer. The writer should only write values for values
    /// contained in the frame. The frame may also have keys for channels that are not
    /// in the writer, which should be ignored.
    virtual xerrors::Error
    write(const std::shared_ptr<device::Device> &dev, const synnax::Frame &fr) = 0;

    /// @returns the keys of all the command channels the writer is responsible for.
    [[nodiscard]] virtual std::vector<synnax::ChannelKey> cmd_keys() const = 0;
};

/// @brief base class for all writer types.
template<typename Channel>
struct BaseWriter : Writer {
    std::vector<Channel> channels;

    explicit BaseWriter(const std::vector<Channel> &channels): channels(channels) {
        channel::sort_by_address(this->channels);
    }

    [[nodiscard]] std::vector<synnax::ChannelKey> cmd_keys() const override {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size());
        for (const auto &ch: channels)
            keys.push_back(ch.channel);
        return keys;
    }
};

/// @brief writes to coils.
class CoilWriter final : public BaseWriter<channel::OutputCoil> {
    /// @brief the current state of the coils for all channels in the writer.
    std::vector<uint8_t> state;

public:
    explicit CoilWriter(const std::vector<channel::OutputCoil> &chs): BaseWriter(chs) {}

    /// @brief initializes state if not already initialized, reading the current state
    /// of coils from the device.
    xerrors::Error initialize_state(const std::shared_ptr<device::Device> &dev) {
        if (!this->state.empty()) return xerrors::NIL;
        state.resize(channels.back().address - channels.front().address + 1);
        return dev->read_bits(
            device::Coil,
            channels.front().address,
            state.size(),
            state.data()
        );
    }

    xerrors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const synnax::Frame &fr
    ) override {
        if (channels.empty()) return xerrors::NIL;
        this->initialize_state(dev);
        const int start_addr = channels.front().address;
        for (const auto &ch: channels)
            if (fr.contains(ch.channel))
                state[ch.address - start_addr] = fr.at<uint8_t>(ch.channel, 0);
        return dev->write_bits(start_addr, state.size(), state.data());
    }
};

/// @brief writes to holding registers.
class RegisterWriter final : public BaseWriter<channel::OutputHoldingRegister> {
    /// @brief the current state of all registers in the writer.
    std::vector<uint16_t> state;

public:
    explicit RegisterWriter(const std::vector<channel::OutputHoldingRegister> &chs):
        BaseWriter(chs) {}

    /// @brief initializes state if not already initialized, reading the current state
    /// of holding registers from the device.
    xerrors::Error initialize_state(const std::shared_ptr<device::Device> &dev) {
        if (!this->state.empty()) return xerrors::NIL;
        const auto &last_ch = channels.back();
        state.resize(
            last_ch.address - channels.front().address +
            last_ch.value_type.density() / 2
        );
        return dev->read_registers(
            device::HoldingRegister,
            channels.front().address,
            state.size(),
            state.data()
        );
    }

    xerrors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const synnax::Frame &fr
    ) override {
        if (channels.empty()) return xerrors::NIL;
        this->initialize_state(dev);
        const int start_addr = channels.front().address;
        for (const auto &channel: channels) {
            if (!fr.contains(channel.channel)) continue;
            const int offset = channel.address - start_addr;
            auto err = util::format_register(
                fr.at(channel.channel, 0),
                state.data() + offset,
                channel.value_type,
                channel.swap_bytes,
                channel.swap_words
            );
            if (err) return err;
        }

        return dev->write_registers(start_addr, state.size(), state.data());
    }
};

/// @brief configuration for a modbus write task.
struct WriteTaskConfig {
    /// @brief the key of the device to read from.
    std::string device_key;
    // @brief the connection configuration for the device.
    device::ConnectionConfig conn;
    /// @brief the list of writers to use for writing data to the device.
    std::vector<std::unique_ptr<Writer>> writers;

    WriteTaskConfig(const std::shared_ptr<synnax::Synnax> &client, xjson::Parser &cfg):
        device_key(cfg.required<std::string>("device")) {
        auto [dev_info, dev_err] = client->hardware.retrieve_device(this->device_key);
        if (dev_err) {
            cfg.field_err("device", dev_err);
            return;
        }
        auto conn_parser = xjson::Parser(dev_info.properties);
        this->conn = device::ConnectionConfig(conn_parser.child("connection"));
        if (conn_parser.error()) {
            cfg.field_err("device", conn_parser.error());
            return;
        }
        std::vector<channel::OutputCoil> coils;
        std::vector<channel::OutputHoldingRegister> registers;
        cfg.iter("channels", [&](xjson::Parser &ch) {
            const auto type = ch.required<std::string>("type");
            if (type == "coil_output")
                coils.emplace_back(ch);
            else if (type == "holding_register_output")
                registers.emplace_back(ch);
            else
                cfg.field_err("channels", "invalid channel type: " + type);
        });
        if (!coils.empty())
            writers.push_back(std::make_unique<CoilWriter>(std::move(coils)));
        if (!registers.empty())
            writers.push_back(std::make_unique<RegisterWriter>(std::move(registers)));
    }

    /// @returns the keys of all command channels used by the writer.
    [[nodiscard]] std::vector<synnax::ChannelKey> cmd_keys() const {
        std::vector<synnax::ChannelKey> keys;
        for (const auto &writer: writers)
            for (const auto &key: writer->cmd_keys())
                keys.push_back(key);
        return keys;
    }

    /// @brief parses the configuration for the task from its JSON representation,
    /// using the provided Synnax client to retrieve device and channel information.
    /// @param client the Synnax client to use to retrieve the device and channel
    /// information.
    /// @param task the task to parse.
    /// @returns a pair containing the parsed configuration and any error that
    /// occurred during parsing.
    static std::pair<WriteTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }
};

/// @brief implements common::Sink to write to a modbus device.
class WriteTaskSink final : public common::Sink {
    /// @brief the configuration for the task.
    const WriteTaskConfig config;
    /// @brief the device to write to.
    std::shared_ptr<device::Device> dev;

public:
    WriteTaskSink(const std::shared_ptr<device::Device> &dev, WriteTaskConfig cfg):
        Sink(cfg.cmd_keys()), config(std::move(cfg)), dev(dev) {}

    xerrors::Error write(const synnax::Frame &frame) override {
        for (const auto &writer: config.writers)
            if (auto err = writer->write(dev, frame)) return err;
        return xerrors::NIL;
    }
};
}
