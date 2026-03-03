// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <unordered_map>

#include "driver/common/common.h"
#include "driver/task/task.h"

/// modules
#include "x/cpp/status/status.h"

namespace driver::common {
const std::string STOP_CMD_TYPE = "stop";
const std::string START_CMD_TYPE = "start";
const std::string SCAN_CMD_TYPE = "scan";
/// @brief a utility structure for managing the state of tasks.
struct StatusHandler {
    /// @brief the task context used to communicate state changes back to Synnax.
    const std::shared_ptr<task::Context> ctx;
    /// @brief the raw synnax task.
    const synnax::task::Task task;
    /// @brief the accumulated error in the task state.
    x::errors::Error accumulated_err = x::errors::NIL;
    /// @brief the wrapped raw task state that will be sent back to Synnax.
    synnax::task::Status status;

    StatusHandler(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ):
        ctx(ctx), task(task) {
        this->status.name = task.name;
        this->status.details.task = task.key;
        this->status.variant = x::status::VARIANT_SUCCESS;
    }

    /// @brief resets the state handler to its initial state.
    void reset() {
        this->status.variant = x::status::VARIANT_SUCCESS;
        this->accumulated_err = x::errors::NIL;
        this->recent_statuses.clear();
    }

    /// @brief register the provided error in the task state. If err is nil, then it
    /// will be ignored, and false will be returned. Otherwise, the provided error
    /// will override any other accumulated errors.
    bool error(const x::errors::Error &err) {
        if (!err) return false;
        this->status.variant = x::status::VARIANT_ERROR;
        this->accumulated_err = err;
        return true;
    }

    /// @brief immediately sends the provided error as a status update. Unlike
    /// error(), which only accumulates the error for later transmission, this
    /// method sends the error status to the task context right away.
    void send_error(const x::errors::Error &err) {
        if (!err) return;
        this->status.key = this->task.status_key();
        this->status.variant = x::status::VARIANT_ERROR;
        this->status.details.running = false;
        this->status.message = err.data;
        this->accumulated_err = err;
        this->maybe_set_status("err");
    }

    void send_warning(const x::errors::Error &err) { this->send_warning(err.data); }

    /// @brief sends the provided warning string to the task. If the task is in
    /// error state, the error message will be communicated instead.
    void send_warning(const std::string &warning) {
        this->status.key = this->task.status_key();
        // If there's already an error bound, communicate it instead.
        if (!this->accumulated_err) {
            this->status.variant = x::status::VARIANT_WARNING;
            this->status.message = warning;
        } else
            this->status.message = this->accumulated_err.data;
        this->maybe_set_status("warn");
    }

    void clear_warning() {
        if (this->status.variant != x::status::VARIANT_WARNING) return;
        this->status.variant = x::status::VARIANT_SUCCESS;
        this->status.message = "Task running";
        this->maybe_set_status("clr");
    }

    /// @brief sends a start message to the task state, using the provided command
    /// key as part of the state. If an error has been accumulated, then the error
    /// will be sent as part of the state. If the error is nil, then the task will
    /// be marked as running. Bypasses the rate limiter because the Console waits
    /// for command acknowledgments keyed by cmd.
    void send_start(const std::string &cmd_key) {
        this->status.key = this->task.status_key();
        this->status.details.cmd = cmd_key;
        if (!this->accumulated_err) {
            this->status.details.running = true;
            this->status.message = "Task started successfully";
        } else {
            this->status.variant = x::status::VARIANT_ERROR;
            this->status.details.running = false;
            this->status.message = this->accumulated_err.data;
        }
        this->set_status();
    }

    /// @brief sends a stop message to the task state, using the provided command
    /// key as part of the state. If an error has been accumulated, then the error
    /// will be sent as part of the state. Regardless of the error state, the task
    /// will be marked as not running. Bypasses the rate limiter because the
    /// Console waits for command acknowledgments keyed by cmd.
    void send_stop(const std::string &cmd_key) {
        this->status.key = this->task.status_key();
        this->status.details.cmd = cmd_key;
        this->status.details.running = false;
        if (this->accumulated_err) {
            this->status.variant = x::status::VARIANT_ERROR;
            this->status.message = this->accumulated_err.data;
        } else
            this->status.message = "Task stopped successfully";
        this->set_status();
    }

    /// @brief max entries in the dedup map to bound memory usage.
    static constexpr size_t MAX_RECENT_STATUSES = 50;

private:
    /// @brief tracks recently sent statuses to suppress identical repeated
    /// updates. Key is "tag:variant:message", value is the timestamp it was
    /// last sent. The tag distinguishes different send methods so that e.g.
    /// send_error and send_stop with the same message don't suppress each other.
    std::unordered_map<std::string, x::telem::TimeStamp> recent_statuses;
    /// @brief how long a status stays suppressed after being sent.
    static inline const auto STATUS_RATE_LIMIT = 5 * x::telem::SECOND;

    /// @brief unconditionally sends the current status to the server. Used by
    /// send_start and send_stop which must always deliver because the Console
    /// waits for command acknowledgments.
    void set_status() {
        this->status.time = x::telem::TimeStamp::now();
        this->ctx->set_status(this->status);
    }

    /// @brief sends the current status to the server, suppressing identical
    /// statuses that were already sent within STATUS_RATE_LIMIT. The tag
    /// parameter namespaces the dedup key per send method.
    void maybe_set_status(const std::string &tag) {
        const auto now = x::telem::TimeStamp::now();
        std::erase_if(this->recent_statuses, [&](const auto &entry) {
            return (now - entry.second) >= STATUS_RATE_LIMIT;
        });
        if (this->recent_statuses.size() >= MAX_RECENT_STATUSES)
            this->recent_statuses.clear();
        const auto key = tag + ":" + std::string(this->status.variant) + ":" +
                         this->status.message;
        if (this->recent_statuses.contains(key)) return;
        this->recent_statuses[key] = now;
        this->status.time = now;
        this->ctx->set_status(this->status);
    }
};

/// @brief a utility function that appropriately handles configuration errors and
/// communicates them back to Synnax in the standard format.
inline std::pair<std::unique_ptr<task::Task>, bool> handle_config_err(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    std::pair<common::ConfigureResult, x::errors::Error> res
) {
    synnax::task::Status status;
    status.key = task.status_key();
    status.name = task.name;
    status.details.task = task.key;
    status.details.running = false;
    if (res.second) {
        status.variant = x::status::VARIANT_ERROR;
        status.message = res.second.message();
    } else {
        status.variant = x::status::VARIANT_SUCCESS;
        if (!res.first.auto_start) { status.message = "Task configured successfully"; }
    }
    if (res.first.auto_start) {
        task::Command start_cmd(task.key, START_CMD_TYPE, {});
        res.first.task->exec(start_cmd);
    } else
        ctx->set_status(status);
    return {std::move(res.first.task), true};
}
}
