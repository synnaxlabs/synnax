// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/can/canlib/canlib.h"
#include "driver/can/canlib/mock.h"

namespace driver::can::canlib {
namespace {
const std::string LEAF = "73-30130-00685-0:12345:1";

const MockAPI::Card LEAF_CARD{
    .description = "Kvaser Leaf Light v2",
    .ean = 0x0007330130006850,
    .serial = 12345,
};

const MockAPI::Card VIRTUAL_CARD{
    .description = "Kvaser Virtual CAN Driver",
    .serial = 1,
    .channels = 2,
    .is_virtual = true,
};

synnax::can::Properties props(const bool fd = false, const bool listen_only = false) {
    synnax::can::Properties p;
    p.backend = synnax::can::BACKEND_CANLIB;
    p.channel = LEAF;
    p.bitrate = 500000;
    p.fd = fd;
    p.data_bitrate = 2000000;
    p.listen_only = listen_only;
    return p;
}

MockAPI::Message data(const long id) {
    MockAPI::Message m;
    m.id = id;
    m.flags = canMSG_STD;
    return m;
}

class CANlibTest : public ::testing::Test {
protected:
    std::shared_ptr<MockAPI> api = std::make_shared<MockAPI>();
    Backend backend{api};
    int leaf = this->api->attach(LEAF_CARD);
    int virtual_card = this->api->attach(VIRTUAL_CARD);

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

TEST(CANlibTranslate, WaitsForAnAbsentOrBusyChannel) {
    MockAPI api;
    for (const auto status:
         {canERR_NOTFOUND,
          canERR_NOCHANNELS,
          canERR_NO_ACCESS,
          canERR_HARDWARE,
          canERR_NOCARD,
          canERR_TIMEOUT,
          canERR_INTERRUPTED}) {
        const auto err = translate(api, status, "CANlib channel 0");
        ASSERT_MATCHES(err, UNREACHABLE_ERROR);
        EXPECT_EQ(err.data, "CANlib channel 0: mock status " + std::to_string(status));
    }
}

TEST(CANlibTranslate, ReportsADriverOrParameterProblemAsAConfigurationError) {
    MockAPI api;
    for (const auto status:
         {canERR_PARAM,
          canERR_DRIVER,
          canERR_NOT_SUPPORTED,
          canERR_NOT_IMPLEMENTED,
          canERR_DYNALOAD,
          canERR_DYNALIB,
          canERR_DYNAINIT,
          canERR_DRIVERLOAD,
          canERR_REGISTRY,
          canERR_LICENSE})
        ASSERT_MATCHES(translate(api, status, "CANlib channel 0"), CONFIG_ERROR);
}

TEST(CANlibTranslate, ReportsAFullTransmitBufferAsATemporaryHardwareError) {
    MockAPI api;
    const auto err = translate(api, canERR_TXBUFOFL, "CANlib channel 0");
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_FALSE(err.matches(UNREACHABLE_ERROR));
}

TEST(CANlibTranslate, ReportsAnyOtherStatusAsACriticalHardwareError) {
    MockAPI api;
    for (const auto status: {canERR_INTERNAL, canERR_NOMEM, canERR_INVHANDLE})
        ASSERT_MATCHES(
            translate(api, status, "CANlib channel 0"),
            CRITICAL_HARDWARE_ERROR
        );
}

TEST_F(CANlibTest, NamesEachChannelByItsCardAndPlaceOnTheCard) {
    const auto channels = ASSERT_NIL_P(this->backend.scan());
    ASSERT_EQ(channels.size(), 3);
    EXPECT_EQ(channels[0].backend, "canlib");
    EXPECT_EQ(channels[0].name, LEAF);
    EXPECT_EQ(channels[0].description, "Kvaser Leaf Light v2, S/N 12345, channel 1");
    EXPECT_EQ(channels[1].name, "00-00000-00000-0:1:1");
    EXPECT_EQ(channels[2].name, "00-00000-00000-0:1:2");
    EXPECT_EQ(channels[2].description, "Kvaser Virtual CAN Driver, S/N 1, channel 2");
}

TEST_F(CANlibTest, ScansACardPluggedInAfterTheLastScan) {
    ASSERT_EQ(ASSERT_NIL_P(this->backend.scan()).size(), 3);
    this->api->attach({.description = "Kvaser U100", .ean = 0x0007330130011939});
    EXPECT_EQ(ASSERT_NIL_P(this->backend.scan()).size(), 4);
}

TEST_F(CANlibTest, OpensAReadHandleAndAWriteHandleOnTheBus) {
    auto bus = this->open();
    ASSERT_EQ(this->api->opened(), 2);
    const auto rx = this->api->handle(0);
    EXPECT_EQ(rx.card, this->leaf);
    EXPECT_EQ(rx.flags, 0);
    EXPECT_EQ(rx.freq, canBITRATE_500K);
    EXPECT_EQ(rx.driver, canDRIVER_NORMAL);
    EXPECT_EQ(rx.timer_scale, 10);
    EXPECT_FALSE(rx.txecho);
    EXPECT_TRUE(rx.on);
    EXPECT_TRUE(this->api->handle(1).on);
}

TEST_F(CANlibTest, OpensAnFDChannelWithBothBitrates) {
    auto bus = this->open(props(true));
    const auto rx = this->api->handle(0);
    EXPECT_EQ(rx.flags, canOPEN_CAN_FD);
    EXPECT_EQ(rx.freq, canFD_BITRATE_500K_80P);
    EXPECT_EQ(rx.freq_brs, canFD_BITRATE_2M_80P);
}

TEST_F(CANlibTest, OpensAVirtualChannelOnlyWhenItsNameSaysSo) {
    auto p = props();
    p.channel = "00-00000-00000-0:1:2";
    auto bus = this->open(p);
    const auto rx = this->api->handle(0);
    EXPECT_EQ(rx.card, this->virtual_card);
    EXPECT_EQ(rx.channel, 1);
    EXPECT_EQ(rx.flags, canOPEN_ACCEPT_VIRTUAL);
}

TEST_F(CANlibTest, OpensOnlyASilentReadHandleWhenListenOnly) {
    auto bus = this->open(props(false, true));
    ASSERT_EQ(this->api->opened(), 1);
    EXPECT_EQ(this->api->handle(0).driver, canDRIVER_SILENT);
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel " + LEAF + " is listen only");
}

TEST_F(CANlibTest, RejectsAChannelThatNamesNoCard) {
    auto p = props();
    p.channel = "0";
    auto [bus, err] = this->backend.open(p);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "CANlib channel '0' must name a card and a channel on it, such as "
        "73-30130-00685-0:12345:1"
    );
}

TEST_F(CANlibTest, WaitsForAChannelWithNoCardAttached) {
    this->api->unplug(this->leaf);
    auto [bus, err] = this->backend.open(props());
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "CANlib channel " + LEAF + ": no adapter is attached");
    EXPECT_EQ(this->api->opened(), 0);
}

TEST_F(CANlibTest, OpensTheChannelItNamesAfterAReplugChangesTheNumbers) {
    ASSERT_EQ(ASSERT_NIL_P(this->backend.scan())[0].name, LEAF);
    this->api->unplug(this->leaf);
    this->api->plug(this->leaf);
    auto bus = this->open();
    EXPECT_EQ(this->api->handle(0).card, this->leaf);
    EXPECT_EQ(this->api->handle(1).card, this->leaf);
}

TEST_F(CANlibTest, WaitsForAChannelAnotherProgramHoldsExclusively) {
    this->api->hold(this->leaf, 0, true);
    auto [bus, err] = this->backend.open(props());
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "CANlib channel " + LEAF + ": mock status -5");
}

TEST_F(CANlibTest, ClosesTheReadHandleWhenConfigurationFails) {
    this->api->bus_params_status = canERR_PARAM;
    auto [bus, err] = this->backend.open(props());
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_TRUE(this->api->handle(0).closed);
}

TEST_F(CANlibTest, ReceivesAnFDFrameWithItsHardwareTime) {
    auto bus = this->open(props(true));
    MockAPI::Message m;
    m.id = 0x1F334455;
    m.dlc = 20;
    m.flags = canMSG_EXT | canFDMSG_FDF | canFDMSG_BRS;
    m.data[19] = 0x77;
    m.time = 1500;
    this->api->deliver(this->leaf, 0, m);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND * 5)));
    EXPECT_EQ(this->api->last_timeout(), 5);
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
    this->api->deliver(this->leaf, 0, first);
    this->api->deliver(this->leaf, 0, second);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND)));
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND)));
    EXPECT_EQ(frame.time.nanoseconds(), static_cast<std::int64_t>(0x100000001) * 10000);
}

TEST_F(CANlibTest, ReportsAnErrorFrameWithItsProtocolErrors) {
    auto bus = this->open();
    MockAPI::Message m;
    m.flags = canMSG_ERROR_FRAME | canMSGERR_STUFF | canMSGERR_BIT0;
    this->api->deliver(this->leaf, 0, m);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND)));
    EXPECT_EQ(frame.type, Type::BUS_ERROR);
    EXPECT_EQ(
        bus_error::describe(frame),
        "the adapter reported bus error, stuff error, dominant bit error"
    );
}

TEST_F(CANlibTest, ReportsAnOverrunBeforeTheFrameThatFollowsIt) {
    auto bus = this->open();
    auto m = data(0x42);
    m.flags |= canMSGERR_HW_OVERRUN;
    this->api->deliver(this->leaf, 0, m);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND)));
    EXPECT_EQ(frame.type, Type::BUS_ERROR);
    EXPECT_EQ(
        bus_error::describe(frame),
        "the adapter reported receive overflow, frames lost"
    );
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND)));
    EXPECT_EQ(frame.type, Type::DATA);
    EXPECT_EQ(frame.id, 0x42);
}

TEST_F(CANlibTest, WaitsForAFrameUntilTheTimeoutElapses) {
    auto bus = this->open();
    Frame frame;
    const auto start = x::telem::TimeStamp::now();
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND * 20)));
    EXPECT_GE(x::telem::TimeStamp::now() - start, x::telem::MILLISECOND * 20);
}

TEST_F(CANlibTest, RoundsAShortTimeoutUpToAMillisecond) {
    auto bus = this->open();
    Frame frame;
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, x::telem::MICROSECOND * 10)));
    EXPECT_EQ(this->api->last_timeout(), 1);
}

TEST_F(CANlibTest, WakesWhenAFrameArrives) {
    auto bus = this->open();
    std::thread producer([this] {
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
        this->api->deliver(this->leaf, 0, data(0x55));
    });
    Frame frame;
    const auto start = x::telem::TimeStamp::now();
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND * 5)));
    EXPECT_LT(x::telem::TimeStamp::now() - start, x::telem::SECOND);
    EXPECT_EQ(frame.id, 0x55);
    producer.join();
}

TEST_F(CANlibTest, ReportsAnUnpluggedCardAsUnreachable) {
    auto bus = this->open();
    this->api->unplug(this->leaf);
    Frame frame;
    auto [received, err] = bus->receive(frame, x::telem::SECOND);
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "CANlib channel " + LEAF + ": mock status -15");
    ASSERT_MATCHES(bus->send(Frame{.id = 1}), UNREACHABLE_ERROR);
}

TEST_F(CANlibTest, ClosesAChannelWhoseCardWasUnplugged) {
    auto bus = this->open();
    this->api->unplug(this->leaf);
    ASSERT_NIL(bus->close());
    EXPECT_TRUE(this->api->handle(0).closed);
    EXPECT_TRUE(this->api->handle(1).closed);
}

TEST_F(CANlibTest, ReopensAChannelAfterAReplug) {
    auto bus = this->open();
    this->api->unplug(this->leaf);
    Frame frame;
    ASSERT_MATCHES(bus->receive(frame, x::telem::SECOND).second, UNREACHABLE_ERROR);
    ASSERT_NIL(bus->close());
    this->api->plug(this->leaf);
    bus = this->open();
    this->api->deliver(this->leaf, 0, data(0x9));
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x9);
}

TEST_F(CANlibTest, SendsThroughTheWriteHandle) {
    auto bus = this->open();
    Frame frame{.id = 0x321, .type = Type::REMOTE, .length = 2};
    ASSERT_NIL(bus->send(frame));
    const auto written = this->api->written();
    ASSERT_EQ(written.size(), 1);
    EXPECT_EQ(written[0].hnd, 1);
    EXPECT_EQ(written[0].id, 0x321);
    EXPECT_EQ(written[0].flags, canMSG_STD | canMSG_RTR);
    EXPECT_EQ(written[0].data.size(), 2);
}

TEST_F(CANlibTest, ReportsAFullTransmitBufferAsATemporaryHardwareError) {
    auto bus = this->open();
    this->api->write_status = canERR_TXBUFOFL;
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_FALSE(err.matches(UNREACHABLE_ERROR));
    EXPECT_EQ(err.data, "CANlib channel " + LEAF + ": transmit queue is full");
}

TEST_F(CANlibTest, TakesBothHandlesOffTheBusOnClose) {
    auto bus = this->open();
    ASSERT_NIL(bus->close());
    for (const canHandle hnd: {0, 1}) {
        EXPECT_FALSE(this->api->handle(hnd).on);
        EXPECT_TRUE(this->api->handle(hnd).closed);
    }
}
}
