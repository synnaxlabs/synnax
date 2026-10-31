// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <condition_variable>
#include <mutex>
#include <optional>
#include <thread>

#include "absl/log/log.h"

#include "x/cpp/json/json.h"
#include "x/cpp/strings/strings.h"

#include "driver/common/common.h"
#include "driver/task/task.h"

namespace driver::common {
const std::string STOP_CMD_TYPE = "stop";
const std::string SCAN_CMD_TYPE = "scan";
/// @brief a utility structure for managing the state of tasks. A status that answers no
/// command is sent at most once per window.
struct StatusHandler {
    /// @brief the task context used to communicate state changes back to Synnax.
    const std::shared_ptr<driver::task::Context> ctx;
    /// @brief the raw synnax task.
    const synnax::task::Task task;
    /// @brief the wrapped raw task state that will be sent back to Synnax.
    synnax::task::Status status;

    StatusHandler(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const x::telem::TimeSpan window = x::telem::SECOND
    ):
        ctx(ctx),
        task(task),
        window(window.chrono()),
        sent_at(std::chrono::steady_clock::now() - this->window) {
        this->status.name = task.name;
        this->status.details.task = task.key;
        // The revision this instance was built from. A difference against the stored
        // hash is what the Console renders as drift.
        this->status.details.config_hash = task.config_hash;
        this->status.variant = synnax::status::VARIANT_SUCCESS;
        this->status.message = "Task configured";
    }

    ~StatusHandler() {
        this->close();
        if (this->flusher.joinable()) this->flusher.join();
    }

    StatusHandler(const StatusHandler &) = delete;
    StatusHandler &operator=(const StatusHandler &) = delete;

    /// @brief drops a pending status and sends no more statuses that answer no command.
    /// Call it when a stop without a stop status starts. Such a task is being deleted,
    /// moved, or rebuilt, so it must write nothing more while its stop waits on
    /// hardware.
    void close() {
        {
            std::lock_guard lock(this->mu);
            this->closed = true;
            this->pending.reset();
        }
        this->cv.notify_one();
    }

    /// @brief resets the state handler to its initial state.
    void reset() {
        this->status.variant = synnax::status::VARIANT_SUCCESS;
        this->accumulated_err = x::errors::NIL;
        this->accumulated_msg.clear();
    }

    /// @brief registers err in the task state, with message saying what failed. A nil
    /// err is ignored and returns false. Otherwise, err overrides any accumulated error
    /// and the call returns true.
    bool error(const x::errors::Error &err, const std::string &message) {
        if (!err) return false;
        this->status.variant = synnax::status::VARIANT_ERROR;
        this->accumulated_err = err;
        this->accumulated_msg = message;
        return true;
    }

    /// @brief registers err like error(), then sends it right away.
    void send_error(const x::errors::Error &err, const std::string &message) {
        if (!this->error(err, message)) return;
        this->status.key = synnax::task::status_key(this->task);
        this->status.details.running = false;
        this->set_text(message, err.data);
        this->maybe_set_status();
    }

    /// @brief sends a warning to the task. If the task is in error state, the error is
    /// communicated instead.
    void send_warning(const std::string &message, const std::string &description) {
        this->status.key = synnax::task::status_key(this->task);
        if (!this->accumulated_err) {
            this->status.variant = synnax::status::VARIANT_WARNING;
            this->set_text(message, description);
        } else
            this->set_text(this->accumulated_msg, this->accumulated_err.data);
        this->maybe_set_status();
    }

    void clear_warning() {
        if (this->status.variant != synnax::status::VARIANT_WARNING) return;
        this->status.variant = synnax::status::VARIANT_SUCCESS;
        this->set_text("Task running");
        this->maybe_set_status();
    }

    /// @brief sends a start message to the task state, using the provided command key
    /// as part of the state. If an error has been accumulated, then the error will be
    /// sent as part of the state. If the error is nil, then the task will be marked as
    /// running. Skips the window because the Console waits for command acknowledgments
    /// keyed by cmd.
    void send_start(const std::string &cmd_key) {
        this->status.key = synnax::task::status_key(this->task);
        this->status.details.cmd = cmd_key;
        if (!this->accumulated_err) {
            this->status.details.running = true;
            this->set_text("Started task");
        } else {
            this->status.variant = synnax::status::VARIANT_ERROR;
            this->status.details.running = false;
            this->set_text(this->accumulated_msg, this->accumulated_err.data);
        }
        this->set_status();
    }

    /// @brief sends a stop message to the task state, using the provided command key as
    /// part of the state. If an error has been accumulated, then the error will be sent
    /// as part of the state. Regardless of the error state, the task will be marked as
    /// not running. Skips the window because the Console waits for command
    /// acknowledgments keyed by cmd.
    void send_stop(const std::string &cmd_key) {
        this->status.key = synnax::task::status_key(this->task);
        this->status.details.cmd = cmd_key;
        this->status.details.running = false;
        if (this->accumulated_err) {
            this->status.variant = synnax::status::VARIANT_ERROR;
            this->set_text(this->accumulated_msg, this->accumulated_err.data);
        } else {
            this->status.variant = synnax::status::VARIANT_SUCCESS;
            this->set_text("Stopped task");
        }
        this->set_status();
    }

    /// @brief answers cmd_key by re-sending the current status unchanged, for a command
    /// that needs no work. Skips the window because the Console waits for command
    /// acknowledgments keyed by cmd.
    void ack(const std::string &cmd_key, const bool running) {
        this->status.key = synnax::task::status_key(this->task);
        this->status.details.running = running;
        this->status.details.cmd = cmd_key;
        this->set_status();
    }

private:
    /// @brief the accumulated error in the task state.
    x::errors::Error accumulated_err = x::errors::NIL;
    /// @brief what failed when accumulated_err occurred, sent as the status message.
    std::string accumulated_msg;
    /// @brief the shortest time between two statuses that answer no command.
    const std::chrono::nanoseconds window;
    /// @brief guards pending, sent, sent_at, and closed, and orders writes to ctx.
    std::mutex mu;
    /// @brief wakes the flusher when a status starts to wait or the handler closes.
    std::condition_variable cv;
    /// @brief the latest status that waits for the window to end.
    std::optional<synnax::task::Status> pending;
    /// @brief the last status written to ctx.
    synnax::task::Status sent;
    /// @brief when the current window started.
    std::chrono::steady_clock::time_point sent_at;
    /// @brief true once close is called.
    bool closed = false;
    /// @brief sends the pending status when its window ends. Started on first use.
    std::thread flusher;

    /// @brief sets the status text. Every transition sets both fields, so a stale
    /// description never outlives the message it explained.
    void set_text(const std::string &message, const std::string &description = "") {
        this->status.message = message;
        this->status.description = description;
    }

    /// @brief writes status to ctx. Requires mu.
    void write(synnax::task::Status status) {
        status.time = x::telem::TimeStamp::now();
        this->ctx->set_status(status);
        this->sent = std::move(status);
    }

    /// @brief sends the current status at once, outside the window, because the Console
    /// waits for command acknowledgments. The status carries the latest state, so it
    /// replaces a pending one.
    void set_status() {
        {
            std::lock_guard lock(this->mu);
            this->pending.reset();
            this->write(this->status);
        }
        // The key answers one command. Later unsolicited updates reuse this status
        // object and must not claim it.
        this->status.details.cmd = driver::task::NO_COMMAND;
    }

    /// @brief sends the current status, at most once per window. A status inside the
    /// window waits for the window to end, and only the latest one is sent. A status
    /// equal to the last one sent is not sent again inside the window.
    void maybe_set_status() {
        std::lock_guard lock(this->mu);
        if (this->closed) return;
        const auto now = std::chrono::steady_clock::now();
        if (now - this->sent_at >= this->window) {
            this->pending.reset();
            this->sent_at = now;
            this->write(this->status);
            return;
        }
        if (this->status.variant == this->sent.variant &&
            this->status.message == this->sent.message &&
            this->status.description == this->sent.description &&
            this->status.details.running == this->sent.details.running) {
            this->pending.reset();
            return;
        }
        this->pending = this->status;
        if (!this->flusher.joinable())
            this->flusher = std::thread([this] { this->flush(); });
        this->cv.notify_one();
    }

    /// @brief sends each pending status when its window ends, until the handler closes.
    void flush() {
        std::unique_lock lock(this->mu);
        while (!this->closed) {
            if (!this->pending.has_value()) {
                this->cv.wait(lock);
                continue;
            }
            // Another thread can start a new window while the flusher waits.
            const auto deadline = this->sent_at + this->window;
            if (std::chrono::steady_clock::now() < deadline) {
                this->cv.wait_until(lock, deadline);
                continue;
            }
            this->sent_at = std::chrono::steady_clock::now();
            auto status = std::move(*this->pending);
            this->pending.reset();
            this->write(std::move(status));
        }
    }
};

/// @brief returns the text of a configuration error, with each field error of a parser
/// validation error on its own line as "path: message".
inline std::string config_err_description(const x::errors::Error &err) {
    const auto parsed = x::json::json::parse(err.data, nullptr, false);
    if (!err.matches(x::errors::VALIDATION) || !parsed.is_object() ||
        !parsed.contains("errors") || !parsed["errors"].is_array())
        return err.data;
    std::vector<std::string> lines;
    for (const auto &field: parsed["errors"])
        lines.push_back(field.value("path", "") + ": " + field.value("message", ""));
    return x::strings::join(lines, "\n");
}

/// @brief a utility function that appropriately handles configuration errors and
/// communicates them back to Synnax in the standard format. cmd_key is the start
/// command driving the deploy, empty at boot. A boot configure that does not
/// auto-start logs its failures instead of reporting them. A successful configure
/// writes no status: the start that follows it answers the command.
inline std::pair<std::unique_ptr<task::Task>, bool> handle_config_err(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    std::pair<common::ConfigureResult, x::errors::Error> res,
    const std::string &cmd_key
) {
    const bool boot = cmd_key == driver::task::NO_COMMAND;
    if (res.second) {
        synnax::task::Status status;
        status.key = synnax::task::status_key(task);
        status.name = task.name;
        status.details.task = task.key;
        status.details.config_hash = task.config_hash;
        status.details.running = false;
        status.variant = synnax::status::VARIANT_ERROR;
        status.message = "Failed to configure task";
        status.description = config_err_description(res.second);
        // Ack the start so a waiting caller resolves with the failure, not a timeout.
        status.details.cmd = cmd_key;
        if (boot && !res.first.auto_start)
            LOG(WARNING) << "[driver] failed to configure task " << task.name << ": "
                         << res.second;
        else
            ctx->set_status(status);
        return {std::move(res.first.task), true};
    }
    if (res.first.auto_start) {
        synnax::task::Command start_cmd{
            .task = task.key,
            .type = synnax::task::START_CMD_TYPE,
        };
        res.first.task->exec(start_cmd);
    }
    return {std::move(res.first.task), true};
}
}
