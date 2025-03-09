// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <vector>

/// external.
#include "glog/logging.h"
#include "nlohmann/json.hpp"

/// internal
#include "driver/ni/daqmx/prod.h"
#include "driver/ni/syscfg/prod.h"
#include "driver/ni/ni.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/ni/write_task.h"
#include "driver/ni/read_task.h"
#include "driver/ni/scan_task.h"

const std::string NO_LIBS_MSG =
        "Cannot create the task because the National Instruments DAQMX and System Configuration libraries are not installed on this system.";

template<typename HardwareT, typename ConfigT, typename SourceSinkT, typename TaskT>
static std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure(
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto [cfg, cfg_err] = ConfigT::parse(ctx->client, task);
    if (cfg_err) return {nullptr, cfg_err};
    TaskHandle handle;
    const std::string dmx_task_name = task.name + " (" + std::to_string(task.key) + ")";
    if (const auto err = dmx->CreateTask(dmx_task_name.c_str(), &handle))
        return {nullptr, err};
    // Very important that we instantiate the Hardware API here, as we pass ownership over the lifecycle of the task
    // handle to it. If we encounter any errors when applying the configuration or cycling the task, we need to make
    // sure it gets cleared.
    auto hw = std::make_unique<HardwareT>(dmx, handle);
    if (const auto err = cfg.apply(dmx, handle)) return {nullptr, err};
    // NI will look for invalid configuration parameters internally, so we quickly
    // cycle the task in order to catch and communicate any errors as soon as possible.
    if (const auto err = hw->start()) return {nullptr, err};
    if (const auto err = hw->stop()) return {nullptr, err};
    return {
        std::make_unique<TaskT>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<SourceSinkT>(std::move(cfg), std::move(hw))
        ),
        xerrors::NIL
    };
}


std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_scan(
    const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto parser = xjson::Parser(task.config);
    auto cfg = ni::ScanTaskConfig(parser);
    if (parser.error()) return {nullptr, parser.error()};
    auto scan_task = std::make_unique<common::ScanTask>(
        std::make_unique<ni::Scanner>(syscfg, cfg, task),
        ctx,
        task,
        breaker::default_config(task.name),
        cfg.rate
    );
    if (cfg.enabled) scan_task->start();
    return {std::move(scan_task), xerrors::NIL,};
}

ni::Factory::Factory(
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const std::shared_ptr<syscfg::SugaredAPI> &syscfg
): dmx(dmx), syscfg(syscfg) {
}

bool ni::Factory::check_health(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) const {
    if (this->dmx != nullptr && this->syscfg != nullptr) return true;
    ctx->set_state({
        .task = task.key,
        .variant = "error",
        .details = json{{"message", NO_LIBS_MSG,}}
    });
    return false;
}

std::unique_ptr<ni::Factory> ni::Factory::create() {
    auto [syscfg, syscfg_err] = syscfg::ProdAPI::load();
    if (syscfg_err)
        LOG(WARNING) << syscfg_err;
    auto [dmx, dmx_err] = daqmx::ProdAPI::load();
    if (dmx_err)
        LOG(WARNING) << dmx_err;
    return std::make_unique<ni::Factory>(
        dmx != nullptr ? std::make_shared<daqmx::SugaredAPI>(dmx) : nullptr,
        syscfg != nullptr ? std::make_shared<syscfg::SugaredAPI>(syscfg) : nullptr
    );
}


std::pair<std::unique_ptr<task::Task>, bool> ni::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    if (!this->check_health(ctx, task)) return {nullptr, true};
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res;
    if (task.type == SCAN_TASK_TYPE)
        res = configure_scan(this->syscfg, ctx, task);
    else if (task.type == ANALOG_READ_TASK_TYPE)
        res = configure<
            hardware::daqmx::AnalogReader,
            ni::ReadTaskConfig,
            ni::ReadTaskSource<double>,
            common::ReadTask
        >(dmx, ctx, task);
    else if (task.type == DIGITAL_READ_TASK_TYPE)
        res = configure<
            hardware::daqmx::DigitalReader,
            ni::ReadTaskConfig,
            ni::ReadTaskSource<uint8_t>,
            common::ReadTask
        >(dmx, ctx, task);
    else if (task.type == ANALOG_WRITE_TASK_TYPE)
        res = configure<
            hardware::daqmx::AnalogWriter,
            ni::WriteTaskConfig,
            ni::WriteTaskSink<double>,
            common::WriteTask
        >(dmx, ctx, task);
    else if (task.type == DIGITAL_WRITE_TASK_TYPE)
        res = configure<
            hardware::daqmx::DigitalWriter,
            ni::WriteTaskConfig,
            ni::WriteTaskSink<uint8_t>,
            common::WriteTask
        >(dmx, ctx, task);
    common::handle_config_err(ctx, task, res.second);
    return {std::move(res.first), true};
}


std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
ni::Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    if (!this->check_health(ctx, synnax::Task())) return {};
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>> tasks;

    auto [existing, err] = rack.tasks.list();
    if (err) {
        LOG(ERROR) << "[ni] failed to list existing tasks: " << err;
        return tasks;
    }

    bool has_scanner = false;
    for (const auto &t: existing)
        if (t.type == SCAN_TASK_TYPE) has_scanner = true;
    if (has_scanner) return tasks;

    auto sy_task = synnax::Task(rack.key, "ni scanner", SCAN_TASK_TYPE, "", true);
    const auto c_err = rack.tasks.create(sy_task);
    if (c_err) {
        LOG(ERROR) << "[ni] failed to create scanner task: " << c_err;
        return tasks;
    }
    auto [task, ok] = configure_task(ctx, sy_task);
    if (!ok) {
        LOG(ERROR) << "[ni] failed to configure scanner task: " << c_err;
        return tasks;
    }
    tasks.emplace_back(
        std::pair<synnax::Task, std::unique_ptr<task::Task>>({
            sy_task,
            std::move(task)
        })
    );
    return tasks;
}
