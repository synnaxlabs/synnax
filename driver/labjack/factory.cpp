// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "driver/common/factory.h"
#include "driver/labjack/labjack.h"
#include "driver/labjack/read_task.h"
#include "driver/labjack/scan_task.h"
#include "driver/labjack/write_task.h"

namespace driver::labjack {
std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const common::TimingConfig timing_cfg
) {
    common::ConfigureResult result;
    auto [cfg, err] = ReadTaskConfig::parse(ctx->client, task, timing_cfg);
    if (err) return {std::move(result), err};
    result.auto_start = cfg.auto_start;
    std::unique_ptr<common::Source> source;
    if (cfg.has_thermocouples())
        source = std::make_unique<UnarySource>(devs, std::move(cfg));
    else
        source = std::make_unique<StreamSource>(devs, std::move(cfg));
    result.task = std::make_unique<common::ReadTask>(
        task,
        ctx,
        x::breaker::default_config(task.name),
        std::move(source)
    );
    return {std::move(result), x::errors::NIL};
}

std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    common::ConfigureResult result;
    auto [cfg, err] = WriteTaskConfig::parse(ctx->client, task);
    if (err) return {std::move(result), err};
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<common::WriteTask>(
        task,
        ctx,
        x::breaker::default_config(task.name),
        std::make_unique<WriteSink>(devs, std::move(cfg))
    );
    return {std::move(result), x::errors::NIL};
}

std::pair<common::ConfigureResult, x::errors::Error> configure_scan(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    auto parser = x::json::Parser(task.config);
    auto cfg = ScanTaskConfig(parser);
    common::ConfigureResult result;
    if (parser.error()) return {std::move(result), parser.error()};
    result.task = std::make_unique<common::ScanTask>(
        std::make_unique<Scanner>(task, cfg, devs),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.rate
    );
    result.auto_start = !cfg.disabled;
    return {std::move(result), x::errors::NIL};
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key
) {
    if (!task.type.starts_with(INTEGRATION_NAME)) return {nullptr, false};
    std::pair<common::ConfigureResult, x::errors::Error> res;
    if (task.type == SCAN_TASK_TYPE) res = configure_scan(this->dev_manager, ctx, task);
    if (task.type == READ_TASK_TYPE)
        res = configure_read(this->dev_manager, ctx, task, this->timing_cfg);
    if (task.type == WRITE_TASK_TYPE)
        res = configure_write(this->dev_manager, ctx, task);
    return common::handle_config_err(ctx, task, std::move(res), cmd_key);
}

std::unique_ptr<Factory> Factory::create(common::TimingConfig timing_cfg) {
    auto [ljm, ljm_err] = ljm::API::load();
    if (ljm_err) LOG(WARNING) << ljm_err;
    return std::make_unique<Factory>(
        ljm != nullptr ? std::make_shared<device::LJMManager>(ljm) : nullptr,
        timing_cfg
    );
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
        "LabJack Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
}
