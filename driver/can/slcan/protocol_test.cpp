// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <optional>
#include <random>
#include <string>
#include <string_view>
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
    if (expected.type == Type::DATA) {
        EXPECT_TRUE(std::ranges::equal(actual.payload(), expected.payload()));
    }
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

TEST(SlcanIdentify, IdentifiesFirmwareFromItsVersion) {
    EXPECT_EQ(identify("V1013"), Firmware::LAWICEL);
    EXPECT_EQ(identify("+Board: MultiBoard\tMCU: STM32G431"), Firmware::CANABLE_2_5);
    EXPECT_EQ(
        identify("b1ff43c5 https://github.com/normaldotcom/canable2-fw"),
        Firmware::CANABLE
    );
    EXPECT_EQ(identify(""), Firmware::CANABLE);
}

TEST(SlcanIdentify, KnowsWhichFirmwareAnswersCommands) {
    EXPECT_TRUE(answers(Firmware::LAWICEL));
    EXPECT_FALSE(answers(Firmware::CANABLE));
    EXPECT_TRUE(answers(Firmware::CANABLE_2_5));
}

TEST(SlcanBitrate, MapsEachFirmwaresBitratesToCommands) {
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(Firmware::LAWICEL, 10000)), "S0\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(Firmware::LAWICEL, 500000)), "S6\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(Firmware::LAWICEL, 800000)), "S7\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(Firmware::LAWICEL, 1000000)), "S8\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(Firmware::CANABLE, 750000)), "S7\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(Firmware::CANABLE, 83333)), "S9\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(Firmware::CANABLE_2_5, 800000)), "S7\r");
    EXPECT_EQ(ASSERT_NIL_P(bitrate_command(Firmware::CANABLE_2_5, 83333)), "S9\r");
    EXPECT_EQ(ASSERT_NIL_P(data_bitrate_command(2000000)), "Y2\r");
    EXPECT_EQ(ASSERT_NIL_P(data_bitrate_command(5000000)), "Y5\r");
}

TEST(SlcanBitrate, RejectsUnsupportedBitrates) {
    const auto [_, err] = bitrate_command(Firmware::LAWICEL, 83333);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "LAWICEL slcan firmware cannot run a classic CAN bus at 83333 bit/s. Supported "
        "bitrates: 10000, 20000, 50000, 100000, 125000, 250000, 500000, 800000, "
        "1000000"
    );
    ASSERT_OCCURRED_AS_P(bitrate_command(Firmware::CANABLE, 800000), CONFIG_ERROR);
    ASSERT_OCCURRED_AS_P(bitrate_command(Firmware::CANABLE_2_5, 750000), CONFIG_ERROR);
    const auto [data_command, data_err] = data_bitrate_command(4000000);
    ASSERT_MATCHES(data_err, CONFIG_ERROR);
    EXPECT_EQ(
        data_err.data,
        "slcan cannot run a CAN FD data phase at 4000000 bit/s. Supported bitrates: "
        "2000000, 5000000"
    );
}

TEST(SlcanOpen, OpensTheBusInNormalModeOnEveryFirmware) {
    for (const auto f: {Firmware::LAWICEL, Firmware::CANABLE, Firmware::CANABLE_2_5})
        EXPECT_EQ(
            ASSERT_NIL_P(open_commands(f, false)),
            std::vector<std::string>({"O\r"})
        );
}

TEST(SlcanOpen, OpensTheBusListenOnlyTheWayEachFirmwareDoes) {
    EXPECT_EQ(
        ASSERT_NIL_P(open_commands(Firmware::LAWICEL, true)),
        std::vector<std::string>({"L\r"})
    );
    EXPECT_EQ(
        ASSERT_NIL_P(open_commands(Firmware::CANABLE_2_5, true)),
        std::vector<std::string>({"M1\r", "O\r"})
    );
    const auto [_, err] = open_commands(Firmware::CANABLE, true);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "CANable firmware cannot open the bus listen only. Flash the CANable 2.5 or "
        "candleLight firmware to listen only"
    );
}

TEST(SlcanDecoder, SplitsTheStreamIntoMessages) {
    Decoder decoder;
    std::vector<Message> messages;
    for (const char c: std::string("\r\az\rV1013\rt123111\rt12\rT0000000A0\r"))
        if (auto msg = decoder.push(c)) messages.push_back(std::move(*msg));
    ASSERT_EQ(messages.size(), 7);
    EXPECT_EQ(messages[0].reply, Reply::ACCEPTED);
    EXPECT_EQ(messages[1].reply, Reply::REJECTED);
    EXPECT_EQ(messages[2].reply, Reply::OTHER);
    EXPECT_EQ(messages[2].text, "z");
    EXPECT_EQ(messages[3].reply, Reply::OTHER);
    EXPECT_EQ(messages[3].text, "V1013");
    EXPECT_EQ(messages[4].reply, Reply::FRAME);
    expect_same(messages[4].frame, frame(0x123, {0x11}));
    EXPECT_EQ(messages[5].reply, Reply::MALFORMED);
    EXPECT_EQ(messages[5].text, "t12");
    EXPECT_EQ(messages[6].reply, Reply::FRAME);
    expect_same(messages[6].frame, frame(0xA, {}, true));
}

TEST(SlcanDecoder, RecoversFromAnOverlongLine) {
    Decoder decoder;
    std::optional<Message> msg;
    for (const char c: std::string(MAX_LINE_LENGTH + 10, '0') + "\r")
        msg = decoder.push(c);
    ASSERT_TRUE(msg.has_value());
    EXPECT_EQ(msg.value().reply, Reply::MALFORMED);
    EXPECT_EQ(msg.value().text, std::string(MAX_LINE_LENGTH, '0'));
    for (const char c: std::string("t123111\r"))
        msg = decoder.push(c);
    ASSERT_TRUE(msg.has_value());
    EXPECT_EQ(msg.value().reply, Reply::FRAME);
    expect_same(msg.value().frame, frame(0x123, {0x11}));
}

TEST(SlcanDecoder, DropsAPartialLineOnBell) {
    Decoder decoder;
    for (const char c: std::string("t12"))
        EXPECT_FALSE(decoder.push(c).has_value());
    EXPECT_EQ(decoder.push('\a').value().reply, Reply::REJECTED);
    EXPECT_EQ(decoder.push('\r').value().reply, Reply::ACCEPTED);
}

namespace {
/// @brief the number of random inputs each property draws.
constexpr std::size_t DRAWS = 20000;
constexpr std::array<std::uint8_t, 7> FD_LENGTHS = {12, 16, 20, 24, 32, 48, 64};

/// @returns a random frame that encode accepts.
Frame random_frame(std::mt19937 &rng) {
    const auto pick = [&rng](const std::uint32_t max) {
        return std::uniform_int_distribution<std::uint32_t>(0, max)(rng);
    };
    Frame f;
    f.extended = pick(1) == 1;
    f.id = pick(f.extended ? MAX_EXTENDED_ID : MAX_STANDARD_ID);
    f.fd = pick(1) == 1;
    if (f.fd) {
        f.bitrate_switched = pick(1) == 1;
        f.length = pick(1) == 1 ? static_cast<std::uint8_t>(pick(MAX_CLASSIC_LENGTH))
                                : FD_LENGTHS[pick(FD_LENGTHS.size() - 1)];
    } else {
        if (pick(3) == 0) f.type = Type::REMOTE;
        f.length = static_cast<std::uint8_t>(pick(MAX_CLASSIC_LENGTH));
    }
    if (f.type == Type::DATA)
        for (std::size_t i = 0; i < f.length; i++)
            f.data[i] = static_cast<std::uint8_t>(pick(0xFF));
    return f;
}

/// @returns a random line without a carriage return or a bell, mostly made of the
/// characters that frame lines use.
std::string random_line(std::mt19937 &rng) {
    static constexpr std::string_view ALPHABET = "tTrRdDbB0123456789ABCDEFabcdefzVxG ";
    const auto
        length = std::uniform_int_distribution<std::size_t>(0, MAX_LINE_LENGTH + 8)(
            rng
        );
    std::string line;
    for (std::size_t i = 0; i < length; i++) {
        if (std::uniform_int_distribution(0, 15)(rng) == 0) {
            char c = 0;
            do {
                c = static_cast<char>(std::uniform_int_distribution(0, 255)(rng));
            } while (c == '\r' || c == '\a');
            line += c;
            continue;
        }
        line += ALPHABET
            [std::uniform_int_distribution<std::size_t>(0, ALPHABET.size() - 1)(rng)];
    }
    return line;
}

bool well_formed(const Frame &f) {
    const auto max_id = f.extended ? MAX_EXTENDED_ID : MAX_STANDARD_ID;
    if (f.id > max_id) return false;
    if (f.bitrate_switched && !f.fd) return false;
    if (!f.fd) return f.length <= MAX_CLASSIC_LENGTH;
    return f.type == Type::DATA && length_to_dlc(f.length).has_value();
}
}

TEST(SlcanProperty, DecodeReturnsEveryEncodedFrame) {
    // A fixed start makes every run draw the same inputs.
    // NOLINTNEXTLINE(bugprone-random-generator-seed): a repeatable sequence.
    std::mt19937 rng(4971);
    for (std::size_t i = 0; i < DRAWS; i++) {
        SCOPED_TRACE("draw " + std::to_string(i));
        const auto f = random_frame(rng);
        const auto line = encode(f);
        ASSERT_TRUE(line.ends_with('\r'));
        const auto decoded = ASSERT_NIL_P(decode(line.substr(0, line.size() - 1)));
        expect_same(decoded, f);
    }
}

TEST(SlcanProperty, DecodeAcceptsOnlyWellFormedFrames) {
    // NOLINTNEXTLINE(bugprone-random-generator-seed): a repeatable sequence.
    std::mt19937 rng(4971);
    std::size_t accepted = 0;
    for (std::size_t i = 0; i < DRAWS; i++) {
        SCOPED_TRACE("draw " + std::to_string(i));
        // Half the lines are frames with one character changed, so many decode.
        std::string line = random_line(rng);
        if (i % 2 == 0) {
            line = encode(random_frame(rng));
            line.pop_back();
            const auto
                at = std::uniform_int_distribution<std::size_t>(0, line.size() - 1)(
                    rng
                );
            line[at] = random_line(rng).append("0")[0];
        }
        const auto [f, err] = decode(line);
        if (err) {
            ASSERT_MATCHES(err, FRAME_ERROR);
            continue;
        }
        accepted++;
        ASSERT_TRUE(well_formed(f)) << line;
        const auto again = encode(f);
        expect_same(ASSERT_NIL_P(decode(again.substr(0, again.size() - 1))), f);
    }
    EXPECT_GT(accepted, DRAWS / 10);
}

TEST(SlcanProperty, DecoderSplitsAStreamIntoItsLines) {
    // NOLINTNEXTLINE(bugprone-random-generator-seed): a repeatable sequence.
    std::mt19937 rng(4971);
    Decoder decoder;
    for (std::size_t i = 0; i < DRAWS; i++) {
        SCOPED_TRACE("draw " + std::to_string(i));
        const auto kind = std::uniform_int_distribution(0, 3)(rng);
        std::string line = kind == 0 ? encode(random_frame(rng)) : random_line(rng);
        if (kind != 0) line += kind == 1 ? '\a' : '\r';
        std::optional<Message> msg;
        for (std::size_t j = 0; j < line.size(); j++) {
            msg = decoder.push(line[j]);
            if (j + 1 < line.size()) ASSERT_FALSE(msg.has_value());
        }
        ASSERT_TRUE(msg.has_value());
        const auto text = line.substr(0, line.size() - 1);
        if (line.back() == '\a') {
            EXPECT_EQ(msg.value().reply, Reply::REJECTED);
        } else if (text.size() > MAX_LINE_LENGTH) {
            EXPECT_EQ(msg.value().reply, Reply::MALFORMED);
            EXPECT_EQ(msg.value().text, text.substr(0, MAX_LINE_LENGTH));
        } else if (text.empty()) {
            EXPECT_EQ(msg.value().reply, Reply::ACCEPTED);
        } else if (std::string_view("tTrRdDbB").contains(text[0])) {
            const auto [f, err] = decode(text);
            EXPECT_EQ(msg.value().reply, err ? Reply::MALFORMED : Reply::FRAME);
            if (!err) expect_same(msg.value().frame, f);
        } else {
            EXPECT_EQ(msg.value().reply, Reply::OTHER);
            EXPECT_EQ(msg.value().text, text);
        }
    }
}
}
