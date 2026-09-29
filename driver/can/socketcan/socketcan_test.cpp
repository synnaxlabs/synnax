// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/can/socketcan/socketcan.h"

namespace driver::can::socketcan {
namespace {
const std::filesystem::path VCAN = "/sys/class/net/vcan0";

synnax::can::Properties props(const bool fd = false, const bool listen_only = false) {
    synnax::can::Properties p;
    p.backend = synnax::can::BACKEND_SOCKETCAN;
    p.channel = "vcan0";
    p.fd = fd;
    p.listen_only = listen_only;
    return p;
}

/// @brief runs against the virtual interface vcan0. Create it with:
/// ip link add dev vcan0 type vcan && ip link set vcan0 mtu 72 up
class SocketCANTest : public ::testing::Test {
protected:
    std::shared_ptr<can::Backend> backend = load();

    void SetUp() override {
        if (!std::filesystem::exists(VCAN)) GTEST_SKIP() << "vcan0 is absent";
    }

    std::unique_ptr<can::Bus> open(const synnax::can::Properties &p = props()) {
        auto [bus, err] = this->backend->open(p);
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }

    [[nodiscard]] static bool fd_capable() {
        std::ifstream in(VCAN / "mtu");
        std::string mtu;
        std::getline(in, mtu);
        return mtu == "72";
    }
};

Frame data_frame(const std::uint32_t id, const std::vector<std::uint8_t> &data) {
    Frame f;
    f.id = id;
    f.length = static_cast<std::uint8_t>(data.size());
    std::ranges::copy(data, f.data.begin());
    return f;
}
}

TEST_F(SocketCANTest, ScanListsTheVirtualInterface) {
    const auto channels = ASSERT_NIL_P(this->backend->scan());
    EXPECT_TRUE(std::ranges::any_of(channels, [](const Channel &c) {
        return c.name == "vcan0" && c.backend == synnax::can::BACKEND_SOCKETCAN;
    }));
}

TEST_F(SocketCANTest, DeliversAFrameToAnotherBus) {
    auto sender = this->open();
    auto receiver = this->open();
    const auto before = x::telem::TimeStamp::now();
    ASSERT_NIL(sender->send(data_frame(0x123, {0xDE, 0xAD})));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(receiver->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x123);
    EXPECT_FALSE(frame.extended);
    EXPECT_EQ(frame.type, Type::DATA);
    EXPECT_EQ(frame.length, 2);
    EXPECT_EQ(frame.data[0], 0xDE);
    EXPECT_EQ(frame.data[1], 0xAD);
    EXPECT_EQ(frame.clock, Clock::HOST);
    EXPECT_GE(frame.time, before);
}

TEST_F(SocketCANTest, DeliversExtendedAndRemoteFrames) {
    auto sender = this->open();
    auto receiver = this->open();
    auto remote = data_frame(0x1ABCDEF0, {});
    remote.extended = true;
    remote.type = Type::REMOTE;
    remote.length = 4;
    ASSERT_NIL(sender->send(remote));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(receiver->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x1ABCDEF0);
    EXPECT_TRUE(frame.extended);
    EXPECT_EQ(frame.type, Type::REMOTE);
    EXPECT_EQ(frame.length, 4);
}

TEST_F(SocketCANTest, NeverReceivesItsOwnFrames) {
    auto bus = this->open();
    ASSERT_NIL(bus->send(data_frame(0x1, {})));
    Frame frame;
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, 50 * x::telem::MILLISECOND)));
}

TEST_F(SocketCANTest, DeliversCANFDFrames) {
    if (!fd_capable()) GTEST_SKIP() << "vcan0 does not run CAN FD";
    auto sender = this->open(props(true));
    auto receiver = this->open(props(true));
    std::vector<std::uint8_t> data(64);
    for (std::size_t i = 0; i < data.size(); i++)
        data[i] = static_cast<std::uint8_t>(i);
    auto sent = data_frame(0x18DAF110, data);
    sent.extended = true;
    sent.fd = true;
    sent.bitrate_switched = true;
    ASSERT_NIL(sender->send(sent));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(receiver->receive(frame, x::telem::SECOND)));
    EXPECT_TRUE(frame.fd);
    EXPECT_TRUE(frame.bitrate_switched);
    EXPECT_EQ(frame.length, 64);
    EXPECT_EQ(frame.data[63], 63);
}

TEST_F(SocketCANTest, RefusesToSendWhenListenOnly) {
    auto bus = this->open(props(false, true));
    const auto err = bus->send(data_frame(0x1, {}));
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel vcan0 is listen only");
}

TEST_F(SocketCANTest, ReturnsFalseWhenNoFrameArrives) {
    auto bus = this->open();
    Frame frame;
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, 20 * x::telem::MILLISECOND)));
}

TEST_F(SocketCANTest, ReportsAMissingInterface) {
    auto p = props();
    p.channel = "nocan9";
    const auto [bus, err] = this->backend->open(p);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "no CAN interface is named nocan9");
}
}
