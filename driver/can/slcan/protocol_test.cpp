// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/can/slcan/protocol.h"

namespace driver::can::slcan {
namespace {
Frame frame(
    const std::uint32_t id,
    const std::vector<std::uint8_t> &data,
    const bool extended = false,
    const bool fd = false,
    const bool bitrate_switched = false
) {
    Frame f;
    f.id = id;
    f.extended = extended;
    f.fd = fd;
    f.bitrate_switched = bitrate_switched;
    f.length = static_cast<std::uint8_t>(data.size());
    std::ranges::copy(data, f.data.begin());
    return f;
}

Frame remote(const std::uint32_t id, const std::uint8_t length, const bool extended) {
    Frame f;
    f.id = id;
    f.extended = extended;
    f.type = Type::REMOTE;
    f.length = length;
    return f;
}

std::vector<std::uint8_t> counting(const std::size_t n) {
    std::vector<std::uint8_t> data(n);
    for (std::size_t i = 0; i < n; i++)
        data[i] = static_cast<std::uint8_t>(i);
    return data;
}

std::string counting_hex(const std::size_t n) {
    std::string out;
    for (std::size_t i = 0; i < n; i++)
        out += std::format("{:02X}", i);
    return out;
}

void expect_same(const Frame &actual, const Frame &expected) {
    EXPECT_EQ(actual.id, expected.id);
    EXPECT_EQ(actual.extended, expected.extended);
    EXPECT_EQ(actual.fd, expected.fd);
    EXPECT_EQ(actual.bitrate_switched, expected.bitrate_switched);
    EXPECT_EQ(actual.type, expected.type);
    EXPECT_EQ(actual.length, expected.length);
    if (expected.type == Type::DATA)
        EXPECT_TRUE(std::ranges::equal(actual.payload(), expected.payload()));
}

struct Vector {
    std::string name;
    Frame frame;
    std::string line;
};

std::vector<Vector> vectors() {
    return {
        {"standard", frame(0x123, {0x11, 0x22}), "t12321122\r"},
        {"empty extended", frame(0x1ABCDEF0, {}, true), "T1ABCDEF00\r"},
        {"eight bytes",
         frame(0x7FF, {1, 2, 3, 4, 5, 6, 7, 8}),
         "t7FF80102030405060708\r"},
        {"standard remote", remote(0x123, 4, false), "r1234\r"},
        {"extended remote", remote(0x1, 8, true), "R000000018\r"},
        {"FD standard",
         frame(0x10, counting(12), false, true),
         "d0109" + counting_hex(12) + "\r"},
        {"FD standard with bitrate switch",
         frame(0x10, counting(20), false, true, true),
         "b010B" + counting_hex(20) + "\r"},
        {"FD extended",
         frame(0x18DAF110, counting(64), true, true),
         "D18DAF110F" + counting_hex(64) + "\r"},
        {"FD extended with bitrate switch",
         frame(0x18DAF110, counting(48), true, true, true),
         "B18DAF110E" + counting_hex(48) + "\r"},
    };
}
}

TEST(SlcanEncode, WritesEveryFrameKind) {
    for (const auto &v: vectors())
        EXPECT_EQ(encode(v.frame), v.line) << v.name;
}

TEST(SlcanDecode, ReadsEveryFrameKind) {
    for (const auto &v: vectors()) {
        const auto line = std::string_view(v.line).substr(0, v.line.size() - 1);
        const auto decoded = ASSERT_NIL_P(decode(line));
        expect_same(decoded, v.frame);
    }
}

TEST(SlcanDecode, DropsATimestampAndAcceptsLowercaseHex) {
    const auto decoded = ASSERT_NIL_P(decode("t1ab2ff00EA60"));
    expect_same(decoded, frame(0x1AB, {0xFF, 0x00}));
}

TEST(SlcanDecode, RejectsMalformedLines) {
    const std::vector<std::pair<std::string, std::string>> cases = {
        {"x12300", "unknown frame type"},
        {"t12", "the line is too short"},
        {"tXYZ0", "the identifier is not hex"},
        {"t8000", "the identifier is out of range"},
        {"T200000000", "the identifier is out of range"},
        {"t1239", "the length code is invalid"},
        {"t1232AB", "the data is shorter than its length"},
        {"t1231ZZ", "the data is not hex"},
        {"t12300A", "unexpected characters after the data"},
        {"r1230AB", "unexpected characters after the data"},
    };
    for (const auto &[line, reason]: cases) {
        const auto [_, err] = decode(line);
        ASSERT_MATCHES(err, FRAME_ERROR);
        EXPECT_EQ(err.data, "malformed slcan frame '" + line + "': " + reason) << line;
    }
}

TEST(SlcanBitrate, MapsStandardBitratesToCommands) {
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(10000)), "S0\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(500000)), "S6\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(1000000)), "S8\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(83333)), "S9\r");
    EXPECT_EQ(ASSERT_NIL_P(data_bitrate_command(2000000)), "Y2\r");
    EXPECT_EQ(ASSERT_NIL_P(data_bitrate_command(5000000)), "Y5\r");
}

TEST(SlcanBitrate, RejectsUnsupportedBitrates) {
    const auto [_, err] = bitrate_command(333);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "slcan has no bitrate of 333 bit/s. Supported: 10000, 20000, 50000, 100000, "
        "125000, 250000, 500000, 750000, 1000000, 83333"
    );
    const auto [__, data_err] = data_bitrate_command(4000000);
    ASSERT_MATCHES(data_err, CONFIG_ERROR);
    EXPECT_EQ(
        data_err.data,
        "slcan has no data bitrate of 4000000 bit/s. Supported: 2000000, 5000000"
    );
}

TEST(SlcanOpen, UsesListenOnlyMode) {
    EXPECT_EQ(open_command(false), "O\r");
    EXPECT_EQ(open_command(true), "L\r");
}

TEST(SlcanDecoder, SplitsTheStreamIntoReplies) {
    Decoder decoder;
    std::vector<Reply> replies;
    std::vector<Frame> frames;
    for (const char c: std::string("\r\az\rV1013\rt123111\rt12\rT0000000A0\r")) {
        Frame f;
        const auto reply = decoder.push(c, f);
        if (!reply.has_value()) continue;
        replies.push_back(*reply);
        if (*reply == Reply::FRAME) frames.push_back(f);
    }
    EXPECT_EQ(
        replies,
        std::vector<Reply>({
            Reply::ACCEPTED,
            Reply::REJECTED,
            Reply::OTHER,
            Reply::OTHER,
            Reply::FRAME,
            Reply::MALFORMED,
            Reply::FRAME,
        })
    );
    ASSERT_EQ(frames.size(), 2);
    expect_same(frames[0], frame(0x123, {0x11}));
    expect_same(frames[1], frame(0xA, {}, true));
}

TEST(SlcanDecoder, RecoversFromAnOverlongLine) {
    Decoder decoder;
    Frame f;
    std::optional<Reply> reply;
    for (const char c: std::string(MAX_LINE_LENGTH + 10, '0') + "\r")
        reply = decoder.push(c, f);
    EXPECT_EQ(reply, Reply::MALFORMED);
    for (const char c: std::string("t123111\r"))
        reply = decoder.push(c, f);
    EXPECT_EQ(reply, Reply::FRAME);
    expect_same(f, frame(0x123, {0x11}));
}

TEST(SlcanDecoder, DropsAPartialLineOnBell) {
    Decoder decoder;
    Frame f;
    for (const char c: std::string("t12"))
        EXPECT_FALSE(decoder.push(c, f).has_value());
    EXPECT_EQ(decoder.push('\a', f), Reply::REJECTED);
    EXPECT_EQ(decoder.push('\r', f), Reply::ACCEPTED);
}
}
