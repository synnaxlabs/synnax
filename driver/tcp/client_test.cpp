// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <cstdint>
#include <span>
#include <string>
#include <string_view>
#include <vector>

#include "asio/connect.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/tcp.hpp"
#include "asio/read.hpp"
#include "asio/write.hpp"
#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/tcp/client.h"
#include "driver/tcp/keepalive.h"
#include "driver/transport/errors.h"

namespace driver::tcp {
namespace {
const auto TIMEOUT = x::telem::SECOND;

/// @brief Server listens on a loopback port, and holds the peer of the connection that
/// it accepts.
struct Server {
    asio::io_context ctx;
    asio::ip::tcp::acceptor acceptor{
        ctx,
        asio::ip::tcp::endpoint(asio::ip::make_address("127.0.0.1"), 0)
    };
    asio::ip::tcp::socket peer{ctx};

    [[nodiscard]] std::uint16_t port() const {
        return this->acceptor.local_endpoint().port();
    }

    /// @brief connects a client to the server, and accepts the connection as peer.
    std::unique_ptr<Client> connect() {
        auto client = ASSERT_NIL_P(Client::connect("127.0.0.1", this->port(), TIMEOUT));
        this->acceptor.accept(this->peer);
        return client;
    }
};

/// @brief returns the bytes of text.
std::span<const std::uint8_t> bytes(const std::string_view text) {
    return {reinterpret_cast<const std::uint8_t *>(text.data()), text.size()};
}

/// @brief returns the text of the bytes of chunk.
std::string text(const transport::Chunk &chunk) {
    return {chunk.data.begin(), chunk.data.end()};
}
}

/// @brief it should read the bytes that the peer sends, and write bytes to the peer.
TEST(Client, ReadAndWrite) {
    Server s;
    auto client = s.connect();
    asio::write(s.peer, asio::buffer(std::string_view("hello")));
    const auto chunk = ASSERT_NIL_P(client->read(TIMEOUT));
    EXPECT_EQ(text(chunk), "hello");
    EXPECT_GT(chunk.time, x::telem::TimeStamp(0));
    ASSERT_NIL(client->write(bytes("world"), TIMEOUT));
    std::array<char, 5> got{};
    asio::read(s.peer, asio::buffer(got));
    EXPECT_EQ(std::string_view(got.data(), got.size()), "world");
}

/// @brief it should return an empty chunk when no bytes arrive before the timeout.
TEST(Client, ReadNothingBeforeTimeout) {
    Server s;
    auto client = s.connect();
    const auto chunk = ASSERT_NIL_P(client->read(10 * x::telem::MILLISECOND));
    EXPECT_TRUE(chunk.data.empty());
    asio::write(s.peer, asio::buffer(std::string_view("late")));
    EXPECT_EQ(text(ASSERT_NIL_P(client->read(TIMEOUT))), "late");
}

/// @brief it should fail when the peer closes the connection, and keep failing.
TEST(Client, FailWhenPeerCloses) {
    Server s;
    auto client = s.connect();
    s.peer.close();
    const auto port = std::to_string(s.port());
    const auto [chunk, err] = client->read(TIMEOUT);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "failed to read from 127.0.0.1:" + port + ": End of file");
    ASSERT_OCCURRED_AS_P(client->read(TIMEOUT), transport::UNREACHABLE_ERROR);
    const auto written = client->write(bytes("x"), TIMEOUT);
    ASSERT_OCCURRED_AS(written, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(written.data, err.data);
}

/// @brief it should fail a write that the peer does not take before the timeout, and
/// keep failing.
TEST(Client, FailWriteAtTimeout) {
    Server s;
    auto client = s.connect();
    const std::vector<std::uint8_t> big(std::size_t{64} * 1024 * 1024);
    const auto err = client->write(big, 50 * x::telem::MILLISECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to write to 127.0.0.1:" + std::to_string(s.port()) +
            ": Connection timed out"
    );
    EXPECT_EQ(client->write(bytes("x"), TIMEOUT).data, err.data);
}

/// @brief it should fail to connect to a port with no listener.
TEST(Client, FailWhenRefused) {
    std::uint16_t port = 0;
    {
        Server s;
        port = s.port();
    }
    const auto err = Client::connect("127.0.0.1", port, TIMEOUT).second;
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to connect to 127.0.0.1:" + std::to_string(port) +
            ": Connection refused"
    );
}

/// @brief it should fail to connect to a host that does not resolve.
TEST(Client, FailWhenHostDoesNotResolve) {
    const auto err = Client::connect("host.invalid", 80, TIMEOUT).second;
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(err.data.starts_with("failed to resolve host.invalid: ")) << err.data;
}

/// @brief it should refuse an empty host and port 0.
TEST(Client, RefuseInvalidAddress) {
    const auto host = Client::connect("", 80, TIMEOUT).second;
    ASSERT_OCCURRED_AS(host, transport::CONFIG_ERROR);
    EXPECT_EQ(host.data, "the host is empty");
    const auto port = Client::connect("127.0.0.1", 0, TIMEOUT).second;
    ASSERT_OCCURRED_AS(port, transport::CONFIG_ERROR);
    EXPECT_EQ(port.data, "the port is 0");
}

/// @brief it should turn on keepalive probes on a socket.
TEST(Keepalive, Enable) {
    asio::io_context ctx;
    asio::ip::tcp::socket socket(ctx, asio::ip::tcp::v4());
    ASSERT_FALSE(keepalive::enable(socket.native_handle()));
    asio::socket_base::keep_alive on;
    socket.get_option(on);
    EXPECT_TRUE(on.value());
}
}
