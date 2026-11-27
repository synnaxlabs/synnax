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

#include "client/cpp/serial/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/bus/device.h"
#include "driver/bus/holders.h"
#include "driver/transport/connection.h"

namespace driver::serial {
/// @brief Device is a serial device that tasks run on.
using Device = bus::Device<synnax::serial::Properties>;

/// @brief Ports shares the serial ports of an integration among its tasks. Each open
/// port belongs to one device, whatever path names it. It is safe for concurrent use.
class Ports {
    transport::Connections connections;
    /// @brief devices holds each open port for its device.
    std::shared_ptr<bus::Holders> devices = std::make_shared<bus::Holders>();

public:
    /// @brief acquires the connection to the port of dev, and opens the port when no
    /// task on dev holds it open. Each open claims the port for dev until the port
    /// closes. A missing or busy port does not fail, since each turn on the connection
    /// opens it again.
    /// @returns transport::CONFIG_ERROR naming the device that has the port open, or
    /// when another task holds dev with other properties or the port cannot apply them,
    /// and transport::CRITICAL_ERROR when reopening cannot fix the failure.
    std::pair<std::shared_ptr<transport::Connection>, x::errors::Error>
    acquire(const Device &dev);
};
}
