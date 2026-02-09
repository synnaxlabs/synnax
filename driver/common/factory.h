// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "glog/logging.h"

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
/// @note If a task of the specified type already exists, returns {false, err} where err
///       is the error from the retrieval operation
inline std::pair<bool, x::errors::Error> create_if_type_not_exists_on_rack(
    const synnax::rack::Rack &rack,
    synnax::task::Task &task
) {
    auto [_, err] = rack.tasks.retrieve_by_type(task.type);
    if (err.matches(x::errors::NOT_FOUND)) return {true, rack.tasks.create(task)};
    return {false, err};
}

/// @brief Creates and configures initial tasks for a factory
/// @tparam F A factory type that implements the configure_task method with signature:
///           std::pair<std::unique_ptr<task::Task>, x::errors::Error> configure_task(
///               const std::shared_ptr<task::Context> &ctx,
///               const synnax::task::Task &task)
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
/// - Logs errors and warnings through glog
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
    auto sy_task = synnax::task::Task(rack.key, task_name, task_type, "", true);
    auto [created, err] = create_if_type_not_exists_on_rack(rack, sy_task);
    if (err) {
        LOG(ERROR) << "[" << integration_name << "] failed to create" << task_name
                   << " on rack " << rack.key << ": " << err;
        return tasks;
    }
    if (!created) {
        VLOG(1) << "[" << integration_name << "] " << task_name
                << " already exists on rack. Skipping creation.";
        return tasks;
    }
    auto [task, _] = factory->configure_task(ctx, sy_task);
    if (task != nullptr)
        tasks.emplace_back(sy_task, std::move(task));
    else
        VLOG(1) << "[" << integration_name
                << "] failure to configure initial scan task";
    return tasks;
}

/// @brief Deletes a task of a specific type from a rack if it exists
/// @param rack The rack to delete the task from
/// @param task_type The type of task to delete
/// @param integration_name Name of the integration for logging purposes
/// @return Error if any occurred during the operation (NOT_FOUND errors are skipped)
/// @note
/// - Logs success/failure through glog
/// - Silently succeeds if no task of the specified type exists
/// - Useful for cleaning up legacy tasks during system upgrades or reconfigurations
inline x::errors::Error delete_legacy_task_by_type(
    const synnax::rack::Rack &rack,
    const std::string &task_type,
    const std::string &integration_name
) {
    auto [old_heartbeat_task, o_err] = rack.tasks.retrieve_by_type(task_type);
    if (o_err) return o_err.skip(x::errors::NOT_FOUND);
    if (const auto del_err = rack.tasks.del(old_heartbeat_task.key))
        LOG(ERROR) << "[" << integration_name
                   << "] failed to delete legacy heartbeat task: " << del_err;
    else
        LOG(INFO) << "[" << integration_name << "] deleted legacy heartbeat task";
    return o_err;
}
}
