// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "glog/logging.h"

/// internal
#include "driver/labjack/labjack.h"
#include "driver/labjack/read_task.h"
#include "driver/labjack/scan_task.h"
#include "driver/labjack/write_task.h"

const std::string NO_LIBS_MSG = "Cannot create task because the LJM Libraries are not "
                                "installed on this System.";

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_read(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    const common::TimingConfig timing_cfg
) {
    auto [cfg, err] = labjack::ReadTaskConfig::parse(ctx->client, task, timing_cfg);
    if (err) return {nullptr, err};
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
        xerrors::Error()
    };
}

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_write(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto [cfg, err] = labjack::WriteTaskConfig::parse(ctx->client, task);
    if (err) return {nullptr, err};
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
    const synnax::Task &task
) {
    auto parser = xjson::Parser(task.config);
    auto cfg = labjack::ScanTaskConfig(parser);
    if (parser.error()) return {nullptr, parser.error()};
    auto scan_task = std::make_unique<common::ScanTask>(
        std::make_unique<labjack::Scanner>(task, cfg, devs),
        ctx,
        task,
        breaker::default_config(task.name),
        cfg.rate
    );
    if (cfg.enabled) scan_task->start();
    return {std::move(scan_task), xerrors::NIL};
}

bool labjack::Factory::check_health(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) const {
    if (this->dev_manager != nullptr) return true;
    ctx->set_state(
        {.task = task.key,
         .variant = status::VARIANT_ERROR,
         .details = json{{
             "message",
             NO_LIBS_MSG,
         }}}
    );
    return false;
}

std::pair<std::unique_ptr<task::Task>, bool> labjack::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    if (!this->check_health(ctx, task)) return {nullptr, true};
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res;
    if (task.type == SCAN_TASK_TYPE) res = configure_scan(this->dev_manager, ctx, task);
    if (task.type == READ_TASK_TYPE)
        res = configure_read(this->dev_manager, ctx, task, this->timing_cfg);
    if (task.type == WRITE_TASK_TYPE)
        res = configure_write(this->dev_manager, ctx, task);
    common::handle_config_err(ctx, task, res.second);
    return {std::move(res.first), true};
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
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>> tasks;

    auto [existing, err] = rack.tasks.retrieve_by_type(SCAN_TASK_TYPE);
    if (err.matches(xerrors::NOT_FOUND)) {
        VLOG(1) << "[labjack] Creating scanner task";
        auto sy_task = synnax::Task(
            rack.key,
            "Labjack Scanner",
            SCAN_TASK_TYPE,
            "",
            true
        );
        if (const auto c_err = rack.tasks.create(sy_task)) {
            LOG(ERROR) << "[labjack] Failed to create scanner task: " << c_err;
            return tasks;
        }
        auto [task, handled] = configure_task(ctx, sy_task);
        if (handled)
            if (task != nullptr)
                tasks.emplace_back(sy_task, std::move(task));
            else
                LOG(ERROR) << "[labjack] Failed to configure scanner task";
    } else if (err) {
        LOG(ERROR) << "[labjack] Failed to list existing tasks: " << err;
        return tasks;
    }
    return tasks;
}
