// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "driver/http/processor/processor.h"
#include "driver/task/task.h"

namespace driver::http {
/// @brief integration name for http.
const std::string INTEGRATION_NAME = "http";

/// @brief implements the task::Factory to configure and operate HTTP tasks. Owns a
/// shared Processor that all tasks use for HTTP I/O.
class Factory final : public task::Factory {
    /// @brief shared processor that drives all HTTP I/O through a single event loop.
    std::shared_ptr<Processor> processor;

public:
    Factory(): processor(std::make_shared<Processor>()) {}

    /// @brief returns the integration name.
    /// @returns the integration name.
    std::string name() override { return INTEGRATION_NAME; }

    /// @brief configures a task from a Synnax task definition.
    /// @param ctx the task context providing access to the Synnax client.
    /// @param task the Synnax task definition to configure.
    /// @returns the configured task and whether this factory handled the type.
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) override;

    /// @brief configures tasks that should start automatically on rack boot.
    /// @param ctx the task context providing access to the Synnax client.
    /// @param rack the rack to configure initial tasks for.
    /// @returns pairs of Synnax task definitions and their configured implementations.
    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override;
};
}
