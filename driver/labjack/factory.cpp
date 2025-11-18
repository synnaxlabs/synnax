// Copyright 2025 Synnax Labs, Inc.
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

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_read(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    const common::TimingConfig timing_cfg,
    bool &auto_start
) {
    auto [cfg, err] = labjack::ReadTaskConfig::parse(ctx->client, task, timing_cfg);
    if (err) return {nullptr, err};
    auto_start = cfg.auto_start;
    auto [dev, d_err] = devs->acquire(cfg.device_key);
    if (d_err) return {nullptr, d_err};
    std::unique_ptr<common::Source> source;
    if (cfg.has_thermocouples())
        source = std::make_unique<labjack::UnarySource>(dev, std::move(cfg));
    else
        source = std::make_unique<labjack::StreamSource>(dev, std::move(cfg));
    return {
        std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::move(source)
        ),
        xerrors::NIL
    };
}

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_write(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    bool &auto_start
) {
    auto [cfg, err] = labjack::WriteTaskConfig::parse(ctx->client, task);
    if (err) return {nullptr, err};
    auto_start = cfg.auto_start;
    auto [dev, d_err] = devs->acquire(cfg.device_key);
    if (d_err) return {nullptr, d_err};
    return {
        std::make_unique<common::WriteTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<labjack::WriteSink>(dev, std::move(cfg))
        ),
        xerrors::NIL
    };
}

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_scan(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    bool &auto_start
) {
    auto parser = xjson::Parser(task.config);
    auto cfg = labjack::ScanTaskConfig(parser);
    if (parser.error()) return {nullptr, parser.error()};
    auto_start = cfg.enabled;
    return {
        std::make_unique<common::ScanTask>(
            std::make_unique<labjack::Scanner>(task, cfg, devs),
            ctx,
            task,
            breaker::default_config(task.name),
            cfg.rate
        ),
        xerrors::NIL
    };
}

bool labjack::Factory::check_health(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) const {
    if (this->dev_manager != nullptr) return true;
    ctx->set_status(
        {.variant = status::variant::ERR,
         .message = NO_LIBS_MSG,
         .details = synnax::TaskStatusDetails{
             .task = task.key,
         }}
    );
    return false;
}

std::pair<std::unique_ptr<task::Task>, bool> labjack::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    if (!this->check_health(ctx, task)) return {nullptr, true};
    bool auto_start = false;
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res = {
        nullptr,
        xerrors::NIL
    };
    if (task.type == SCAN_TASK_TYPE)
        res = configure_scan(this->dev_manager, ctx, task, auto_start);
    if (task.type == READ_TASK_TYPE)
        res = configure_read(
            this->dev_manager,
            ctx,
            task,
            this->timing_cfg,
            auto_start
        );
    if (task.type == WRITE_TASK_TYPE)
        res = configure_write(this->dev_manager, ctx, task, auto_start);
    return common::handle_config_err(
        ctx,
        task,
        std::move(res.first),
        res.second,
        auto_start
    );
}

std::unique_ptr<labjack::Factory>
labjack::Factory::create(common::TimingConfig timing_cfg) {
    auto [ljm, ljm_err] = ljm::API::load();
    if (ljm_err) LOG(WARNING) << ljm_err;
    return std::make_unique<labjack::Factory>(
        ljm != nullptr ? std::make_shared<device::Manager>(ljm) : nullptr,
        timing_cfg
    );
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
labjack::Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    if (!this->check_health(ctx, synnax::Task())) return {};
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "LabJack Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
