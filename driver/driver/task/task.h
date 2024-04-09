#pragma once

#include <memory>
#include "glog/logging.h"
#include "nlohmann/json.hpp"
#include "client/cpp/synnax/synnax.h"
#include "driver/driver/config/config.h"

using json = nlohmann::json;

namespace task {
/// @brief A command that can be executed on a task in order to change its state.
struct Command {
    /// @brief the key of the task to be commanded.
    TaskKey task = 0;
    /// @brief the type of the command to execute.
    std::string type = "";
    /// @brief json arguments to the command.
    json args = {};

    Command() = default;

    /// @brief constructs the command from the provided configuration parser.
    explicit Command(
        config::Parser parser
    ): task(parser.required<TaskKey>("task")),
       type(parser.required<std::string>("type")),
       args(parser.required<json>("args")) {
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

    /// @brief executes the command on the task. The task is responsible for updating
    /// its state.
    virtual void exec(Command &cmd) = 0;

    /// @brief stops the task, halting activities and freeing all resources. stop
    /// is called when the task is no longer needed, and is typically followed by a
    /// a call to the destructor.
    virtual void stop() = 0;

    virtual ~Task() = default;
};

const std::string TASK_FAILED = "failed";

/// @brief struct that represents the network portable state of a task. Used both
/// internally by a task and externally by the driver to track its state.
struct State {
    /// @brief the key of the task.
    TaskKey task = 0;
    /// @brief the type of the task.
    std::string type = "";
    /// @brief relevant details about the current state of the task.
    json details = {};

    json toJSON() {
        json j;
        j["task"] = task;
        j["type"] = type;
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

    explicit Context(std::shared_ptr<Synnax> client): client(client) {
    }

    /// @brief updates the state of the task in the Synnax cluster.
    virtual void setState(State state) = 0;
};

/// @brief a mock context that can be used for testing tasks.
class MockContext final : public Context {
public:
    std::vector<State> states{};

    explicit MockContext(std::shared_ptr<Synnax> client): Context(client) {
    }


    void setState(State state) override {
        state_mutex.lock();
        states.push_back(state);
        state_mutex.unlock();
    }

private:
    std::mutex state_mutex;
};

class SynnaxContext final : public Context {
public:
    explicit SynnaxContext(std::shared_ptr<Synnax> client): Context(client) {
    }

    void setState(State state) override {
        state_mutex.lock();
        if (state_updater == nullptr) {
            auto [task_state_ch, err] = client->channels.retrieve(TASK_STATE_CHANNEL);
            if (err) {
                LOG(ERROR) << "Failed to retrieve channel to update task state" << err.
                        message();
                state_mutex.unlock();
                return;
            }
            task_state_channel = task_state_ch;
            auto [su, su_err] = client->telem.openWriter(WriterConfig{
                .channels = {task_state_ch.key}
            });
            if (err) {
                LOG(ERROR) << "Failed to open writer to update task state" << su_err.
                        message();
                state_mutex.unlock();
                return;
            }
            state_updater = std::make_unique<Writer>(std::move(su));
        }
        auto fr = Frame(1);
        fr.add(task_state_channel.key,
               Series(std::vector{to_string(state.toJSON())}, JSON));
        if (!state_updater->write(std::move(fr))) {
            auto err = state_updater->close();
            LOG(ERROR) << "Failed to write task state update" << err.message();
            state_updater = nullptr;
        }
        state_mutex.unlock();
    }

private:
    std::mutex state_mutex;
    std::unique_ptr<Writer> state_updater;
    Channel task_state_channel;
};

class Factory {
public:
    virtual std::vector<std::pair<synnax::Task, std::unique_ptr<Task>> > configureInitialTasks(
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

    std::vector<std::pair<synnax::Task, std::unique_ptr<Task>> > configureInitialTasks(
        const std::shared_ptr<Context> &ctx,
        const synnax::Rack &rack
    ) override {
        std::vector<std::pair<synnax::Task, std::unique_ptr<Task>> > tasks;
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
}
