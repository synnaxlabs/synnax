// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <cstddef>
#include <cstdint>
#include <memory>
#include <optional>
#include <random>
#include <ranges>
#include <span>
#include <string>
#include <string_view>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/bench/alloc.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/codec/framing/framing.h"

namespace driver::codec::framing {
namespace {
namespace icd = synnax::icd;
using x::testutil::bytes;

std::vector<std::uint8_t> vec(const std::string_view s) {
    return {s.begin(), s.end()};
}

std::unique_ptr<Framer> delimited_by(const std::string_view d) {
    return ASSERT_NIL_P(create(icd::DelimitedFraming{.delimiter = std::string(d)}));
}

std::unique_ptr<Framer> fixed(const std::uint32_t length) {
    return ASSERT_NIL_P(create(icd::FixedFraming{.length = length}));
}

std::unique_ptr<Framer> cobs() {
    return ASSERT_NIL_P(create(icd::CobsFraming{}));
}

std::unique_ptr<Framer> slip() {
    return ASSERT_NIL_P(create(icd::SlipFraming{}));
}

/// @brief writes chunk to f and returns the frames that it completes, failing on an
/// error.
std::vector<std::string> frames(Framer &f, const std::string_view chunk) {
    f.write(bytes(chunk));
    std::vector<std::string> out;
    while (true) {
        const auto frame = ASSERT_NIL_P(f.next());
        if (!frame.has_value()) return out;
        out.emplace_back(frame->begin(), frame->end());
    }
}

using Frames = std::vector<std::string>;

/// @brief asserts that f encodes frame with an error of text.
void assert_refuses(
    const Framer &f,
    const std::string_view frame,
    const std::string &text
) {
    std::vector<std::uint8_t> out = {7};
    const auto err = f.encode(bytes(frame), out);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, text);
    EXPECT_EQ(out, std::vector<std::uint8_t>{7});
}

/// @brief asserts that the frames f encodes from random payloads come back from a
/// framer of the same framing, whatever chunks the stream arrives in.
void assert_round_trips(
    Framer &f,
    const std::vector<std::vector<std::uint8_t>> &payloads
) {
    std::vector<std::uint8_t> stream;
    for (const auto &p: payloads)
        ASSERT_NIL(f.encode(p, stream));
    // NOLINTNEXTLINE(bugprone-random-generator-seed): a fixed seed repeats each run.
    std::mt19937 rng(6322);
    std::size_t at = 0;
    std::vector<std::vector<std::uint8_t>> got;
    while (at < stream.size()) {
        const auto n = std::min<std::size_t>(rng() % 17 + 1, stream.size() - at);
        f.write(std::span(stream).subspan(at, n));
        at += n;
        while (true) {
            const auto frame = ASSERT_NIL_P(f.next());
            if (!frame.has_value()) break;
            got.emplace_back(frame->begin(), frame->end());
        }
    }
    EXPECT_EQ(got, payloads);
}
}

/// @brief it should split a stream on its delimiter.
TEST(Delimited, Split) {
    const auto f = delimited_by("\n");
    EXPECT_EQ(frames(*f, "a\nbc\nd"), (Frames{"a", "bc"}));
    EXPECT_EQ(frames(*f, "e\n"), (Frames{"de"}));
}

/// @brief it should find a delimiter of several bytes that two chunks split.
TEST(Delimited, SplitDelimiterAcrossChunks) {
    const auto f = delimited_by("\r\n");
    EXPECT_EQ(frames(*f, "ab\r"), Frames{});
    EXPECT_EQ(frames(*f, "\ncd\r"), (Frames{"ab"}));
    EXPECT_EQ(frames(*f, "\r\n"), (Frames{"cd\r"}));
}

/// @brief it should drop empty frames with no error.
TEST(Delimited, DropEmptyFrames) {
    const auto f = delimited_by("\n");
    EXPECT_EQ(frames(*f, "\n\na\n\n"), (Frames{"a"}));
}

/// @brief it should keep a partial frame when the stream goes quiet.
TEST(Delimited, KeepPartialFrameWhenIdle) {
    const auto f = delimited_by("\n");
    EXPECT_EQ(frames(*f, "ab"), Frames{});
    ASSERT_NIL(f->idle());
    EXPECT_EQ(frames(*f, "\n"), (Frames{"ab"}));
}

/// @brief it should drop a partial frame on clear.
TEST(Delimited, Clear) {
    const auto f = delimited_by("\n");
    EXPECT_EQ(frames(*f, "ab"), Frames{});
    f->clear();
    EXPECT_EQ(frames(*f, "c\n"), (Frames{"c"}));
}

/// @brief it should accept a frame of MAX_SIZE bytes whose delimiter two chunks split.
TEST(Delimited, AcceptMaxSize) {
    const auto f = delimited_by("\r\n");
    const std::string big(MAX_SIZE, 'x');
    EXPECT_EQ(frames(*f, big + "\r"), Frames{});
    EXPECT_EQ(frames(*f, "\n"), (Frames{big}));
}

/// @brief it should drop a frame longer than MAX_SIZE once it has no room for a
/// delimiter, and read the frames after it.
TEST(Delimited, DropFrameThatGrowsPastMaxSize) {
    const auto f = delimited_by("\n");
    f->write(bytes(std::string(MAX_SIZE + 1, 'x')));
    const auto [frame, err] = f->next();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "dropped a frame longer than 65536 bytes");
    EXPECT_FALSE(frame.has_value());
    EXPECT_EQ(frames(*f, std::string(MAX_SIZE * 2, 'y')), Frames{});
    EXPECT_EQ(frames(*f, "tail\nok\n"), (Frames{"ok"}));
}

/// @brief it should drop a frame longer than MAX_SIZE that arrives whole.
TEST(Delimited, DropWholeFrameOverMaxSize) {
    const auto f = delimited_by("\n");
    f->write(bytes(std::string(MAX_SIZE + 1, 'x') + "\nok\n"));
    const auto [frame, err] = f->next();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "dropped a frame longer than 65536 bytes");
    EXPECT_EQ(frames(*f, ""), (Frames{"ok"}));
}

/// @brief it should encode frames that a framer of the same delimiter reads back.
TEST(Delimited, RoundTrip) {
    const auto f = delimited_by("\r\n");
    // NOLINTNEXTLINE(bugprone-random-generator-seed): a fixed seed repeats each run.
    std::mt19937 rng(6322);
    std::vector<std::vector<std::uint8_t>> payloads;
    while (payloads.size() < 500) {
        std::vector<std::uint8_t> p(rng() % 40 + 1);
        for (auto &b: p)
            b = static_cast<std::uint8_t>(rng() % 4 == 0 ? '\r' : rng());
        std::vector<std::uint8_t> out;
        if (!f->encode(p, out)) payloads.push_back(std::move(p));
    }
    assert_round_trips(*f, payloads);
}

/// @brief it should refuse to encode a frame that a read would not give back.
TEST(Delimited, RefuseEncode) {
    assert_refuses(*delimited_by("\n"), "", "the frame is empty");
    assert_refuses(*delimited_by("\n"), "a\nb", "the frame holds the delimiter");
    assert_refuses(*delimited_by("aa"), "xa", "the frame holds the delimiter");
    assert_refuses(
        *delimited_by("\n"),
        std::string(MAX_SIZE + 1, 'x'),
        "the frame has 65537 bytes, more than the maximum of 65536"
    );
}

/// @brief it should split a stream into frames of its length.
TEST(Fixed, Split) {
    const auto f = fixed(3);
    EXPECT_EQ(frames(*f, "abcde"), (Frames{"abc"}));
    EXPECT_EQ(frames(*f, "fghi"), (Frames{"def", "ghi"}));
}

/// @brief it should drop a partial frame when the stream goes quiet, so the next frame
/// starts in step.
TEST(Fixed, DropPartialFrameWhenIdle) {
    const auto f = fixed(3);
    EXPECT_EQ(frames(*f, "ab"), Frames{});
    const auto err = f->idle();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "dropped 2 bytes of a partial frame");
    ASSERT_NIL(f->idle());
    EXPECT_EQ(frames(*f, "xyz"), (Frames{"xyz"}));
}

/// @brief it should drop a partial frame on clear.
TEST(Fixed, Clear) {
    const auto f = fixed(2);
    EXPECT_EQ(frames(*f, "a"), Frames{});
    f->clear();
    EXPECT_EQ(frames(*f, "bc"), (Frames{"bc"}));
}

/// @brief it should encode frames that a framer of the same length reads back.
TEST(Fixed, RoundTrip) {
    const auto f = fixed(5);
    std::vector<std::vector<std::uint8_t>> payloads;
    payloads.reserve(100);
    for (std::uint8_t i = 0; i < 100; i++)
        payloads.push_back({i, 0, '\n', 255, i});
    assert_round_trips(*f, payloads);
}

/// @brief it should refuse to encode a frame of another length.
TEST(Fixed, RefuseEncode) {
    assert_refuses(
        *fixed(4),
        "abc",
        "the frame has 3 bytes, not the fixed length of 4"
    );
}

namespace {
using Bytes = std::vector<std::uint8_t>;

/// @brief returns the bytes from first to last, in order.
Bytes run(const std::uint8_t first, const std::uint8_t last) {
    Bytes out;
    for (unsigned b = first; b <= last; b++)
        out.push_back(static_cast<std::uint8_t>(b));
    return out;
}

Bytes join(const Bytes &a, const Bytes &b) {
    Bytes out = a;
    out.insert(out.end(), b.begin(), b.end());
    return out;
}

/// @brief asserts that f encodes frame as wire, and reads wire back as frame.
void assert_codes(Framer &f, const Bytes &frame, const Bytes &wire) {
    Bytes out;
    ASSERT_NIL(f.encode(frame, out));
    EXPECT_EQ(out, wire);
    f.write(wire);
    const auto got = ASSERT_NIL_P(f.next());
    ASSERT_TRUE(got.has_value());
    EXPECT_EQ(Bytes(got.value().begin(), got.value().end()), frame);
    EXPECT_FALSE(ASSERT_NIL_P(f.next()).has_value());
}

/// @brief asserts that reading wire with f drops one frame with text, then reads "ok",
/// which wire must end with in the framing of f.
void assert_drops(Framer &f, const Bytes &wire, const std::string &text) {
    f.write(wire);
    const auto [frame, err] = f.next();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, text);
    const auto ok = ASSERT_NIL_P(f.next());
    ASSERT_TRUE(ok.has_value());
    EXPECT_EQ(Bytes(ok.value().begin(), ok.value().end()), vec("ok"));
}

/// @brief returns random frames of 1 to 600 bytes, a quarter of them special.
std::vector<Bytes> random_frames(const std::uint8_t special) {
    // NOLINTNEXTLINE(bugprone-random-generator-seed): a fixed seed repeats each run.
    std::mt19937 rng(6322);
    std::vector<Bytes> frames(300);
    for (auto &p: frames) {
        p.resize(rng() % 600 + 1);
        for (auto &b: p)
            b = static_cast<std::uint8_t>(rng() % 4 == 0 ? special : rng());
    }
    return frames;
}
}

/// @brief it should encode and decode the standard COBS examples.
TEST(Cobs, Vectors) {
    const auto f = cobs();
    assert_codes(*f, {0x00}, {0x01, 0x01, 0x00});
    assert_codes(*f, {0x00, 0x00}, {0x01, 0x01, 0x01, 0x00});
    assert_codes(*f, {0x00, 0x11, 0x00}, {0x01, 0x02, 0x11, 0x01, 0x00});
    assert_codes(*f, {0x11, 0x22, 0x00, 0x33}, {0x03, 0x11, 0x22, 0x02, 0x33, 0x00});
    assert_codes(*f, {0x11, 0x22, 0x33, 0x44}, {0x05, 0x11, 0x22, 0x33, 0x44, 0x00});
    assert_codes(*f, {0x11, 0x00, 0x00, 0x00}, {0x02, 0x11, 0x01, 0x01, 0x01, 0x00});
    assert_codes(*f, run(0x01, 0xFE), join(join({0xFF}, run(0x01, 0xFE)), {0x00}));
    assert_codes(
        *f,
        join({0x00}, run(0x01, 0xFE)),
        join(join({0x01, 0xFF}, run(0x01, 0xFE)), {0x00})
    );
    assert_codes(
        *f,
        run(0x01, 0xFF),
        join(join({0xFF}, run(0x01, 0xFE)), {0x02, 0xFF, 0x00})
    );
    assert_codes(
        *f,
        join(run(0x02, 0xFF), {0x00}),
        join(join({0xFF}, run(0x02, 0xFF)), {0x01, 0x01, 0x00})
    );
}

/// @brief it should drop a frame whose code byte runs past its end, and read the frames
/// after it.
TEST(Cobs, DropInvalidFrame) {
    assert_drops(
        *cobs(),
        {0x05, 0x11, 0x22, 0x00, 0x03, 'o', 'k', 0x00},
        "dropped a frame with invalid COBS bytes"
    );
}

/// @brief it should drop a frame that decodes to more than MAX_SIZE bytes, and read the
/// frame after it.
TEST(Cobs, DropFrameOverMaxSize) {
    // Each 0x01 code byte decodes to one zero, so the raw frame stays under the
    // splitter bound.
    Bytes wire(MAX_SIZE + 2, 0x01);
    wire.insert(wire.end(), {0x00, 0x03, 'o', 'k', 0x00});
    assert_drops(*cobs(), wire, "dropped a frame longer than 65536 bytes");
}

/// @brief it should accept a frame of MAX_SIZE nonzero bytes, whose encoding is the
/// longest that the framer reads.
TEST(Cobs, AcceptMaxSize) {
    const auto f = cobs();
    const Bytes frame(MAX_SIZE, 0x11);
    Bytes wire;
    ASSERT_NIL(f->encode(frame, wire));
    EXPECT_EQ(wire.size(), MAX_SIZE + (MAX_SIZE / 254) + 2);
    f->write(wire);
    const auto got = ASSERT_NIL_P(f->next());
    ASSERT_TRUE(got.has_value());
    EXPECT_EQ(Bytes(got.value().begin(), got.value().end()), frame);
}

/// @brief it should drop an empty frame with no error.
TEST(Cobs, DropEmptyFrame) {
    const auto f = cobs();
    f->write(Bytes{0x01, 0x00, 0x03, 'o', 'k', 0x00});
    EXPECT_EQ(frames(*f, ""), (Frames{"ok"}));
}

/// @brief it should encode frames that a COBS framer reads back.
TEST(Cobs, RoundTrip) {
    const auto f = cobs();
    assert_round_trips(*f, random_frames(0x00));
}

/// @brief it should refuse to encode an empty frame.
TEST(Cobs, RefuseEmptyFrame) {
    assert_refuses(*cobs(), "", "the frame is empty");
}

/// @brief it should escape END and ESC bytes, and start and end a frame with END.
TEST(Slip, Vectors) {
    const auto f = slip();
    assert_codes(*f, {0x01, 0x02}, {0xC0, 0x01, 0x02, 0xC0});
    assert_codes(*f, {0xC0, 0xDB, 0x01}, {0xC0, 0xDB, 0xDC, 0xDB, 0xDD, 0x01, 0xC0});
    assert_codes(*f, {0xDC, 0xDD}, {0xC0, 0xDC, 0xDD, 0xC0});
}

/// @brief it should drop a frame with an invalid escape, and read the frames after it.
TEST(Slip, DropInvalidEscape) {
    assert_drops(
        *slip(),
        {0xC0, 0x01, 0xDB, 0x01, 0xC0, 'o', 'k', 0xC0},
        "dropped a frame with an invalid SLIP escape"
    );
    assert_drops(
        *slip(),
        {0x01, 0xDB, 0xC0, 'o', 'k', 0xC0},
        "dropped a frame with an invalid SLIP escape"
    );
}

/// @brief it should drop a frame that decodes to more than MAX_SIZE bytes.
TEST(Slip, DropFrameOverMaxSize) {
    Bytes wire(MAX_SIZE + 1, 0x01);
    wire.insert(wire.end(), {0xC0, 'o', 'k', 0xC0});
    assert_drops(*slip(), wire, "dropped a frame longer than 65536 bytes");
}

/// @brief it should accept a frame of MAX_SIZE bytes that each need an escape.
TEST(Slip, AcceptEscapedMaxSize) {
    const auto f = slip();
    const Bytes frame(MAX_SIZE, 0xC0);
    Bytes wire;
    ASSERT_NIL(f->encode(frame, wire));
    f->write(wire);
    const auto got = ASSERT_NIL_P(f->next());
    ASSERT_TRUE(got.has_value());
    EXPECT_EQ(got.value().size(), MAX_SIZE);
}

/// @brief it should encode frames that a SLIP framer reads back.
TEST(Slip, RoundTrip) {
    const auto f = slip();
    auto frames = random_frames(0xC0);
    for (std::size_t i = 0; i < frames.size(); i += 2)
        std::ranges::replace(frames[i], std::uint8_t{0xC0}, std::uint8_t{0xDB});
    assert_round_trips(*f, frames);
}

/// @brief it should refuse to encode an empty frame.
TEST(Slip, RefuseEmptyFrame) {
    assert_refuses(*slip(), "", "the frame is empty");
}

namespace {
/// @brief returns a sync framing with sync bytes AA 55 and a 1-byte length after them.
icd::SyncFraming
sync_framing(const std::optional<icd::Checksum> &checksum = std::nullopt) {
    return {.sync = {0xAA, 0x55}, .length_offset = 2, .checksum = checksum};
}

/// @brief returns a little-endian CRC-32 checksum.
icd::Checksum crc32() {
    return {.algorithm = icd::CHECKSUM_ALGORITHM_CRC_32};
}

/// @brief returns a frame with a blank 3-byte header before payload.
Bytes with_header(const Bytes &payload) {
    return join({0, 0, 0}, payload);
}

/// @brief asserts that f gives frames, then nothing, with no error.
void assert_frames(Framer &f, const std::vector<Bytes> &frames) {
    for (const auto &want: frames) {
        const auto got = ASSERT_NIL_P(f.next());
        ASSERT_TRUE(got.has_value());
        EXPECT_EQ(Bytes(got.value().begin(), got.value().end()), want);
    }
    EXPECT_FALSE(ASSERT_NIL_P(f.next()).has_value());
}

/// @brief asserts that the next call to f.next drops bytes with text.
void assert_next_drops(Framer &f, const std::string &text) {
    const auto [frame, err] = f.next();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, text);
    EXPECT_FALSE(frame.has_value());
}
}

/// @brief it should compute each checksum as its standard check value of "123456789".
TEST(Sync, ChecksumCheckValues) {
    const auto digits = vec("123456789");
    const auto check =
        [&](const icd::Checksum &c, const std::int32_t adjustment, const Bytes &sum) {
            // The sync bytes "12" and a length byte of '3' make the checksum cover the
            // nine digits alone.
            const auto f = ASSERT_NIL_P(create(
                icd::SyncFraming{
                    .sync = vec("12"),
                    .length_offset = 2,
                    .length_adjustment = adjustment,
                    .checksum = c,
                }
            ));
            assert_codes(*f, digits, join(digits, sum));
        };
    check(
        {
            .algorithm = icd::CHECKSUM_ALGORITHM_CRC_16_CCITT_FALSE,
            .byte_order = icd::BYTE_ORDER_BIG_ENDIAN,
        },
        -43,
        {0x29, 0xB1}
    );
    check({.algorithm = icd::CHECKSUM_ALGORITHM_CRC_16_MODBUS}, -43, {0x37, 0x4B});
    check(crc32(), -41, {0x26, 0x39, 0xF4, 0xCB});
}

/// @brief it should write the sync bytes and a big-endian length into the header.
TEST(Sync, EncodeHeader) {
    auto config = sync_framing();
    config.length_size = icd::LENGTH_SIZE_TWO;
    config.byte_order = icd::BYTE_ORDER_BIG_ENDIAN;
    config.length_adjustment = 1;
    const auto f = ASSERT_NIL_P(create(config));
    const Bytes wire = {0xAA, 0x55, 0x00, 0x02, 7, 8, 9};
    Bytes out;
    ASSERT_NIL(f->encode(Bytes{0, 0, 0, 0, 7, 8, 9}, out));
    EXPECT_EQ(out, wire);
    f->write(out);
    assert_frames(*f, {wire});
}

/// @brief it should drop the bytes before a sync sequence, and find one that two chunks
/// split.
TEST(Sync, DropBytesBeforeSync) {
    const auto f = ASSERT_NIL_P(create(sync_framing()));
    f->write(Bytes{1, 2, 0xAA});
    assert_next_drops(*f, "dropped 2 bytes that are not a frame");
    assert_frames(*f, {});
    f->write(Bytes{0x55, 1, 9});
    assert_frames(*f, {{0xAA, 0x55, 1, 9}});
}

/// @brief it should drop a frame with a bad checksum, and read the frame after it.
TEST(Sync, DropBadChecksum) {
    const auto f = ASSERT_NIL_P(create(sync_framing(crc32())));
    Bytes wire;
    ASSERT_NIL(f->encode(with_header({1, 2}), wire));
    wire[3] ^= 0xFF;
    ASSERT_NIL(f->encode(with_header({3}), wire));
    f->write(wire);
    assert_next_drops(*f, "dropped a frame with a bad checksum");
    assert_next_drops(*f, "dropped 8 bytes that are not a frame");
    assert_frames(*f, {{0xAA, 0x55, 5, 3}});
}

/// @brief it should drop a frame whose length is past the maximum or too short for its
/// checksum, and read the frame after it.
TEST(Sync, DropInvalidLength) {
    auto config = sync_framing(crc32());
    config.max_length = 16;
    const auto f = ASSERT_NIL_P(create(config));
    f->write(Bytes{0xAA, 0x55, 20, 0xAA, 0x55, 3});
    assert_next_drops(*f, "dropped a frame with an invalid length of 20");
    assert_next_drops(*f, "dropped 2 bytes that are not a frame");
    assert_next_drops(*f, "dropped a frame with an invalid length of 3");
    Bytes wire;
    ASSERT_NIL(f->encode(with_header({7}), wire));
    f->write(wire);
    assert_next_drops(*f, "dropped 2 bytes that are not a frame");
    assert_frames(*f, {{0xAA, 0x55, 5, 7}});
}

/// @brief it should drop a partial frame on idle, and read the frame after it.
TEST(Sync, IdleDropsPartialFrame) {
    const auto f = ASSERT_NIL_P(create(sync_framing()));
    f->write(Bytes{0xAA, 0x55, 200, 1, 2, 0xAA, 0x55, 1, 9});
    assert_frames(*f, {});
    const auto err = f->idle();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "dropped 5 bytes of a partial frame");
    assert_frames(*f, {{0xAA, 0x55, 1, 9}});
    ASSERT_NIL(f->idle());
}

/// @brief it should read a frame of max_length bytes, which its checksum follows.
TEST(Sync, AcceptMaxLength) {
    auto config = sync_framing(crc32());
    config.length_size = icd::LENGTH_SIZE_FOUR;
    const auto f = ASSERT_NIL_P(create(config));
    Bytes frame(MAX_SIZE, 0x11);
    Bytes wire;
    ASSERT_NIL(f->encode(frame, wire));
    EXPECT_EQ(wire.size(), MAX_SIZE + 4);
    std::ranges::copy(wire | std::views::take(6), frame.begin());
    f->write(wire);
    assert_frames(*f, {frame});
}

/// @brief it should encode frames that a framer of the same config reads back.
TEST(Sync, RoundTrip) {
    for (const std::string algorithm:
         {icd::CHECKSUM_ALGORITHM_CRC_16_CCITT_FALSE,
          icd::CHECKSUM_ALGORITHM_CRC_16_MODBUS,
          icd::CHECKSUM_ALGORITHM_CRC_32}) {
        auto config = sync_framing(
            icd::Checksum{
                .algorithm = algorithm,
                .byte_order = icd::BYTE_ORDER_BIG_ENDIAN,
            }
        );
        config.length_size = icd::LENGTH_SIZE_TWO;
        const auto f = ASSERT_NIL_P(create(config));
        std::vector<Bytes> frames;
        for (const auto &p: random_frames(0xAA)) {
            Bytes out;
            ASSERT_NIL(f->encode(with_header(p), out));
            frames.emplace_back(
                out.begin(),
                out.end() - (algorithm == icd::CHECKSUM_ALGORITHM_CRC_32 ? 4 : 2)
            );
        }
        assert_round_trips(*f, frames);
    }
}

/// @brief it should refuse to encode a frame that its header, length field, or maximum
/// length cannot hold.
TEST(Sync, RefuseEncode) {
    const auto f = ASSERT_NIL_P(create(sync_framing()));
    assert_refuses(
        *f,
        "ab",
        "the frame has 2 bytes, fewer than the 3 bytes of its header"
    );
    assert_refuses(
        *f,
        std::string(300, 'x'),
        "the length 297 does not fit a 1-byte length field"
    );
    auto config = sync_framing();
    config.max_length = 4;
    assert_refuses(
        *ASSERT_NIL_P(create(config)),
        "abcde",
        "the frame has 5 bytes, more than the maximum length of 4"
    );
}

namespace {
/// @brief asserts that create refuses c with text.
void assert_refuses_config(const icd::SyncFraming &c, const std::string &text) {
    const auto [f, err] = create(c);
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, text);
    EXPECT_EQ(f, nullptr);
}
}

/// @brief it should refuse a length field that overlaps the sync sequence.
TEST(Sync, RefuseLengthOverSync) {
    auto c = sync_framing();
    c.length_offset = 1;
    assert_refuses_config(c, "the length field overlaps the sync sequence");
}

/// @brief it should refuse a maximum length that cannot hold the header.
TEST(Sync, RefuseMaxLengthUnderHeader) {
    auto c = sync_framing();
    c.max_length = 2;
    assert_refuses_config(
        c,
        "the maximum length of 2 is less than the 3 bytes of the header"
    );
}

/// @brief it should refuse a length size, byte order, or checksum algorithm that it
/// does not know.
TEST(Sync, RefuseUnknownEnumValue) {
    auto c = sync_framing();
    c.length_size = "three";
    assert_refuses_config(c, "unknown length size three");
    c = sync_framing();
    c.byte_order = "middle_endian";
    assert_refuses_config(c, "unknown byte order middle_endian");
    c = sync_framing(icd::Checksum{.algorithm = "crc8"});
    assert_refuses_config(c, "unknown checksum algorithm crc8");
    c = sync_framing(
        icd::Checksum{
            .algorithm = icd::CHECKSUM_ALGORITHM_CRC_32,
            .byte_order = "middle_endian",
        }
    );
    assert_refuses_config(c, "unknown byte order middle_endian");
}

/// @brief it should encode a frame of MAX_SIZE bytes in every framing, and refuse one
/// more byte.
TEST(Framer, EncodeMaxSize) {
    std::vector<std::unique_ptr<Framer>> framers;
    framers.push_back(delimited_by("\n"));
    framers.push_back(fixed(MAX_SIZE));
    framers.push_back(cobs());
    framers.push_back(slip());
    framers.push_back(ASSERT_NIL_P(create(
        icd::SyncFraming{
            .sync = {0xAA},
            .length_offset = 1,
            .length_size = icd::LENGTH_SIZE_FOUR,
            .checksum = crc32(),
        }
    )));
    for (const auto &f: framers) {
        Bytes wire;
        ASSERT_NIL(f->encode(Bytes(MAX_SIZE, 0x11), wire));
        ASSERT_OCCURRED_AS(
            f->encode(Bytes(MAX_SIZE + 1, 0x11), wire),
            x::errors::VALIDATION
        );
    }
}

/// @brief it should split a stream without allocating once its buffer has grown.
TEST(Framer, SplitWithoutAllocating) {
    const auto d = delimited_by("\r\n");
    const auto f = fixed(4);
    const auto c = cobs();
    const auto s = slip();
    const auto y = ASSERT_NIL_P(create(sync_framing(crc32())));
    Bytes synced;
    ASSERT_NIL(y->encode(with_header({1, 2, 3}), synced));
    const auto chunk = bytes("abc\r\ndefg\r\nhi");
    const Bytes stuffed = {0x02, 'a', 0x02, 'b', 0x00};
    const Bytes escaped = {0x01, 0xC0, 0xDB, 0xDC, 0xC0};
    const auto drain = [](Framer &framer) {
        std::size_t n = 0;
        while (true) {
            const auto frame = ASSERT_NIL_P(framer.next());
            if (!frame.has_value()) return n;
            n++;
        }
    };
    d->write(chunk);
    f->write(chunk);
    c->write(stuffed);
    s->write(escaped);
    y->write(synced);
    drain(*y);
    drain(*d);
    drain(*f);
    drain(*c);
    drain(*s);
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++) {
        d->write(chunk);
        f->write(chunk);
        c->write(stuffed);
        s->write(escaped);
        y->write(synced);
        EXPECT_EQ(drain(*y), 1);
        EXPECT_EQ(drain(*d), 2);
        EXPECT_GE(drain(*f), 3);
        EXPECT_EQ(drain(*c), 1);
        EXPECT_EQ(drain(*s), 2);
    }
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
}
}
