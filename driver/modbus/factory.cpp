// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/modbus/read_task.h"
#include "driver/modbus/scan_task.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/modbus.h"

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_read(
    const std::shared_ptr<modbus::device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto [cfg, err] = modbus::ReadTaskConfig::parse(ctx->client, task);
    auto [dev, d_err] = devs->acquire(cfg.conn);
    if (d_err) return {nullptr, d_err};
    return {
        std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<modbus::ReadTaskSource>(nullptr, std::move(cfg))
        ),
        xerrors::NIL
    };
}

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_scan(
    const std::shared_ptr<modbus::device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto scan_task = std::make_unique<modbus::ScanTask>(ctx, task, devs);
    return {std::move(scan_task), xerrors::NIL};
}


std::pair<std::unique_ptr<task::Task>, bool> modbus::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx, const synnax::Task &task) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res;
    if (task.type == READ_TASK_TYPE)
        res = configure_read(this->devices, ctx, task);
    if (task.type == SCAN_TASK_TYPE)
        res = configure_scan(this->devices, ctx, task);
    common::handle_config_err(ctx, task, res.second);
    return {std::move(res.first), true};
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > modbus::Factory::
configure_initial_tasks(const std::shared_ptr<task::Context> &ctx,
                        const synnax::Rack &rack) {
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;

    auto [existing, err] = rack.tasks.list();
    if (err) {
        LOG(ERROR) << "[modbus] failed to list existing tasks: " << err;
        return tasks;
    }

    bool has_scanner = false;
    for (const auto &t: existing)
        if (t.type == SCAN_TASK_TYPE) has_scanner = true;
    if (has_scanner) return tasks;

    auto sy_task = synnax::Task(rack.key, "modbus scanner", SCAN_TASK_TYPE, "", true);
    const auto c_err = rack.tasks.create(sy_task);
    if (c_err) {
        LOG(ERROR) << "[modbus] failed to create scanner task: " << c_err;
        return tasks;
    }
    auto [task, ok] = configure_task(ctx, sy_task);
    if (!ok) {
        LOG(ERROR) << "[modbus] failed to configure scanner task: " << c_err;
        return tasks;
    }
    tasks.emplace_back(
        std::pair<synnax::Task, std::unique_ptr<task::Task> >({
            sy_task,
            std::move(task)
        })
    );
    return tasks;
}
