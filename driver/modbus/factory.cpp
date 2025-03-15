// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "read_task.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/modbus.h"

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_read(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto [cfg, err] = modbus::ReadTaskConfig::parse(ctx->client, task);
    if (err) return {nullptr, err};
    // auto [dev, d_err] = devs->acquire(cfg.device_key);
    // if (d_err) return {nullptr, d_err};
    return {
        std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<modbus::ReadTaskSource>(nullptr, std::move(cfg))
        ),
    };
}

std::pair<std::unique_ptr<task::Task>, bool> modbus::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx, const synnax::Task &task) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res;
    if (task.type == READ_TASK_TYPE)
        res = configure_read(this->devices, ctx, task);
    common::handle_config_err(ctx, task, res.second);
    return {std::move(res.first), true};
}
