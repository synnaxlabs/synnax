// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/labjack/device/device.h"
#include "driver/task/task.h"

namespace labjack {
/// @brief labjack integration name.
const std::string INTEGRATION_NAME = "labjack";
/// @brief T4 model name.
const std::string T4 = "LJM_dtT4";
/// @brief T7 model name.
const std::string T7 = "LJM_dtT7";
/// @brief T8 model name.
const std::string T8 = "LJM_dtT8";
const std::string SCAN_TASK_TYPE = "labjack_scan";
const std::string READ_TASK_TYPE = "labjack_read";
const std::string WRITE_TASK_TYPE = "labjack_write";

/// @brief LJM errors that indicate the device is currently unreachable but may be
/// reachable again in the near future.
const std::vector UNREACHABLE_ERRORS = {
    ljm::NO_RESPONSE_BYTES_RECEIVED,
    ljm::STREAM_NOT_INITIALIZED,
    ljm::RECONNECT_FAILED,
    ljm::SYNCHRONIZATION_TIMEOUT
};

/// @brief translates LJM errors into useful errors for managing the task lifecycle.
inline xerrors::Error translate_error(const xerrors::Error &err) {
    if (err.matches(UNREACHABLE_ERRORS))
        return ljm::TEMPORARILY_UNREACHABLE;
    return err;
}

/// @brief factory for creating and operating labjack tasks.
class Factory final : public task::Factory {
    std::shared_ptr<device::Manager> dev_manager;

    /// @brief checks whether the factory is healthy and capable of creating tasks.
    /// If not, the factory will automatically send an error back through the
    /// task state and return false.
    [[nodiscard]] bool check_health(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) const;

public:
    explicit Factory(const std::shared_ptr<device::Manager> &dev_manager):
        dev_manager(dev_manager) {
    }

    /// @brief creates a new Labjack factory, loading the LJM library.
    static std::unique_ptr<Factory> create();

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
