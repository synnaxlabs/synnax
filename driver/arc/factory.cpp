// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sys/stat.h>

#include "driver/arc/arc.h"
#include "driver/arc/task.h"

namespace arc {
std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type != TASK_TYPE) return {nullptr, false};
    synnax::TaskStatus stat{
        .key = task.status_key(),
        .name = task.name,
        .variant = "error",
        .details = synnax::TaskStatusDetails{
            .task = task.key,
            .running = false,
        },
    };

    auto parser = xjson::Parser(task.config);
    auto [cfg, cfg_err] = TaskConfig::parse(ctx->client, parser);
    if (cfg_err) {
        stat.message = "Failed to configure";
        stat.description = cfg_err.message();
        ctx->set_status(stat);
        return {nullptr, true};
    }

    auto [runtime, rt_err] = load_runtime(cfg, ctx->client);
    if (rt_err) {
        stat.message = "Failed to configure";
        stat.description = rt_err.message();
        ctx->set_status(stat);
        return {nullptr, true};
    }
    return {std::make_unique<Task>(task, ctx, std::move(runtime), cfg), true};
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    return {};
}

std::string Factory::name() {
    return INTEGRATION_NAME;
}
}
