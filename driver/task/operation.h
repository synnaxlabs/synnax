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
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/telem.h"

namespace driver::task {
/// @brief one command that the task manager runs on a task. It sends the steps of the
/// command, warns when a step runs past its limit, and sends exactly one final status
/// to the command and to each start that joined it. Safe for concurrent use.
class Operation {
public:
    /// @brief writes a status to the Core.
    using Writer = std::function<void(synnax::task::Status &)>;

    /// @param write sends each status of the operation.
    /// @param task the task the operation runs on.
    /// @param name the name of the task.
    /// @param cmd the command the operation runs. An operation with no command key
    /// answers no command.
    /// @param config_hash the config the operation runs, or empty while unknown.
    /// @param limit how long a step with no limit of its own may run.
    Operation(
        Writer write,
        synnax::task::Key task,
        std::string name,
        synnax::task::Command cmd,
        std::string config_hash,
        x::telem::TimeSpan limit
    );

    ~Operation();

    Operation(const Operation &) = delete;
    Operation &operator=(const Operation &) = delete;

    /// @brief the command the operation runs.
    const synnax::task::Command cmd;
    /// @brief the task the operation runs on.
    const synnax::task::Key task;

    /// @brief records the name and config of the task, once a fetch knows them.
    void set_task(const std::string &name, const std::string &config_hash);

    /// @returns the name and config hash of the task, for a status of the operation.
    [[nodiscard]] std::pair<std::string, std::string> task_info();

    /// @returns the token that stops when the operation is cancelled. Pass it to the
    /// configure and the exec that the operation runs.
    x::breaker::Breaker &token() { return this->breaker; }

    /// @brief cancels a start that has no final status yet. A stop with a key gets a
    /// loading status on the next check, so its sender knows the stop arrived.
    /// @param stop the stop command that cancels the start, or empty when the task
    /// manager cancels it as it stops its tasks.
    /// @returns true if the operation was cancelled.
    bool cancel(const synnax::task::Command &stop = {});

    /// @brief adds start to this operation if this operation is an open start of
    /// config_hash. The joined start gets the current step on the next check, and the
    /// later statuses of the operation.
    /// @returns true if start joined.
    bool join(const synnax::task::Command &start, const std::string &config_hash);

    /// @brief sends a loading status that names the current step and restarts the
    /// clock of the step.
    /// @param limit how long the step may run before the operation warns.
    void step(synnax::task::Status status, x::telem::TimeSpan limit);

    /// @brief sends status to the command and each joined start. A loading status is a
    /// step that keeps the limit of the current step, and any other status is the final
    /// one. A status after the final one is sent with no command key, so it answers
    /// nothing.
    void answer(synnax::task::Status status);

    /// @brief sends the current step to the starts that joined since the last status,
    /// and a loading status to a stop that cancelled the operation. Then warns once
    /// when the current step has run past its limit, with a loading status, so the
    /// warning is not final. An operation with no step only logs. Does nothing after
    /// the final status or close.
    void check(x::telem::TimeStamp now);

    /// @brief stops the statuses of check, once the worker is done with the operation.
    /// Waits for a check in progress, so no status of check follows a status that the
    /// task sends after this call. A cancelled start with no final status gets one
    /// here, and so does each start that joined it.
    /// @returns the joined starts that got no final status, or none if the operation
    /// was cancelled. The caller runs them as their own commands.
    std::vector<synnax::task::Command> close();

private:
    const Writer write;
    x::breaker::Breaker breaker{x::breaker::Config{.name = "task.operation"}};
    /// @brief orders the writes of the operation, so no step follows the final status.
    /// Taken before mu.
    std::mutex write_mu;
    /// @brief guards the fields below. Never held during a write.
    std::mutex mu;
    std::string name;
    std::string config_hash;
    std::vector<synnax::task::Command> joined;
    /// @brief the keys of joined starts that got no status yet.
    std::vector<std::string> unanswered;
    /// @brief the stop that cancelled the operation, until check answers it.
    std::optional<synnax::task::Command> stopping;
    /// @brief the last step sent.
    std::optional<synnax::task::Status> current;
    /// @brief when the current step started, or the operation when it has no step.
    x::telem::TimeStamp step_started;
    x::telem::TimeSpan limit;
    bool warned = false;
    /// @brief true when a stop command cancelled the operation.
    bool stopped = false;
    bool finished = false;
    bool closed = false;

    /// @brief records status as the current step. Requires mu.
    void begin_step(const synnax::task::Status &status);
    /// @returns the keys of the command and each joined start. Requires mu.
    [[nodiscard]] std::vector<std::string> keys() const;
    /// @brief writes status once for each key. Requires write_mu.
    void send(synnax::task::Status status, const std::vector<std::string> &keys);
};
}
