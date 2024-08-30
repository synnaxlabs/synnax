
// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <thread>
#include <utility>
#include "glog/logging.h"
#include "nlohmann/json.hpp"
#include "client/cpp/synnax.h"
#include "driver/config/config.h"
#include "driver/breaker/breaker.h"

using json = nlohmann::json;

namespace task {
/// @brief A command that can be executed on a task in order to change its state.
struct Command {
    /// @brief the key of the task to be commanded.
    TaskKey task = 0;
    /// @brief the type of the command to execute.
    std::string type = "";
    /// @brief an optional key to assign to the command. This is useful for tracking
    /// state updates related to the command.
    std::string key = "";
    /// @brief json arguments to the command.
    json args = {};

    Command() = default;

    /// @brief constructs the command from the provided configuration parser.
    explicit Command(
        config::Parser parser
    ) : task(parser.required<TaskKey>("task")),
        type(parser.required<std::string>("type")),
        key(parser.optional<std::string>("key", "")),
        args(parser.optional<json>("args", json{})) {
    }

    /// @brief Construct a new Task Command object
    Command(const TaskKey task, std::string type, json args)
        : task(task), type(std::move(type)), args(std::move(args)) {
    }
};

/// @brief interface for a task that can be executed by the driver. Tasks should be
/// constructed by an @see Factory.
class Task {
public:
    synnax::TaskKey key = 0;

    virtual std::string name() { return ""; }

    /// @brief executes the command on the task. The task is responsible for updating
    /// its state.
    virtual void exec(Command &cmd) {
    }

    /// @brief stops the task, halting activities and freeing all resources. stop
    /// is called when the task is no longer needed, and is typically followed by a
    /// a call to the destructor.
    virtual void stop() = 0;

    virtual ~Task() = default;
};

const std::string TASK_FAILED = "error";

/// @brief struct that represents the network portable state of a task. Used both
/// internally by a task and externally by the driver to track its state.
struct State {
    /// @brief the key of the task.
    TaskKey task = 0;
    /// @brief an optional key to assign to the state update. This is particularly
    /// useful for identifying responses to commands.
    std::string key = "";
    /// @brief the type of the task.
    std::string variant = "";
    /// @brief relevant details about the current state of the task.
    json details = {};

    [[nodiscard]] json toJSON() const {
        json j;
        j["task"] = task;
        j["key"] = key;
        j["variant"] = variant;
        j["details"] = details;
        return j;
    }
};

/// @brief name of the channel used in Synnax to communicate state updates.
const std::string TASK_STATE_CHANNEL = "sy_task_state";

/// @brief an interface for a standard context that is provided to every task in the
/// driver. This context provides access to the Synnax client and alllows tasks to
/// easily update their state.
class Context {
public:
    /// @brief the client used to communicate with the Synnax server.
    std::shared_ptr<Synnax> client;

    Context() = default;

    virtual ~Context() = default;

    explicit Context(std::shared_ptr<Synnax> client) : client(std::move(client)) {
    }

    /// @brief updates the state of the task in the Synnax cluster.
    virtual void setState(const State &state) = 0;
};

/// @brief a mock context that can be used for testing tasks.
class MockContext final : public Context {
public:
    std::vector<State> states{};

    explicit MockContext(std::shared_ptr<Synnax> client) : Context(client) {
    }


    void setState(const State &state) override {
        state_mutex.lock();
        states.push_back(state);
        state_mutex.unlock();
    }

private:
    std::mutex state_mutex;
};

class SynnaxContext final : public Context {
public:
    explicit SynnaxContext(std::shared_ptr<Synnax> client) : Context(client) {
    }

    void setState(const State &state) override {
        std::unique_lock lock(state_mutex);
        if (state_updater == nullptr) {
            auto [ch, err] = client->channels.retrieve(TASK_STATE_CHANNEL);
            if (err) {
                LOG(ERROR) <<
                           "[task.context] failed to retrieve channel to update task state"
                           << err.
                               message();
                return;
            }
            chan = ch;
            auto [su, su_err] = client->telem.openWriter(WriterConfig{
                .channels = {ch.key}
            });
            if (err) {
                LOG(ERROR) <<
                           "[task.context] failed to open writer to update task state"
                           <<
                           su_err.
                               message();
                return;
            }
            state_updater = std::make_unique<Writer>(std::move(su));
        }
        auto s = Series(to_string(state.toJSON()), JSON);
        auto fr = Frame(chan.key, std::move(s));
        if (state_updater->write(fr)) return;
        auto err = state_updater->close();
        LOG(ERROR) << "[task.context] failed to write task state update" << err;
        state_updater = nullptr;
    }

private:
    std::mutex state_mutex;
    std::unique_ptr<Writer> state_updater;
    Channel chan;
};

class Factory {
public:
    virtual std::vector<std::pair<synnax::Task, std::unique_ptr<Task> > >
    configureInitialTasks(
        const std::shared_ptr<Context> &ctx,
        const synnax::Rack &rack
    ) { return {}; }

    virtual std::pair<std::unique_ptr<Task>, bool> configureTask(
        const std::shared_ptr<Context> &ctx,
        const synnax::Task &task
    ) = 0;

    virtual ~Factory() = default;
};

class MultiFactory final : public Factory {
public:
    explicit MultiFactory(std::vector<std::shared_ptr<Factory> > &&factories)
        : factories(std::move(factories)) {
    }

    std::vector<std::pair<synnax::Task, std::unique_ptr<Task> > >
    configureInitialTasks(
        const std::shared_ptr<Context> &ctx,
        const synnax::Rack &rack
    ) override {
        std::vector<std::pair<synnax::Task, std::unique_ptr<Task> > > tasks;
        for (const auto &factory: factories) {
            auto new_tasks = factory->configureInitialTasks(ctx, rack);
            for (auto &task: new_tasks) tasks.emplace_back(std::move(task));
        }
        return tasks;
    }

    std::pair<std::unique_ptr<Task>, bool>
    configureTask(
        const std::shared_ptr<Context> &ctx,
        const synnax::Task &task
    ) override {
        for (const auto &factory: factories) {
            auto [t, ok] = factory->configureTask(ctx, task);
            if (ok) return {std::move(t), true};
        }
        return {nullptr, false};
    }

private:
    std::vector<std::shared_ptr<Factory> > factories;
};

/// @brief TaskManager is responsible for configuring, executing, and commanding data
/// acqusition and control tasks.
class Manager {
public:
    Manager(
        const Rack &rack,
        const std::shared_ptr<Synnax> &client,
        std::unique_ptr<task::Factory> factory,
        const breaker::Config &breaker
    );

    ~Manager();

    freighter::Error start(std::atomic<bool> &done);

    freighter::Error stop();

private:
    RackKey rack_key;
    Rack internal;
    std::shared_ptr<task::Context> ctx;
    std::unique_ptr<Streamer> streamer;
    std::unique_ptr<task::Factory> factory;
    std::unordered_map<std::uint64_t, std::unique_ptr<task::Task> > tasks{};

    Channel task_set_channel;
    Channel task_delete_channel;
    Channel task_cmd_channel;
    Channel task_state_channel;

    breaker::Breaker breaker;

    std::thread run_thread;
    freighter::Error run_err;

    void run(std::atomic<bool> &done);

    freighter::Error runGuarded();

    freighter::Error startGuarded();

    void processTaskSet(const Series &series);

    void processTaskDelete(const Series &series);

    void processTaskCmd(const Series &series);
};
}
