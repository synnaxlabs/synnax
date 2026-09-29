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

#include "x/cpp/test/test.h"

#include "driver/can/pcan/mock.h"
#include "driver/can/pcan/pcan.h"

namespace driver::can::pcan {
namespace {
synnax::can::Properties props(const bool fd = false, const bool listen_only = false) {
    synnax::can::Properties p;
    p.backend = synnax::can::BACKEND_PCAN;
    p.channel = "PCAN_USBBUS1";
    p.bitrate = 500000;
    p.fd = fd;
    p.data_bitrate = 2000000;
    p.listen_only = listen_only;
    return p;
}

class PCANTest : public ::testing::Test {
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

TEST(PCANChannel, ParsesUSBPCIAndLANNames) {
    EXPECT_EQ(parse_channel("PCAN_USBBUS1"), 0x51);
    EXPECT_EQ(parse_channel("PCAN_USBBUS8"), 0x58);
    EXPECT_EQ(parse_channel("PCAN_USBBUS9"), 0x509);
    EXPECT_EQ(parse_channel("PCAN_USBBUS16"), 0x510);
    EXPECT_EQ(parse_channel("PCAN_PCIBUS16"), 0x410);
    EXPECT_EQ(parse_channel("PCAN_LANBUS3"), 0x803);
    EXPECT_FALSE(parse_channel("PCAN_USBBUS17").has_value());
    EXPECT_FALSE(parse_channel("can0").has_value());
}

TEST(PCANBaudCode, MapsAStandardBitrate) {
    EXPECT_EQ(ASSERT_NIL_P(baud_code(500000)), PCAN_BAUD_500K);
    EXPECT_EQ(ASSERT_NIL_P(baud_code(83333)), PCAN_BAUD_83K);
}

TEST(PCANBaudCode, RejectsABitrateWithNoCode) {
    auto [code, err] = baud_code(400000);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "PCAN-Basic cannot run a classic CAN bus at 400000 bit/s. Supported bitrates: "
        "1000000, 800000, 500000, 250000, 125000, 100000, 95238, 83333, 50000, "
        "47619, 33333, 20000, 10000, 5000"
    );
}

TEST(PCANFDBitrate, BuildsTheBitrateStringOfA500kBusWithA2MDataPhase) {
    EXPECT_EQ(
        ASSERT_NIL_P(fd_bitrate(500000, 2000000)),
        "f_clock=80000000,nom_brp=1,nom_tseg1=139,nom_tseg2=20,nom_sjw=20,"
        "data_brp=1,data_tseg1=29,data_tseg2=10,data_sjw=10"
    );
}

TEST_F(PCANTest, InitializesAClassicChannelWithItsBaudCode) {
    auto bus = this->open();
    EXPECT_EQ(this->api->btr0btr1, PCAN_BAUD_500K);
    EXPECT_TRUE(this->api->listen_only.empty());
}

TEST_F(PCANTest, InitializesAnFDChannelWithItsBitrateString) {
    auto bus = this->open(props(true));
    EXPECT_EQ(this->api->fd_bitrate.substr(0, 16), "f_clock=80000000");
}

TEST_F(PCANTest, PutsAListenOnlyChannelInListenOnlyModeBeforeInitializing) {
    auto bus = this->open(props(false, true));
    ASSERT_EQ(this->api->listen_only.size(), 1);
    EXPECT_EQ(this->api->listen_only[0], PCAN_USBBUS1);
}

TEST_F(PCANTest, RejectsAnUnknownChannel) {
    auto p = props();
    p.channel = "can0";
    auto [bus, err] = this->backend.open(p);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "unknown PCAN-Basic channel 'can0'. Expected a name such as PCAN_USBBUS1"
    );
}

TEST_F(PCANTest, ReportsAnInitializeFailureAsATemporaryHardwareError) {
    this->api->initialize_status = 0x00400;
    auto [bus, err] = this->backend.open(props());
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "PCAN_USBBUS1: mock status 1024");
}

TEST_F(PCANTest, ReportsAnIllegalParameterAsAConfigurationError) {
    this->api->initialize_status = PCAN_ERROR_ILLPARAMVAL;
    auto [bus, err] = this->backend.open(props(true));
    ASSERT_MATCHES(err, CONFIG_ERROR);
}

TEST_F(PCANTest, ReceivesAClassicExtendedFrameWithItsHardwareTime) {
    auto bus = this->open();
    MockAPI::Received read;
    read.msg.ID = 0x18FEF100;
    read.msg.MSGTYPE = PCAN_MESSAGE_EXTENDED;
    read.msg.DLC = 3;
    read.msg.DATA[0] = 0x01;
    read.msg.DATA[2] = 0x03;
    read.timestamp_us = 0x100000000ULL * 1000 + 1234567;
    this->api->reads.push_back(read);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x18FEF100);
    EXPECT_TRUE(frame.extended);
    EXPECT_FALSE(frame.fd);
    EXPECT_EQ(frame.type, Type::DATA);
    EXPECT_EQ(frame.length, 3);
    EXPECT_EQ(frame.data[0], 0x01);
    EXPECT_EQ(frame.data[2], 0x03);
    EXPECT_EQ(frame.clock, Clock::HARDWARE);
    EXPECT_EQ(
        frame.time.nanoseconds(),
        static_cast<std::int64_t>(0x100000000ULL * 1000 + 1234567) * 1000
    );
}

TEST_F(PCANTest, ReceivesAnFDFrameThroughItsDataLengthCode) {
    auto bus = this->open(props(true));
    MockAPI::Received read;
    read.msg.ID = 0x123;
    read.msg.MSGTYPE = PCAN_MESSAGE_FD | PCAN_MESSAGE_BRS | PCAN_MESSAGE_ESI;
    read.msg.DLC = 13;
    read.msg.DATA[31] = 0xEE;
    read.timestamp_us = 42;
    this->api->reads.push_back(read);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_TRUE(frame.fd);
    EXPECT_TRUE(frame.bitrate_switched);
    EXPECT_TRUE(frame.error_passive);
    EXPECT_EQ(frame.length, 32);
    EXPECT_EQ(frame.data[31], 0xEE);
    EXPECT_EQ(frame.time.nanoseconds(), 42000);
}

TEST_F(PCANTest, SkipsStatusMessages) {
    auto bus = this->open();
    MockAPI::Received status;
    status.msg.MSGTYPE = PCAN_MESSAGE_STATUS;
    MockAPI::Received data;
    data.msg.ID = 0x7;
    this->api->reads.push_back(status);
    this->api->reads.push_back(data);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x7);
}

TEST_F(PCANTest, ReceivesARemoteFrame) {
    auto bus = this->open();
    MockAPI::Received read;
    read.msg.ID = 0x10;
    read.msg.MSGTYPE = PCAN_MESSAGE_RTR;
    read.msg.DLC = 4;
    this->api->reads.push_back(read);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.type, Type::REMOTE);
    EXPECT_EQ(frame.length, 4);
}

TEST_F(PCANTest, RegistersAReceiveEventOnTheChannel) {
    auto bus = this->open();
    EXPECT_EQ(this->api->receive_events, std::vector<TPCANHandle>{PCAN_USBBUS1});
    EXPECT_TRUE(this->api->released.empty());
}

TEST_F(PCANTest, UninitializesTheChannelWhenTheReceiveEventFails) {
    this->api->receive_event_error = x::errors::Error(
        CRITICAL_HARDWARE_ERROR,
        "cannot register the receive event: mock status 1"
    );
    auto [bus, err] = this->backend.open(props());
    ASSERT_MATCHES(err, CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(
        err.data,
        "PCAN_USBBUS1: cannot register the receive event: mock status 1"
    );
    EXPECT_EQ(this->api->uninitialized, std::vector<TPCANHandle>{PCAN_USBBUS1});
}

TEST_F(PCANTest, WaitsOnTheReceiveEventWhenTheTimeoutElapses) {
    auto bus = this->open();
    Frame frame;
    const auto start = x::telem::TimeStamp::now();
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND * 20)));
    EXPECT_GE(x::telem::TimeStamp::now() - start, x::telem::MILLISECOND * 20);
    EXPECT_GE(this->api->waits, 1);
}

TEST_F(PCANTest, WakesOnTheReceiveEventWhenAFrameArrives) {
    auto bus = this->open();
    std::thread producer([this] {
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
        MockAPI::Received read;
        read.msg.ID = 0x55;
        this->api->receive(read);
    });
    Frame frame;
    const auto start = x::telem::TimeStamp::now();
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND * 5)));
    EXPECT_LT(x::telem::TimeStamp::now() - start, x::telem::SECOND);
    EXPECT_EQ(frame.id, 0x55);
    producer.join();
}

TEST_F(PCANTest, DrainsTheQueuedFramesBeforeWaiting) {
    auto bus = this->open();
    MockAPI::Received first;
    first.msg.ID = 0x1;
    MockAPI::Received second;
    second.msg.ID = 0x2;
    this->api->receive(first);
    this->api->receive(second);
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x1);
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x2);
    EXPECT_EQ(this->api->waits, 0);
}

TEST_F(PCANTest, ReportsAFailedWaitAsACriticalHardwareError) {
    auto bus = this->open();
    this->api->wait_error = x::errors::Error(
        CRITICAL_HARDWARE_ERROR,
        "cannot wait on the receive event: Windows error 6"
    );
    Frame frame;
    auto [received, err] = bus->receive(frame, x::telem::SECOND);
    ASSERT_MATCHES(err, CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(
        err.data,
        "PCAN_USBBUS1: cannot wait on the receive event: Windows error 6"
    );
    EXPECT_FALSE(received);
}

TEST_F(PCANTest, ReportsBusOffAsATemporaryHardwareError) {
    auto bus = this->open();
    MockAPI::Received read;
    read.status = PCAN_ERROR_BUSOFF;
    this->api->reads.push_back(read);
    Frame frame;
    auto [received, err] = bus->receive(frame, x::telem::SECOND);
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "PCAN_USBBUS1: bus off");
}

TEST_F(PCANTest, SendsAClassicFrame) {
    auto bus = this->open();
    Frame frame{.id = 0x1ABCDE, .extended = true, .length = 2};
    frame.data[1] = 0x99;
    ASSERT_NIL(bus->send(frame));
    ASSERT_EQ(this->api->written.size(), 1);
    const auto &msg = this->api->written[0];
    EXPECT_EQ(msg.ID, 0x1ABCDE);
    EXPECT_EQ(msg.MSGTYPE, PCAN_MESSAGE_EXTENDED);
    EXPECT_EQ(msg.LEN, 2);
    EXPECT_EQ(msg.DATA[1], 0x99);
}

TEST_F(PCANTest, SendsAnFDFrameWithItsDataLengthCode) {
    auto bus = this->open(props(true));
    Frame frame{.id = 0x100, .fd = true, .bitrate_switched = true, .length = 48};
    frame.data[47] = 0x31;
    ASSERT_NIL(bus->send(frame));
    ASSERT_EQ(this->api->written_fd.size(), 1);
    const auto &msg = this->api->written_fd[0];
    EXPECT_EQ(msg.MSGTYPE, PCAN_MESSAGE_FD | PCAN_MESSAGE_BRS);
    EXPECT_EQ(msg.DLC, 14);
    EXPECT_EQ(msg.DATA[47], 0x31);
}

TEST_F(PCANTest, RefusesToSendOnAListenOnlyChannel) {
    auto bus = this->open(props(false, true));
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel PCAN_USBBUS1 is listen only");
    EXPECT_TRUE(this->api->written.empty());
}

TEST_F(PCANTest, RejectsAnFDFrameOnAClassicChannel) {
    auto bus = this->open();
    ASSERT_MATCHES(bus->send(Frame{.fd = true}), FRAME_ERROR);
}

TEST_F(PCANTest, ReportsAFullTransmitQueueAsATemporaryHardwareError) {
    auto bus = this->open();
    this->api->write_status = PCAN_ERROR_QXMTFULL;
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "PCAN_USBBUS1: transmit queue is full");
}

TEST_F(PCANTest, UninitializesTheChannelOnceOnClose) {
    auto bus = this->open();
    ASSERT_NIL(bus->close());
    ASSERT_NIL(bus->close());
    bus.reset();
    ASSERT_EQ(this->api->uninitialized.size(), 1);
    EXPECT_EQ(this->api->uninitialized[0], PCAN_USBBUS1);
}

TEST_F(PCANTest, ReleasesTheReceiveEventAfterUninitializingOnClose) {
    auto bus = this->open();
    ASSERT_NIL(bus->close());
    ASSERT_EQ(this->api->released.size(), 1);
    EXPECT_EQ(this->api->released[0].channel, PCAN_USBBUS1);
    EXPECT_TRUE(this->api->released[0].after_uninitialize);
}

TEST_F(PCANTest, ScansTheChannelsThatAreAttached) {
    this->api->channels[PCAN_USBBUS1] = {
        .condition = PCAN_CHANNEL_AVAILABLE,
        .hardware_name = "PCAN-USB FD",
        .features = FEATURE_FD_CAPABLE,
    };
    this->api->channels[0x52] = {
        .condition = PCAN_CHANNEL_OCCUPIED,
        .hardware_name = "PCAN-USB",
    };
    this->api->channels[0x53] = {.condition = PCAN_CHANNEL_UNAVAILABLE};
    const auto channels = ASSERT_NIL_P(this->backend.scan());
    ASSERT_EQ(channels.size(), 2);
    EXPECT_EQ(channels[0].backend, "pcan");
    EXPECT_EQ(channels[0].name, "PCAN_USBBUS1");
    EXPECT_EQ(channels[0].description, "PCAN-USB FD, CAN FD");
    EXPECT_EQ(channels[1].name, "PCAN_USBBUS2");
    EXPECT_EQ(channels[1].description, "PCAN-USB, in use");
}
}
