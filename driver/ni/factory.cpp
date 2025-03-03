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

template<typename Hardware, typename ConfigType, typename SourceType, typename TaskType>
static std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure(
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto [cfg, cfg_err] = ConfigType::parse(ctx->client, task);
    if (cfg_err) return {nullptr, cfg_err};
    TaskHandle handle;
    const std::string dmx_task_name = task.name + " (" + std::to_string(task.key) + ")";
    if (const auto err = dmx->CreateTask(dmx_task_name.c_str(), &handle))
        return {
            nullptr, err
        };
    // Very important that we instantiate the Hardware API here, as we pass ownership over the lifecycle of the task
    // handle to it. If we encounter any errors when applying the configuration or cycling the task, we need to make
    // sure it gets cleared.
    auto hw = std::make_unique<Hardware>(dmx, handle);
    if (const auto err = cfg.apply(dmx, handle)) return {nullptr, err};
    // NI will look for invalid configuration parameters internally, so we quickly
    // cycle the task in order to catch and communicate any errors as soon as possible.
    if (const auto err = hw->start()) return {nullptr, err};
    if (const auto err = hw->stop()) return {nullptr, err};
    return {
        std::make_unique<TaskType>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_shared<SourceType>(cfg)
        ),
        xerrors::NIL
    };
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

using configure_ni_analog_read = configure<
    hardware::daqmx::AnalogReader,
    ni::ReadTaskConfig,
    ni::ReadTaskSource<double>,
    common::ReadTask
>;

using configure_ni_digital_read = configure<
    hardware::daqmx::DigitalReader,
    ni::ReadTaskConfig,
    ni::CommandTaskSink<uint8_t>,
    common::ReadTask
>;

using configure_ni_analog_write = configure<
    hardware::daqmx::AnalogWriter,
    ni::WriteTaskConfig,
    ni::CommandTaskSink<double>,
    common::WriteTask
>;

using configure_ni_digital_write = configure<
    hardware::daqmx::DigitalWriter,
    ni::WriteTaskConfig,
    ni::CommandTaskSink<uint8_t>,
    common::WriteTask
>;


std::pair<std::unique_ptr<task::Task>, bool> ni::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (!this->check_health(ctx, task)) return {nullptr, false};
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res;
    if (task.type == "ni_scanner")
        res = ni::ScanTask::configure(this->syscfg, ctx, task);
    else if (task.type == "ni_analog_read")
        res = configure_ni_analog_read(dmx, ctx, task);
    else if (task.type == "ni_digital_read")
        res = configure_ni_digital_read(dmx, ctx, task);
    else if (task.type == "ni_analog_write")
        res = configure_ni_analog_write(dmx, ctx, task);
    else if (task.type == "ni_digital_write")
        res = configure_ni_digital_write(dmx, ctx, task);
    else return {nullptr, false};
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
        if (t.type == "ni_scanner") has_scanner = true;

    if (has_scanner) return tasks;

    auto sy_task = synnax::Task(rack.key, "ni scanner", "ni_scanner", "", true);
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
