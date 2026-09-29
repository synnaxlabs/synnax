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

#include "driver/can/canlib/canlib.h"
#include "driver/can/canlib/mock.h"

namespace driver::can::canlib {
namespace {
synnax::can::Properties props(const bool fd = false, const bool listen_only = false) {
    synnax::can::Properties p;
    p.backend = synnax::can::BACKEND_CANLIB;
    p.channel = "0";
    p.bitrate = 500000;
    p.fd = fd;
    p.data_bitrate = 2000000;
    p.listen_only = listen_only;
    return p;
}

class CANlibTest : public ::testing::Test {
protected:
    std::shared_ptr<MockAPI> api = std::make_shared<MockAPI>();
    Backend backend{api};

    std::unique_ptr<can::Bus> open(const synnax::can::Properties &p = props()) {
        auto [bus, err] = this->backend.open(p);
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }
};
}

TEST(CANlibBitrate, MapsAClassicBitrateToItsConstant) {
    EXPECT_EQ(ASSERT_NIL_P(bitrate_constant(250000)), canBITRATE_250K);
}

TEST(CANlibBitrate, RejectsAClassicBitrateWithNoConstant) {
    auto [c, err] = bitrate_constant(800000);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "CANlib cannot run a classic CAN bus at 800000 bit/s. Supported bitrates: "
        "1000000, 500000, 250000, 125000, 100000, 83333, 62500, 50000, 10000"
    );
}

TEST(CANlibBitrate, MapsFDArbitrationAndDataBitrates) {
    EXPECT_EQ(ASSERT_NIL_P(fd_bitrate_constant(500000, false)), canFD_BITRATE_500K_80P);
    EXPECT_EQ(ASSERT_NIL_P(fd_bitrate_constant(2000000, true)), canFD_BITRATE_2M_80P);
    auto [c, err] = fd_bitrate_constant(2000000, false);
    ASSERT_MATCHES(err, CONFIG_ERROR);
}

TEST_F(CANlibTest, OpensAReadHandleAndAWriteHandleOnTheBus) {
    auto bus = this->open();
    ASSERT_EQ(this->api->handles.size(), 2);
    const auto &rx = this->api->handles.at(0);
    EXPECT_EQ(rx.channel, 0);
    EXPECT_EQ(rx.flags, canOPEN_ACCEPT_VIRTUAL);
    EXPECT_EQ(rx.freq, canBITRATE_500K);
    EXPECT_EQ(rx.driver, canDRIVER_NORMAL);
    EXPECT_EQ(rx.timer_scale, 10);
    EXPECT_FALSE(rx.txecho);
    EXPECT_TRUE(rx.on);
    EXPECT_TRUE(this->api->handles.at(1).on);
}

TEST_F(CANlibTest, OpensAnFDChannelWithBothBitrates) {
    auto bus = this->open(props(true));
    const auto &rx = this->api->handles.at(0);
    EXPECT_EQ(rx.flags, canOPEN_ACCEPT_VIRTUAL | canOPEN_CAN_FD);
    EXPECT_EQ(rx.freq, canFD_BITRATE_500K_80P);
    EXPECT_EQ(rx.freq_brs, canFD_BITRATE_2M_80P);
}

TEST_F(CANlibTest, OpensOnlyASilentReadHandleWhenListenOnly) {
    auto bus = this->open(props(false, true));
    ASSERT_EQ(this->api->handles.size(), 1);
    EXPECT_EQ(this->api->handles.at(0).driver, canDRIVER_SILENT);
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel 0 is listen only");
}

TEST_F(CANlibTest, RejectsAChannelThatIsNotANumber) {
    auto p = props();
    p.channel = "kvaser0";
    auto [bus, err] = this->backend.open(p);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "CANlib channel 'kvaser0' must be a channel number, such as 0");
}

TEST_F(CANlibTest, ReportsAMissingChannelAsAConfigurationError) {
    this->api->open_status = canERR_NOTFOUND;
    auto [bus, err] = this->backend.open(props());
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "CANlib channel 0: mock status -3");
}

TEST_F(CANlibTest, ClosesTheReadHandleWhenConfigurationFails) {
    this->api->bus_params_status = canERR_PARAM;
    auto [bus, err] = this->backend.open(props());
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_TRUE(this->api->handles.at(0).closed);
}

TEST_F(CANlibTest, ReceivesAnFDFrameWithItsHardwareTime) {
    auto bus = this->open(props(true));
    MockAPI::Message m;
    m.id = 0x1F334455;
    m.dlc = 20;
    m.flags = canMSG_EXT | canFDMSG_FDF | canFDMSG_BRS;
    m.data[19] = 0x77;
    m.time = 1500;
    this->api->reads.push_back(m);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND * 5)));
    EXPECT_EQ(this->api->last_timeout, 5);
    EXPECT_EQ(frame.id, 0x1F334455);
    EXPECT_TRUE(frame.extended);
    EXPECT_TRUE(frame.fd);
    EXPECT_TRUE(frame.bitrate_switched);
    EXPECT_EQ(frame.length, 20);
    EXPECT_EQ(frame.data[19], 0x77);
    EXPECT_EQ(frame.clock, Clock::HARDWARE);
    EXPECT_EQ(frame.time.nanoseconds(), 15000000);
}

TEST_F(CANlibTest, ExtendsTheTimerAcrossAWrap) {
    auto bus = this->open();
    MockAPI::Message first;
    first.time = 0xFFFFFFFF;
    MockAPI::Message second;
    second.time = 1;
    this->api->reads.push_back(first);
    this->api->reads.push_back(second);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND)));
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND)));
    EXPECT_EQ(frame.time.nanoseconds(), static_cast<std::int64_t>(0x100000001) * 10000);
}

TEST_F(CANlibTest, ReceivesAnErrorFrame) {
    auto bus = this->open();
    MockAPI::Message m;
    m.flags = canMSG_ERROR_FRAME;
    this->api->reads.push_back(m);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND)));
    EXPECT_EQ(frame.type, Type::BUS_ERROR);
}

TEST_F(CANlibTest, ReturnsNoFrameWhenTheTimeoutElapses) {
    auto bus = this->open();
    Frame frame;
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, x::telem::MICROSECOND * 10)));
    EXPECT_EQ(this->api->last_timeout, 1);
}

TEST_F(CANlibTest, SendsThroughTheWriteHandle) {
    auto bus = this->open();
    Frame frame{.id = 0x321, .type = Type::REMOTE, .length = 2};
    ASSERT_NIL(bus->send(frame));
    ASSERT_EQ(this->api->written.size(), 1);
    const auto &w = this->api->written[0];
    EXPECT_EQ(w.hnd, 1);
    EXPECT_EQ(w.id, 0x321);
    EXPECT_EQ(w.flags, canMSG_STD | canMSG_RTR);
    EXPECT_EQ(w.data.size(), 2);
}

TEST_F(CANlibTest, ReportsAFullTransmitBufferAsATemporaryHardwareError) {
    auto bus = this->open();
    this->api->write_status = canERR_TXBUFOFL;
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "CANlib channel 0: transmit queue is full");
}

TEST_F(CANlibTest, TakesBothHandlesOffTheBusOnClose) {
    auto bus = this->open();
    ASSERT_NIL(bus->close());
    for (const auto &[hnd, h]: this->api->handles) {
        EXPECT_FALSE(h.on);
        EXPECT_TRUE(h.closed);
    }
}

TEST_F(CANlibTest, ScansEveryChannel) {
    this->api->channels = {
        {.description = "Kvaser Leaf Light v2", .serial = 12345, .number = 0},
        {.description = "Kvaser Virtual CAN Driver", .serial = 0, .number = 1},
    };
    const auto channels = ASSERT_NIL_P(this->backend.scan());
    ASSERT_EQ(channels.size(), 2);
    EXPECT_EQ(channels[0].backend, "canlib");
    EXPECT_EQ(channels[0].name, "0");
    EXPECT_EQ(channels[0].description, "Kvaser Leaf Light v2, S/N 12345, channel 1");
    EXPECT_EQ(channels[1].name, "1");
}
}
