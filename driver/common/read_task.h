// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "absl/log/log.h"

#include "client/cpp/task/json.gen.h"

#include "driver/bypass/pipeline/factory.h"
#include "driver/common/common.h"
#include "driver/common/sample_clock.h"
#include "driver/common/status.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/task/task.h"
#include "driver/transform/transform.h"

namespace driver::common {
/// @brief common read task configuration shared across hardware acquisition tasks.
/// Wraps the schema-generated read config (auto_start, data_saving_disabled,
/// sample_rate, stream_rate) with the driver-only timing options, so the field set
/// has a single definition in the oracle schema.
struct BaseReadTaskConfig : ::synnax::task::ReadConfig {
    /// @brief timing configuration options for the task.
    TimingConfig timing;

    explicit BaseReadTaskConfig(
        x::json::Parser &cfg,
        const common::TimingConfig timing_cfg = common::TimingConfig(),
        const bool stream_rate_required = true
    ):
        ::synnax::task::ReadConfig(::synnax::task::ReadConfig::parse(cfg)),
        timing(timing_cfg) {
        if (this->sample_rate <= x::telem::Rate(0))
            cfg.field_err("sample_rate", "must be greater than 0");
        if (stream_rate_required && this->stream_rate <= x::telem::Rate(0))
            cfg.field_err("stream_rate", "must be greater than 0");
        if (stream_rate_required && this->sample_rate < this->stream_rate)
            cfg.field_err(
                "sample_rate",
                "must be greater than or equal to stream rate"
            );
    }
};

/// @brief Initializes a frame with the correct size and series for all channels
template<typename ChannelContainer>
void initialize_frame(
    x::telem::Frame &fr,
    const ChannelContainer &channels,
    const std::set<synnax::channel::Key> &index_keys,
    const size_t samples_per_chan
) {
    if (fr.size() == channels.size() + index_keys.size()) return;
    fr.reserve(channels.size() + index_keys.size());
    for (const auto &ch: channels)
        fr.emplace(
            ch->synnax_key,
            x::telem::Series(ch->ch.data_type, samples_per_chan)
        );
    for (const auto &idx: index_keys)
        fr.emplace(idx, x::telem::Series(x::telem::TIMESTAMP_T, samples_per_chan));
}

/// @brief a problem that does not stop a read, sent as a task warning.
struct Warning {
    /// @brief what happened, in one short line.
    std::string message;
    /// @brief why it happened and what to do now.
    std::string description;

    [[nodiscard]] bool empty() const { return this->message.empty(); }
};

/// @brief the warning a source reports when it falls behind the hardware. It holds no
/// sample count, so its text stays the same from read to read.
const Warning SKEW_WARNING{
    .message = "Falling behind hardware",
    .description = "The Driver cannot keep up with the hardware and is falling "
                   "behind. Lower the stream rate for the task.",
};

struct ReadResult {
    x::errors::Error error;
    Warning warning;
};

/// @brief a source that can be used to read data from a hardware device.
struct Source {
    /// @brief the configuration used to open a writer for the source.
    [[nodiscard]] virtual synnax::framer::WriterConfig writer_config() const = 0;

    [[nodiscard]] virtual std::vector<synnax::channel::Channel> channels() const = 0;

    /// @brief an optional function called to start the source.
    /// @returns an error if the source fails to start, at which point the task
    /// will not proceed with the rest of startup.
    virtual x::errors::Error start() { return x::errors::NIL; }

    /// @brief an optional function called to stop the source.
    virtual x::errors::Error stop() { return x::errors::NIL; }

    virtual ReadResult read(x::breaker::Breaker &breaker, x::telem::Frame &data) = 0;

    virtual ~Source() = default;
};

/// @brief a read task that can pull from both analog and digital channels.
class ReadTask final : public driver::task::Task {
    /// @brief the task context used to communicate state changes back to Synnax.
    /// @brief tare middleware used for taring values.
    driver::transform::Tare tare;
    /// @brief handles communicating the task state back to the cluster.
    StatusHandler state;

    /// @brief a wrapped source that gracefully handles shutdown when a hardware
    /// read fails or the pipeline fails to write to Synnax.
    class InternalSource final : public driver::pipeline::Source {
        /// @brief the parent read task.
        ReadTask &p;

    public:
        /// @brief the wrapped, hardware-specific source.
        std::unique_ptr<common::Source> internal;

        InternalSource(ReadTask &p, std::unique_ptr<common::Source> internal):
            p(p), internal(std::move(internal)) {}

        void stopped_with_err(const x::errors::Error &err) override {
            this->p.state.error(err, "Failed to read data");
            this->p.stop("", true);
        }

        x::errors::Error read(
            x::breaker::Breaker &breaker,
            x::telem::Frame &fr,
            pipeline::Authorities &authorities
        ) override {
            auto [err, warning] = this->internal->read(breaker, fr);
            // Three cases.
            // 1. We have an error, but it's temporary, so we trigger the breaker
            // by returning the error and  send a warning to start retrying at
            // scaled intervals.
            // 2. We have a critical error, in which case we return it directly.
            // 3. We have a warning, in which case we communicate it and return nil.
            if (err) {
                if (err.matches(errors::TEMPORARY_HARDWARE_ERROR)) {
                    auto description = err.data;
                    if (const auto retry = breaker.next_retry(); !retry.empty()) {
                        if (!description.empty()) description += ' ';
                        description += "Next " + retry + ".";
                    }
                    LOG(WARNING)
                        << this->p.name() << ": [" << err.type << "] " << description;
                    this->p.state.send_warning("Failed to read data", description);
                } else
                    LOG(ERROR) << this->p.name() << ": " << err.message();
                return err;
            }
            if (!warning.empty()) {
                LOG(WARNING) << this->p.name() << ": " << warning.message << ": "
                             << warning.description;
                this->p.state.send_warning(warning.message, warning.description);
            } else
                this->p.state.clear_warning();
            return this->p.tare.transform(fr);
        }

        [[nodiscard]] synnax::framer::WriterConfig writer_config() const {
            auto cfg = this->internal->writer_config();
            if (cfg.subject.key.empty())
                cfg.subject.key = this->p.state.task.key.to_string();
            if (cfg.subject.name.empty()) cfg.subject.name = this->p.name();
            return cfg;
        }
    };

    std::shared_ptr<InternalSource> source;

    /// @brief the pipeline used to read data from the hardware and pipe it to
    /// Synnax.
    driver::pipeline::Acquisition pipe;

public:
    /// @brief base constructor that takes in a pipeline writer factory to allow the
    /// caller to stub cluster communication during tests.
    explicit ReadTask(
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        const x::breaker::Config &breaker_cfg,
        std::unique_ptr<Source> source,
        const std::shared_ptr<driver::pipeline::WriterFactory> &factory
    ):
        tare(driver::transform::Tare(source->channels())),
        state(ctx, task),
        source(std::make_shared<InternalSource>(*this, std::move(source))),
        pipe(
            factory,
            this->source->writer_config(),
            this->source,
            breaker_cfg,
            task.name
        ) {}

    /// @brief primary constructor that uses the task context's Synnax client in
    /// order to communicate with the cluster.
    explicit ReadTask(
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        const x::breaker::Config &breaker_cfg,
        std::unique_ptr<Source> source
    ):
        ReadTask(
            task,
            ctx,
            breaker_cfg,
            std::move(source),
            bypass::pipeline::create_writer_factory(ctx)
        ) {}

    /// @brief executes the given command on the task.
    void exec(synnax::task::Command &cmd) override {
        if (cmd.type == "start")
            this->start(cmd.key);
        else if (cmd.type == "stop")
            this->stop(cmd.key, true);
        else if (cmd.type == "tare")
            this->tare.tare(cmd.args);
    }

    /// @brief stops the task.
    void stop(const bool will_reconfigure) override {
        if (will_reconfigure) this->state.close();
        this->stop("", !will_reconfigure);
    }

    /// @brief stops the task, using the given command key as reference for
    /// communicating success state.
    bool stop(const std::string &cmd_key, const bool propagate_state) {
        const auto stopped = this->pipe.stop();
        if (stopped)
            this->state.error(this->source->internal->stop(), "Failed to stop task");
        if (propagate_state) this->state.send_stop(cmd_key);
        return stopped;
    }

    /// @brief starts the task, using the given command key as a reference for
    /// communicating task state. A task that is already running is not restarted:
    /// the command is answered with the current status.
    bool start(const std::string &cmd_key) {
        if (this->pipe.running()) {
            this->state.ack(cmd_key, true);
            return false;
        }
        this->state.reset();
        const auto start_ok = !this->state.error(
            this->source->internal->start(),
            "Failed to start task"
        );
        if (start_ok) this->pipe.start();
        this->state.send_start(cmd_key);
        return start_ok;
    }

    /// @brief implements driver::task::Task.
    std::string name() const override { return this->state.task.name; }
};

template<typename T>
void transfer_buf(
    const std::vector<T> &buf,
    const x::telem::Frame &fr,
    const size_t n_channels,
    const size_t n_samples_per_channel
) {
    for (size_t i = 0; i < n_channels; ++i) {
        auto &s = fr.series->at(i);
        s.clear();
        s.write_casted(buf.data() + i * n_samples_per_channel, n_samples_per_channel);
    }
}
}
