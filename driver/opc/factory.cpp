// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"
#include "driver/opc/opc.h"
#include "driver/opc/scanner.h"
#include "driver/opc/reader.h"
//#include "driver/opc/writer.h"

std::pair<std::unique_ptr<task::Task>, bool> opc::Factory::configureTask(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type == "opcScanner")
        LOG(INFO) << "[opc] Configuring scanner task";
        return {std::make_unique<Scanner>(ctx, task), true};
    if (task.type == "opc_read")
        return {Reader::configure(ctx, task), true};
//    if (task.type == "opc_write"){
//        LOG(INFO) << "[opc] Configuring writer task";
//        return {WriterTask::configure(ctx, task), true};
//    }
    return {nullptr, false};
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
opc::Factory::configureInitialTasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;
    auto [existing, err] = rack.tasks.retrieveByType("opcScanner");
    if (err.matches(synnax::NOT_FOUND)) {
        auto sy_task = synnax::Task(
            rack.key,
            "opc Scanner",
            "opcScanner",
            "",
            true
        );
        const auto c_err = rack.tasks.create(sy_task);
        if (c_err) {
            LOG(ERROR) << "[opc] Failed to create scanner task: " << c_err;
            return tasks;
        }
        auto [task, ok] = configureTask(ctx, sy_task);
        if (ok && task != nullptr)
            tasks.emplace_back(sy_task, std::move(task));
        else
            LOG(ERROR) << "[opc] Failed to configure scanner task";
    } else if (err) {
        LOG(ERROR) << "[opc] Failed to list existing tasks: " << err;
        return tasks;
    }
    return tasks;
}
