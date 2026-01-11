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
#include <future>
#include <list>
#include <memory>
#include <mutex>
#include <thread>
#include <utility>
#include <vector>

#include "glog/logging.h"
#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/json/json.h"
#include "x/cpp/log/log.h"

using json = nlohmann::json;

namespace task {
/// @brief A command that can be executed on a task in order to change its state.
struct Command {
    /// @brief the key of the task to be commanded.
    synnax::task::Key task = 0;
    /// @brief the type of the command to execute.
    std::string type;
    /// @brief an optional key to assign to the command. This is useful for tracking
    /// state updates related to the command.
    std::string key;
    /// @brief json arguments to the command.
    json args = {};

    Command() = default;

    /// @brief constructs the command from the provided configuration parser.
    explicit Command(x::json::Parser parser):
        task(parser.field<synnax::task::Key>("task")),
        type(parser.field<std::string>("type")),
        key(parser.field<std::string>("key", "")),
        args(parser.field<json>("args", json{})) {}

    /// @brief Construct a new Task Command object
    Command(const synnax::task::Key task, std::string type, json args):
        task(task), type(std::move(type)), args(std::move(args)) {}

    [[nodiscard]] json to_json() const {
        return {{"task", task}, {"type", type}, {"key", key}, {"args", args}};
    }

    friend std::ostream &operator<<(std::ostream &os, const Command &cmd) {
        os << cmd.type << " (key=" << cmd.key << ",task=" << cmd.task << ")";
        return os;
    }
};

/// @brief interface for a task that can be executed by the driver. Tasks should be
/// constructed by an @see Factory.
class Task {
public:
    /// @brief the key of the task
    synnax::task::Key key = 0;

    [[nodiscard]] virtual std::string name() const { return ""; }

    /// @brief executes the command on the task. The task is responsible for
    /// updating its state.
    virtual void exec(Command &cmd) {}

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
public:
    /// @brief the client used to communicate with the Synnax server.
    std::shared_ptr<synnax::Synnax> client;

    Context() = default;

    virtual ~Context() = default;

    explicit Context(std::shared_ptr<synnax::Synnax> client):
        client(std::move(client)) {}

    /// @brief updates the state of the task in the Synnax cluster.
    virtual void set_status(synnax::task::Status &status) = 0;
};

/// @brief a mock context that can be used for testing tasks.
class MockContext final : public Context {
    std::mutex mu;

public:
    std::vector<synnax::task::Status> statuses{};

    explicit MockContext(const std::shared_ptr<synnax::Synnax> &client):
        Context(client) {}

    void set_status(synnax::task::Status &status) override {
        mu.lock();
        statuses.push_back(status);
        mu.unlock();
    }
};

class SynnaxContext final : public Context {
public:
    explicit SynnaxContext(const std::shared_ptr<synnax::Synnax> &client):
        Context(client) {}

    void set_status(synnax::task::Status &status) override {
        if (status.time == 0) status.time = x::telem::TimeStamp::now();
        if (const auto err = this->client->statuses.set<synnax::task::StatusDetails>(
                status
            );
            err)
            LOG(ERROR) << "[task.context] failed to write task status update: " << err;
    }
};

class Factory {
public:
    virtual std::vector<std::pair<synnax::task::Task, std::unique_ptr<Task>>>
    configure_initial_tasks(
        const std::shared_ptr<Context> &ctx,
        const synnax::rack::Rack &rack
    ) {
        return {};
    }

    virtual std::string name() { return ""; }

    virtual std::pair<std::unique_ptr<Task>, bool>
    configure_task(const std::shared_ptr<Context> &ctx, const synnax::task::Task &task) = 0;

    virtual ~Factory() = default;
};

class MultiFactory final : public Factory {
    std::vector<std::unique_ptr<Factory>> factories;

public:
    explicit MultiFactory(std::vector<std::unique_ptr<Factory>> &&factories):
        factories(std::move(factories)) {}

    std::vector<std::pair<synnax::task::Task, std::unique_ptr<Task>>> configure_initial_tasks(
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
        const synnax::task::Task &task
    ) override {
        for (const auto &factory: factories) {
            auto [t, ok] = factory->configure_task(ctx, task);
            if (ok) return {std::move(t), true};
        }
        return {nullptr, false};
    }
};

/// @brief configuration for the task manager.
struct ManagerConfig {
    /// @brief duration before reporting stuck operations.
    x::telem::TimeSpan op_timeout = 60 * x::telem::SECOND;
    /// @brief interval between timeout checks.
    x::telem::TimeSpan poll_interval = 1 * x::telem::SECOND;
    /// @brief max time to wait for workers during shutdown before detaching.
    x::telem::TimeSpan shutdown_timeout = 30 * x::telem::SECOND;
    /// @brief number of worker threads for task operations.
    size_t worker_count = 4;

    template<typename Parser>
    void override(Parser &p) {
        const auto op_timeout_s = p.field(
            "op_timeout",
            static_cast<double>(this->op_timeout.seconds())
        );
        this->op_timeout = x::telem::TimeSpan(static_cast<int64_t>(op_timeout_s * 1e9));
        const auto poll_interval_s = p.field(
            "poll_interval",
            static_cast<double>(this->poll_interval.seconds())
        );
        this->poll_interval = x::telem::TimeSpan(
            static_cast<int64_t>(poll_interval_s * 1e9)
        );
        const auto shutdown_timeout_s = p.field(
            "shutdown_timeout",
            static_cast<double>(this->shutdown_timeout.seconds())
        );
        this->shutdown_timeout = x::telem::TimeSpan(
            static_cast<int64_t>(shutdown_timeout_s * 1e9)
        );
        this->worker_count = p.field(
            "worker_count",
            static_cast<int>(this->worker_count)
        );
        if (this->worker_count < 1) this->worker_count = 1;
        if (this->worker_count > 64) this->worker_count = 64;
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
        std::unique_ptr<task::Factory> factory,
        const ManagerConfig &cfg = {}
    ):
        rack(std::move(rack)),
        ctx(std::make_shared<SynnaxContext>(client)),
        factory(std::move(factory)),
        op_timeout(cfg.op_timeout),
        poll_interval(cfg.poll_interval),
        shutdown_timeout(cfg.shutdown_timeout),
        worker_count(cfg.worker_count) {}

    /// @brief runs the main task manager loop, blocking until stop() is called.
    /// Safe to call stop() from another thread.
    x::errors::Error run(std::function<void()> on_started = nullptr);

    /// @brief stops the task manager, halting all tasks and freeing resources.
    void stop();

private:
    /// @brief the rack this manager belongs to.
    synnax::rack::Rack rack;
    /// @brief shared context passed to all tasks.
    std::shared_ptr<Context> ctx;
    /// @brief creates device-specific tasks.
    std::unique_ptr<Factory> factory;
    /// @brief duration before reporting stuck operations.
    x::telem::TimeSpan op_timeout;
    /// @brief interval between timeout checks.
    x::telem::TimeSpan poll_interval;
    /// @brief max time to wait for workers during shutdown before detaching.
    x::telem::TimeSpan shutdown_timeout;
    /// @brief number of worker threads for task operations.
    size_t worker_count;

    /// @brief an operation to be executed by a worker.
    struct Op {
        /// @brief types of operations that can be queued.
        enum class Type { CONFIGURE, COMMAND, SHUTDOWN, REMOVE };
        Type type;
        synnax::task::Key task_key;
        synnax::task::Task task;
        Command cmd;
    };

    /// @brief per-task state tracked by the manager.
    struct Entry {
        std::unique_ptr<Task> task;
        /// @brief true while a worker is processing an operation for this task.
        std::atomic<bool> processing{false};
        /// @brief when the current operation started (0 if idle).
        std::atomic<x::telem::TimeStamp> op_started{x::telem::TimeStamp(0)};
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
    /// @brief thread that checks for stuck operations.
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
    } channels;

    /// @brief returns true if the task belongs to a different rack.
    [[nodiscard]] bool skip_foreign_rack(const synnax::task::Key &task_key) const;
    /// @brief opens the streamer for task set/delete/cmd channels.
    x::errors::Error open_streamer();
    /// @brief loads and queues all existing tasks from the cluster.
    x::errors::Error configure_initial_tasks();
    /// @brief stops all running tasks.
    void stop_all_tasks();
    /// @brief handles task create/update events.
    void process_task_set(const x::telem::Series &series);
    /// @brief handles task deletion events.
    void process_task_delete(const x::telem::Series &series);
    /// @brief handles task command events.
    void process_task_cmd(const x::telem::Series &series);
    /// @brief starts the worker pool and monitor thread.
    void start_workers();
    /// @brief stops workers and waits for them to finish.
    void stop_workers();
    /// @brief main loop for worker threads - pops and executes operations.
    void worker_loop();
    /// @brief checks for operations that have exceeded op_timeout.
    void monitor_loop();
    /// @brief executes a single operation on an entry.
    void execute_op(const Op &op, const std::shared_ptr<Entry> &entry) const;
};
}
