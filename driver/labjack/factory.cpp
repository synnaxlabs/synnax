// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#include "labjack.h"
#include "glog/logging.h"
#include "driver/labjack/scan_task.h"
#include "driver/labjack/read_task.h"
#include "driver/labjack/write_task.h"

const std::string NO_LIBS_MSG =
        "Cannot create task because the LJM Libraries are not installed on this System.";

std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_read(
    const std::shared_ptr<device::Manager> &devs,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto [cfg, err] = labjack::ReadTaskConfig::parse(ctx->client, task);
    if (err) return {nullptr, err};
    auto [dev, d_err] = devs->acquire(cfg.device_key);
    if (d_err) return {nullptr, d_err};
    std::unique_ptr<common::Source> source;
    if (cfg.has_tcs())
        source = std::make_unique<labjack::UnarySource>(
            dev, std::move(cfg));
    else source = std::make_unique<labjack::StreamSource>(dev, std::move(cfg));
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

bool labjack::Factory::check_health(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) const {
    if (this->dev_manager != nullptr) return true;
    ctx->set_state({
        .task = task.key,
        .variant = "error",
        .details = json{{"message", NO_LIBS_MSG,}}
    });
    return false;
}

std::pair<std::unique_ptr<task::Task>, bool> labjack::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    if (!this->check_health(ctx, task)) return {nullptr, false};
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res;
    if (task.type == "labjack_scan")
        res = labjack::ScanTask::configure(ctx, task, this->dev_manager);
    if (task.type == "labjack_read")
        res = configure_read(this->dev_manager, ctx, task);
    if (task.type == "labjack_write")
        res = configure_write(this->dev_manager, ctx, task);
    auto [tsk, err] = std::move(res);
    if (err)
        ctx->set_state({
            .task = task.key,
            .variant = "error",
            .details = json{
                {"message", err.message()}
            }
        });
    else
        ctx->set_state({
            .task = task.key,
            .variant = "success",
            .details = json{
                {"message", "Task configured successfully"},
            }
        });
    return {std::move(tsk), true};
}

std::unique_ptr<labjack::Factory> labjack::Factory::create() {
    auto [ljm, ljm_err] = ljm::API::load();
    if (ljm_err)
        LOG(WARNING) << ljm_err;
    return std::make_unique<labjack::Factory>(
        ljm != nullptr ? std::make_shared<device::Manager>(ljm) : nullptr
    );
}


std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
labjack::Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>> tasks;

    auto [existing, err] = rack.tasks.retrieveByType("labjack_scan");
    if (err.matches(xerrors::NOT_FOUND)) {
        VLOG(1) << "[labjack] Creating scanner task";
        auto sy_task = synnax::Task(
            rack.key,
            "labjack scanner",
            "labjack_scan",
            "",
            true
        );
        if (const auto c_err = rack.tasks.create(sy_task)) {
            LOG(ERROR) << "[labjack] Failed to create scanner task: " << c_err;
            return tasks;
        }
        auto [task, ok] = configure_task(ctx, sy_task);
        if (ok && task != nullptr)
            tasks.emplace_back(sy_task, std::move(task));
        else
            LOG(ERROR) << "[labjack] Failed to configure scanner task";
    } else if (err) {
        LOG(ERROR) << "[labjack] Failed to list existing tasks: " << err;
        return tasks;
    }
    return tasks;
}
