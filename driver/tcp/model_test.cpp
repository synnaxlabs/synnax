// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <memory>
#include <span>
#include <string>
#include <system_error>
#include <vector>

#include "asio/error.hpp"
#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/testutil/contract.h"
#include "driver/bus/transport.h"
#include "driver/tcp/client.h"
#include "driver/tcp/model.h"

namespace driver::tcp {
namespace {
std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

std::string as_string(const std::span<const std::uint8_t> data) {
    return {reinterpret_cast<const char *>(data.data()), data.size()};
}

const asio::ip::tcp::endpoint DEVICE{asio::ip::make_address("10.0.0.2"), 502};
}

/// @brief drives a client over a modeled network with one device listening.
class ModelTest : public ::testing::Test {
protected:
    model::Network network;

    void SetUp() override {
        this->network.name("plc.local", DEVICE.address());
        this->network.listen(DEVICE);
    }

    [[nodiscard]] static synnax::tcp::Properties props() {
        synnax::tcp::Properties p;
        p.host = "plc.local";
        p.port = DEVICE.port();
        return p;
    }

    [[nodiscard]] Config
    config(const x::telem::TimeSpan connect_timeout = x::telem::SECOND) const {
        return Config{
            .connect_timeout = connect_timeout,
            .sockets = this->network.sockets(),
        };
    }

    [[nodiscard]] std::pair<std::unique_ptr<Client>, std::shared_ptr<model::Peer>>
    connect() const {
        auto [client, err] = Client::open(props(), this->config());
        EXPECT_FALSE(err) << err;
        return {std::move(client), this->network.accept(DEVICE)};
    }
};

TEST_F(ModelTest, CompletesAWriteThatTheSocketTakesInPieces) {
    this->network.write_size(3);
    const auto [client, peer] = this->connect();
    ASSERT_NIL(client->write(bytes("MEAS:VOLT?\n"), x::telem::SECOND));
    EXPECT_EQ(peer->receive(11), "MEAS:VOLT?\n");
}

TEST_F(ModelTest, DeliversCappedReadsInOrder) {
    this->network.read_size(2);
    const auto [client, peer] = this->connect();
    peer->send("abcdefg");
    std::string got;
    while (got.size() < 7) {
        const auto chunk = ASSERT_NIL_P(client->read(x::telem::SECOND));
        ASSERT_FALSE(chunk.data.empty());
        ASSERT_LE(chunk.data.size(), 2);
        got += as_string(chunk.data);
    }
    EXPECT_EQ(got, "abcdefg");
}

TEST_F(ModelTest, ReturnsAnEmptyChunkWhileThePeerIsSilent) {
    const auto [client, peer] = this->connect();
    const x::telem::Stopwatch sw;
    const auto chunk = ASSERT_NIL_P(client->read(50 * x::telem::MILLISECOND));
    EXPECT_TRUE(chunk.data.empty());
    EXPECT_GE(sw.elapsed(), 50 * x::telem::MILLISECOND);
    peer->send("late");
    EXPECT_EQ(as_string(ASSERT_NIL_P(client->read(x::telem::SECOND)).data), "late");
}

TEST_F(ModelTest, TurnsOnKeepaliveForEveryConnection) {
    const auto [client, peer] = this->connect();
    EXPECT_EQ(this->network.configured(), 1);
}

TEST_F(ModelTest, TimesOutResolvingAHostWhoseNameServerHangs) {
    this->network.hang("plc.local");
    const x::telem::Stopwatch sw;
    ASSERT_OCCURRED_AS_P(
        Client::open(props(), this->config(100 * x::telem::MILLISECOND)),
        transport::UNREACHABLE_ERROR
    );
    EXPECT_LT(sw.elapsed(), x::telem::SECOND);
}

TEST_F(ModelTest, TimesOutConnectingToAPeerThatDropsTheAttempt) {
    this->network.unlisten(DEVICE);
    this->network.drop(DEVICE);
    const x::telem::Stopwatch sw;
    const auto [client, err] = Client::open(
        props(),
        this->config(100 * x::telem::MILLISECOND)
    );
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to connect to plc.local:502: " +
            std::make_error_code(std::errc::timed_out).message()
    );
    EXPECT_GE(sw.elapsed(), 100 * x::telem::MILLISECOND);
    EXPECT_LT(sw.elapsed(), x::telem::SECOND);
}

TEST_F(ModelTest, FailsToOpenANameThatDoesNotResolve) {
    auto p = props();
    p.host = "missing.local";
    ASSERT_OCCURRED_AS_P(Client::open(p, this->config()), transport::UNREACHABLE_ERROR);
}

TEST_F(ModelTest, StaysClosedAfterThePeerCloses) {
    const auto [client, peer] = this->connect();
    peer->send("last");
    peer->close();
    EXPECT_EQ(as_string(ASSERT_NIL_P(client->read(x::telem::SECOND)).data), "last");
    const auto [chunk, err] = client->read(x::telem::SECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    ASSERT_OCCURRED_AS_P(client->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
    EXPECT_EQ(client->write(bytes("ping"), x::telem::SECOND), err);
    EXPECT_TRUE(peer->closed());
    EXPECT_EQ(this->network.accept(DEVICE, 10 * x::telem::MILLISECOND), nullptr);
}

TEST_F(ModelTest, FailsToWriteAfterThePeerCloses) {
    const auto [client, peer] = this->connect();
    peer->close();
    ASSERT_OCCURRED_AS(
        client->write(bytes("ping"), x::telem::SECOND),
        transport::UNREACHABLE_ERROR
    );
}

struct Step {
    std::size_t step;
    std::error_code ec;
    x::errors::Error expected;
};

TEST_F(ModelTest, MapsAFailureAtEachOpeningStepThroughTranslate) {
    for (const auto &[step, ec, expected]: std::vector<Step>{
             {1, asio::error::host_not_found, transport::UNREACHABLE_ERROR},
             {1, asio::error::no_memory, transport::CRITICAL_ERROR},
             {2, asio::error::connection_refused, transport::UNREACHABLE_ERROR},
             {2, asio::error::network_unreachable, transport::UNREACHABLE_ERROR},
             {2, asio::error::no_descriptors, transport::CRITICAL_ERROR},
             {2, asio::error::access_denied, transport::CONFIG_ERROR},
             {3, asio::error::no_protocol_option, transport::CONFIG_ERROR},
         }) {
        SCOPED_TRACE(ec.message());
        this->network.fail_at(step, ec);
        ASSERT_OCCURRED_AS_P(Client::open(props(), this->config()), expected);
    }
}

TEST_F(ModelTest, MapsAFailedReadOrWriteThroughTranslate) {
    for (const auto &[step, ec, expected]: std::vector<Step>{
             {1, asio::error::connection_reset, transport::UNREACHABLE_ERROR},
             {1, asio::error::timed_out, transport::UNREACHABLE_ERROR},
             {1, asio::error::no_buffer_space, transport::CRITICAL_ERROR},
         }) {
        SCOPED_TRACE(ec.message());
        auto [reader, reader_peer] = this->connect();
        this->network.fail_at(step, ec);
        ASSERT_OCCURRED_AS_P(reader->read(x::telem::SECOND), expected);
        auto [writer, writer_peer] = this->connect();
        this->network.fail_at(step, ec);
        ASSERT_OCCURRED_AS(writer->write(bytes("ping"), x::telem::SECOND), expected);
    }
}

namespace {
/// @brief plays a TCP device on a modeled network.
class ModelRig final : public bus::testutil::Rig {
    model::Network network;
    std::shared_ptr<model::Peer> peer;

    /// @returns the device end of the newest connection, accepting it first.
    std::shared_ptr<model::Peer> accepted() {
        if (this->peer == nullptr) this->peer = this->network.accept(DEVICE);
        EXPECT_NE(this->peer, nullptr) << "no connection reached the device";
        return this->peer;
    }

    bus::Opener open(const std::string &host) const {
        synnax::tcp::Properties p;
        p.host = host;
        p.port = DEVICE.port();
        const Config config{.sockets = this->network.sockets()};
        return
            [p,
             config]() -> std::pair<std::unique_ptr<bus::Transport>, x::errors::Error> {
                auto [client, err] = Client::open(p, config);
                if (err) return {nullptr, err};
                return {
                    std::make_unique<bus::Adapter<Client>>(std::move(client)),
                    x::errors::NIL
                };
            };
    }

public:
    ModelRig() {
        this->network.name("plc.local", DEVICE.address());
        this->network.listen(DEVICE);
    }

    bus::Opener opener() override { return this->open("plc.local"); }

    bus::Opener absent() override { return this->open("10.0.0.3"); }

    void send(const std::string &data) override {
        if (const auto p = this->accepted()) p->send(data);
    }

    std::string receive(const std::size_t n) override {
        if (const auto p = this->accepted()) return p->receive(n);
        return "";
    }

    void hang_up() override {
        if (const auto p = this->accepted()) p->close();
        this->peer.reset();
    }

    void restore() override {}
};

const auto MODEL = ::testing::Values(
    bus::testutil::Case{
        .name = "model",
        .rig = [] { return std::make_unique<ModelRig>(); },
    }
);
}

using bus::testutil::StreamContract;
using bus::testutil::TransportContract;
INSTANTIATE_TEST_SUITE_P(TcpModel, TransportContract, MODEL, bus::testutil::name);
INSTANTIATE_TEST_SUITE_P(TcpModel, StreamContract, MODEL, bus::testutil::name);
}
