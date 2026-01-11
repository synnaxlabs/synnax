// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std.
#include <string>

#include "nlohmann/json.hpp"

extern "C" {
#include <lauxlib.h>
#include <lua.h>
}

#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"

/// internal.
#include "driver/sequence/plugins/plugins.h"
#include "driver/task/task.h"

using json = x::json::json;

namespace driver::sequence {
/// @brief integration name for use in driver configuration.
const std::string INTEGRATION_NAME = "sequence";
/// @brief task type for use in driver configuration.
const std::string TASK_TYPE = INTEGRATION_NAME;
/// @brief base error for all sequencing problems.
const x::errors::Error BASE_ERROR = x::errors::SY.sub("sequence");
/// @brief returned when a sequence fails to compile.
const x::errors::Error COMPILATION_ERROR = BASE_ERROR.sub("compilation");
/// @brief returned when the sequence encounters a runtime error.
const x::errors::Error RUNTIME_ERROR = BASE_ERROR.sub("runtime");

/// @brief TaskConfig is the configuration for creating a sequence task.
struct TaskConfig {
    /// @brief rate is the rate at which the script loop will execute.
    x::telem::Rate rate;
    /// @brief script is the lua scrip that will be executed ihn the fixed rate
    /// loop.
    std::string script;
    /// @brief read is the list of channels that the task will need to read from in
    /// real-time.
    std::vector<synnax::channel::Key> read;
    /// @brief write_to is the channels that the task will need write access to for
    /// control.
    std::vector<synnax::channel::Key> write;
    /// @brief globals is a JSON object whose keys are global variables that will be
    /// available within the Lua script.
    json globals;
    /// @brief authority is the base authority level that the sequence will have;
    x::telem::Authority authority;

    explicit TaskConfig(x::json::Parser &parser):
        // this comment keeps the formatter happy
        rate(x::telem::Rate(parser.field<float>("rate"))),
        script(parser.field<std::string>("script")),
        read(parser.field<std::vector<synnax::channel::Key>>("read")),
        write(parser.field<std::vector<synnax::channel::Key>>("write")),
        globals(parser.field<json>("globals", json::object())),
        authority(parser.field<x::telem::Authority>("authority", 150)) {}
};

/// @brief deleted used to clean up lua unique pointers to ensure resources are
/// free.
struct LuaStateDeleter {
    void operator()(lua_State *L) const {
        if (L) lua_close(L);
    }
};

class Sequence {
public:
    Sequence(const std::shared_ptr<plugins::Plugin> &plugins, std::string script);

    ~Sequence();

    /// @brief compiles the script in the sequence. It is not strictly necessary to
    /// run this before calling start(), although it can be used to check for
    /// compilation errors early.
    [[nodiscard]] x::errors::Error compile();

    /// @brief starts the sequence, initializing all plugins. Note that this
    /// function does not actually run the sequence, but prepares it for execution.
    [[nodiscard]] x::errors::Error begin();

    /// @brief executes the next iteration in the sequence.
    [[nodiscard]] x::errors::Error next() const;

    /// @brief ends the sequence, cleaning up any resources that were allocated.
    [[nodiscard]] x::errors::Error end() const;

private:
    /// @brief source is used to bind relevant variables to the lua state.
    std::shared_ptr<plugins::Plugin> plugins;
    /// @brief L is the lua program state.
    std::unique_ptr<lua_State, LuaStateDeleter> L;
    /// @brief script is the raw lua script.
    std::string script;
    /// @brief script_ref is the reference to the cache, compiled lua script.
    int script_ref = LUA_NOREF;
};

/// @brief an implementation of a driver task used for configuring and running
/// automated sequences.
class Task final : public driver::task::Task {
    /// @brief cfg is the configuration for the task.
    const TaskConfig cfg;
    /// @brief task is the synnax task configuration.
    const synnax::task::Task task;
    /// @brief the list of channels that the task will write to.
    x::breaker::Breaker breaker;
    /// @brief thread is the thread that will execute the sequence.
    std::thread thread;
    /// @brief ctx is the task execution context for communicating with the Synnax
    /// cluster and updating the task state.
    std::shared_ptr<driver::task::Context> ctx;
    /// @brief the compiled sequence that will be executed within the task.
    std::unique_ptr<driver::sequence::Sequence> seq;
    /// @brief the current task state.
    synnax::task::Status status;

public:
    /// @brief static helper function used to configure the sequence.
    /// @returns the configured sequence if configuration was successful, otherwise
    /// returns a nullptr. Configuration errors are communicated through the task
    /// context.
    static std::unique_ptr<driver::task::Task>
    configure(const std::shared_ptr<driver::task::Context> &ctx, const synnax::task::Task &task);

    Task(
        const std::shared_ptr<driver::task::Context> &ctx,
        synnax::task::Task task,
        TaskConfig cfg,
        std::unique_ptr<driver::sequence::Sequence> seq,
        const x::breaker::Config &breaker_config
    );

    /// @brief returns the name of the task for logging.
    std::string name() const override { return this->task.name; }

    /// @brief main run loop that will execute in a separate thread.
    void run();

    /// @brief stops the task, implementing driver::task::Task.
    void stop(bool will_reconfigure) override;

    /// @brief stops the task, using the provided key as the key of the command that
    /// was executed.
    void stop(const std::string &key, bool will_reconfigure);

    /// @brief executes a command on the task, implementing driver::task::Task.
    void exec(synnax::task::Command &cmd) override;

    /// @brief starts the task, using the provided key as the key of the command
    /// that was executed.
    void start(const std::string &key);
};

/// @brief factory used to configure control sequences from within the driver's task
/// manager.
class Factory final : public driver::task::Factory {
public:
    Factory() = default;

    std::pair<std::unique_ptr<driver::task::Task>, bool> configure_task(
        const std::shared_ptr<driver::task::Context> &ctx,
        const synnax::task::Task &task
    ) override {
        if (task.type != TASK_TYPE) return {nullptr, false};
        return {driver::sequence::Task::configure(ctx, task), true};
    }

    std::string name() override { return INTEGRATION_NAME; }
};
}
