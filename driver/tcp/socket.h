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
#include <functional>
#include <memory>
#include <span>
#include <string>
#include <system_error>
#include <utility>
#include <vector>

#include "asio/io_context.hpp"
#include "asio/ip/tcp.hpp"

#include "x/cpp/telem/telem.h"

namespace driver::tcp {
/// @brief completes an asynchronous socket operation with its error and the number of
/// bytes it moved.
using Handler = std::function<void(const std::error_code &, std::size_t)>;

/// @brief the operating system calls that a Client makes for one connection. Each
/// asynchronous operation completes through the io_context that the socket was made
/// for.
class Socket {
public:
    virtual ~Socket() = default;

    /// @brief resolves host to its addresses, giving up after timeout.
    /// @returns std::errc::timed_out when the timeout elapses first.
    virtual std::pair<std::vector<asio::ip::tcp::endpoint>, std::error_code> resolve(
        const std::string &host,
        std::uint16_t port,
        x::telem::TimeSpan timeout
    ) = 0;

    /// @brief opens the socket if it is closed and connects it to endpoint.
    virtual void
    async_connect(const asio::ip::tcp::endpoint &endpoint, Handler handler) = 0;

    /// @brief turns off Nagle's algorithm and turns on keepalive probes.
    virtual std::error_code configure() = 0;

    /// @brief reads at least one byte into buffer.
    virtual void async_read_some(std::span<std::uint8_t> buffer, Handler handler) = 0;

    /// @brief writes at least one byte of data.
    virtual void
    async_write_some(std::span<const std::uint8_t> data, Handler handler) = 0;

    /// @brief completes the pending operations with asio::error::operation_aborted.
    virtual void cancel() = 0;

    /// @brief closes the socket and aborts its pending operations. Later operations
    /// fail with asio::error::bad_descriptor.
    virtual void close() = 0;
};

/// @brief makes the socket for a connection on ctx.
using Sockets = std::function<std::unique_ptr<Socket>(asio::io_context &ctx)>;

/// @brief a Socket on the host's network stack.
class AsioSocket final : public Socket {
    asio::ip::tcp::socket socket;

public:
    explicit AsioSocket(asio::io_context &ctx);

    /// @brief resolves through getaddrinfo, which cannot be cancelled, so a resolve
    /// that times out finishes on its own thread.
    std::pair<std::vector<asio::ip::tcp::endpoint>, std::error_code> resolve(
        const std::string &host,
        std::uint16_t port,
        x::telem::TimeSpan timeout
    ) override;

    void
    async_connect(const asio::ip::tcp::endpoint &endpoint, Handler handler) override;

    std::error_code configure() override;

    void async_read_some(std::span<std::uint8_t> buffer, Handler handler) override;

    void async_write_some(std::span<const std::uint8_t> data, Handler handler) override;

    void cancel() override;

    void close() override;
};
}
