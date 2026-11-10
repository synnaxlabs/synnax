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
#include <optional>
#include <span>
#include <string>
#include <system_error>
#include <vector>

#include "asio/error.hpp"
#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/bus/testutil/contract.h"
#include "driver/bus/transport.h"
#include "driver/udp/model.h"
#include "driver/udp/socket.h"

namespace driver::udp {
namespace {
std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

std::string as_string(const std::span<const std::uint8_t> data) {
    return {reinterpret_cast<const char *>(data.data()), data.size()};
}

const asio::ip::udp::endpoint DEVICE{asio::ip::make_address("10.0.0.2"), 4000};
const asio::ip::udp::endpoint HOST{asio::ip::address_v4::loopback(), 5000};

/// @returns properties for a socket on HOST that talks to the device at rtu.local.
synnax::udp::Properties props() {
    synnax::udp::Properties p;
    p.port = HOST.port();
    p.remote_host = "rtu.local";
    p.remote_port = DEVICE.port();
    return p;
}
}

/// @brief drives a socket over a modeled network with a device at DEVICE.
class ModelTest : public ::testing::Test {
protected:
    model::Network network;
    model::Peer device = this->network.bind(DEVICE);

    [[nodiscard]] Config config() const {
        return Config{.stacks = this->network.stacks()};
    }

    void SetUp() override { this->network.name("rtu.local", DEVICE.address()); }
};

TEST_F(ModelTest, SendsAQueryToTheRemote) {
    const auto socket = ASSERT_NIL_P(Socket::open(props(), this->config()));
    ASSERT_NIL(socket->write(bytes("status?"), x::telem::SECOND));
    const auto got = this->device.receive();
    ASSERT_TRUE(got.has_value());
    EXPECT_EQ(got.value().first, "status?");
    EXPECT_EQ(got.value().second, HOST);
}

TEST_F(ModelTest, DropsDatagramsFromSendersOtherThanTheRemote) {
    const auto socket = ASSERT_NIL_P(Socket::open(props(), this->config()));
    const auto foreign = this->network.bind({asio::ip::make_address("10.0.0.9"), 4000});
    const auto other_port = this->network.bind({DEVICE.address(), 4001});
    foreign.send("spoofed host", HOST);
    other_port.send("spoofed port", HOST);
    this->device.send("genuine", HOST);
    EXPECT_EQ(as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data), "genuine");
    EXPECT_TRUE(ASSERT_NIL_P(socket->read(10 * x::telem::MILLISECOND)).data.empty());
}

TEST_F(ModelTest, ReceivesFromAnySenderWithoutARemote) {
    auto p = props();
    p.remote_host = "";
    p.remote_port = 0;
    const auto socket = ASSERT_NIL_P(Socket::open(p, this->config()));
    const auto other = this->network.bind({asio::ip::make_address("10.0.0.9"), 4000});
    other.send("anyone", HOST);
    EXPECT_EQ(as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data), "anyone");
}

TEST_F(ModelTest, PassesADuplicatedDatagramThroughTwice) {
    const auto socket = ASSERT_NIL_P(Socket::open(props(), this->config()));
    this->network.duplicate(1);
    this->device.send("twice", HOST);
    EXPECT_EQ(as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data), "twice");
    EXPECT_EQ(as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data), "twice");
}

TEST_F(ModelTest, ReturnsAnEmptyChunkForALostDatagram) {
    const auto socket = ASSERT_NIL_P(Socket::open(props(), this->config()));
    this->network.lose(1);
    this->device.send("lost", HOST);
    this->device.send("kept", HOST);
    EXPECT_EQ(as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data), "kept");
    const x::telem::Stopwatch sw;
    EXPECT_TRUE(ASSERT_NIL_P(socket->read(50 * x::telem::MILLISECOND)).data.empty());
    EXPECT_GE(sw.elapsed(), 50 * x::telem::MILLISECOND);
}

TEST_F(ModelTest, RejectsAPortInUse) {
    const auto first = ASSERT_NIL_P(Socket::open(props(), this->config()));
    ASSERT_OCCURRED_AS_P(
        Socket::open(props(), this->config()),
        transport::CONFIG_ERROR
    );
}

TEST_F(ModelTest, FailsToBindALocalAddressTheHostLacks) {
    auto p = props();
    p.local_address = "192.168.7.2";
    ASSERT_OCCURRED_AS_P(Socket::open(p, this->config()), transport::UNREACHABLE_ERROR);
    this->network.add(asio::ip::make_address("192.168.7.2"));
    ASSERT_NIL_P(Socket::open(p, this->config()));
}

TEST_F(ModelTest, JoinsAMulticastGroupOnTheLocalAddressInterface) {
    this->network.add(asio::ip::make_address("192.168.7.2"));
    auto p = props();
    p.remote_host = "";
    p.remote_port = 0;
    p.multicast_group = "239.255.0.7";
    p.local_address = "192.168.7.2";
    const auto socket = ASSERT_NIL_P(Socket::open(p, this->config()));
    this->device.send(
        "to the group",
        {asio::ip::make_address("239.255.0.7"), HOST.port()}
    );
    EXPECT_EQ(
        as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data),
        "to the group"
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
             {2, asio::error::no_descriptors, transport::CRITICAL_ERROR},
             {2, asio::error::address_family_not_supported, transport::CONFIG_ERROR},
             {3, asio::error::access_denied, transport::CONFIG_ERROR},
             {3,
              std::make_error_code(std::errc::address_not_available),
              transport::UNREACHABLE_ERROR},
         }) {
        SCOPED_TRACE(ec.message());
        this->network.fail_at(step, ec);
        ASSERT_OCCURRED_AS_P(Socket::open(props(), this->config()), expected);
    }
}

TEST_F(ModelTest, MapsAFailedReceiveOrSendThroughTranslate) {
    for (const auto &[step, ec, expected]: std::vector<Step>{
             {1, asio::error::network_down, transport::UNREACHABLE_ERROR},
             {1, asio::error::message_size, transport::CONFIG_ERROR},
             {1, asio::error::no_buffer_space, transport::CRITICAL_ERROR},
         }) {
        SCOPED_TRACE(ec.message());
        {
            const auto socket = ASSERT_NIL_P(Socket::open(props(), this->config()));
            this->network.fail_at(step, ec);
            ASSERT_OCCURRED_AS_P(socket->read(x::telem::SECOND), expected);
        }
        const auto socket = ASSERT_NIL_P(Socket::open(props(), this->config()));
        this->network.fail_at(step, ec);
        ASSERT_OCCURRED_AS(socket->write(bytes("ping"), x::telem::SECOND), expected);
    }
}

TEST_F(ModelTest, KeepsReceivingAfterAnICMPRefusal) {
    const auto socket = ASSERT_NIL_P(Socket::open(props(), this->config()));
    this->network.fail_at(1, asio::error::connection_refused);
    this->device.send("still here", HOST);
    EXPECT_EQ(
        as_string(ASSERT_NIL_P(socket->read(x::telem::SECOND)).data),
        "still here"
    );
}

namespace {
/// @brief plays a UDP device on a modeled network.
class ModelRig final : public bus::testutil::Rig {
    model::Network network;
    std::optional<model::Peer> device;

    [[nodiscard]] bus::Opener open(const std::string &local_address) const {
        auto p = props();
        p.local_address = local_address;
        const Config config{.stacks = this->network.stacks()};
        return
            [p,
             config]() -> std::pair<std::unique_ptr<bus::Transport>, x::errors::Error> {
                auto [socket, err] = Socket::open(p, config);
                if (err) return {nullptr, err};
                return {
                    std::make_unique<bus::Adapter<Socket>>(std::move(socket)),
                    x::errors::NIL
                };
            };
    }

public:
    ModelRig() {
        this->network.name("rtu.local", DEVICE.address());
        this->device.emplace(this->network.bind(DEVICE));
    }

    bus::Opener opener() override { return this->open(""); }

    bus::Opener absent() override { return this->open("192.168.7.2"); }

    void send(const std::string &data) override {
        this->device.value().send(data, HOST);
    }

    std::string receive(const std::size_t n) override {
        std::string out;
        while (out.size() < n) {
            const auto datagram = this->device.value().receive();
            if (!datagram.has_value()) break;
            out += datagram->first;
        }
        return out;
    }

    void hang_up() override { this->device.value().close(); }

    void restore() override { this->device.emplace(this->network.bind(DEVICE)); }
};

const auto MODEL = ::testing::Values(
    bus::testutil::Case{
        .name = "model",
        .rig = [] { return std::make_unique<ModelRig>(); },
    }
);
}

using bus::testutil::TransportContract;
INSTANTIATE_TEST_SUITE_P(UdpModel, TransportContract, MODEL, bus::testutil::name);
}
