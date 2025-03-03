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
#include <map>
#include <set>

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/ni/channel/channels.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/write_task.h"

namespace ni {
/// @brief WriteTaskConfig is the configuration for creating an NI Digital or Analog
/// Write Task.
struct WriteTaskConfig {
    /// @brief the key of the device the task is writing to.
    const std::string device_key;
    /// @brief the rate at which the task will publish the states of the outputs
    /// back to the Synnax cluster.
    const telem::Rate state_rate;
    /// @brief whether data saving is enabled for the task.
    const bool data_saving;
    /// @brief a map of command channel keys to the configurations for each output
    /// channel in the task.
    std::map<synnax::ChannelKey, std::unique_ptr<channel::Output>> channels;
    /// @brief the index channel keys for all the state channels. This is used
    /// to make sure we write correct timestamps for each state channel.
    std::set<synnax::ChannelKey> state_indexes_;
    /// @brief a map of channel keys to their index positions within the tasks
    /// write buffer. This map is only valid after apply() has been called on the
    /// configuration.
    std::unordered_map<synnax::ChannelKey, std::size_t> buf_indexes;

    /// @brief move constructor to deal with output channel unique pointers.
    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        device_key(other.device_key),
        state_rate(other.state_rate),
        data_saving(other.data_saving),
        channels(std::move(other.channels)),
        state_indexes_(std::move(other.state_indexes_)),
        buf_indexes(std::move(other.buf_indexes)) {
    }

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
    ): device_key(cfg.required<std::string>("device")),
       state_rate(telem::Rate(cfg.required<float>("state_rate"))),
       data_saving(cfg.optional<bool>("data_saving", false)) {
        cfg.iter("channels", [&](xjson::Parser &ch_cfg) {
            auto ch = channel::parse_output(ch_cfg);
            if (ch->enabled) this->channels[ch->cmd_ch_key] = std::move(ch);
        });
        if (channels.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        std::vector<synnax::ChannelKey> state_keys;
        state_keys.reserve(this->channels.size());
        std::unordered_map<synnax::ChannelKey, synnax::ChannelKey> state_to_cmd;
        size_t index = 0;
        for (const auto &[_, ch]: this->channels) {
            state_keys.push_back(ch->state_ch_key);
            state_to_cmd[ch->state_ch_key] = ch->cmd_ch_key;
            buf_indexes[ch->cmd_ch_key] = index++;
        }
        auto [dev, err] = client->hardware.retrieve_device(this->device_key);
        if (err) {
            cfg.field_err("device", "failed to retrieve device " + err.message());
            return;
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
            if (state_ch.index != 0) this->state_indexes_.insert(state_ch.index);
        }
    }

    /// @brief returns the configuration necessary for opening the writer
    /// to communicate state values back to Synnax.
    [[nodiscard]] std::vector<synnax::ChannelKey> state_channels() {
        std::vector<synnax::ChannelKey> keys(channels.size());
        for (const auto &[_, ch]: channels) keys.push_back(ch->state_ch_key);
        return keys;
    }

    [[nodiscard]] std::vector<synnax::ChannelKey> cmd_channels() {
        std::vector<synnax::ChannelKey> keys(channels.size());
        for (const auto &[_, ch]: channels) keys.push_back(ch->cmd_ch_key);
        return keys;
    }

    /// @brief returns the configuration necessary for opening a streamer to
    /// receive values form Synnax.
    [[nodiscard]] std::set<synnax::ChannelKey> state_indexes() {
        return this->state_indexes_;
    }

    /// @brief parses the task from the given configuration, returning an error
    /// if the task could not be parsed.
    static std::pair<WriteTaskConfig, xerrors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }

    /// @brief applies the configuration to the given DAQmx task.
    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) {
        for (const auto &[_, ch]: channels)
            if (const auto err = ch->apply(dmx, task_handle)) return err;
        return xerrors::NIL;
    }
};


/// @brief sink is passed to the command pipeline in order to receive incoming
/// data from Synnax, write it to the device, and update the state.
template<typename T>
class CommandTaskSink final : public common::Sink {
    const WriteTaskConfig cfg;
public:
    /// @brief constructs a CommandSink bound to the provided parent WriteTask.
    explicit CommandTaskSink(WriteTaskConfig &cfg):
        Sink(
            cfg.state_rate,
            cfg.state_indexes(),
            cfg.state_channels(),
            cfg.cmd_channels(),
            cfg.data_saving
        ),
        cfg(std::move(cfg)),
        buf(cfg.channels.size()) {
    }

private:
    /// @brief automatically infer the data type from the template parameter. This
    /// will either be UINT8_T or FLOAT64_T. We use this to appropriately cast
    /// the data read from the hardware.
    const telem::DataType data_type = telem::DataType::infer<T>();
    /// @brief the parent write task.
    /// @brief a pre-allocated write buffer that is flushed to the device every
    /// time a command is provided.
    std::vector<T> buf;
    std::unique_ptr<hardware::Writer<T>> hw_writer;

    xerrors::Error start() override {
        return this->hw_writer->start();
    }

    xerrors::Error stop() override {
        return this->hw_writer->stop();
    }

    /// @brief implements pipeline::Sink to write the incoming frame to the
    /// underlying hardware. If the values are successfully written, updates
    /// the write tasks state to match the output values.
    xerrors::Error write(const synnax::Frame &frame) override {
        if (frame.empty()) return xerrors::NIL;
        for (const auto &[key, series]: frame) {
            auto it = this->cfg.buf_indexes.find(key);
            if (it != this->cfg.buf_indexes.end())
                buf[it->second] = telem::cast<T>(series.at(-1));
        }
        if (const auto err = this->hw_writer->write(buf)) {
            if (daqmx::ANALOG_WRITE_OUT_OF_BOUNDS.matches(err)) {
                return xerrors::NIL;
            }
            return err;
        }

        std::lock_guard lock{this->chan_state_lock};
        for (const auto &[key, series]: frame) {
            const auto it = this->cfg.channels.find(key);
            if (it != this->cfg.channels.end()) {
                this->chan_state[it->second->state_ch_key] = it->second->state_ch.
                        data_type.cast(series.at(-1));
            }
        }
        return xerrors::NIL;
    }
};
}
