// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "glog/logging.h"
#include "nlohmann/json.hpp"

#include "x/cpp/xos/xos.h"

#include "driver/ni/daqmx/prod.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/ni/ni.h"
#include "driver/ni/read_task.h"
#include "driver/ni/scan_task.h"
#include "driver/ni/syscfg/prod.h"
#include "driver/ni/write_task.h"
#include "driver/task/common/factory.h"

const std::string
    NO_LIBS_MSG = "Cannot create the task because the NI-DAQmx and "
                  "System Configuration libraries are not installed on this system.";

ni::Factory::Factory(
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
    const common::TimingConfig timing_cfg
):
    dmx(dmx), syscfg(syscfg), timing_cfg(timing_cfg) {}

bool ni::Factory::check_health() const {
    return this->dmx != nullptr && this->syscfg != nullptr;
}

bool ni::Factory::check_health(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) const {
    if (this->check_health()) return true;
    synnax::TaskStatus status{
        .key = task.status_key(),
        .name = task.name,
        .variant = status::variant::ERR,
        .message = NO_LIBS_MSG,
        .details = synnax::TaskStatusDetails{
            .task = task.key,
            .running = false
        },
    };
    ctx->set_status(status);
    return false;
}

std::unique_ptr<ni::Factory> ni::Factory::create(common::TimingConfig timing_cfg) {
    if (xos::get() == xos::MACOS_NAME)
        LOG(WARNING) << "[ni] integration is not supported on macOS";
    auto [syscfg, syscfg_err] = syscfg::ProdAPI::load();
    if (syscfg_err) LOG(WARNING) << syscfg_err;
    auto [dmx, dmx_err] = daqmx::ProdAPI::load();
    if (dmx_err) LOG(WARNING) << dmx_err;
    return std::make_unique<ni::Factory>(
        dmx != nullptr ? std::make_shared<daqmx::SugaredAPI>(dmx) : nullptr,
        syscfg != nullptr ? std::make_shared<syscfg::SugaredAPI>(syscfg) : nullptr,
        timing_cfg
    );
}

std::pair<std::unique_ptr<task::Task>, bool> ni::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    if (!this->check_health(ctx, task)) return {nullptr, true};
    std::pair<common::ConfigureResult, xerrors::Error> res;
    if (task.type == SCAN_TASK_TYPE)
        res = configure_scan(ctx, task);
    else if (task.type == ANALOG_READ_TASK_TYPE)
        res = configure<
            hardware::daqmx::AnalogReader,
            ni::ReadTaskConfig,
            ni::ReadTaskSource<double>,
            common::ReadTask>(ctx, task);
    else if (task.type == DIGITAL_READ_TASK_TYPE)
        res = configure<
            hardware::daqmx::DigitalReader,
            ni::ReadTaskConfig,
            ni::ReadTaskSource<uint8_t>,
            common::ReadTask>(ctx, task);
    else if (task.type == COUNTER_READ_TASK_TYPE)
        res = configure<
            hardware::daqmx::CounterReader,
            ni::ReadTaskConfig,
            ni::ReadTaskSource<double>,
            common::ReadTask>(ctx, task);
    else if (task.type == ANALOG_WRITE_TASK_TYPE)
        res = configure<
            hardware::daqmx::AnalogWriter,
            ni::WriteTaskConfig,
            ni::WriteTaskSink<double>,
            common::WriteTask>(ctx, task);
    else if (task.type == DIGITAL_WRITE_TASK_TYPE)
        res = configure<
            hardware::daqmx::DigitalWriter,
            ni::WriteTaskConfig,
            ni::WriteTaskSink<uint8_t>,
            common::WriteTask>(ctx, task);
    return common::handle_config_err(ctx, task, std::move(res));
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
ni::Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    if (!this->check_health()) return {};
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "NI Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}

std::pair<common::ConfigureResult, xerrors::Error> ni::Factory::configure_scan(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto parser = xjson::Parser(task.config);
    auto cfg = ScanTaskConfig(parser);
    common::ConfigureResult res;
    if (parser.error()) return {std::move(res), parser.error()};
    res.task = std::make_unique<common::ScanTask>(
        std::make_unique<ni::Scanner>(this->syscfg, cfg, task),
        ctx,
        task,
        breaker::default_config(task.name),
        cfg.rate
    );
    res.auto_start = cfg.enabled;
    return {std::move(res), xerrors::NIL};
}
