// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "absl/log/log.h"

#include "driver/task/task.h"

namespace driver::common {
/// @brief Creates a task on a rack if a task of the specified type doesn't already
/// exist
/// @param rack The rack to create the task on
/// @param task The task configuration to create
/// @return A pair containing:
///         - bool: true if a new task was created, false if a task of that type already
///         exists
///         - Error: any error that occurred during the operation
inline std::pair<bool, x::errors::Error> create_if_type_not_exists_on_rack(
    const synnax::rack::Rack &rack,
    synnax::task::Task &task
) {
    auto [existing, err] = rack.tasks.retrieve_by_type(std::vector{task.type});
    if (err) return {false, err};
    if (!existing.empty()) return {false, x::errors::NIL};
    return {true, rack.tasks.create(task)};
}

/// @brief Creates and configures initial tasks for a factory
/// @tparam F A factory type implementing task::Factory::configure_task
/// @param factory Pointer to the factory instance that will configure the tasks
/// @param ctx Shared context for task execution
/// @param rack The rack to create tasks for
/// @param task_name Name to assign to the new task
/// @param task_type Type identifier for the task
/// @param integration_name Name of the integration for logging purposes
/// @return Vector of pairs containing:
///         - synnax::task::Task: The created Synnax task configuration
///         - std::unique_ptr<task::Task>: The configured task implementation
/// @note
/// - Returns an empty vector if:
///   1. A task of the specified type already exists
///   2. Task creation fails
///   3. Task configuration fails
/// - Logs errors and warnings
template<typename F>
std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
configure_initial_factory_tasks(
    F *factory,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::rack::Rack &rack,
    const std::string &task_name,
    const std::string &task_type,
    const std::string &integration_name
) {
    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>> tasks;
    auto sy_task = synnax::task::Task{
        .rack = rack.key,
        .name = task_name,
        .type = task_type,
        .config = x::json::json::object(),
        .internal = true
    };
    auto [created, err] = create_if_type_not_exists_on_rack(rack, sy_task);
    if (err) {
        LOG(ERROR) << "[" << integration_name << "] failed to create " << task_name
                   << " on rack " << rack.key << ": " << err;
        return tasks;
    }
    if (!created) {
        VLOG(1) << "[" << integration_name << "] " << task_name
                << " already exists on rack. Skipping creation.";
        return tasks;
    }
    auto [task, _] = factory->configure_task(ctx, sy_task, driver::task::NO_COMMAND);
    if (task != nullptr)
        tasks.emplace_back(sy_task, std::move(task));
    else
        VLOG(1) << "[" << integration_name
                << "] failure to configure initial scan task";
    return tasks;
}

/// @brief Deletes every task of a specific type from a rack
/// @param rack The rack to delete the tasks from
/// @param task_type The type of task to delete
/// @param integration_name Name of the integration for logging purposes
/// @return Error if retrieving the tasks failed
/// @note
/// - Logs success/failure
/// - Silently succeeds if no task of the specified type exists
/// - Useful for cleaning up legacy tasks during system upgrades or reconfigurations
inline x::errors::Error delete_legacy_task_by_type(
    const synnax::rack::Rack &rack,
    const std::string &task_type,
    const std::string &integration_name
) {
    auto [legacy, err] = rack.tasks.retrieve_by_type(std::vector{task_type});
    if (err) return err;
    for (const auto &task: legacy)
        if (const auto del_err = rack.tasks.del(task.key))
            LOG(ERROR) << "[" << integration_name << "] failed to delete legacy "
                       << task_type << " task: " << del_err;
        else
            LOG(INFO) << "[" << integration_name << "] deleted legacy " << task_type
                      << " task";
    return x::errors::NIL;
}
}
