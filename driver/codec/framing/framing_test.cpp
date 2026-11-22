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
#include <random>
#include <span>
#include <string>
#include <string_view>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/bench/alloc.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/codec/framing/framing.h"

namespace driver::codec::framing {
namespace {
std::span<const std::uint8_t> bytes(const std::string_view s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

std::vector<std::uint8_t> vec(const std::string_view s) {
    return {s.begin(), s.end()};
}

std::unique_ptr<Framer> delimited_by(const std::string_view d) {
    return ASSERT_NIL_P(delimited(vec(d)));
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

/// @brief it should refuse an empty delimiter.
TEST(Delimited, RefuseEmptyDelimiter) {
    const auto [f, err] = delimited({});
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "the delimiter is empty");
}

/// @brief it should split a stream into frames of its length.
TEST(Fixed, Split) {
    const auto f = ASSERT_NIL_P(fixed(3));
    EXPECT_EQ(frames(*f, "abcde"), (Frames{"abc"}));
    EXPECT_EQ(frames(*f, "fghi"), (Frames{"def", "ghi"}));
}

/// @brief it should drop a partial frame when the stream goes quiet, so the next frame
/// starts in step.
TEST(Fixed, DropPartialFrameWhenIdle) {
    const auto f = ASSERT_NIL_P(fixed(3));
    EXPECT_EQ(frames(*f, "ab"), Frames{});
    const auto err = f->idle();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "dropped 2 bytes of a partial frame");
    ASSERT_NIL(f->idle());
    EXPECT_EQ(frames(*f, "xyz"), (Frames{"xyz"}));
}

/// @brief it should drop a partial frame on clear.
TEST(Fixed, Clear) {
    const auto f = ASSERT_NIL_P(fixed(2));
    EXPECT_EQ(frames(*f, "a"), Frames{});
    f->clear();
    EXPECT_EQ(frames(*f, "bc"), (Frames{"bc"}));
}

/// @brief it should encode frames that a framer of the same length reads back.
TEST(Fixed, RoundTrip) {
    const auto f = ASSERT_NIL_P(fixed(5));
    std::vector<std::vector<std::uint8_t>> payloads;
    payloads.reserve(100);
    for (std::uint8_t i = 0; i < 100; i++)
        payloads.push_back({i, 0, '\n', 255, i});
    assert_round_trips(*f, payloads);
}

/// @brief it should refuse to encode a frame of another length.
TEST(Fixed, RefuseEncode) {
    assert_refuses(
        *ASSERT_NIL_P(fixed(4)),
        "abc",
        "the frame has 3 bytes, not the fixed length of 4"
    );
}

/// @brief it should refuse a length of zero or past MAX_SIZE.
TEST(Fixed, RefuseLength) {
    for (const auto length: {std::size_t{0}, MAX_SIZE + 1}) {
        const auto [f, err] = fixed(length);
        ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
        EXPECT_EQ(
            err.data,
            "the length must be from 1 to 65536, not " + std::to_string(length)
        );
    }
}

/// @brief it should split a stream without allocating once its buffer has grown.
TEST(Framer, SplitWithoutAllocating) {
    const auto d = delimited_by("\r\n");
    const auto f = ASSERT_NIL_P(fixed(4));
    const auto chunk = bytes("abc\r\ndefg\r\nhi");
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
    drain(*d);
    drain(*f);
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++) {
        d->write(chunk);
        f->write(chunk);
        EXPECT_EQ(drain(*d), 2);
        EXPECT_GE(drain(*f), 3);
    }
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
}
}
