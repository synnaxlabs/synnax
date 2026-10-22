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
#include <memory>
#include <optional>
#include <string>
#include <system_error>
#include <utility>

#include "asio/ip/udp.hpp"

#include "x/cpp/telem/telem.h"

#include "driver/udp/stack.h"

/// @brief an in-process model of the host's network stack that replaces the operating
/// system calls under a Socket. Its semantics follow udp(7), bind(2), and ip(7): a
/// datagram keeps its boundaries and arrives whole or not at all, a datagram to a port
/// that no socket holds vanishes, a socket bound to the any address receives on every
/// local address, a bind fails with address_in_use unless both sockets share the port,
/// and a bind to an address that the host does not hold fails with
/// address_not_available. A test can lose or duplicate datagrams, as a real network
/// may.
namespace driver::udp::model {
struct State;

/// @brief another host's socket on the modeled network: the device, or a foreign
/// sender. Any thread may call it.
class Peer {
    std::shared_ptr<State> net;
    asio::ip::udp::endpoint at;

public:
    Peer(std::shared_ptr<State> net, asio::ip::udp::endpoint at);

    /// @returns the endpoint that the peer sends from and receives on.
    [[nodiscard]] asio::ip::udp::endpoint endpoint() const;

    /// @brief sends data as one datagram to the endpoint.
    void send(const std::string &data, const asio::ip::udp::endpoint &to) const;

    /// @brief waits up to timeout for a datagram.
    /// @returns the datagram and its source, or nothing on timeout.
    [[nodiscard]] std::optional<std::pair<std::string, asio::ip::udp::endpoint>>
    receive(x::telem::TimeSpan timeout = x::telem::SECOND) const;

    /// @brief closes the peer's socket. Datagrams to its endpoint then vanish.
    void close() const;
};

/// @brief a modeled network. Copies share one network. Any thread may call it.
class Network {
    std::shared_ptr<State> state;

public:
    /// @brief starts a network whose host holds 127.0.0.1.
    Network();

    /// @brief gives the host under test another local address.
    void add(const asio::ip::address &address) const;

    /// @brief makes host resolve to address.
    void name(const std::string &host, const asio::ip::address &address) const;

    /// @brief binds a peer socket on another host.
    [[nodiscard]] Peer bind(const asio::ip::udp::endpoint &endpoint) const;

    /// @brief loses the next n datagrams that anyone sends.
    void lose(std::size_t n) const;

    /// @brief delivers each of the next n datagrams that anyone sends twice.
    void duplicate(std::size_t n) const;

    /// @brief fails the nth stack call from now, counting from 1, with ec. Every
    /// resolve, open, share, bind, join, receive, and send counts as one call.
    void fail_at(std::size_t n, std::error_code ec) const;

    /// @returns a factory for Config::stacks that makes stacks on this network.
    [[nodiscard]] Stacks stacks() const;
};
}
