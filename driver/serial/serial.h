// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
#include <utility>

#include "client/cpp/device/device.h"
#include "client/cpp/serial/types.gen.h"
#include "client/cpp/task/task.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"

#include "driver/task/task.h"
#include "driver/transport/connection.h"

namespace driver::serial {
/// @brief INTEGRATION_NAME is the name of the serial integration.
const std::string INTEGRATION_NAME = "serial";

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

/// @brief Ports shares the serial ports of an integration among its tasks. It gives
/// each port path to one read task, and to the tasks of one device at a time. It is
/// safe for concurrent use.
class Ports {
    /// @brief Holders gives each port path to one holder at a time.
    class Holders {
        struct Holder {
            std::string id;
            std::string name;
            std::size_t claims = 0;
        };
        std::mutex mu;
        std::unordered_map<std::string, Holder> held;

    public:
        /// @brief claims path for the holder with id.
        /// @returns the name of the holder of path when another holder has it.
        std::optional<std::string>
        take(const std::string &path, const std::string &id, const std::string &name);

        /// @brief releases one claim on path.
        void release(const std::string &path);
    };

    transport::Connections connections;
    std::shared_ptr<Holders> readers = std::make_shared<Holders>();
    std::shared_ptr<Holders> devices = std::make_shared<Holders>();

public:
    /// @brief Claim holds a port path until it is destroyed.
    class Claim {
        std::shared_ptr<Holders> holders;
        std::string path;
        Claim(std::shared_ptr<Holders> holders, std::string path);
        friend class Ports;

    public:
        ~Claim();
        Claim(const Claim &) = delete;
        Claim &operator=(const Claim &) = delete;
    };

    /// @brief Lease is the use of the port of a device by a running task. Destroying it
    /// releases the port, which closes when no other task on the device holds it.
    struct Lease {
        /// @brief claim holds the port path for the device.
        std::unique_ptr<Claim> claim;
        /// @brief conn is the connection to the port.
        std::shared_ptr<transport::Connection> conn;
    };

    /// @brief claims the port at path for task until the claim is destroyed.
    /// @returns CONFIGURATION_ERROR naming the read task that holds path.
    std::pair<std::unique_ptr<Claim>, x::errors::Error>
    claim_read(const std::string &path, const synnax::task::Task &task);

    /// @brief acquires the connection to the port of dev, and opens the port when no
    /// task on dev holds it open. A missing or busy port does not fail, since each turn
    /// on the connection opens it again.
    /// @returns transport::CONFIG_ERROR naming the device that uses the port, or when
    /// another task holds dev with other properties or the port cannot apply them, and
    /// transport::CRITICAL_ERROR when reopening cannot fix the failure.
    std::pair<Lease, x::errors::Error> acquire(const Device &dev);
};

/// @brief Factory configures the tasks that read ICD messages from serial ports. The
/// tasks on one device share its port.
class Factory final : public task::Factory {
    std::shared_ptr<Ports> ports = std::make_shared<Ports>();

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
