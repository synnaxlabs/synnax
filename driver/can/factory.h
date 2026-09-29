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

#include "driver/can/can.h"
#include "driver/can/link.h"
#include "driver/task/task.h"

namespace driver::can {
/// @brief the name of the CAN integration.
const std::string INTEGRATION_NAME = "can";
/// @brief the make of CAN devices.
const std::string MAKE = "CAN";
/// @brief the type of CAN read tasks.
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";
/// @brief the type of CAN write tasks.
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";
/// @brief the type of the task that lists CAN channels.
const std::string SCAN_TASK_TYPE = INTEGRATION_NAME + "_scan";

/// @brief configures CAN read, write, and scan tasks. Read and write tasks on one
/// device share one bus.
class Factory final : public task::Factory {
    const std::shared_ptr<const Backends> backends;
    const std::shared_ptr<Links> links;

public:
    /// @param backends the backends that open CAN channels, keyed by name.
    explicit Factory(Backends backends):
        backends(std::make_shared<const Backends>(std::move(backends))),
        links(std::make_shared<Links>()) {}

    std::string name() override { return INTEGRATION_NAME; }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override;

    /// @brief creates the rack's CAN scan task when it has none.
    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override;
};
}
