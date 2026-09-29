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

TEST(Counter, ExtendsAcrossAWrap) {
    Counter counter;
    EXPECT_EQ(counter.extend(0xFFFFFFF0), 0xFFFFFFF0);
    EXPECT_EQ(counter.extend(0xFFFFFFFF), 0xFFFFFFFF);
    EXPECT_EQ(counter.extend(0x00000005), 0x100000005);
    EXPECT_EQ(counter.extend(0x00000010), 0x100000010);
    EXPECT_EQ(counter.extend(0x00000001), 0x200000001);
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
