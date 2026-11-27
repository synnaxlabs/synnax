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

#include "client/cpp/tcp/types.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/device.h"
#include "driver/bus/holders.h"
#include "driver/task/task.h"
#include "driver/transport/connection.h"

namespace driver::tcp {
/// @brief INTEGRATION_NAME is the name of the TCP integration.
const std::string INTEGRATION_NAME = "tcp";

/// @brief READ_TASK_TYPE is the type of the task that reads ICD messages from a TCP
/// device.
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";

/// @brief WRITE_TASK_TYPE is the type of the task that sends ICD messages to a TCP
/// device.
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";

/// @brief CONNECT_TIMEOUT bounds the resolve and the connect of each open of a device.
const x::telem::TimeSpan CONNECT_TIMEOUT = 2 * x::telem::SECOND;

/// @brief Device is a TCP device that tasks run on.
using Device = bus::Device<synnax::tcp::Properties>;

/// @brief Devices shares the connection to each TCP device among the tasks on the
/// device. It is safe for concurrent use.
class Devices {
    transport::Connections connections;

public:
    /// @brief acquires the connection to dev, and connects when no task on dev holds it
    /// open. A device that does not answer does not fail, since each turn on the
    /// connection connects again.
    /// @returns transport::CONFIG_ERROR when the properties of dev name no host or no
    /// port, or when another task holds dev with other properties.
    std::pair<std::shared_ptr<transport::Connection>, x::errors::Error>
    acquire(const Device &dev);
};

/// @brief Factory configures the tasks that read and send ICD messages on TCP
/// connections. The tasks on one device share its connection.
class Factory final : public task::Factory {
    std::shared_ptr<Devices> devices = std::make_shared<Devices>();
    bus::Readers readers{"TCP"};

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
