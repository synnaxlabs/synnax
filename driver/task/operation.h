// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <mutex>
#include <optional>
#include <string>
#include <utility>

#include "client/cpp/synnax.h"
#include "x/cpp/telem/telem.h"

namespace driver::task {
/// @brief one command that the task manager runs on a task. It sends the steps of the
/// command, warns when a step runs past its limit, and sends exactly one final status.
/// Safe for concurrent use.
class Operation {
public:
    /// @brief writes a status to the Core.
    using Writer = std::function<void(synnax::task::Status &)>;

    /// @param write sends each status of the operation.
    /// @param name the name of the task.
    /// @param cmd the command the operation runs. An operation with no command key
    /// answers no command.
    /// @param config_hash the config the operation runs, or empty while unknown.
    /// @param limit how long a step with no limit of its own may run.
    Operation(
        Writer write,
        std::string name,
        synnax::task::Command cmd,
        std::string config_hash,
        x::telem::TimeSpan limit
    );

    /// @brief the command the operation runs.
    const synnax::task::Command cmd;

    /// @brief records the name and config of the task, once a fetch knows them.
    void set_task(const std::string &name, const std::string &config_hash);

    /// @returns the name and config hash of the task, for a status of the operation.
    [[nodiscard]] std::pair<std::string, std::string> task_info();

    /// @brief sends a loading status that names the current step and restarts the
    /// clock of the step.
    /// @param limit how long the step may run before the operation warns.
    void step(synnax::task::Status status, x::telem::TimeSpan limit);

    /// @brief sends status with the command key. A loading status is a step that
    /// keeps the limit of the current step, and any other status is the final one. A
    /// status after the final one is sent with no command key, so it answers nothing.
    void answer(synnax::task::Status status);

    /// @brief warns once when the current step has run past its limit. The warning is
    /// a loading status, so it is not a final status. An operation with no step has no
    /// limit.
    void check(x::telem::TimeStamp now);

private:
    const Writer write;
    /// @brief orders the writes of the operation, so no step follows the final status.
    /// Taken before mu.
    std::mutex write_mu;
    /// @brief guards the fields below. Never held during a write.
    std::mutex mu;
    std::string name;
    std::string config_hash;
    /// @brief the last step sent.
    std::optional<synnax::task::Status> current;
    x::telem::TimeStamp step_started;
    x::telem::TimeSpan limit;
    bool warned = false;
    bool finished = false;

    /// @brief records status as the current step. Requires mu.
    void begin_step(const synnax::task::Status &status);
};
}
