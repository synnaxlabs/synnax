// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <cstdint>
#include <iostream>
#include <random>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/library/types.gen.h"
#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/arinc429.h"
#include "driver/codec/errors.h"
#include "driver/codec/plan.h"

namespace driver::codec::arinc429 {
namespace {
namespace library = synnax::library;

library::BinaryField field(
    const std::string &name,
    const std::uint16_t start_bit,
    const std::uint8_t bit_length,
    const bool signed_ = false,
    const double scale = 1
) {
    library::BinaryField f;
    f.key = x::uuid::create();
    f.name = name;
    f.start_bit = start_bit;
    f.bit_length = bit_length;
    f.signed_ = signed_;
    f.scale = scale;
    return f;
}

library::MessageEntry message(
    const std::uint8_t label,
    std::vector<library::BinaryField> fields,
    const std::optional<std::uint8_t> sdi = std::nullopt
) {
    library::MessageEntry m;
    m.key = x::uuid::create();
    m.name = "label";
    m.identifier = library::Arinc429Identifier{
        .label = label,
        .sdi = sdi.value_or(0),
        .sdi_matched = sdi.has_value(),
    };
    for (auto &f: fields)
        m.fields.emplace_back(std::move(f));
    return m;
}

/// @brief encodes values into a word the way a write task does.
Word encode(const library::MessageEntry &m, const std::vector<double> &values) {
    const auto plan = ASSERT_NIL_P(Plan::compile(m));
    auto v = plan.values();
    for (std::size_t i = 0; i < values.size(); i++)
        v.set(i, values[i]);
    std::vector<std::uint8_t> payload(PAYLOAD_SIZE, 0);
    const auto err = plan.encode(v, payload);
    EXPECT_FALSE(err) << err;
    const auto &id = std::get<library::Arinc429Identifier>(*m.identifier);
    return Word::pack(
        id.label,
        id.sdi,
        id.sdi_matched,
        std::span<const std::uint8_t, PAYLOAD_SIZE>(payload.data(), PAYLOAD_SIZE)
    );
}

/// @brief decodes a word the way a read task does.
std::vector<double> decode(const library::MessageEntry &m, const Word word) {
    const auto plan = ASSERT_NIL_P(Plan::compile(m));
    auto v = plan.values();
    const auto bytes = word.bytes();
    const auto err = plan.decode(bytes, v);
    EXPECT_FALSE(err) << err;
    std::vector<double> out;
    for (std::size_t i = 0; i < plan.size(); i++)
        out.push_back(v.get(i));
    return out;
}

library::MessageEntry altitude() {
    return message(0203, {field("altitude", 11, 18, true), field("ssm", 29, 2)});
}
}

TEST(Label, ReversesTheBitsOfTheLabel) {
    // Label 213 is written to the label octet as 0xD1.
    EXPECT_EQ(reverse(0213), 0xD1);
    for (int l = 0; l < 256; l++)
        EXPECT_EQ(reverse(reverse(static_cast<std::uint8_t>(l))), l);
}

TEST(Word, ReadsThePublishedLabel260Word) {
    const Word w(0x918C440D);
    EXPECT_EQ(w.label(), 0260);
    EXPECT_EQ(w.sdi(), 0);
    EXPECT_EQ(w.ssm(), 0);
    EXPECT_TRUE(w.parity_valid());
}

TEST(Word, FailsParityWhenAnyBitFlips) {
    const auto w = Word(0x918C440D);
    for (int bit = 0; bit < 32; bit++)
        EXPECT_FALSE(Word(w.raw() ^ 1u << bit).parity_valid()) << bit;
}

TEST(Word, SetsOddParity) {
    EXPECT_EQ(Word(0).with_parity().raw(), 0x80000000);
    EXPECT_EQ(Word(1).with_parity().raw(), 1);
    EXPECT_EQ(Word(0x80000000).with_parity().raw(), 0x80000000);
}

TEST(Word, PacksTheLabelSDIAndParityAroundThePayload) {
    const std::array<std::uint8_t, 4> payload{0xFF, 0x0C, 0x00, 0x60};
    const auto w = Word::pack(0203, 2, true, payload);
    EXPECT_EQ(w.label(), 0203);
    EXPECT_EQ(w.sdi(), 2);
    EXPECT_EQ(w.ssm(), 3);
    EXPECT_TRUE(w.parity_valid());
    EXPECT_EQ(w.raw() & 0x7FFFFC00, 0x60000C00u);
}

TEST(Word, KeepsThePayloadSDIBitsWhenTheSDIIsNotMatched) {
    const std::array<std::uint8_t, 4> payload{0x00, 0x03, 0x00, 0x00};
    EXPECT_EQ(Word::pack(0203, 0, false, payload).sdi(), 3);
    EXPECT_EQ(Word::pack(0203, 1, true, payload).sdi(), 1);
}

/// Selected airspeed, label 103: 11 bits with a scale of 512 knots, so bit 19 is 0.5
/// knots. The published example sets bits 28, 23, and 22 for 268 knots. The example
/// leaves parity zero, so the word here carries the parity the standard requires.
TEST(Golden, EncodesThePublishedLabel103AirspeedBNR) {
    const auto m = message(
        0103,
        {field("airspeed", 18, 11, true, 0.5), field("ssm", 29, 2)}
    );
    const auto w = encode(m, {268, 3});
    EXPECT_EQ(w.raw(), 0xE86000C2);
    EXPECT_EQ(decode(m, w), (std::vector<double>{268, 3}));
}

TEST(Golden, DecodesANegativeBNRValue) {
    const auto m = message(0103, {field("airspeed", 18, 11, true, 0.5)});
    EXPECT_EQ(decode(m, encode(m, {-268}))[0], -268);
}

/// Altitude, label 203: 17 bits with a resolution of 1 foot and a sign at bit 29.
TEST(Golden, EncodesLabel203AltitudeBNR) {
    EXPECT_EQ(encode(altitude(), {35000, 3}).raw(), 0x6445C0C1);
    EXPECT_EQ(encode(altitude(), {-1000, 3}).raw(), 0x7FE0C0C1);
    EXPECT_EQ(decode(altitude(), Word(0x6445C0C1)), (std::vector<double>{35000, 3}));
    EXPECT_EQ(decode(altitude(), Word(0x7FE0C0C1)), (std::vector<double>{-1000, 3}));
}

/// DME distance, label 201: the published BCD example carries 25786 as five digits.
/// A BCD word is one 4-bit field per digit, with a 3-bit most significant digit.
TEST(Golden, DecodesThePublishedBCDDigits) {
    const auto m = message(
        0201,
        {
            field("digit_1", 26, 3),
            field("digit_2", 22, 4),
            field("digit_3", 18, 4),
            field("digit_4", 14, 4),
            field("digit_5", 10, 4),
            field("ssm", 29, 2),
        }
    );
    const Word w(0x095E1881);
    EXPECT_TRUE(w.parity_valid());
    EXPECT_EQ(w.label(), 0201);
    EXPECT_EQ(decode(m, w), (std::vector<double>{2, 5, 7, 8, 6, 0}));
    EXPECT_EQ(encode(m, {2, 5, 7, 8, 6, 0}), w);
}

TEST(Golden, DecodesDiscreteBits) {
    const auto m = message(
        0270,
        {field("a", 10, 1), field("b", 11, 1), field("c", 12, 1)},
        0
    );
    const auto w = encode(m, {1, 0, 1});
    EXPECT_EQ(w.raw() >> 10 & 0x7, 0b101u);
    EXPECT_EQ(decode(m, w), (std::vector<double>{1, 0, 1}));
}

TEST(RoundTrip, PreservesRandomBNRValuesLabelsAndSDIs) {
    std::random_device rd;
    const auto start = rd();
    std::cout << "generator start " << start << std::endl;
    std::mt19937 gen(start);
    std::uniform_int_distribution<int> byte(0, 255);
    std::uniform_int_distribution<int> sdi_dist(0, 3);
    std::uniform_int_distribution<int> sig_dist(1, 18);
    for (int i = 0; i < 2000; i++) {
        const auto label = static_cast<std::uint8_t>(byte(gen));
        const auto sdi = static_cast<std::uint8_t>(sdi_dist(gen));
        const int sig = sig_dist(gen);
        const double resolution = std::ldexp(1.0, byte(gen) % 16 - 8);
        const auto m = message(
            label,
            {field(
                 "value",
                 static_cast<std::uint16_t>(28 - sig),
                 sig + 1,
                 true,
                 resolution
             ),
             field("ssm", 29, 2)},
            sdi
        );
        const std::int64_t max = (std::int64_t{1} << sig) - 1;
        std::uniform_int_distribution<std::int64_t> raw(-max - 1, max);
        const double value = static_cast<double>(raw(gen)) * resolution;
        const double ssm = sdi_dist(gen);
        const auto w = encode(m, {value, ssm});
        ASSERT_TRUE(w.parity_valid());
        ASSERT_EQ(w.label(), label);
        ASSERT_EQ(w.sdi(), sdi);
        ASSERT_EQ(decode(m, w), (std::vector<double>{value, ssm}))
            << "generator start " << start;
    }
}

TEST(Validate, AcceptsAFieldInTheDataBits) {
    ASSERT_NIL(validate(altitude()));
}

TEST(Validate, RejectsAFieldOverTheLabel) {
    ASSERT_OCCURRED_AS(validate(message(0203, {field("x", 7, 4)})), CONFIG_ERROR);
}

TEST(Validate, RejectsAFieldOverTheParityBit) {
    ASSERT_OCCURRED_AS(validate(message(0203, {field("x", 29, 3)})), CONFIG_ERROR);
}

TEST(Validate, RejectsAFieldOverTheSDIWhenTheSDIIsMatched) {
    ASSERT_OCCURRED_AS(validate(message(0203, {field("x", 8, 4)}, 1)), CONFIG_ERROR);
    ASSERT_NIL(validate(message(0203, {field("x", 8, 4)})));
}

TEST(Validate, RejectsABigEndianField) {
    auto f = field("x", 12, 4);
    f.byte_order = library::BYTE_ORDER_BIG_ENDIAN;
    ASSERT_OCCURRED_AS(validate(message(0203, {f})), CONFIG_ERROR);
}

TEST(Validate, RejectsAnotherIdentifier) {
    auto m = altitude();
    m.identifier = library::CanIdentifier{.id = 1};
    ASSERT_OCCURRED_AS(validate(m), CONFIG_ERROR);
    m.identifier = std::nullopt;
    ASSERT_OCCURRED_AS(validate(m), CONFIG_ERROR);
}

TEST(Validate, RejectsALengthOtherThan4) {
    auto m = altitude();
    m.length = 8;
    ASSERT_OCCURRED_AS(validate(m), CONFIG_ERROR);
}
}
