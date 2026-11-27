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

#include "client/cpp/serial/json.gen.h"
#include "x/cpp/json/json.h"

#include "driver/bus/tasks.h"
#include "driver/common/status.h"
#include "driver/serial/serial.h"

namespace driver::serial {
namespace {
/// @returns the bus::Acquire of the port of dev, which ports shares.
bus::Acquire acquire(const std::shared_ptr<Ports> &ports, const Device &dev) {
    return [ports, dev] { return ports->acquire(dev); };
}

/// @brief configures a serial read task.
/// @returns VALIDATION when the config names no device or no entry, the error of the
/// retrieve of the device, and the error of bus::configure_read.
std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<Ports> &ports,
    bus::Readers &readers
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::serial::ReadConfig::parse(parser);
    if (cfg.device.empty()) parser.field_err("device", "this field is required");
    if (cfg.entries.empty())
        parser.field_err("entries", "at least one entry is required");
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [dev, dev_err] = retrieve_device(ctx->client->devices, cfg.device);
    if (dev_err) return {common::ConfigureResult{}, dev_err};
    return bus::configure_read(
        ctx,
        task,
        cfg,
        cfg.entries,
        dev.key,
        acquire(ports, dev),
        readers
    );
}

/// @brief configures a serial write task.
/// @returns VALIDATION when the config names no device or no target, the error of the
/// retrieve of the device, and the error of bus::configure_write.
std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<Ports> &ports
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::serial::WriteConfig::parse(parser);
    if (cfg.device.empty()) parser.field_err("device", "this field is required");
    if (cfg.targets.empty())
        parser.field_err("targets", "at least one target is required");
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [dev, dev_err] = retrieve_device(ctx->client->devices, cfg.device);
    if (dev_err) return {common::ConfigureResult{}, dev_err};
    return bus::configure_write(ctx, task, cfg, cfg.targets, acquire(ports, dev));
}
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key,
    x::breaker::Breaker &cancel
) {
    if (task.type == READ_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_read(ctx, task, this->ports, this->readers),
            cmd_key,
            cancel
        );
    if (task.type == WRITE_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_write(ctx, task, this->ports),
            cmd_key,
            cancel
        );
    return {nullptr, false};
}
}
