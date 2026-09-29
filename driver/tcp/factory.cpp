// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/tcp/json.gen.h"

#include "driver/bus/task.h"
#include "driver/common/factory.h"
#include "driver/common/status.h"
#include "driver/tcp/client.h"
#include "driver/tcp/scan_task.h"
#include "driver/tcp/tcp.h"

namespace driver::tcp {
namespace {
std::pair<common::ConfigureResult, x::errors::Error> configure_scan(
    const std::shared_ptr<bus::Connections> &connections,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::tcp::ScanConfig::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::ScanTask>(
                std::make_unique<Scanner>(
                    task,
                    Config{.connect_timeout = 1 * x::telem::SECOND},
                    connections
                ),
                ctx,
                task,
                x::breaker::default_config(task.name),
                cfg.rate
            ),
            .auto_start = !cfg.disabled,
        },
        x::errors::NIL,
    };
}
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key
) {
    if (task.type == READ_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            bus::configure_read<
                Client,
                synnax::tcp::Properties,
                synnax::tcp::ReadConfig>(this->connections, ctx, task),
            cmd_key
        );
    if (task.type == WRITE_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            bus::configure_write<
                Client,
                synnax::tcp::Properties,
                synnax::tcp::WriteConfig>(this->connections, ctx, task),
            cmd_key
        );
    if (task.type == SCAN_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_scan(this->connections, ctx, task),
            cmd_key
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
        "TCP Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
}
