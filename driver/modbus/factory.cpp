// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/modbus/device/device.h"
#include "driver/modbus/modbus.h"
#include "driver/modbus/read_task.h"
#include "driver/modbus/scan_task.h"
#include "driver/modbus/write_task.h"
#include "driver/task/common/factory.h"

namespace modbus {
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";
const std::string SCAN_TASK_TYPE = INTEGRATION_NAME + "_scan";
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_read(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    bool &auto_start
) {
    auto [cfg, err] = ReadTaskConfig::parse(ctx->client, task);
    if (err) return {nullptr, err};
    auto_start = cfg.auto_start;
    auto [dev, d_err] = devs->acquire(cfg.conn);
    if (d_err) return {nullptr, d_err};
    return {
        std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<ReadTaskSource>(dev, std::move(cfg))
        ),
        xerrors::NIL
    };
}

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_scan(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    bool &auto_start
) {
    auto_start = true;
    return {std::make_unique<ScanTask>(ctx, task, devs), xerrors::NIL};
}

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_write(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    bool &auto_start
) {
    auto [cfg, err] = WriteTaskConfig::parse(ctx->client, task);
    if (err) return {nullptr, err};
    auto_start = cfg.auto_start;
    auto [dev, d_err] = devs->acquire(cfg.conn);
    if (d_err) return {nullptr, d_err};
    return {
        std::make_unique<common::WriteTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<WriteTaskSink>(dev, std::move(cfg))
        ),
        xerrors::NIL
    };
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    bool auto_start = false;
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res = {
        nullptr,
        xerrors::NIL
    };
    if (task.type == READ_TASK_TYPE)
        res = configure_read(this->devices, ctx, task, auto_start);
    else if (task.type == WRITE_TASK_TYPE)
        res = configure_write(this->devices, ctx, task, auto_start);
    else if (task.type == SCAN_TASK_TYPE)
        res = configure_scan(this->devices, ctx, task, auto_start);
    return common::handle_config_err(ctx, task, std::move(res), auto_start);
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "Modbus Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
}
