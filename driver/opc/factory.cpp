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
#include "driver/opc/scan_task.h"
#include "driver/opc/read_task.h"
#include "driver/opc/write_task.h"

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto [cfg, err] = opc::ReadTaskConfig::parse(ctx->client, task);
    if (err) return {nullptr, err};
    auto [client, c_err] = util::connect(cfg.conn, "ABC");
    if (c_err) return {nullptr, err};
    std::unique_ptr<common::Source> s;
    if (cfg.array_size > 1)
        s = std::make_unique<opc::ArrayReadTaskSource>(client, cfg);
    else
        s = std::make_unique<opc::UnaryReadTaskSource>(client, cfg);
    return {
        std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::move(s)
        ),
        xerrors::NIL
    };
}

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto [cfg, err] = opc::WriteTaskConfig::parse(ctx->client, task);
    if (err) return {nullptr, err};
    auto [client, c_err] = util::connect(cfg.conn, "ABC");
    if (c_err) return {nullptr, err};
    return {
        std::make_unique<common::WriteTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<opc::WriteTaskSink>(std::move(cfg), client)
        ),
        xerrors::NIL
    };
}


std::pair<std::unique_ptr<task::Task>, bool> opc::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res;
    if (task.type == SCAN_TASK_TYPE)
        return {std::make_unique<ScanTask>(ctx, task), true};
    if (task.type == READ_TASK_TYPE)
        res = configure_read(ctx, task);
    if (task.type == WRITE_TASK_TYPE)
        res = configure_write(ctx, task);
    handle_config_err(ctx, task, res.second);
    return {std::move(res.first), true};
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
opc::Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>> tasks;

    auto [old_scanner, err2] = rack.tasks.retrieveByType("opcScanner");
    if (err2 == xerrors::NIL) {
        LOG(INFO) << "[opc] Removing old scanner task";
        if (auto del_err = rack.tasks.del(old_scanner.key)) {
            LOG(ERROR) << "[opc] Failed to delete old scanner task: " << del_err;
            return tasks;
        }
    }

    auto [existing, err] = rack.tasks.retrieveByType("opc_scan");
    if (err.matches(xerrors::NOT_FOUND)) {
        LOG(INFO) << "[opc] creating scanner task";
        auto sy_task = synnax::Task(
            rack.key,
            "opc Scanner",
            SCAN_TASK_TYPE,
            "",
            true
        );
        if (const auto c_err = rack.tasks.create(sy_task)) {
            LOG(ERROR) << "[opc] Failed to create scanner task: " << c_err;
            return tasks;
        }
        auto [task, ok] = configure_task(ctx, sy_task);
        if (ok && task != nullptr)
            tasks.emplace_back(sy_task, std::move(task));
        else
            LOG(ERROR) << "[opc] Failed to configure scanner task";
    } else if (err) {
        LOG(ERROR) << "[opc] Failed to list existing tasks: " << err;
        return tasks;
    }
    return tasks;
}
