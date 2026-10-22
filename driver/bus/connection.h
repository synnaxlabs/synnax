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
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/clock.h"
#include "driver/bus/registry.h"
#include "driver/bus/transport.h"

namespace driver::bus {
/// @brief one transport to a device, shared by every task on the device. Tasks use it
/// one at a time, in the order they lock it. The connection owns reopening: after a
/// failed open or a failed transport, it waits out a backoff that every task on the
/// device shares before it opens again. Each open runs before the opening task takes
/// the lock, so a slow open never blocks the tasks that hold it.
class Connection {
public:
    /// @brief configures how a connection opens and reopens its transport.
    struct Config {
        /// @brief opens the transport.
        Opener open;
        /// @brief the wait before the first reopen after a failure. Each failure in a
        /// row doubles it, up to max_backoff, and a successful open resets it.
        x::telem::TimeSpan min_backoff = 250 * x::telem::MILLISECOND;
        /// @brief the longest wait between open attempts.
        x::telem::TimeSpan max_backoff = 10 * x::telem::SECOND;
        /// @brief the steady clock that times the backoff.
        x::telem::NowFunc now = steady_now;
    };

    explicit Connection(Config cfg);

    Connection(const Connection &) = delete;
    Connection &operator=(const Connection &) = delete;

    /// @brief exclusive use of a connection until destruction.
    class Guard {
    public:
        explicit Guard(Connection &conn);
        ~Guard();

        Guard(const Guard &) = delete;
        Guard &operator=(const Guard &) = delete;

        /// @returns the open transport, valid until the guard is destroyed or fail is
        /// called. While the transport is closed, the error that closed it or that
        /// failed the last open: transport::UNREACHABLE_ERROR while the device may come
        /// back, and transport::CONFIG_ERROR or transport::CRITICAL_ERROR when it
        /// cannot.
        std::pair<Transport *, x::errors::Error> transport();

        /// @brief closes the transport after it failed with err. The next open waits
        /// out the backoff.
        void fail(const x::errors::Error &err);

        /// @returns the number of times the transport has opened. A caller that buffers
        /// bytes across guards discards them when this changes.
        [[nodiscard]] std::uint64_t opens() const;

    private:
        Connection &conn;
    };

    /// @brief waits for any open another task has started, then opens the transport
    /// when it is closed and the backoff has passed. Then blocks until every earlier
    /// caller has released the connection.
    [[nodiscard]] Guard lock();

    /// @returns the error that closed the transport or failed the last open, or nil
    /// while the transport is open. Does not wait for the lock.
    [[nodiscard]] x::errors::Error error();

    /// @returns the steady time at which the next open may run.
    [[nodiscard]] x::telem::TimeStamp retry_at();

private:
    const Config cfg;
    std::mutex mu;
    std::condition_variable cv;
    std::uint64_t next_ticket = 0;
    std::uint64_t serving = 0;
    /// @brief the open transport. Guarded by mu; only the holder of a guard closes it.
    std::unique_ptr<Transport> current;
    /// @brief the number of successful opens. Guarded by mu.
    std::uint64_t opened = 0;
    /// @brief true while a task opens the transport. Guarded by mu.
    bool opening = false;
    /// @brief the error that closed the transport. Guarded by mu.
    x::errors::Error err;
    /// @brief the steady time before which no open runs. Guarded by mu.
    x::telem::TimeStamp next_open = x::telem::TimeStamp::min();
    /// @brief the wait after the next failure. Guarded by mu.
    x::telem::TimeSpan backoff;

    /// @brief opens the transport outside the lock when it is due.
    void open();
    /// @brief records a failure and schedules the next open. Requires mu.
    void failed(const x::errors::Error &err);
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
