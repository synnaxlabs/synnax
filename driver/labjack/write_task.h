// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/labjack/device/device.h"
#include "driver/labjack/labjack.h"
#include "driver/task/common/write_task.h"

namespace labjack {
/// @brief configuration for an output channel on a LabJack device.
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

    explicit OutputChan(xjson::Parser &parser):
        port(parser.optional<std::string>("port", "")),
        enabled(parser.optional<bool>("enabled", true)),
        cmd_ch_key(parser.required<uint32_t>("cmd_key", "cmd_channel")),
        state_ch_key(parser.required<uint32_t>("state_key", "state_channel")) {}

    /// @brief binds cluster information about the channel after it has been
    /// externally fetched.
    void bind_remote_info(const synnax::Channel &state_ch) {
        this->state_ch = state_ch;
    }
};

/// @brief the configuration for opening a write task.
struct WriteTaskConfig : common::BaseWriteTaskConfig {
    /// @brief the rate at which to propagate state updates back to Synnax.
    const telem::Rate state_rate;
    /// @brief the connection method to the device.
    const std::string conn_method;
    /// @brief the model of the device.
    std::string dev_model;
    /// @brief configurations for the enabled channels on the device.
    std::map<synnax::ChannelKey, std::unique_ptr<OutputChan>> channels;
    /// @brief the set of index channel keys for the state channels.
    std::set<synnax::ChannelKey> state_index_keys;

    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        common::BaseWriteTaskConfig(std::move(other)),
        state_rate(other.state_rate),
        conn_method(other.conn_method),
        dev_model(std::move(other.dev_model)),
        channels(std::move(other.channels)),
        state_index_keys(std::move(other.state_index_keys)) {}

    WriteTaskConfig(const WriteTaskConfig &) = delete;

    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &parser
    ):
        common::BaseWriteTaskConfig(parser),
        state_rate(telem::Rate(parser.optional<int>("state_rate", 1))),
        conn_method(parser.optional<std::string>("connection_type", "")) {
        std::unordered_map<synnax::ChannelKey, synnax::ChannelKey> state_to_cmd;
        parser.iter("channels", [this, &state_to_cmd](xjson::Parser &p) {
            auto ch = std::make_unique<OutputChan>(p);
            if (!ch->enabled) return;
            state_to_cmd[ch->state_ch_key] = ch->cmd_ch_key;
            this->channels[ch->cmd_ch_key] = std::move(ch);
        });
        if (this->channels.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        auto [dev, err] = client->hardware.retrieve_device(this->device_key);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        this->dev_model = dev.model;
        std::vector<synnax::ChannelKey> state_channels;
        state_channels.reserve(this->channels.size());
        for (const auto &[_, ch]: this->channels)
            state_channels.push_back(ch->state_ch_key);
        const auto [channels, ch_err] = client->channels.retrieve(state_channels);
        if (ch_err) {
            parser.field_err(
                "channels",
                "failed to retrieve channels: " + ch_err.message()
            );
            return;
        }
        for (const auto &state_ch: channels) {
            if (state_ch.index != 0) this->state_index_keys.insert(state_ch.index);
            auto &ch = this->channels[state_to_cmd[state_ch.key]];
            ch->bind_remote_info(state_ch);
        }
    }

    /// @brief parses the configuration from the given Synnax task.
    /// @returns a configuration and error if one occurs. If xerrors::Error is not
    /// NIL, then validation failed and the configuration is invalid.
    static std::pair<WriteTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }

    /// @brief returns the list of state channels used in the task.
    [[nodiscard]] std::vector<synnax::Channel> state_channels() const {
        std::vector<synnax::Channel> state_channels;
        state_channels.reserve(this->channels.size());
        for (const auto &[_, ch]: this->channels)
            state_channels.push_back(ch->state_ch);
        return state_channels;
    }

    /// @brief returns the list of command channel keys used in the task.
    [[nodiscard]] std::vector<synnax::ChannelKey> cmd_channels() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &[_, ch]: this->channels)
            keys.push_back(ch->cmd_ch_key);
        return keys;
    }
};

/// @brief an implementation of the common task sink that writes data to a LabJack
/// device.
class WriteSink final : public common::Sink {
    /// @brief the configuration for the sink.
    const WriteTaskConfig cfg;
    /// @brief the API of the device we're writing to.
    const std::shared_ptr<device::Device> dev;
    /// @brief the buffer of ports to use for the next write.
    std::vector<const char *> ports_buf;
    /// @brief the buffer of values to use for the next write.
    std::vector<double> values_buf;
    /// @brief the most recent error accumulated from writing to the device.
    /// Primarily used to track when the device has recovered from an error.
    xerrors::Error curr_dev_err = xerrors::NIL;

public:
    explicit WriteSink(const std::shared_ptr<device::Device> &dev, WriteTaskConfig cfg):
        Sink(
            cfg.state_rate,
            cfg.state_index_keys,
            cfg.state_channels(),
            cfg.cmd_channels(),
            cfg.data_saving
        ),
        cfg(std::move(cfg)),
        dev(dev) {}

    /// @brief clears the current write port and values buffer and re-reserves it
    /// to the allocated size.
    void reset_buffer(const size_t alloc) {
        this->ports_buf.clear();
        this->values_buf.clear();
        this->ports_buf.reserve(alloc);
        this->values_buf.reserve(alloc);
    }

    /// @brief starts the sink, pulling values to their initial state.
    xerrors::Error start() override { return this->write_curr_state_to_dev(); }

    xerrors::Error write_curr_state_to_dev() {
        /// pull all values to the initial state (which is the current state).
        this->reset_buffer(this->cfg.channels.size());
        for (const auto &[_, ch]: this->cfg.channels) {
            this->ports_buf.push_back(ch->port.c_str());
            this->values_buf.push_back(
                telem::cast<double>(this->chan_state[ch->state_ch_key])
            );
        }
        return this->write_buf_to_dev();
    }

    /// @brief flushes the current value buffer to the labjack device, executing the
    /// write.
    xerrors::Error write_buf_to_dev() const {
        int err_addr = 0;
        auto locs = this->ports_buf;
        return this->dev->e_write_names(
            static_cast<int>(this->ports_buf.size()),
            locs.data(),
            this->values_buf.data(),
            &err_addr
        );
    }

    /// @brief implements pipeline::Sink to write to the Labjack device.
    xerrors::Error write(const synnax::Frame &frame) override {
        this->reset_buffer(this->cfg.channels.size());
        for (const auto &[cmd_key, s]: frame)
            if (const auto it = this->cfg.channels.find(cmd_key);
                it != this->cfg.channels.end()) {
                const auto &ch = it->second;
                this->ports_buf.push_back(ch->port.c_str());
                this->values_buf.push_back(telem::cast<double>(s.at(-1)));
            }
        const auto prev_flush_err = this->curr_dev_err;
        this->curr_dev_err = translate_error(this->write_buf_to_dev());
        if (this->curr_dev_err) return this->curr_dev_err;
        this->set_state(frame);
        // This means we just recovered from a temporary error, in which case
        // we should flush the entirety of the current state to the device so that
        // it matches our internal state again.
        if (prev_flush_err)
            this->curr_dev_err = translate_error(this->write_curr_state_to_dev());
        return this->curr_dev_err;
    }
};
}
