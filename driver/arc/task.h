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
#include <mutex>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/arc/arc.h"
#include "client/cpp/synnax.h"
#include "client/cpp/task/json.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/json/json.h"
#include "x/cpp/uuid/uuid.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/runtime.h"
#include "arc/cpp/runtime/state/state.h"
#include "driver/arc/ranges/ranges.h"
#include "driver/arc/status/status.h"
#include "driver/bypass/pipeline/factory.h"
#include "driver/common/common.h"
#include "driver/common/status.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"

namespace driver::arc {
/// @brief configuration for an arc runtime task.
struct TaskConfig : ::synnax::task::PersistConfig {
    x::uuid::UUID arc_key;
    ::arc::program::Program program;
    ::arc::runtime::loop::Config loop;

    TaskConfig(TaskConfig &&other) noexcept:
        ::synnax::task::PersistConfig(std::move(other)),
        arc_key(other.arc_key),
        program(std::move(other.program)),
        loop(other.loop) {}

    TaskConfig(const TaskConfig &) = delete;
    const TaskConfig &operator=(const TaskConfig &) = delete;

    explicit TaskConfig(x::json::Parser &parser):
        ::synnax::task::PersistConfig(::synnax::task::PersistConfig::parse(parser)),
        arc_key(parser.field<x::uuid::UUID>("arc_key")),
        loop(parser) {}

    static std::pair<TaskConfig, x::errors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, x::json::Parser &parser) {
        auto cfg = TaskConfig(parser);
        if (!parser.ok()) return {std::move(cfg), parser.error()};
        auto [arc_data, arc_err] = client->arcs.retrieve_by_key(
            cfg.arc_key,
            synnax::arc::RetrieveOptions{.compile = true}
        );
        if (arc_err) return {std::move(cfg), arc_err};
        if (!arc_data.program.has_value())
            return {
                std::move(cfg),
                x::errors::Error(
                    x::errors::INTERNAL,
                    "The Arc program is not compiled."
                )
            };
        cfg.program = *arc_data.program;
        return {std::move(cfg), x::errors::NIL};
    }
};

/// @brief arc runtime task that manages both read and write pipelines.
class Task final : public task::Task {
    std::shared_ptr<::arc::runtime::Runtime> runtime;
    std::unique_ptr<pipeline::Acquisition> acquisition;
    std::unique_ptr<pipeline::Control> control;
    /// @brief guards state, opening, and pending_starts, which the command, pipeline,
    /// and runtime threads all touch.
    std::mutex state_mu;
    common::StatusHandler state;
    /// @brief true from a start until the command streamer opens or fails to open.
    bool opening = false;
    /// @brief the start commands to answer once the command streamer opens.
    std::vector<std::string> pending_starts;

    /// @brief answers every pending start with the current status. Requires state_mu.
    void answer_pending_starts() {
        for (const auto &key: this->pending_starts)
            this->state.send_start(key);
        this->pending_starts.clear();
        this->opening = false;
    }

    /// @brief source that reads output data from the arc runtime.
    class Source final : public pipeline::Source {
        Task &task;

    public:
        explicit Source(Task &task): task(task) {}

        x::errors::Error read(
            x::breaker::Breaker &breaker,
            x::telem::Frame &fr,
            pipeline::Authorities &authorities
        ) override {
            ::arc::runtime::Output out;
            if (!this->task.runtime->read(out)) return errors::NOMINAL_SHUTDOWN_ERROR;
            fr = std::move(out.frame);
            for (auto &c: out.authority_changes) {
                if (c.channel_key.has_value())
                    authorities.keys.push_back(*c.channel_key);
                authorities.authorities.push_back(c.authority);
            }
            return x::errors::NIL;
        }

        void stopped_with_err(const x::errors::Error &err) override {
            {
                std::lock_guard lock(this->task.state_mu);
                this->task.state.error(err, "Failed to run Arc program");
            }
            this->task.stop("", true);
        }
    };

    /// @brief sink that receives input data from Synnax.
    class Sink final : public pipeline::Sink {
        Task &task;

    public:
        explicit Sink(Task &task): task(task) {}

        x::errors::Error write(x::telem::Frame &frame) override {
            if (frame.empty()) return x::errors::NIL;
            return this->task.runtime->write(std::move(frame));
        }

        // The start status waits for the streamer, so a client that waits for it can
        // send commands at once.
        void opened() override {
            std::lock_guard lock(this->task.state_mu);
            if (this->task.opening) this->task.answer_pending_starts();
        }

        void stopped_with_err(const x::errors::Error &err) override {
            std::lock_guard lock(this->task.state_mu);
            const std::string message = "Failed to stream commands to the Arc program";
            if (!this->task.opening) {
                this->task.state.send_error(err, message);
                return;
            }
            this->task.state.error(err, message);
            this->task.answer_pending_starts();
        }
    };

    Task(
        const synnax::task::Task &task_meta,
        const std::shared_ptr<task::Context> &ctx
    ):
        state(ctx, task_meta) {}

public:
    static std::pair<std::unique_ptr<Task>, x::errors::Error> create(
        const synnax::task::Task &task_meta,
        const std::shared_ptr<task::Context> &ctx,
        const TaskConfig &cfg,
        std::shared_ptr<pipeline::WriterFactory> writer_factory = nullptr,
        std::shared_ptr<pipeline::StreamerFactory> streamer_factory = nullptr,
        const std::shared_ptr<x::thread::rt::Manager> &rt_manager = nullptr
    ) {
        auto task = std::unique_ptr<Task>(new Task(task_meta, ctx));

        std::shared_ptr<x::thread::rt::Handle> rt_handle;
        if (rt_manager != nullptr) {
            x::thread::rt::Config base_rt;
            base_rt.enabled = true;
            base_rt.priority = ::arc::runtime::loop::DEFAULT_RT_PRIORITY;
            rt_handle = std::make_shared<x::thread::rt::Handle>(
                rt_manager->allocate(base_rt)
            );
        }

        const ::arc::runtime::Config runtime_cfg{
            .program = cfg.program,
            .breaker = x::breaker::default_config("arc_runtime"),
            .retrieve_channels = [client = ctx->client](
                                     const std::vector<::arc::types::ChannelKey> &keys
                                 )
                -> std::pair<
                    std::vector<::arc::runtime::state::ChannelDigest>,
                    x::errors::Error> {
                auto [channels, err] = client->channels.retrieve(keys);
                if (err) return {{}, err};
                std::vector<::arc::runtime::state::ChannelDigest> digests;
                for (const auto &ch: channels)
                    digests.push_back({ch.key, ch.data_type, ch.index});
                return {digests, x::errors::NIL};
            },
            .loop = cfg.loop,
            .factories =
                {
                    std::make_shared<::driver::arc::status::Module>(
                        ctx->client,
                        [task_ptr = task.get()](
                            const std::string &message,
                            const std::string &description
                        ) {
                            std::lock_guard lock(task_ptr->state_mu);
                            task_ptr->state.send_warning(message, description);
                        }
                    ),
                    std::make_shared<::driver::arc::ranges::Module>(
                        ctx->client,
                        [task_ptr = task.get()](
                            const std::string &message,
                            const std::string &description
                        ) {
                            std::lock_guard lock(task_ptr->state_mu);
                            task_ptr->state.send_warning(message, description);
                        }
                    ),
                },
            .rt_handle = rt_handle,
        };

        const auto fail = [task_ptr = task.get()](const x::errors::Error &err) {
            {
                std::lock_guard lock(task_ptr->state_mu);
                task_ptr->state.send_error(err, "Failed to run Arc program");
            }
            task_ptr->runtime->close_outputs();
        };
        auto [rt, err] = ::arc::runtime::load(
            runtime_cfg,
            fail,
            [task_ptr = task.get(),
             fail](const ::arc::ir::Node &node, const x::errors::Error &err) {
                const auto description = "The " + node.type +
                                         " node failed: " + err.data;
                if (err.matches(::arc::runtime::errors::WARNING)) {
                    std::lock_guard lock(task_ptr->state_mu);
                    task_ptr->state.send_warning("Node failed", description);
                } else
                    fail(x::errors::Error(err, description));
            }
        );
        if (err) return {nullptr, err};

        task->runtime = std::move(rt);

        if (!writer_factory)
            writer_factory = bypass::pipeline::create_writer_factory(ctx);
        if (!streamer_factory)
            streamer_factory = bypass::pipeline::create_streamer_factory(
                ctx,
                {task_meta.name, task_meta.name}
            );
        if (!task->runtime->write_channels.empty()) {
            auto source = std::make_unique<Source>(*task);
            auto initial_authorities = ::arc::runtime::build_authorities(
                cfg.program.authorities,
                task->runtime->write_channels
            );
            task->acquisition = std::make_unique<pipeline::Acquisition>(
                writer_factory,
                synnax::framer::WriterConfig{
                    .channels = task->runtime->write_channels,
                    .authorities = std::move(initial_authorities),
                    .subject =
                        x::control::Subject{
                            .key = task_meta.key.to_string(),
                            .name = task_meta.name,
                        },
                    .mode = common::data_saving_writer_mode(cfg.data_saving_disabled),
                },
                std::move(source),
                x::breaker::default_config("arc_acquisition"),
                "arc_acquisition",
                /* err_on_unauthorized */ false
            );
        }
        if (!task->runtime->read_channels.empty()) {
            auto sink = std::make_unique<Sink>(*task);
            task->control = std::make_unique<pipeline::Control>(
                streamer_factory,
                synnax::framer::StreamerConfig{
                    .channels = task->runtime->read_channels,
                },
                std::move(sink),
                x::breaker::default_config("arc_control"),
                "arc_control"
            );
        }

        return {std::move(task), x::errors::NIL};
    }

    /// @brief starts the task. A task whose runtime is already running is not
    /// restarted: the command is answered with the current status.
    bool start(const std::string &cmd_key) {
        if (!this->runtime->start()) {
            std::lock_guard lock(this->state_mu);
            if (this->opening)
                this->pending_starts.push_back(cmd_key);
            else
                this->state.ack(cmd_key, true);
            return false;
        }
        auto acq_started = true;
        if (this->acquisition) acq_started = this->acquisition->start();
        if (!this->control) {
            std::lock_guard lock(this->state_mu);
            this->state.send_start(cmd_key);
            return acq_started;
        }
        {
            std::lock_guard lock(this->state_mu);
            this->opening = true;
            this->pending_starts = {cmd_key};
        }
        return this->control->start() && acq_started;
    }

    bool stop(const std::string &cmd_key, const bool propagate_state) {
        auto control_stopped = true;
        if (this->control) control_stopped = this->control->stop();
        this->runtime->close_outputs();
        auto acq_stopped = true;
        if (this->acquisition) acq_stopped = this->acquisition->stop();
        const auto runtime_stopped = this->runtime->stop();
        std::lock_guard lock(this->state_mu);
        // A start the streamer never answered still needs its reply.
        for (const auto &key: this->pending_starts)
            this->state.ack(key, false);
        this->pending_starts.clear();
        this->opening = false;
        if (propagate_state) this->state.send_stop(cmd_key);
        return control_stopped && acq_stopped && runtime_stopped;
    }

    void exec(synnax::task::Command &cmd) override {
        if (cmd.type == "start")
            this->start(cmd.key);
        else if (cmd.type == "stop")
            this->stop(cmd.key, true);
        else
            LOG(WARNING) << "[arc] unknown command type: " << cmd.type;
    }

    void stop(const bool will_reconfigure) override {
        if (will_reconfigure) this->state.close();
        this->stop("", !will_reconfigure);
    }

    [[nodiscard]] std::string name() const override { return "Arc Runtime Task"; }
};
}
