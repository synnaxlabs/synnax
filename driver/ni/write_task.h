// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <set>
#include <string>

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/ni/channel/channels.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/ni/ni.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/common.h"
#include "driver/task/common/write_task.h"

namespace ni {
/// @brief WriteTaskConfig is the configuration for creating an NI Write Task.
struct WriteTaskConfig : common::BaseWriteTaskConfig {
    /// @brief the rate at which the task will publish the states of the outputs
    /// back to the Synnax cluster.
    const telem::Rate state_rate;
    /// @brief a map of command channel keys to the configurations for each output
    /// channel in the task that supports runtime control (AO, DO).
    std::map<synnax::ChannelKey, std::unique_ptr<channel::Output>> channels;
    /// @brief configuration-only channels that don't support runtime control
    /// (e.g., CO Pulse Output). These are applied to the DAQmx task but don't
    /// participate in the command/state pipeline.
    std::vector<std::unique_ptr<channel::Output>> config_only_channels;
    /// @brief the index channel keys for all the state channels. This is used
    /// to make sure we write correct timestamps for each state channel.
    std::set<synnax::ChannelKey> state_index_keys;
    /// @brief a map of channel keys to their index positions within the tasks
    /// write buffer. This map is only valid after apply() has been called on the
    /// configuration.
    std::unordered_map<synnax::ChannelKey, std::size_t> buf_indexes;

    /// @brief move constructor to deal with output channel unique pointers.
    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        common::BaseWriteTaskConfig(std::move(other)),
        state_rate(other.state_rate),
        channels(std::move(other.channels)),
        config_only_channels(std::move(other.config_only_channels)),
        state_index_keys(std::move(other.state_index_keys)),
        buf_indexes(std::move(other.buf_indexes)) {}

    /// @brief delete copy constructor and copy assignment to prevent accidental
    /// copies.
    WriteTaskConfig(const WriteTaskConfig &) = delete;

    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    /// @brief constructs the configuration from the provided JSON parser, using the
    /// client to fetch any remote data from the Synnax cluster.
    /// @param client - Synnax client used to fetch remote data from the cluster.
    /// @param cfg - The JSON configuration for the task.
    /// @details any errors encountered while parsing the configuration will be
    /// added as field errors to the provided parser. The caller should use
    /// cfg.error() after this constructor in order to check for these errors.
    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ):
        common::BaseWriteTaskConfig(cfg),
        state_rate(telem::Rate(cfg.required<float>("state_rate"))) {
        cfg.iter("channels", [&](xjson::Parser &ch_cfg) {
            auto ch = channel::parse_output(ch_cfg);
            if (ch != nullptr && ch->enabled) {
                // Separate channels that support runtime control from config-only
                // channels
                if (ch->supports_runtime_control() && ch->cmd_ch_key != 0) {
                    this->channels[ch->cmd_ch_key] = std::move(ch);
                } else {
                    // Config-only channels (e.g., CO Pulse Output) don't participate
                    // in command/state pipeline but still need to be applied to task
                    this->config_only_channels.push_back(std::move(ch));
                }
            }
        });
        if (channels.empty() && config_only_channels.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        auto [dev, err] = client->hardware.retrieve_device(this->device_key);
        if (err) {
            cfg.field_err("device", "failed to retrieve device " + err.message());
            return;
        }

        // Only fetch state channels if we have runtime-controlled channels
        if (!channels.empty()) {
            std::vector<synnax::ChannelKey> state_keys;
            state_keys.reserve(this->channels.size());
            std::unordered_map<synnax::ChannelKey, synnax::ChannelKey> state_to_cmd;
            size_t index = 0;
            for (const auto &[_, ch]: this->channels) {
                state_keys.push_back(ch->state_ch_key);
                state_to_cmd[ch->state_ch_key] = ch->cmd_ch_key;
                buf_indexes[ch->cmd_ch_key] = index++;
            }
            auto [state_channels, ch_err] = client->channels.retrieve(state_keys);
            if (ch_err) {
                cfg.field_err(
                    "channels",
                    "failed to retrieve state channels: " + ch_err.message()
                );
                return;
            }
            for (const auto &state_ch: state_channels) {
                auto &ch = this->channels[state_to_cmd[state_ch.key]];
                ch->bind_remote_info(state_ch, dev.location);
                if (state_ch.index != 0) this->state_index_keys.insert(state_ch.index);
            }
        }

        // Bind device location to config-only channels (they don't have state channels)
        for (auto &ch: config_only_channels) {
            ch->dev_loc = dev.location;
        }
    }

    /// @brief returns the configuration necessary for opening the writer
    /// to communicate state values back to Synnax.
    [[nodiscard]] std::vector<synnax::Channel> state_channels() {
        std::vector<synnax::Channel> state_channels;
        state_channels.reserve(this->channels.size());
        for (const auto &[_, ch]: this->channels)
            state_channels.push_back(ch->state_ch);
        return state_channels;
    }

    [[nodiscard]] std::vector<synnax::ChannelKey> cmd_channels() {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &[_, ch]: this->channels)
            keys.push_back(ch->cmd_ch_key);
        return keys;
    }

    /// @brief returns the configuration necessary for opening a streamer to
    /// receive values form Synnax.
    [[nodiscard]] std::set<synnax::ChannelKey> state_indexes() {
        return this->state_index_keys;
    }

    /// @brief parses the task from the given configuration, returning an error
    /// if the task could not be parsed.
    static std::pair<WriteTaskConfig, xerrors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task,
        /// We include this ignored parameter to make the parse method have the
        /// same signature as the read task, so we can save code duplication in
        /// the factory.
        common::TimingConfig
    ) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }

    /// @brief applies the configuration to the given DAQmx task.
    xerrors::Error
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx, TaskHandle task_handle) {
        // Apply runtime-controlled channels
        for (const auto &[_, ch]: channels)
            if (const auto err = ch->apply(dmx, task_handle)) return err;
        // Apply config-only channels (e.g., CO Pulse Output)
        for (const auto &ch: config_only_channels)
            if (const auto err = ch->apply(dmx, task_handle)) return err;
        return xerrors::NIL;
    }
};

/// @brief sink is passed to the command pipeline in order to receive incoming
/// data from Synnax, write it to the device, and update the state.
template<typename T>
class WriteTaskSink final : public common::Sink {
    const WriteTaskConfig cfg;

public:
    /// @brief constructs a CommandSink bound to the provided parent WriteTask.
    explicit WriteTaskSink(
        WriteTaskConfig cfg,
        std::unique_ptr<hardware::Writer<T>> hw_writer
    ):
        Sink(
            cfg.state_rate,
            cfg.state_indexes(),
            cfg.state_channels(),
            cfg.cmd_channels(),
            cfg.data_saving
        ),
        cfg(std::move(cfg)),
        hw_writer(std::move(hw_writer)),
        buf(this->cfg.channels.size()) {}

private:
    /// @brief the underlying DAQmx hardware we write data to.
    const std::unique_ptr<hardware::Writer<T>> hw_writer;
    /// @brief the parent write task.
    /// @brief a pre-allocated write buffer that is flushed to the device every
    /// time a command is provided.
    std::vector<T> buf;

    /// @brief implements common::Task to start the hardware writer.
    xerrors::Error start() override { return this->hw_writer->start(); }

    /// @brief implements common::Task to stop the hardware writer.
    xerrors::Error stop() override { return this->hw_writer->stop(); }

    /// @brief implements pipeline::Sink to write the incoming frame to the
    /// underlying hardware. If the values are successfully written, updates
    /// the write tasks state to match the output values.
    xerrors::Error write(const synnax::Frame &frame) override {
        for (const auto &[cmd_key, series]: frame)
            if (auto it = this->cfg.buf_indexes.find(cmd_key);
                it != this->cfg.buf_indexes.end()) {
                const auto buf_index = it->second;
                buf[buf_index] = telem::cast<T>(series.at(-1));
            }
        if (const auto err = this->hw_writer->write(buf)) return translate_error(err);
        this->set_state(frame);
        return xerrors::NIL;
    }
};
}
