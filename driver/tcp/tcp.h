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
#include <utility>
#include <vector>

#include "driver/bus/connection.h"
#include "driver/task/task.h"

namespace driver::tcp {
/// @brief the name of the TCP integration.
const std::string INTEGRATION_NAME = "tcp";
/// @brief the make of TCP devices.
const std::string MAKE = "TCP";
/// @brief the type of TCP read tasks.
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";
/// @brief the type of TCP write tasks.
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";
/// @brief the type of the task that checks each TCP device is reachable.
const std::string SCAN_TASK_TYPE = INTEGRATION_NAME + "_scan";

/// @brief configures TCP read, write, and scan tasks.
class Factory final : public task::Factory {
    /// @brief the device connections that the integration's tasks share.
    const std::shared_ptr<bus::Connections> connections;

public:
    Factory(): connections(std::make_shared<bus::Connections>()) {}

    std::string name() override { return INTEGRATION_NAME; }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override;

    /// @brief creates the rack's TCP scan task when it has none.
    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override;
};
}
