// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/modbus/device/device.h"
#include "driver/task/task.h"

namespace modbus {
/// @brief integration name for modbus.
const std::string INTEGRATION_NAME = "modbus";

/// @brief implements the task::Factory to configure and operate modbus tasks.
class Factory final : public task::Factory {
    /// @brief central device manager that connects and controls access to devices.
    const std::shared_ptr<device::Manager> devices;

public:
    Factory(): devices(std::make_shared<device::Manager>()) {}

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;
};
}
