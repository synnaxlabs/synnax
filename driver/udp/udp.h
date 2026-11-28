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

#include "client/cpp/udp/types.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/device.h"
#include "driver/bus/holders.h"
#include "driver/task/task.h"
#include "driver/transport/connection.h"

namespace driver::udp {
/// @brief INTEGRATION_NAME is the name of the UDP integration.
const std::string INTEGRATION_NAME = "udp";

/// @brief READ_TASK_TYPE is the type of the task that reads ICD messages from a UDP
/// device.
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";

/// @brief WRITE_TASK_TYPE is the type of the task that sends ICD messages to a UDP
/// device.
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";

/// @brief RESOLVE_TIMEOUT bounds the resolve of the remote host on each open of a
/// device.
const x::telem::TimeSpan RESOLVE_TIMEOUT = 2 * x::telem::SECOND;

/// @brief Device is a UDP device that tasks run on.
using Device = bus::Device<synnax::udp::Properties>;

/// @brief Devices shares the socket of each UDP device among the tasks on the device.
/// It is safe for concurrent use.
class Devices {
    transport::Connections connections;

public:
    /// @brief acquires the socket of dev, and binds it when no task on dev holds it
    /// open. A remote host that does not resolve does not fail, since each turn on the
    /// socket opens it again.
    /// @returns transport::CONFIG_ERROR when the properties of dev are invalid, when
    /// another socket holds the local port, or when another task holds dev with other
    /// properties.
    std::pair<std::shared_ptr<transport::Connection>, x::errors::Error>
    acquire(const Device &dev);
};

/// @brief Factory configures the tasks that read and send ICD messages on UDP sockets.
/// The tasks on one device share its socket.
class Factory final : public task::Factory {
    std::shared_ptr<Devices> devices = std::make_shared<Devices>();
    bus::Readers readers{"UDP"};

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
