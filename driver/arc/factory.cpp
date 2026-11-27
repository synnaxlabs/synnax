// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "driver/arc/arc.h"
#include "driver/arc/task.h"
#include "driver/common/status.h"

namespace driver::arc {
Factory::Factory(
    std::shared_ptr<x::thread::rt::Manager> rt_manager,
    const std::filesystem::path &keys_path,
    const std::string &cluster,
    const std::size_t max_keys,
    x::fs::OnCreate on_create
):
    rt_manager(std::move(rt_manager)) {
    auto [keys, err] = status::Keys::open(
        keys_path,
        cluster,
        max_keys,
        std::move(on_create)
    );
    if (err) {
        err = x::errors::Error(
            err,
            "The Driver cannot load the status keys in " + keys_path.string() + ": " +
                err.data + ". Move " + keys_path.string() +
                " aside, then restart the Driver."
        );
        LOG(ERROR) << "[arc] " << err.data;
    }
    this->keys = {std::move(keys), err};
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key,
    x::breaker::Breaker &cancel
) {
    if (task.type != TASK_TYPE) return {nullptr, false};
    return common::handle_config_err(ctx, task, configure(ctx, task), cmd_key, cancel);
}

std::pair<common::ConfigureResult, x::errors::Error> Factory::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    common::ConfigureResult result;
    auto parser = x::json::Parser(task.config);
    auto [cfg, cfg_err] = TaskConfig::parse(ctx->client, parser);
    if (cfg_err) return {std::move(result), cfg_err};

    auto [arc_task, task_err] = Task::create(
        task,
        ctx,
        cfg,
        this->keys,
        nullptr,
        nullptr,
        this->rt_manager
    );
    if (task_err) return {std::move(result), task_err};

    result.task = std::move(arc_task);
    result.auto_start = cfg.auto_start;
    return {std::move(result), x::errors::NIL};
}

std::string Factory::name() {
    return INTEGRATION_NAME;
}
}
