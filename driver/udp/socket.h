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

namespace driver::udp {
/// @brief a UDP socket bound to a local port. It receives from any sender and sends to
/// an optional remote. It is not safe for concurrent use.
class Socket {
    asio::io_context ctx{1};
    asio::ip::udp::socket socket{ctx};
    std::optional<asio::ip::udp::endpoint> remote;
    std::vector<std::uint8_t> buffer = std::vector<std::uint8_t>(
        transport::BUFFER_SIZE
    );

    Socket() = default;

public:
    /// @brief binds props.port on every interface, resolves the remote in
    /// props.remote_host and props.remote_port when set, and joins
    /// props.multicast_group when set.
    /// @returns CONFIG_ERROR when props is invalid, and UNREACHABLE_ERROR when the
    /// remote cannot be resolved, the port cannot be bound, or the group cannot be
    /// joined.
    static std::pair<std::unique_ptr<Socket>, x::errors::Error>
    open(const synnax::udp::Properties &props);

    /// @brief blocks until a datagram arrives or the timeout elapses.
    /// @returns one datagram, or an empty chunk on timeout. UNREACHABLE_ERROR when the
    /// socket fails.
    std::pair<transport::Chunk, x::errors::Error> read(x::telem::TimeSpan timeout);

    /// @brief sends data to the remote as one datagram.
    /// @returns CONFIG_ERROR when no remote is set, and UNREACHABLE_ERROR when the send
    /// fails or does not finish within the timeout.
    x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout);

    /// @brief closes the socket. Reads and writes after close fail.
    void close();
};
}
