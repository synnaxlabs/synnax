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
#include <string>
#include <utility>
#include <vector>

#include "asio/io_context.hpp"
#include "asio/ip/udp.hpp"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/transport/transport.h"

namespace driver::udp {
/// @brief Address sets where a socket listens and where it sends.
struct Address {
    /// @brief local_address is the IP address to bind, or empty for every interface.
    std::string local_address;
    /// @brief port is the local port to bind, or 0 for any free port.
    std::uint16_t port = 0;
    /// @brief remote_host is the host to send to and the only one to receive from, on
    /// any of its ports, or empty to receive from any sender and send nothing.
    std::string remote_host;
    /// @brief remote_port is the port to send to. It is required with remote_host.
    std::uint16_t remote_port = 0;
};

/// @brief Socket is a UDP socket bound to a local port. A read gives one datagram.
class Socket final : public transport::Transport {
    asio::io_context ctx{1};
    asio::ip::udp::socket socket{ctx};
    std::optional<asio::ip::udp::endpoint> remote;
    /// @brief sender is the source of the last datagram.
    asio::ip::udp::endpoint sender;
    std::vector<std::uint8_t> buffer = std::vector<std::uint8_t>(
        transport::BUFFER_SIZE
    );

    Socket() = default;

public:
    /// @brief binds addr.port on addr.local_address, and resolves the remote host when
    /// addr sets one.
    /// @param timeout bounds the resolve.
    /// @returns transport::CONFIG_ERROR when addr is invalid or the port is in use, and
    /// transport::UNREACHABLE_ERROR when the remote host does not resolve or no
    /// interface holds the local address.
    static std::pair<std::unique_ptr<Socket>, x::errors::Error>
    open(const Address &addr, x::telem::TimeSpan timeout);

    /// @brief blocks until a datagram from the remote host, or from any sender when
    /// there is none, arrives or timeout elapses. It skips empty datagrams, which carry
    /// no message.
    /// @returns one datagram, or an empty chunk on timeout.
    /// transport::UNREACHABLE_ERROR when the network fails, and transport::CONFIG_ERROR
    /// or transport::CRITICAL_ERROR when reopening the socket cannot fix the failure.
    std::pair<transport::Chunk, x::errors::Error>
    read(x::telem::TimeSpan timeout) override;

    /// @brief sends data to the remote host as one datagram.
    /// @returns transport::CONFIG_ERROR when there is no remote host or data is too
    /// large for a datagram, and transport::UNREACHABLE_ERROR when the send fails or
    /// timeout elapses.
    x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout) override;
};
}
