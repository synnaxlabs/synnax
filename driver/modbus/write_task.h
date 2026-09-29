// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <variant>

#include "client/cpp/modbus/json.gen.h"

#include "driver/codec/plan.h"
#include "driver/common/write_task.h"
#include "driver/modbus/channels.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/registers/registers.h"

namespace driver::modbus {
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
    virtual x::errors::Error
    write(const std::shared_ptr<device::Device> &dev, const x::telem::Frame &fr) = 0;

    /// @returns the keys of all the command channels the writer is responsible for.
    [[nodiscard]] virtual std::vector<synnax::channel::Key> cmd_keys() const = 0;
};

/// @brief base class for all writer types.
template<typename Channel>
struct BaseWriter : Writer {
    std::vector<Channel> channels;
    /// @brief encodes each channel's command into the payload.
    codec::Plan plan;
    /// @brief holds the commands of one write, reused across writes.
    codec::Values values;
    /// @brief the current state of the device, as bytes. Empty until initialized.
    std::vector<uint8_t> payload;

    /// @param channels sorted by address.
    /// @param plan from channel::compile(channels).
    BaseWriter(const std::vector<Channel> &channels, codec::Plan plan):
        channels(channels), plan(std::move(plan)), values(this->plan.values()) {}

    [[nodiscard]] std::vector<synnax::channel::Key> cmd_keys() const override {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(channels.size());
        for (const auto &ch: channels)
            keys.push_back(ch.channel);
        return keys;
    }

protected:
    /// @brief encodes the latest command in the frame for each channel into the
    /// payload, converting it to the channel's value type first.
    x::errors::Error encode(const x::telem::Frame &fr) {
        this->values.clear();
        for (size_t i = 0; i < this->channels.size(); i++) {
            const auto &ch = this->channels[i];
            if (!fr.contains(ch.channel)) continue;
            const auto command = ch.value_type.cast(fr.at(ch.channel, -1));
            if (const auto err = this->values.set(i, command)) return err;
        }
        return this->plan.encode(this->values, this->payload);
    }
};

/// @brief writes to coils.
class CoilWriter final : public BaseWriter<channel::OutputCoil> {
public:
    using BaseWriter::BaseWriter;

    /// @brief initializes state if not already initialized, reading the current state
    /// of coils from the device.
    x::errors::Error initialize_state(const std::shared_ptr<device::Device> &dev) {
        if (!this->payload.empty()) return x::errors::NIL;
        this->payload.resize(this->plan.length());
        return dev->read_bits(
            device::Coil,
            channels.front().address,
            this->payload.size(),
            this->payload.data()
        );
    }

    x::errors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const x::telem::Frame &fr
    ) override {
        if (channels.empty()) return x::errors::NIL;
        this->initialize_state(dev);
        if (const auto err = this->encode(fr)) return err;
        return dev->write_bits(
            channels.front().address,
            this->payload.size(),
            this->payload.data()
        );
    }
};

/// @brief writes to holding registers.
class RegisterWriter final : public BaseWriter<channel::OutputHoldingRegister> {
    /// @brief the registers exchanged with the device.
    std::vector<uint16_t> buffer;

public:
    RegisterWriter(
        const std::vector<channel::OutputHoldingRegister> &chs,
        codec::Plan plan
    ):
        BaseWriter(chs, std::move(plan)), buffer((this->plan.length() + 1) / 2) {}

    /// @brief initializes state if not already initialized, reading the current state
    /// of holding registers from the device.
    x::errors::Error initialize_state(const std::shared_ptr<device::Device> &dev) {
        if (!this->payload.empty()) return x::errors::NIL;
        this->payload.resize(this->buffer.size() * 2);
        const auto err = dev->read_registers(
            device::HoldingRegister,
            channels.front().address,
            this->buffer.size(),
            this->buffer.data()
        );
        registers::to_bytes(this->buffer, this->payload);
        return err;
    }

    x::errors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const x::telem::Frame &fr
    ) override {
        if (channels.empty()) return x::errors::NIL;
        this->initialize_state(dev);
        if (const auto err = this->encode(fr)) return err;
        registers::to_registers(this->payload, this->buffer);
        return dev->write_registers(
            channels.front().address,
            this->buffer.size(),
            this->buffer.data()
        );
    }
};

/// @brief configuration for a modbus write task.
struct WriteTaskConfig : common::BaseWriteTaskConfig {
    /// @brief the connection configuration for the device.
    /// Dynamically populated from device properties.
    device::ConnectionConfig conn;
    /// @brief the list of writers to use for writing data to the device.
    std::vector<std::unique_ptr<Writer>> writers;

    WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &cfg
    ):
        common::BaseWriteTaskConfig(cfg) {
        auto [dev_info, dev_err] = client->devices.retrieve(this->device);
        if (dev_err) {
            cfg.field_err("device", dev_err);
            return;
        }
        auto conn_parser = x::json::Parser(dev_info.properties);
        this->conn = device::ConnectionConfig(conn_parser.child("connection"));
        if (conn_parser.error()) {
            cfg.field_err("device", conn_parser.error());
            return;
        }
        std::vector<channel::OutputCoil> coils;
        std::vector<channel::OutputHoldingRegister> holding_registers;
        cfg.iter("channels", [&](x::json::Parser &ch) {
            const auto parsed = ::synnax::modbus::parse_write_channel(ch);
            const auto &base = std::visit(
                [](const auto &c) -> const ::synnax::modbus::BaseWriteChannel & {
                    return c;
                },
                parsed
            );
            if (base.disabled) return;
            if (base.channel == 0)
                return ch.field_err("channel", "channel must be specified");
            if (const auto *c = std::get_if<
                    ::synnax::modbus::HoldingRegisterWriteChannel>(&parsed))
                holding_registers.emplace_back(*c);
            else
                coils.emplace_back(
                    std::get<::synnax::modbus::CoilWriteChannel>(parsed)
                );
        });
        channel::sort_by_address(coils);
        channel::sort_by_address(holding_registers);
        for (const auto &err:
             {channel::append<CoilWriter>(this->writers, std::move(coils)),
              channel::append<RegisterWriter>(
                  this->writers,
                  std::move(holding_registers)
              )})
            if (err) cfg.field_err("channels", err);
    }

    /// @returns the keys of all command channels used by the writer.
    [[nodiscard]] std::vector<synnax::channel::Key> cmd_keys() const {
        std::vector<synnax::channel::Key> keys;
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
    /// @returns a pair containing the parsed configuration and any error that occurred.
    static std::pair<WriteTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        WriteTaskConfig cfg(client, parser);
        return {std::move(cfg), parser.error()};
    }
};

/// @brief implements common::Sink to write to a Modbus server.
class WriteTaskSink final : public common::Sink {
    /// @brief the configuration for the task.
    const WriteTaskConfig config;
    /// @brief creates the device connection on each start.
    const std::shared_ptr<device::Manager> devs;
    /// @brief the device to write to. Populated on start.
    std::shared_ptr<device::Device> dev;

public:
    WriteTaskSink(const std::shared_ptr<device::Manager> &devs, WriteTaskConfig cfg):
        Sink(x::telem::Rate(0), {}, {}, cfg.cmd_keys(), cfg.data_saving_disabled),
        config(std::move(cfg)),
        devs(devs) {}

    /// @brief connects on every start so a server restart between runs cannot
    /// leave the task writing to a dead socket.
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

    x::errors::Error write(x::telem::Frame &frame) override {
        for (const auto &writer: config.writers)
            if (auto err = writer->write(dev, frame)) return err;
        return x::errors::NIL;
    }
};
}
