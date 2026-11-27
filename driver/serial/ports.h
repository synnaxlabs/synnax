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
#include "client/cpp/task/task.h"
#include "x/cpp/errors/errors.h"

#include "driver/transport/connection.h"

namespace driver::serial {
/// @brief Device is a serial device that tasks run on.
struct Device {
    /// @brief key is the key of the device.
    std::string key;
    /// @brief name is the name of the device.
    std::string name;
    /// @brief props is the properties of the port of the device.
    synnax::serial::Properties props;
};

/// @brief retrieves the serial device with key.
/// @returns the error of the retrieve, or VALIDATION when the properties are invalid.
std::pair<Device, x::errors::Error>
retrieve_device(const synnax::device::Client &devices, const std::string &key);

/// @brief Ports shares the serial ports of an integration among its tasks. Each device
/// has one read task. Each open port belongs to one device, whatever path names it. It
/// is safe for concurrent use.
class Ports {
    /// @brief Holders gives each key to one holder at a time.
    class Holders;

    transport::Connections connections;
    std::shared_ptr<Holders> readers;
    std::shared_ptr<Holders> devices;

public:
    Ports();

    /// @brief Claim holds a device for a read task, or a port for a device, until it is
    /// destroyed.
    class Claim {
        std::shared_ptr<Holders> holders;
        std::string key;
        Claim(std::shared_ptr<Holders> holders, std::string key);
        friend class Ports;

    public:
        ~Claim();
        Claim(const Claim &) = delete;
        Claim &operator=(const Claim &) = delete;
    };

    /// @brief claims the device with key for task until the claim is destroyed. A task
    /// can claim a device that it holds again.
    /// @returns CONFIGURATION_ERROR naming the read task that holds the device.
    std::pair<std::unique_ptr<Claim>, x::errors::Error>
    claim_read(const std::string &device, const synnax::task::Task &task);

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
