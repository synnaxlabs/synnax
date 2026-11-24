// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
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
#include "x/cpp/telem/telem.h"

#include "driver/transport/transport.h"

namespace driver::transport {
/// @brief Opener opens a new transport to a device.
/// @returns UNREACHABLE_ERROR when the device cannot be reached now, and CONFIG_ERROR
/// or CRITICAL_ERROR when reopening cannot fix the failure.
using Opener = std::function<std::pair<std::unique_ptr<Transport>, x::errors::Error>()>;

/// @brief Clock returns the time on a steady clock, which times the backoff.
using Clock = std::function<std::chrono::steady_clock::time_point()>;

/// @brief Connection is one transport to a device, shared by every task on the device.
/// Tasks take turns in the order that they ask for one. After a failed open or a failed
/// transport, the connection waits out a backoff that every task shares before it opens
/// again. An open runs before the opening task waits for its turn, so a slow open never
/// blocks the task whose turn it is. It is safe for concurrent use.
class Connection {
public:
    /// @brief Config sets how a connection opens and reopens its transport.
    struct Config {
        Opener open;
        /// @brief min_backoff is the wait before the first open after a failure. Each
        /// failure in a row doubles it, up to max_backoff. A turn that takes the open
        /// transport and does not fail resets it.
        x::telem::TimeSpan min_backoff = 250 * x::telem::MILLISECOND;
        x::telem::TimeSpan max_backoff = 10 * x::telem::SECOND;
        Clock now = std::chrono::steady_clock::now;
    };

    explicit Connection(Config cfg);
    Connection(const Connection &) = delete;
    Connection &operator=(const Connection &) = delete;

    /// @brief Guard is the turn of a task on a connection, which ends when the guard is
    /// destroyed.
    class Guard {
        friend class Connection;
        Connection &conn;
        /// @brief healthy is true when the turn took the open transport and did not
        /// fail.
        bool healthy = false;

        Guard(Connection &conn, std::uint64_t ticket);

    public:
        ~Guard();
        Guard(const Guard &) = delete;
        Guard &operator=(const Guard &) = delete;

        /// @brief returns the open transport, valid until the guard ends or fail runs.
        /// @returns the error that closed the transport or failed the last open while
        /// the transport is closed.
        std::pair<Transport *, x::errors::Error> transport();

        /// @brief closes the transport after it failed with err. The next open waits
        /// out the backoff.
        void fail(const x::errors::Error &err);

        /// @brief returns the number of times the transport has opened. A caller that
        /// keeps bytes across turns drops them when it changes.
        [[nodiscard]] std::uint64_t opens() const;
    };

    /// @brief takes a place in line, then opens the transport when it is closed and the
    /// backoff has passed, after any open that another task started. Then blocks until
    /// the turns of earlier callers end.
    [[nodiscard]] Guard lock();

    /// @brief returns the time on Config::now at which the next open may run. A task
    /// that finds the transport closed waits until then.
    [[nodiscard]] std::chrono::steady_clock::time_point retry_at();

private:
    const Config cfg;
    std::mutex mu;
    std::condition_variable cv;
    /// @brief next_ticket and serving order the turns of the tasks.
    std::uint64_t next_ticket = 0;
    std::uint64_t serving = 0;
    std::unique_ptr<Transport> current;
    std::uint64_t opened = 0;
    /// @brief opening is true while a task opens the transport.
    bool opening = false;
    /// @brief err is the error that closed the transport or failed the last open.
    x::errors::Error err;
    /// @brief next_open is the time before which no open runs.
    std::chrono::steady_clock::time_point
        next_open = std::chrono::steady_clock::time_point::min();
    /// @brief backoff is the wait after the next failure.
    x::telem::TimeSpan backoff;

    /// @brief opens the transport outside mu when it is closed and due.
    void open();
    /// @brief records err and schedules the next open. Requires mu.
    void failed(const x::errors::Error &err);
};

/// @brief Connections holds the live connection of each device, which every task on the
/// device shares. A connection closes when the last task releases it. It is safe for
/// concurrent use.
class Connections {
    struct Entry {
        std::weak_ptr<Connection> conn;
        x::json::json settings;
    };

    std::mutex mu;
    std::unordered_map<std::string, Entry> entries;

public:
    /// @brief returns the live connection of the device with key, or creates one from
    /// cfg. A new connection opens on its first lock.
    /// @param settings how the device opens. Every task on the device must use the same
    /// settings.
    /// @returns CONFIG_ERROR when another task has the device open with other settings.
    std::pair<std::shared_ptr<Connection>, x::errors::Error> acquire(
        const std::string &key,
        const x::json::json &settings,
        Connection::Config cfg
    );
};
}
