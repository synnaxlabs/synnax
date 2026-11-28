// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>

#include "client/cpp/labjack/device/types.gen.h"
#include "x/cpp/lib/lazy.h"

#include "driver/common/sample_clock.h"
#include "driver/labjack/device/device.h"
#include "driver/labjack/names.h"
#include "driver/task/task.h"

namespace driver::labjack {

/// @brief T4 model name.
const std::string T4 = synnax::labjack::device::MODEL_T_4;
/// @brief T7 model name.
const std::string T7 = synnax::labjack::device::MODEL_T_7;
/// @brief T8 model name.
const std::string T8 = synnax::labjack::device::MODEL_T_8;

/// @brief factory for creating and operating LabJack tasks.
class Factory final : public task::Factory {
    std::shared_ptr<x::lib::Lazy<device::Manager>> dev_manager;
    common::TimingConfig timing_cfg;

public:
    explicit Factory(
        x::lib::Lazy<device::Manager>::Loader load_dev_manager,
        const common::TimingConfig timing_cfg = common::TimingConfig()
    ):
        dev_manager(
            std::make_shared<x::lib::Lazy<device::Manager>>(std::move(load_dev_manager))
        ),
        timing_cfg(timing_cfg) {}

    /// @brief creates a new LabJack factory that loads the LJM library on first use.
    static std::unique_ptr<Factory>
    create(common::TimingConfig timing_cfg = common::TimingConfig());

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key,
        x::breaker::Breaker &cancel
    ) override;

    std::string name() override { return INTEGRATION_NAME; }

    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack,
        x::breaker::Breaker &cancel
    ) override;
};
}
