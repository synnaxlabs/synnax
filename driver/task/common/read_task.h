// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "state.h"
#include "driver/task/task.h"
#include "driver/pipeline/acquisition.h"

namespace common {
/// @brief a source that can be used to read data from a hardware device.
class Source : public pipeline::Source {
public:
    /// @brief starts the source.
    virtual xerrors::Error start() = 0;

    virtual xerrors::Error stop() = 0;

    virtual synnax::WriterConfig writer_config() const = 0;
};

/// @brief a read task that can pull from both analog and digital channels.
class ReadTask final : public task::Task {
    /// @brief the task context used to communicate state changes back to Synnax.
    /// @brief tare middleware used for taring values.
    std::shared_ptr<pipeline::TareMiddleware> tare_mw;
    /// @brief handles communicating the task state back to the cluster.
    TaskStateHandler state;

    /// @brief a wrapped source that gracefully handles shutdown when a hardware
    /// read fails or the pipeline fails to write to Synnax.
    class InternalSource final : public pipeline::Source {
        /// @brief the parent read task.
        ReadTask &p;
    public:
        /// @brief the wrapped, hardware-specific source.
        std::unique_ptr<common::Source> internal;

        InternalSource(
            ReadTask &p,
            std::unique_ptr<common::Source> internal
        ): p(p), internal(std::move(internal)) {
        }

        void stopped_with_err(const xerrors::Error &err) override {
            this->p.state.error(err);
            this->p.stop("", false);
        }

        std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
            return this->internal->read(breaker);
        }
    };

    std::shared_ptr<InternalSource> source;

    /// @brief the pipeline used to read data from the hardware and pipe it to Synnax.
    pipeline::Acquisition pipe;

public:
    /// @brief base constructor that takes in a pipeline writer factory to allow the
    /// caller to stub cluster communication during tests.
    explicit ReadTask(
        const synnax::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<Source> source,
        const std::shared_ptr<pipeline::WriterFactory> &factory
    ):
        tare_mw(std::make_shared<pipeline::TareMiddleware>(
            source->writer_config().channels)),
        state(ctx, task),
        source(std::make_shared<InternalSource>(*this, std::move(source))),
        pipe(
            factory,
            this->source->internal->writer_config(),
            this->source,
            breaker_cfg
        ) {
        this->pipe.use(this->tare_mw);
    }

    /// @brief primary constructor that uses the task context's Synnax client in order
    /// to communicate with the cluster.
    explicit ReadTask(
        synnax::Task task,
        const std::shared_ptr<task::Context> &ctx,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<Source> source
    ): ReadTask(
        std::move(task),
        ctx,
        breaker_cfg,
        std::move(source),
        std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client)
    ) {
    }

    /// @brief executes the given command on the task.
    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key, false);
        else if (cmd.type == "tare") this->tare_mw->tare(cmd.args);
    }

    /// @brief stops the task.
    void stop(const bool will_reconfigure) override {
        this->stop("", will_reconfigure);
    }

    /// @brief stops the task, using the given command key as reference for
    /// communicating success state.
    void stop(const std::string &cmd_key, const bool will_reconfigure) {
        if (const auto was_running = this->pipe.stop(); !was_running) return;
        this->state.error(this->source->internal->stop());
        if (will_reconfigure) return;
        this->state.send_stop(cmd_key);
    }

    /// @brief starts the task, using the given command key as a reference for
    /// communicating task state.
    void start(const std::string &cmd_key) {
        if (this->pipe.running()) return;
        if (!this->state.error(this->source->internal->start()))
            this->pipe.start();
        this->state.send_start(cmd_key);
    }

    /// @brief implements task::Task.
    std::string name() override { return this->state.task.name; }
};
}
