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
#include "driver/task/common/state.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"

namespace common {
class Sink : public pipeline::Sink, public pipeline::Source {
    /// @brief a timer that is used to control the rate at which the state is
    /// propagated.
    loop::Timer state_timer;
    /// @brief the vector of channels to stream for commands.
    const std::vector<synnax::ChannelKey> cmd_channels;
    /// @brief the vector of channels to write state updates for.
    const std::vector<synnax::ChannelKey> state_channels;
    /// @brief the index keys of the state channels.
    const std::set<synnax::ChannelKey> state_indexes;
    /// @brief whether data saving is enabled for the task.
    bool data_saving;

public:
    /// @brief used to lock concurrent access to the channel state.
    std::mutex chan_state_lock;
    /// @brief the current state of all the outputs. This is shared between
/// the command sink and state source.
    std::unordered_map<synnax::ChannelKey, telem::SampleValue> chan_state;

    Sink(
        const telem::Rate state_rate,
        std::set<synnax::ChannelKey> state_indexes,
        std::vector<synnax::ChannelKey> state_channels,
        std::vector<synnax::ChannelKey> cmd_channels,
        const bool data_saving
    ): state_timer(state_rate),
       cmd_channels(std::move(cmd_channels)),
       state_channels(std::move(state_channels)),
       state_indexes(std::move(state_indexes)),
       data_saving(data_saving) {
    }

    virtual xerrors::Error start() = 0;

    virtual xerrors::Error stop() = 0;

    synnax::StreamerConfig streamer_config() const {
        return synnax::StreamerConfig{.channels = this->state_channels};
    }

    synnax::WriterConfig writer_config() const {
        return synnax::WriterConfig{
            .channels = this->cmd_channels,
            .mode = synnax::data_saving_writer_mode(this->data_saving),
            .enable_auto_commit = true,
        };
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        this->state_timer.wait(breaker);
        std::lock_guard lock{this->chan_state_lock};
        auto fr = synnax::Frame(
            this->chan_state,
            this->chan_state.size() + this->state_indexes.size()
        );
        if (!this->state_indexes.empty()) {
            const auto idx_ser = telem::Series(telem::TimeStamp::now());
            for (const auto idx: this->state_indexes)
                fr.emplace(idx, idx_ser.deep_copy());
        }
        return {std::move(fr), xerrors::NIL};
    }
};

/// @brief a write task that can write to both digital and analog output channels,
/// and communicate their state back to Synnax.
class WriteTask final : public task::Task {
    TaskStateHandler state;

    class WrappedSink final : public pipeline::Sink, public pipeline::Source {
    public:
        WriteTask &p;
        std::unique_ptr<common::Sink> wrapped;

        WrappedSink(
            WriteTask &p,
            std::unique_ptr<common::Sink> sink
        ): p(p), wrapped(std::move(sink)) {
        }

        void stopped_with_err(const xerrors::Error &err) override {
            this->p.state.error(err);
            this->p.stop("");
        }

        std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
            return this->wrapped->read(breaker);
        }

        xerrors::Error write(const synnax::Frame &frame) override {
            return this->wrapped->write(frame);
        }
    };

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
            this->sink->wrapped->streamer_config(),
            this->sink,
            breaker_cfg
        ),
        state_write_pipe(
            writer_factory,
            this->sink->wrapped->writer_config(),
            this->sink,
            breaker_cfg
        ) {
    }

    /// @brief primary constructor that uses the task context's Synnax client for
    /// cluster communication.
    explicit WriteTask(
        synnax::Task task,
        const std::shared_ptr<task::Context> &ctx,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<Sink> sink
    ): WriteTask(
        std::move(task),
        ctx,
        breaker_cfg,
        std::move(sink),
        std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client),
        std::make_shared<pipeline::SynnaxStreamerFactory>(ctx->client)
    ) {
    }

    /// @brief implements task::Task to execute teh provided command on the task.
    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key, false);
    }

    /// @brief implements task::Task to stop the task.
    void stop(const bool will_reconfigure) override {
        this->stop("", will_reconfigure);
    }

    /// @brief stops the task.
    /// @param cmd_key - A reference to the command key used to execute the stop. Will
    /// be used internally to communicate the task state.
    /// @param will_reconfigure whether the task will be reconfigured after it was stopped.
    void stop(const std::string &cmd_key, const bool will_reconfigure) {
        this->cmd_write_pipe.stop();
        this->state_write_pipe.stop();
        this->state.error(this->sink->wrapped->stop());
        if (will_reconfigure) return;
        this->state.send_stop(cmd_key);
    }

    /// @brief starts the task.
    /// @param cmd_key - A reference to the command key used to execute the start. Will
    /// be used internally to communicate the task state.
    void start(const std::string &cmd_key) {
        if (!this->state.error(this->sink->wrapped->start())) {
            this->cmd_write_pipe.start();
            this->state_write_pipe.start();
        }
        this->state.send_start(cmd_key);
    }

    /// @brief implements task::Task to return the task's name.
    std::string name() override { return this->state.task.name; }
};
}
