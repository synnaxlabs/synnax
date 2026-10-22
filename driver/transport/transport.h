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
#include <cstdint>
#include <optional>
#include <span>
#include <system_error>

#include "asio/error.hpp"
#include "asio/io_context.hpp"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/transport/errors.h"

/// @brief the shape shared by the serial, TCP, and UDP transports. Each transport reads
/// with a timeout on the caller's thread by starting one asynchronous operation and
/// running its own io_context until the operation completes or the timeout cancels it,
/// because Asio's synchronous calls cannot time out and a separate I/O thread would add
/// locking that a single-threaded reader never needs.
namespace driver::transport {
/// @brief the largest chunk one read returns. It holds any UDP datagram.
constexpr std::size_t BUFFER_SIZE = 64 * 1024;

/// @brief bytes received from a transport.
struct Chunk {
    /// @brief the received bytes, valid until the next call on the transport. Empty
    /// when nothing arrived before the timeout.
    std::span<const std::uint8_t> data;
    /// @brief the host time at which the bytes arrived.
    x::telem::TimeStamp time;
    /// @brief the number of bytes the line received with a parity or framing error and
    /// dropped from data.
    std::size_t errors = 0;
    /// @brief the number of receive overruns the line reported since the last read:
    /// bytes the UART or the OS dropped because the Driver read too slowly.
    std::size_t overruns = 0;
};

/// @brief the outcome of one asynchronous operation.
struct Completion {
    /// @brief the operation's error. asio::error::operation_aborted on timeout.
    std::error_code ec;
    /// @brief the number of bytes the operation transferred.
    std::size_t size = 0;
    /// @brief the host time at which the operation completed.
    x::telem::TimeStamp time;
    /// @brief true when run_for cancelled the operation at its timeout.
    bool cancelled = false;

    /// @returns true when the timeout cancelled the operation. An operation that the OS
    /// aborted on its own, as Windows does when a device goes away, is an error.
    [[nodiscard]] bool timed_out() const {
        return this->cancelled && this->ec == asio::error::operation_aborted;
    }
};

/// @brief starts one asynchronous operation and runs ctx on the calling thread until it
/// completes or the timeout elapses, then cancels it. It returns only after the
/// operation's handler has run, so no operation outlives the call.
/// @param ctx the context that owns the operation's I/O objects. It must have no other
/// pending work.
/// @param timeout the longest the operation may run before cancellation.
/// @param start starts the operation, completing into the handler it receives. The
/// handler takes an std::error_code and a byte count.
/// @param cancel cancels the operation.
template<typename Start, typename Cancel>
Completion run_for(
    asio::io_context &ctx,
    const x::telem::TimeSpan timeout,
    Start &&start,
    Cancel &&cancel
) {
    std::optional<Completion> done;
    start([&done](const std::error_code &ec, const std::size_t size) {
        done = Completion{.ec = ec, .size = size, .time = x::telem::TimeStamp::now()};
    });
    ctx.restart();
    ctx.run_for(timeout.chrono());
    if (!done.has_value()) {
        cancel();
        ctx.restart();
        ctx.run();
        done->cancelled = true;
    }
    return *done;
}
}
