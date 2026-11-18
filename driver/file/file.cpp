// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/json/json.h"

#include "driver/common/factory.h"
#include "driver/file/file.h"

namespace driver::file {
Task::Task(
    std::shared_ptr<task::Context> ctx,
    synnax::task::Task task,
    Browser browser
):
    ctx(std::move(ctx)), task(std::move(task)), browser(std::move(browser)) {
    this->key = this->task.key;
}

void Task::exec(synnax::task::Command &cmd, x::breaker::Breaker &) {
    synnax::task::Status status{
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details =
            synnax::task::StatusDetails{
                .running = true,
                .cmd = cmd.key,
                .config_hash = this->task.config_hash,
            },
        .owner = synnax::task::resource_id(this->task.key),
    };
    if (cmd.type != LIST_CMD_TYPE) {
        status.message = "Failed to execute command";
        status.description = "The task does not accept the '" + cmd.type + "' command.";
        this->ctx->set_status(status);
        return;
    }
    x::json::Parser parser(cmd.args);
    const auto path = parser.field<std::string>("path", "");
    if (parser.error()) {
        status.message = "Failed to list files";
        status.description = "The path must be a string.";
        this->ctx->set_status(status);
        return;
    }
    auto [data, err] = this->browser.list(path);
    if (err) {
        status.message = "Failed to list files";
        status.description = err.data;
        this->ctx->set_status(status);
        return;
    }
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = "Listed files";
    status.details.data = data.get<x::json::json::object_t>();
    this->ctx->set_status(status);
}

Factory::Factory(Config cfg): cfg(std::move(cfg)) {}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &,
    x::breaker::Breaker &
) {
    if (task.type != TASK_TYPE) return {nullptr, false};
    return {std::make_unique<Task>(ctx, task, Browser(this->cfg)), true};
}

std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::rack::Rack &rack,
    x::breaker::Breaker &cancel
) {
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        TASK_NAME,
        TASK_TYPE,
        INTEGRATION_NAME,
        cancel
    );
}
}
