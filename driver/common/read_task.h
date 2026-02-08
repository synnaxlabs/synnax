// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/common/common.h"
#include "driver/common/sample_clock.h"
#include "driver/common/status.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/task/task.h"
#include "driver/transform/transform.h"

namespace driver::common {
/// @brief common read task configuration parameters used across multiple drivers.
struct BaseReadTaskConfig : BaseTaskConfig {
    /// @brief sets the sample rate for the task.
    const x::telem::Rate sample_rate;
    /// @brief sets the stream rate for the task.
    const x::telem::Rate stream_rate;
    /// @brief timing configuration options for the task.
    common::TimingConfig timing;

    BaseReadTaskConfig(BaseReadTaskConfig &&other) noexcept:
        BaseTaskConfig(std::move(other)),
        sample_rate(other.sample_rate),
        stream_rate(other.stream_rate),
        timing(other.timing) {}

    BaseReadTaskConfig(const BaseReadTaskConfig &) = delete;

    const BaseReadTaskConfig &operator=(const BaseReadTaskConfig &) = delete;

    explicit BaseReadTaskConfig(
        x::json::Parser &cfg,
        const common::TimingConfig timing_cfg = common::TimingConfig(),
        const bool stream_rate_required = true
    ):
        BaseTaskConfig(cfg),
        sample_rate(x::telem::Rate(cfg.field<float>("sample_rate", 0))),
        stream_rate(x::telem::Rate(cfg.field<float>("stream_rate", 0))),
        timing(timing_cfg) {
        if (sample_rate <= x::telem::Rate(0))
            cfg.field_err("sample_rate", "must be greater than 0");
        if (stream_rate_required && stream_rate <= x::telem::Rate(0))
            cfg.field_err("stream_rate", "must be greater than 0");
        if (stream_rate_required && sample_rate < stream_rate)
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

struct ReadResult {
    x::errors::Error error;
    std::string warning;
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
class ReadTask final : public task::Task {
    /// @brief the task context used to communicate state changes back to Synnax.
    /// @brief tare middleware used for taring values.
    transform::Tare tare;
    /// @brief handles communicating the task state back to the cluster.
    StatusHandler state;

    /// @brief a wrapped source that gracefully handles shutdown when a hardware
    /// read fails or the pipeline fails to write to Synnax.
    class InternalSource final : public pipeline::Source {
        /// @brief the parent read task.
        ReadTask &p;

    public:
        /// @brief the wrapped, hardware-specific source.
        std::unique_ptr<common::Source> internal;

        InternalSource(ReadTask &p, std::unique_ptr<common::Source> internal):
            p(p), internal(std::move(internal)) {}

        void stopped_with_err(const x::errors::Error &err) override {
            this->p.state.error(err);
            this->p.stop("", true);
        }

        x::errors::Error
        read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
            auto [err, warning] = this->internal->read(breaker, fr);
            // Three cases.
            // 1. We have an error, but it's temporary, so we trigger the breaker
            // by returning the error and  send a warning to start retrying at
            // scaled intervals.
            // 2. We have a critical error, in which case we return it directly.
            // 3. We have a warning, in which case we communicate it and return nil.
            if (err) {
                if (err.matches(errors::TEMPORARY_HARDWARE_ERROR)) {
                    LOG(WARNING) << this->p.name() << ": " << err.message();
                    this->p.state.send_warning(err.message());
                } else
                    LOG(ERROR) << this->p.name() << ": " << err.message();
                return err;
            }
            if (!warning.empty()) {
                LOG(WARNING) << this->p.name() << ": " << warning;
                this->p.state.send_warning(warning);
            } else
                this->p.state.clear_warning();
            return this->p.tare.transform(fr);
        }

        [[nodiscard]] synnax::framer::WriterConfig writer_config() const {
            auto cfg = this->internal->writer_config();
            if (cfg.subject.name.empty()) cfg.subject.name = this->p.name();
            return cfg;
        }
    };

    std::shared_ptr<InternalSource> source;

    /// @brief the pipeline used to read data from the hardware and pipe it to
    /// Synnax.
    pipeline::Acquisition pipe;

public:
    /// @brief base constructor that takes in a pipeline writer factory to allow the
    /// caller to stub cluster communication during tests.
    explicit ReadTask(
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        const x::breaker::Config &breaker_cfg,
        std::unique_ptr<Source> source,
        const std::shared_ptr<pipeline::WriterFactory> &factory
    ):
        tare(transform::Tare(source->channels())),
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
            std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client)
        ) {}

    /// @brief executes the given command on the task.
    void exec(task::Command &cmd) override {
        if (cmd.type == "start")
            this->start(cmd.key);
        else if (cmd.type == "stop")
            this->stop(cmd.key, true);
        else if (cmd.type == "tare")
            this->tare.tare(cmd.args);
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
    std::string name() const override { return this->state.task.name; }
};

inline std::string skew_warning(const size_t skew) {
    return "Synnax Driver can't keep up with hardware data acquisition, and is "
           "trailing " +
           std::to_string(skew) +
           " samples behind. Lower the stream rate for the task.";
}

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
