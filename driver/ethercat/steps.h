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
#include <memory>
#include <string>

#include "client/cpp/synnax.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/task/task.h"

namespace driver::ethercat {
/// @brief sends the steps of the starts of one task as loading statuses of the task.
/// An empty StartSteps sends nothing.
struct StartSteps {
    std::shared_ptr<task::Context> ctx;
    synnax::task::Task task;

    /// @brief returns the receiver of the steps of the start that cmd_key runs, or
    /// nullptr when there is no context.
    [[nodiscard]] std::function<void(const engine::Engine::Step &)>
    of(const std::string &cmd_key) const {
        if (this->ctx == nullptr) return nullptr;
        return [ctx = this->ctx,
                task = this->task,
                cmd_key](const engine::Engine::Step &step) {
            synnax::task::Status status;
            status.owner = synnax::task::resource_id(task.key);
            status.name = task.name;
            status.variant = synnax::status::VARIANT_LOADING;
            status.message = step.message;
            status.description = step.description;
            status.details.running = false;
            status.details.config_hash = task.config_hash;
            status.details.cmd = cmd_key;
            ctx->step(status, step.limit);
        };
    }
};
}
