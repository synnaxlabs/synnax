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
#include "driver/modbus/device/device.h"
#include "driver/modbus/channels.h"
#include "driver/task/common/write_task.h"
#include "driver/modbus/util/util.h"

namespace modbus {
class Writer {
public:
    virtual ~Writer() = default;

    virtual xerrors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const synnax::Frame &fr
    ) = 0;

    virtual std::vector<synnax::ChannelKey> cmd_keys() const = 0;
};

class CoilWriter final : public Writer {
    std::vector<channel::OutputCoilChannel> channels;
    std::vector<uint8_t> buffer;

public:
    explicit CoilWriter(std::vector<channel::OutputCoilChannel> chs):
        channels(std::move(chs)) {
        // Sort channels by address to optimize writes
        std::sort(channels.begin(), channels.end(),
                  [](const auto &a, const auto &b) { return a.address < b.address; });
        // Buffer size is the span from first to last address
        buffer.resize(channels.back().address - channels.front().address + 1);
    }

    xerrors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const synnax::Frame &fr
    ) override {
        if (channels.empty()) return xerrors::NIL;

        const int start_addr = channels.front().address;
        std::fill(buffer.begin(), buffer.end(), 0);

        for (const auto &ch: channels)
            if (fr.contains(ch.channel))
                buffer[ch.address - start_addr] =
                    fr.at<uint8_t>(ch.channel, 0);

        return dev->write_bits(start_addr, buffer.size(), buffer.data());
    }

    std::vector<synnax::ChannelKey> cmd_keys() const override {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size());
        for (const auto &ch: channels)
            keys.push_back(ch.channel);
        return keys;
    }
};

class RegisterWriter final : public Writer {
    std::vector<channel::OutputHoldingRegisterChannel> channels;
    std::vector<uint16_t> buffer;

public:
    explicit RegisterWriter(std::vector<channel::OutputHoldingRegisterChannel> chs):
        channels(std::move(chs)) {
        std::sort(channels.begin(), channels.end(),
                  [](const auto &a, const auto &b) { return a.address < b.address; });
        const auto &last_ch = channels.back();
        buffer.resize(
            last_ch.address - channels.front().address +
            last_ch.value_type.density() / 2
        );
    }

    xerrors::Error write(
        const std::shared_ptr<device::Device> &dev,
        const synnax::Frame &fr
    ) override {
        if (channels.empty()) return xerrors::NIL;
        const int start_addr = channels.front().address;
        std::fill(buffer.begin(), buffer.end(), 0);
        for (const auto &channel: channels) {
            if (!fr.contains(channel.channel)) continue;
            const int offset = channel.address - start_addr;
            auto err = util::format_register(
                fr.at(channel.channel, 0),
                buffer.data() + offset,
                channel.value_type,
                channel.swap_bytes,
                channel.swap_words
            );
            if (err) return err;
        }

        return dev->write_registers(
            start_addr,
            buffer.size(),
            buffer.data()
        );
    }

    std::vector<synnax::ChannelKey> cmd_keys() const override {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size());
        for (const auto &channel: channels)
            keys.push_back(channel.channel);
        return keys;
    }
};

struct WriteTaskConfig {
    device::ConnectionConfig conn;
    std::string dev;
    std::vector<std::unique_ptr<Writer> > writers;

    WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ): dev(cfg.required<std::string>("device")) {
        auto [dev_info, dev_err] = client->hardware.retrieve_device(this->dev);
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
        std::vector<channel::OutputCoilChannel> coils;
        std::vector<channel::OutputHoldingRegisterChannel> registers;
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

    [[nodiscard]] std::vector<synnax::ChannelKey> cmd_keys() const {
        std::vector<synnax::ChannelKey> keys;
        for (const auto &writer: writers)
            for (const auto &key: writer->cmd_keys())
                keys.push_back(key);
        return keys;
    }

    static std::pair<WriteTaskConfig, xerrors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }
};

class WriteTaskSink final : public common::Sink {
    const WriteTaskConfig config;
    std::shared_ptr<device::Device> dev;

public:
    WriteTaskSink(
        const std::shared_ptr<device::Device> &dev,
        WriteTaskConfig cfg
    ): Sink(cfg.cmd_keys()),
       config(std::move(cfg)),
       dev(dev) {
    }

    xerrors::Error write(const synnax::Frame &frame) override {
        for (const auto &writer: config.writers)
            if (auto err = writer->write(dev, frame))
                return err;
        return xerrors::NIL;
    }
};
}
