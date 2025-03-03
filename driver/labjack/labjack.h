// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/task/task.h"
#include "ljm/device_manager.h"

namespace labjack {
const std::string INTEGRATION_NAME = "labjack";
const std::string T4 = "LJM_dtT4";
const std::string T7 = "LJM_dtT7";
const std::string T8 = "LJM_dtT8";

class Factory final : public task::Factory {
    std::shared_ptr<ljm::DeviceManager> device_manager;
public:
    explicit Factory(const std::shared_ptr<ljm::DeviceManager> &device_manager):
        device_manager(device_manager) {
    }

    /// @brief creates a new NI factory, loading the DAQmx and system configuration
    /// libraries.
    static std::unique_ptr<Factory> create();


    /// @brief checks whether the factory is healthy and capable of creating tasks.
    /// If not, the factory will automatically send an error back through the
    /// task state and return false.
    [[nodiscard]] bool check_health(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) const;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;
};
} // namespace labjack
