// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/can/can.h"
#include "driver/can/loopback/loopback.h"

namespace driver::can {
TEST(DLC, MapsEveryCodeToItsFDLength) {
    const std::array<std::uint8_t, 16> expected =
        {0, 1, 2, 3, 4, 5, 6, 7, 8, 12, 16, 20, 24, 32, 48, 64};
    for (std::uint8_t dlc = 0; dlc < 16; dlc++) {
        EXPECT_EQ(dlc_to_length(dlc, true), expected[dlc]);
        ASSERT_EQ(length_to_dlc(expected[dlc]), dlc);
    }
}

TEST(DLC, CapsClassicCodesAboveEightAtEightBytes) {
    EXPECT_EQ(dlc_to_length(8, false), 8);
    EXPECT_EQ(dlc_to_length(9, false), 8);
    EXPECT_EQ(dlc_to_length(15, false), 8);
}

TEST(DLC, HasNoCodeForALengthBetweenFDSteps) {
    EXPECT_FALSE(length_to_dlc(9).has_value());
    EXPECT_FALSE(length_to_dlc(33).has_value());
    EXPECT_FALSE(length_to_dlc(65).has_value());
}

TEST(Validate, AcceptsAClassicDataFrame) {
    Frame frame{.id = 0x123, .length = 8};
    ASSERT_NIL(validate(frame, false));
}

TEST(Validate, AcceptsAnExtendedFDFrameOnAnFDBus) {
    Frame frame{
        .id = 0x1ABCDEF0,
        .extended = true,
        .fd = true,
        .bitrate_switched = true,
        .length = 64,
    };
    ASSERT_NIL(validate(frame, true));
}

TEST(Validate, RejectsAStandardIdentifierWiderThanElevenBits) {
    const auto err = validate(Frame{.id = 0x800}, false);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "identifier 0x800 exceeds the 11-bit range");
}

TEST(Validate, RejectsAnExtendedIdentifierWiderThanTwentyNineBits) {
    const auto err = validate(Frame{.id = 0x20000000, .extended = true}, false);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "identifier 0x20000000 exceeds the 29-bit range");
}

TEST(Validate, RejectsAnFDFrameOnAClassicBus) {
    const auto err = validate(Frame{.fd = true}, false);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "a CAN FD frame cannot be sent on a classic CAN bus");
}

TEST(Validate, RejectsAClassicFrameLongerThanEightBytes) {
    const auto err = validate(Frame{.length = 12}, true);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "length 12 is not a valid classic CAN length");
}

TEST(Validate, RejectsAnFDLengthWithNoDataLengthCode) {
    const auto err = validate(Frame{.fd = true, .length = 10}, true);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "length 10 is not a valid CAN FD length");
}

TEST(Validate, RejectsAnFDRemoteFrame) {
    const auto err = validate(Frame{.fd = true, .type = Type::REMOTE}, true);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "CAN FD has no remote frames");
}

TEST(Validate, RejectsABitrateSwitchOnAClassicFrame) {
    const auto err = validate(Frame{.bitrate_switched = true}, true);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "only a CAN FD frame can switch bitrates");
}

TEST(Validate, RejectsABusErrorFrame) {
    const auto err = validate(Frame{.type = Type::BUS_ERROR}, true);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "a bus error frame cannot be sent");
}

TEST(BusError, DescribesEachClassItReports) {
    auto frame = bus_error::make(
        bus_error::NO_ACK | bus_error::CONTROLLER | bus_error::COUNTERS,
        bus_error::CONTROLLER_RX_OVERFLOW | bus_error::CONTROLLER_TX_WARNING,
        x::telem::TimeStamp::now()
    );
    frame.data[6] = 97;
    frame.data[7] = 3;
    EXPECT_EQ(
        bus_error::describe(frame),
        "the adapter reported no acknowledgement, receive overflow, frames lost, "
        "transmit error warning, 97 transmit and 3 receive errors"
    );
}

TEST(BusError, DescribesTheKindOfAProtocolViolation) {
    auto frame = bus_error::make(bus_error::PROTOCOL, 0, x::telem::TimeStamp::now());
    EXPECT_EQ(bus_error::describe(frame), "the adapter reported protocol violation");
    frame.data[2] = bus_error::PROTOCOL_STUFF | bus_error::PROTOCOL_FORM;
    EXPECT_EQ(
        bus_error::describe(frame),
        "the adapter reported form error, stuff error"
    );
}

TEST(BusError, DescribesAControllerFaultWithoutDetails) {
    const auto frame = bus_error::make(
        bus_error::CONTROLLER,
        0,
        x::telem::TimeStamp::now()
    );
    EXPECT_EQ(bus_error::describe(frame), "the adapter reported controller fault");
}

TEST(BusError, ReportsNoFaultForAReturnToErrorActive) {
    auto frame = bus_error::make(
        bus_error::CONTROLLER | bus_error::COUNTERS,
        bus_error::CONTROLLER_ACTIVE,
        x::telem::TimeStamp::now()
    );
    EXPECT_EQ(bus_error::describe(frame), std::nullopt);
    frame.id |= bus_error::RESTARTED;
    EXPECT_EQ(
        bus_error::describe(frame),
        "the adapter reported controller restarted, 0 transmit and 0 receive errors"
    );
}

TEST(BusError, DescribesAFrameWithNoClass) {
    const auto frame = bus_error::make(0, 0, x::telem::TimeStamp::now());
    EXPECT_EQ(bus_error::describe(frame), "the adapter reported an unknown bus error");
}

namespace {
const auto US = x::telem::MICROSECOND;
const std::uint64_t WRAP = 1ULL << 32;
const x::telem::TimeStamp START(1000 * x::telem::SECOND);

/// @returns the host time at which a counter with a tick of US reads count.
x::telem::TimeStamp at(const std::uint64_t count) {
    return START + US * static_cast<std::int64_t>(count);
}
}

TEST(Counter, ExtendsAcrossWrapsItSees) {
    Counter counter(US);
    for (const std::uint64_t count:
         {0xFFFFFFF0ULL, 0xFFFFFFFFULL, WRAP + 5, WRAP + 0x10, 2 * WRAP + 1})
        EXPECT_EQ(counter.extend(static_cast<std::uint32_t>(count), at(count)), count);
}

TEST(Counter, CountsAWrapMissedWhileTheBusWasQuiet) {
    Counter counter(US);
    EXPECT_EQ(counter.extend(100, at(100)), 100);
    const auto count = 100 + WRAP + 1000;
    EXPECT_EQ(counter.extend(static_cast<std::uint32_t>(count), at(count)), count);
}

TEST(Counter, CountsSeveralMissedWraps) {
    Counter counter(US);
    EXPECT_EQ(counter.extend(100, at(100)), 100);
    const auto count = 100 + 3 * WRAP - 50;
    EXPECT_EQ(counter.extend(static_cast<std::uint32_t>(count), at(count)), count);
}

TEST(Counter, IgnoresHostDelayShorterThanHalfAWrap) {
    Counter counter(US);
    EXPECT_EQ(counter.extend(100, at(100)), 100);
    const auto late = at(200) + WRAP / 2 * US - x::telem::SECOND;
    EXPECT_EQ(counter.extend(200, late), 200);
    EXPECT_EQ(counter.extend(150, at(WRAP + 150)), WRAP + 150);
}

TEST(Counter, NeverCountsWrapsWhenTheHostClockRunsBackward) {
    Counter counter(US);
    EXPECT_EQ(counter.extend(1000, at(1000)), 1000);
    EXPECT_EQ(counter.extend(2000, START - x::telem::SECOND), 2000);
}

TEST(Counter, ScalesHostTimeByTheTick) {
    const auto tick = 10 * US;
    Counter counter(tick);
    EXPECT_EQ(counter.extend(7, START), 7);
    const auto count = 7 + WRAP + 3;
    const auto host = START + tick * static_cast<std::int64_t>(WRAP + 3);
    EXPECT_EQ(counter.extend(static_cast<std::uint32_t>(count), host), count);
}

TEST(Aligner, FollowsAnAdapterClockThatDrifts) {
    Aligner aligner;
    const auto step = 10 * x::telem::MILLISECOND;
    // The adapter clock runs 100 ppm fast, and each frame takes up to 3 ms to arrive.
    x::telem::TimeStamp host(1000 * x::telem::SECOND);
    x::telem::TimeStamp adapter(5 * x::telem::SECOND);
    x::telem::TimeStamp last{};
    x::telem::TimeSpan error;
    for (int i = 0; i < 360000; i++) {
        host = host + step;
        adapter = adapter + step + step / 10000;
        const auto delay = (i % 4) * x::telem::MILLISECOND;
        const auto aligned = aligner.align(adapter, host + delay);
        if (i > 0) { ASSERT_GT(aligned, last) << i; }
        last = aligned;
        error = aligned - host;
    }
    EXPECT_LT(std::abs(error.nanoseconds()), (2 * x::telem::MILLISECOND).nanoseconds());
}

TEST(Open, OpensTheBackendThePropertiesName) {
    const Backends backends = {{"loopback", std::make_shared<loopback::Backend>()}};
    synnax::can::Properties props;
    props.backend = "loopback";
    props.channel = "vcan0";
    const auto bus = ASSERT_NIL_P(open(backends, props));
    ASSERT_NE(bus, nullptr);
}

TEST(Open, RejectsAnUnknownBackend) {
    synnax::can::Properties props;
    props.backend = "missing";
    auto [bus, err] = open(Backends{}, props);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "unknown CAN backend 'missing'");
}

TEST(Open, ReturnsTheLoadErrorOfAnUnavailableBackend) {
    const x::errors::Error load_err(UNSUPPORTED_ERROR, "not on this platform");
    const Backends backends = {{"pcan", std::make_shared<Unavailable>(load_err)}};
    synnax::can::Properties props;
    props.backend = "pcan";
    auto [bus, err] = open(backends, props);
    ASSERT_MATCHES(err, UNSUPPORTED_ERROR);
    EXPECT_EQ(err.data, "not on this platform");
    auto [channels, scan_err] = backends.at("pcan")->scan();
    ASSERT_MATCHES(scan_err, UNSUPPORTED_ERROR);
    EXPECT_TRUE(channels.empty());
}
}
