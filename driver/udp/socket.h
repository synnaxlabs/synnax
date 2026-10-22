// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <optional>
#include <span>
#include <utility>
#include <vector>

#include "asio/io_context.hpp"
#include "asio/ip/udp.hpp"

#include "client/cpp/udp/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/transport/transport.h"
#include "driver/udp/errors.h"
#include "driver/udp/stack.h"

namespace driver::udp {
/// @brief configures how a socket reaches the network.
struct Config {
    /// @brief makes the socket's stack. Tests substitute a model of the network.
    Stacks stacks = [](asio::io_context &ctx) {
        return std::make_unique<AsioStack>(ctx);
    };
    /// @brief how long the socket waits for its remote host to resolve.
    x::telem::TimeSpan resolve_timeout = 5 * x::telem::SECOND;
};

/// @brief a UDP socket bound to a local port. It sends to an optional remote and
/// receives from that remote alone, or from any sender when no remote is set. Calls
/// must not overlap, but any thread may make them. Errors map through translate.
class Socket {
    asio::io_context ctx{1};
    std::unique_ptr<Stack> stack;
    std::optional<asio::ip::udp::endpoint> remote;
    asio::ip::udp::endpoint sender;
    std::vector<std::uint8_t> buffer = std::vector<std::uint8_t>(
        transport::BUFFER_SIZE
    );

    explicit Socket(const Config &config);

public:
    /// @brief binds props.port on props.local_address, or on every interface when it is
    /// empty, resolves the remote in props.remote_host and props.remote_port when set,
    /// and joins props.multicast_group when set. A multicast socket binds every
    /// interface and joins the group on the interface that holds props.local_address.
    /// @returns CONFIG_ERROR when props is invalid or the port is in use, and
    /// UNREACHABLE_ERROR when the remote does not resolve, no interface holds the local
    /// address, or the group cannot be joined.
    static std::pair<std::unique_ptr<Socket>, x::errors::Error>
    open(const synnax::udp::Properties &props, const Config &config = Config{});

    /// @brief blocks until a datagram arrives or the timeout elapses.
    /// @returns one datagram, or an empty chunk on timeout. UNREACHABLE_ERROR when the
    /// socket fails.
    std::pair<transport::Chunk, x::errors::Error> read(x::telem::TimeSpan timeout);

    /// @brief sends data to the remote as one datagram.
    /// @returns CONFIG_ERROR when no remote is set or the datagram is too large, and
    /// UNREACHABLE_ERROR when the send fails or does not finish within the timeout.
    x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout);

    /// @brief closes the socket. Reads and writes after close fail.
    void close();
};
}
