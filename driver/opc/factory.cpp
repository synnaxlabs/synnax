// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"

#include "driver/opc/opc.h"
#include "driver/opc/read_task.h"
#include "driver/opc/scan_task.h"
#include "driver/opc/write_task.h"
#include "driver/task/common/factory.h"

common::ConfigureResult configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    const std::shared_ptr<opc::connection::Pool> &pool
) {
    common::ConfigureResult result;
    auto [cfg, err] = opc::ReadTaskConfig::parse(ctx->client, task);
    if (err) {
        result.error = err;
        return result;
    }
    std::unique_ptr<common::Source> s;
    if (cfg.array_size > 1)
        s = std::make_unique<opc::ArrayReadTaskSource>(pool, std::move(cfg));
    else
        s = std::make_unique<opc::UnaryReadTaskSource>(pool, std::move(cfg));
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::move(s)
    );
    return result;
}

common::ConfigureResult configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    const std::shared_ptr<opc::connection::Pool> &pool
) {
    common::ConfigureResult result;
    auto [cfg, err] = opc::WriteTaskConfig::parse(ctx->client, task);
    if (err) {
        result.error = err;
        return result;
    }
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<common::WriteTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<opc::WriteTaskSink>(pool, std::move(cfg))
    );
    return result;
}

std::pair<std::unique_ptr<task::Task>, bool> opc::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    common::ConfigureResult res;
    if (task.type == SCAN_TASK_TYPE)
        return {std::make_unique<ScanTask>(ctx, task, conn_pool_), true};
    if (task.type == READ_TASK_TYPE) res = configure_read(ctx, task, conn_pool_);
    if (task.type == WRITE_TASK_TYPE) res = configure_write(ctx, task, conn_pool_);
    return common::handle_config_err(ctx, task, res);
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
opc::Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    common::delete_legacy_task_by_type(rack, "opcScanner", INTEGRATION_NAME);
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "OPC UA Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
