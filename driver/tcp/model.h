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
#include <deque>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <string>
#include <system_error>

#include "asio/ip/tcp.hpp"

#include "x/cpp/telem/telem.h"

#include "driver/tcp/socket.h"

/// @brief an in-process model of the host's network stack that replaces the operating
/// system calls under a Client, after Boost.Beast's test::stream. Each connection is a
/// pair of byte queues. Its semantics follow tcp(7) and socket(7): a read returns the
/// bytes queued so far or waits for some, a read after the peer closes and the queue
/// drains returns end of file, a write after the peer closes fails with a reset, and a
/// call on a closed socket fails with a bad descriptor.
namespace driver::tcp::model {
struct Pipe;

/// @brief the device end of a modeled connection. Any thread may call it.
class Peer {
    std::shared_ptr<Pipe> pipe;

public:
    explicit Peer(std::shared_ptr<Pipe> pipe);

    /// @brief queues data for the client to read.
    void send(const std::string &data) const;

    /// @brief waits up to timeout for n bytes from the client.
    /// @returns the bytes that arrived, which are fewer than n on timeout.
    [[nodiscard]] std::string
    receive(std::size_t n, x::telem::TimeSpan timeout = x::telem::SECOND) const;

    /// @brief closes the device end. The client reads end of file once it drains the
    /// queued bytes, and its writes fail with connection_reset.
    void close() const;

    /// @returns true once the client closes its end.
    [[nodiscard]] bool closed() const;
};

/// @brief the shared state of a modeled network.
struct State;

/// @brief a modeled network. Copies share one network. Any thread may call it.
class Network {
    std::shared_ptr<State> state;

public:
    Network();

    /// @brief makes host resolve to address.
    void name(const std::string &host, const asio::ip::address &address) const;

    /// @brief makes resolving host take longer than any timeout, as when a DNS server
    /// does not answer.
    void hang(const std::string &host) const;

    /// @brief accepts connections on endpoint. Accept returns their device ends.
    void listen(const asio::ip::tcp::endpoint &endpoint) const;

    /// @brief stops accepting connections on endpoint, so attempts are refused.
    void unlisten(const asio::ip::tcp::endpoint &endpoint) const;

    /// @brief drops connection attempts to endpoint without an answer, as a firewall
    /// does, so they complete only when cancelled.
    void drop(const asio::ip::tcp::endpoint &endpoint) const;

    /// @brief fails the nth socket call from now, counting from 1, with ec. Every
    /// resolve, connect, configure, read, and write counts as one call.
    void fail_at(std::size_t n, std::error_code ec) const;

    /// @brief caps the bytes that one read returns on later connections.
    void read_size(std::size_t n) const;

    /// @brief caps the bytes that one write takes on later connections, so a writer
    /// must loop.
    void write_size(std::size_t n) const;

    /// @brief waits up to timeout for a connection on endpoint.
    /// @returns the connection's device end, or nullptr on timeout.
    [[nodiscard]] std::shared_ptr<Peer> accept(
        const asio::ip::tcp::endpoint &endpoint,
        x::telem::TimeSpan timeout = x::telem::SECOND
    ) const;

    /// @returns the number of sockets that turned on keepalive probes.
    [[nodiscard]] std::size_t configured() const;

    /// @returns a factory for Config::sockets that makes sockets on this network.
    [[nodiscard]] Sockets sockets() const;
};
}
