// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <vector>

#include "client/cpp/arc/arc.h"
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xjson/xjson.h"

#include "arc/cpp/module/module.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/runtime.h"
#include "arc/cpp/runtime/state/state.h"
#include "driver/arc/arc.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/status.h"
#include "driver/task/task.h"

namespace arc {
/// @brief configuration for an arc runtime task.
struct TaskConfig {
    /// @brief the key of the arc program to retrieve from Synnax.
    std::string arc_key;
    /// @brief the arc module retrieved from Synnax (already constructed).
    module::Module module;
    /// @brief execution loop configuration.
    runtime::loop::Config loop_config;

    TaskConfig() = default;

    TaskConfig(TaskConfig &&other) noexcept:
        arc_key(std::move(other.arc_key)),
        module(std::move(other.module)),
        loop_config(std::move(other.loop_config)) {}

    TaskConfig(const TaskConfig &) = delete;
    const TaskConfig &operator=(const TaskConfig &) = delete;

    static std::pair<TaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, xjson::Parser &parser) {
        TaskConfig cfg;

        cfg.arc_key = parser.field<std::string>("arc_key");
        if (!parser.ok()) return {std::move(cfg), parser.error()};

        auto [arc_data, arc_err] = client->arcs.retrieve_by_key(
            cfg.arc_key,
            synnax::RetrieveOptions{.compile = true}
        );
        if (arc_err) return {std::move(cfg), arc_err};

        cfg.module = module::Module(arc_data.module);

        const auto mode_str = parser.field<std::string>("execution_mode", "HIGH_RATE");
        auto mode = runtime::loop::ExecutionMode::HIGH_RATE;
        if (mode_str == "BUSY_WAIT") {
            mode = runtime::loop::ExecutionMode::BUSY_WAIT;
        } else if (mode_str == "HIGH_RATE")
            mode = runtime::loop::ExecutionMode::HIGH_RATE;
        else if (mode_str == "RT_EVENT")
            mode = runtime::loop::ExecutionMode::RT_EVENT;
        else if (mode_str == "HYBRID")
            mode = runtime::loop::ExecutionMode::HYBRID;
        else if (mode_str == "EVENT_DRIVEN")
            mode = runtime::loop::ExecutionMode::EVENT_DRIVEN;
        else {
            parser.field_err(
                "execution_mode",
                "invalid execution mode: " + mode_str +
                    " (must be BUSY_WAIT, HIGH_RATE, RT_EVENT, HYBRID, or EVENT_DRIVEN)"
            );
            return {std::move(cfg), parser.error()};
        }

        cfg.loop_config = runtime::loop::Config{
            .mode = mode,
            .interval = telem::TimeSpan(
                parser.field<uint64_t>("interval_ns", 10000000ULL)
            ),
            .rt_priority = parser.field<int>("rt_priority", 47),
            .cpu_affinity = parser.field<int>("cpu_affinity", -1),
        };

        return {std::move(cfg), xerrors::NIL};
    }
};

/// @brief helper function to load the arc runtime.
/// @param config the task configuration containing the module and loop config.
/// @param client the Synnax client for channel retrieval.
/// @returns a pair containing the runtime instance and any error that occurred.
inline std::pair<std::shared_ptr<runtime::Runtime>, xerrors::Error>
load_runtime(const TaskConfig &config, const std::shared_ptr<synnax::Synnax> &client) {
    const runtime::Config runtime_cfg{
        .mod = config.module,
        .breaker = breaker::default_config("arc_runtime"),
        .retrieve_channels = [client](const std::vector<types::ChannelKey> &keys)
            -> std::pair<std::vector<runtime::state::ChannelDigest>, xerrors::Error> {
            auto [channels, err] = client->channels.retrieve(keys);
            if (err) return {{}, err};
            std::vector<runtime::state::ChannelDigest> digests;
            for (const auto &ch: channels) {
                digests.push_back(
                    runtime::state::ChannelDigest{
                        .key = ch.key,
                        .data_type = ch.data_type,
                        .index = ch.index
                    }
                );
            }
            return {digests, xerrors::NIL};
        }
    };
    return runtime::load(runtime_cfg);
}

/// @brief source that reads output data from the arc runtime and sends it to Synnax.
class Source final : public pipeline::Source {
    std::shared_ptr<runtime::Runtime> runtime;

public:
    explicit Source(const std::shared_ptr<runtime::Runtime> &runtime):
        runtime(runtime) {}

    xerrors::Error read(breaker::Breaker &breaker, telem::Frame &data) override {
        this->runtime->read(data);
        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {
        LOG(ERROR) << "[arc] runtime stopped with error: " << err.message();
    }
};

/// @brief sink that receives input data from Synnax and sends it to the arc runtime.
class Sink final : public pipeline::Sink {
    std::shared_ptr<runtime::Runtime> runtime;

public:
    explicit Sink(const std::shared_ptr<runtime::Runtime> &runtime): runtime(runtime) {}

    xerrors::Error write(telem::Frame &frame) override {
        if (frame.empty()) return xerrors::NIL;
        VLOG(1) << "[arc.sink] writing to runtime " << frame;
        return this->runtime->write(std::move(frame));
    }
};

/// @brief arc runtime task that manages both read and write pipelines.
class Task final : public task::Task {
    std::shared_ptr<runtime::Runtime> runtime;
    std::unique_ptr<pipeline::Acquisition> acquisition;
    std::unique_ptr<pipeline::Control> control;
    common::StatusHandler state;

public:
    explicit Task(
        const synnax::Task &task_meta,
        const std::shared_ptr<task::Context> &ctx,
        std::shared_ptr<runtime::Runtime> runtime,
        const TaskConfig &cfg,
        std::shared_ptr<pipeline::WriterFactory> writer_factory = nullptr,
        std::shared_ptr<pipeline::StreamerFactory> streamer_factory = nullptr
    ):
        runtime(std::move(runtime)), state(ctx, task_meta) {
        auto source = std::make_unique<Source>(this->runtime);
        auto sink = std::make_unique<Sink>(this->runtime);
        if (!writer_factory)
            writer_factory = std::make_shared<pipeline::SynnaxWriterFactory>(
                ctx->client
            );
        if (!streamer_factory)
            streamer_factory = std::make_shared<pipeline::SynnaxStreamerFactory>(
                ctx->client
            );
        this->acquisition = std::make_unique<pipeline::Acquisition>(
            writer_factory,
            synnax::WriterConfig{
                .channels = this->runtime->write_channels,
                .start = telem::TimeStamp::now(),
                .mode = synnax::WriterMode::PersistStream,
            },
            std::move(source),
            breaker::default_config("arc_acquisition"),
            "arc_acquisition"
        );
        this->control = std::make_unique<pipeline::Control>(
            streamer_factory,
            synnax::StreamerConfig{.channels = this->runtime->read_channels},
            std::move(sink),
            breaker::default_config("arc_control"),
            "arc_control"
        );
    }

    bool start(const std::string &cmd_key) {
        // Runtime
        const auto runtime_started = this->runtime->start();
        // Outgoing telemetry
        const auto acq_started = this->acquisition->start();
        // Incoming telemetry
        const auto control_started = this->control->start();
        this->state.send_start(cmd_key);
        return acq_started && control_started && runtime_started;
    }

    bool stop(const std::string &cmd_key, const bool propagate_state) {
        // Incoming telemetry
        const auto control_stopped = this->control->stop();
        // Close the output queue to unblock acquisition pipeline
        this->runtime->close_outputs();
        // Outgoing telemetry (now unblocked)
        const auto acq_stopped = this->acquisition->stop();
        // Runtime
        const auto runtime_stopped = this->runtime->stop();
        if (propagate_state) this->state.send_stop(cmd_key);
        return control_stopped && acq_stopped && runtime_stopped;
    }

    void exec(task::Command &cmd) override {
        if (cmd.type == "start")
            this->start(cmd.key);
        else if (cmd.type == "stop")
            this->stop(false);
        else
            LOG(WARNING) << "[arc] unknown command type: " << cmd.type;
    }

    void stop(const bool will_reconfigure) override {
        this->stop("", !will_reconfigure);
    }

    /// @brief returns the name of the task.
    [[nodiscard]] std::string name() const override { return "Arc Runtime Task"; }
};
}
