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
#include <cstddef>
#include <cstdint>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <span>
#include <string>
#include <system_error>
#include <thread>
#include <type_traits>
#include <utility>
#include <vector>

#include "asio/error.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/address.hpp"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace driver::transport {
/// @brief BUFFER_SIZE is the most bytes that one read returns. It holds any UDP
/// datagram.
constexpr std::size_t BUFFER_SIZE = std::size_t{64} * 1024;

/// @brief Chunk is the bytes that one read gives.
struct Chunk {
    /// @brief data is the bytes that arrived, valid until the next call on the
    /// transport. It is empty when nothing arrived before the timeout.
    std::span<const std::uint8_t> data;
    /// @brief time is the host time at which the bytes arrived.
    x::telem::TimeStamp time;
};

/// @brief Transport is a connection to a device, such as a serial port, a TCP client,
/// or a UDP socket. A read from a byte stream gives the bytes that arrived, and a read
/// from a datagram socket gives one datagram. It is not safe for concurrent use.
class Transport {
public:
    virtual ~Transport() = default;

    /// @brief blocks until bytes arrive or timeout elapses.
    /// @returns the bytes, or an empty chunk on timeout. UNREACHABLE_ERROR when the
    /// connection fails.
    virtual std::pair<Chunk, x::errors::Error> read(x::telem::TimeSpan timeout) = 0;

    /// @brief blocks until it writes all of data or timeout elapses.
    /// @returns UNREACHABLE_ERROR when the connection fails or timeout elapses, because
    /// a partial write breaks the stream.
    virtual x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout) = 0;
};

/// @brief Completion is the outcome of one asynchronous operation.
struct Completion {
    /// @brief ec is the error of the operation. It is asio::error::operation_aborted on
    /// a timeout.
    std::error_code ec;
    /// @brief size is the number of bytes that the operation moved.
    std::size_t size = 0;
    /// @brief time is the host time at which the operation completed.
    x::telem::TimeStamp time;
    /// @brief cancelled is true when run_for cancelled the operation at its timeout.
    bool cancelled = false;

    /// @returns true when the timeout cancelled the operation. An operation that the OS
    /// aborts on its own, as Windows does when a device goes away, is an error.
    [[nodiscard]] bool timed_out() const {
        return this->cancelled && this->ec == asio::error::operation_aborted;
    }
};

/// @brief Handler is the completion handler that run_for gives an operation.
using Handler = std::function<void(const std::error_code &, std::size_t)>;

/// @brief starts one asynchronous operation and runs ctx on the calling thread until
/// the operation completes or timeout elapses, then cancels it. It returns only after
/// the handler of the operation runs, so no operation outlives the call. Asio's
/// synchronous calls cannot time out, and an I/O thread would need locks that one
/// reader does not.
/// @param ctx owns the I/O objects of the operation. It must have no other pending
/// work.
/// @param start starts the operation with the handler it takes, which takes an
/// std::error_code and a byte count.
/// @param cancel cancels the operation.
template<typename Start, typename Cancel>
Completion run_for(
    asio::io_context &ctx,
    const x::telem::TimeSpan timeout,
    Start &&start,
    Cancel &&cancel
) {
    std::optional<Completion> done;
    std::forward<Start>(start)([&done](
                                   const std::error_code &ec,
                                   const std::size_t size
                               ) {
        done = Completion{.ec = ec, .size = size, .time = x::telem::TimeStamp::now()};
    });
    ctx.restart();
    ctx.run_for(timeout.chrono());
    if (!done.has_value()) {
        std::forward<Cancel>(cancel)();
        ctx.restart();
        ctx.run();
        done->cancelled = true;
    }
    return *done;
}

/// @brief runs call, which blocks and cannot be cancelled, such as getaddrinfo, on its
/// own thread, and stops waiting for it after timeout. A call that it stops waiting for
/// keeps its thread until it returns, and its result is dropped.
/// @param call returns a pair of a value and an std::error_code. It can outlive the
/// caller, so it must own everything that it uses, such as a host string, and must not
/// capture a reference or this.
/// @returns the result of call, or a default value and std::errc::timed_out when
/// timeout elapses first.
template<typename Call>
auto abandon_after(const x::telem::TimeSpan timeout, Call call)
    -> std::invoke_result_t<Call> {
    using Result = std::invoke_result_t<Call>;
    struct State {
        std::mutex mu;
        std::condition_variable cv;
        std::optional<Result> result;
    };
    auto state = std::make_shared<State>();
    std::thread([state, call = std::move(call)]() mutable {
        auto result = call();
        std::lock_guard lock(state->mu);
        state->result = std::move(result);
        state->cv.notify_one();
    }).detach();
    std::unique_lock lock(state->mu);
    if (!state->cv.wait_for(lock, timeout.chrono(), [&state] {
            return state->result.has_value();
        }))
        return {{}, std::make_error_code(std::errc::timed_out)};
    return std::move(*state->result);
}

/// @brief resolves host on port. An IP address resolves at once. A name resolves
/// through getaddrinfo, which cannot be cancelled, so a resolve that times out finishes
/// on its own thread.
/// @tparam Protocol is asio::ip::tcp or asio::ip::udp.
/// @returns the endpoints of host, or the error of the resolve. std::errc::timed_out
/// when timeout elapses first.
template<typename Protocol>
std::pair<std::vector<typename Protocol::endpoint>, std::error_code> resolve(
    const std::string &host,
    const std::uint16_t port,
    const x::telem::TimeSpan timeout
) {
    using Endpoint = Protocol::endpoint;
    std::error_code ec;
    if (const auto literal = asio::ip::make_address(host, ec); !ec)
        return {{Endpoint(literal, port)}, {}};
    return abandon_after(timeout, [host, port] {
        asio::io_context ctx;
        typename Protocol::resolver resolver(ctx);
        std::error_code err;
        const auto results = resolver.resolve(host, std::to_string(port), err);
        std::vector<Endpoint> endpoints;
        for (const auto &result: results)
            endpoints.push_back(result.endpoint());
        return std::pair{endpoints, err};
    });
}
}
