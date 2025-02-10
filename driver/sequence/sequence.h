// Copyright 2025 Synnax Labs, Inc.
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

/// external.
#include "nlohmann/json.hpp"

extern "C" {
#include <lua.h>
#include <lauxlib.h>
}

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/xerrors/errors.h"

/// internal.
#include "driver/sequence/plugins/plugins.h"

using json = nlohmann::json;

namespace sequence {
/// @brief integration name for use in driver configuration.
const std::string INTEGRATION_NAME = "sequence";
/// @brief base error for all sequencing problems.
const xerrors::Error BASE_ERROR = xerrors::BASE_ERROR.sub("sequence");
/// @brief returned when a sequence fails to compile.
const xerrors::Error COMPILATION_ERROR = BASE_ERROR.sub("compilation");
/// @brief returned when the sequence encounters a runtime error.
const xerrors::Error RUNTIME_ERROR = BASE_ERROR.sub("runtime");

/// @brief TaskConfig is the configuration for creating a sequence task.
struct TaskConfig {
    /// @brief rate is the rate at which the script loop will execute.
    telem::Rate rate;
    /// @brief script is the lua scrip that will be executed ihn the fixed rate loop.
    std::string script;
    /// @brief read is the list of channels that the task will need to read from in
    /// real-time.
    std::vector<synnax::ChannelKey> read;
    /// @brief write_to is the channels that the task will need write access to for
    /// control.
    std::vector<synnax::ChannelKey> write;
    /// @brief globals is a JSON object whose keys are global variables that will be
    /// available within the Lua script.
    json globals;

    explicit TaskConfig(config::Parser &parser):
        // this comment keeps the formatter happy
        rate(telem::Rate(parser.required<float>("rate"))),
        script(parser.required<std::string>("script")),
        read(parser.required_vector<synnax::ChannelKey>("read")),
        write(parser.required_vector<synnax::ChannelKey>("write")),
        globals(parser.optional<json>("globals", json::object())) {
    }
};

/// @brief deleted used to clean up lua unique pointers to ensure resources are free.
struct LuaStateDeleter {
    void operator()(lua_State *L) const { if (L) lua_close(L); }
};

class Sequence {
public:
    Sequence(
        const std::shared_ptr<plugins::Plugin> &plugins,
        std::string script
    );

    ~Sequence();

    /// @brief compiles the script in the sequence. It is not strictly necessary to run
    /// this before calling start(), although it can be used to check for compilation
    /// errors early.
    [[nodiscard]] xerrors::Error compile();

    /// @brief starts the sequence, initializing all plugins. Note that this function
    /// does not actually run the sequence, but prepares it for execution.
    [[nodiscard]] xerrors::Error start();

    /// @brief executes the next iteration in the sequence.
    [[nodiscard]] xerrors::Error next() const;

    /// @brief ends the sequence, cleaning up any resources that were allocated.
    [[nodiscard]] xerrors::Error end() const;
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
class Task final : public task::Task {
    /// @brief cfg is the configuration for the task.
    const TaskConfig cfg;
    /// @brief task is the synnax task configuration.
    const synnax::Task task;
    /// @brief the list of channels that the task will write to.
    breaker::Breaker breaker;
    /// @brief thread is the thread that will execute the sequence.
    std::thread thread;
    /// @brief ctx is the task execution context for communicating with the Synnax cluster
    /// and updating the task state.
    std::shared_ptr<task::Context> ctx;
    /// @brief the compiled sequence that will be executed within the task.
    std::unique_ptr<sequence::Sequence> seq;
    /// @brief the current task state.
    task::State state;

public:
    /// @brief static helper function used to configure the sequence.
    /// @returns the configured sequence if configuration was successful, otherwise
    /// returns a nullptr. Configuration errors are communicated through the task
    /// context.
    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    Task(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        TaskConfig cfg,
        std::unique_ptr<sequence::Sequence> seq,
        const breaker::Config &breaker_config
    );

    /// @brief main run loop that will execute in a separate thread.
    void run();

    /// @brief stops the task, implementing task::Task.
    void stop() override;

    /// @brief stops the task, using the provided key as the key of the command that
    /// was executed.
    void stop(const std::string &key);

    /// @brief executes a command on the task, implementing task::Task.
    void exec(task::Command &cmd) override;

    /// @brief starts the task, using the provided key as the key of the command that
    /// was executed.
    void start(const std::string &key);
};

/// @brief factory used to configure control sequences from within the driver's task
/// manager.
class Factory final : public task::Factory {
public:
    Factory() = default;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        if (task.type != "sequence") return {nullptr, false};
        return {sequence::Task::configure(ctx, task), true};
    }
};
}
