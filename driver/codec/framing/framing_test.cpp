// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <random>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/bus/types.gen.h"
#include "x/cpp/test/test.h"

#include "driver/codec/framing/checksum.h"
#include "driver/codec/framing/framing.h"

namespace driver::codec::framing {
namespace {
namespace bus = synnax::bus;
using Bytes = std::vector<std::uint8_t>;

Bytes bytes(const std::string &s) {
    return {s.begin(), s.end()};
}

/// @brief Collector records the frames a framer emits.
struct Collector {
    std::vector<Bytes> frames;

    [[nodiscard]] Handler handler() {
        return [this](const std::span<const std::uint8_t> f) {
            this->frames.emplace_back(f.begin(), f.end());
        };
    }
};

std::unique_ptr<Framer> open(const bus::Framing &framing) {
    return ASSERT_NIL_P(create(framing));
}

/// @brief writes data in chunks of random sizes and returns the frames.
std::vector<Bytes> feed(Framer &framer, const Bytes &data, std::mt19937_64 &rng) {
    Collector c;
    std::size_t pos = 0;
    while (pos < data.size()) {
        const auto n = std::min<std::size_t>(rng() % 17 + 1, data.size() - pos);
        EXPECT_FALSE(framer.write(std::span(data).subspan(pos, n), c.handler()));
        pos += n;
    }
    return c.frames;
}

/// @brief writes data one byte at a time and returns the frames.
std::vector<Bytes> trickle(Framer &framer, const Bytes &data) {
    Collector c;
    for (const auto b: data) {
        const std::uint8_t one[] = {b};
        EXPECT_FALSE(framer.write(one, c.handler()));
    }
    return c.frames;
}

/// @brief encodes each frame and feeds the stream back in random chunks.
void expect_round_trip(Framer &framer, const std::vector<Bytes> &frames, int seed) {
    std::mt19937_64 rng(seed);
    Bytes wire;
    for (const auto &f: frames)
        ASSERT_NIL(framer.encode(f, wire));
    EXPECT_EQ(feed(framer, wire, rng), frames);
    EXPECT_EQ(framer.errors(), 0);
}

Bytes random_payload(std::mt19937_64 &rng, const std::size_t max) {
    Bytes p(rng() % max + 1);
    for (auto &b: p)
        b = static_cast<std::uint8_t>(rng() % 4 == 0 ? 0 : rng());
    return p;
}

Bytes cobs(const Bytes &payload) {
    Bytes out = {0};
    std::size_t code_at = 0;
    std::uint8_t code = 1;
    for (const auto b: payload) {
        if (b != 0) {
            out.push_back(b);
            code++;
        }
        if (b == 0 || code == 0xFF) {
            out[code_at] = code;
            code_at = out.size();
            out.push_back(0);
            code = 1;
        }
    }
    out[code_at] = code;
    out.push_back(0);
    return out;
}

Bytes slip(const Bytes &payload) {
    Bytes out = {0xC0};
    for (const auto b: payload) {
        if (b == 0xC0) {
            out.push_back(0xDB);
            out.push_back(0xDC);
        } else if (b == 0xDB) {
            out.push_back(0xDB);
            out.push_back(0xDD);
        } else
            out.push_back(b);
    }
    out.push_back(0xC0);
    return out;
}

void append_uint(
    Bytes &out,
    const std::uint64_t v,
    const std::size_t n,
    const bool big
) {
    for (std::size_t i = 0; i < n; i++) {
        const auto shift = big ? (n - 1 - i) * 8 : i * 8;
        out.push_back(static_cast<std::uint8_t>(v >> shift));
    }
}

/// @brief SYNC_MODBUS frames are AA 55, a one-byte body length, the body, and a
/// little-endian CRC-16/MODBUS over everything before it.
const bus::SyncFraming SYNC_MODBUS{
    .sync = "aa55",
    .length_offset = 2,
    .length_size = 1,
    .length_adjustment = 2,
    .checksum = bus::CHECKSUM_CRC_16_MODBUS,
};

/// @brief MODBUS_STYLE frames are an address byte, a big-endian two-byte body length,
/// the body, and a little-endian CRC-16/MODBUS as in Modbus RTU.
const bus::SyncFraming MODBUS_STYLE{
    .sync = "01",
    .length_offset = 1,
    .length_size = 2,
    .byte_order = synnax::library::BYTE_ORDER_BIG_ENDIAN,
    .length_adjustment = 2,
    .checksum = bus::CHECKSUM_CRC_16_MODBUS,
};

/// @returns the wire bytes of a SYNC_MODBUS frame, built by hand.
Bytes sync_wire(const Bytes &body) {
    Bytes f = {0xAA, 0x55, static_cast<std::uint8_t>(body.size())};
    f.insert(f.end(), body.begin(), body.end());
    append_uint(f, checksum(Checksum::CRC16_MODBUS, f), 2, false);
    return f;
}

/// @returns the frame SYNC_MODBUS decodes from sync_wire(body).
Bytes sync_frame(const Bytes &body) {
    auto w = sync_wire(body);
    w.resize(w.size() - 2);
    return w;
}
}

TEST(Delimiter, SplitsFramesDeliveredOneByteAtATime) {
    const auto f = open(bus::DelimiterFraming{.delimiter = "\r\n"});
    EXPECT_EQ(
        trickle(*f, bytes("a\r\nbb\r\n")),
        (std::vector{bytes("a"), bytes("bb")})
    );
}

TEST(Delimiter, EmitsEveryFrameInAMergedChunk) {
    const auto f = open(bus::DelimiterFraming{});
    Collector c;
    ASSERT_NIL(f->write(bytes("1,2\n3,4\n5"), c.handler()));
    EXPECT_EQ(c.frames, (std::vector{bytes("1,2"), bytes("3,4")}));
    ASSERT_NIL(f->write(bytes("\n"), c.handler()));
    EXPECT_EQ(c.frames.back(), bytes("5"));
}

TEST(Delimiter, SkipsEmptyFrames) {
    const auto f = open(bus::DelimiterFraming{});
    Collector c;
    ASSERT_NIL(f->write(bytes("\n\nx\n"), c.handler()));
    EXPECT_EQ(c.frames, std::vector<Bytes>{bytes("x")});
}

TEST(Delimiter, DropsAnOversizedFrameAndRecovers) {
    const auto f = open(bus::DelimiterFraming{.delimiter = "\r\n"});
    auto stream = Bytes(MAX_SIZE + 10, 'x');
    const auto tail = bytes("\r\nok\r\n");
    stream.insert(stream.end(), tail.begin(), tail.end());
    Collector c;
    ASSERT_OCCURRED_AS(f->write(stream, c.handler()), OVERFLOW_ERROR);
    EXPECT_EQ(c.frames, std::vector<Bytes>{bytes("ok")});
    EXPECT_EQ(f->errors(), 1);
}

TEST(Delimiter, RecoversFromAnOverflowSplitAcrossChunks) {
    const auto f = open(bus::DelimiterFraming{.delimiter = "\r\n"});
    Collector c;
    ASSERT_NIL(f->write(Bytes(MAX_SIZE - 1, 'x'), c.handler()));
    auto over = Bytes(10, 'x');
    over.push_back('\r');
    ASSERT_OCCURRED_AS(f->write(over, c.handler()), OVERFLOW_ERROR);
    ASSERT_NIL(f->write(bytes("\nok\r\n"), c.handler()));
    EXPECT_EQ(c.frames, std::vector<Bytes>{bytes("ok")});
}

TEST(Delimiter, DropsThePartialFrameOnReset) {
    const auto f = open(bus::DelimiterFraming{});
    Collector c;
    ASSERT_NIL(f->write(bytes("stale"), c.handler()));
    f->reset();
    ASSERT_NIL(f->write(bytes("fresh\n"), c.handler()));
    EXPECT_EQ(c.frames, std::vector<Bytes>{bytes("fresh")});
}

TEST(Delimiter, EncodeAppendsTheDelimiter) {
    const auto f = open(bus::DelimiterFraming{.delimiter = "\r\n"});
    Bytes out = bytes(">");
    ASSERT_NIL(f->encode(bytes("MEAS?"), out));
    EXPECT_EQ(out, bytes(">MEAS?\r\n"));
}

TEST(Delimiter, EncodeRejectsAFrameContainingTheDelimiter) {
    const auto f = open(bus::DelimiterFraming{});
    Bytes out;
    ASSERT_OCCURRED_AS(f->encode(bytes("a\nb"), out), ENCODE_ERROR);
    EXPECT_TRUE(out.empty());
}

TEST(Delimiter, EncodeRejectsATailThatFormsAnEarlierDelimiter) {
    const auto f = open(bus::DelimiterFraming{.delimiter = "aa"});
    Bytes out;
    ASSERT_OCCURRED_AS(f->encode(bytes("xa"), out), ENCODE_ERROR);
}

TEST(Delimiter, EncodeRejectsAnEmptyFrame) {
    const auto f = open(bus::DelimiterFraming{});
    Bytes out;
    ASSERT_OCCURRED_AS(f->encode(Bytes{}, out), ENCODE_ERROR);
}

TEST(Delimiter, RoundTripsRandomFramesAcrossRandomChunks) {
    std::mt19937_64 rng(1);
    std::vector<Bytes> frames;
    for (int i = 0; i < 500; i++)
        frames.push_back(
            bytes(std::to_string(rng()) + "," + std::to_string(rng() % 100))
        );
    const auto f = open(bus::DelimiterFraming{.delimiter = "\r\n"});
    expect_round_trip(*f, frames, 1);
}

TEST(Fixed, SplitsAndMergesFrames) {
    const auto f = open(bus::FixedFraming{.length = 3});
    Collector c;
    ASSERT_NIL(f->write(bytes("ab"), c.handler()));
    ASSERT_NIL(f->write(bytes("cdef"), c.handler()));
    ASSERT_NIL(f->write(bytes("g"), c.handler()));
    ASSERT_NIL(f->write(bytes("hijkl"), c.handler()));
    EXPECT_EQ(
        c.frames,
        (std::vector{bytes("abc"), bytes("def"), bytes("ghi"), bytes("jkl")})
    );
}

TEST(Fixed, EncodeRejectsTheWrongSize) {
    const auto f = open(bus::FixedFraming{.length = 3});
    Bytes out;
    ASSERT_OCCURRED_AS(f->encode(bytes("ab"), out), ENCODE_ERROR);
    ASSERT_OCCURRED_AS(f->encode(bytes("abcd"), out), ENCODE_ERROR);
}

TEST(Fixed, RoundTripsRandomFramesAcrossRandomChunks) {
    std::mt19937_64 rng(2);
    std::vector<Bytes> frames;
    for (int i = 0; i < 200; i++) {
        Bytes f(8);
        for (auto &b: f)
            b = static_cast<std::uint8_t>(rng());
        frames.push_back(std::move(f));
    }
    const auto f = open(bus::FixedFraming{.length = 8});
    expect_round_trip(*f, frames, 2);
}

TEST(Sync, DecodesHandBuiltFramesWithoutTheirChecksum) {
    const auto f = open(SYNC_MODBUS);
    auto stream = sync_wire({1, 2, 3});
    const auto second = sync_wire({0xAA, 0x55});
    stream.insert(stream.end(), second.begin(), second.end());
    EXPECT_EQ(
        trickle(*f, stream),
        (std::vector{sync_frame({1, 2, 3}), sync_frame({0xAA, 0x55})})
    );
    EXPECT_EQ(f->errors(), 0);
}

TEST(Sync, EncodeWritesTheSyncLengthAndChecksum) {
    const auto f = open(SYNC_MODBUS);
    Bytes out;
    ASSERT_NIL(f->encode(Bytes{0, 0, 0, 7, 8, 9}, out));
    EXPECT_EQ(out, sync_wire({7, 8, 9}));
}

TEST(Sync, RoundTripsRandomFramesAcrossRandomChunks) {
    std::mt19937_64 rng(3);
    std::vector<Bytes> frames;
    for (int i = 0; i < 500; i++)
        frames.push_back(sync_frame(random_payload(rng, 40)));
    const auto f = open(SYNC_MODBUS);
    expect_round_trip(*f, frames, 3);
}

TEST(Sync, DropsGarbageBeforeASync) {
    const auto f = open(SYNC_MODBUS);
    auto stream = bytes("garbage");
    const auto wire = sync_wire({1, 2, 3});
    stream.insert(stream.end(), wire.begin(), wire.end());
    EXPECT_EQ(trickle(*f, stream), std::vector<Bytes>{sync_frame({1, 2, 3})});
    EXPECT_GE(f->errors(), 1);
}

TEST(Sync, ResynchronizesAfterABadChecksum) {
    const auto f = open(SYNC_MODBUS);
    auto stream = sync_wire({1, 2, 3});
    stream[4] ^= 0xFF;
    const auto good = sync_wire({4, 5});
    stream.insert(stream.end(), good.begin(), good.end());
    Collector c;
    ASSERT_NIL(f->write(stream, c.handler()));
    EXPECT_EQ(c.frames, std::vector<Bytes>{sync_frame({4, 5})});
    EXPECT_GE(f->errors(), 1);
}

TEST(Sync, ResynchronizesAfterALengthBeyondTheMaximum) {
    const auto f = open(
        bus::SyncFraming{
            .sync = "7E",
            .length_offset = 1,
            .length_size = 2,
            .byte_order = synnax::library::BYTE_ORDER_BIG_ENDIAN,
        }
    );
    Bytes stream = {0x7E, 0xFF, 0xFF, 0x01, 0x7E, 0x00, 0x01, 0x42};
    Collector c;
    ASSERT_NIL(f->write(stream, c.handler()));
    EXPECT_EQ(c.frames, (std::vector<Bytes>{{0x7E, 0x00, 0x01, 0x42}}));
    EXPECT_GE(f->errors(), 1);
}

TEST(Sync, FindsASyncSplitAcrossChunks) {
    const auto f = open(SYNC_MODBUS);
    const auto wire = sync_wire({7, 7});
    Collector c;
    ASSERT_NIL(f->write(std::span(wire).first(1), c.handler()));
    ASSERT_NIL(f->write(std::span(wire).subspan(1), c.handler()));
    EXPECT_EQ(c.frames, std::vector<Bytes>{sync_frame({7, 7})});
    EXPECT_EQ(f->errors(), 0);
}

TEST(Sync, RoundTripsABigEndianCRC32Frame) {
    const auto f = open(
        bus::SyncFraming{
            .sync = "7e",
            .length_offset = 1,
            .length_size = 2,
            .byte_order = synnax::library::BYTE_ORDER_BIG_ENDIAN,
            .length_adjustment = 4,
            .checksum = bus::CHECKSUM_CRC_32,
            .checksum_byte_order = synnax::library::BYTE_ORDER_BIG_ENDIAN,
        }
    );
    const Bytes frame = {0x7E, 0x00, 0x03, 0x7E, 0x00, 0x42};
    Bytes wire;
    ASSERT_NIL(f->encode(Bytes{0, 0, 0, 0x7E, 0x00, 0x42}, wire));
    Bytes expected = frame;
    append_uint(expected, checksum(Checksum::CRC32, frame), 4, true);
    EXPECT_EQ(wire, expected);
    EXPECT_EQ(trickle(*f, wire), std::vector<Bytes>{frame});
    wire.back() ^= 1;
    EXPECT_TRUE(trickle(*f, wire).empty());
}

TEST(Sync, RoundTripsAModbusStyleFrameWithMixedByteOrders) {
    const auto f = open(MODBUS_STYLE);
    const Bytes frame = {0x01, 0x00, 0x03, 0x03, 0x02, 0x00};
    const auto crc = checksum(Checksum::CRC16_MODBUS, frame);
    Bytes expected = frame;
    expected.push_back(static_cast<std::uint8_t>(crc));
    expected.push_back(static_cast<std::uint8_t>(crc >> 8));
    Bytes wire;
    ASSERT_NIL(f->encode(Bytes{0, 0, 0, 0x03, 0x02, 0x00}, wire));
    EXPECT_EQ(wire, expected);
    EXPECT_EQ(trickle(*f, wire), std::vector<Bytes>{frame});
    std::swap(wire[wire.size() - 1], wire[wire.size() - 2]);
    EXPECT_TRUE(trickle(*f, wire).empty());
    std::mt19937_64 rng(8);
    std::vector<Bytes> frames;
    for (int i = 0; i < 200; i++) {
        auto body = random_payload(rng, 300);
        body.insert(
            body.begin(),
            {0x01,
             static_cast<std::uint8_t>(body.size() >> 8),
             static_cast<std::uint8_t>(body.size())}
        );
        frames.push_back(std::move(body));
    }
    const auto g = open(MODBUS_STYLE);
    expect_round_trip(*g, frames, 8);
}

TEST(Sync, RoundTripsALengthThatCountsTheWholeFrame) {
    const auto f = open(
        bus::SyncFraming{
            .sync = "02",
            .length_offset = 1,
            .length_size = 1,
            .length_adjustment = -2,
            .checksum = bus::CHECKSUM_CRC_16_CCITT,
        }
    );
    Bytes wire;
    ASSERT_NIL(f->encode(Bytes{0, 0, 0x31, 0x32}, wire));
    EXPECT_EQ(wire[1], 6);
    std::mt19937_64 rng(4);
    std::vector<Bytes> frames;
    for (int i = 0; i < 200; i++) {
        auto frame = random_payload(rng, 30);
        frame.insert(
            frame.begin(),
            {0x02, static_cast<std::uint8_t>(frame.size() + 4)}
        );
        frames.push_back(std::move(frame));
    }
    expect_round_trip(*f, frames, 4);
}

TEST(Sync, SplitsALengthPrefixedStreamWithNoSync) {
    const auto f = open(
        bus::SyncFraming{
            .length_size = 2,
            .byte_order = synnax::library::BYTE_ORDER_BIG_ENDIAN,
        }
    );
    const Bytes stream = {0x00, 0x02, 0xAA, 0xBB, 0x00, 0x01, 0xCC};
    EXPECT_EQ(
        trickle(*f, stream),
        (std::vector<Bytes>{{0x00, 0x02, 0xAA, 0xBB}, {0x00, 0x01, 0xCC}})
    );
}

TEST(Sync, EncodeRejectsAFrameShorterThanItsHeader) {
    const auto f = open(SYNC_MODBUS);
    Bytes out;
    ASSERT_OCCURRED_AS(f->encode(Bytes{0xAA, 0x55}, out), ENCODE_ERROR);
}

TEST(Sync, EncodeRejectsALengthTheFieldCannotHold) {
    const auto f = open(SYNC_MODBUS);
    Bytes out;
    ASSERT_OCCURRED_AS(f->encode(Bytes(300, 1), out), ENCODE_ERROR);
}

TEST(COBS, EncodesLikeAReferenceEncoder) {
    std::mt19937_64 rng(5);
    const auto f = open(bus::CobsFraming{});
    for (int i = 0; i < 100; i++) {
        auto payload = random_payload(rng, i % 10 == 0 ? 600 : 40);
        if (i % 7 == 0) std::fill(payload.begin(), payload.end(), 0x11);
        Bytes out;
        ASSERT_NIL(f->encode(payload, out));
        ASSERT_EQ(out, cobs(payload));
    }
}

TEST(COBS, RoundTripsRandomFramesAcrossRandomChunks) {
    std::mt19937_64 rng(6);
    std::vector<Bytes> frames;
    for (int i = 0; i < 300; i++) {
        auto payload = random_payload(rng, i % 10 == 0 ? 600 : 40);
        if (i % 7 == 0) std::fill(payload.begin(), payload.end(), 0x11);
        frames.push_back(std::move(payload));
    }
    const auto f = open(bus::CobsFraming{});
    expect_round_trip(*f, frames, 6);
}

TEST(COBS, DropsAFrameThatEndsInsideABlock) {
    const auto f = open(bus::CobsFraming{});
    Bytes stream = {0x05, 0x11, 0x22, 0x00};
    const auto good = cobs({1, 0, 2});
    stream.insert(stream.end(), good.begin(), good.end());
    Collector c;
    ASSERT_NIL(f->write(stream, c.handler()));
    EXPECT_EQ(c.frames, (std::vector<Bytes>{{1, 0, 2}}));
    EXPECT_EQ(f->errors(), 1);
}

TEST(COBS, DropsAnOversizedFrameAndRecovers) {
    const auto f = open(bus::CobsFraming{});
    auto stream = cobs(Bytes(MAX_SIZE + 1, 7));
    const auto good = cobs({7});
    stream.insert(stream.end(), good.begin(), good.end());
    Collector c;
    ASSERT_OCCURRED_AS(f->write(stream, c.handler()), OVERFLOW_ERROR);
    EXPECT_EQ(c.frames, (std::vector<Bytes>{{7}}));
}

TEST(COBS, EncodeRejectsAFrameOverTheMaximum) {
    const auto f = open(bus::CobsFraming{});
    Bytes out;
    ASSERT_OCCURRED_AS(f->encode(Bytes(MAX_SIZE + 1, 7), out), ENCODE_ERROR);
}

TEST(SLIP, DecodesEscapedBytes) {
    const auto f = open(bus::SlipFraming{});
    const Bytes stream = {0xC0, 0x01, 0xDB, 0xDC, 0x02, 0xDB, 0xDD, 0xC0};
    EXPECT_EQ(trickle(*f, stream), (std::vector<Bytes>{{0x01, 0xC0, 0x02, 0xDB}}));
}

TEST(SLIP, EncodesLikeAReferenceEncoder) {
    const auto f = open(bus::SlipFraming{});
    const Bytes payload = {0x01, 0xC0, 0x02, 0xDB, 0xDC};
    Bytes out;
    ASSERT_NIL(f->encode(payload, out));
    EXPECT_EQ(out, slip(payload));
}

TEST(SLIP, RoundTripsRandomFramesAcrossRandomChunks) {
    std::mt19937_64 rng(7);
    std::vector<Bytes> frames;
    for (int i = 0; i < 300; i++) {
        auto payload = random_payload(rng, 40);
        if (i % 5 == 0) payload.insert(payload.begin(), {0xC0, 0xDB});
        frames.push_back(std::move(payload));
    }
    const auto f = open(bus::SlipFraming{});
    expect_round_trip(*f, frames, 7);
}

TEST(SLIP, DropsAFrameWithAnInvalidEscape) {
    const auto f = open(bus::SlipFraming{});
    const Bytes stream = {0x01, 0xDB, 0x05, 0x02, 0xC0, 0x03, 0xC0};
    Collector c;
    ASSERT_NIL(f->write(stream, c.handler()));
    EXPECT_EQ(c.frames, (std::vector<Bytes>{{0x03}}));
    EXPECT_EQ(f->errors(), 1);
}

TEST(SLIP, DropsAnOversizedFrameAndRecovers) {
    const auto f = open(bus::SlipFraming{});
    auto stream = slip(Bytes(MAX_SIZE + 1, 7));
    const auto good = slip({4});
    stream.insert(stream.end(), good.begin(), good.end());
    Collector c;
    ASSERT_OCCURRED_AS(f->write(stream, c.handler()), OVERFLOW_ERROR);
    EXPECT_EQ(c.frames, (std::vector<Bytes>{{4}}));
    EXPECT_EQ(f->errors(), 1);
}

TEST(Create, RejectsAnEmptyDelimiter) {
    ASSERT_OCCURRED_AS_P(create(bus::DelimiterFraming{.delimiter = ""}), CONFIG_ERROR);
}

TEST(Create, RejectsAZeroFixedLength) {
    ASSERT_OCCURRED_AS_P(create(bus::FixedFraming{.length = 0}), CONFIG_ERROR);
}

TEST(Create, RejectsALengthSizeOf3) {
    ASSERT_OCCURRED_AS_P(create(bus::SyncFraming{.length_size = 3}), CONFIG_ERROR);
}

TEST(Create, RejectsASyncThatIsNotHex) {
    ASSERT_OCCURRED_AS_P(
        create(bus::SyncFraming{.sync = "AZ", .length_offset = 1}),
        CONFIG_ERROR
    );
}

TEST(Create, RejectsASyncWithAnOddNumberOfDigits) {
    ASSERT_OCCURRED_AS_P(
        create(bus::SyncFraming{.sync = "AA5", .length_offset = 2}),
        CONFIG_ERROR
    );
}

TEST(Create, RejectsAnUnknownChecksum) {
    ASSERT_OCCURRED_AS_P(create(bus::SyncFraming{.checksum = "md5"}), CONFIG_ERROR);
}

TEST(Create, RejectsAnUnknownChecksumByteOrder) {
    ASSERT_OCCURRED_AS_P(
        create(bus::SyncFraming{.checksum_byte_order = "middle"}),
        CONFIG_ERROR
    );
}

TEST(Create, RejectsAnUnknownByteOrder) {
    ASSERT_OCCURRED_AS_P(
        create(bus::SyncFraming{.byte_order = "middle"}),
        CONFIG_ERROR
    );
}

TEST(Create, RejectsALengthFieldInsideTheSync) {
    ASSERT_OCCURRED_AS_P(
        create(bus::SyncFraming{.sync = "AA55", .length_offset = 1}),
        CONFIG_ERROR
    );
}
}
