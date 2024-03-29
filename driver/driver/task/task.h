//
// Created by Emiliano Bonilla on 3/27/24.
//

#pragma once

#include <memory>
#include <sys/stat.h>

#include "client/cpp/synnax/synnax.h"
#include "glog/logging.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

namespace task {
/// @brief A command that can be executed on a task in order to change its state.
struct Command {
    /// @brief the key of the task to be commanded.
    TaskKey task;
    /// @brief the type of the command to execute.
    std::string type;
    /// @brief json arguments to the command.
    json args;

    Command(): task(0), args(json::object()) {
    }

    /// @brief Construct a new Task Command object
    Command(TaskKey task, std::string type, json args)
        : task(task), type(std::move(type)), args(std::move(args)) {
    }

    /// @brief parses the task command from JSON.
    static std::pair<Command, freighter::Error> fromJSON(const json& cmd) {
        auto task_iter = cmd.find("task");
        if (task_iter == cmd.end()) {
            return {Command{}, freighter::Error{"task key not found"}};
        }
        auto type_iter = cmd.find("type");
        if (type_iter == cmd.end()) {
            return {Command{}, freighter::Error{"type not found"}};
        }
        auto args_iter = cmd.find("args");
        json args = json::object();
        if (args_iter != cmd.end()) args = args_iter.value();
        return {
            Command(
                task_iter.value().get<TaskKey>(),
                type_iter.value().get<std::string>(),
                args
            ),
            {},
        };
    }
};

class Task {
public:
    virtual void exec(Command& cmd) = 0;

    virtual void stop() = 0;

    virtual ~Task() = default;
};

const std::string TASK_FAILED = "failed";

struct State {
    TaskKey task;
    std::string type;
    json details;

    json toJSON() {
        json j;
        j["task"] = task;
        j["type"] = type;
        j["details"] = details;
        return j;
    }
};

const std::string TASK_STATE_CHANNEL = "sy_task_state";

class Context {
public:
    std::shared_ptr<Synnax> client;

    Context() = default;

    virtual ~Context() = default;

    explicit Context(std::shared_ptr<Synnax> client): client(std::move(client)) {
    }

    virtual void setState(State state) = 0;
};

class MockContext final : public Context {
private:
    std::mutex state_mutex;

public:
    std::vector<State> states;

    explicit MockContext(std::shared_ptr<Synnax> client): Context(std::move(client)) {
    }


    void setState(State state) override {
        state_mutex.lock();
        states.push_back(state);
        state_mutex.unlock();
    }
};

class ContextImpl final : public Context {
public:
    std::shared_ptr<Synnax> client;

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
    virtual std::pair<std::unique_ptr<Task>, bool> configureTask(
        const std::shared_ptr<Context>& ctx,
        const synnax::Task& task
    ) = 0;

    virtual ~Factory() = default;
};

class MultiFactory final : public Factory {
public:
    MultiFactory(std::vector<std::shared_ptr<Factory>>&& factories)
        : factories(std::move(factories)) {
    }

    std::pair<std::unique_ptr<Task>, bool>
    configureTask(const std::shared_ptr<Context>& ctx,
                  const synnax::Task& task) override {
        for (const auto& factory: factories) {
            auto [t, ok] = factory->configureTask(ctx, task);
            if (ok) return {std::move(t), true};
        }
        return {nullptr, false};
    }

private:
    std::vector<std::shared_ptr<Factory>> factories;
};
}
