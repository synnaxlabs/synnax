// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/http/device/device.h"
#include "driver/task/task.h"

namespace driver::http {
/// @brief integration name for http.
const std::string INTEGRATION_NAME = "http";

/// @brief implements the task::Factory to configure and operate HTTP tasks.
class Factory final : public task::Factory {
    /// @brief central device manager that pools HTTP client connections.
    const std::shared_ptr<device::Manager> devices;

public:
    Factory(): devices(std::make_shared<device::Manager>()) {}

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
};
}
