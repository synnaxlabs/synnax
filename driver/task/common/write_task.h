// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/loop/loop.h"

#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/status.h"
#include "driver/task/task.h"

namespace common {
/// @brief common write task configuration parameters used across multiple drivers.
struct BaseWriteTaskConfig : BaseTaskConfig {
    /// @brief the key of the device the task is writing to.
    const std::string device_key;

    BaseWriteTaskConfig(BaseWriteTaskConfig &&other) noexcept:
        BaseTaskConfig(std::move(other)), device_key(other.device_key) {}

    BaseWriteTaskConfig(const BaseWriteTaskConfig &) = delete;

    const BaseWriteTaskConfig &operator=(const BaseWriteTaskConfig &) = delete;

    explicit BaseWriteTaskConfig(xjson::Parser &cfg):
        BaseTaskConfig(cfg), device_key(cfg.field<std::string>("device")) {}
};
class Sink : public pipeline::Sink, public pipeline::Source {
    /// @brief the vector of channels to stream for commands.
    const std::vector<synnax::ChannelKey> cmd_channels;
    /// @brief the vector of channels to write state updates for.
    std::unordered_map<synnax::ChannelKey, synnax::Channel> state_channels;
    /// @brief the index keys of the state channels.
    const std::set<synnax::ChannelKey> state_indexes;
    /// @brief whether data saving is enabled for the task.
    bool data_saving;

public:
    /// @brief the rate at which to communicate state values down the channel.
    telem::Rate state_rate;
    /// @brief used to lock concurrent access to the channel state.
    std::mutex chan_state_lock;
    /// @brief used to signal the state source to send values whenever a command
    /// has been executed.
    std::condition_variable chan_state_cv;
    /// @brief the current state of all the outputs. This is shared between
    /// the command sink and state source.
    std::unordered_map<synnax::ChannelKey, telem::SampleValue> chan_state;

    explicit Sink(std::vector<synnax::ChannelKey> cmd_channels):
        cmd_channels(std::move(cmd_channels)),
        state_indexes({}),
        data_saving(true),
        state_rate(0) {}

    Sink(
        const telem::Rate state_rate,
        std::set<synnax::ChannelKey> state_indexes,
        const std::vector<synnax::Channel> &state_channels,
        std::vector<synnax::ChannelKey> cmd_channels,
        const bool data_saving
    ):
        cmd_channels(std::move(cmd_channels)),
        state_indexes(std::move(state_indexes)),
        data_saving(data_saving),
        state_rate(state_rate) {
        auto idx = 0;
        for (const auto &ch: state_channels) {
            this->chan_state[ch.key] = ch.data_type.cast(0);
            this->state_channels[this->cmd_channels[idx]] = ch;
            idx++;
        }
    }

    virtual xerrors::Error start() { return xerrors::NIL; }

    virtual xerrors::Error stop() { return xerrors::NIL; }

    [[nodiscard]] synnax::StreamerConfig streamer_config() const {
        return synnax::StreamerConfig{.channels = this->cmd_channels};
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        for (const auto &[_, ch]: this->state_channels)
            keys.push_back(ch.key);
        for (const auto idx: this->state_indexes)
            keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .mode = common::data_saving_writer_mode(this->data_saving),
        };
    }

    void set_state(const telem::Frame &frame) {
        std::lock_guard lock{this->chan_state_lock};
        this->chan_state_cv.notify_all();
        for (const auto &[cmd_key, s]: frame) {
            const auto it = this->state_channels.find(cmd_key);
            if (it == this->state_channels.end()) continue;
            const auto state_key = it->second.key;
            this->chan_state[state_key] = it->second.data_type.cast(s.at(-1));
        }
    }

    xerrors::Error read(breaker::Breaker &breaker, telem::Frame &fr) override {
        std::unique_lock lock(chan_state_lock);
        this->chan_state_cv.wait_for(lock, this->state_rate.period().chrono());
        fr.clear();
        fr.reserve(this->chan_state.size());
        for (const auto &[key, value]: this->chan_state)
            fr.emplace(key, telem::Series(value));
        const auto now = telem::TimeStamp::now();
        for (const auto idx: this->state_indexes)
            fr.emplace(idx, telem::Series(now));
        return xerrors::NIL;
    }
};

/// @brief a write task that can write to both digital and analog output channels,
/// and communicate their state back to Synnax.
class WriteTask final : public task::Task {
    class WrappedSink final : public pipeline::Sink, public pipeline::Source {
    public:
        /// @brief the parent write task.
        WriteTask &p;
        /// @brief the underlying wrapped sink that actually executes commands on
        /// the hardware.
        std::unique_ptr<common::Sink> internal;

        WrappedSink(WriteTask &p, std::unique_ptr<common::Sink> sink):
            p(p), internal(std::move(sink)) {}

        /// @brief implements pipeline::Sink, and pipeline:Source
        void stopped_with_err(const xerrors::Error &err) override {
            this->p.state.error(err);
            this->p.stop("", true);
        }

        xerrors::Error read(breaker::Breaker &breaker, telem::Frame &fr) override {
            return this->internal->read(breaker, fr);
        }

        xerrors::Error write(const telem::Frame &frame) override {
            if (frame.empty()) return xerrors::NIL;
            auto err = this->internal->write(frame);
            if (!err)
                this->p.state.clear_warning();
            else if (err.matches(driver::TEMPORARY_HARDWARE_ERROR))
                this->p.state.send_warning(err);
            return err;
        }

        [[nodiscard]] synnax::WriterConfig writer_config() const {
            auto cfg = this->internal->writer_config();
            if (cfg.subject.name.empty()) cfg.subject.name = this->p.name();
            return cfg;
        }
    };

    /// @brief used to manage and communicate the task's state.
    StatusHandler state;
    /// @brief the hardware interface for writing data
    std::shared_ptr<WrappedSink> sink;
    /// @brief the pipeline used to receive commands from Synnax and write them to
    /// the device.
    pipeline::Control cmd_write_pipe;
    /// @brief the pipeline used to receive state changes from the device and write
    /// to Synnax.
    pipeline::Acquisition state_write_pipe;

public:
    /// @brief base constructor that takes in pipeline factories to allow the
    /// caller to stub cluster communication during tests.
    explicit WriteTask(
        const synnax::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<Sink> sink,
        const std::shared_ptr<pipeline::WriterFactory> &writer_factory,
        const std::shared_ptr<pipeline::StreamerFactory> &streamer_factory
    ):
        state(ctx, task),
        sink(std::make_shared<WrappedSink>(*this, std::move(sink))),
        cmd_write_pipe(
            streamer_factory,
            this->sink->internal->streamer_config(),
            this->sink,
            breaker_cfg,
            task.name
        ),
        state_write_pipe(
            writer_factory,
            this->sink->writer_config(),
            this->sink,
            breaker_cfg,
            task.name + ":state"
        ) {}

    /// @brief primary constructor that uses the task context's Synnax client for
    /// cluster communication.
    explicit WriteTask(
        const synnax::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<Sink> sink
    ):
        WriteTask(
            task,
            ctx,
            breaker_cfg,
            std::move(sink),
            std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client),
            std::make_shared<pipeline::SynnaxStreamerFactory>(ctx->client)
        ) {}

    /// @brief implements task::Task to execute the provided command on the task.
    void exec(task::Command &cmd) override {
        if (cmd.type == "start")
            this->start(cmd.key);
        else if (cmd.type == "stop")
            this->stop(cmd.key, true);
    }

    /// @brief implements task::Task to stop the task.
    void stop(const bool will_reconfigure) override {
        this->stop("", !will_reconfigure);
    }

    /// @brief stops the task.
    /// @param cmd_key - A reference to the command key used to execute the stop.
    /// Will be used internally to communicate the task state.
    /// @param propagate_state whether the task will be reconfigured after it was
    /// stopped.
    bool stop(const std::string &cmd_key, const bool propagate_state) {
        const auto write_pipe_stopped = this->cmd_write_pipe.stop();
        const auto state_pipe_stopped = this->state_write_pipe.stop();
        const auto stopped = write_pipe_stopped && state_pipe_stopped;
        if (stopped) this->state.error(this->sink->internal->stop());
        if (propagate_state) this->state.send_stop(cmd_key);
        return stopped;
    }

    /// @brief starts the task.
    /// @param cmd_key - A reference to the command key used to execute the start.
    /// Will be used internally to communicate the task state.
    bool start(const std::string &cmd_key) {
        this->stop("", false);
        const auto sink_started = !this->state.error(this->sink->internal->start());
        if (sink_started) {
            this->cmd_write_pipe.start();
            if (!this->sink->internal->writer_config().channels.empty())
                this->state_write_pipe.start();
        }
        this->state.send_start(cmd_key);
        return sink_started;
    }

    /// @brief implements task::Task to return the task's name.
    std::string name() const override { return this->state.task.name; }
};
}
