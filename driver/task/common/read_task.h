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
#include "driver/task/task.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/transform/transform.h"

namespace common {
struct BaseReadTaskConfig {
    /// @brief whether data saving is enabled for the task.
    const bool data_saving;
    /// @brief sets the sample rate for the task.
    const telem::Rate sample_rate;
    /// @brief sets the stream rate for the task.
    const telem::Rate stream_rate;

    BaseReadTaskConfig(BaseReadTaskConfig &&other) noexcept:
        data_saving(other.data_saving),
        sample_rate(other.sample_rate),
        stream_rate(other.stream_rate) {
    }

    BaseReadTaskConfig(const BaseReadTaskConfig &) = delete;
    const BaseReadTaskConfig &operator=(const BaseReadTaskConfig &) = delete;

    explicit BaseReadTaskConfig(xjson::Parser &cfg):
        data_saving(cfg.optional<bool>("data_saving", false)),
        sample_rate(telem::Rate(cfg.required<float>("sample_rate"))),
        stream_rate(telem::Rate(cfg.required<float>("stream_rate"))) {
        if (sample_rate < telem::Rate(0)) cfg.field_err("sample_rate", "must be greater than 0");
        if (stream_rate < telem::Rate(0)) cfg.field_err("stream_rate", "must be greater than 0");
        if (sample_rate < stream_rate)
            cfg.field_err("sample_rate", "must be greater than or equal to stream rate");
    }
};


/// @brief a source that can be used to read data from a hardware device.
struct Source : pipeline::Source {
    /// @brief the configuration used to open a writer for the source.
    virtual synnax::WriterConfig writer_config() const = 0;

    virtual std::vector<synnax::Channel> channels() const = 0;

    /// @brief an optional function called to start the source.
    /// @returns an error if the source fails to start, at which point the task
    /// will not proceed with the rest of startup.
    virtual xerrors::Error start() { return xerrors::NIL; }

    /// @brief an optional function called to stop the source.
    virtual xerrors::Error stop() { return xerrors::NIL; }
};

/// @brief a read task that can pull from both analog and digital channels.
class ReadTask final : public task::Task {
    /// @brief the task context used to communicate state changes back to Synnax.
    /// @brief tare middleware used for taring values.
    transform::Tare tare;
    /// @brief handles communicating the task state back to the cluster.
    StateHandler state;

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
            this->p.stop("", true);
        }

        std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
            auto [fr, err] = this->internal->read(breaker);
            if (!err)
                this->p.state.clear_warning();
            else if (err.matches(driver::TEMPORARY_HARDWARE_ERROR))
                this->p.state.send_warning(err.message());
            if (err) return {std::move(fr), err};
            err = this->p.tare.transform(fr);
            return {std::move(fr), err};
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
        tare(transform::Tare(source->channels())),
        state(ctx, task),
        source(std::make_shared<InternalSource>(*this, std::move(source))),
        pipe(
            factory,
            this->source->internal->writer_config(),
            this->source,
            breaker_cfg
        ) {
    }

    /// @brief primary constructor that uses the task context's Synnax client in order
    /// to communicate with the cluster.
    explicit ReadTask(
        const synnax::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<Source> source
    ): ReadTask(
        task,
        ctx,
        breaker_cfg,
        std::move(source),
        std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client)
    ) {
    }

    /// @brief executes the given command on the task.
    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key, true);
        else if (cmd.type == "tare") this->tare.tare(cmd.args);
    }

    /// @brief stops the task.
    void stop(const bool will_reconfigure) override {
        this->stop("", !will_reconfigure);
    }

    /// @brief stops the task, using the given command key as reference for
    /// communicating success state.
    bool stop(const std::string &cmd_key, const bool propagate_state) {
        const auto stopped = this->pipe.stop();
        if (stopped) this->state.error(this->source->internal->stop());
        if (propagate_state) this->state.send_stop(cmd_key);
        return stopped;
    }

    /// @brief starts the task, using the given command key as a reference for
    /// communicating task state.
    bool start(const std::string &cmd_key) {
        this->stop("", false);
        this->state.reset();
        if (this->pipe.running()) return false;
        const auto start_ok = !this->state.error(this->source->internal->start());
        if (start_ok) this->pipe.start();
        this->state.send_start(cmd_key);
        return start_ok;
    }

    /// @brief implements task::Task.
    std::string name() override { return this->state.task.name; }
};
}
