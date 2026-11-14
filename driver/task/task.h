// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <atomic>
#include <condition_variable>
#include <list>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <unordered_map>
#include <utility>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/log/log.h"

#include "driver/bypass/bypass.h"
#include "driver/control/state.h"
#include "driver/task/operation.h"
#include "driver/task/status_cache.h"

namespace driver::task {
/// @brief interface for a task that can be executed by the driver. Tasks should be
/// constructed by an @see Factory.
class Task {
public:
    /// @brief the key of the task
    synnax::task::Key key;

    [[nodiscard]] virtual std::string name() const { return ""; }

    /// @brief executes the command on the task. The task is responsible for
    /// updating its state.
    virtual void exec(synnax::task::Command &cmd) {}

    /// @brief stops the task, halting activities and freeing all resources. stop
    /// is called when the task is no longer needed, and is typically followed by a
    /// a call to the destructor.
    virtual void stop(bool will_reconfigure) = 0;

    virtual ~Task() = default;
};

/// @brief an interface for a standard context that is provided to every task in the
/// driver. This context provides access to the Synnax client and allows tasks to
/// easily update their state.
class Context {
    std::mutex rack_name_mu;
    std::string current_rack_name;

public:
    /// @brief the client used to communicate with the Synnax server.
    std::shared_ptr<synnax::Synnax> client;

    Context() = default;

    virtual ~Context() = default;

    explicit Context(std::shared_ptr<synnax::Synnax> client):
        client(std::move(client)) {}

    /// @brief returns the local telemetry bus, or nullptr if not available.
    virtual std::shared_ptr<bypass::Bus> bus() { return nullptr; }

    /// @brief returns the shared control authority states.
    virtual std::shared_ptr<control::States> control_states() { return nullptr; }

    /// @brief returns the rack key for this driver, used as the group identity for
    /// Core-side deduplication filtering.
    virtual synnax::rack::Key rack_key() { return 0; }

    /// @brief returns the current name of the driver's rack. Safe for concurrent use.
    std::string rack_name() {
        std::lock_guard lock{this->rack_name_mu};
        return this->current_rack_name;
    }

    /// @brief records a new name for the driver's rack. Safe for concurrent use.
    void set_rack_name(std::string name) {
        std::lock_guard lock{this->rack_name_mu};
        this->current_rack_name = std::move(name);
    }

    /// @brief updates the state of the task in the Synnax cluster.
    virtual void set_status(synnax::task::Status &status) = 0;

    /// @brief reports a step of a command, such as a slow bus start.
    /// @param status a loading status with the key of the command.
    /// @param limit how long the step may run before the task manager warns.
    virtual void step(synnax::task::Status &status, x::telem::TimeSpan limit) {
        this->set_status(status);
    }

    /// @brief sends the latest status of each task again, to repair statuses lost while
    /// the Driver could not reach the Core.
    /// @returns the first error of the writes. Call it again on error.
    [[nodiscard]] virtual x::errors::Error resend_statuses() = 0;

    /// @brief returns true when a status write failed, a re-send included, and no
    /// re-send has started since.
    [[nodiscard]] virtual bool statuses_stale() = 0;
};

/// @brief a mock context that can be used for testing tasks.
class MockContext final : public Context {
    std::mutex mu;

public:
    std::vector<synnax::task::Status> statuses;

    explicit MockContext(const std::shared_ptr<synnax::Synnax> &client):
        Context(client) {}

    void set_status(synnax::task::Status &status) override {
        mu.lock();
        statuses.push_back(status);
        mu.unlock();
    }

    x::errors::Error resend_statuses() override { return x::errors::NIL; }

    bool statuses_stale() override { return false; }

    /// @brief returns a copy of the statuses, safe to read while tasks send more.
    std::vector<synnax::task::Status> snapshot() {
        std::lock_guard lock(this->mu);
        return this->statuses;
    }
};

/// @brief the context of the task manager. A status that answers the command of an
/// open operation goes through that operation.
class SynnaxContext final : public Context {
    std::shared_ptr<bypass::Bus> bypass_bus;
    std::shared_ptr<control::States> states;
    synnax::rack::Key rack;
    std::shared_ptr<StatusCache> statuses;
    /// @brief guards operations.
    std::mutex operations_mu;
    /// @brief the open operations, by the key of their command.
    std::unordered_map<std::string, std::shared_ptr<Operation>> operations;

    /// @returns the open operation of the command that status answers, or null.
    std::shared_ptr<Operation> operation(const synnax::task::Status &status) {
        if (status.details.cmd.empty()) return nullptr;
        std::lock_guard lock{this->operations_mu};
        const auto it = this->operations.find(status.details.cmd);
        return it == this->operations.end() ? nullptr : it->second;
    }

    void write(synnax::task::Status &status) {
        if (status.time == 0) status.time = x::telem::TimeStamp::now();
        status.details.rack = this->rack;
        if (const auto err = this->statuses->set(status); err)
            LOG(ERROR) << "[task.context] failed to write task status update: " << err;
    }

public:
    SynnaxContext(
        const std::shared_ptr<synnax::Synnax> &client,
        std::shared_ptr<StatusCache> statuses,
        const std::shared_ptr<bypass::Bus> &bus = nullptr,
        const std::shared_ptr<control::States> &control_states = nullptr,
        const synnax::rack::Key rack_key = 0
    ):
        Context(client),
        bypass_bus(bus),
        states(control_states),
        rack(rack_key),
        statuses(std::move(statuses)) {}

    std::shared_ptr<bypass::Bus> bus() override { return this->bypass_bus; }

    std::shared_ptr<control::States> control_states() override { return this->states; }

    synnax::rack::Key rack_key() override { return this->rack; }

    void set_status(synnax::task::Status &status) override {
        if (const auto op = this->operation(status); op != nullptr)
            op->answer(status);
        else
            this->write(status);
    }

    void step(synnax::task::Status &status, const x::telem::TimeSpan limit) override {
        if (const auto op = this->operation(status); op != nullptr)
            op->step(status, limit);
        else
            this->write(status);
    }

    x::errors::Error resend_statuses() override { return this->statuses->resend(); }

    bool statuses_stale() override { return this->statuses->stale(); }

    /// @brief opens an operation that runs cmd on the task called name. The statuses
    /// that answer cmd go through it until end. The operation must not outlive this
    /// context.
    /// @param limit how long a step with no limit of its own may run.
    std::shared_ptr<Operation> begin(
        const std::string &name,
        const synnax::task::Command &cmd,
        const std::string &config_hash,
        const x::telem::TimeSpan limit
    ) {
        auto op = std::make_shared<Operation>(
            [this](synnax::task::Status &status) { this->write(status); },
            name,
            cmd,
            config_hash,
            limit
        );
        if (!cmd.key.empty()) {
            std::lock_guard lock{this->operations_mu};
            this->operations[cmd.key] = op;
        }
        return op;
    }

    /// @brief stops sending the statuses of op through it and closes op. A later
    /// status for its command is written as it is.
    void end(const std::shared_ptr<Operation> &op) {
        {
            std::lock_guard lock{this->operations_mu};
            const auto it = this->operations.find(op->cmd.key);
            if (it != this->operations.end() && it->second == op)
                this->operations.erase(it);
        }
        op->close();
    }
};

/// @brief the cmd_key given to configure_task when no command drives the configure,
/// as at boot. Nothing is waiting on the outcome.
inline constexpr auto NO_COMMAND = "";

class Factory {
public:
    /// @brief creates the tasks this factory runs on every rack, such as a scanner.
    /// @returns each created task with its live instance, or with null when its
    /// configure failed.
    virtual std::vector<std::pair<synnax::task::Task, std::unique_ptr<Task>>>
    configure_initial_tasks(
        const std::shared_ptr<Context> &ctx,
        const synnax::rack::Rack &rack
    ) {
        return {};
    }

    virtual std::string name() { return ""; }

    /// @brief builds a live instance of the task if this factory handles its type.
    /// @param cmd_key the start command driving the deploy, NO_COMMAND at boot.
    /// @returns the instance and whether this factory handled the type.
    virtual std::pair<std::unique_ptr<Task>, bool> configure_task(
        const std::shared_ptr<Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) = 0;

    virtual ~Factory() = default;
};

class MultiFactory final : public Factory {
    std::vector<std::unique_ptr<Factory>> factories;

public:
    explicit MultiFactory(std::vector<std::unique_ptr<Factory>> &&factories):
        factories(std::move(factories)) {}

    std::vector<std::pair<synnax::task::Task, std::unique_ptr<Task>>>
    configure_initial_tasks(
        const std::shared_ptr<Context> &ctx,
        const synnax::rack::Rack &rack
    ) override {
        std::vector<std::pair<synnax::task::Task, std::unique_ptr<Task>>> tasks;
        for (const auto &factory: factories) {
            const std::string factory_name = factory->name();
            VLOG(1) << "[" << factory_name << "] configuring initial tasks";
            auto new_tasks = factory->configure_initial_tasks(ctx, rack);
            VLOG(1) << "[" << factory_name << "] configured " << new_tasks.size()
                    << " initial tasks";
            for (auto &task: new_tasks)
                tasks.emplace_back(std::move(task));
        }
        return tasks;
    }

    std::pair<std::unique_ptr<Task>, bool> configure_task(
        const std::shared_ptr<Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override {
        for (const auto &factory: factories) {
            auto [t, ok] = factory->configure_task(ctx, task, cmd_key);
            if (ok) return {std::move(t), true};
        }
        return {nullptr, false};
    }
};

/// @brief configuration for the task manager.
struct ManagerConfig {
    /// @brief how long a step of a command may run before the task manager warns, for
    /// a step with no limit of its own.
    x::telem::TimeSpan op_timeout = 60 * x::telem::SECOND;
    /// @brief interval between checks of the step limits.
    x::telem::TimeSpan poll_interval = 1 * x::telem::SECOND;
    /// @brief max time to wait for workers during shutdown before detaching.
    x::telem::TimeSpan shutdown_timeout = 30 * x::telem::SECOND;
    /// @brief number of worker threads for task operations.
    size_t worker_count = 4;

    template<typename Parser>
    void override(Parser &p) {
        const auto op_timeout_s = p.field("op_timeout", this->op_timeout.seconds());
        this->op_timeout = x::telem::TimeSpan(static_cast<int64_t>(op_timeout_s * 1e9));
        const auto poll_interval_s = p.field(
            "poll_interval",
            this->poll_interval.seconds()
        );
        this->poll_interval = x::telem::TimeSpan(
            static_cast<int64_t>(poll_interval_s * 1e9)
        );
        const auto shutdown_timeout_s = p.field(
            "shutdown_timeout",
            this->shutdown_timeout.seconds()
        );
        this->shutdown_timeout = x::telem::TimeSpan(
            static_cast<int64_t>(shutdown_timeout_s * 1e9)
        );
        this->worker_count = p.field(
            "worker_count",
            static_cast<int>(this->worker_count)
        );
        this->worker_count = std::max<size_t>(this->worker_count, 1);
        this->worker_count = std::min<size_t>(this->worker_count, 64);
    }

    friend std::ostream &operator<<(std::ostream &os, const ManagerConfig &cfg) {
        os << "  " << x::log::SHALE() << "op timeout" << x::log::RESET() << ": "
           << cfg.op_timeout.seconds() << "s\n"
           << "  " << x::log::SHALE() << "poll interval" << x::log::RESET() << ": "
           << cfg.poll_interval.seconds() << "s\n"
           << "  " << x::log::SHALE() << "shutdown timeout" << x::log::RESET() << ": "
           << cfg.shutdown_timeout.seconds() << "s\n"
           << "  " << x::log::SHALE() << "worker count" << x::log::RESET() << ": "
           << cfg.worker_count;
        return os;
    }
};

/// @brief TaskManager is responsible for configuring, executing, and commanding
/// data acquisition and control tasks.
class Manager {
public:
    Manager(
        synnax::rack::Rack rack,
        const std::shared_ptr<synnax::Synnax> &client,
        std::unique_ptr<Factory> factory,
        const ManagerConfig &cfg = {}
    ):
        rack(std::move(rack)),
        control_states(std::make_shared<control::States>()),
        statuses(std::make_shared<StatusCache>(client->statuses)),
        ctx(std::make_shared<SynnaxContext>(
            client,
            this->statuses,
            std::make_shared<bypass::Bus>(),
            this->control_states,
            this->rack.key
        )),
        factory(std::move(factory)),
        op_timeout(cfg.op_timeout),
        poll_interval(cfg.poll_interval),
        shutdown_timeout(cfg.shutdown_timeout),
        worker_count(cfg.worker_count) {
        this->ctx->set_rack_name(this->rack.name);
    }

    /// @brief runs the main task manager loop, blocking until stop() is called.
    /// Safe to call stop() from another thread.
    x::errors::Error run(const std::function<void()> &on_started = nullptr);

    /// @brief stops the task manager, halting all tasks and freeing resources.
    void stop();

private:
    /// @brief the rack this manager belongs to.
    synnax::rack::Rack rack;
    /// @brief shared control authority states, fed by the manager's streamer.
    std::shared_ptr<control::States> control_states;
    /// @brief the latest status of each task on this Driver.
    std::shared_ptr<StatusCache> statuses;
    /// @brief shared context passed to all tasks.
    std::shared_ptr<SynnaxContext> ctx;
    /// @brief creates device-specific tasks.
    std::unique_ptr<Factory> factory;
    /// @brief how long a step of a command may run before the task manager warns, for
    /// a step with no limit of its own.
    x::telem::TimeSpan op_timeout;
    /// @brief interval between checks of the step limits.
    x::telem::TimeSpan poll_interval;
    /// @brief max time to wait for workers during shutdown before detaching.
    x::telem::TimeSpan shutdown_timeout;
    /// @brief number of worker threads for task operations.
    size_t worker_count;

    /// @brief an operation to be executed by a worker.
    struct Op {
        /// @brief types of operations that can be queued. RELEASE frees the live
        /// instance without a terminal status: a successor on another rack owns
        /// status reporting. A DEPLOY carrying the task body trusts it (boot);
        /// one without fetches the stored task on the worker, keeping the
        /// streamer loop off the network.
        enum class Type { DEPLOY, COMMAND, SHUTDOWN, REMOVE, RELEASE };
        Type type{};
        synnax::task::Key task_key;
        synnax::task::Task task;
        synnax::task::Command cmd;
    };

    /// @brief per-task state tracked by the manager. Exists exactly while the row is
    /// on this rack or this driver holds a live instance.
    struct Entry {
        /// @brief guards row and deployed_hash. Always taken after Manager::mu.
        std::mutex mu;
        /// @brief the last row seen for this task. Set events omit config, so a
        /// deploy must fetch the row instead of using this one.
        synnax::task::Task row;
        /// @brief the config hash instance was built from. Empty when instance is
        /// null.
        std::string deployed_hash;
        /// @brief the live instance, null when nothing is deployed. Guarded by
        /// processing: only the claiming worker touches it.
        std::unique_ptr<Task> instance;
        /// @brief true while a worker is processing an operation for this task.
        std::atomic<bool> processing{false};
        /// @brief the operation the claiming worker runs, null when idle. Guarded by
        /// Manager::mu.
        std::shared_ptr<Operation> op;

        [[nodiscard]] bool relevant(const synnax::rack::Key rack) {
            std::lock_guard lock{this->mu};
            return this->row.rack == rack || this->instance != nullptr;
        }
    };

    /// @brief maps task keys to their state. Uses shared_ptr for stable references.
    std::unordered_map<synnax::task::Key, std::shared_ptr<Entry>> entries;
    /// @brief pending operations to be processed by workers.
    std::list<Op> op_queue;
    /// @brief notified when ops are queued or workers should wake.
    std::condition_variable cv;
    /// @brief a worker thread and its completion flag.
    struct Worker {
        std::thread thread;
        std::shared_ptr<std::atomic<bool>> done;
    };
    /// @brief worker threads that execute operations.
    std::vector<Worker> workers;
    /// @brief thread that checks the step limits of running operations.
    std::thread monitor_thread;
    /// @brief controls worker and monitor thread lifecycle.
    x::breaker::Breaker breaker{x::breaker::Config{.name = "task.manager"}};

    /// @brief protects entries, op_queue, and streamer.
    std::mutex mu;
    /// @brief receives task set/delete/cmd events from the cluster.
    std::unique_ptr<synnax::framer::Streamer> streamer;
    /// @brief signals early shutdown before streamer is opened.
    std::atomic<bool> exit_early{false};

    /// @brief channels used to receive task events.
    struct {
        synnax::channel::Channel task_set;
        synnax::channel::Channel task_delete;
        synnax::channel::Channel task_cmd;
        synnax::channel::Channel control_state;
        synnax::channel::Channel rack_set;
    } channels;

    /// @brief opens the streamer for task set/delete/cmd channels.
    x::errors::Error open_streamer();
    /// @brief replaces the control state mirror with the cluster's current state.
    /// Best-effort: a failed read is logged and leaves the mirror empty.
    void retrieve_initial_control_states();
    /// @brief loads and queues all existing tasks from the cluster.
    x::errors::Error configure_initial_tasks();
    /// @brief stops all running tasks.
    void stop_all_tasks();
    /// @brief re-reads this rack's name so a rename made before the streamer opened
    /// is not lost. Best-effort: a failed read is logged and keeps the boot name.
    void retrieve_rack_name();
    void process_rack_set(const x::telem::Series &series);
    /// @brief refreshes the local row for each changed task. Never deploys or stops;
    /// a rack move is acted on by the next start command.
    void process_task_set(const x::telem::Series &series);
    /// @brief handles task deletion events.
    void process_task_delete(const x::telem::Series &series);
    /// @brief handles task command events.
    void process_task_cmd(const x::telem::Series &series);
    /// @brief runs a start command, redeploying first when the config has changed.
    void process_start(const synnax::task::Command &cmd);
    /// @brief returns the entry for key, creating it if absent. Callers must hold mu.
    std::shared_ptr<Entry> entry_for(const synnax::task::Key &key);
    /// @brief drops the entry for key and forgets its status, unless the entry is still
    /// relevant or claimed by a worker. even_if_processing is for a worker's own entry.
    /// Callers must hold mu.
    void remove(const synnax::task::Key &key, bool even_if_processing = false);
    /// @brief starts the worker pool and monitor thread.
    void start_workers();
    /// @brief stops workers and waits for them to finish.
    void stop_workers();
    /// @brief main loop for worker threads - pops and executes operations.
    void worker_loop();
    /// @brief warns about steps that run past their limits.
    void monitor_loop();
    /// @brief executes a single operation on an entry.
    void
    execute_op(const Op &op, const std::shared_ptr<Entry> &entry, Operation &operation);
    /// @brief clears the deployed hash so the next start rebuilds.
    static void clear_deploy(const std::shared_ptr<Entry> &entry);
    /// @brief tells the sender of a start command that the start began, so it waits
    /// for the result instead of taking the Driver as down. Does nothing for any other
    /// command.
    void announce_start(const synnax::task::Key &key, Operation &operation);
    /// @brief replaces a "Starting task" status that a previous Driver run left
    /// stored with a failed start.
    void fail_interrupted_start(const synnax::task::Task &tsk);
    /// @brief answers a pinned start whose config changed after the sender saw it.
    void refuse_pinned_start(const synnax::task::Task &tsk, Operation &operation);
    /// @brief answers a command for a task with no live instance with an error, so its
    /// sender does not wait out a timeout.
    void refuse_command(const synnax::task::Key &key, Operation &operation);
};
}
