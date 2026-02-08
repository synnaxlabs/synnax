// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <vector>

#include "driver/common/common.h"
#include "driver/ethercat/engine/pool.h"
#include "driver/task/task.h"

namespace driver::ethercat {

/// @brief integration name for EtherCAT.
const std::string INTEGRATION_NAME = "ethercat";
/// @brief device make identifier.
const std::string DEVICE_MAKE = INTEGRATION_NAME;
/// @brief device model for slave devices.
const std::string SLAVE_DEVICE_MODEL = "slave";
/// @brief task type for read tasks.
const std::string READ_TASK_TYPE = "ethercat_read";
/// @brief task type for write tasks.
const std::string WRITE_TASK_TYPE = "ethercat_write";
/// @brief task type for scan tasks.
const std::string SCAN_TASK_TYPE = "ethercat_scan";

/// @brief creates the default manager for the current platform.
std::unique_ptr<master::Manager> default_manager();

class Factory final : public task::Factory {
    std::shared_ptr<engine::Pool> pool;

    std::pair<common::ConfigureResult, x::errors::Error> configure_read(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) const;

    std::pair<common::ConfigureResult, x::errors::Error> configure_write(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) const;

    std::pair<common::ConfigureResult, x::errors::Error> configure_scan(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    );

public:
    /// @brief constructs a Factory with default manager.
    Factory();

    /// @brief constructs a Factory with custom manager.
    explicit Factory(std::unique_ptr<master::Manager> manager);

    ~Factory() override = default;

    Factory(const Factory &) = delete;
    Factory &operator=(const Factory &) = delete;

    std::string name() override { return INTEGRATION_NAME; }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) override;

    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override;

    /// @brief checks if a master has an active engine.
    bool is_interface_active(const std::string &key) const;

    /// @brief returns cached slaves for a master.
    std::vector<slave::Properties> get_cached_slaves(const std::string &key) const;
};

}
