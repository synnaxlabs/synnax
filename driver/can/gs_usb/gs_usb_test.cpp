// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/can/gs_usb/gs_usb.h"
#include "driver/can/gs_usb/mock.h"

namespace driver::can::gs_usb {
namespace {
synnax::can::Properties props(const std::string &channel = "ABC123:0") {
    synnax::can::Properties p;
    p.backend = synnax::can::BACKEND_GS_USB;
    p.channel = channel;
    p.bitrate = 500000;
    p.data_bitrate = 2000000;
    return p;
}

std::uint32_t u32(const std::vector<std::uint8_t> &bytes, const std::size_t at) {
    return bytes[at] | bytes[at + 1] << 8 | bytes[at + 2] << 16 |
           static_cast<std::uint32_t>(bytes[at + 3]) << 24;
}

/// @returns a received host frame with a timestamp.
std::vector<std::uint8_t> received(
    const std::uint32_t id,
    const std::uint32_t timestamp,
    const std::uint8_t ch = 0
) {
    std::vector<std::uint8_t> bytes(24, 0);
    bytes[0] = bytes[1] = bytes[2] = bytes[3] = 0xFF;
    bytes[4] = static_cast<std::uint8_t>(id);
    bytes[5] = static_cast<std::uint8_t>(id >> 8);
    bytes[8] = 1;
    bytes[9] = ch;
    bytes[12] = 0x5A;
    for (int i = 0; i < 4; i++)
        bytes[20 + i] = static_cast<std::uint8_t>(timestamp >> (8 * i));
    return bytes;
}

class GsUsbTest : public ::testing::Test {
protected:
    std::shared_ptr<MockAPI> api = std::make_shared<MockAPI>();
    Backend backend{api};

    void SetUp() override {
        this->api->devices.push_back({.serial = "ABC123", .channels = 2});
    }

    std::unique_ptr<can::Bus> open(const synnax::can::Properties &p = props()) {
        auto [bus, err] = this->backend.open(p);
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }
};
}

TEST_F(GsUsbTest, ScansEveryChannelOfEveryAdapter) {
    this->api->devices.push_back({.vendor = 0x1234, .serial = "OTHER"});
    const auto channels = ASSERT_NIL_P(this->backend.scan());
    ASSERT_EQ(channels.size(), 2);
    EXPECT_EQ(channels[0].backend, "gs_usb");
    EXPECT_EQ(channels[0].name, "ABC123:0");
    EXPECT_EQ(channels[0].description, "candleLight USB to CAN adapter channel 0");
    EXPECT_EQ(channels[1].name, "ABC123:1");
    EXPECT_FALSE(this->api->devices[0].open);
}

TEST_F(GsUsbTest, ConfiguresAndStartsTheChannel) {
    auto bus = this->open(props("ABC123:1"));
    EXPECT_TRUE(this->api->devices[0].claimed);
    const auto timing = this->api->controls_of(Request::BITTIMING);
    ASSERT_EQ(timing.size(), 1);
    EXPECT_EQ(timing[0].value, 1);
    EXPECT_EQ(u32(timing[0].body, 16), 6);
    const auto modes = this->api->controls_of(Request::MODE);
    ASSERT_EQ(modes.size(), 2);
    EXPECT_EQ(u32(modes[0].body, 0), MODE_RESET);
    EXPECT_EQ(u32(modes[1].body, 0), MODE_START);
    EXPECT_EQ(u32(modes[1].body, 4), FEATURE_HW_TIMESTAMP);
}

TEST_F(GsUsbTest, AcceptsASerialNumberWithoutAChannelIndex) {
    auto bus = this->open(props("ABC123"));
    EXPECT_EQ(this->api->controls_of(Request::BITTIMING)[0].value, 0);
}

TEST_F(GsUsbTest, ConfiguresTheDataPhaseOfAnFDChannel) {
    auto &bt = this->api->devices[0].bt_const;
    bt.features |= FEATURE_FD | FEATURE_BT_CONST_EXT;
    bt.clock_hz = 80000000;
    bt.nominal = {.tseg1_max = 256, .tseg2_max = 128, .sjw_max = 128, .brp_max = 1024};
    bt.data = {.tseg1_max = 32, .tseg2_max = 16, .sjw_max = 16, .brp_max = 1024};
    auto p = props();
    p.fd = true;
    auto bus = this->open(p);
    ASSERT_EQ(this->api->controls_of(Request::DATA_BITTIMING).size(), 1);
    const auto modes = this->api->controls_of(Request::MODE);
    EXPECT_EQ(u32(modes[1].body, 4), FEATURE_FD | FEATURE_HW_TIMESTAMP);
}

TEST_F(GsUsbTest, RejectsFDOnAClassicAdapter) {
    auto p = props();
    p.fd = true;
    auto [bus, err] = this->backend.open(p);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "ABC123:0: the adapter does not support CAN FD");
    EXPECT_FALSE(this->api->devices[0].open);
    EXPECT_FALSE(this->api->devices[0].claimed);
}

TEST_F(GsUsbTest, RejectsAChannelIndexTheAdapterLacks) {
    auto [bus, err] = this->backend.open(props("ABC123:2"));
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "ABC123:2: the adapter has 2 channel(s)");
}

TEST_F(GsUsbTest, RejectsAnAdapterThatIsNotAttached) {
    auto [bus, err] = this->backend.open(props("MISSING:0"));
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "no gs_usb adapter with serial number MISSING is attached");
}

TEST_F(GsUsbTest, RejectsAMalformedChannelIndex) {
    auto [bus, err] = this->backend.open(props("ABC123:x"));
    ASSERT_MATCHES(err, CONFIG_ERROR);
}

TEST_F(GsUsbTest, ReportsAnAdapterInUse) {
    this->api->devices[0].claim_status = LIBUSB_ERROR_BUSY;
    auto [bus, err] = this->backend.open(props());
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "ABC123:0: the adapter is in use: LIBUSB_ERROR_BUSY");
}

TEST_F(GsUsbTest, StartsAListenOnlyChannelInListenOnlyMode) {
    auto p = props();
    p.listen_only = true;
    auto bus = this->open(p);
    const auto modes = this->api->controls_of(Request::MODE);
    EXPECT_EQ(u32(modes[1].body, 4), FEATURE_LISTEN_ONLY | FEATURE_HW_TIMESTAMP);
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel ABC123:0 is listen only");
}

TEST_F(GsUsbTest, ReceivesAFrameWithItsExtendedHardwareTime) {
    auto bus = this->open();
    this->api->in.push_back(received(0x10, 0xFFFFFFF0));
    this->api->in.push_back(received(0x11, 0x00000010));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x10);
    EXPECT_EQ(frame.data[0], 0x5A);
    EXPECT_EQ(frame.clock, Clock::HARDWARE);
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.time.nanoseconds(), static_cast<std::int64_t>(0x100000010) * 1000);
}

TEST_F(GsUsbTest, SkipsEchoesAndFramesOfOtherChannels) {
    auto bus = this->open();
    auto echo = received(0x1, 0);
    echo[0] = echo[1] = echo[2] = echo[3] = 0;
    this->api->in.push_back(echo);
    this->api->in.push_back(received(0x2, 0, 1));
    this->api->in.push_back(received(0x3, 0));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x3);
}

TEST_F(GsUsbTest, ReturnsNoFrameWhenTheTimeoutElapses) {
    auto bus = this->open();
    Frame frame;
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND * 20)));
    EXPECT_GE(this->api->last_timeout, 1);
    EXPECT_LE(this->api->last_timeout, 20);
}

TEST_F(GsUsbTest, SendsAHostFrameWithACyclingEchoId) {
    auto bus = this->open();
    ASSERT_NIL(bus->send(Frame{.id = 0x100, .length = 2}));
    ASSERT_NIL(bus->send(Frame{.id = 0x101, .length = 2}));
    ASSERT_EQ(this->api->out.size(), 2);
    EXPECT_EQ(this->api->out[0].size(), 20);
    EXPECT_EQ(u32(this->api->out[0], 0), 0);
    EXPECT_EQ(u32(this->api->out[1], 0), 1);
    EXPECT_EQ(u32(this->api->out[1], 4), 0x101);
}

TEST_F(GsUsbTest, SharesTheAdapterBetweenItsChannels) {
    auto first = this->open(props("ABC123:0"));
    auto second = this->open(props("ABC123:1"));
    EXPECT_EQ(this->api->controls_of(Request::DEVICE_CONFIG).size(), 1);
    ASSERT_NIL(first->close());
    EXPECT_TRUE(this->api->devices[0].claimed);
    EXPECT_TRUE(this->api->devices[0].open);
    ASSERT_NIL(second->close());
    EXPECT_FALSE(this->api->devices[0].claimed);
    EXPECT_FALSE(this->api->devices[0].open);
}

TEST_F(GsUsbTest, RejectsASecondBusOnAnOpenChannel) {
    auto bus = this->open();
    auto [second, err] = this->backend.open(props());
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "ABC123:0: the channel is in use");
}

TEST_F(GsUsbTest, ReopensAChannelAfterItCloses) {
    ASSERT_NIL(this->open()->close());
    auto bus = this->open();
    EXPECT_TRUE(this->api->devices[0].claimed);
}

TEST_F(GsUsbTest, RoutesEachFrameToTheBusOfItsChannel) {
    auto first = this->open(props("ABC123:0"));
    auto second = this->open(props("ABC123:1"));
    this->api->in.push_back(received(0x21, 0, 1));
    this->api->in.push_back(received(0x20, 0, 0));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(first->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x20);
    ASSERT_TRUE(ASSERT_NIL_P(second->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x21);
}

TEST_F(GsUsbTest, DropsTheOldestFramesOfAChannelThatIsNotReceiving) {
    auto first = this->open(props("ABC123:0"));
    auto second = this->open(props("ABC123:1"));
    for (std::uint32_t id = 0; id <= MAX_QUEUED; id++)
        this->api->in.push_back(received(id, 0, 1));
    this->api->in.push_back(received(0x7FF, 0, 0));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(first->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x7FF);
    ASSERT_TRUE(ASSERT_NIL_P(second->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 1);
}

TEST_F(GsUsbTest, ResetsReleasesAndClosesOnClose) {
    auto bus = this->open();
    ASSERT_NIL(bus->close());
    EXPECT_EQ(this->api->controls_of(Request::MODE).size(), 3);
    EXPECT_FALSE(this->api->devices[0].claimed);
    EXPECT_FALSE(this->api->devices[0].open);
}
}
