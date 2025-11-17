// Copyright 2025 Synnax Labs, Inc.
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
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/status.h"
#include "driver/task/task.h"

namespace arc {
/// @brief integration name for arc runtime.
const std::string INTEGRATION_NAME = "arc";
/// @brief task type for arc runtime tasks.
const std::string TASK_TYPE = INTEGRATION_NAME + "_runtime";

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

        auto [arc_data, arc_err] = client->arcs.retrieve_by_key(cfg.arc_key);
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
        return this->runtime->read(data);
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
        return this->runtime->write(std::move(frame));
    }
};

/// @brief arc runtime task that manages both read and write pipelines.
class Task final : public task::Task {
    /// @brief the arc runtime instance.
    std::shared_ptr<runtime::Runtime> runtime;
    /// @brief acquisition pipeline for reading runtime outputs.
    std::unique_ptr<pipeline::Acquisition> acquisition;
    /// @brief control pipeline for writing runtime inputs.
    std::unique_ptr<pipeline::Control> control;
    /// @brief status handler for reporting task status.
    common::StatusHandler state;

public:
    explicit Task(
        const synnax::Task &task_meta,
        const std::shared_ptr<task::Context> &ctx,
        std::shared_ptr<runtime::Runtime> runtime,
        const TaskConfig &cfg
    ):
        runtime(std::move(runtime)), state(ctx, task_meta) {
        auto source = std::make_unique<Source>(this->runtime);
        auto sink = std::make_unique<Sink>(this->runtime);
        this->acquisition = std::make_unique<pipeline::Acquisition>(
            ctx->client,
            synnax::WriterConfig{
                .channels = this->runtime->write_channels,
                .start = telem::TimeStamp::now(),
                .mode = synnax::WriterMode::PersistStream,
            },
            std::move(source),
            breaker::default_config("arc_acquisition")
        );
        this->control = std::make_unique<pipeline::Control>(
            ctx->client,
            synnax::StreamerConfig{.channels = this->runtime->read_channels},
            std::move(sink),
            breaker::default_config("arc_control")
        );
    }

    /// @brief executes a command on the task.
    /// @param cmd the command to execute.
    void exec(task::Command &cmd) override {
        if (cmd.type == "stop")
            this->stop(false);
        else
            LOG(WARNING) << "[arc] unknown command type: " << cmd.type;
    }

    /// @brief stops the task and cleans up resources.
    void stop(bool will_reconfigure) override {
        this->acquisition.reset();
        this->control.reset();
    }

    /// @brief returns the name of the task.
    [[nodiscard]] std::string name() const override { return "Arc Runtime Task"; }
};

/// @brief factory for creating arc runtime tasks.
class Factory final : public task::Factory {
public:
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        if (task.type != TASK_TYPE) return {nullptr, false};

        auto parser = xjson::Parser(task.config);
        auto [cfg, cfg_err] = TaskConfig::parse(ctx->client, parser);
        if (cfg_err) {
            ctx->set_status(
                synnax::TaskStatus{
                    .variant = "error",
                    .details = synnax::TaskStatusDetails{
                        .task = task.key,
                        .running = false,
                    },
                }
            );
            return {nullptr, true};
        }

        // Load runtime
        auto [runtime, rt_err] = load_runtime(cfg, ctx->client);
        if (rt_err) {
            ctx->set_status(
                synnax::TaskStatus{
                    .variant = "error",
                    .details = synnax::TaskStatusDetails{
                        .task = task.key,
                        .running = false,
                    },
                }
            );
            return {nullptr, true};
        }

        return {std::make_unique<Task>(task, ctx, std::move(runtime), cfg), true};
    }

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override {
        return {};
    }

    [[nodiscard]] std::string name() override { return INTEGRATION_NAME; }
};
}
