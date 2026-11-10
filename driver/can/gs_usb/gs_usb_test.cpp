// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/can/gs_usb/gs_usb.h"
#include "driver/can/gs_usb/mock.h"

namespace driver::can::gs_usb {
namespace {
const std::string SERIAL = "ABC123";

synnax::can::Properties props(const std::string &channel = SERIAL + ":0") {
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

Frame data(const std::uint32_t id, const std::uint8_t byte = 0x5A) {
    Frame frame{.id = id, .length = 1};
    frame.data[0] = byte;
    return frame;
}

class GsUsbTest : public ::testing::Test {
protected:
    std::shared_ptr<MockAPI> api = std::make_shared<MockAPI>();
    std::unique_ptr<Backend> backend;

    void SetUp() override {
        this->api->attach({.serial = SERIAL, .channels = 2});
        this->backend = std::make_unique<Backend>(this->api);
    }

    std::unique_ptr<can::Bus> open(const synnax::can::Properties &p = props()) {
        auto [bus, err] = this->backend->open(p);
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }

    /// @brief delivers a frame once the adapter has handed every earlier frame to the
    /// host, so the firmware never loses it.
    void deliver(const std::uint8_t channel, const Frame &frame, std::uint32_t us = 0) {
        ASSERT_EVENTUALLY_EQ(this->api->queued(SERIAL), 0);
        this->api->deliver(SERIAL, channel, frame, us);
    }
};
}

TEST(GsUsbTranslate, TreatsAGoneBusyOrSilentAdapterAsUnreachable) {
    MockAPI api;
    for (const int rc:
         {LIBUSB_ERROR_NO_DEVICE,
          LIBUSB_ERROR_NOT_FOUND,
          LIBUSB_ERROR_BUSY,
          LIBUSB_ERROR_ACCESS,
          LIBUSB_ERROR_IO,
          LIBUSB_ERROR_TIMEOUT,
          LIBUSB_ERROR_INTERRUPTED})
        ASSERT_MATCHES(translate(api, rc, "ABC123"), UNREACHABLE_ERROR);
}

TEST(GsUsbTranslate, TreatsAnUnsupportedOperationAsAConfigError) {
    MockAPI api;
    const auto err = translate(api, LIBUSB_ERROR_NOT_SUPPORTED, "ABC123");
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "ABC123: LIBUSB_ERROR_NOT_SUPPORTED");
}

TEST(GsUsbTranslate, TreatsAnyOtherErrorAsCritical) {
    MockAPI api;
    for (const int rc: {LIBUSB_ERROR_PIPE, LIBUSB_ERROR_OVERFLOW, LIBUSB_ERROR_OTHER})
        ASSERT_MATCHES(translate(api, rc, "ABC123"), CRITICAL_HARDWARE_ERROR);
}

TEST_F(GsUsbTest, ScansEveryChannelOfEveryAdapter) {
    this->api->attach({.vendor = 0x1234, .serial = "OTHER"});
    const auto channels = ASSERT_NIL_P(this->backend->scan());
    ASSERT_EQ(channels.size(), 2);
    EXPECT_EQ(channels[0].backend, "gs_usb");
    EXPECT_EQ(channels[0].name, "ABC123:0");
    EXPECT_EQ(channels[0].description, "candleLight USB to CAN adapter channel 0");
    EXPECT_EQ(channels[1].name, "ABC123:1");
    EXPECT_EQ(this->api->handles(SERIAL), 0);
}

TEST_F(GsUsbTest, ScansAnAdapterItHoldsWithoutOpeningItAgain) {
    this->api = std::make_shared<MockAPI>(MockAPI::Host::WINDOWS);
    this->api->attach({.serial = SERIAL, .channels = 2});
    this->backend = std::make_unique<Backend>(this->api);
    auto bus = this->open();
    const auto channels = ASSERT_NIL_P(this->backend->scan());
    ASSERT_EQ(channels.size(), 2);
    EXPECT_EQ(channels[0].name, "ABC123:0");
    EXPECT_EQ(channels[1].description, "candleLight USB to CAN adapter channel 1");
    EXPECT_EQ(this->api->opens(SERIAL), 1);
}

TEST_F(GsUsbTest, ConfiguresAndStartsTheChannel) {
    auto bus = this->open(props("ABC123:1"));
    EXPECT_TRUE(this->api->claimed(SERIAL));
    const auto timing = this->api->controls_of(Request::BITTIMING);
    ASSERT_EQ(timing.size(), 1);
    EXPECT_EQ(timing[0].value, 1);
    EXPECT_EQ(u32(timing[0].body, 16), 6);
    const auto modes = this->api->controls_of(Request::MODE);
    ASSERT_EQ(modes.size(), 2);
    EXPECT_EQ(u32(modes[0].body, 0), MODE_RESET);
    EXPECT_EQ(u32(modes[1].body, 0), MODE_START);
    EXPECT_EQ(u32(modes[1].body, 4), FEATURE_HW_TIMESTAMP);
    EXPECT_TRUE(this->api->started(SERIAL, 1));
}

TEST_F(GsUsbTest, AcceptsASerialNumberWithoutAChannelIndex) {
    auto bus = this->open(props(SERIAL));
    EXPECT_EQ(this->api->controls_of(Request::BITTIMING)[0].value, 0);
}

TEST_F(GsUsbTest, ConfiguresTheDataPhaseOfAnFDChannel) {
    MockAPI::Device device{.serial = "FD1"};
    device.bt_const.features |= FEATURE_FD | FEATURE_BT_CONST_EXT;
    device.bt_const.clock_hz = 80000000;
    device.bt_const.nominal =
        {.tseg1_max = 256, .tseg2_max = 128, .sjw_max = 128, .brp_max = 1024};
    device.bt_const
        .data = {.tseg1_max = 32, .tseg2_max = 16, .sjw_max = 16, .brp_max = 1024};
    this->api->attach(device);
    auto p = props("FD1:0");
    p.fd = true;
    auto bus = this->open(p);
    ASSERT_EQ(this->api->controls_of(Request::DATA_BITTIMING).size(), 1);
    const auto modes = this->api->controls_of(Request::MODE);
    EXPECT_EQ(u32(modes[1].body, 4), FEATURE_FD | FEATURE_HW_TIMESTAMP);
}

TEST_F(GsUsbTest, RejectsFDOnAClassicAdapter) {
    auto p = props();
    p.fd = true;
    auto [bus, err] = this->backend->open(p);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "ABC123:0: the adapter does not support CAN FD");
    EXPECT_EQ(this->api->handles(SERIAL), 0);
    EXPECT_FALSE(this->api->claimed(SERIAL));
}

TEST_F(GsUsbTest, RejectsAChannelIndexTheAdapterLacks) {
    auto [bus, err] = this->backend->open(props("ABC123:2"));
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "ABC123:2: the adapter has 2 channel(s)");
}

TEST_F(GsUsbTest, RejectsAMalformedChannelIndex) {
    auto [bus, err] = this->backend->open(props("ABC123:x"));
    ASSERT_MATCHES(err, CONFIG_ERROR);
}

TEST_F(GsUsbTest, WaitsForAnAdapterThatIsNotAttached) {
    auto [bus, err] = this->backend->open(props("MISSING:0"));
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "no gs_usb adapter with serial number MISSING is attached");
}

TEST_F(GsUsbTest, WaitsForAnAdapterAnotherProgramClaimed) {
    this->api->hold(SERIAL, true);
    auto [bus, err] = this->backend->open(props());
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "ABC123:0: the adapter is in use: LIBUSB_ERROR_BUSY");
    EXPECT_EQ(this->api->handles(SERIAL), 0);
    this->api->hold(SERIAL, false);
    EXPECT_NE(this->open(), nullptr);
}

TEST_F(GsUsbTest, WaitsForAnAdapterAnotherProgramOpenedOnWindows) {
    this->api = std::make_shared<MockAPI>(MockAPI::Host::WINDOWS);
    this->api->attach({.serial = SERIAL});
    this->api->hold(SERIAL, true);
    this->backend = std::make_unique<Backend>(this->api);
    auto [bus, err] = this->backend->open(props());
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "cannot open a gs_usb adapter: LIBUSB_ERROR_ACCESS");
}

TEST_F(GsUsbTest, FailsWhenTheAdapterStallsAControlRequest) {
    this->api->fail_controls(SERIAL, LIBUSB_ERROR_PIPE);
    auto [bus, err] = this->backend->open(props());
    ASSERT_MATCHES(err, CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(
        err.data,
        "ABC123:0: cannot read the adapter configuration: LIBUSB_ERROR_PIPE"
    );
    EXPECT_FALSE(this->api->claimed(SERIAL));
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
    this->deliver(0, data(0x10), 0xFFFFFFF0);
    this->deliver(0, data(0x11), 0x00000010);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x10);
    EXPECT_EQ(frame.data[0], 0x5A);
    EXPECT_EQ(frame.clock, Clock::HARDWARE);
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.time.nanoseconds(), static_cast<std::int64_t>(0x100000010) * 1000);
}

TEST_F(GsUsbTest, PassesOnTheBusErrorsTheAdapterReports) {
    auto bus = this->open();
    auto error = bus_error::make(
        bus_error::CONTROLLER,
        bus_error::CONTROLLER_TX_PASSIVE,
        x::telem::TimeStamp::now()
    );
    this->deliver(0, error);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.type, Type::BUS_ERROR);
    EXPECT_EQ(frame.id, bus_error::CONTROLLER);
    EXPECT_EQ(frame.data[1], bus_error::CONTROLLER_TX_PASSIVE);
}

TEST_F(GsUsbTest, SkipsEchoesAndFramesOfOtherChannels) {
    auto first = this->open(props("ABC123:0"));
    auto second = this->open(props("ABC123:1"));
    ASSERT_NIL(first->send(data(0x1)));
    this->deliver(1, data(0x2));
    this->deliver(0, data(0x3));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(first->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x3);
}

TEST_F(GsUsbTest, ReturnsNoFrameWhenTheTimeoutElapses) {
    auto bus = this->open();
    Frame frame;
    const auto start = std::chrono::steady_clock::now();
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, 20 * x::telem::MILLISECOND)));
    const auto elapsed = std::chrono::steady_clock::now() - start;
    EXPECT_GE(elapsed, std::chrono::milliseconds(20));
    EXPECT_LT(elapsed, std::chrono::milliseconds(500));
}

TEST_F(GsUsbTest, SendsAHostFrameWithACyclingEchoId) {
    auto bus = this->open();
    ASSERT_NIL(bus->send(Frame{.id = 0x100, .length = 2}));
    ASSERT_NIL(bus->send(Frame{.id = 0x101, .length = 2}));
    const auto sent = this->api->sent(SERIAL);
    ASSERT_EQ(sent.size(), 2);
    EXPECT_EQ(sent[0].size(), 20);
    EXPECT_EQ(u32(sent[0], 0), 0);
    EXPECT_EQ(u32(sent[1], 0), 1);
    EXPECT_EQ(u32(sent[1], 4), 0x101);
}

TEST_F(GsUsbTest, KeepsSendingWhileNoBusReceives) {
    auto bus = this->open();
    for (std::uint32_t i = 0; i < 1000; i++) {
        const auto err = bus->send(data(i & 0x7FF));
        ASSERT_FALSE(err) << i << ": " << err;
    }
    EXPECT_EQ(this->api->sent(SERIAL).size(), 1000);
}

TEST_F(GsUsbTest, HoldsReceivedFramesWhileTheBusSendsAndDoesNotReceive) {
    auto bus = this->open();
    for (std::uint8_t i = 0; i < 100; i++) {
        for (int j = 0; j < 10; j++)
            ASSERT_NIL(bus->send(data(0x700)));
        this->deliver(0, data(i, i));
    }
    EXPECT_EQ(this->api->lost(SERIAL), 0);
    Frame frame;
    for (std::uint8_t i = 0; i < 100; i++) {
        ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
        EXPECT_EQ(frame.id, i);
    }
}

TEST_F(GsUsbTest, SharesTheAdapterBetweenItsChannels) {
    auto first = this->open(props("ABC123:0"));
    auto second = this->open(props("ABC123:1"));
    EXPECT_EQ(this->api->controls_of(Request::DEVICE_CONFIG).size(), 1);
    ASSERT_NIL(first->close());
    EXPECT_TRUE(this->api->claimed(SERIAL));
    EXPECT_EQ(this->api->handles(SERIAL), 1);
    ASSERT_NIL(second->close());
    EXPECT_FALSE(this->api->claimed(SERIAL));
    EXPECT_EQ(this->api->handles(SERIAL), 0);
}

TEST_F(GsUsbTest, WaitsForAChannelABusHolds) {
    auto bus = this->open();
    auto [second, err] = this->backend->open(props());
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "ABC123:0: the channel is in use");
}

TEST_F(GsUsbTest, ReopensAChannelAfterItCloses) {
    ASSERT_NIL(this->open()->close());
    auto bus = this->open();
    EXPECT_TRUE(this->api->claimed(SERIAL));
}

TEST_F(GsUsbTest, RoutesEachFrameToTheBusOfItsChannel) {
    auto first = this->open(props("ABC123:0"));
    auto second = this->open(props("ABC123:1"));
    this->deliver(1, data(0x21));
    this->deliver(0, data(0x20));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(first->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x20);
    ASSERT_TRUE(ASSERT_NIL_P(second->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x21);
}

TEST_F(GsUsbTest, ReportsTheFramesAChannelThatIsNotReceivingDropped) {
    auto first = this->open(props("ABC123:0"));
    auto second = this->open(props("ABC123:1"));
    for (std::uint32_t id = 0; id <= MAX_QUEUED; id++)
        this->deliver(1, data(id));
    this->deliver(0, data(0x7FF));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(first->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x7FF);
    ASSERT_TRUE(ASSERT_NIL_P(second->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.type, Type::BUS_ERROR);
    EXPECT_EQ(frame.id, bus_error::CONTROLLER);
    EXPECT_EQ(frame.data[1], bus_error::CONTROLLER_RX_OVERFLOW);
    ASSERT_TRUE(ASSERT_NIL_P(second->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.type, Type::DATA);
    EXPECT_EQ(frame.id, 1);
}

TEST_F(GsUsbTest, ReportsAnUnpluggedAdapterAsUnreachable) {
    auto bus = this->open();
    this->api->unplug(SERIAL);
    Frame frame;
    const auto [got, err] = bus->receive(frame, x::telem::SECOND);
    EXPECT_FALSE(got);
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "gs_usb adapter ABC123: LIBUSB_ERROR_NO_DEVICE");
    ASSERT_MATCHES(bus->send(data(0x1)), UNREACHABLE_ERROR);
    ASSERT_NIL(bus->close());
}

TEST_F(GsUsbTest, ReopensAnAdapterAfterAReplug) {
    auto bus = this->open();
    this->api->unplug(SERIAL);
    ASSERT_NIL(bus->close());
    auto [missing, err] = this->backend->open(props());
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "no gs_usb adapter with serial number ABC123 is attached");
    this->api->plug(SERIAL);
    bus = this->open();
    ASSERT_NE(bus, nullptr);
    this->deliver(0, data(0x42));
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x42);
}

TEST_F(GsUsbTest, OpensAReplugWhileAnotherChannelHoldsTheUnpluggedAdapter) {
    auto first = this->open(props("ABC123:0"));
    auto second = this->open(props("ABC123:1"));
    this->api->unplug(SERIAL);
    Frame frame;
    ASSERT_MATCHES(second->receive(frame, x::telem::SECOND).second, UNREACHABLE_ERROR);
    this->api->plug(SERIAL);
    ASSERT_NIL(first->close());
    first = this->open(props("ABC123:0"));
    ASSERT_NE(first, nullptr);
    ASSERT_NIL(first->send(data(0x5)));
    EXPECT_EQ(this->api->sent(SERIAL).size(), 1);
}

TEST_F(GsUsbTest, ResetsReleasesAndClosesOnClose) {
    auto bus = this->open();
    ASSERT_NIL(bus->close());
    EXPECT_EQ(this->api->controls_of(Request::MODE).size(), 3);
    EXPECT_FALSE(this->api->started(SERIAL, 0));
    EXPECT_FALSE(this->api->claimed(SERIAL));
    EXPECT_EQ(this->api->handles(SERIAL), 0);
}
}
