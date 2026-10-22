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
#include <optional>
#include <span>
#include <string>
#include <system_error>
#include <utility>
#include <vector>

#include "asio/io_context.hpp"
#include "asio/ip/udp.hpp"

namespace driver::udp {
/// @brief completes an asynchronous operation with its error and the number of bytes it
/// moved.
using Handler = std::function<void(const std::error_code &, std::size_t)>;

/// @brief the operating system calls that a Socket makes. Each asynchronous operation
/// completes through the io_context that the stack was made for.
class Stack {
public:
    virtual ~Stack() = default;

    /// @brief resolves host to its addresses.
    virtual std::pair<std::vector<asio::ip::udp::endpoint>, std::error_code>
    resolve(const std::string &host, std::uint16_t port) = 0;

    /// @brief opens a socket for protocol.
    virtual std::error_code open(const asio::ip::udp &protocol) = 0;

    /// @brief lets other sockets bind the same port, as multicast receivers do.
    virtual std::error_code share() = 0;

    /// @brief binds the socket to local. Port 0 takes a free port.
    virtual std::error_code bind(const asio::ip::udp::endpoint &local) = 0;

    /// @brief joins the multicast group on the interface that holds iface, or on the
    /// default interface when iface is empty.
    virtual std::error_code join(
        const asio::ip::address &group,
        const std::optional<asio::ip::address> &iface
    ) = 0;

    /// @brief receives one datagram into buffer and its source into sender. The caller
    /// keeps buffer and sender alive until the handler runs.
    virtual void async_receive_from(
        std::span<std::uint8_t> buffer,
        asio::ip::udp::endpoint &sender,
        Handler handler
    ) = 0;

    /// @brief sends data as one datagram to the endpoint.
    virtual void async_send_to(
        std::span<const std::uint8_t> data,
        const asio::ip::udp::endpoint &to,
        Handler handler
    ) = 0;

    /// @brief completes the pending operations with asio::error::operation_aborted.
    virtual void cancel() = 0;

    /// @brief closes the socket and aborts its pending operations. Later operations
    /// fail with asio::error::bad_descriptor.
    virtual void close() = 0;
};

/// @brief makes the stack for a socket on ctx.
using Stacks = std::function<std::unique_ptr<Stack>(asio::io_context &ctx)>;

/// @brief a Stack on the host's network stack.
class AsioStack final : public Stack {
    asio::io_context &ctx;
    asio::ip::udp::socket socket;

public:
    explicit AsioStack(asio::io_context &ctx);

    std::pair<std::vector<asio::ip::udp::endpoint>, std::error_code>
    resolve(const std::string &host, std::uint16_t port) override;

    std::error_code open(const asio::ip::udp &protocol) override;

    std::error_code share() override;

    std::error_code bind(const asio::ip::udp::endpoint &local) override;

    std::error_code join(
        const asio::ip::address &group,
        const std::optional<asio::ip::address> &iface
    ) override;

    void async_receive_from(
        std::span<std::uint8_t> buffer,
        asio::ip::udp::endpoint &sender,
        Handler handler
    ) override;

    void async_send_to(
        std::span<const std::uint8_t> data,
        const asio::ip::udp::endpoint &to,
        Handler handler
    ) override;

    void cancel() override;

    void close() override;
};
}
