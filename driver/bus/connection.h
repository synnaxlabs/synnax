// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <cstdint>
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/bus/registry.h"
#include "driver/bus/transport.h"

namespace driver::bus {
/// @brief one transport to a device, shared by every task on the device. Tasks use
/// it one at a time, in the order they lock it. The transport opens on first use and
/// closes when the connection is destroyed.
class Connection {
public:
    explicit Connection(Opener open): open(std::move(open)) {}

    Connection(const Connection &) = delete;
    Connection &operator=(const Connection &) = delete;

    /// @brief exclusive use of a connection until destruction.
    class Guard {
    public:
        explicit Guard(Connection &conn);
        ~Guard();

        Guard(const Guard &) = delete;
        Guard &operator=(const Guard &) = delete;

        /// @returns the transport, opened first when it is closed. Valid until the
        /// guard is destroyed or close is called. The error of the Opener when the
        /// transport cannot open.
        std::pair<Transport *, x::errors::Error> transport();

        /// @brief closes the transport, so the next caller of transport reopens it.
        void close();

        /// @returns the number of times the transport has opened. A caller that
        /// buffers bytes across guards discards them when this changes.
        [[nodiscard]] std::uint64_t opens() const { return this->conn.opened; }

    private:
        Connection &conn;
    };

    /// @brief blocks until every earlier caller has released the connection.
    [[nodiscard]] Guard lock() { return Guard(*this); }

private:
    Opener open;
    std::mutex mu;
    std::condition_variable cv;
    std::uint64_t next_ticket = 0;
    std::uint64_t serving = 0;
    /// @brief the open transport, used only by the holder of a guard.
    std::unique_ptr<Transport> current;
    std::uint64_t opened = 0;
};

/// @brief the byte-stream and datagram connections of an integration.
using Connections = Registry<Connection>;

/// @brief acquires the shared connection to a task's device.
using Acquire = Acquirer<Connection>;

/// @returns an Acquire of the device's connection from connections.
Acquire acquirer(
    std::shared_ptr<Connections> connections,
    std::string key,
    x::json::json settings,
    Opener open
);
}
