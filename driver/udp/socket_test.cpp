// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <cstdint>
#include <memory>
#include <mutex>
#include <optional>
#include <span>
#include <string>
#include <thread>

#include "asio/buffer.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/multicast.hpp"
#include "asio/ip/udp.hpp"
#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/testutil/contract.h"
#include "driver/bus/transport.h"
#include "driver/udp/socket.h"

namespace driver::udp {
namespace {
std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

std::string as_string(const std::span<const std::uint8_t> data) {
    return {reinterpret_cast<const char *>(data.data()), data.size()};
}
}

/// @brief plays the device with a socket on a free localhost port.
class SocketTest : public ::testing::Test {
protected:
    asio::io_context ctx;
    asio::ip::udp::socket peer{
        ctx,
        asio::ip::udp::endpoint(asio::ip::address_v4::loopback(), 0)
    };

    /// @returns a local port that no socket holds.
    std::uint16_t free_port() {
        asio::ip::udp::socket probe(
            this->ctx,
            asio::ip::udp::endpoint(asio::ip::udp::v4(), 0)
        );
        return probe.local_endpoint().port();
    }

    void send(const std::string &data, const asio::ip::udp::endpoint &to) {
        this->peer.send_to(asio::buffer(data), to);
    }

    static asio::ip::udp::endpoint local(const std::uint16_t port) {
        return {asio::ip::address_v4::loopback(), port};
    }
};

TEST_F(SocketTest, ReceivesADatagram) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    const auto before = x::telem::TimeStamp::now();
    this->send("hello", local(p.port));
    const auto chunk = ASSERT_NIL_P(socket->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "hello");
    EXPECT_GE(chunk.time, before);
    EXPECT_LE(chunk.time, x::telem::TimeStamp::now());
}

TEST_F(SocketTest, ReturnsOneDatagramPerRead) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    this->send("first", local(p.port));
    this->send("second", local(p.port));
    EXPECT_EQ(as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data), "first");
    EXPECT_EQ(as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data), "second");
}

TEST_F(SocketTest, ReturnsAnEmptyChunkWhenNothingArrivesBeforeTheTimeout) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    const x::telem::Stopwatch sw;
    const auto chunk = ASSERT_NIL_P(socket->read(50 * x::telem::MILLISECOND));
    EXPECT_TRUE(chunk.data.empty());
    EXPECT_GE(sw.elapsed(), 50 * x::telem::MILLISECOND);
}

TEST_F(SocketTest, AnswersAQueryInARoundTrip) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    p.remote_host = "127.0.0.1";
    p.remote_port = this->peer.local_endpoint().port();
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    ASSERT_NIL(socket->write(bytes("status?"), x::telem::SECOND));
    std::string query(64, '\0');
    asio::ip::udp::endpoint sender;
    query.resize(this->peer.receive_from(asio::buffer(query), sender));
    EXPECT_EQ(query, "status?");
    EXPECT_EQ(sender.port(), p.port);
    this->send("ok", sender);
    EXPECT_EQ(as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data), "ok");
}

TEST_F(SocketTest, ReadsAfterAWriteFromAnotherThread) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    p.remote_host = "127.0.0.1";
    p.remote_port = this->peer.local_endpoint().port();
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    std::mutex mu;
    std::string got;
    std::atomic running = true;
    std::thread reader([&] {
        while (running) {
            {
                std::lock_guard lock(mu);
                got += as_string(
                    ASSERT_NIL_P(socket->read(20 * x::telem::MILLISECOND)).data
                );
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        }
    });
    for (int i = 0; i < 5; i++) {
        {
            std::lock_guard lock(mu);
            ASSERT_NIL(socket->write(bytes("Q?"), x::telem::SECOND));
        }
        std::string query(16, '\0');
        asio::ip::udp::endpoint sender;
        query.resize(this->peer.receive_from(asio::buffer(query), sender));
        ASSERT_EQ(query, "Q?");
        this->send(std::to_string(i), sender);
        ASSERT_EVENTUALLY_TRUE([&] {
            std::lock_guard lock(mu);
            return got.size() == static_cast<std::size_t>(i + 1);
        }());
    }
    running = false;
    reader.join();
    EXPECT_EQ(got, "01234");
}

TEST_F(SocketTest, KeepsReceivingAfterTheRemoteRefusesADatagram) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    p.remote_host = "127.0.0.1";
    p.remote_port = this->free_port();
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    ASSERT_NIL(socket->write(bytes("nobody"), x::telem::SECOND));
    asio::ip::udp::socket remote(this->ctx, local(p.remote_port));
    remote.send_to(asio::buffer(std::string("still here")), local(p.port));
    EXPECT_EQ(
        as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data),
        "still here"
    );
}

TEST_F(SocketTest, RejectsAWriteWithoutARemote) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    ASSERT_OCCURRED_AS(
        socket->write(bytes("x"), x::telem::SECOND),
        transport::CONFIG_ERROR
    );
}

TEST_F(SocketTest, RejectsAPortInUse) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    const auto first = ASSERT_NIL_P(Socket::open(p));
    ASSERT_OCCURRED_AS_P(Socket::open(p), transport::CONFIG_ERROR);
}

TEST_F(SocketTest, DropsDatagramsFromSendersOtherThanTheRemote) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    p.remote_host = "127.0.0.1";
    p.remote_port = this->peer.local_endpoint().port();
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    asio::ip::udp::socket foreign(this->ctx, local(0));
    foreign.send_to(asio::buffer(std::string("spoofed")), local(p.port));
    this->send("genuine", local(p.port));
    EXPECT_EQ(as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data), "genuine");
}

TEST_F(SocketTest, BindsTheLocalAddress) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    p.local_address = "127.0.0.1";
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    this->send("on loopback", local(p.port));
    EXPECT_EQ(
        as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data),
        "on loopback"
    );
}

TEST_F(SocketTest, FailsToBindAnAddressNoInterfaceHolds) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    p.local_address = "192.0.2.1";
    ASSERT_OCCURRED_AS_P(Socket::open(p), transport::UNREACHABLE_ERROR);
}

TEST_F(SocketTest, FailsToReadAfterClose) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    const auto socket = ASSERT_NIL_P(Socket::open(p));
    socket->close();
    ASSERT_OCCURRED_AS_P(socket->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
}

TEST_F(SocketTest, ReceivesFromAMulticastGroup) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    p.multicast_group = "239.255.43.21";
    auto [socket, err] = Socket::open(p);
    if (err.matches(transport::UNREACHABLE_ERROR))
        SKIP_MISSING_CAPABILITY("the host refused to join a multicast group: " << err);
    ASSERT_NIL(err);
    asio::ip::udp::socket sender(this->ctx, asio::ip::udp::v4());
    sender.set_option(asio::ip::multicast::enable_loopback(true));
    std::error_code ec;
    sender.send_to(
        asio::buffer(std::string("to the group")),
        {asio::ip::make_address(p.multicast_group), p.port},
        0,
        ec
    );
    if (ec)
        SKIP_MISSING_CAPABILITY(
            "the host refused to send to a multicast group: " << ec.message()
        );
    EXPECT_EQ(
        as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data),
        "to the group"
    );
}

TEST(Socket, RejectsInvalidProperties) {
    synnax::udp::Properties p;
    p.multicast_group = "10.0.0.1";
    ASSERT_OCCURRED_AS_P(Socket::open(p), transport::CONFIG_ERROR);
    p.multicast_group = "not an address";
    ASSERT_OCCURRED_AS_P(Socket::open(p), transport::CONFIG_ERROR);
    p.multicast_group = "";
    p.remote_host = "127.0.0.1";
    ASSERT_OCCURRED_AS_P(Socket::open(p), transport::CONFIG_ERROR);
    p.remote_host = "";
    p.remote_port = 5000;
    ASSERT_OCCURRED_AS_P(Socket::open(p), transport::CONFIG_ERROR);
    p.remote_port = 0;
    p.local_address = "not an address";
    ASSERT_OCCURRED_AS_P(Socket::open(p), transport::CONFIG_ERROR);
}

namespace {
/// @brief plays a UDP device with a socket on a free localhost port.
class LoopbackRig final : public bus::testutil::Rig {
    asio::io_context ctx;
    std::optional<asio::ip::udp::socket> peer;
    std::uint16_t peer_port;
    std::uint16_t port;

    [[nodiscard]] synnax::udp::Properties props() const {
        synnax::udp::Properties p;
        p.port = this->port;
        p.remote_host = "127.0.0.1";
        p.remote_port = this->peer_port;
        return p;
    }

    static asio::ip::udp::endpoint local(const std::uint16_t port) {
        return {asio::ip::address_v4::loopback(), port};
    }

public:
    LoopbackRig() {
        this->peer.emplace(this->ctx, local(0));
        this->peer_port = this->peer->local_endpoint().port();
        const asio::ip::udp::socket probe(this->ctx, local(0));
        this->port = probe.local_endpoint().port();
    }

    bus::Opener opener() override { return bus::opener<Socket>(this->props()); }

    bus::Opener absent() override {
        auto p = this->props();
        p.local_address = "192.0.2.1";
        return bus::opener<Socket>(p);
    }

    void send(const std::string &data) override {
        this->peer->send_to(asio::buffer(data), local(this->port));
    }

    std::string receive(const std::size_t n) override {
        std::string out;
        const auto deadline = x::telem::TimeStamp::now() + x::telem::SECOND;
        while (out.size() < n && x::telem::TimeStamp::now() < deadline) {
            if (this->peer->available() == 0) {
                std::this_thread::sleep_for(std::chrono::milliseconds(1));
                continue;
            }
            std::string datagram(this->peer->available(), '\0');
            datagram.resize(this->peer->receive(asio::buffer(datagram)));
            out += datagram;
        }
        return out;
    }

    void hang_up() override { this->peer.reset(); }

    void restore() override { this->peer.emplace(this->ctx, local(this->peer_port)); }
};

const auto LOOPBACK = ::testing::Values(
    bus::testutil::Case{
        .name = "loopback",
        .rig = [] { return std::make_unique<LoopbackRig>(); },
    }
);
}

using bus::testutil::TransportContract;
INSTANTIATE_TEST_SUITE_P(Udp, TransportContract, LOOPBACK, bus::testutil::name);
}
