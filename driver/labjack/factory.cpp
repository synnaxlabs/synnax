// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#include "glog/logging.h"
#include "driver/labjack/labjack.h"
#include "driver/labjack/scanner.h"
#include "driver/labjack/reader.h"
#include "driver/labjack/writer.h"

std::pair<std::unique_ptr<task::Task>, bool> labjack::Factory::configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
) {
    if (task.type == "labjack_scan")
        return {labjack::ScannerTask::configure(ctx,task), true};
    if (task.type == "labjack_read")
        return {labjack::ReaderTask::configure(ctx, task), true};
    if(task.type == "labjack_write")
        return {labjack::WriterTask::configure(ctx, task), true};
    return {nullptr, false};
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
labjack::Factory::configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
) {
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;

    auto [existing, err] = rack.tasks.retrieveByType("labjack_scan");
    if (err.matches(synnax::NOT_FOUND)) {
        LOG(INFO) << "[labjack] Creating scanner task";
        auto sy_task = synnax::Task(
                rack.key,
                "labjack scanner",
                "labjack_scan",
                "",
                true
        );
        const auto c_err = rack.tasks.create(sy_task);
        if (c_err) {
            LOG(ERROR) << "[labjack] Failed to create scanner task: " << c_err;
            return tasks;
        }
        auto [task, ok] = configure_task(ctx, sy_task);
        if (ok && task != nullptr)
            tasks.emplace_back(sy_task, std::move(task));
        else
            LOG(ERROR) << "[labjack] Failed to configure scanner task";
    } else if (err) {
        LOG(ERROR) << "[labjack] Failed to list existing tasks: " << err;
        return tasks;
    }
    return tasks;
}