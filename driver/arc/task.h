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
#include <condition_variable>
#include <memory>
#include <mutex>
#include <set>
#include <string>
#include <thread>
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
/// @brief retrieves the compiled program of the automation with the given key.
inline std::pair<::arc::program::Program, x::errors::Error>
retrieve_program(const synnax::Synnax &client, const x::uuid::UUID &key) {
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
    return {std::move(*automation->program), x::errors::NIL};
}

/// @brief configuration for an arc runtime task.
struct TaskConfig : ::synnax::task::PersistConfig {
    x::uuid::UUID arc_key;
    ::arc::program::Program program;
    ::arc::runtime::loop::Config loop;
    /// @brief loop iterations and function calls that the program can run in one
    /// cycle. Not read from the task's JSON config.
    std::int64_t loop_budget = ::arc::runtime::wasm::DEFAULT_LOOP_BUDGET;
    /// @brief how long to wait after a calculated channel changes before loading the
    /// calculations again. A Driver setting, not read from the task's JSON config.
    x::telem::TimeSpan reload_delay = 100 * x::telem::MILLISECOND;

    TaskConfig(TaskConfig &&other) noexcept:
        ::synnax::task::PersistConfig(std::move(other)),
        arc_key(other.arc_key),
        program(std::move(other.program)),
        loop(other.loop),
        loop_budget(other.loop_budget),
        reload_delay(other.reload_delay) {}

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
        auto [program, err] = retrieve_program(*client, cfg.arc_key);
        if (err) return {std::move(cfg), err};
        cfg.program = std::move(program);
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
    std::unique_ptr<Calculator> reloaded;
    /// @brief true while reloaded holds a calculator. Lets the control thread skip
    /// reload_mu on each frame.
    std::atomic<bool> has_reloaded = false;
    std::unique_ptr<pipeline::Acquisition> acquisition;
    std::unique_ptr<pipeline::Control> control;
    class Reloader;
    /// @brief loads the calculations again when they change.
    std::unique_ptr<Reloader> reloader;
    /// @brief streams the channel changes that ask the reloader for a reload.
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
                const auto next = std::move(this->task.reloaded);
                this->task.calculator = std::move(*next);
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

    /// @brief loads the calculations again on its own thread, so the control thread
    /// never waits on the Core. Requests that come within the reload delay of each
    /// other load once.
    class Reloader {
        Task &task;
        std::shared_ptr<synnax::Synnax> client;
        ::arc::runtime::Config cfg;
        ::arc::runtime::scheduler::ErrorHandler on_error;
        x::telem::TimeSpan delay;
        /// @brief guards watched, omitted, requested, running, and thread.
        mutable std::mutex mu;
        std::condition_variable cv;
        /// @brief the channels whose change or delete reloads the calculations. It
        /// only grows, so that a calculation the Core leaves out stays watched.
        std::set<::arc::types::ChannelKey> watched;
        /// @brief true while the Core leaves out a calculation that the Arc reads. A
        /// channel created or renamed to a name it uses can bring it back, so then any
        /// channel set reloads.
        bool omitted = false;
        bool requested = false;
        bool running = false;
        std::thread thread;
        /// @brief the calculations last loaded.
        x::json::json loaded;
        /// @brief each calculated channel last loaded and the channels it reads.
        std::vector<
            std::pair<::arc::types::ChannelKey, std::set<::arc::types::ChannelKey>>>
            dependencies;
        /// @brief the channels that the control pipeline streams.
        std::vector<::arc::types::ChannelKey> streamed;

        void run() {
            std::unique_lock lock(this->mu);
            while (true) {
                this->cv.wait(lock, [this] {
                    return this->requested || !this->running;
                });
                if (!this->running) return;
                this->cv.wait_for(lock, this->delay.chrono(), [this] {
                    return !this->running;
                });
                if (!this->running) return;
                this->requested = false;
                lock.unlock();
                if (const auto err = this->reload()) {
                    std::lock_guard state_lock(this->task.state_mu);
                    this->task.state.send_warning(
                        "Failed to reload calculations",
                        "The Arc keeps its previous calculations. " + err.data
                    );
                }
                lock.lock();
            }
        }

        /// @returns an error if the Arc, or a calculation in next, still needs a
        /// calculation that the Core left out, as the Core keeps computing its last
        /// good version. A calculation whose channel, or a channel it reads, was
        /// deleted can go.
        [[nodiscard]] x::errors::Error check_dropped(const Calculator &next) const {
            const auto &arc_reads = this->task.runtime->read_channels;
            std::set needed(arc_reads.begin(), arc_reads.end());
            std::set<::arc::types::ChannelKey> kept;
            for (const auto &[key, reads]: next.dependencies()) {
                kept.insert(key);
                needed.insert(reads.begin(), reads.end());
            }
            std::set<::arc::types::ChannelKey> gone;
            for (const auto &[key, reads]: this->dependencies) {
                if (kept.contains(key)) continue;
                if (std::ranges::any_of(reads, [&](const auto k) {
                        return gone.contains(k);
                    })) {
                    gone.insert(key);
                    continue;
                }
                std::vector keys(reads.begin(), reads.end());
                keys.push_back(key);
                const auto [channels, err] = this->client->channels.retrieve(keys);
                if (err.matches(x::errors::NOT_FOUND)) {
                    gone.insert(key);
                    continue;
                }
                if (err) return err;
                if (needed.contains(key))
                    return x::errors::Error(
                        x::errors::VALIDATION,
                        "The Core cannot compute " + channels.back().name +
                            ". Fix its expression."
                    );
            }
            return x::errors::NIL;
        }

        x::errors::Error reload() {
            auto [compiled, err] = this->client->arcs.calculations(
                this->task.runtime->read_channels
            );
            if (err) return err;
            {
                std::lock_guard lock(this->mu);
                this->watched.insert(
                    compiled.channels.begin(),
                    compiled.channels.end()
                );
                this->omitted = !compiled.omitted.empty();
            }
            auto loaded = x::json::to_array(compiled.calculations);
            if (loaded == this->loaded) return x::errors::NIL;
            auto [calc, load_err] = Calculator::load(
                compiled.calculations,
                this->cfg,
                this->on_error
            );
            if (load_err) return load_err;
            if (const auto drop_err = this->check_dropped(calc)) return drop_err;
            auto streamed = calc.streamed(this->task.runtime->read_channels);
            if (streamed != this->streamed) {
                if (const auto set_err = this->task.control->set_channels(streamed))
                    return set_err;
                this->streamed = std::move(streamed);
            }
            this->loaded = std::move(loaded);
            this->dependencies = calc.dependencies();
            std::lock_guard lock(this->task.reload_mu);
            this->task.reloaded = std::make_unique<Calculator>(std::move(calc));
            this->task.has_reloaded.store(true, std::memory_order_release);
            return x::errors::NIL;
        }

    public:
        Reloader(
            Task &task,
            std::shared_ptr<synnax::Synnax> client,
            ::arc::runtime::Config cfg,
            ::arc::runtime::scheduler::ErrorHandler on_error,
            const x::telem::TimeSpan delay,
            const synnax::arc::Calculations &compiled
        ):
            task(task),
            client(std::move(client)),
            cfg(std::move(cfg)),
            on_error(std::move(on_error)),
            delay(delay),
            watched(compiled.channels.begin(), compiled.channels.end()),
            omitted(!compiled.omitted.empty()),
            loaded(x::json::to_array(compiled.calculations)),
            dependencies(task.calculator.dependencies()),
            streamed(task.calculator.streamed(task.runtime->read_channels)) {
            this->watched.insert(
                task.runtime->read_channels.begin(),
                task.runtime->read_channels.end()
            );
        }

        /// @brief starts the reload thread. Does nothing if it runs.
        void start() {
            std::lock_guard lock(this->mu);
            if (this->running) return;
            this->running = true;
            this->requested = false;
            this->thread = std::thread(&Reloader::run, this);
        }

        /// @brief stops the reload thread and waits for a reload in progress. Safe to
        /// call from any thread but the reload thread.
        void stop() {
            std::thread done;
            {
                std::lock_guard lock(this->mu);
                this->running = false;
                done = std::move(this->thread);
            }
            this->cv.notify_all();
            if (done.joinable()) done.join();
        }

        /// @brief asks for a reload after the delay.
        void request() {
            {
                std::lock_guard lock(this->mu);
                this->requested = true;
            }
            this->cv.notify_all();
        }

        /// @brief returns true if frame, from the channel set or delete channel,
        /// changes a watched channel, or sets any channel while a calculation is
        /// omitted.
        [[nodiscard]] bool changes(
            const x::telem::Frame &frame,
            const synnax::channel::Key set_key,
            const synnax::channel::Key delete_key
        ) const {
            std::lock_guard lock(this->mu);
            for (const auto [key, series]: frame)
                if (key == delete_key) {
                    for (const auto k: series.values<synnax::channel::Key>())
                        if (this->watched.contains(k)) return true;
                } else if (key == set_key) {
                    if (this->omitted && !series.empty()) return true;
                    for (const auto &ch: series.json_values())
                        if (this->watched.contains(
                                ch.at("key").get<synnax::channel::Key>()
                            ))
                            return true;
                }
            return false;
        }
    };

    /// @brief sink that asks for a reload of the calculations when a channel they use
    /// changes, and each time its streamer opens, so that edits made during an outage
    /// apply.
    class Watch final : public pipeline::Sink {
        Task &task;
        synnax::channel::Key set_key;
        synnax::channel::Key delete_key;

    public:
        Watch(
            Task &task,
            const synnax::channel::Key set_key,
            const synnax::channel::Key delete_key
        ):
            task(task), set_key(set_key), delete_key(delete_key) {}

        x::errors::Error write(x::telem::Frame &frame) override {
            if (this->task.reloader->changes(frame, this->set_key, this->delete_key))
                this->task.reloader->request();
            return x::errors::NIL;
        }

        void opened() override { this->task.reloader->request(); }

        void stopped_with_err(const x::errors::Error &err) override {
            std::lock_guard lock(this->task.state_mu);
            this->task.state.send_warning(
                "Failed to watch calculations",
                "The Arc keeps its previous calculations. " + err.data
            );
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
        synnax::arc::Calculations compiled;
        if (!task->runtime->read_channels.empty()) {
            auto [calcs, calcs_err] = ctx->client->arcs.calculations(
                task->runtime->read_channels
            );
            if (calcs_err) return {nullptr, calcs_err};
            compiled = std::move(calcs);
        }
        auto [calculator, calc_err] = Calculator::load(
            compiled.calculations,
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
        if (task->control && !compiled.channels.empty()) {
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
            task->reloader = std::make_unique<Reloader>(
                *task,
                ctx->client,
                std::move(calc_cfg),
                on_calc_error,
                cfg.reload_delay,
                compiled
            );
            task->watch = std::make_unique<pipeline::Control>(
                std::move(watch_factory),
                synnax::framer::StreamerConfig{.channels = {set_key, delete_key}},
                std::make_unique<Watch>(*task, set_key, delete_key),
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
            // No compute runs while the control pipeline is stopped. A reload that
            // came after the last compute is already fresh.
            if (this->has_reloaded.exchange(false, std::memory_order_acq_rel)) {
                std::lock_guard lock(this->reload_mu);
                const auto next = std::move(this->reloaded);
                this->calculator = std::move(*next);
            } else if (const auto err = this->calculator.reset()) {
                std::lock_guard lock(this->state_mu);
                this->state.error(err, "Failed to load calculations");
                this->answer_pending_starts();
                return false;
            }
            if (this->control->start()) {
                if (this->watch) {
                    this->reloader->start();
                    this->watch->start();
                }
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
        if (this->watch) {
            this->watch->stop();
            this->reloader->stop();
        }
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
