// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <unordered_map>
#include <variant>

#include "client/cpp/modbus/json.gen.h"

#include "driver/common/write_task.h"
#include "driver/modbus/block.h"
#include "driver/modbus/channels.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/util/util.h"

namespace driver::modbus {
/// @brief interface for writing to different types of modbus registers/bits.
class Writer {
public:
    virtual ~Writer() = default;

    /// @brief writes the commands in the frame to the device. Addresses with no command
    /// in the frame are left untouched.
    /// @param dev the device to write to.
    /// @param fr the frame of commands. It may hold only some of the writer's channels,
    /// and channels of other writers, which are ignored.
    virtual x::errors::Error
    write(const std::shared_ptr<device::Device> &dev, const x::telem::Frame &fr) = 0;

    /// @returns the keys of all the command channels the writer is responsible for.
    [[nodiscard]] virtual std::vector<synnax::channel::Key> cmd_keys() const = 0;
};

/// @brief base class for all writer types.
template<typename Channel>
struct BaseWriter : Writer {
    /// @brief the channels of the writer, sorted by address.
    std::vector<Channel> channels;

    explicit BaseWriter(const std::vector<Channel> &channels): channels(channels) {
        channel::sort_by_address(this->channels);
    }

    [[nodiscard]] std::vector<synnax::channel::Key> cmd_keys() const override {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(channels.size());
        for (const auto &ch: channels)
            keys.push_back(ch.channel);
        return keys;
    }

protected:
    /// @returns the channels with a command in the frame, sorted by address.
    [[nodiscard]] std::vector<Channel> commanded(const x::telem::Frame &fr) const {
        std::vector<Channel> result;
        for (const auto &ch: this->channels)
            if (fr.contains(ch.channel)) result.push_back(ch);
        return result;
    }

    /// @brief names the channels and addresses of a failed block write. A lost
    /// connection is about the device, not the block, so it passes through.
    static x::errors::Error wrap(
        const x::errors::Error &err,
        const std::string &table,
        const block::Block &b,
        const std::vector<Channel> &chs
    ) {
        if (err.matches(device::UNREACHABLE)) return err;
        std::vector<std::string> names;
        for (std::size_t i = b.first; i < b.last; i++)
            names.push_back(chs[i].name);
        return block::wrap(err, table, b, names);
    }
};

/// @brief writes to coils.
class CoilWriter final : public BaseWriter<channel::OutputCoil> {
public:
    explicit CoilWriter(const std::vector<channel::OutputCoil> &chs): BaseWriter(chs) {}

    x::errors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const x::telem::Frame &fr
    ) override {
        const auto cmds = this->commanded(fr);
        for (const auto &b: block::plan(cmds, MODBUS_MAX_WRITE_BITS, 0)) {
            std::vector<uint8_t> bits(b.count);
            for (std::size_t i = b.first; i < b.last; i++) {
                const auto value = fr.at(cmds[i].channel, -1);
                bits[cmds[i].address - b.start] = x::telem::cast<double>(value) != 0;
            }
            if (auto err = dev->write_bits(b.start, b.count, bits.data()))
                return wrap(err, "coils", b, cmds);
        }
        return x::errors::NIL;
    }
};

/// @brief writes to holding registers.
class RegisterWriter final : public BaseWriter<channel::OutputHoldingRegister> {
public:
    explicit RegisterWriter(const std::vector<channel::OutputHoldingRegister> &chs):
        BaseWriter(chs) {}

    x::errors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const x::telem::Frame &fr
    ) override {
        const auto cmds = this->commanded(fr);
        for (const auto &b: block::plan(cmds, MODBUS_MAX_WRITE_REGISTERS, 0)) {
            std::vector<uint16_t> registers(b.count);
            for (std::size_t i = b.first; i < b.last; i++) {
                const auto &ch = cmds[i];
                if (auto err = util::format_register(
                        fr.at(ch.channel, -1),
                        registers.data() + (ch.address - b.start),
                        ch.value_type,
                        ch.bytes_swapped,
                        ch.words_swapped
                    ))
                    return err;
            }
            if (auto err = dev->write_registers(b.start, b.count, registers.data()))
                return wrap(err, "holding registers", b, cmds);
        }
        return x::errors::NIL;
    }
};

/// @brief writes single bits of holding registers. The commanded bits of one register
/// go in one mask write.
class RegisterBitWriter final : public BaseWriter<channel::OutputRegisterBit> {
public:
    explicit RegisterBitWriter(const std::vector<channel::OutputRegisterBit> &chs):
        BaseWriter(chs) {}

    x::errors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const x::telem::Frame &fr
    ) override {
        const auto cmds = this->commanded(fr);
        for (std::size_t first = 0, last = 0; first < cmds.size(); first = last) {
            const auto address = cmds[first].address;
            std::uint16_t and_mask = 0xFFFF;
            std::uint16_t or_mask = 0;
            for (last = first; last < cmds.size() && cmds[last].address == address;
                 last++) {
                const auto bit = static_cast<std::uint16_t>(1U << cmds[last].bit);
                and_mask &= ~bit;
                if (x::telem::cast<double>(fr.at(cmds[last].channel, -1)) != 0)
                    or_mask |= bit;
            }
            if (auto err = dev->mask_write_register(address, and_mask, or_mask))
                return wrap(err, "holding registers", {address, 1, first, last}, cmds);
        }
        return x::errors::NIL;
    }
};

/// @returns the first channel whose addresses overlap those of ch, or nullptr.
template<typename Channel, typename Other>
const Channel *find_overlap(const std::vector<Channel> &channels, const Other &ch) {
    for (const auto &other: channels)
        if (ch.address < other.address + other.width() &&
            other.address < ch.address + ch.width())
            return &other;
    return nullptr;
}

/// @brief configuration for a modbus write task.
struct WriteTaskConfig : common::BaseWriteTaskConfig {
    /// @brief the properties of the device. Dynamically populated at configure.
    device::Properties props;
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
        auto props_parser = x::json::Parser(dev_info.properties);
        this->props = device::Properties(props_parser);
        if (props_parser.error()) {
            cfg.field_err("device", props_parser.error());
            return;
        }
        std::vector<channel::OutputCoil> coils;
        std::vector<channel::OutputHoldingRegister> registers;
        std::vector<channel::OutputRegisterBit> bits;
        cfg.iter("channels", [&](x::json::Parser &ch) {
            const auto parsed = ::synnax::modbus::parse_write_channel(ch);
            const auto &base = std::visit(
                [](const auto &c) -> const ::synnax::modbus::BaseWriteChannel & {
                    return c;
                },
                parsed
            );
            if (base.disabled) return;
            if (base.channel == 0) {
                ch.field_err("channel", "channel must be specified");
                return;
            }
            if (const auto *c = std::get_if<
                    ::synnax::modbus::HoldingRegisterWriteChannel>(&parsed)) {
                if (!channel::is_register_type(c->data_type)) {
                    ch.field_err(
                        "data_type",
                        "unsupported data type: " + c->data_type.name()
                    );
                    return;
                }
                const channel::OutputHoldingRegister reg(*c, this->props.connection);
                if (const auto *other = find_overlap(registers, reg)) {
                    ch.field_err(
                        "address",
                        "overlaps the holding register at address " +
                            std::to_string(other->address)
                    );
                    return;
                }
                if (const auto *other = find_overlap(bits, reg)) {
                    ch.field_err(
                        "address",
                        "overlaps the holding register bit at address " +
                            std::to_string(other->address)
                    );
                    return;
                }
                registers.push_back(reg);
                return;
            }
            if (const auto *c = std::get_if<
                    ::synnax::modbus::HoldingRegisterBitWriteChannel>(&parsed)) {
                if (c->bit > 15) {
                    ch.field_err("bit", "must be from 0 to 15");
                    return;
                }
                const channel::OutputRegisterBit bit(*c);
                if (const auto *other = find_overlap(registers, bit)) {
                    ch.field_err(
                        "address",
                        "overlaps the holding register at address " +
                            std::to_string(other->address)
                    );
                    return;
                }
                for (const auto &other: bits)
                    if (other.address == bit.address && other.bit == bit.bit) {
                        ch.field_err(
                            "bit",
                            "another channel writes this bit of the register"
                        );
                        return;
                    }
                bits.push_back(bit);
                return;
            }
            const channel::OutputCoil coil(
                std::get<::synnax::modbus::CoilWriteChannel>(parsed)
            );
            if (const auto *other = find_overlap(coils, coil)) {
                ch.field_err(
                    "address",
                    "overlaps the coil at address " + std::to_string(other->address)
                );
                return;
            }
            coils.push_back(coil);
        });
        std::vector<synnax::channel::Key> keys;
        keys.reserve(coils.size() + registers.size() + bits.size());
        for (const auto &ch: coils)
            keys.push_back(ch.channel);
        for (const auto &ch: registers)
            keys.push_back(ch.channel);
        for (const auto &ch: bits)
            keys.push_back(ch.channel);
        auto [chs, chs_err] = client->channels.retrieve(keys);
        if (chs_err) {
            cfg.field_err("channels", chs_err);
            return;
        }
        std::unordered_map<synnax::channel::Key, std::string> names;
        for (const auto &ch: chs)
            names.emplace(ch.key, ch.name);
        for (auto &ch: coils)
            ch.name = names.at(ch.channel);
        for (auto &ch: registers)
            ch.name = names.at(ch.channel);
        for (auto &ch: bits)
            ch.name = names.at(ch.channel);
        if (!coils.empty())
            writers.push_back(std::make_unique<CoilWriter>(std::move(coils)));
        if (!registers.empty())
            writers.push_back(std::make_unique<RegisterWriter>(std::move(registers)));
        if (!bits.empty())
            writers.push_back(std::make_unique<RegisterBitWriter>(std::move(bits)));
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
    /// @brief shares the connection to the device with other tasks.
    const std::shared_ptr<device::Manager> devs;
    /// @brief the device to write to. Populated on start.
    std::shared_ptr<device::Device> dev;

public:
    WriteTaskSink(const std::shared_ptr<device::Manager> &devs, WriteTaskConfig cfg):
        Sink(cfg.cmd_keys()), config(std::move(cfg)), devs(devs) {}

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

    x::errors::Error write(x::telem::Frame &frame) override {
        for (const auto &writer: this->config.writers)
            if (auto err = writer->write(this->dev, frame)) return err;
        return x::errors::NIL;
    }
};
}
