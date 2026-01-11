// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"

#include "driver/labjack/labjack.h"
#include "driver/labjack/read_task.h"
#include "driver/labjack/scan_task.h"
#include "driver/labjack/write_task.h"
#include "driver/task/common/factory.h"

const std::string NO_LIBS_MSG = "Cannot create task because the LJM Libraries are not "
                                "installed on this System.";

std::pair<driver::task::common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<driver::task::Context> &ctx,
    const synnax::task::Task &task,
    const driver::task::common::TimingConfig timing_cfg
) {
    driver::task::common::ConfigureResult result;
    auto [cfg, err] = driver::labjack::ReadTaskConfig::parse(ctx->client, task, timing_cfg);
    if (err) return {std::move(result), err};
    auto [dev, d_err] = devs->acquire(cfg.device_key);
    if (d_err) return {std::move(result), d_err};
    std::unique_ptr<driver::task::common::Source> source;
    if (cfg.has_thermocouples())
        source = std::make_unique<driver::labjack::UnarySource>(dev, std::move(cfg));
    else
        source = std::make_unique<driver::labjack::StreamSource>(dev, std::move(cfg));
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<driver::task::common::ReadTask>(
        task,
        ctx,
        x::breaker::default_config(task.name),
        std::move(source)
    );
    return {std::move(result), x::errors::NIL};
}

std::pair<driver::task::common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<driver::task::Context> &ctx,
    const synnax::task::Task &task
) {
    driver::task::common::ConfigureResult result;
    auto [cfg, err] = driver::labjack::WriteTaskConfig::parse(ctx->client, task);
    if (err) return {std::move(result), err};
    auto [dev, d_err] = devs->acquire(cfg.device_key);
    if (d_err) return {std::move(result), d_err};
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<driver::task::common::WriteTask>(
        task,
        ctx,
        x::breaker::default_config(task.name),
        std::make_unique<driver::labjack::WriteSink>(dev, std::move(cfg))
    );
    return {std::move(result), x::errors::NIL};
}

std::pair<driver::task::common::ConfigureResult, x::errors::Error> configure_scan(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<driver::task::Context> &ctx,
    const synnax::task::Task &task
) {
    auto parser = x::json::Parser(task.config);
    auto cfg = driver::labjack::ScanTaskConfig(parser);
    driver::task::common::ConfigureResult result;
    if (parser.error()) return {std::move(result), parser.error()};
    result.task = std::make_unique<driver::task::common::ScanTask>(
        std::make_unique<driver::labjack::Scanner>(task, cfg, devs),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.scan_rate
    );
    result.auto_start = cfg.enabled;
    return {std::move(result), x::errors::NIL};
}

bool driver::labjack::Factory::check_health(
    const std::shared_ptr<driver::task::Context> &ctx,
    const synnax::task::Task &task
) const {
    if (this->dev_manager != nullptr) return true;
    synnax::task::Status status{
        .key = synnax::task::status_key(task),
        .name = task.name,
        .variant = x::status::VARIANT_ERROR,
        .message = NO_LIBS_MSG,
        .details = synnax::task::StatusDetails{.task = task.key}
    };
    ctx->set_status(status);
    return false;
}

std::pair<std::unique_ptr<driver::task::Task>, bool> driver::labjack::Factory::configure_task(
    const std::shared_ptr<driver::task::Context> &ctx,
    const synnax::task::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    if (!this->check_health(ctx, task)) return {nullptr, true};
    std::pair<driver::task::common::ConfigureResult, x::errors::Error> res;
    if (task.type == SCAN_TASK_TYPE) res = configure_scan(this->dev_manager, ctx, task);
    if (task.type == READ_TASK_TYPE)
        res = configure_read(this->dev_manager, ctx, task, this->timing_cfg);
    if (task.type == WRITE_TASK_TYPE)
        res = configure_write(this->dev_manager, ctx, task);
    return driver::task::common::handle_config_err(ctx, task, std::move(res));
}

std::unique_ptr<driver::labjack::Factory>
driver::labjack::Factory::create(driver::task::common::TimingConfig timing_cfg) {
    auto [ljm, ljm_err] = ljm::API::load();
    if (ljm_err) LOG(WARNING) << ljm_err;
    return std::make_unique<driver::labjack::Factory>(
        ljm != nullptr ? std::make_shared<device::Manager>(ljm) : nullptr,
        timing_cfg
    );
}

std::vector<std::pair<synnax::task::Task, std::unique_ptr<driver::task::Task>>>
driver::labjack::Factory::configure_initial_tasks(
    const std::shared_ptr<driver::task::Context> &ctx,
    const synnax::rack::Rack &rack
) {
    return driver::task::common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "LabJack Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
