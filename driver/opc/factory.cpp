// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/json/json.h"

#include "driver/common/factory.h"
#include "driver/common/scan_task.h"
#include "driver/opc/opc.h"
#include "driver/opc/read_task.h"
#include "driver/opc/scan_task.h"
#include "driver/opc/write_task.h"

namespace driver::opc {
std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<connection::Pool> &pool
) {
    common::ConfigureResult result;
    auto [cfg, err] = ReadTaskConfig::parse(ctx->client, task);
    if (err) return {std::move(result), err};
    std::unique_ptr<driver::task::common::Source> s;
    if (cfg.array_size > 1)
        s = std::make_unique<ArrayReadTaskSource>(pool, std::move(cfg));
    else
        s = std::make_unique<UnaryReadTaskSource>(pool, std::move(cfg));
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<driver::task::common::ReadTask>(
        task,
        ctx,
        x::breaker::default_config(task.name),
        std::move(s)
    );
    return {std::move(result), x::errors::NIL};
}

std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<connection::Pool> &pool
) {
    common::ConfigureResult result;
    auto [cfg, err] = WriteTaskConfig::parse(ctx->client, task);
    if (err) return {std::move(result), err};
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<driver::task::common::WriteTask>(
        task,
        ctx,
        x::breaker::default_config(task.name),
        std::make_unique<WriteTaskSink>(pool, std::move(cfg))
    );
    return {std::move(result), x::errors::NIL};
}

std::pair<common::ConfigureResult, x::errors::Error> configure_scan(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<connection::Pool> &pool
) {
    common::ConfigureResult result;
    auto parser = x::json::Parser(task.config);
    auto cfg = ScanTaskConfig(parser);
    if (parser.error()) return {std::move(result), parser.error()};
    result.task = std::make_unique<common::ScanTask>(
        std::make_unique<Scanner>(ctx, task, pool),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.scan_rate
    );
    result.auto_start = cfg.enabled;
    return {std::move(result), x::errors::NIL};
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    std::pair<common::ConfigureResult, x::errors::Error> res;
    if (task.type == SCAN_TASK_TYPE)
        res = configure_scan(ctx, task, conn_pool_);
    else if (task.type == READ_TASK_TYPE)
        res = configure_read(ctx, task, conn_pool_);
    else if (task.type == WRITE_TASK_TYPE)
        res = configure_write(ctx, task, conn_pool_);
    return driver::task::common::handle_config_err(ctx, task, std::move(res));
}

std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::rack::Rack &rack
) {
    driver::task::common::delete_legacy_task_by_type(
        rack,
        "opcScanner",
        INTEGRATION_NAME
    );
    return driver::task::common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "OPC UA Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
}
