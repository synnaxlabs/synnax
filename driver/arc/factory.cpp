// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/arc/arc.h"
#include "driver/arc/task.h"
#include "driver/task/common/status.h"

namespace driver::arc {
std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    if (task.type != TASK_TYPE) return {nullptr, false};
    return task::common::handle_config_err(ctx, task, configure(ctx, task));
}

std::pair<task::common::ConfigureResult, x::errors::Error> Factory::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    task::common::ConfigureResult result;
    auto parser = x::json::Parser(task.config);
    auto [cfg, cfg_err] = TaskConfig::parse(ctx->client, parser);
    if (cfg_err) return {std::move(result), cfg_err};

    auto [arc_task, task_err] = Task::create(task, ctx, cfg);
    if (task_err) return {std::move(result), task_err};

    result.task = std::move(arc_task);
    result.auto_start = cfg.auto_start;
    return {std::move(result), x::errors::NIL};
}

std::string Factory::name() {
    return INTEGRATION_NAME;
}
}
