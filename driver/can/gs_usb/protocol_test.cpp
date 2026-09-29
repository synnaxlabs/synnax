// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/can/gs_usb/protocol.h"

namespace driver::can::gs_usb {
namespace {
std::vector<std::uint8_t> encode(const Frame &frame, const std::uint32_t echo = 0) {
    std::array<std::uint8_t, MAX_FRAME_SIZE> buffer{};
    const auto size = encode_frame(frame, echo, 1, buffer);
    return {buffer.begin(), buffer.begin() + static_cast<std::ptrdiff_t>(size)};
}
}

TEST(GsUsbEncode, EncodesAClassicStandardFrame) {
    Frame frame{.id = 0x123, .length = 3};
    frame.data[0] = 0xAA;
    frame.data[1] = 0xBB;
    frame.data[2] = 0xCC;
    const std::vector<std::uint8_t> expected = {
        0x05, 0x00, 0x00, 0x00, // echo id
        0x23, 0x01, 0x00, 0x00, // can id
        0x03, 0x01, 0x00, 0x00, // dlc, channel, flags, reserved
        0xAA, 0xBB, 0xCC, 0x00, 0x00, 0x00, 0x00, 0x00,
    };
    EXPECT_EQ(encode(frame, 5), expected);
}

TEST(GsUsbEncode, SetsTheExtendedAndRemoteBitsOfTheIdentifier) {
    const auto bytes = encode(
        Frame{.id = 0x1ABCDEF0, .extended = true, .type = Type::REMOTE, .length = 4}
    );
    ASSERT_EQ(bytes.size(), 20);
    EXPECT_EQ(bytes[4], 0xF0);
    EXPECT_EQ(bytes[5], 0xDE);
    EXPECT_EQ(bytes[6], 0xBC);
    EXPECT_EQ(bytes[7], 0xDA);
    EXPECT_EQ(bytes[8], 4);
    EXPECT_EQ(bytes[12], 0);
}

TEST(GsUsbEncode, EncodesAnFDFrameWithItsDataLengthCodeAndFlags) {
    Frame frame{.id = 0x7FF, .fd = true, .bitrate_switched = true, .length = 64};
    frame.data[63] = 0x42;
    const auto bytes = encode(frame);
    ASSERT_EQ(bytes.size(), 76);
    EXPECT_EQ(bytes[8], 15);
    EXPECT_EQ(bytes[10], FRAME_FD | FRAME_BRS);
    EXPECT_EQ(bytes[75], 0x42);
}

TEST(GsUsbDecode, DecodesAClassicExtendedFrameWithATimestamp) {
    const std::vector<std::uint8_t> bytes = {
        0xFF, 0xFF, 0xFF, 0xFF, // echo id: received
        0x00, 0xF1, 0xFE, 0x98, // extended 0x18FEF100
        0x02, 0x00, 0x01, 0x00, // dlc 2, channel 0, overflow
        0x11, 0x22, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x78, 0x56, 0x34, 0x12, // timestamp
    };
    const auto hf = ASSERT_NIL_P(decode_frame(bytes, true));
    EXPECT_EQ(hf.echo_id, ECHO_ID_RX);
    EXPECT_EQ(hf.channel, 0);
    EXPECT_TRUE(hf.overflowed);
    EXPECT_TRUE(hf.frame.extended);
    EXPECT_EQ(hf.frame.id, 0x18FEF100);
    EXPECT_EQ(hf.frame.length, 2);
    EXPECT_EQ(hf.frame.data[1], 0x22);
    ASSERT_TRUE(hf.timestamp_us.has_value());
    EXPECT_EQ(*hf.timestamp_us, 0x12345678);
}

TEST(GsUsbDecode, DecodesAnFDFrameWithItsTimestampAfterSixtyFourDataBytes) {
    std::vector<std::uint8_t> bytes(80, 0);
    bytes[0] = bytes[1] = bytes[2] = bytes[3] = 0xFF;
    bytes[4] = 0x23;
    bytes[5] = 0x01;
    bytes[8] = 13;
    bytes[10] = FRAME_FD | FRAME_BRS | FRAME_ESI;
    bytes[12 + 31] = 0x99;
    bytes[76] = 0x01;
    const auto hf = ASSERT_NIL_P(decode_frame(bytes, true));
    EXPECT_TRUE(hf.frame.fd);
    EXPECT_TRUE(hf.frame.bitrate_switched);
    EXPECT_TRUE(hf.frame.error_passive);
    EXPECT_EQ(hf.frame.id, 0x123);
    EXPECT_EQ(hf.frame.length, 32);
    EXPECT_EQ(hf.frame.data[31], 0x99);
    EXPECT_EQ(*hf.timestamp_us, 1);
}

TEST(GsUsbDecode, DecodesAFrameWithoutATimestamp) {
    const auto bytes = encode(Frame{.id = 0x10, .length = 1});
    const auto hf = ASSERT_NIL_P(decode_frame(bytes, false));
    EXPECT_EQ(hf.frame.id, 0x10);
    EXPECT_FALSE(hf.timestamp_us.has_value());
}

TEST(GsUsbDecode, DecodesABusErrorFrame) {
    std::vector<std::uint8_t> bytes(20, 0);
    bytes[4] = 0x04;
    bytes[7] = 0x20;
    bytes[8] = 8;
    const auto hf = ASSERT_NIL_P(decode_frame(bytes, false));
    EXPECT_EQ(hf.frame.type, Type::BUS_ERROR);
    EXPECT_EQ(hf.frame.id, 0x04);
}

TEST(GsUsbDecode, RejectsAFrameShorterThanItsFlagsDescribe) {
    std::vector<std::uint8_t> bytes(24, 0);
    bytes[10] = FRAME_FD;
    auto [hf, err] = decode_frame(bytes, false);
    ASSERT_MATCHES(err, CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "gs_usb adapter sent a 24-byte frame");
}

TEST(GsUsbControl, DecodesTheDeviceConfig) {
    const std::vector<std::uint8_t> bytes = {0, 0, 0, 1, 2, 0, 0, 0, 3, 0, 0, 0};
    const auto config = ASSERT_NIL_P(decode_device_config(bytes));
    EXPECT_EQ(config.channels, 2);
    EXPECT_EQ(config.software_version, 2);
    EXPECT_EQ(config.hardware_version, 3);
}

TEST(GsUsbControl, DecodesTheBitTimingConstants) {
    std::vector<std::uint8_t> bytes;
    for (const std::uint32_t v: {0x111u, 48000000u, 1u, 16u, 1u, 8u, 4u, 1u, 1024u, 1u})
        for (int i = 0; i < 4; i++)
            bytes.push_back(static_cast<std::uint8_t>(v >> (8 * i)));
    const auto c = ASSERT_NIL_P(decode_bt_const(bytes, false));
    EXPECT_EQ(c.features, 0x111);
    EXPECT_EQ(c.clock_hz, 48000000);
    EXPECT_EQ(c.nominal.tseg1_max, 16);
    EXPECT_EQ(c.nominal.tseg2_max, 8);
    EXPECT_EQ(c.nominal.sjw_max, 4);
    EXPECT_EQ(c.nominal.brp_max, 1024);
    auto [ext, err] = decode_bt_const(bytes, true);
    ASSERT_MATCHES(err, CRITICAL_HARDWARE_ERROR);
}

TEST(GsUsbControl, SplitsTimeSegmentOneIntoPropagationAndPhase) {
    const auto body = encode_bittiming({.brp = 6, .tseg1 = 13, .tseg2 = 2, .sjw = 1});
    const std::array<std::uint8_t, 20> expected = {
        6, 0, 0, 0, 7, 0, 0, 0, 2, 0, 0, 0, 1, 0, 0, 0, 6, 0, 0, 0,
    };
    EXPECT_EQ(body, expected);
}

TEST(GsUsbControl, EncodesAMode) {
    const std::array<std::uint8_t, 8> expected = {1, 0, 0, 0, 0x11, 0x01, 0, 0};
    EXPECT_EQ(
        encode_mode(
            MODE_START,
            FEATURE_FD | FEATURE_HW_TIMESTAMP | FEATURE_LISTEN_ONLY
        ),
        expected
    );
}
}
