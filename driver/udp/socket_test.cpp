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
#include <mutex>
#include <span>
#include <string>
#include <thread>

#include "asio/buffer.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/multicast.hpp"
#include "asio/ip/udp.hpp"
#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

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
    this->send("still here", local(p.port));
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

TEST_F(SocketTest, FailsToBindAPortInUse) {
    synnax::udp::Properties p;
    p.port = this->free_port();
    const auto first = ASSERT_NIL_P(Socket::open(p));
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
        GTEST_SKIP() << "the host refused to join a multicast group: " << err;
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
    if (ec) GTEST_SKIP() << "the host refused to send to a multicast group: " << ec;
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
}
}
