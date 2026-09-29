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
#include <unordered_map>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/bus/transport.h"

namespace driver::bus {
/// @brief opens a new T for a task's device.
template<typename T>
using BasicOpener = std::function<std::pair<std::unique_ptr<T>, x::errors::Error>()>;

/// @brief one open T, such as a transport or a card channel, shared by every task on
/// a device. Tasks use it one at a time, in the order they lock it. T opens on first
/// use and closes when the connection is destroyed.
template<typename T>
class BasicConnection {
public:
    explicit BasicConnection(BasicOpener<T> open): open(std::move(open)) {}

    BasicConnection(const BasicConnection &) = delete;
    BasicConnection &operator=(const BasicConnection &) = delete;

    /// @brief exclusive use of a connection until destruction.
    class Guard {
    public:
        explicit Guard(BasicConnection &conn): conn(conn) {
            std::unique_lock lock(conn.mu);
            const auto ticket = conn.next_ticket++;
            conn.cv.wait(lock, [&] { return conn.serving == ticket; });
        }

        ~Guard() {
            {
                std::lock_guard lock(this->conn.mu);
                this->conn.serving++;
            }
            this->conn.cv.notify_all();
        }

        Guard(const Guard &) = delete;
        Guard &operator=(const Guard &) = delete;

        /// @returns the T, opened first when it is closed. Valid until the guard is
        /// destroyed or close is called. The error of the opener when T cannot open.
        std::pair<T *, x::errors::Error> transport() {
            if (this->conn.current == nullptr) {
                auto [t, err] = this->conn.open();
                if (err) return {nullptr, err};
                this->conn.current = std::move(t);
                this->conn.opened++;
            }
            return {this->conn.current.get(), x::errors::NIL};
        }

        /// @brief closes the T, so the next caller of transport reopens it.
        void close() { this->conn.current.reset(); }

        /// @returns the number of times T has opened. A caller that buffers bytes
        /// across guards discards them when this changes.
        [[nodiscard]] std::uint64_t opens() const { return this->conn.opened; }

    private:
        BasicConnection &conn;
    };

    /// @brief blocks until every earlier caller has released the connection.
    [[nodiscard]] Guard lock() { return Guard(*this); }

private:
    BasicOpener<T> open;
    std::mutex mu;
    std::condition_variable cv;
    std::uint64_t next_ticket = 0;
    std::uint64_t serving = 0;
    /// @brief the open T, used only by the holder of a guard.
    std::unique_ptr<T> current;
    std::uint64_t opened = 0;
};

/// @brief acquires the shared connection to a task's device.
/// @returns transport::CONFIG_ERROR when the device is open with other settings.
template<typename T>
using BasicAcquire = std::function<
    std::pair<std::shared_ptr<BasicConnection<T>>, x::errors::Error>()>;

/// @brief the open connections of an integration, one per device. Safe for concurrent
/// use.
template<typename T>
class BasicConnections {
public:
    /// @brief returns the live connection to the device, or creates one.
    /// @param key the device key.
    /// @param settings how the device opens. Every task on the device must use the
    /// same settings.
    /// @param open opens T.
    /// @returns transport::CONFIG_ERROR when the device is open with other settings.
    std::pair<std::shared_ptr<BasicConnection<T>>, x::errors::Error> acquire(
        const std::string &key,
        const x::json::json &settings,
        const BasicOpener<T> &open
    ) {
        std::lock_guard lock(this->mu);
        auto &entry = this->entries[key];
        if (auto conn = entry.conn.lock()) {
            if (entry.settings != settings)
                return {
                    nullptr,
                    x::errors::Error(
                        transport::CONFIG_ERROR,
                        "another task has device " + key +
                            " open with different settings. Stop or reconfigure it"
                    )
                };
            return {std::move(conn), x::errors::NIL};
        }
        auto conn = std::make_shared<BasicConnection<T>>(open);
        entry = {.conn = conn, .settings = settings};
        return {std::move(conn), x::errors::NIL};
    }

private:
    struct Entry {
        std::weak_ptr<BasicConnection<T>> conn;
        x::json::json settings;
    };

    std::mutex mu;
    std::unordered_map<std::string, Entry> entries;
};

/// @brief one transport to a device, shared by every task on the device.
using Connection = BasicConnection<Transport>;
/// @brief acquires the shared transport to a task's device.
using Acquire = BasicAcquire<Transport>;
/// @brief the open transports of an integration, one per device.
using Connections = BasicConnections<Transport>;

/// @returns an Acquire of the device's connection from connections.
Acquire acquirer(
    std::shared_ptr<Connections> connections,
    std::string key,
    x::json::json settings,
    Opener open
);
}
