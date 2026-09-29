// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <span>
#include <string>
#include <thread>

#include "asio/buffer.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/tcp.hpp"
#include "asio/read.hpp"
#include "asio/write.hpp"
#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/tcp/client.h"

namespace driver::tcp {
namespace {
std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

std::string as_string(const std::span<const std::uint8_t> data) {
    return {reinterpret_cast<const char *>(data.data()), data.size()};
}

const Config FAST{
    .connect_timeout = x::telem::SECOND,
    .min_backoff = 10 * x::telem::MILLISECOND,
    .max_backoff = 100 * x::telem::MILLISECOND,
};

/// @brief reads from the client until bytes arrive or one second of reads passes.
std::pair<std::string, x::errors::Error> read_eventually(Client &client) {
    for (int i = 0; i < 100; i++) {
        auto [chunk, err] = client.read(10 * x::telem::MILLISECOND);
        if (err || !chunk.data.empty()) return {as_string(chunk.data), err};
    }
    return {"", x::errors::NIL};
}
}

/// @brief plays the device: a listener on a free localhost port.
class ClientTest : public ::testing::Test {
protected:
    asio::io_context ctx;
    asio::ip::tcp::acceptor acceptor{
        ctx,
        asio::ip::tcp::endpoint(asio::ip::address_v4::loopback(), 0)
    };

    [[nodiscard]] synnax::tcp::Properties props() const {
        synnax::tcp::Properties p;
        p.host = "127.0.0.1";
        p.port = this->acceptor.local_endpoint().port();
        return p;
    }

    asio::ip::tcp::socket accept() { return this->acceptor.accept(); }

    static std::string receive(asio::ip::tcp::socket &peer, const std::size_t n) {
        std::string out(n, '\0');
        asio::read(peer, asio::buffer(out));
        return out;
    }
};

TEST_F(ClientTest, ReadsWhatThePeerSends) {
    const auto client = ASSERT_NIL_P(Client::open(this->props(), FAST));
    auto peer = this->accept();
    const auto before = x::telem::TimeStamp::now();
    asio::write(peer, asio::buffer(std::string("hello")));
    const auto chunk = ASSERT_NIL_P(client->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "hello");
    EXPECT_GE(chunk.time, before);
    EXPECT_LE(chunk.time, x::telem::TimeStamp::now());
}

TEST_F(ClientTest, AnswersAQueryInARoundTrip) {
    const auto client = ASSERT_NIL_P(Client::open(this->props(), FAST));
    auto peer = this->accept();
    ASSERT_NIL(client->write(bytes("*IDN?\n"), x::telem::SECOND));
    EXPECT_EQ(receive(peer, 6), "*IDN?\n");
    asio::write(peer, asio::buffer(std::string("SYNNAX,PSU\n")));
    const auto chunk = ASSERT_NIL_P(client->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "SYNNAX,PSU\n");
}

TEST_F(ClientTest, ReturnsAnEmptyChunkWhenNothingArrivesBeforeTheTimeout) {
    const auto client = ASSERT_NIL_P(Client::open(this->props(), FAST));
    auto peer = this->accept();
    const x::telem::Stopwatch sw;
    const auto chunk = ASSERT_NIL_P(client->read(50 * x::telem::MILLISECOND));
    EXPECT_TRUE(chunk.data.empty());
    EXPECT_GE(sw.elapsed(), 50 * x::telem::MILLISECOND);
}

TEST_F(ClientTest, ReconnectsAfterThePeerClosesTheSocket) {
    const auto client = ASSERT_NIL_P(Client::open(this->props(), FAST));
    {
        auto peer = this->accept();
        asio::write(peer, asio::buffer(std::string("first")));
        ASSERT_EQ(
            as_string(ASSERT_NIL_P(client->read(x::telem::SECOND)).data),
            "first"
        );
        peer.close();
    }
    ASSERT_OCCURRED_AS_P(client->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
    std::thread device([this] {
        auto peer = this->accept();
        asio::write(peer, asio::buffer(std::string("second")));
        EXPECT_EQ(receive(peer, 4), "ping");
    });
    const auto [data, err] = read_eventually(*client);
    ASSERT_NIL(err);
    EXPECT_EQ(data, "second");
    ASSERT_NIL(client->write(bytes("ping"), x::telem::SECOND));
    device.join();
}

TEST_F(ClientTest, WaitsOutTheBackoffBeforeReconnecting) {
    Config config = FAST;
    config.min_backoff = 200 * x::telem::MILLISECOND;
    const auto client = ASSERT_NIL_P(Client::open(this->props(), config));
    this->accept().close();
    this->acceptor.close();
    ASSERT_OCCURRED_AS_P(client->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
    const auto waiting = ASSERT_NIL_P(client->read(10 * x::telem::MILLISECOND));
    EXPECT_TRUE(waiting.data.empty());
    ASSERT_OCCURRED_AS(
        client->write(bytes("ping"), x::telem::SECOND),
        transport::UNREACHABLE_ERROR
    );
    const x::telem::Stopwatch sw;
    ASSERT_OCCURRED_AS_P(client->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
    EXPECT_GE(sw.elapsed(), 150 * x::telem::MILLISECOND);
}

TEST_F(ClientTest, FailsToOpenWhenNothingListens) {
    const auto p = this->props();
    this->acceptor.close();
    ASSERT_OCCURRED_AS_P(Client::open(p, FAST), transport::UNREACHABLE_ERROR);
}

TEST(Client, RejectsAMissingHostOrPort) {
    synnax::tcp::Properties p;
    p.port = 5025;
    ASSERT_OCCURRED_AS_P(Client::open(p), transport::CONFIG_ERROR);
    p.host = "127.0.0.1";
    p.port = 0;
    ASSERT_OCCURRED_AS_P(Client::open(p), transport::CONFIG_ERROR);
}
}
