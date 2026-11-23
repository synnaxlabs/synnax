// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <string>
#include <variant>
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
#include "driver/arc/calculator.h"
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
/// @brief retrieves the automation with the given key, with its compiled program and
/// calculations.
inline std::pair<synnax::arc::AutomationArc, x::errors::Error>
retrieve_automation(const synnax::Synnax &client, const x::uuid::UUID &key) {
    auto [arc_data, err] = client.arcs.retrieve(
        key,
        synnax::arc::RetrieveOptions{.include_program = true}
    );
    if (err) return {{}, err};
    auto *automation = std::get_if<synnax::arc::AutomationArc>(&arc_data);
    if (automation == nullptr)
        return {
            {},
            x::errors::Error(
                x::errors::VALIDATION,
                "Modules cannot be deployed independently. Import \"" +
                    synnax::arc::base(arc_data).name + "\" from an automation instead."
            )
        };
    if (!automation->program.has_value())
        return {
            {},
            x::errors::Error(x::errors::INTERNAL, "The Arc program is not compiled.")
        };
    return {std::move(*automation), x::errors::NIL};
}

/// @brief configuration for an arc runtime task.
struct TaskConfig : ::synnax::task::PersistConfig {
    x::uuid::UUID arc_key;
    ::arc::program::Program program;
    /// @brief the calculated channels that the program reads, dependencies first.
    std::vector<synnax::arc::Calculation> calculations;
    ::arc::runtime::loop::Config loop;
    /// @brief loop iterations and function calls that the program can run in one
    /// cycle. Not read from the task's JSON config.
    std::int64_t loop_budget = ::arc::runtime::wasm::DEFAULT_LOOP_BUDGET;

    TaskConfig(TaskConfig &&other) noexcept:
        ::synnax::task::PersistConfig(std::move(other)),
        arc_key(other.arc_key),
        program(std::move(other.program)),
        calculations(std::move(other.calculations)),
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
        auto [automation, err] = retrieve_automation(*client, cfg.arc_key);
        if (err) return {std::move(cfg), err};
        cfg.program = std::move(*automation.program);
        cfg.calculations = std::move(automation.calculations);
        return {std::move(cfg), x::errors::NIL};
    }
};

/// @brief arc runtime task that manages both read and write pipelines.
class Task final : public task::Task {
    std::shared_ptr<::arc::runtime::Runtime> runtime;
    /// @brief computes calculated channels on the control thread.
    Calculator calculator;
    /// @brief guards reloaded.
    std::mutex reload_mu;
    /// @brief a calculator to swap in before the next compute.
    std::optional<Calculator> reloaded;
    /// @brief true while reloaded holds a calculator. Lets the control thread skip
    /// reload_mu on each frame.
    std::atomic<bool> has_reloaded = false;
    std::unique_ptr<pipeline::Acquisition> acquisition;
    std::unique_ptr<pipeline::Control> control;
    /// @brief reloads the calculations when they change.
    std::unique_ptr<pipeline::Control> watch;
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
            if (this->task.has_reloaded.load(std::memory_order_acquire)) {
                std::lock_guard lock(this->task.reload_mu);
                this->task.calculator = std::move(*this->task.reloaded);
                this->task.reloaded.reset();
                this->task.has_reloaded.store(false, std::memory_order_release);
            }
            this->task.calculator.compute(frame);
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

    /// @brief sink that loads the calculations again when a channel they use changes,
    /// and each time its streamer opens, so that edits made during an outage apply. It
    /// runs on its own thread, so the control thread never waits on the Core.
    class Watch final : public pipeline::Sink {
        Task &task;
        std::shared_ptr<synnax::Synnax> client;
        x::uuid::UUID arc_key;
        synnax::channel::Key set_key;
        synnax::channel::Key delete_key;
        ::arc::runtime::Config cfg;
        ::arc::runtime::scheduler::ErrorHandler on_error;
        /// @brief the calculations last loaded.
        x::json::json loaded;
        /// @brief the channels whose change reloads the calculations.
        std::set<::arc::types::ChannelKey> watched;
        /// @brief the channels that the control pipeline streams.
        std::vector<::arc::types::ChannelKey> streamed;

        /// @brief returns the channels that calc uses and the channels the Arc reads,
        /// so that an edit of a channel that calc leaves out still reloads.
        [[nodiscard]] std::set<::arc::types::ChannelKey>
        watch(const Calculator &calc) const {
            auto keys = calc.channels();
            const auto &reads = this->task.runtime->read_channels;
            keys.insert(reads.begin(), reads.end());
            return keys;
        }

        [[nodiscard]] bool changed(const x::telem::Frame &frame) const {
            for (const auto [key, series]: frame)
                if (key == this->delete_key) {
                    for (const auto k: series.values<synnax::channel::Key>())
                        if (this->watched.contains(k)) return true;
                } else if (key == this->set_key)
                    for (const auto &ch: series.json_values())
                        if (this->watched.contains(
                                ch.at("key").get<synnax::channel::Key>()
                            ))
                            return true;
            return false;
        }

        x::errors::Error reload() {
            auto [automation, err] = retrieve_automation(*this->client, this->arc_key);
            if (err) return err;
            auto loaded = x::json::to_array(automation.calculations);
            if (loaded == this->loaded) return x::errors::NIL;
            auto [calc, load_err] = Calculator::load(
                automation.calculations,
                this->cfg,
                this->on_error
            );
            if (load_err) return load_err;
            this->loaded = std::move(loaded);
            this->watched = this->watch(calc);
            auto streamed = calc.streamed(this->task.runtime->read_channels);
            // The streamer cannot change its channels, so it opens again with the new
            // ones. Frames sent while it reopens are lost.
            const auto restart = streamed != this->streamed;
            auto running = false;
            if (restart) {
                running = this->task.control->stop();
                this->task.control->set_channels(streamed);
                this->streamed = std::move(streamed);
            }
            {
                std::lock_guard lock(this->task.reload_mu);
                this->task.reloaded = std::move(calc);
                this->task.has_reloaded.store(true, std::memory_order_release);
            }
            if (running) this->task.control->start();
            return x::errors::NIL;
        }

        void warn(const x::errors::Error &err, const std::string &message) const {
            if (!err) return;
            std::lock_guard lock(this->task.state_mu);
            this->task.state.send_warning(
                message,
                "The Arc keeps its previous calculations. " + err.data
            );
        }

    public:
        Watch(
            Task &task,
            std::shared_ptr<synnax::Synnax> client,
            const x::uuid::UUID &arc_key,
            const synnax::channel::Key set_key,
            const synnax::channel::Key delete_key,
            ::arc::runtime::Config cfg,
            ::arc::runtime::scheduler::ErrorHandler on_error,
            const std::vector<synnax::arc::Calculation> &calculations
        ):
            task(task),
            client(std::move(client)),
            arc_key(arc_key),
            set_key(set_key),
            delete_key(delete_key),
            cfg(std::move(cfg)),
            on_error(std::move(on_error)),
            loaded(x::json::to_array(calculations)),
            watched(this->watch(task.calculator)),
            streamed(task.calculator.streamed(task.runtime->read_channels)) {}

        x::errors::Error write(x::telem::Frame &frame) override {
            if (this->changed(frame))
                this->warn(this->reload(), "Failed to reload calculations");
            return x::errors::NIL;
        }

        void opened() override {
            this->warn(this->reload(), "Failed to reload calculations");
        }

        void stopped_with_err(const x::errors::Error &err) override {
            this->warn(err, "Failed to watch calculations");
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
        const std::shared_ptr<x::thread::rt::Manager> &rt_manager = nullptr,
        std::shared_ptr<pipeline::StreamerFactory> watch_factory = nullptr
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
            .report =
                [task_ptr = task.get()](
                    const std::string &message,
                    const std::string &description
                ) {
                    std::lock_guard lock(task_ptr->state_mu);
                    task_ptr->state.send_warning(message, description);
                },
            .loop_budget = cfg.loop_budget,
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
                                         " node failed: " + err.data;
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
        const auto on_calc_error = [task_ptr = task.get()](
                                       const ::arc::ir::Node &node,
                                       const x::errors::Error &err
                                   ) {
            std::lock_guard lock(task_ptr->state_mu);
            task_ptr->state.send_warning(
                "Failed to run calculation",
                "The " + node.type + " node failed: " + err.data
            );
        };
        auto [calculator, calc_err] = Calculator::load(
            cfg.calculations,
            runtime_cfg,
            on_calc_error
        );
        if (calc_err) return {nullptr, calc_err};
        task->calculator = std::move(calculator);

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
                    .channels = task->calculator.streamed(task->runtime->read_channels),
                },
                std::move(sink),
                x::breaker::default_config("arc_control"),
                "arc_control"
            );
        }
        if (task->control && !cfg.calculations.empty()) {
            auto [signals, signals_err] = ctx->client->channels.retrieve(
                std::vector{
                    synnax::channel::SET_CHANNEL,
                    synnax::channel::DELETE_CHANNEL,
                }
            );
            if (signals_err) return {nullptr, signals_err};
            synnax::channel::Key set_key = 0;
            synnax::channel::Key delete_key = 0;
            for (const auto &ch: signals)
                if (ch.name == synnax::channel::SET_CHANNEL)
                    set_key = ch.key;
                else if (ch.name == synnax::channel::DELETE_CHANNEL)
                    delete_key = ch.key;
            if (!watch_factory)
                watch_factory = std::make_shared<pipeline::SynnaxStreamerFactory>(
                    ctx->client
                );
            auto calc_cfg = runtime_cfg;
            calc_cfg.program = {};
            task->watch = std::make_unique<pipeline::Control>(
                std::move(watch_factory),
                synnax::framer::StreamerConfig{.channels = {set_key, delete_key}},
                std::make_unique<Watch>(
                    *task,
                    ctx->client,
                    cfg.arc_key,
                    set_key,
                    delete_key,
                    std::move(calc_cfg),
                    on_calc_error,
                    cfg.calculations
                ),
                x::breaker::default_config("arc_watch"),
                "arc_watch"
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
            if (this->control) {
                this->opening = true;
                this->pending_starts = {cmd_key};
            }
        }
        if (this->control) {
            // A watch can outlive a streamer that failed to open, and it changes the
            // calculator and the control pipeline.
            if (this->watch) this->watch->stop();
            // No compute runs while the control pipeline is stopped.
            if (const auto err = this->calculator.reset()) {
                std::lock_guard lock(this->state_mu);
                this->state.error(err, "Failed to load calculations");
                this->answer_pending_starts();
                return false;
            }
            if (this->control->start()) {
                if (this->watch) this->watch->start();
                return true;
            }
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
        return started;
    }

    bool stop(const std::string &cmd_key, const bool propagate_state) {
        // The watch stops first, as it can restart the control pipeline.
        if (this->watch) this->watch->stop();
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
};
}
