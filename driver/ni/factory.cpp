// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"
#include "driver/ni/ni.h"
#include "nlohmann/json.hpp"
#include <vector>

ni::Factory::Factory() {
    this->dlls_present = ni::dlls_available();
}


std::pair<std::unique_ptr<task::Task>, bool> ni::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (!this->dlls_present) {
        log_dll_error(ctx, task);
        return {nullptr, false};
    }

    if (task.type == "ni_scanner")
        return {ni::ScannerTask::configure(ctx, task), true};
    if (task.type == "ni_analog_read" || task.type == "ni_digital_read")
        return {ni::ReaderTask::configure(ctx, task), true};
    if (task.type == "ni_digital_write")
        return {ni::WriterTask::configure(ctx, task), true};

    LOG(ERROR) << "[ni] Unknown task type: " << task.type << std::endl;
    return {nullptr, false};
}


// creates initial task (scanner)
std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
ni::Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    if (!this->dlls_present) {
        LOG(ERROR) << "[ni] Required NI DLLs not found, cannot configure task." <<
                std::endl;
        return {};
    }
    // generate task list
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;

    // check for existing tasks
    auto [existing, err] = rack.tasks.list();
    if (err) {
        LOG(ERROR) << "[ni] Failed to list existing tasks: " << err;
        return tasks;
    }

    bool hasScanner = false;
    for (const auto &t: existing) {
        if (t.type == "ni_scanner") {
            LOG(INFO) << "[ni] found existing scanner task with key: " << t.key <<
                    " skipping creation." << std::endl;
            hasScanner = true;
        }
    }

    if (!hasScanner) {
        auto sy_task = synnax::Task(
            rack.key,
            "ni scanner",
            "ni_scanner",
            "",
            true
        );
        auto err = rack.tasks.create(sy_task);
        LOG(INFO) << "[ni] created scanner task with key: " << sy_task.key;
        if (err) {
            LOG(ERROR) << "[ni] Failed to create scanner task: " << err;
            return tasks;
        }
        auto [task, ok] = configure_task(ctx, sy_task);
        if (!ok) {
            LOG(ERROR) << "[ni] Failed to configure scanner task: " << err;
            return tasks;
        }
        tasks.emplace_back(
            std::pair<synnax::Task, std::unique_ptr<task::Task> >({
                sy_task,
                std::move(
                    task)
            }));
    }
    return tasks;
}
