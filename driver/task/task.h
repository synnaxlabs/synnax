// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <future>
#include <memory>
#include <mutex>
#include <thread>
#include <utility>

#include "glog/logging.h"
#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xjson/xjson.h"

using json = nlohmann::json;

namespace task {
/// @brief A command that can be executed on a task in order to change its state.
struct Command {
    /// @brief the key of the task to be commanded.
    synnax::TaskKey task = 0;
    /// @brief the type of the command to execute.
    std::string type;
    /// @brief an optional key to assign to the command. This is useful for tracking
    /// state updates related to the command.
    std::string key;
    /// @brief json arguments to the command.
    json args = {};

    Command() = default;

    /// @brief constructs the command from the provided configuration parser.
    explicit Command(xjson::Parser parser):
        task(parser.field<synnax::TaskKey>("task")),
        type(parser.field<std::string>("type")),
        key(parser.field<std::string>("key", "")),
        args(parser.field<json>("args", json{})) {}

    /// @brief Construct a new Task Command object
    Command(const synnax::TaskKey task, std::string type, json args):
        task(task), type(std::move(type)), args(std::move(args)) {}

    [[nodiscard]] json to_json() const {
        return {{"task", task}, {"type", type}, {"key", key}, {"args", args}};
    }
};

/// @brief interface for a task that can be executed by the driver. Tasks should be
/// constructed by an @see Factory.
class Task {
public:
    /// @brief the key of the task
    synnax::TaskKey key = 0;

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
    virtual void set_status(synnax::TaskStatus &status) = 0;
};

/// @brief a mock context that can be used for testing tasks.
class MockContext final : public Context {
    std::mutex mu;

public:
    std::vector<synnax::TaskStatus> statuses{};

    explicit MockContext(const std::shared_ptr<synnax::Synnax> &client):
        Context(client) {}

    void set_status(synnax::TaskStatus &status) override {
        mu.lock();
        statuses.push_back(status);
        mu.unlock();
    }
};

class SynnaxContext final : public Context {
public:
    explicit SynnaxContext(const std::shared_ptr<synnax::Synnax> &client):
        Context(client) {}

    void set_status(synnax::TaskStatus &status) override {
        if (status.time == 0) status.time = telem::TimeStamp::now();
        if (const auto err = this->client->statuses.set<synnax::TaskStatusDetails>(
                status
            );
            err)
            LOG(ERROR) << "[task.context] failed to write task state update" << err;
    }
};

class Factory {
public:
    virtual std::vector<std::pair<synnax::Task, std::unique_ptr<Task>>>
    configure_initial_tasks(
        const std::shared_ptr<Context> &ctx,
        const synnax::Rack &rack
    ) {
        return {};
    }

    virtual std::string name() { return ""; }

    virtual std::pair<std::unique_ptr<Task>, bool>
    configure_task(const std::shared_ptr<Context> &ctx, const synnax::Task &task) = 0;

    virtual ~Factory() = default;
};

class MultiFactory final : public Factory {
    std::vector<std::unique_ptr<Factory>> factories;

public:
    explicit MultiFactory(std::vector<std::unique_ptr<Factory>> &&factories):
        factories(std::move(factories)) {}

    std::vector<std::pair<synnax::Task, std::unique_ptr<Task>>> configure_initial_tasks(
        const std::shared_ptr<Context> &ctx,
        const synnax::Rack &rack
    ) override {
        std::vector<std::pair<synnax::Task, std::unique_ptr<Task>>> tasks;
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
        const synnax::Task &task
    ) override {
        for (const auto &factory: factories) {
            auto [t, ok] = factory->configure_task(ctx, task);
            if (ok) return {std::move(t), true};
        }
        return {nullptr, false};
    }
};

/// @brief TaskManager is responsible for configuring, executing, and commanding
/// data acquisition and control tasks.
class Manager {
public:
    Manager(
        synnax::Rack rack,
        const std::shared_ptr<synnax::Synnax> &client,
        std::unique_ptr<task::Factory> factory
    ):
        rack(std::move(rack)),
        ctx(std::make_shared<SynnaxContext>(client)),
        factory(std::move(factory)),
        channels({}) {}

    /// @brief runs the main task manager loop, booting up initial tasks retrieved
    /// from the cluster, and processing task modifications (set, delete, and
    /// command) requests through streamed channel values. Note that this function
    /// does not for a thread to run in, and blocks until stop() is called.
    ///
    /// This function NOT be called concurrently with any other calls
    /// to run(). It is safe to call run() concurrently with stop().
    ///
    /// @param on_started an optional callback that will be called when the manager
    /// has started successfully.
    xerrors::Error run(std::function<void()> on_started = nullptr);

    /// @brief stops the task manager, halting all tasks and freeing all resources.
    /// Once the manager has shut down, the run() function will return with any
    /// errors encountered during operation.
    void stop();

private:
    /// @brief the rack that this task manager belongs to.
    synnax::Rack rack;
    /// @brief a common context object passed to all tasks.
    std::shared_ptr<task::Context> ctx;
    /// @brief the factory used to create tasks.
    std::unique_ptr<task::Factory> factory;
    /// @brief a map of tasks that have been configured on the rack.
    std::unordered_map<synnax::TaskKey, std::unique_ptr<task::Task>> tasks{};

    /// @brief the streamer variable is read from in both the run() and stop()
    /// functions, so we need to lock its assignment.
    std::mutex mu;
    /// @brief receives streamed values from the Synnax server to change tasks in
    /// the manager.
    std::unique_ptr<synnax::Streamer> streamer;
    std::atomic<bool> exit_early = false;

    /// @brief information on channels we need to work with tasks.
    struct {
        synnax::Channel task_set;
        synnax::Channel task_delete;
        synnax::Channel task_cmd;
    } channels;

    [[nodiscard]] bool skip_foreign_rack(const synnax::TaskKey &task_key) const;

    /// @brief opens the streamer for the task manager, which is used to listen for
    /// incoming task set, delete, and command requests.
    xerrors::Error open_streamer();

    /// @brief retrieves and configures all initial tasks for the rack from the
    /// server.
    xerrors::Error configure_initial_tasks();

    /// @brief stops all tasks.
    void stop_all_tasks();

    /// @brief processes when a new task is created or an existing task needs to be
    /// reconfigured.
    void process_task_set(const telem::Series &series);

    /// @brief processes when a task is deleted.
    void process_task_delete(const telem::Series &series);

    /// @brief processes when a command needs to be executed on a configured task.
    void process_task_cmd(const telem::Series &series);
};
}
