// Copyright 2026 Synnax Labs, Inc.
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

#include "x/cpp/os/os.h"

#include "driver/common/factory.h"
#include "driver/ni/daqmx/prod.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/ni/ni.h"
#include "driver/ni/read_task.h"
#include "driver/ni/scan_task.h"
#include "driver/ni/syscfg/prod.h"
#include "driver/ni/write_task.h"

namespace driver::ni {
const std::string
    NO_LIBS_MSG = "Cannot create the task because the NI-DAQmx and "
                  "System Configuration libraries are not installed on this system.";

Factory::Factory(
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
    const driver::task::common::TimingConfig timing_cfg
):
    dmx(dmx), syscfg(syscfg), timing_cfg(timing_cfg) {}

bool Factory::check_health() const {
    return this->dmx != nullptr && this->syscfg != nullptr;
}

bool Factory::check_health(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) const {
    if (this->check_health()) return true;
    synnax::task::Status status{
        .key = task.status_key(),
        .name = task.name,
        .variant = x::status::VARIANT_ERROR,
        .message = NO_LIBS_MSG,
        .details = synnax::task::StatusDetails{.task = task.key, .running = false},
    };
    ctx->set_status(status);
    return false;
}

std::unique_ptr<Factory> Factory::create(common::TimingConfig timing_cfg) {
    if (x::os::get() == x::os::MACOS_NAME)
        LOG(WARNING) << "[ni] integration is not supported on macOS";
    auto [syscfg, syscfg_err] = syscfg::ProdAPI::load();
    if (syscfg_err) LOG(WARNING) << syscfg_err;
    auto [dmx, dmx_err] = daqmx::ProdAPI::load();
    if (dmx_err) LOG(WARNING) << dmx_err;
    return std::make_unique<Factory>(
        dmx != nullptr ? std::make_shared<daqmx::SugaredAPI>(dmx) : nullptr,
        syscfg != nullptr ? std::make_shared<syscfg::SugaredAPI>(syscfg) : nullptr,
        timing_cfg
    );
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    if (!this->check_health(ctx, task)) return {nullptr, true};
    std::pair<common::ConfigureResult, x::errors::Error> res;
    if (task.type == SCAN_TASK_TYPE)
        res = configure_scan(ctx, task);
    else if (task.type == ANALOG_READ_TASK_TYPE)
        res = configure<
            hardware::daqmx::AnalogReader,
            ReadTaskConfig,
            ReadTaskSource<double>,
            common::ReadTask>(ctx, task);
    else if (task.type == DIGITAL_READ_TASK_TYPE)
        res = configure<
            hardware::daqmx::DigitalReader,
            ReadTaskConfig,
            ReadTaskSource<uint8_t>,
            common::ReadTask>(ctx, task);
    else if (task.type == COUNTER_READ_TASK_TYPE)
        res = configure<
            hardware::daqmx::CounterReader,
            ReadTaskConfig,
            ReadTaskSource<double>,
            common::ReadTask>(ctx, task);
    else if (task.type == ANALOG_WRITE_TASK_TYPE)
        res = configure<
            hardware::daqmx::AnalogWriter,
            WriteTaskConfig,
            WriteTaskSink<double>,
            common::WriteTask>(ctx, task);
    else if (task.type == DIGITAL_WRITE_TASK_TYPE)
        res = configure<
            hardware::daqmx::DigitalWriter,
            WriteTaskConfig,
            WriteTaskSink<uint8_t>,
            common::WriteTask>(ctx, task);
    return common::handle_config_err(ctx, task, std::move(res));
}

std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::rack::Rack &rack
) {
    return driver::task::common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "NI Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}

std::pair<common::ConfigureResult, x::errors::Error> Factory::configure_scan(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    auto parser = x::json::Parser(task.config);
    auto cfg = ScanTaskConfig(parser);
    driver::task::common::ConfigureResult res;
    if (parser.error()) return {std::move(res), parser.error()};
    res.task = std::make_unique<common::ScanTask>(
        std::make_unique<Scanner>(this->syscfg, cfg, task),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.scan_rate
    );
    res.auto_start = cfg.enabled;
    return {std::move(res), x::errors::NIL};
}
}
