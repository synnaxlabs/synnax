// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "glog/logging.h"

#include "driver/visa/device/device.h"
#include "driver/visa/api/api.h"
#include "driver/task/task.h"

namespace visa {
/// @brief integration name for VISA.
const std::string INTEGRATION_NAME = "visa";

/// @brief implements the task::Factory to configure and operate VISA tasks.
class Factory final : public task::Factory {
    /// @brief the VISA API wrapper (null if VISA not installed).
    std::shared_ptr<visa_api::API> api;
    /// @brief central device manager that connects and controls access to devices.
    std::shared_ptr<device::Manager> devices;

    /// @brief checks if the VISA API is available and sets an error status if not.
    bool check_health(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) const;

public:
    Factory() {
        // Attempt to load VISA API
        auto [loaded_api, err] = visa_api::API::load();
        if (err) {
            LOG(WARNING) << err;
            api = nullptr;
            devices = nullptr;
        } else {
            api = loaded_api;
            devices = std::make_shared<device::Manager>(api);
        }
    }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;

    std::string name() override { return INTEGRATION_NAME; }
};
}