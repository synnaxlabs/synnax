// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"
#include "driver/driver/opcua/opcua.h"
#include "driver/driver/opcua/scanner.h"
#include "driver/driver/opcua/reader.h"

std::pair<std::unique_ptr<task::Task>, bool> opcua::Factory::configureTask(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type == "opcuaScanner") {
        auto scanner = std::make_unique<Scanner>(ctx, task);
        return {std::move(scanner), true};
    }
    if (task.type == "opcuaReader") {
        auto reader = std::make_unique<Reader>(ctx, task);
        std::cout << "opcuaReader" << std::endl;
        return {std::move(reader), true};
    }
    return {nullptr, false};
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
opcua::Factory::configureInitialTasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;
    auto [existing, err] = rack.tasks.list();
    if (err) {
        LOG(ERROR) << "[OPCUA] Failed to list existing tasks: " << err;
        return tasks;
    }
    // check if a task with the same type and name already exists
    bool hasScanner = false;
    for (const auto &t: existing) {
        if (t.type == "opcuaScanner") {
            LOG(INFO) << "[OPCUA] found existing scanner task. skipping creation";
            hasScanner = true;
        }
    }

    if (!hasScanner) {
        LOG(INFO) << "[OPCUA] creating scanner task";
        auto sy_task = synnax::Task(
            rack.key,
            "OPCUA Scanner",
            "opcuaScanner",
            ""
        );

        std::cout << rack.key << std::endl;
        auto err= rack.tasks.create(sy_task);
        if (err) {
            LOG(ERROR) << "[OPCUA] Failed to create scanner task: " << err;
            return tasks;
        }
        auto [task, ok] = configureTask(ctx, sy_task);
        auto pair = std::make_pair(sy_task, std::move(task));
        if (!ok) tasks.emplace_back(std::move(pair));
    }
    return tasks;
}
