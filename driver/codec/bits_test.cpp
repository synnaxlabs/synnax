// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <random>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/codec/bits.h"
#include "driver/codec/errors.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

icd::BinaryBaseField
field(const std::uint16_t start, const std::uint8_t length, const bool big) {
    icd::BinaryBaseField f;
    f.start_bit = start;
    f.bit_length = length;
    f.byte_order = big ? icd::BYTE_ORDER_BIG_ENDIAN : icd::BYTE_ORDER_LITTLE_ENDIAN;
    return f;
}

/// @brief Walker visits the bits of a field one at a time by the DBC rules: up from the
/// start bit for little-endian fields, and down through each byte from the start bit,
/// then on to bit 7 of the next byte, for big-endian fields.
struct Walker {
    std::size_t start;
    std::size_t length;
    bool big;

    /// @brief returns the payload position of each bit of the field, least significant
    /// first.
    [[nodiscard]] std::vector<std::size_t> positions() const {
        std::vector<std::size_t> out;
        if (!this->big) {
            for (std::size_t i = 0; i < this->length; i++)
                out.push_back(this->start + i);
            return out;
        }
        std::size_t pos = this->start;
        for (std::size_t i = 0; i < this->length; i++) {
            out.push_back(pos);
            pos = pos % 8 == 0 ? pos + 15 : pos - 1;
        }
        std::ranges::reverse(out);
        return out;
    }

    [[nodiscard]] std::size_t end() const {
        return std::ranges::max(this->positions()) / 8 + 1;
    }

    [[nodiscard]] std::uint64_t read(const std::vector<std::uint8_t> &payload) const {
        std::uint64_t raw = 0;
        const auto ps = this->positions();
        for (std::size_t i = 0; i < ps.size(); i++)
            raw |= static_cast<std::uint64_t>((payload[ps[i] / 8] >> (ps[i] % 8)) & 1)
                << i;
        return raw;
    }

    void write(std::vector<std::uint8_t> &payload, const std::uint64_t raw) const {
        const auto ps = this->positions();
        for (std::size_t i = 0; i < ps.size(); i++) {
            const auto mask = static_cast<std::uint8_t>(1U << (ps[i] % 8));
            if (((raw >> i) & 1) != 0)
                payload[ps[i] / 8] |= mask;
            else
                payload[ps[i] / 8] &= static_cast<std::uint8_t>(~mask);
        }
    }
};
}

/// @brief it should read a little-endian field that crosses a byte boundary.
TEST(BitRange, ReadLittleEndianAcrossBytes) {
    const auto r = ASSERT_NIL_P(BitRange::create(field(4, 12, false)));
    const std::vector<std::uint8_t> payload = {0xB0, 0xCA};
    EXPECT_EQ(r.read(payload.data()), 0xCAB);
    EXPECT_EQ(r.end(), 2);
}

/// @brief it should read a big-endian field from its most significant start bit into
/// the next byte.
TEST(BitRange, ReadBigEndianAcrossBytes) {
    const auto r = ASSERT_NIL_P(BitRange::create(field(3, 12, true)));
    const std::vector<std::uint8_t> payload = {0x0A, 0xBC};
    EXPECT_EQ(r.read(payload.data()), 0xABC);
    EXPECT_EQ(r.end(), 2);
}

/// @brief it should read a field as a two's complement integer of its bit length.
TEST(BitRange, ReadSigned) {
    const auto r = ASSERT_NIL_P(BitRange::create(field(0, 12, false)));
    const std::vector<std::uint8_t> negative = {0xFF, 0x0F};
    EXPECT_EQ(r.read_signed(negative.data()), -1);
    const std::vector<std::uint8_t> positive = {0xFF, 0x07};
    EXPECT_EQ(r.read_signed(positive.data()), 2047);
}

/// @brief it should write a 64-bit field across nine bytes and keep the bits next to
/// it.
TEST(BitRange, Write64BitsAcrossNineBytes) {
    const auto r = ASSERT_NIL_P(BitRange::create(field(4, 64, false)));
    std::vector<std::uint8_t> payload(9, 0xFF);
    r.write(payload.data(), 0xFEDCBA9876543210);
    EXPECT_EQ(r.read(payload.data()), 0xFEDCBA9876543210);
    EXPECT_EQ(r.end(), 9);
    EXPECT_EQ(payload[0] & 0x0F, 0x0F);
    EXPECT_EQ(payload[8] & 0xF0, 0xF0);
}

/// @brief it should reject a bit length of zero.
TEST(BitRange, RejectZeroLength) {
    const auto [r, err] = BitRange::create(field(0, 0, false));
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "bit length must be from 1 to 64, got 0");
}

/// @brief it should reject a bit length over 64.
TEST(BitRange, RejectLengthOver64) {
    const auto [r, err] = BitRange::create(field(0, 65, false));
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "bit length must be from 1 to 64, got 65");
}

/// @brief it should reject an unknown byte order.
TEST(BitRange, RejectUnknownByteOrder) {
    auto f = field(0, 8, false);
    f.byte_order = "middle_endian";
    const auto [r, err] = BitRange::create(f);
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "unknown byte order middle_endian");
}

/// @brief it should read and write the same bits as a walk of the DBC rules, for any
/// layout.
TEST(BitRange, MatchDBCWalk) {
    // NOLINTNEXTLINE(bugprone-random-generator-seed): a fixed seed repeats each run.
    std::mt19937_64 rng(6321);
    for (int trial = 0; trial < 20000; trial++) {
        const bool big = rng() % 2 == 0;
        const auto length = static_cast<std::uint8_t>(rng() % 64 + 1);
        const auto size = static_cast<std::size_t>(rng() % 16 + 9);
        const auto start = static_cast<std::uint16_t>(rng() % (size * 8));
        const Walker ref{.start = start, .length = length, .big = big};
        if (ref.end() > size) continue;
        SCOPED_TRACE(
            "start " + std::to_string(start) + ", length " + std::to_string(length) +
            (big ? ", big-endian" : ", little-endian")
        );
        const auto r = ASSERT_NIL_P(BitRange::create(field(start, length, big)));
        ASSERT_EQ(r.end(), ref.end());
        std::vector<std::uint8_t> payload(size);
        for (auto &b: payload)
            b = static_cast<std::uint8_t>(rng());
        const auto raw = ref.read(payload);
        ASSERT_EQ(r.read(payload.data()), raw);
        const auto shift = 64 - length;
        ASSERT_EQ(
            r.read_signed(payload.data()),
            static_cast<std::int64_t>(raw << shift) >> shift
        );
        const auto next = rng();
        auto written = payload;
        auto expected = payload;
        r.write(written.data(), next);
        ref.write(expected, next);
        ASSERT_EQ(written, expected);
    }
}
}
