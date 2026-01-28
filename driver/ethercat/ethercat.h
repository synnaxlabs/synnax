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

#include "driver/task/task.h"

namespace ethercat {
/// Integration name used for configuration and logging.
const std::string INTEGRATION_NAME = "ethercat";

/// Read task type identifier.
const std::string READ_TASK_TYPE = "ethercat_read";

/// Write task type identifier.
const std::string WRITE_TASK_TYPE = "ethercat_write";

/// Scan task type identifier for device discovery.
const std::string SCAN_TASK_TYPE = "ethercat_scan";

/// Factory for creating EtherCAT tasks.
///
/// The factory manages CyclicEngine instances per network interface. Tasks share
/// the engine for their interface, allowing multiple tasks to use the same
/// EtherCAT master for cyclic PDO exchange.
class Factory final : public task::Factory {
    struct Impl;
    std::unique_ptr<Impl> impl_;

public:
    Factory();
    ~Factory() override;

    Factory(const Factory &) = delete;
    Factory &operator=(const Factory &) = delete;

    /// Creates a task based on the task configuration.
    /// @param ctx Task context providing access to Synnax client and state.
    /// @param task The Synnax task to configure.
    /// @returns A pair containing:
    ///          - unique_ptr<task::Task>: The created task, or nullptr if not matched.
    ///          - bool: True if this factory handled the task type.
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    /// Creates initial tasks when the driver starts (e.g., scan tasks).
    /// @param ctx Task context.
    /// @param rack_key The key of the rack this driver belongs to.
    /// @returns Vector of task pairs (task configuration, task instance).
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;
};
}
