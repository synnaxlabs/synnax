// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <utility>

#include "client/cpp/synnax.h"
#include "x/cpp/telem/telem.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/task/task.h"

namespace driver::ethercat {
/// @brief sends the steps of the starts of one task as loading statuses of the task.
/// The Engine reports a step with its lock held, so a thread of StartSteps sends it.
/// Only the latest step waits to be sent, so a slow Core skips old steps. An empty
/// StartSteps sends nothing.
class StartSteps {
    std::shared_ptr<task::Context> ctx;
    synnax::task::Task task;
    std::mutex mu;
    std::condition_variable cv;
    /// @brief the latest step that is not sent yet, with its limit. Guarded by mu.
    std::optional<std::pair<synnax::task::Status, x::telem::TimeSpan>> latest;
    /// @brief true once flush() asks the sender to stop. Guarded by mu.
    bool flushing = false;
    std::thread sender;

    void send() {
        std::unique_lock lock(this->mu);
        while (true) {
            this->cv.wait(lock, [this] {
                return this->latest.has_value() || this->flushing;
            });
            if (!this->latest.has_value()) return;
            auto [status, limit] = std::move(*this->latest);
            this->latest.reset();
            lock.unlock();
            this->ctx->step(status, limit);
            lock.lock();
        }
    }

public:
    StartSteps() = default;

    StartSteps(std::shared_ptr<task::Context> ctx, synnax::task::Task task):
        ctx(std::move(ctx)), task(std::move(task)) {}

    StartSteps(const StartSteps &) = delete;
    StartSteps &operator=(const StartSteps &) = delete;

    /// @brief starts sending the steps of the start that cmd_key runs. Call flush()
    /// before the start answers.
    /// @returns the receiver of the steps, which never blocks, or nullptr for an empty
    /// StartSteps.
    [[nodiscard]] std::function<void(const engine::Engine::Step &)>
    begin(const std::string &cmd_key) {
        if (this->ctx == nullptr) return nullptr;
        {
            std::lock_guard lock(this->mu);
            this->flushing = false;
        }
        this->sender = std::thread(&StartSteps::send, this);
        return [this, cmd_key](const engine::Engine::Step &step) {
            synnax::task::Status status;
            status.owner = synnax::task::resource_id(this->task.key);
            status.name = this->task.name;
            status.variant = synnax::status::VARIANT_LOADING;
            status.message = step.message;
            status.description = step.description;
            status.details.running = false;
            status.details.config_hash = this->task.config_hash;
            status.details.cmd = cmd_key;
            {
                std::lock_guard lock(this->mu);
                this->latest.emplace(std::move(status), step.limit);
            }
            this->cv.notify_one();
        };
    }

    /// @brief sends the step that waits, if any, and stops the sender of the start.
    /// Blocks until the sends end.
    void flush() {
        {
            std::lock_guard lock(this->mu);
            this->flushing = true;
        }
        this->cv.notify_one();
        if (this->sender.joinable()) this->sender.join();
    }
};
}
