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

#include "client/cpp/device/device.h"
#include "client/cpp/serial/types.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"

#include "driver/task/task.h"
#include "driver/transport/connection.h"

namespace driver::serial {
/// @brief INTEGRATION_NAME is the name of the serial integration.
const std::string INTEGRATION_NAME = "serial";

/// @brief retrieves the properties of the serial device with key.
/// @returns the error of the retrieve, or VALIDATION when the properties are invalid.
std::pair<synnax::serial::Properties, x::errors::Error>
retrieve_properties(const synnax::device::Client &devices, const std::string &key);

/// @brief acquires the connection to the port of the device with key, and opens the
/// port when no task on the device holds it open. A port that is missing or in use
/// does not fail, since each turn on the connection opens it again.
/// @returns transport::CONFIG_ERROR when another task holds the device with other
/// properties or the port cannot apply props, and transport::CRITICAL_ERROR when
/// reopening cannot fix the failure.
std::pair<std::shared_ptr<transport::Connection>, x::errors::Error> acquire(
    transport::Connections &connections,
    const std::string &key,
    const synnax::serial::Properties &props
);

/// @brief Factory configures the tasks that read and send ICD messages on serial ports.
/// The tasks on one device share its port.
class Factory final : public task::Factory {
    std::shared_ptr<transport::Connections>
        connections = std::make_shared<transport::Connections>();

public:
    std::string name() override { return INTEGRATION_NAME; }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key,
        x::breaker::Breaker &cancel
    ) override;
};
}
