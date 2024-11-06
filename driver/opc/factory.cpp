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
#include "driver/opc/writer.h"

std::pair<std::unique_ptr<task::Task>, bool> opc::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type == "opc_scan"){
        return {std::make_unique<Scanner>(ctx, task), true};
    }
    if (task.type == "opc_read")
        return {ReaderTask::configure(ctx, task), true};
    if (task.type == "opc_write")
        return {WriterTask::configure(ctx, task), true};
    return {nullptr, false};
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
opc::Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;

    auto [old_scanner, err2] = rack.tasks.retrieveByType("opcScanner");
    if(err2 == freighter::NIL) {
        LOG(INFO) << "[opc] Removing old scanner task";
       rack.tasks.del(old_scanner.key);
    }

    auto [existing, err] = rack.tasks.retrieveByType("opc_scan");
    if (err.matches(synnax::NOT_FOUND)) {
        LOG(INFO) << "[opc] Creating scanner task";
        auto sy_task = synnax::Task(
            rack.key,
            "opc Scanner",
            "opc_scan",
            "",
            true
        );
        const auto c_err = rack.tasks.create(sy_task);
        if (c_err) {
            LOG(ERROR) << "[opc] Failed to create scanner task: " << c_err;
            return tasks;
        }
        auto [task, ok] = configure_task(ctx, sy_task);
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
