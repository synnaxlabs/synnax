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

#include "driver/ethercat/engine/pool.h"
#include "driver/task/common/common.h"
#include "driver/task/task.h"

namespace ethercat {

const std::string INTEGRATION_NAME = "ethercat";
const std::string DEVICE_MAKE = INTEGRATION_NAME;
const std::string SLAVE_DEVICE_MODEL = "slave";
const std::string READ_TASK_TYPE = "ethercat_read";
const std::string WRITE_TASK_TYPE = "ethercat_write";
const std::string SCAN_TASK_TYPE = "ethercat_scan";

/// Creates the default manager for the current platform.
/// Returns IgH manager if IgH masters are configured, otherwise SOEM.
std::unique_ptr<master::Manager> default_manager();

class Factory final : public task::Factory {
    std::shared_ptr<engine::Pool> pool;

    std::pair<common::ConfigureResult, xerrors::Error> configure_read(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) const;

    std::pair<common::ConfigureResult, xerrors::Error> configure_write(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) const;

    std::pair<common::ConfigureResult, xerrors::Error>
    configure_scan(const std::shared_ptr<task::Context> &ctx, const synnax::Task &task);

public:
    /// Constructs a Factory with default manager.
    Factory();

    /// Constructs a Factory with custom manager.
    explicit Factory(std::unique_ptr<master::Manager> manager);

    ~Factory() override = default;

    Factory(const Factory &) = delete;
    Factory &operator=(const Factory &) = delete;

    std::string name() override { return INTEGRATION_NAME; }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;

    /// Checks if a master has an active engine.
    /// @param key The master key (e.g., "igh:0" or "eth0").
    bool is_interface_active(const std::string &key) const;

    /// Returns cached slaves for a master.
    /// @param key The master key.
    std::vector<SlaveInfo> get_cached_slaves(const std::string &key) const;
};

}
