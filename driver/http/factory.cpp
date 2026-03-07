// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/common/factory.h"
#include "driver/common/status.h"
#include "driver/http/http.h"
#include "driver/http/read_task.h"
#include "driver/http/scan_task.h"

namespace driver::http {

std::pair<common::ConfigureResult, x::errors::Error> configure_scan(
    const std::shared_ptr<Processor> &processor,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    common::ConfigureResult result;
    x::json::Parser parser(task.config);
    ScanTaskConfig cfg(parser);
    if (parser.error()) return {std::move(result), parser.error()};
    result.task = std::make_unique<common::ScanTask>(
        std::make_unique<Scanner>(ctx, task, processor),
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
    if (task.type == SCAN_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_scan(processor, ctx, task)
        );
    if (task.type == READ_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_read(ctx, task, processor)
        );
    return {nullptr, false};
}

std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::rack::Rack &rack
) {
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "HTTP Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
}
