// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

// std.
#include <string>
#include <unordered_map>
#include <thread>
#include <latch>

// external.
#include "nlohmann/json.hpp"
#include "jsonutil/jsonutil.h"

// internal.
#include "client/cpp/synnax/synnax.h"
#include "driver/driver/breaker/breaker.h"

using json = nlohmann::json;

namespace driver {
/// @brief A command that can be executed on a task in order to change its state.
struct TaskCommand {
    /// @brief the key of the task to be commanded.
    TaskKey task;
    /// @brief the type of the command to execute.
    std::string type;
    /// @brief json arguments to the command.
    json args;

    TaskCommand(): task(0), args(json::object()) {
    }

    /// @brief Construct a new Task Command object
    TaskCommand(TaskKey task, std::string type, json args)
        : task(task), type(std::move(type)), args(std::move(args)) {
    }

    /// @brief parses the task command from JSON.
    std::pair<TaskCommand, freighter::Error> fromJSON(const json& cmd) {
        auto task_iter = cmd.find("task");
        if (task_iter == cmd.end()) {
            return {TaskCommand{}, freighter::Error{"task key not found"}};
        }
        auto type_iter = cmd.find("type");
        if (type_iter == cmd.end()) {
            return {TaskCommand{}, freighter::Error{"type not found"}};
        }
        auto args_iter = cmd.find("args");
        json args = json::object();
        if (args_iter != cmd.end()) args = args_iter.value();
        return {
            TaskCommand(
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
    void exec(TaskCommand& cmd);

    void stop();

    virtual ~Task() = default;
};

const std::string TASK_FAILED = "failed";

struct TaskState {
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

class TaskContext {
public:
    TaskContext(const std::shared_ptr<Synnax>& client);


    void setState(TaskState state);
private:
    std::shared_ptr<Synnax> client;

    std::mutex state_mutex;
    std::unique_ptr<Writer> state_updater;
    Channel task_state_channel;

};

class TaskFactory {
public:
    virtual std::unique_ptr<Task> configureTask(
        const std::shared_ptr<TaskContext>& ctx,
        const synnax::Task& task
    ) = 0;

    virtual ~TaskFactory() = default;
};

class TaskManager {
public:
    [[maybe_unused]] TaskManager(
        RackKey rack_key,
        const std::shared_ptr<Synnax>& client,
        std::unique_ptr<TaskFactory> factory,
        breaker::Breaker breaker
    );

    freighter::Error start(std::latch& latch);

    freighter::Error stop();

private:
    RackKey rack_key;
    Rack internal;

    const std::shared_ptr<Synnax> client;
    std::unique_ptr<TaskFactory> factory;
    std::unique_ptr<Streamer> streamer;


    std::unordered_map<std::uint64_t, std::unique_ptr<Task>> tasks;

    Channel task_set_channel;
    Channel task_delete_channel;
    Channel task_cmd_channel;
    Channel task_state_channel;

    std::thread exec_thread;
    freighter::Error exit_err;
    breaker::Breaker breaker;

    void run(std::latch& latch);

    freighter::Error runInternal();

    freighter::Error startInternal();

    void processTaskSet(const Series& series, Writer& comms);

    void processTaskDelete(const Series& series, Writer& comms);

    void processTaskCmd(const Series& series, Writer& comms);
};

class Heartbeat {
public:
    Heartbeat(
        RackKey rack_key,
        std::shared_ptr<Synnax> client,
        breaker::Breaker breaker
    );

    freighter::Error start(std::latch& latch);

    freighter::Error stop();

private:
    // Synnax
    RackKey rack_key;
    const std::shared_ptr<Synnax> client;

    Channel channel;

    // Heartbeat
    std::uint32_t generation;
    std::uint32_t version;

    // Breaker
    breaker::Breaker breaker;

    // Threading
    std::atomic<bool> running;
    freighter::Error exit_err;
    std::thread exec_thread;

    void run();
};

class Driver {
public:
    Driver(
        RackKey key,
        const std::shared_ptr<Synnax>& client,
        std::unique_ptr<TaskFactory> module_factory,
        const breaker::Breaker& brk
    );

    freighter::Error run();

private:
    RackKey key;
    TaskManager task_manager;
    Heartbeat heartbeat;
};
}
