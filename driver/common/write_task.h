// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/task/json.gen.h"

#include "driver/bypass/pipeline/factory.h"
#include "driver/common/common.h"
#include "driver/common/status.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"

namespace driver::common {
/// @brief common write task configuration shared across hardware control tasks.
/// Wraps the schema-generated write config (auto_start, data_saving_disabled,
/// device) so the field set has a single definition in the oracle schema.
struct BaseWriteTaskConfig : ::synnax::task::WriteConfig {
    explicit BaseWriteTaskConfig(x::json::Parser &cfg):
        ::synnax::task::WriteConfig(::synnax::task::WriteConfig::parse(cfg)) {
        if (this->device.empty()) cfg.field_err("device", "this field is required");
    }
};

class Sink : public pipeline::Sink, public pipeline::Source {
public:
    /// @brief an output of a write task.
    struct Output {
        /// @brief the channel the output receives commands from.
        synnax::channel::Key command = 0;
        /// @brief the channel the output writes its state to, if it has one.
        std::optional<synnax::channel::Channel> state;
    };

private:
    /// @brief the vector of channels to stream for commands.
    std::vector<synnax::channel::Key> cmd_channels;
    /// @brief the state channel of each output, keyed by its command channel.
    std::unordered_map<synnax::channel::Key, synnax::channel::Channel> state_channels;
    /// @brief the index keys of the state channels.
    std::set<synnax::channel::Key> state_indexes;
    /// @brief whether data saving is disabled for the task.
    bool data_saving_disabled;
    /// @brief the warning reported during the current write.
    Warning warning;

protected:
    /// @brief reports a problem with the frame in the current write that does not stop
    /// the task, such as a command the sink skipped. The task shows it as a warning
    /// without a retry wait. The next write without a warning clears it.
    void warn(Warning warning) { this->warning = std::move(warning); }

public:
    /// @brief the rate at which to communicate state values down the channel.
    x::telem::Rate state_rate;
    /// @brief used to lock concurrent access to the channel state.
    std::mutex chan_state_lock;
    /// @brief used to signal the state source to send values whenever a command
    /// has been executed.
    std::condition_variable chan_state_cv;
    /// @brief the current state of all the outputs. This is shared between
    /// the command sink and state source.
    std::unordered_map<synnax::channel::Key, x::telem::SampleValue> chan_state;
    /// @brief whether a command has changed the state since the last read.
    bool chan_state_changed = false;

    explicit Sink(std::vector<synnax::channel::Key> cmd_channels):
        cmd_channels(std::move(cmd_channels)),
        state_indexes({}),
        data_saving_disabled(false),
        state_rate(0) {}

    Sink(
        const x::telem::Rate state_rate,
        const std::vector<Output> &outputs,
        const bool data_saving_disabled
    ):
        data_saving_disabled(data_saving_disabled), state_rate(state_rate) {
        for (const auto &out: outputs) {
            this->cmd_channels.push_back(out.command);
            if (!out.state) continue;
            this->state_channels[out.command] = *out.state;
            this->chan_state[out.state->key] = out.state->data_type.cast(0);
            if (out.state->index != 0) this->state_indexes.insert(out.state->index);
        }
    }

    /// @brief an optional function called to start the sink.
    /// @param cmd_key the start command, or driver::task::NO_COMMAND at boot.
    /// @param cancel stops running when the Driver cancels the start.
    /// @returns an error if the sink fails to start, at which point the task will not
    /// proceed with the rest of startup.
    virtual x::errors::Error
    start(const std::string &cmd_key, x::breaker::Breaker &cancel) {
        return x::errors::NIL;
    }

    /// @brief returns the warning reported during the last write, and clears it. Empty
    /// if the write reported none.
    Warning take_warning() { return std::exchange(this->warning, {}); }

    virtual x::errors::Error stop() { return x::errors::NIL; }

    [[nodiscard]] synnax::framer::StreamerConfig streamer_config() const {
        return synnax::framer::StreamerConfig{.channels = this->cmd_channels};
    }

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->state_channels.size());
        for (const auto &[_, ch]: this->state_channels)
            keys.push_back(ch.key);
        for (const auto idx: this->state_indexes)
            keys.push_back(idx);
        return synnax::framer::WriterConfig{
            .channels = keys,
            .mode = data_saving_writer_mode(this->data_saving_disabled),
        };
    }

    void set_state(const x::telem::Frame &frame) {
        std::lock_guard lock{this->chan_state_lock};
        this->chan_state_changed = true;
        this->chan_state_cv.notify_all();
        for (const auto &[cmd_key, s]: frame) {
            const auto it = this->state_channels.find(cmd_key);
            if (it == this->state_channels.end()) continue;
            const auto state_key = it->second.key;
            this->chan_state[state_key] = it->second.data_type.cast(s.at(-1));
        }
    }

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        pipeline::Authorities &authorities
    ) override {
        std::unique_lock lock(chan_state_lock);
        this->chan_state_cv.wait_for(lock, this->state_rate.period().chrono(), [this] {
            return this->chan_state_changed;
        });
        this->chan_state_changed = false;
        fr.clear();
        fr.reserve(this->chan_state.size());
        for (const auto &[key, value]: this->chan_state)
            fr.emplace(key, x::telem::Series(value));
        const auto now = x::telem::TimeStamp::now();
        for (const auto idx: this->state_indexes)
            fr.emplace(idx, x::telem::Series(now));
        return x::errors::NIL;
    }
};

/// @brief a write task that can write to both digital and analog output channels,
/// and communicate their state back to Synnax.
class WriteTask final : public driver::task::Task {
    class WrappedSink final : public driver::pipeline::Sink {
    public:
        /// @brief the parent write task.
        WriteTask &p;
        /// @brief the underlying wrapped sink that actually executes commands on
        /// the hardware.
        std::unique_ptr<common::Sink> internal;
        /// @brief the warning last written to the log. Empty if there is none.
        Warning logged_warning;

        WrappedSink(WriteTask &p, std::unique_ptr<common::Sink> sink):
            p(p), internal(std::move(sink)) {}

        /// @brief implements pipeline::Sink.
        void stopped_with_err(const x::errors::Error &err) override {
            {
                std::lock_guard lock(this->p.state_mu);
                if (this->p.opening) {
                    this->p.state.error(err, "Failed to start task");
                    this->p.answer_pending_starts();
                } else
                    this->p.state.error(err, "Failed to write commands");
                this->p.request_stop();
            }
        }

        // The start status waits for the streamer, so a client that waits for it can
        // send commands at once.
        void opened() override {
            std::lock_guard lock(this->p.state_mu);
            if (this->p.opening) this->p.answer_pending_starts();
        }

        x::errors::Error write(x::telem::Frame &frame) override {
            if (frame.empty()) return x::errors::NIL;
            auto err = this->internal->write(frame);
            const auto warning = this->internal->take_warning();
            if (err) {
                this->logged_warning = {};
                if (err.matches(errors::TEMPORARY_HARDWARE_ERROR))
                    this->p.state.send_warning("Failed to write commands", err.data);
                return err;
            }
            if (warning.empty()) {
                this->logged_warning = {};
                this->p.state.clear_warning();
                return x::errors::NIL;
            }
            if (warning != this->logged_warning)
                LOG(WARNING) << this->p.name() << ": " << warning.message << ": "
                             << warning.description;
            this->logged_warning = warning;
            this->p.state.send_warning(warning.message, warning.description);
            return x::errors::NIL;
        }
    };

    /// @brief reads output state from the sink for the state pipeline.
    class StateSource final : public driver::pipeline::Source {
        /// @brief the parent write task.
        WriteTask &p;

    public:
        /// @brief whether the writer opened since the task last started. Guarded by
        /// state_mu.
        bool writer_open = false;

        explicit StateSource(WriteTask &p): p(p) {}

        void writer_opened() override {
            std::lock_guard lock(this->p.state_mu);
            this->writer_open = true;
        }

        void stopped_with_err(const x::errors::Error &err) override {
            {
                std::lock_guard lock(this->p.state_mu);
                if (this->p.opening || !this->writer_open)
                    this->p.state.error(err, "Failed to start task");
                else
                    this->p.state.error(err, "Failed to write commands");
                if (this->p.opening) this->p.answer_pending_starts();
                this->p.request_stop();
            }
        }

        x::errors::Error read(
            x::breaker::Breaker &breaker,
            x::telem::Frame &fr,
            pipeline::Authorities &authorities
        ) override {
            return this->p.sink->internal->read(breaker, fr, authorities);
        }

        [[nodiscard]] synnax::framer::WriterConfig writer_config() const {
            auto cfg = this->p.sink->internal->writer_config();
            if (cfg.subject.key.empty())
                cfg.subject.key = this->p.state.task.key.to_string();
            if (cfg.subject.name.empty()) cfg.subject.name = this->p.name();
            return cfg;
        }
    };

    /// @brief guards state, opening, and pending_starts, which the command and
    /// pipeline threads both touch.
    std::mutex state_mu;
    /// @brief used to manage and communicate the task's state.
    StatusHandler state;
    /// @brief true from a start until the command streamer opens or fails to open.
    bool opening = false;
    /// @brief true while a sink with no command channels holds its hardware. Such a
    /// sink never starts the command pipe.
    bool running_without_commands = false;
    /// @brief true from a successful sink start until the task stops.
    bool sink_started = false;
    /// @brief the start commands to answer once the command streamer opens.
    std::vector<std::string> pending_starts;
    /// @brief the hardware interface for writing data
    std::shared_ptr<WrappedSink> sink;
    /// @brief the source of output state for the state pipeline.
    std::shared_ptr<StateSource> state_source;
    /// @brief the pipeline used to receive commands from Synnax and write them to
    /// the device.
    driver::pipeline::Control cmd_write_pipe;
    /// @brief the pipeline used to receive state changes from the device and write
    /// to Synnax.
    driver::pipeline::Acquisition state_write_pipe;

    /// @brief serializes starts and stops. Acquire before state_mu.
    std::mutex run_mu;
    /// @brief true from a pipeline failure until the task stops. Guarded by state_mu.
    bool failed = false;
    /// @brief true once the task is destroyed. Guarded by state_mu.
    bool closing = false;
    /// @brief wakes the stopper when failed or closing changes.
    std::condition_variable stopper_cv;
    /// @brief stops the task after a pipeline fails. A pipeline thread cannot stop the
    /// task itself, since the stop joins the other pipeline, which may be stopping the
    /// task too. Guarded by state_mu.
    std::thread stopper;

    /// @brief asks the stopper to stop the task. Requires state_mu.
    void request_stop() {
        this->failed = true;
        if (!this->stopper.joinable())
            this->stopper = std::thread(&WriteTask::run_stopper, this);
        this->stopper_cv.notify_one();
    }

    void run_stopper() {
        std::unique_lock lock(this->state_mu);
        while (true) {
            this->stopper_cv.wait(lock, [this] {
                return this->failed || this->closing;
            });
            if (this->closing) return;
            lock.unlock();
            {
                std::lock_guard run_lock(this->run_mu);
                // A start or stop may have ended the failed run while this waited.
                if (this->is_failed()) this->stop_run("", true);
            }
            lock.lock();
        }
    }

    bool is_failed() {
        std::lock_guard lock(this->state_mu);
        return this->failed;
    }

    /// @brief stops the task. Requires run_mu.
    bool stop_run(const std::string &cmd_key, const bool propagate_state) {
        this->cmd_write_pipe.stop();
        this->state_write_pipe.stop();
        std::lock_guard lock(this->state_mu);
        this->failed = false;
        this->running_without_commands = false;
        const auto stopped = std::exchange(this->sink_started, false);
        if (stopped)
            this->state.error(this->sink->internal->stop(), "Failed to stop task");
        // A start the streamer never answered still needs its reply.
        for (const auto &key: this->pending_starts)
            this->state.ack(key, false);
        this->pending_starts.clear();
        this->opening = false;
        if (propagate_state) this->state.send_stop(cmd_key);
        return stopped;
    }

    /// @brief answers every pending start with the current status. Requires state_mu.
    void answer_pending_starts() {
        for (const auto &key: this->pending_starts)
            this->state.send_start(key);
        this->pending_starts.clear();
        this->opening = false;
    }

public:
    /// @brief base constructor that takes in pipeline factories to allow the
    /// caller to stub cluster communication during tests.
    explicit WriteTask(
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        const x::breaker::Config &breaker_cfg,
        std::unique_ptr<Sink> sink,
        const std::shared_ptr<driver::pipeline::WriterFactory> &writer_factory,
        const std::shared_ptr<driver::pipeline::StreamerFactory> &streamer_factory
    ):
        state(ctx, task),
        sink(std::make_shared<WrappedSink>(*this, std::move(sink))),
        state_source(std::make_shared<StateSource>(*this)),
        cmd_write_pipe(
            streamer_factory,
            this->sink->internal->streamer_config(),
            this->sink,
            breaker_cfg,
            task.name
        ),
        state_write_pipe(
            writer_factory,
            this->state_source->writer_config(),
            this->state_source,
            breaker_cfg,
            task.name + ":state"
        ) {}

    /// @brief primary constructor that uses the task context's Synnax client for
    /// cluster communication.
    explicit WriteTask(
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        const x::breaker::Config &breaker_cfg,
        std::unique_ptr<Sink> sink
    ):
        WriteTask(
            task,
            ctx,
            breaker_cfg,
            std::move(sink),
            bypass::pipeline::create_writer_factory(ctx),
            bypass::pipeline::create_streamer_factory(ctx, {task.name, task.name})
        ) {}

    ~WriteTask() override {
        std::thread t;
        {
            std::lock_guard lock(this->state_mu);
            this->closing = true;
            t = std::move(this->stopper);
        }
        this->stopper_cv.notify_all();
        if (t.joinable()) t.join();
    }

    /// @brief implements driver::task::Task to execute the provided command on the
    /// task.
    void exec(synnax::task::Command &cmd, x::breaker::Breaker &cancel) override {
        if (cmd.type == "start")
            this->start(cmd.key, cancel);
        else if (cmd.type == "stop")
            this->stop(cmd.key, true);
    }

    /// @brief implements driver::task::Task to stop the task.
    void stop(const bool will_reconfigure) override {
        if (will_reconfigure) this->state.close();
        this->stop("", !will_reconfigure);
    }

    /// @brief stops the task.
    /// @param cmd_key - A reference to the command key used to execute the stop.
    /// Will be used internally to communicate the task state.
    /// @param propagate_state whether the task will be reconfigured after it was
    /// stopped.
    bool stop(const std::string &cmd_key, const bool propagate_state) {
        std::lock_guard run_lock(this->run_mu);
        return this->stop_run(cmd_key, propagate_state);
    }

    /// @brief starts the task.
    /// @param cmd_key - A reference to the command key used to execute the start.
    /// Will be used internally to communicate the task state. A task that is
    /// already running is not restarted: the command is answered with the current
    /// status. The start is answered once the command streamer opens, so commands
    /// written after the answer reach the sink. A sink with no command channels is
    /// answered at once and runs without pipelines.
    bool start(const std::string &cmd_key, x::breaker::Breaker &cancel) {
        std::lock_guard run_lock(this->run_mu);
        bool restart = false;
        {
            std::lock_guard lock(this->state_mu);
            // A pipeline sets failed before it stops, so read running first.
            const auto running = this->cmd_write_pipe.running() ||
                                 this->running_without_commands;
            // A command pipeline can also exit without failing, such as when the Core
            // closes the streamer. Its sink is still started.
            restart = this->failed || (!running && this->sink_started);
            if (running && !restart) {
                if (this->opening)
                    this->pending_starts.push_back(cmd_key);
                else
                    this->state.ack(cmd_key, true);
                return false;
            }
        }
        if (restart) this->stop_run("", true);
        {
            std::lock_guard lock(this->state_mu);
            this->state.reset();
            this->sink->logged_warning = {};
            if (this->state.error(
                    this->sink->internal->start(cmd_key, cancel),
                    "Failed to start task"
                )) {
                this->state.send_start(cmd_key);
                return false;
            }
            this->sink_started = true;
            if (this->sink->internal->streamer_config().channels.empty()) {
                this->running_without_commands = true;
                this->state.send_start(cmd_key);
                return true;
            }
            this->opening = true;
            this->state_source->writer_open = false;
            this->pending_starts = {cmd_key};
        }
        this->cmd_write_pipe.start();
        if (!this->sink->internal->writer_config().channels.empty())
            this->state_write_pipe.start();
        return true;
    }

    /// @brief implements driver::task::Task to return the task's name.
    std::string name() const override { return this->state.task.name; }
};
}
