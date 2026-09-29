// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <random>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/library/types.gen.h"
#include "x/cpp/test/test.h"

#include "driver/codec/bits.h"
#include "driver/codec/errors.h"

namespace driver::codec {
namespace {
namespace library = synnax::library;

/// @brief Walker visits a field's bits one at a time from least to most significant,
/// following the DBC rules directly rather than the codec's linearization.
struct Walker {
    std::size_t start;
    std::size_t length;
    bool big;

    /// @returns the payload bit position of each field bit, least significant first.
    [[nodiscard]] std::vector<std::size_t> positions() const {
        if (!big) {
            std::vector<std::size_t> out;
            for (std::size_t i = 0; i < length; i++)
                out.push_back(start + i);
            return out;
        }
        std::vector<std::size_t> msb_first;
        std::size_t pos = start;
        for (std::size_t i = 0; i < length; i++) {
            msb_first.push_back(pos);
            pos = pos % 8 == 0 ? pos + 15 : pos - 1;
        }
        return {msb_first.rbegin(), msb_first.rend()};
    }

    [[nodiscard]] std::size_t end() const {
        std::size_t last = 0;
        for (const auto p: positions())
            last = std::max(last, p);
        return last / 8 + 1;
    }

    [[nodiscard]] std::uint64_t read(const std::vector<std::uint8_t> &payload) const {
        std::uint64_t raw = 0;
        const auto ps = positions();
        for (std::size_t i = 0; i < ps.size(); i++)
            raw |= static_cast<std::uint64_t>((payload[ps[i] / 8] >> (ps[i] % 8)) & 1)
                << i;
        return raw;
    }

    void write(std::vector<std::uint8_t> &payload, const std::uint64_t raw) const {
        const auto ps = positions();
        for (std::size_t i = 0; i < ps.size(); i++) {
            const auto mask = static_cast<std::uint8_t>(1u << (ps[i] % 8));
            if ((raw >> i) & 1)
                payload[ps[i] / 8] |= mask;
            else
                payload[ps[i] / 8] &= static_cast<std::uint8_t>(~mask);
        }
    }
};

const char *order(const bool big) {
    return big ? library::BYTE_ORDER_BIG_ENDIAN : library::BYTE_ORDER_LITTLE_ENDIAN;
}
}

TEST(BitRange, ReadsALittleEndianFieldAcrossAByteBoundary) {
    const auto r = ASSERT_NIL_P(BitRange::compile(4, 12, order(false)));
    const std::vector<std::uint8_t> payload = {0xB0, 0xCA};
    EXPECT_EQ(r.read(payload.data()), 0xCAB);
    EXPECT_EQ(r.end(), 2);
}

TEST(BitRange, ReadsABigEndianFieldAcrossAByteBoundary) {
    // Motorola start bit 3 is the MSB: 4 bits of byte 0, then all of byte 1.
    const auto r = ASSERT_NIL_P(BitRange::compile(3, 12, order(true)));
    const std::vector<std::uint8_t> payload = {0x0A, 0xBC};
    EXPECT_EQ(r.read(payload.data()), 0xABC);
    EXPECT_EQ(r.end(), 2);
}

TEST(BitRange, SignExtendsNegativeValues) {
    const auto r = ASSERT_NIL_P(BitRange::compile(0, 12, order(false)));
    const std::vector<std::uint8_t> payload = {0xFF, 0x0F};
    EXPECT_EQ(r.read_signed(payload.data()), -1);
    const std::vector<std::uint8_t> positive = {0xFF, 0x07};
    EXPECT_EQ(r.read_signed(positive.data()), 2047);
}

TEST(BitRange, ReadsA64BitFieldSpanningNineBytes) {
    const auto r = ASSERT_NIL_P(BitRange::compile(4, 64, order(false)));
    std::vector<std::uint8_t> payload(9, 0);
    r.write(payload.data(), 0xFEDCBA9876543210);
    EXPECT_EQ(r.read(payload.data()), 0xFEDCBA9876543210);
    EXPECT_EQ(r.end(), 9);
    EXPECT_EQ(payload[0] & 0x0F, 0);
    EXPECT_EQ(payload[8] & 0xF0, 0);
}

TEST(BitRange, RejectsAZeroLength) {
    ASSERT_OCCURRED_AS_P(BitRange::compile(0, 0, order(false)), LAYOUT_ERROR);
}

TEST(BitRange, RejectsALengthOver64) {
    ASSERT_OCCURRED_AS_P(BitRange::compile(0, 65, order(false)), LAYOUT_ERROR);
}

TEST(BitRange, RejectsAnUnknownByteOrder) {
    ASSERT_OCCURRED_AS_P(BitRange::compile(0, 8, "middle_endian"), LAYOUT_ERROR);
}

TEST(BitRange, ReadsABigEndianWordSwappedField) {
    const auto r = ASSERT_NIL_P(
        BitRange::compile(7, 32, BYTE_ORDER_BIG_ENDIAN_WORD_SWAPPED)
    );
    const std::vector<std::uint8_t> payload = {0x56, 0x78, 0x12, 0x34};
    EXPECT_EQ(r.read(payload.data()), 0x12345678);
    EXPECT_EQ(r.end(), 4);
}

TEST(BitRange, ReadsALittleEndianWordSwappedField) {
    const auto r = ASSERT_NIL_P(
        BitRange::compile(0, 32, BYTE_ORDER_LITTLE_ENDIAN_WORD_SWAPPED)
    );
    const std::vector<std::uint8_t> payload = {0x34, 0x12, 0x78, 0x56};
    EXPECT_EQ(r.read(payload.data()), 0x12345678);
}

TEST(BitRange, ReversesEveryWordOfA64BitWordSwappedField) {
    const auto r = ASSERT_NIL_P(
        BitRange::compile(15, 64, BYTE_ORDER_BIG_ENDIAN_WORD_SWAPPED)
    );
    std::vector<std::uint8_t> payload(10, 0xEE);
    r.write(payload.data(), 0x0102030405060708);
    EXPECT_EQ(
        payload,
        (std::vector<
            std::uint8_t>{0xEE, 0x07, 0x08, 0x05, 0x06, 0x03, 0x04, 0x01, 0x02, 0xEE})
    );
    EXPECT_EQ(r.read(payload.data()), 0x0102030405060708);
    EXPECT_EQ(r.end(), 9);
}

TEST(BitRange, TreatsAWordSwapped16BitFieldAsItsBaseOrder) {
    const std::vector<std::uint8_t> payload = {0x12, 0x34};
    const auto big = ASSERT_NIL_P(
        BitRange::compile(7, 16, BYTE_ORDER_BIG_ENDIAN_WORD_SWAPPED)
    );
    EXPECT_EQ(big.read(payload.data()), 0x1234);
    const auto little = ASSERT_NIL_P(
        BitRange::compile(0, 16, BYTE_ORDER_LITTLE_ENDIAN_WORD_SWAPPED)
    );
    EXPECT_EQ(little.read(payload.data()), 0x3412);
}

TEST(BitRange, RejectsAWordSwappedFieldThatIsNotByteAligned) {
    ASSERT_OCCURRED_AS_P(
        BitRange::compile(4, 32, BYTE_ORDER_LITTLE_ENDIAN_WORD_SWAPPED),
        LAYOUT_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        BitRange::compile(3, 32, BYTE_ORDER_BIG_ENDIAN_WORD_SWAPPED),
        LAYOUT_ERROR
    );
}

TEST(BitRange, RejectsAWordSwappedFieldThatIsNotWholeWords) {
    ASSERT_OCCURRED_AS_P(
        BitRange::compile(0, 24, BYTE_ORDER_LITTLE_ENDIAN_WORD_SWAPPED),
        LAYOUT_ERROR
    );
}

TEST(BitRange, MatchesItsBaseOrderOverAWordReversedPayload) {
    std::mt19937_64 rng(4972);
    for (int trial = 0; trial < 2000; trial++) {
        const bool big = rng() % 2 == 0;
        const auto words = static_cast<std::size_t>(rng() % 4 + 1);
        const auto byte = static_cast<std::size_t>(rng() % 8);
        const auto start = static_cast<std::uint16_t>(byte * 8 + (big ? 7 : 0));
        const auto length = static_cast<std::uint8_t>(words * 16);
        const auto swapped = ASSERT_NIL_P(
            BitRange::compile(
                start,
                length,
                big ? BYTE_ORDER_BIG_ENDIAN_WORD_SWAPPED
                    : BYTE_ORDER_LITTLE_ENDIAN_WORD_SWAPPED
            )
        );
        const auto base = ASSERT_NIL_P(BitRange::compile(start, length, order(big)));
        std::vector<std::uint8_t> payload(byte + words * 2 + 2);
        for (auto &b: payload)
            b = static_cast<std::uint8_t>(rng());
        auto reversed = payload;
        for (std::size_t w = 0; w < words; w++)
            for (std::size_t i = 0; i < 2; i++)
                reversed[byte + w * 2 + i] = payload[byte + (words - 1 - w) * 2 + i];
        ASSERT_EQ(swapped.read(payload.data()), base.read(reversed.data()));

        const auto raw = rng();
        auto written = payload;
        auto expected = reversed;
        swapped.write(written.data(), raw);
        base.write(expected.data(), raw);
        auto unreversed = expected;
        for (std::size_t w = 0; w < words; w++)
            for (std::size_t i = 0; i < 2; i++)
                unreversed[byte + w * 2 + i] = expected[byte + (words - 1 - w) * 2 + i];
        ASSERT_EQ(written, unreversed);
    }
}

TEST(BitRange, MatchesAReferenceExtractorOverRandomLayouts) {
    std::mt19937_64 rng(4971);
    for (int trial = 0; trial < 20000; trial++) {
        const bool big = rng() % 2 == 0;
        const auto length = static_cast<std::uint8_t>(rng() % 64 + 1);
        const auto size = static_cast<std::size_t>(rng() % 16 + 9);
        const auto start = static_cast<std::uint16_t>(rng() % (size * 8));
        const Walker ref{start, length, big};
        if (ref.end() > size) continue;
        const auto r = ASSERT_NIL_P(BitRange::compile(start, length, order(big)));
        ASSERT_EQ(r.end(), ref.end());

        std::vector<std::uint8_t> payload(size);
        for (auto &b: payload)
            b = static_cast<std::uint8_t>(rng());
        ASSERT_EQ(r.read(payload.data()), ref.read(payload))
            << "start=" << start << " length=" << int(length) << " big=" << big;
        const auto shift = 64 - length;
        const auto expected_signed = static_cast<std::int64_t>(
                                         ref.read(payload) << shift
                                     ) >>
                                     shift;
        ASSERT_EQ(r.read_signed(payload.data()), expected_signed);

        const auto raw = rng();
        auto written = payload;
        auto expected = payload;
        r.write(written.data(), raw);
        ref.write(expected, raw);
        ASSERT_EQ(written, expected)
            << "start=" << start << " length=" << int(length) << " big=" << big;
    }
}
}
