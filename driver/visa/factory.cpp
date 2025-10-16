// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"

#include "driver/visa/device/device.h"
#include "driver/visa/visa.h"
#include "driver/visa/read_task.h"
#include "driver/visa/write_task.h"
#include "driver/visa/scan_task.h"
#include "driver/task/common/factory.h"

namespace visa {
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";
const std::string SCAN_TASK_TYPE = INTEGRATION_NAME + "_scan";

const std::string NO_LIBS_MSG = "Cannot create task because the VISA libraries are not "
                                "installed on this System.";

common::ConfigureResult configure_read(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    common::ConfigureResult result;
    auto [cfg, err] = ReadTaskConfig::parse(ctx->client, task);
    if (result.error = err; result.error) return result;
    auto [dev, d_err] = devs->acquire(cfg.conn);
    if (result.error = d_err; result.error) return result;
    result.task = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<ReadTaskSource>(dev, std::move(cfg))
    );
    return result;
}

common::ConfigureResult configure_write(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    common::ConfigureResult result;
    auto [cfg, err] = WriteTaskConfig::parse(ctx->client, task);
    if (result.error = err; result.error) return result;
    auto [dev, d_err] = devs->acquire(cfg.conn);
    if (result.error = d_err; result.error) return result;
    result.task = std::make_unique<common::WriteTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<WriteTaskSink>(dev, std::move(cfg))
    );
    return result;
}

common::ConfigureResult configure_scan(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    common::ConfigureResult result;
    result.task = std::make_unique<ScanTask>(ctx, task, devs);
    result.auto_start = true;
    return result;
}

bool Factory::check_health(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) const {
    if (this->devices != nullptr) return true;
    ctx->set_status(
        {.variant = status::variant::ERR,
         .message = NO_LIBS_MSG,
         .details = synnax::TaskStatusDetails{
             .task = task.key,
         }}
    );
    return false;
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    if (!this->check_health(ctx, task)) return {nullptr, true};
    common::ConfigureResult res;
    if (task.type == READ_TASK_TYPE)
        res = configure_read(this->devices, ctx, task);
    else if (task.type == WRITE_TASK_TYPE)
        res = configure_write(this->devices, ctx, task);
    else if (task.type == SCAN_TASK_TYPE)
        res = configure_scan(this->devices, ctx, task);
    return common::handle_config_err(ctx, task, res);
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    if (!this->check_health(ctx, synnax::Task())) return {};
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "VISA Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
}