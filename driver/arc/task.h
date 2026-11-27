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
#include <memory>
#include <mutex>
#include <optional>
#include <thread>
#include <variant>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/arc/arc.h"
#include "client/cpp/arc/task/json.gen.h"
#include "client/cpp/synnax.h"
#include "client/cpp/task/json.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/json/json.h"
#include "x/cpp/strings/strings.h"
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
/// @brief the separator between the keys of a scope path in a status message.
inline const std::string PATH_SEPARATOR = " › ";

/// @brief returns the status message and description that show the active top-level
/// scopes of a program.
inline std::pair<std::string, std::string>
scope_text(const synnax::arc::task::StatusData &data) {
    if (data.scopes.empty())
        return {"Idle", "No sequence or stage runs. A trigger can start one."};
    std::vector<std::string> paths;
    for (const auto &sc: data.scopes)
        paths.push_back(x::strings::join(sc.path, PATH_SEPARATOR));
    return {"Running " + x::strings::join(paths, ", "), ""};
}

/// @brief configuration for an arc runtime task.
struct TaskConfig : ::synnax::task::PersistConfig {
    x::uuid::UUID arc_key;
    ::arc::program::Program program;
    ::arc::runtime::loop::Config loop;
    /// @brief loop iterations and function calls that the program can run in one
    /// cycle. Not read from the task's JSON config.
    std::int64_t loop_budget = ::arc::runtime::wasm::DEFAULT_LOOP_BUDGET;

    TaskConfig(TaskConfig &&other) noexcept:
        ::synnax::task::PersistConfig(std::move(other)),
        arc_key(other.arc_key),
        program(std::move(other.program)),
        loop(other.loop),
        loop_budget(other.loop_budget) {}

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
        auto [arc_data, arc_err] = client->arcs.retrieve(
            cfg.arc_key,
            synnax::arc::RetrieveOptions{.include_program = true}
        );
        if (arc_err) return {std::move(cfg), arc_err};
        const auto *automation = std::get_if<synnax::arc::AutomationArc>(&arc_data);
        if (automation == nullptr)
            return {
                std::move(cfg),
                x::errors::Error(
                    x::errors::VALIDATION,
                    "Modules cannot be deployed independently. Import \"" +
                        synnax::arc::base(arc_data).name +
                        "\" from an automation instead."
                )
            };
        if (!automation->program.has_value())
            return {
                std::move(cfg),
                x::errors::Error(
                    x::errors::INTERNAL,
                    "The Arc program is not compiled."
                )
            };
        cfg.program = *automation->program;
        return {std::move(cfg), x::errors::NIL};
    }
};

/// @brief arc runtime task that manages both read and write pipelines.
class Task final : public task::Task {
    std::shared_ptr<::arc::runtime::Runtime> runtime;
    /// @brief allocates the RT core of each run. Null for a loop that runs unpinned.
    std::shared_ptr<x::thread::rt::Manager> rt_manager;
    x::thread::rt::Config rt_config;
    /// @brief the RT core of the run. The loop holds the handle from load, and the task
    /// fills it at each start and releases it at each stop, so a task built beside a
    /// running one takes no core.
    std::shared_ptr<x::thread::rt::Handle> rt_handle;
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
    /// @brief the active top-level scopes of the program, or nullopt before its first
    /// cycle in this run. Requires state_mu.
    std::optional<synnax::arc::task::StatusData> scopes;
    /// @brief guards pending_scopes and scopes_closed.
    std::mutex scopes_mu;
    /// @brief wakes scopes_thread when scopes wait or the task closes.
    std::condition_variable scopes_cv;
    /// @brief the latest scopes that the runtime reported, until scopes_thread takes
    /// them.
    std::optional<synnax::arc::task::StatusData> pending_scopes;
    /// @brief true once the task closes, which ends scopes_thread.
    bool scopes_closed = false;
    /// @brief shows the scopes that the runtime reports, so the runtime thread never
    /// waits on a status lock.
    std::thread scopes_thread;

    /// @brief answers every pending start with the current status. Requires state_mu.
    void answer_pending_starts() {
        for (const auto &key: this->pending_starts)
            this->state.send_start(key);
        this->pending_starts.clear();
        this->opening = false;
        this->send_scopes();
    }

    /// @brief shows the active scopes in the status of a task that runs. Requires
    /// state_mu.
    void send_scopes() {
        if (!this->scopes.has_value()) return;
        const auto [message, description] = scope_text(*this->scopes);
        this->state.send_running(
            message,
            description,
            this->scopes->to_json().get<x::json::json::object_t>()
        );
    }

    /// @brief hands the active paths of the program to scopes_thread. Runs on the
    /// runtime thread, and never blocks on a status write.
    void set_scopes(const std::vector<::arc::runtime::scheduler::Path> &paths) {
        synnax::arc::task::StatusData data;
        for (const auto &p: paths)
            data.scopes.push_back({.path = p});
        {
            std::lock_guard lock(this->scopes_mu);
            this->pending_scopes = std::move(data);
        }
        this->scopes_cv.notify_one();
    }

    /// @brief shows each pending set of scopes until the task closes.
    void run_scopes() {
        std::unique_lock lock(this->scopes_mu);
        while (true) {
            this->scopes_cv.wait(lock, [this] {
                return this->scopes_closed || this->pending_scopes.has_value();
            });
            if (this->scopes_closed) return;
            auto data = std::move(*this->pending_scopes);
            this->pending_scopes.reset();
            lock.unlock();
            {
                std::lock_guard state_lock(this->state_mu);
                this->scopes = std::move(data);
                this->send_scopes();
            }
            lock.lock();
        }
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
            const auto err = this->task.runtime->write(std::move(frame));
            // A runtime that stopped itself already reported why, so the frame drops.
            if (err.matches(::arc::runtime::errors::CLOSED)) return x::errors::NIL;
            return err;
        }

        // The start status and the runtime wait for the streamer. A client can then
        // send commands at once, and the program reads its own first writes.
        void opened() override {
            this->task.start_runtime();
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

    /// @brief starts the runtime and the pipeline that writes its outputs.
    /// @returns false if the runtime already runs.
    bool start_runtime() {
        if (this->runtime->running()) return false;
        if (this->rt_manager != nullptr)
            *this->rt_handle = this->rt_manager->allocate(this->rt_config);
        if (!this->runtime->start()) return false;
        return !this->acquisition || this->acquisition->start();
    }

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

        if (rt_manager != nullptr) {
            task->rt_manager = rt_manager;
            task->rt_config.enabled = true;
            task->rt_config.priority = ::arc::runtime::loop::DEFAULT_RT_PRIORITY;
            task->rt_handle = std::make_shared<x::thread::rt::Handle>(
                x::thread::rt::CPU_AFFINITY_NONE,
                task->rt_config,
                nullptr
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
            .rt_handle = task->rt_handle,
            .report =
                [task_ptr = task.get()](
                    const std::string &message,
                    const std::string &description
                ) {
                    std::lock_guard lock(task_ptr->state_mu);
                    task_ptr->state.send_warning(message, description);
                },
            .loop_budget = cfg.loop_budget,
            .on_scopes = [task_ptr = task.get()](
                             const std::vector<::arc::runtime::scheduler::Path> &paths
                         ) { task_ptr->set_scopes(paths); },
        };

        const auto fail = [task_ptr = task.get()](const x::errors::Error &err) {
            const auto *message = err.matches(::arc::runtime::errors::LOOP_BUDGET)
                                    ? "Program ran too long"
                                    : "Failed to run Arc program";
            {
                std::lock_guard lock(task_ptr->state_mu);
                task_ptr->state.send_error(err, message);
            }
            task_ptr->runtime->close_outputs();
        };
        auto [rt, err] = ::arc::runtime::load(
            runtime_cfg,
            fail,
            [task_ptr = task.get(),
             fail](const ::arc::ir::Node &node, const x::errors::Error &err) {
                const auto description = "The " + node.type +
                                         " node failed: " + err.data();
                if (err.matches(::arc::runtime::errors::LOOP_BUDGET)) {
                    fail(x::errors::Error(err, description));
                    return;
                }
                std::lock_guard lock(task_ptr->state_mu);
                task_ptr->state.send_warning("Node failed", description);
            }
        );
        if (err) return {nullptr, err};

        task->runtime = std::move(rt);
        task->scopes_thread = std::thread([task_ptr = task.get()] {
            task_ptr->run_scopes();
        });

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
        {
            std::lock_guard lock(this->state_mu);
            if (this->opening) {
                this->pending_starts.push_back(cmd_key);
                return false;
            }
            if (this->runtime->running()) {
                this->state.ack(cmd_key, true);
                return false;
            }
            this->scopes.reset();
            if (this->control) {
                this->opening = true;
                this->pending_starts = {cmd_key};
            }
        }
        if (this->control) {
            if (this->control->start()) return true;
            // The pipeline of a start that failed can still be on its way out.
            std::lock_guard lock(this->state_mu);
            for (const auto &key: this->pending_starts)
                this->state.ack(key, false);
            this->pending_starts.clear();
            this->opening = false;
            return false;
        }
        const auto started = this->start_runtime();
        std::lock_guard lock(this->state_mu);
        this->state.send_start(cmd_key);
        this->send_scopes();
        return started;
    }

    bool stop(const std::string &cmd_key, const bool propagate_state) {
        auto control_stopped = true;
        if (this->control) control_stopped = this->control->stop();
        this->runtime->close_outputs();
        auto acq_stopped = true;
        if (this->acquisition) acq_stopped = this->acquisition->stop();
        const auto runtime_stopped = this->runtime->stop();
        if (this->rt_handle != nullptr) this->rt_handle->release();
        std::lock_guard lock(this->state_mu);
        // A start the streamer never answered still needs its reply.
        for (const auto &key: this->pending_starts)
            this->state.ack(key, false);
        this->pending_starts.clear();
        this->opening = false;
        {
            std::lock_guard scopes_lock(this->scopes_mu);
            this->pending_scopes.reset();
        }
        this->scopes.reset();
        this->state.set_data(std::nullopt);
        if (propagate_state) this->state.send_stop(cmd_key);
        return control_stopped && acq_stopped && runtime_stopped;
    }

    void exec(synnax::task::Command &cmd, x::breaker::Breaker &) override {
        if (cmd.type == synnax::task::START_CMD_TYPE)
            this->start(cmd.key);
        else if (cmd.type == synnax::task::STOP_CMD_TYPE)
            this->stop(cmd.key, true);
        else
            LOG(WARNING) << "[arc] unknown command type: " << cmd.type;
    }

    void stop(const bool will_reconfigure) override {
        if (will_reconfigure) this->state.close();
        this->stop("", !will_reconfigure);
    }

    [[nodiscard]] std::string name() const override { return "Arc Runtime Task"; }

    ~Task() override {
        {
            std::lock_guard lock(this->scopes_mu);
            this->scopes_closed = true;
        }
        this->scopes_cv.notify_one();
        if (this->scopes_thread.joinable()) this->scopes_thread.join();
    }
};
}
