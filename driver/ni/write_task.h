// Copyright 2024 Synnax Labs, Inc.
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
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/ni/channel/channels.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"

namespace ni {
using OutputChannelMap = std::map<synnax::ChannelKey, std::unique_ptr<channel::Output> >;
using CommandToStateMap = std::map<synnax::ChannelKey, synnax::ChannelKey>;

/// @brief WriteTaskConfig is the configuration for creating an NI Digital or Analog
/// Write Task.
struct WriteTaskConfig {
    /// @brief the key of the device the task is writing to.
    const std::string device_key;
    /// @brief the rate at which the task will publish the states of the outputs
    /// back to the Synnax cluster.
    const telem::Rate state_rate;
    /// @brief the buffer size to allocate for writing data to the device.
    const size_t buffer_size;
    /// @brief whether data saving is enabled for the task.
    const bool data_saving;
    /// @brief a map of command channel keys to the configurations for each output
    /// channel in the task.
    OutputChannelMap channels;
    /// @brief a map of command channel keys to the state channel keys for each
    /// channel in the task.
    CommandToStateMap cmd_to_state;
    /// @brief the index channel keys for all of the state channels. This is used
    /// to make sure we write correct timestamps for each state channel.
    std::set<synnax::ChannelKey> state_indexes;

    /// @brief move constructor to deeal with output channel unique pointers.
    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        device_key(other.device_key),
        state_rate(other.state_rate),
        buffer_size(other.buffer_size),
        data_saving(other.data_saving),
        channels(std::move(other.channels)),
        cmd_to_state(std::move(other.cmd_to_state)),
        state_indexes(std::move(other.state_indexes)) {
    }

    /// @brief delete copy constructor and copy assignment to prevent accidenal
    /// copies.
    WriteTaskConfig(const WriteTaskConfig &) = delete;

    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    explicit WriteTaskConfig(
        xjson::Parser &cfg,
        OutputChannelMap &&channels,
        CommandToStateMap &&cmd_to_state,
        std::set<synnax::ChannelKey> &&state_indexes,
        const size_t buffer_size
    ): device_key(cfg.required<std::string>("device")),
       state_rate(telem::Rate(cfg.required<float>("state_rate"))),
       buffer_size(buffer_size),
       data_saving(cfg.optional<bool>("data_saving", false)),
       channels(std::move(channels)),
       cmd_to_state(std::move(cmd_to_state)),
       state_indexes(std::move(state_indexes)) {
    }

    /// @brief returns the configuration necessary for opening the writer
    /// to communicate state values back to Synnax.
    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size() + state_indexes.size());
        for (const auto &[_, ch]: channels) keys.push_back(ch->state_ch_key);
        for (const auto &idx: state_indexes) keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .start = telem::TimeStamp::now(),
            .enable_auto_commit = true,
            .mode = synnax::data_saving_writer_mode(this->data_saving)
        };
    }

    /// @brief returns the configuration necessary for opening a streamer to
    /// receive values form Synnax.
    [[nodiscard]] synnax::StreamerConfig streamer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size());
        for (const auto &[_, ch]: channels) keys.push_back(ch->cmd_ch_key);
        return synnax::StreamerConfig{.channels = keys};
    }

    /// @brief parses the task from the given configuration, returning an error
    /// if the task could not be parsed.
    static std::pair<WriteTaskConfig, xerrors::Error> parse(
        std::shared_ptr<synnax::Synnax> client,
        synnax::Task task
    ) {
        auto parser = xjson::Parser(task.config);
        OutputChannelMap channels;
        bool is_digital = task.type == "ni_digital_write";
        parser.iter("channels", [&](xjson::Parser &ch_cfg) {
            std::unique_ptr<channel::Output> ch;
            if (is_digital) ch = std::make_unique<channel::DO>(ch_cfg);
            else ch = channel::parse_output(ch_cfg);
            if (ch->enabled) channels[ch->cmd_ch_key] = std::move(ch);
        });
        CommandToStateMap cmd_to_state;
        std::set<synnax::ChannelKey> state_indexes;
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size() * 2);
        for (const auto &[_, ch]: channels) {
            keys.push_back(ch->state_ch_key);
            cmd_to_state[ch->cmd_ch_key] = ch->state_ch_key;
        }
        auto buffer_size = channels.size();
        if (!is_digital) buffer_size *= telem::FLOAT64_T.density();
        auto [channels_vec, err] = client->channels.retrieve(keys);
        for (const auto &ch: channels_vec)
            if (ch.index != 0) state_indexes.insert(ch.key);
        return {
            WriteTaskConfig(
                parser,
                std::move(channels),
                std::move(cmd_to_state),
                std::move(state_indexes),
                buffer_size
            ),
            parser.error()
        };
    }

    /// @brief applies the configuration to the given DAQmx task.
    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const {
        if (const auto err = cycle_task_to_detect_cfg_errors(dmx, task_handle))
            return
                    err;
        for (const auto &[_, ch]: channels)
            if (const auto err = ch->apply(dmx, task_handle)) return err;
        return xerrors::NIL;
    }
};

inline xerrors::Error write_digital(
    const std::shared_ptr<SugaredDAQmx> &dmx,
    TaskHandle task_handle,
    const uint8_t *data
) {
    return dmx->WriteDigitalU8(
        task_handle,
        1,
        1,
        10.0,
        DAQmx_Val_GroupByChannel,
        data,
        nullptr,
        nullptr
    );
}

inline xerrors::Error write_analog(
    const std::shared_ptr<SugaredDAQmx> &dmx,
    TaskHandle task_handle,
    const double *data
) {
    return dmx->WriteAnalogF64(
        task_handle,
        1,
        1,
        10.0,
        DAQmx_Val_GroupByChannel,
        data,
        nullptr,
        nullptr
    );
}

template<typename T>
class WriteTask final : public task::Task {
    /// @brief the raw synnax task configuration.
    const synnax::Task task;
    /// @brief the configuration for the task.
    const WriteTaskConfig cfg;
    /// @brief the DAQmx interface used to communicate with the device.
    const std::shared_ptr<SugaredDAQmx> dmx;
    /// @brief the task context used to communicate state changes back to Synnax.
    std::shared_ptr<task::Context> ctx;
    /// @brief the pipeline used to receive commands from Synnax and write them to
    /// the device.
    pipeline::Control cmd_write_pipe;
    /// @brief the pipeline used to receive state changes from the device and write
    /// to Synnax.
    pipeline::Acquisition state_write_pipe;
    /// @brief the task handle for the DAQmx task.
    TaskHandle handle;
    /// @brief the current state of all the outputs. This is shared between
    /// the command sink and state source.
    std::unordered_map<synnax::ChannelKey, telem::SampleValue> channel_state;
    /// @brief used to lock concurrent access to the channel state.
    std::mutex channel_state_lock;
    /// @brief the current state of the task.
    task::State state;

public:
    explicit WriteTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        WriteTaskConfig cfg,
        const breaker::Config &breaker_cfg,
        TaskHandle handle
    ): task(std::move(task)),
       cfg(std::move(cfg)),
       ctx(ctx),
       cmd_write_pipe(
           this->ctx->client,
           this->cfg.streamer_config(),
           std::make_shared<CommandSink>(*this),
           breaker_cfg
       ),
       state_write_pipe(
           this->ctx->client,
           this->cfg.writer_config(),
           std::make_shared<StateSource>(*this),
           breaker_cfg
       ),
       handle(handle) {
    }

    ~WriteTask() override {
        /// the task should always be stopped before this gets destructed, so
        /// we only need to clear the task.
        this->dmx->ClearTask(this->handle);
    }

    /// @brief StateSource is passed to the state pipeline in order to continually
    /// communicate the current output states to Synnax.
    class StateSource final : public pipeline::Source {
        /// @brief the parent write task.
        WriteTask &task;
        /// @brief a timer that is used to control the rate at which the state is
        /// propagated.
        loop::Timer state_timer;

    public:
        explicit StateSource(WriteTask &task)
            : task(task), state_timer(task.cfg.state_rate) {
        }

        std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
            this->state_timer.wait(breaker);
            std::lock_guard{this->task.channel_state_lock};
            auto fr = synnax::Frame(
                this->task.channel_state,
                this->task.channel_state.size() + this->task.cfg.state_indexes.size()
            );
            if (!this->task.cfg.state_indexes.empty()) {
                const auto idx_ser = telem::Series(telem::TimeStamp::now());
                for (const auto idx: this->task.cfg.state_indexes)
                    fr.emplace(idx, idx_ser.deep_copy());
            }
            return {std::move(fr), xerrors::NIL};
        }
    };

    /// @brief sink is passed to the command pipeline in order to receive incoming
    /// data from Synnax, write it to the device, and update the state.
    class CommandSink final : public pipeline::Sink {
        /// @brief the parent write task.
        WriteTask &task;
        /// @brief a pre-allocated write buffer that is flushed to the device every
        /// time a command is provided.
        std::unique_ptr<T> write_buffer = nullptr;

    public:
        explicit CommandSink(WriteTask &task): task(task) {
        }

        xerrors::Error write(const synnax::Frame &frame) override {
            auto data = this->format_data(frame);
            if (this->task.task.type == "ni_digital_write")
                write_digital(this->task.dmx, this->task.handle,
                              reinterpret_cast<uint8_t *>(data));
            else if (this->task.task.type == "ni_analog_write")
                write_analog(this->task.dmx, this->task.handle,
                             reinterpret_cast<double *>(data));
            std::lock_guard{this->task.channel_state_lock};
            for (const auto &[key, series]: frame) {
                const auto state_key = this->task.cfg.cmd_to_state.at(key);
                this->task.channel_state[state_key] = series.at(0);
            }
            return xerrors::NIL;
        }

        T *format_data(const synnax::Frame &frame) {
            for (const auto &[key, series]: frame) {
                auto it = this->task.cfg.channels.find(key);
                if (it == this->task.cfg.channels.end()) continue;
                auto buf = this->write_buffer.get();
                buf[it->second->index] = telem::cast_numeric_sample_value<T>(
                    series.at_numeric(0)
                );
            }
            return this->write_buffer.get();
        }
    };

    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key);
    }

    void stop() override { this->stop(""); }

    void stop(const std::string &cmd_key) {
        if (!this->state.details["running"]) return;
        this->state.key = cmd_key;
        this->cmd_write_pipe.stop();
        this->state_write_pipe.stop();
        this->state.details["running"] = false;
        if (const auto err = this->dmx->StopTask(this->handle)) {
            this->state.variant = "error";
            this->state.details["message"] = err.message();
        } else {
            this->state.variant = "success";
            this->state.details["message"] = "Task stopped successfully";
        }
        this->ctx->set_state(this->state);
    }

    void start(const std::string &cmd_key) {
        if (this->state.details["running"]) return;
        this->state.key = cmd_key;
        if (const auto err = this->dmx->StartTask(this->handle)) {
            this->state.variant = "error";
            this->state.details["message"] = err.message();
        } else {
            this->cmd_write_pipe.start();
            this->state_write_pipe.start();
            this->state.variant = "success";
            this->state.details["message"] = "Task started successfully";
            this->state.details["running"] = true;
        }
        this->ctx->set_state(this->state);
    }

    std::string name() override { return task.name; }

    static std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        auto [cfg, parse_err] = WriteTaskConfig::parse(ctx->client, task);
        if (parse_err) return {nullptr, parse_err};

        TaskHandle task_handle;
        if (const auto err = dmx->CreateTask("", &task_handle)) return {nullptr, err};
        if (const auto err = cfg.apply(dmx, task_handle)) return {nullptr, err};
        if (const auto err = cycle_task_to_detect_cfg_errors(dmx, task_handle))
            return {nullptr, err};
        return {
            std::make_unique<WriteTask>(
                ctx,
                task,
                std::move(cfg),
                breaker::default_config(task.name),
                task_handle
            ),
            xerrors::NIL
        };
    }
};
}
