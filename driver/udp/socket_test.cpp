// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <span>
#include <string>
#include <string_view>
#include <thread>
#include <vector>

#include "asio/buffer.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/address.hpp"
#include "asio/ip/udp.hpp"
#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/transport/errors.h"
#include "driver/transport/transport.h"
#include "driver/udp/socket.h"

namespace driver::udp {
namespace {
const auto TIMEOUT = x::telem::SECOND;
const auto SHORT = 20 * x::telem::MILLISECOND;

/// @brief Peer is a plain UDP socket on a loopback port that talks to a Socket.
struct Peer {
    asio::io_context ctx;
    asio::ip::udp::socket socket;
    /// @brief from is the source of the last datagram that the peer received.
    asio::ip::udp::endpoint from;

    explicit Peer(const std::string &address = "127.0.0.1"):
        socket(ctx, asio::ip::udp::endpoint(asio::ip::make_address(address), 0)) {}

    [[nodiscard]] std::uint16_t port() const {
        return this->socket.local_endpoint().port();
    }

    /// @brief returns the text of the next datagram.
    std::string receive() {
        std::array<char, 64> got{};
        const auto n = this->socket.receive_from(asio::buffer(got), this->from);
        return {got.data(), n};
    }

    void send(const std::string_view text, const asio::ip::udp::endpoint &to) {
        this->socket.send_to(asio::buffer(text), to);
    }
};

/// @brief returns a loopback port that no socket holds now.
std::uint16_t free_port() {
    Peer p;
    return p.port();
}

asio::ip::udp::endpoint loopback(const std::uint16_t port) {
    return {asio::ip::make_address("127.0.0.1"), port};
}

std::span<const std::uint8_t> bytes(const std::string_view text) {
    return {reinterpret_cast<const std::uint8_t *>(text.data()), text.size()};
}

std::string text(const transport::Chunk &chunk) {
    return {chunk.data.begin(), chunk.data.end()};
}

/// @brief reports whether an interface holds address.
bool held(const std::string &address) {
    asio::io_context ctx;
    asio::ip::udp::socket s(ctx);
    const asio::ip::udp::endpoint ep(asio::ip::make_address(address), 0);
    std::error_code ec;
    if (s.open(ep.protocol(), ec)) return false;
    return !s.bind(ep, ec);
}

/// @brief opens a socket on loopback that sends to and receives from peer.
std::unique_ptr<Socket> open_to(const Peer &peer) {
    return ASSERT_NIL_P(
        Socket::open(
            {.local_address = "127.0.0.1",
             .remote_host = "127.0.0.1",
             .remote_port = peer.port()},
            TIMEOUT
        )
    );
}
}

/// @brief it should send datagrams to the remote host and read the ones it sends back.
TEST(Socket, SendAndReceive) {
    Peer peer;
    auto s = open_to(peer);
    ASSERT_NIL(s->write(bytes("ping"), TIMEOUT));
    EXPECT_EQ(peer.receive(), "ping");
    peer.send("pong", peer.from);
    const auto chunk = ASSERT_NIL_P(s->read(TIMEOUT));
    EXPECT_EQ(text(chunk), "pong");
    EXPECT_GT(chunk.time, x::telem::TimeStamp(0));
}

/// @brief it should give each datagram in its own read.
TEST(Socket, ReadOneDatagramAtATime) {
    Peer peer;
    auto s = open_to(peer);
    ASSERT_NIL(s->write(bytes("hello"), TIMEOUT));
    peer.receive();
    peer.send("a", peer.from);
    peer.send("bc", peer.from);
    EXPECT_EQ(text(ASSERT_NIL_P(s->read(TIMEOUT))), "a");
    EXPECT_EQ(text(ASSERT_NIL_P(s->read(TIMEOUT))), "bc");
}

/// @brief it should return an empty chunk when no datagram arrives before the timeout.
TEST(Socket, ReadNothingBeforeTimeout) {
    Peer peer;
    auto s = open_to(peer);
    EXPECT_TRUE(ASSERT_NIL_P(s->read(SHORT)).data.empty());
}

/// @brief it should receive from any port of the remote host.
TEST(Socket, ReceiveFromAnyPortOfRemoteHost) {
    Peer peer;
    Peer moved;
    auto s = open_to(peer);
    ASSERT_NIL(s->write(bytes("hello"), TIMEOUT));
    peer.receive();
    moved.send("moved", peer.from);
    EXPECT_EQ(text(ASSERT_NIL_P(s->read(TIMEOUT))), "moved");
}

/// @brief it should drop datagrams from hosts other than the remote host.
TEST(Socket, IgnoreOtherHosts) {
    if (!held("::1")) GTEST_SKIP() << "no interface holds ::1";
    const auto port = free_port();
    auto any = ASSERT_NIL_P(
        Socket::open({.local_address = "::", .port = port}, TIMEOUT)
    );
    Peer probe;
    probe.send("probe", loopback(port));
    if (ASSERT_NIL_P(any->read(SHORT)).data.empty())
        GTEST_SKIP() << "IPv6 sockets on this host do not take IPv4 datagrams";
    any.reset();
    Peer peer("::1");
    auto s = ASSERT_NIL_P(
        Socket::open(
            {.local_address = "::",
             .port = port,
             .remote_host = "::1",
             .remote_port = peer.port()},
            TIMEOUT
        )
    );
    Peer other;
    other.send("intruder", loopback(port));
    EXPECT_TRUE(ASSERT_NIL_P(s->read(SHORT)).data.empty());
    ASSERT_NIL(s->write(bytes("hello"), TIMEOUT));
    peer.receive();
    peer.send("reply", peer.from);
    EXPECT_EQ(text(ASSERT_NIL_P(s->read(TIMEOUT))), "reply");
}

/// @brief it should read from any sender when there is no remote host, and refuse to
/// write.
TEST(Socket, ReceiveFromAnySender) {
    const auto port = free_port();
    auto s = ASSERT_NIL_P(
        Socket::open({.local_address = "127.0.0.1", .port = port}, TIMEOUT)
    );
    Peer peer;
    peer.send("anyone", loopback(port));
    EXPECT_EQ(text(ASSERT_NIL_P(s->read(TIMEOUT))), "anyone");
    const auto err = s->write(bytes("x"), TIMEOUT);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_EQ(err.data, "no remote host is set to send to");
}

/// @brief it should keep reading and writing after a send to a port with no listener.
TEST(Socket, IgnorePortUnreachable) {
    auto s = ASSERT_NIL_P(
        Socket::open(
            {.local_address = "127.0.0.1",
             .remote_host = "127.0.0.1",
             .remote_port = free_port()},
            TIMEOUT
        )
    );
    ASSERT_NIL(s->write(bytes("x"), TIMEOUT));
    EXPECT_TRUE(ASSERT_NIL_P(s->read(SHORT)).data.empty());
    ASSERT_NIL(s->write(bytes("y"), TIMEOUT));
    EXPECT_TRUE(ASSERT_NIL_P(s->read(SHORT)).data.empty());
}

/// @brief it should skip empty datagrams.
TEST(Socket, SkipEmptyDatagrams) {
    Peer peer;
    auto s = open_to(peer);
    ASSERT_NIL(s->write(bytes("hello"), TIMEOUT));
    peer.receive();
    peer.send("", peer.from);
    peer.send("x", peer.from);
    EXPECT_EQ(text(ASSERT_NIL_P(s->read(TIMEOUT))), "x");
}

/// @brief it should return at the timeout while empty datagrams flood the socket.
TEST(Socket, ReturnAtTimeoutUnderFlood) {
    Peer peer;
    Peer source;
    auto s = open_to(peer);
    ASSERT_NIL(s->write(bytes("hello"), TIMEOUT));
    peer.receive();
    const auto to = peer.from;
    std::atomic done = false;
    std::atomic capped = false;
    std::thread flood([&] {
        const auto cap = std::chrono::steady_clock::now() + std::chrono::seconds(2);
        while (!done) {
            if (std::chrono::steady_clock::now() > cap) {
                capped = true;
                return;
            }
            source.send("", to);
        }
    });
    EXPECT_TRUE(ASSERT_NIL_P(s->read(SHORT)).data.empty());
    done = true;
    flood.join();
    EXPECT_FALSE(capped);
}

/// @brief it should refuse a datagram larger than UDP allows.
TEST(Socket, RefuseOversizedDatagram) {
    Peer peer;
    auto s = open_to(peer);
    const std::vector<std::uint8_t> big(70000);
    const auto err = s->write(big, TIMEOUT);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to send to 127.0.0.1:" + std::to_string(peer.port()) +
            ": 70000 bytes exceed the 65507 bytes of a datagram"
    );
}

/// @brief it should fail when no interface holds the local address.
TEST(Socket, FailWhenNoInterfaceHoldsAddress) {
    const Address addr{.local_address = "192.0.2.1"};
    const auto err = Socket::open(addr, TIMEOUT).second;
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(err.data.starts_with("failed to bind UDP port 192.0.2.1:0: "))
        << err.data;
}

/// @brief it should refuse a port that another socket holds.
TEST(Socket, RefusePortInUse) {
    Peer peer;
    const Address addr{.local_address = "127.0.0.1", .port = peer.port()};
    const auto err = Socket::open(addr, TIMEOUT).second;
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    const auto prefix = "failed to bind UDP port 127.0.0.1:" +
                        std::to_string(peer.port()) + ": ";
    EXPECT_TRUE(err.data.starts_with(prefix)) << err.data;
}

/// @brief it should fail when the remote host does not resolve.
TEST(Socket, FailWhenHostDoesNotResolve) {
    const Address addr{.remote_host = "host.invalid", .remote_port = 80};
    const auto err = Socket::open(addr, TIMEOUT).second;
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(err.data.starts_with("failed to resolve host.invalid: ")) << err.data;
}

/// @brief it should refuse an invalid address.
TEST(Socket, RefuseInvalidAddress) {
    const auto assert_refuses = [](const Address &addr, const std::string &message) {
        const auto err = Socket::open(addr, TIMEOUT).second;
        ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
        EXPECT_EQ(err.data, message);
    };
    assert_refuses(
        {.remote_host = "127.0.0.1"},
        "a remote port is required with a remote host"
    );
    assert_refuses({.remote_port = 80}, "a remote host is required with a remote port");
    assert_refuses({.local_address = "nope"}, "'nope' is not an IP address");
    assert_refuses(
        {.local_address = "127.0.0.1", .remote_host = "::1", .remote_port = 80},
        "remote host ::1 has no address of the IP version of local address 127.0.0.1"
    );
}
}
