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

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/strings/strings.h"

#include "driver/common/common.h"
#include "driver/task/task.h"

namespace driver::common {
const std::string SCAN_CMD_TYPE = "scan";
/// @brief a utility structure for managing the state of tasks. A status that answers no
/// command is sent at most once per window, and only when it changes.
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
        this->status.owner = synnax::task::resource_id(task.key);
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
        this->status.details.running = false;
        this->set_text(message, err.data);
        this->maybe_set_status();
    }

    /// @brief sends a warning to the task. If the task is in error state, the error is
    /// communicated instead.
    void send_warning(const std::string &message, const std::string &description) {
        if (!this->accumulated_err) {
            this->status.variant = synnax::status::VARIANT_WARNING;
            this->set_text(message, description);
        } else
            this->set_text(this->accumulated_msg, this->accumulated_err.data);
        this->maybe_set_status();
    }

    /// @brief shows warning while the task runs with no warning or error of its own. An
    /// empty warning stops showing it. A warning or error of the task hides it, and
    /// clear_warning does not clear it. send_stop adds its description to the stop
    /// status and clears it. The flusher thread sends the change, so the caller never
    /// writes to the Core, but it can wait for a write in progress. Safe to call from
    /// any thread.
    void set_background_warning(const Warning &warning) {
        std::lock_guard lock(this->mu);
        if (this->closed || this->background == warning) return;
        this->background = warning;
        if (this->last.has_value()) this->defer(this->shown(*this->last));
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
    /// keyed by cmd. A start that failed with x::errors::CANCELLED sends nothing and
    /// clears the error, since its operation answers it as stopped.
    void send_start(const std::string &cmd_key) {
        // A start ended by its cancel is answered as stopped, and its error must not
        // answer the stop that follows.
        if (this->accumulated_err.matches(x::errors::CANCELLED)) {
            this->reset();
            return;
        }
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
    /// not running. Adds the description of a background warning, and clears it. Skips
    /// the window because the Console waits for command acknowledgments keyed by cmd.
    void send_stop(const std::string &cmd_key) {
        this->status.details.cmd = cmd_key;
        this->status.details.running = false;
        if (this->accumulated_err) {
            this->status.variant = synnax::status::VARIANT_ERROR;
            this->set_text(this->accumulated_msg, this->accumulated_err.data);
        } else {
            this->status.variant = synnax::status::VARIANT_SUCCESS;
            this->set_text("Stopped task");
        }
        {
            std::lock_guard lock(this->mu);
            if (!this->background.empty()) {
                auto &description = this->status.description;
                description += (description.empty() ? "" : " ") +
                               this->background.description;
                this->background = {};
            }
        }
        this->set_status();
    }

    /// @brief answers cmd_key by re-sending the current status unchanged, for a command
    /// that needs no work. Skips the window because the Console waits for command
    /// acknowledgments keyed by cmd.
    void ack(const std::string &cmd_key, const bool running) {
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
    /// @brief guards pending, sent, sent_at, closed, background, and last, and orders
    /// writes to ctx.
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
    /// @brief the warning that set_background_warning shows.
    Warning background;
    /// @brief the last status the task sent, before background is applied.
    std::optional<synnax::task::Status> last;
    /// @brief sends the pending status when its window ends. Started on first use.
    std::thread flusher;

    /// @brief sets the status text. Every transition sets both fields, so a stale
    /// description never outlives the message it explained.
    void set_text(const std::string &message, const std::string &description = "") {
        this->status.message = message;
        this->status.description = description;
    }

    [[nodiscard]] static bool
    same(const synnax::task::Status &a, const synnax::task::Status &b) {
        return a.variant == b.variant && a.message == b.message &&
               a.description == b.description && a.details.running == b.details.running;
    }

    /// @brief returns status with background applied. Requires mu.
    [[nodiscard]] synnax::task::Status shown(synnax::task::Status status) const {
        if (this->background.empty() || !status.details.running ||
            status.variant != synnax::status::VARIANT_SUCCESS)
            return status;
        status.variant = synnax::status::VARIANT_WARNING;
        status.message = this->background.message;
        status.description = this->background.description;
        return status;
    }

    /// @brief hands status to the flusher, unless it equals the last one sent.
    /// Requires mu.
    void defer(synnax::task::Status status) {
        if (same(status, this->sent)) {
            this->pending.reset();
            return;
        }
        this->pending = std::move(status);
        if (!this->flusher.joinable())
            this->flusher = std::thread([this] { this->flush(); });
        this->cv.notify_one();
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
            this->last = this->status;
            this->last->details.cmd = driver::task::NO_COMMAND;
            // A command answer never shows background, so show it again after.
            if (!this->closed) this->defer(this->shown(*this->last));
        }
        // The key answers one command. Later unsolicited updates reuse this status
        // object and must not claim it.
        this->status.details.cmd = driver::task::NO_COMMAND;
    }

    /// @brief sends the current status, at most once per window. A status inside the
    /// window waits for the window to end, and only the latest one is sent. A status
    /// equal to the last one sent is not sent again.
    void maybe_set_status() {
        std::lock_guard lock(this->mu);
        if (this->closed) return;
        this->last = this->status;
        auto status = this->shown(this->status);
        const auto now = std::chrono::steady_clock::now();
        if (same(status, this->sent) || now - this->sent_at < this->window) {
            this->defer(std::move(status));
            return;
        }
        this->pending.reset();
        this->sent_at = now;
        this->write(std::move(status));
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
/// command driving the deploy, or NO_COMMAND at boot. A failed configure writes an
/// error status, except at boot for a task that another rack runs. A successful
/// configure writes no status: the start that follows it answers the command. Only a
/// configure with no command auto-starts the task, because a start command that drives
/// a deploy starts the task with its own key. An auto-start first writes a loading
/// status, which the task replaces when it answers the start. cancel is passed to the
/// auto-start. A configure that fails with x::errors::CANCELLED writes no status, so
/// the start is answered as stopped.
inline std::pair<std::unique_ptr<task::Task>, bool> handle_config_err(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    std::pair<common::ConfigureResult, x::errors::Error> res,
    const std::string &cmd_key,
    x::breaker::Breaker &cancel
) {
    const bool commanded = cmd_key != driver::task::NO_COMMAND;
    if (res.second.matches(x::errors::CANCELLED))
        return {std::move(res.first.task), true};
    if (res.second) {
        synnax::task::Status status;
        status.owner = synnax::task::resource_id(task.key);
        status.name = task.name;
        status.details.config_hash = task.config_hash;
        status.details.running = false;
        status.variant = synnax::status::VARIANT_ERROR;
        status.message = "Failed to configure task";
        status.description = config_err_description(res.second);
        // Ack the start so a waiting caller resolves with the failure, not a timeout.
        status.details.cmd = cmd_key;
        // A task moved here runs on its old Driver until a start arrives.
        const bool moved = task.status.has_value() && task.status->details.running &&
                           task.status->details.rack != task.rack;
        if (!commanded && moved)
            LOG(WARNING) << "[driver] failed to configure task " << task.name << ": "
                         << res.second;
        else
            ctx->set_status(status);
        return {std::move(res.first.task), true};
    }
    if (!commanded && res.first.auto_start) {
        // A task may answer the start later, from its own thread. Until then, this
        // status replaces the one stored by the last run.
        synnax::task::Status starting;
        starting.owner = synnax::task::resource_id(task.key);
        starting.name = task.name;
        starting.variant = synnax::status::VARIANT_LOADING;
        starting.message = "Starting task";
        starting.details.config_hash = task.config_hash;
        ctx->set_status(starting);
        synnax::task::Command start_cmd{
            .task = task.key,
            .type = synnax::task::START_CMD_TYPE,
        };
        res.first.task->exec(start_cmd, cancel);
    }
    return {std::move(res.first.task), true};
}
}
