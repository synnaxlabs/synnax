// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <unordered_map>

#include "driver/visa/channels.h"
#include "driver/visa/device/device.h"
#include "driver/task/common/write_task.h"

namespace visa {

/// @brief Configuration for a VISA write task.
struct WriteTaskConfig {
    /// @brief The key of the device to write to.
    std::string device_key;
    /// @brief The connection configuration for the device.
    device::ConnectionConfig conn;
    /// @brief The list of output channels to write.
    std::vector<channel::OutputChannel> channels;

    /// @brief Default constructor for testing.
    WriteTaskConfig():
        device_key(""),
        conn(),
        channels({}) {}

    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        device_key(std::move(other.device_key)),
        conn(std::move(other.conn)),
        channels(std::move(other.channels)) {}

    WriteTaskConfig(const WriteTaskConfig &) = delete;

    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ):
        device_key(cfg.required<std::string>("device")) {
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
            this->channels.emplace_back(ch);
        });

        // Retrieve Synnax channels
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size());
        for (const auto &ch: channels)
            keys.push_back(ch.synnax_key);

        auto [synnax_channels, err] = client->channels.retrieve(keys);
        if (err) {
            cfg.field_err("channels", err.message());
            return;
        }

        for (size_t i = 0; i < channels.size(); i++)
            channels[i].ch = synnax_channels[i];
    }

    /// @brief parses the configuration for the task from its JSON representation.
    static std::pair<WriteTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }
};

/// @brief Implements common::Sink to write to a VISA device.
class WriteTaskSink final : public common::Sink {
    /// @brief The configuration for the task.
    const WriteTaskConfig config;
    /// @brief The VISA session to write to.
    std::shared_ptr<device::Session> session;
    /// @brief Map from Synnax channel key to output channel config for O(1) lookup.
    std::unordered_map<synnax::ChannelKey, const channel::OutputChannel*> channel_map;

public:
    explicit WriteTaskSink(
        const std::shared_ptr<device::Session> &sess,
        WriteTaskConfig cfg
    ):
        common::Sink([&cfg]() {
            std::vector<synnax::ChannelKey> keys;
            keys.reserve(cfg.channels.size());
            for (const auto &ch: cfg.channels)
                keys.push_back(ch.synnax_key);
            return keys;
        }()),
        config(std::move(cfg)),
        session(sess) {
        // Build channel map for O(1) lookup
        for (const auto &ch: this->config.channels)
            channel_map[ch.synnax_key] = &ch;
    }

    xerrors::Error write(const synnax::Frame &fr) override {
        for (const auto &[key, series]: fr) {
            // O(1) lookup instead of O(n) linear search
            const auto it = channel_map.find(key);
            if (it == channel_map.end()) continue;
            if (series.size() == 0) continue;

            const auto *ch = it->second;
            // Get the latest value
            const auto value = series.at<double>(series.size() - 1);

            // Format the command using the template
            std::string command = ch->command_template;
            const size_t pos = command.find("{value}");
            if (pos == std::string::npos)
                return xerrors::Error(
                    "command template missing {value} placeholder: " + ch->command_template
                );

            command.replace(pos, 7, std::to_string(value));

            // Send the command
            size_t written;
            if (const auto err = session->write(
                    reinterpret_cast<const uint8_t *>(command.c_str()),
                    command.length(),
                    written
                ); err)
                return err;
        }

        return xerrors::NIL;
    }
};

}