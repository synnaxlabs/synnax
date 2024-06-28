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

#ifdef _WIN32
#include "dll_check_windows.h"
#else
#include "dll_check_linux.h"
#endif

ni::Factory::Factory() {
    std::vector<std::string> dlls = {
        "nicaiu.dll",
        "nipalu.dll",
        "nimdbgu.dll",
        "nidmxfu.dll",
        "niorbu.dll",
        "nimxdfu.dll",
        "nimru2u.dll",
        "nipalut.dll",
        "nicrtsiu.dll",
        "nimhwcfu.dll",
        "nidimu.dll",
        "nirpc.dll",
        "nimdnsResponder.dll",
        "nirocoapi.dll",
        "nisysapi.dll",
        "niprtsiu.dll"
    };

    this->dlls_present = true;
    for (const auto &dll: dlls) {
        if (!does_dll_exist(dll.c_str())) {
            this->dlls_present = false;
        }
    }
    if (this->dlls_present) LOG(INFO) << "[ni] All required DLLs found.";
}


std::pair<std::unique_ptr<task::Task>, bool> ni::Factory::configureTask(
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
    else
        LOG(ERROR) << "[ni] Unknown task type: " << task.type << std::endl;
    return {nullptr, false};
}


// creates initial task (scanner)
std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
ni::Factory::configureInitialTasks(
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
        std::cout << "Creating niScanner task" << std::endl;
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
        auto [task, ok] = configureTask(ctx, sy_task);
        if (!ok) {
            LOG(ERROR) << "[ni] Failed to configure scanner task: " << err;
            return tasks;
        }
        tasks.emplace_back(
            std::pair<synnax::Task, std::unique_ptr<task::Task> >({
                sy_task, std::move(task)
            }));
    }
    return tasks;
}
