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
#include "driver/ni/daqmx/daqmx_prod.h"
#include "driver/ni/syscfg/syscfg_prod.h"
#include "driver/ni/ni.h"
#include "driver/ni/hardware.h"
#include "driver/ni/write_task.h"
#include "driver/ni/read_task.h"
#include "driver/ni/scan_task.h"

const std::string NO_LIBS_MSG =
        "Cannot create the task because the National Instruments DAQMX and System Configuration libraries are not installed on this system.";

ni::Factory::Factory(
    const std::shared_ptr<SugaredDAQmx> &dmx,
    const std::shared_ptr<SugaredSysCfg> &syscfg
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
    auto [syscfg, syscfg_err] = SysCfgProd::load();
    if (syscfg_err)
        LOG(WARNING) << syscfg_err;
    auto [dmx, dmx_err] = DAQmxProd::load();
    if (dmx_err)
        LOG(WARNING) << dmx_err;
    return std::make_unique<ni::Factory>(
        dmx != nullptr ? std::make_shared<SugaredDAQmx>(dmx) : nullptr,
        syscfg != nullptr ? std::make_shared<SugaredSysCfg>(syscfg) : nullptr
    );
}

std::pair<std::unique_ptr<task::Task>, bool> ni::Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (!this->check_health(ctx, task)) return {nullptr, false};
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> res;
    if (task.type == "ni_scanner")
        res = ni::ScanTask::configure(this->syscfg, ctx, task);
    else if (task.type == "ni_analog_read")
        res = configure<AnalogHardwareReader, ReadTask<double>, ReadTaskConfig>(dmx, ctx, task);
    else if (task.type == "ni_digital_read")
        res = configure<DigitalHardwareReader, ReadTask<uint8_t>, ReadTaskConfig>(dmx, ctx, task);
    else if (task.type == "ni_analog_write")
        res = configure<AnalogHardwareWriter, WriteTask<double>, WriteTaskConfig>(dmx, ctx, task);
    else if (task.type == "ni_digital_write")
        res = configure<DigitalHardwareWriter, WriteTask<uint8_t>, WriteTaskConfig>(dmx, ctx, task);
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


std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
ni::Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    if (!this->check_health(ctx, synnax::Task())) return {};
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;

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
        std::pair<synnax::Task, std::unique_ptr<task::Task> >({
            sy_task,
            std::move(task)
        })
    );
    return tasks;
}
