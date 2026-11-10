// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <set>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/can/nixnet/mock.h"
#include "driver/can/nixnet/nixnet.h"

namespace driver::can::nixnet {
namespace {
synnax::can::Properties props(const bool fd = false, const bool listen_only = false) {
    synnax::can::Properties p;
    p.backend = synnax::can::BACKEND_NIXNET;
    p.channel = "CAN1";
    p.bitrate = 500000;
    p.fd = fd;
    p.data_bitrate = 2000000;
    p.listen_only = listen_only;
    return p;
}

class NIXNETTest : public ::testing::Test {
protected:
    std::shared_ptr<MockAPI> api = std::make_shared<MockAPI>();
    Backend backend{api};

    void SetUp() override {
        this->api->add({.name = "CAN1", .device = "NI 9862", .serial = 0x1A2B});
    }

    std::unique_ptr<can::Bus> open(const synnax::can::Properties &p = props()) {
        auto [bus, err] = this->backend.open(p);
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }
};

/// @returns a raw NI-XNET bus error frame with its payload.
std::vector<std::uint8_t> bus_error_bytes(const std::array<std::uint8_t, 5> &payload) {
    auto bytes = encode(Frame{.length = 5}, false);
    bytes[12] = nxFrameType_CAN_BusError;
    std::memcpy(bytes.data() + HEADER_SIZE, payload.data(), payload.size());
    return bytes;
}

/// @returns the IDs of every property the backend set on a session.
std::set<u32> property_ids(const MockAPI::Session &session) {
    std::set<u32> ids;
    for (const auto &[id, value]: session.properties)
        ids.insert(id);
    return ids;
}
}

TEST(NIXNETFrame, PadsThePayloadToAMultipleOfEightBytes) {
    EXPECT_EQ(raw_size(0), 24);
    EXPECT_EQ(raw_size(8), 24);
    EXPECT_EQ(raw_size(12), 32);
    EXPECT_EQ(raw_size(64), 80);
}

TEST(NIXNETFrame, EncodesAClassicStandardFrame) {
    Frame frame{.id = 0x123, .length = 3};
    frame.data[0] = 0xAA;
    frame.data[2] = 0xCC;
    const auto bytes = encode(frame, false);
    const std::vector<std::uint8_t> expected = {
        0,    0,    0,    0, 0, 0, 0, 0, // timestamp
        0x23, 0x01, 0,    0, // identifier
        0x00, // type
        0,    0, // flags and info
        3, // payload length
        0xAA, 0,    0xCC, 0, 0, 0, 0, 0,
    };
    EXPECT_EQ(bytes, expected);
}

TEST(NIXNETFrame, MarksAClassicFrameOnAnFDInterfaceAsCAN20) {
    const auto bytes = encode(Frame{.id = 1}, true);
    EXPECT_EQ(bytes[12], nxFrameType_CAN20_Data);
}

TEST(NIXNETFrame, EncodesAnExtendedFDFrameWithABitrateSwitch) {
    Frame frame{
        .id = 0x1ABCDEF0,
        .extended = true,
        .fd = true,
        .bitrate_switched = true,
        .length = 12,
    };
    frame.data[11] = 0x5A;
    const auto bytes = encode(frame, true);
    ASSERT_EQ(bytes.size(), 32);
    EXPECT_EQ(bytes[8], 0xF0);
    EXPECT_EQ(bytes[9], 0xDE);
    EXPECT_EQ(bytes[10], 0xBC);
    EXPECT_EQ(bytes[11], 0x3A);
    EXPECT_EQ(bytes[12], nxFrameType_CANFDBRS_Data);
    EXPECT_EQ(bytes[15], 12);
    EXPECT_EQ(bytes[16 + 11], 0x5A);
}

TEST(NIXNETFrame, DecodesWhatItEncodes) {
    Frame sent{.id = 0x7FF, .fd = true, .length = 64};
    for (std::uint8_t i = 0; i < 64; i++)
        sent.data[i] = i;
    auto bytes = encode(sent, true);
    const std::uint64_t ticks = UNIX_EPOCH_TICKS + 17000000000000000ULL;
    for (std::size_t i = 0; i < 8; i++)
        bytes[i] = static_cast<std::uint8_t>(ticks >> (8 * i));
    Frame got;
    const auto size = ASSERT_NIL_P(decode(bytes, got));
    EXPECT_EQ(size, 80);
    EXPECT_EQ(got.id, 0x7FF);
    EXPECT_FALSE(got.extended);
    EXPECT_TRUE(got.fd);
    EXPECT_FALSE(got.bitrate_switched);
    EXPECT_EQ(got.length, 64);
    EXPECT_EQ(got.data[63], 63);
    EXPECT_EQ(got.time.nanoseconds(), 1700000000000000000LL);
    EXPECT_EQ(got.clock, Clock::HARDWARE);
}

TEST(NIXNETFrame, DecodesARemoteFrame) {
    auto bytes = encode(Frame{.id = 5, .type = Type::REMOTE, .length = 2}, false);
    Frame got;
    ASSERT_NIL_P(decode(bytes, got));
    EXPECT_EQ(got.type, Type::REMOTE);
    EXPECT_EQ(got.length, 2);
}

TEST(NIXNETFrame, RejectsATruncatedFrame) {
    auto bytes = encode(Frame{.fd = true, .length = 32}, true);
    bytes.resize(30);
    Frame got;
    auto [size, err] = decode(bytes, got);
    ASSERT_MATCHES(err, CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "NI-XNET returned a truncated frame");
}

TEST(NIXNETTranslate, WaitsForAnAbsentLostOrBusyInterface) {
    MockAPI api;
    for (const auto status:
         {nxErrInvalidInterface,
          nxErrCommunicationLost,
          nxErrLowLevelCommunicationTimeout,
          nxErrDongleCommunicationLost,
          nxErrCdaqConnectionLost,
          nxErrCdaqTransferAborted,
          nxErrFirmwareNoResponse,
          nxErrPowerSuspended,
          nxErrMissingBusPower,
          nxErrDifferentClusterOpen}) {
        const auto err = translate(api, status, "NI-XNET CAN1");
        ASSERT_MATCHES(err, UNREACHABLE_ERROR);
        EXPECT_EQ(err.data, "NI-XNET CAN1: mock status " + std::to_string(status));
    }
}

TEST(NIXNETTranslate, ReportsAnInvalidBitrateOrPayloadAsAConfigurationError) {
    MockAPI api;
    for (const auto status:
         {nxErrInvalidBaudRate,
          nxErrBaudRateXcvrMismatch,
          nxErrInvalidBitTimings,
          nxErrPayloadTooLarge})
        ASSERT_MATCHES(translate(api, status, "NI-XNET CAN1"), CONFIG_ERROR);
}

TEST(NIXNETTranslate, ReportsAFullOutputQueueAsATemporaryHardwareError) {
    MockAPI api;
    const auto err = translate(api, nxErrOutputQueueOverflow, "NI-XNET CAN1");
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_FALSE(err.matches(UNREACHABLE_ERROR));
}

TEST(NIXNETTranslate, ReportsAnyOtherStatusAsACriticalHardwareError) {
    MockAPI api;
    for (const auto status:
         {nxErrInternalError,
          nxErrInvalidSessionHandle,
          nxErrMaxSessions,
          nxErrInterfaceRunning})
        ASSERT_MATCHES(translate(api, status, "NI-XNET CAN1"), CRITICAL_HARDWARE_ERROR);
}

TEST(NIXNETFrame, DecodesABusErrorFrameIntoItsStateErrorAndCounters) {
    Frame got;
    ASSERT_NIL_P(decode(bus_error_bytes({1, 128, 5, nxCANLastErr_Ack, 0}), got));
    EXPECT_EQ(got.type, Type::BUS_ERROR);
    EXPECT_EQ(
        bus_error::describe(got),
        "the adapter reported no acknowledgement, bus error, transmit error passive, "
        "128 transmit and 5 receive errors"
    );
    ASSERT_NIL_P(decode(bus_error_bytes({2, 255, 0, nxCANLastErr_Stuff, 1}), got));
    EXPECT_EQ(
        bus_error::describe(got),
        "the adapter reported transceiver fault, bus off, bus error, stuff error, 255 "
        "transmit and 0 receive errors"
    );
    ASSERT_NIL_P(decode(bus_error_bytes({0, 3, 100, nxCANLastErr_Bit0, 0}), got));
    EXPECT_EQ(
        bus_error::describe(got),
        "the adapter reported bus error, dominant bit error, receive error warning, 3 "
        "transmit and 100 receive errors"
    );
}

TEST_F(NIXNETTest, OpensAnInputAndAnOutputSessionAtTheBitrate) {
    auto bus = this->open();
    ASSERT_EQ(this->api->created(), 2);
    const auto in = this->api->session(1);
    const auto out = this->api->session(2);
    EXPECT_EQ(in.interface, "CAN1");
    EXPECT_EQ(in.mode, nxMode_FrameInStream);
    EXPECT_EQ(out.mode, nxMode_FrameOutStream);
    EXPECT_EQ(
        property_ids(in),
        (std::set<u32>{nxPropSession_IntfBaudRate64, nxPropSession_IntfBusErrToInStrm})
    );
    EXPECT_EQ(in.property<u8>(nxPropSession_IntfBusErrToInStrm), 1);
    EXPECT_EQ(property_ids(out), std::set<u32>{nxPropSession_IntfBaudRate64});
    for (const auto &session: {in, out}) {
        EXPECT_EQ(session.database, ":memory:");
        EXPECT_EQ(session.property<u64>(nxPropSession_IntfBaudRate64), 500000);
        EXPECT_TRUE(session.started);
    }
}

TEST_F(NIXNETTest, OpensFDSessionsOnTheFDDatabaseAtTheDataBitrate) {
    auto bus = this->open(props(true));
    ASSERT_EQ(this->api->created(), 2);
    for (const nxSessionRef_t ref: {1, 2}) {
        const auto session = this->api->session(ref);
        EXPECT_EQ(session.database, ":can_fd_brs:");
        EXPECT_TRUE(property_ids(session).contains(nxPropSession_IntfCanFdBaudRate64));
        EXPECT_EQ(session.property<u64>(nxPropSession_IntfBaudRate64), 500000);
        EXPECT_EQ(session.property<u64>(nxPropSession_IntfCanFdBaudRate64), 2000000);
    }
}

TEST_F(NIXNETTest, OpensOnlyAListenOnlyInputSessionWhenListenOnly) {
    auto bus = this->open(props(false, true));
    ASSERT_EQ(this->api->created(), 1);
    const auto in = this->api->session(1);
    EXPECT_EQ(
        property_ids(in),
        (std::set<u32>{
            nxPropSession_IntfBaudRate64,
            nxPropSession_IntfBusErrToInStrm,
            nxPropSession_IntfCANLstnOnly,
        })
    );
    EXPECT_EQ(in.property<u8>(nxPropSession_IntfCANLstnOnly), 1);
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel CAN1 is listen only");
}

TEST_F(NIXNETTest, WaitsForAnInterfaceTheSystemDoesNotHave) {
    auto p = props();
    p.channel = "CAN9";
    auto [bus, err] = this->backend.open(p);
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "NI-XNET has no interface named CAN9");
}

TEST_F(
    NIXNETTest,
    WaitsForAnotherApplicationToStopRunningTheInterfaceAtAnotherBitrate
) {
    this->api->hold("CAN1", 250000);
    auto [bus, err] = this->backend.open(props());
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "NI-XNET CAN1: mock status " + std::to_string(nxErrDifferentClusterOpen)
    );
    EXPECT_TRUE(this->api->session(1).cleared);
    EXPECT_TRUE(this->api->session(2).cleared);
    this->api->hold("CAN1", std::nullopt);
    this->open();
}

TEST_F(NIXNETTest, ReceivesEveryFrameOfOneRead) {
    auto bus = this->open();
    auto chunk = encode(Frame{.id = 1, .length = 1}, false);
    const auto second = encode(Frame{.id = 2, .extended = true, .length = 8}, false);
    chunk.insert(chunk.end(), second.begin(), second.end());
    this->api->deliver("CAN1", {.bytes = chunk});
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 1);
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 2);
    EXPECT_TRUE(frame.extended);
}

TEST_F(NIXNETTest, SkipsTransmitEchoes) {
    auto bus = this->open();
    auto echo = encode(Frame{.id = 1}, false);
    echo[13] = nxFrameFlags_TransmitEcho;
    this->api->deliver("CAN1", {.bytes = echo});
    this->api->deliver("CAN1", {.bytes = encode(Frame{.id = 2}, false)});
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 2);
}

TEST_F(NIXNETTest, ReportsAnInputQueueOverflowBeforeTheFramesThatFollow) {
    auto bus = this->open();
    this->api->deliver("CAN1", {.status = nxErrInputQueueOverflow});
    this->api->deliver("CAN1", {.bytes = encode(Frame{.id = 3}, false)});
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.type, Type::BUS_ERROR);
    EXPECT_EQ(
        bus_error::describe(frame),
        "the adapter reported receive overflow, frames lost"
    );
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 3);
}

TEST_F(NIXNETTest, ReceivesABusErrorFrame) {
    auto bus = this->open();
    this->api->deliver(
        "CAN1",
        {.bytes = bus_error_bytes({1, 130, 0, nxCANLastErr_Form, 0})}
    );
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.type, Type::BUS_ERROR);
    EXPECT_EQ(
        bus_error::describe(frame),
        "the adapter reported bus error, form error, transmit error passive, 130 "
        "transmit and 0 receive errors"
    );
}

TEST_F(NIXNETTest, ReturnsNoFrameWhenTheTimeoutElapses) {
    auto bus = this->open();
    Frame frame;
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, x::telem::MILLISECOND * 5)));
}

TEST_F(NIXNETTest, ReportsARemovedInterfaceAsUnreachable) {
    auto bus = this->open();
    this->api->unplug("CAN1");
    Frame frame;
    auto [received, err] = bus->receive(frame, x::telem::SECOND);
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "NI-XNET CAN1: mock status " + std::to_string(nxErrCommunicationLost)
    );
    ASSERT_MATCHES(bus->send(Frame{.id = 1}), UNREACHABLE_ERROR);
}

TEST_F(NIXNETTest, ReopensAnInterfaceAfterItReturns) {
    auto bus = this->open();
    this->api->unplug("CAN1");
    Frame frame;
    ASSERT_MATCHES(bus->receive(frame, x::telem::SECOND).second, UNREACHABLE_ERROR);
    ASSERT_NIL(bus->close());
    auto [absent, err] = this->backend.open(props());
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    this->api->plug("CAN1");
    bus = this->open();
    this->api->deliver("CAN1", {.bytes = encode(Frame{.id = 9}, false)});
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 9);
}

TEST_F(NIXNETTest, SendsThroughTheOutputSession) {
    auto bus = this->open(props(true));
    ASSERT_NIL(bus->send(Frame{.id = 0x10, .fd = true, .length = 16}));
    const auto written = this->api->written();
    ASSERT_EQ(written.size(), 1);
    EXPECT_EQ(written[0].size(), 32);
    EXPECT_EQ(written[0][12], nxFrameType_CANFD_Data);
}

TEST_F(NIXNETTest, ReportsAFullOutputQueueAsATemporaryHardwareError) {
    auto bus = this->open();
    this->api->write_status = nxErrOutputQueueOverflow;
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_FALSE(err.matches(UNREACHABLE_ERROR));
    EXPECT_EQ(err.data, "NI-XNET CAN1: transmit queue is full");
}

TEST_F(NIXNETTest, ClearsBothSessionsOnClose) {
    auto bus = this->open();
    ASSERT_NIL(bus->close());
    EXPECT_TRUE(this->api->session(1).cleared);
    EXPECT_TRUE(this->api->session(2).cleared);
}

TEST_F(NIXNETTest, ScansEveryCANInterfaceThatIsPresent) {
    this->api->add({.name = "CAN2", .device = "PXIe-8510", .serial = 0xFF});
    this->api->add({.name = "CAN3", .device = "USB-8502", .serial = 0x10});
    this->api->unplug("CAN3");
    const auto channels = ASSERT_NIL_P(this->backend.scan());
    ASSERT_EQ(channels.size(), 2);
    EXPECT_EQ(channels[0].backend, "nixnet");
    EXPECT_EQ(channels[0].name, "CAN1");
    EXPECT_EQ(channels[0].description, "NI 9862, S/N 1A2B");
    EXPECT_EQ(channels[1].name, "CAN2");
    EXPECT_EQ(channels[1].description, "PXIe-8510, S/N FF");
}
}
