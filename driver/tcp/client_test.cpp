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
#include "asio/ip/tcp.hpp"
#include "asio/read.hpp"
#include "asio/write.hpp"
#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/testutil/contract.h"
#include "driver/bus/transport.h"
#include "driver/tcp/client.h"

namespace driver::tcp {
namespace {
std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

std::string as_string(const std::span<const std::uint8_t> data) {
    return {reinterpret_cast<const char *>(data.data()), data.size()};
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
    const auto client = ASSERT_NIL_P(Client::open(this->props()));
    auto peer = this->accept();
    const auto before = x::telem::TimeStamp::now();
    asio::write(peer, asio::buffer(std::string("hello")));
    const auto chunk = ASSERT_NIL_P(client->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "hello");
    EXPECT_GE(chunk.time, before);
    EXPECT_LE(chunk.time, x::telem::TimeStamp::now());
}

TEST_F(ClientTest, AnswersAQueryInARoundTrip) {
    const auto client = ASSERT_NIL_P(Client::open(this->props()));
    auto peer = this->accept();
    ASSERT_NIL(client->write(bytes("*IDN?\n"), x::telem::SECOND));
    EXPECT_EQ(receive(peer, 6), "*IDN?\n");
    asio::write(peer, asio::buffer(std::string("SYNNAX,PSU\n")));
    const auto chunk = ASSERT_NIL_P(client->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "SYNNAX,PSU\n");
}

TEST_F(ClientTest, ReadsAfterAWriteFromAnotherThread) {
    const auto client = ASSERT_NIL_P(Client::open(this->props()));
    auto peer = this->accept();
    std::mutex mu;
    std::string got;
    std::atomic running = true;
    std::thread reader([&] {
        while (running) {
            {
                std::lock_guard lock(mu);
                got += as_string(
                    ASSERT_NIL_P(client->read(20 * x::telem::MILLISECOND)).data
                );
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        }
    });
    for (int i = 0; i < 5; i++) {
        {
            std::lock_guard lock(mu);
            ASSERT_NIL(client->write(bytes("Q?\n"), x::telem::SECOND));
        }
        ASSERT_EQ(receive(peer, 3), "Q?\n");
        asio::write(peer, asio::buffer(std::to_string(i)));
        ASSERT_EVENTUALLY_TRUE([&] {
            std::lock_guard lock(mu);
            return got.size() == static_cast<std::size_t>(i + 1);
        }());
    }
    running = false;
    reader.join();
    EXPECT_EQ(got, "01234");
}

TEST_F(ClientTest, ReturnsAnEmptyChunkWhenNothingArrivesBeforeTheTimeout) {
    const auto client = ASSERT_NIL_P(Client::open(this->props()));
    auto peer = this->accept();
    const x::telem::Stopwatch sw;
    const auto chunk = ASSERT_NIL_P(client->read(50 * x::telem::MILLISECOND));
    EXPECT_TRUE(chunk.data.empty());
    EXPECT_GE(sw.elapsed(), 50 * x::telem::MILLISECOND);
}

TEST_F(ClientTest, StaysClosedAfterThePeerDrops) {
    const auto client = ASSERT_NIL_P(Client::open(this->props()));
    this->accept().close();
    ASSERT_OCCURRED_AS_P(client->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    ASSERT_OCCURRED_AS_P(client->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
    ASSERT_OCCURRED_AS(
        client->write(bytes("ping"), x::telem::SECOND),
        transport::UNREACHABLE_ERROR
    );
    this->acceptor.non_blocking(true);
    std::error_code ec;
    this->acceptor.accept(ec);
    EXPECT_EQ(ec, asio::error::would_block);
}

TEST_F(ClientTest, FailsToOpenWhenNothingListens) {
    const auto p = this->props();
    this->acceptor.close();
    ASSERT_OCCURRED_AS_P(Client::open(p), transport::UNREACHABLE_ERROR);
}

TEST(Client, RejectsAMissingHostOrPort) {
    synnax::tcp::Properties p;
    p.port = 5025;
    ASSERT_OCCURRED_AS_P(Client::open(p), transport::CONFIG_ERROR);
    p.host = "127.0.0.1";
    p.port = 0;
    ASSERT_OCCURRED_AS_P(Client::open(p), transport::CONFIG_ERROR);
}

namespace {
/// @brief plays a TCP device with a listener on a free localhost port.
class LoopbackRig final : public bus::testutil::Rig {
    asio::io_context ctx;
    asio::ip::tcp::acceptor acceptor{
        ctx,
        asio::ip::tcp::endpoint(asio::ip::address_v4::loopback(), 0)
    };
    std::optional<asio::ip::tcp::socket> peer;

    asio::ip::tcp::socket &accepted() {
        if (!this->peer.has_value()) this->peer.emplace(this->acceptor.accept());
        return *this->peer;
    }

    static bus::Opener open(const std::uint16_t port) {
        synnax::tcp::Properties p;
        p.host = "127.0.0.1";
        p.port = port;
        return bus::opener<Client>(p);
    }

public:
    bus::Opener opener() override {
        return open(this->acceptor.local_endpoint().port());
    }

    bus::Opener absent() override {
        const asio::ip::tcp::acceptor probe(
            this->ctx,
            asio::ip::tcp::endpoint(asio::ip::address_v4::loopback(), 0)
        );
        return open(probe.local_endpoint().port());
    }

    void send(const std::string &data) override {
        asio::write(this->accepted(), asio::buffer(data));
    }

    std::string receive(const std::size_t n) override {
        auto &socket = this->accepted();
        std::string out;
        const auto deadline = x::telem::TimeStamp::now() + x::telem::SECOND;
        while (out.size() < n && x::telem::TimeStamp::now() < deadline) {
            if (socket.available() == 0) {
                std::this_thread::sleep_for(std::chrono::milliseconds(1));
                continue;
            }
            std::string chunk(socket.available(), '\0');
            chunk.resize(socket.read_some(asio::buffer(chunk)));
            out += chunk;
        }
        return out;
    }

    void hang_up() override {
        this->accepted().close();
        this->peer.reset();
    }

    void restore() override {}
};

const auto LOOPBACK = ::testing::Values(
    bus::testutil::Case{
        .name = "loopback",
        .rig = [] { return std::make_unique<LoopbackRig>(); },
    }
);
}

using bus::testutil::StreamContract;
using bus::testutil::TransportContract;
INSTANTIATE_TEST_SUITE_P(Tcp, TransportContract, LOOPBACK, bus::testutil::name);
INSTANTIATE_TEST_SUITE_P(Tcp, StreamContract, LOOPBACK, bus::testutil::name);
}
