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
#include <vector>

#include "client/cpp/labjack/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"

#include "driver/common/write_task.h"
#include "driver/labjack/device/device.h"
#include "driver/labjack/labjack.h"

namespace driver::labjack {
/// @brief configuration for an output channel on a LabJack device.
struct OutputChan {
    /// @brief the parsed generated wire configuration for the channel.
    const ::synnax::labjack::BaseOutputChannel cfg;
    /// @brief whether the channel is enabled.
    const bool enabled;
    /// @brief the synnax channel object for the state channel.
    synnax::channel::Channel state_ch;

    explicit OutputChan(const ::synnax::labjack::BaseOutputChannel &cfg):
        cfg(cfg), enabled(!cfg.disabled) {}

    /// @brief binds cluster information about the channel after it has been
    /// externally fetched.
    void bind_remote_info(const synnax::channel::Channel &state_ch) {
        this->state_ch = state_ch;
    }
};

/// @brief the configuration for opening a write task.
struct WriteTaskConfig : ::synnax::labjack::WriteConfig {
    /// @brief configurations for the enabled channels, keyed by command channel.
    std::map<synnax::channel::Key, std::unique_ptr<OutputChan>> outputs;
    /// @brief the set of index channel keys for the state channels.
    /// Dynamically populated by querying the core.
    std::set<synnax::channel::Key> state_index_keys;

    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        ::synnax::labjack::WriteConfig(std::move(other)),
        outputs(std::move(other.outputs)),
        state_index_keys(std::move(other.state_index_keys)) {}

    WriteTaskConfig(const WriteTaskConfig &) = delete;

    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &parser
    ):
        ::synnax::labjack::WriteConfig(::synnax::labjack::WriteConfig::parse(parser)) {
        std::unordered_map<synnax::channel::Key, synnax::channel::Key> state_to_cmd;
        for (const auto &parsed: this->channels) {
            auto ch = std::visit(
                [](const auto &c) { return std::make_unique<OutputChan>(c); },
                parsed
            );
            if (!ch->enabled) continue;
            if (ch->cfg.cmd_channel == 0 || ch->cfg.state_channel == 0) {
                parser.field_err(
                    "channels",
                    "output channels must set cmd_channel and state_channel"
                );
                continue;
            }
            state_to_cmd[ch->cfg.state_channel] = ch->cfg.cmd_channel;
            this->outputs[ch->cfg.cmd_channel] = std::move(ch);
        }
        if (this->outputs.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        if (const auto err = client->devices.retrieve(this->device).second) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        std::vector<synnax::channel::Key> state_channels;
        state_channels.reserve(this->outputs.size());
        for (const auto &[_, ch]: this->outputs)
            state_channels.push_back(ch->cfg.state_channel);
        const auto [sy_channels, ch_err] = client->channels.retrieve(state_channels);
        if (ch_err) {
            parser.field_err(
                "channels",
                "failed to retrieve channels: " + ch_err.message()
            );
            return;
        }
        for (const auto &state_ch: sy_channels) {
            if (state_ch.index != 0) this->state_index_keys.insert(state_ch.index);
            auto &ch = this->outputs[state_to_cmd[state_ch.key]];
            ch->bind_remote_info(state_ch);
        }
    }

    /// @brief parses the configuration from the given Synnax task.
    /// @returns a configuration and error if one occurs. If x::errors::Error is not
    /// NIL, then validation failed and the configuration is invalid.
    static std::pair<WriteTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }

    /// @brief returns the list of state channels used in the task.
    [[nodiscard]] std::vector<synnax::channel::Channel> state_channels() const {
        std::vector<synnax::channel::Channel> chs;
        chs.reserve(this->outputs.size());
        for (const auto &[_, ch]: this->outputs)
            chs.push_back(ch->state_ch);
        return chs;
    }

    /// @brief returns the list of command channel keys used in the task.
    [[nodiscard]] std::vector<synnax::channel::Key> cmd_channels() const {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->outputs.size());
        for (const auto &[_, ch]: this->outputs)
            keys.push_back(ch->cfg.cmd_channel);
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
    x::errors::Error curr_dev_err = x::errors::NIL;

public:
    explicit WriteSink(const std::shared_ptr<device::Device> &dev, WriteTaskConfig cfg):
        Sink(
            cfg.state_rate,
            cfg.state_index_keys,
            cfg.state_channels(),
            cfg.cmd_channels(),
            cfg.data_saving_disabled
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
    x::errors::Error start() override { return this->write_curr_state_to_dev(); }

    x::errors::Error write_curr_state_to_dev() {
        /// pull all values to the initial state (which is the current state).
        this->reset_buffer(this->cfg.outputs.size());
        for (const auto &[_, ch]: this->cfg.outputs) {
            this->ports_buf.push_back(ch->cfg.port.c_str());
            this->values_buf.push_back(
                x::telem::cast<double>(this->chan_state[ch->cfg.state_channel])
            );
        }
        return this->write_buf_to_dev();
    }

    /// @brief flushes the current value buffer to the LabJack device, executing the
    /// write.
    x::errors::Error write_buf_to_dev() const {
        int err_addr = 0;
        auto locs = this->ports_buf;
        return this->dev->e_write_names(
            static_cast<int>(this->ports_buf.size()),
            locs.data(),
            this->values_buf.data(),
            &err_addr
        );
    }

    /// @brief implements pipeline::Sink to write to the LabJack device.
    x::errors::Error write(x::telem::Frame &frame) override {
        this->reset_buffer(this->cfg.outputs.size());
        for (const auto &[cmd_key, s]: frame)
            if (const auto it = this->cfg.outputs.find(cmd_key);
                it != this->cfg.outputs.end()) {
                const auto &ch = it->second;
                this->ports_buf.push_back(ch->cfg.port.c_str());
                this->values_buf.push_back(x::telem::cast<double>(s.at(-1)));
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
