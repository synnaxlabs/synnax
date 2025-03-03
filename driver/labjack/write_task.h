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
#include <string>
#include <vector>

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/labjack/ljm/device_manager.h"
#include "driver/task/common/write_task.h"

namespace labjack {
struct OutputChan {
    /// @brief the port location of the output channel e.g. "DIO4"
    const std::string port;
    /// @brief whether the channel is enabled.
    const bool enabled;
    /// @brief the key of the synnax channel to receive commands from.
    const synnax::ChannelKey cmd_ch_key;
    //// @brief the key fo the synnax channel to propagate state changes to.
    const synnax::ChannelKey state_ch_key;
    /// @brief the synnax channel object for the state channel.
    synnax::Channel state_ch;

    explicit OutputChan(xjson::Parser &parser)
        : port(parser.optional<std::string>("port", "")),
          enabled(parser.optional<bool>("enabled", true)),
          cmd_ch_key(parser.required<uint32_t>("cmd_key")),
          state_ch_key(parser.required<uint32_t>("state_key")) {
    }

    void bind_remote_info(const synnax::Channel &state_ch) {
        this->state_ch = state_ch;
    }
};

struct WriteTaskConfig {
    /// @brief whether data saving is enabled for the task.
    const bool data_saving;
    /// @brief the device key to write to.
    const std::string device_key;
    /// @brief the rate at which to propagate state updates back to Synnax.
    const telem::Rate state_rate;
    /// @brief the connection method to the device.
    const std::string conn_method;
    /// @brief the model of the device.
    std::string dev_model;
    /// @brief configurations for the enabled channels on the device.
    std::unordered_map<synnax::ChannelKey, std::unique_ptr<OutputChan>> channels;
    /// @brief the set of index channel keys for the state channels.
    std::set<synnax::ChannelKey> state_index_keys;

    WriteTaskConfig(
        WriteTaskConfig &&other
    ) noexcept: data_saving(other.data_saving),
                device_key(other.device_key),
                state_rate(other.state_rate),
                conn_method(other.conn_method),
                dev_model(std::move(other.dev_model)),
                channels(std::move(other.channels)),
                state_index_keys(std::move(other.state_index_keys)) {
    }

    WriteTaskConfig(const WriteTaskConfig &) = delete;

    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &parser
    ): data_saving(parser.optional<bool>("data_saving", false)),
       device_key(parser.required<std::string>("device")),
       state_rate(telem::Rate(parser.optional<int>("state_rate", 1))),
       conn_method(parser.optional<std::string>("connection_type", "")) {
        std::unordered_map<synnax::ChannelKey, synnax::ChannelKey> state_to_cmd;
        parser.iter(
            "channels",
            [this, &state_to_cmd](xjson::Parser &p) {
                auto ch = std::make_unique<OutputChan>(p);
                if (!ch->enabled) return;
                this->channels[ch->cmd_ch_key] = std::move(ch);
                state_to_cmd[ch->state_ch_key] = ch->cmd_ch_key;
            }
        );
        auto [dev, err] = client->hardware.retrieve_device(this->device_key);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        this->dev_model = dev.model;
        std::vector<synnax::ChannelKey> state_channels;
        state_channels.reserve(this->channels.size());
        for (const auto &[_, ch]: this->channels) state_channels.push_back(ch->state_ch_key);
        const auto [channels, ch_err] = client->channels.retrieve(state_channels);
        if (ch_err) {
            parser.field_err("channels",
                             "failed to retrieve channels: " + ch_err.message());
            return;
        }
        for (const auto &state_ch: channels) {
            if (state_ch.index != 0) this->state_index_keys.insert(state_ch.key);
            auto &ch = this->channels[state_to_cmd[state_ch.key]];
            ch->bind_remote_info(state_ch);
        }
    }

    static std::pair<WriteTaskConfig, xerrors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }

    [[nodiscard]] std::vector<synnax::Channel> state_channels() const {
        std::vector<synnax::Channel> state_channels;
        state_channels.reserve(this->state_index_keys.size());
        for (const auto &[_, ch]: this->channels) state_channels.push_back(ch->state_ch);
        return state_channels;
    }

    [[nodiscard]] std::vector<synnax::ChannelKey> cmd_channels() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &[_, ch]: this->channels) keys.push_back(ch->cmd_ch_key);
        return keys;
    }
};

class WriteSink final : public common::Sink {
    const WriteTaskConfig cfg;
    std::shared_ptr<ljm::DeviceAPI> dev;

public:
    explicit WriteSink(
        const std::shared_ptr<ljm::DeviceAPI> &dev,
        WriteTaskConfig cfg
    ): Sink(
           cfg.state_rate,
           cfg.state_index_keys,
           cfg.state_channels(),
           cfg.cmd_channels(),
           cfg.data_saving
       ),
       cfg(std::move(cfg)),
       dev(dev) {
    }

    xerrors::Error start() override {
        std::vector<const char *> locs;
        std::vector<double> values;
        locs.reserve(this->cfg.channels.size());
        values.reserve(this->cfg.channels.size());
        for (const auto &[_, ch]: this->cfg.channels) {
            locs.push_back(ch->port.c_str());
            values.push_back(0);
        }
        return this->write(locs, values);
    }

    xerrors::Error stop() override {
        return xerrors::NIL;
    }

    xerrors::Error write(
        std::vector<const char *> &locs,
        const std::vector<double> &values
    ) const {
        return this->dev->eWriteNames(
            static_cast<int>(locs.size()),
            locs.data(),
            values.data(),
            nullptr
        );
    }

    xerrors::Error write(const synnax::Frame &frame) override {
        if (frame.empty()) return xerrors::NIL;
        std::vector<const char *> locs;
        std::vector<double> values;
        locs.reserve(frame.size());
        values.reserve(frame.size());
        for (const auto &[key, s]: frame) {
            auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end()) continue;
            const auto &ch = it->second;
            locs.push_back(ch->port.c_str());
            values.push_back(telem::cast<double>(s.at(-1)));
        }
        if (const auto err = this->write(locs, values)) return err;
        std::lock_guard lock{this->chan_state_lock};
        for (const auto &[key, s]: frame) {
            const auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end()) continue;
            const auto state_ch = it->second->state_ch;
            this->chan_state[state_ch.key] = state_ch.data_type.cast(s.at(-1));
        }
        return xerrors::NIL;
    }
};
}
