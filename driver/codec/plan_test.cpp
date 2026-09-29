// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <limits>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/library/types.gen.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/codec/plan.h"

namespace driver::codec {
namespace {
namespace library = synnax::library;

library::BinaryField binary(
    const std::string &name,
    const std::uint16_t start_bit,
    const std::uint8_t bit_length,
    const bool signed_ = false
) {
    library::BinaryField f;
    f.key = x::uuid::create();
    f.name = name;
    f.start_bit = start_bit;
    f.bit_length = bit_length;
    f.signed_ = signed_;
    return f;
}

library::BinaryField multiplexed(
    library::BinaryField f,
    const library::BinaryField &mux,
    const std::int32_t value
) {
    f.multiplexor = mux.key;
    f.multiplex_values = {value};
    return f;
}

library::DelimitedField delimited(const std::string &name, const std::uint32_t pos) {
    library::DelimitedField f;
    f.key = x::uuid::create();
    f.name = name;
    f.position = pos;
    return f;
}

library::TaggedField tagged(const std::string &name, const std::string &tag) {
    library::TaggedField f;
    f.key = x::uuid::create();
    f.name = name;
    f.tag = tag;
    return f;
}

library::MessageEntry message(std::vector<library::Field> fields) {
    library::MessageEntry m;
    m.key = x::uuid::create();
    m.name = "message";
    m.fields = std::move(fields);
    return m;
}

library::MessageEntry text(std::vector<library::Field> fields) {
    auto m = message(std::move(fields));
    m.format = library::FORMAT_TEXT;
    return m;
}

std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

std::string str(const std::vector<std::uint8_t> &b) {
    return {b.begin(), b.end()};
}
}

TEST(PlanCompile, RejectsAnUnknownFormat) {
    auto m = message({binary("a", 0, 8)});
    m.format = "hex";
    ASSERT_OCCURRED_AS_P(Plan::compile(m), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAKeyThatIsNotInTheMessage) {
    const auto m = message({binary("a", 0, 8)});
    ASSERT_OCCURRED_AS_P(Plan::compile(m, {x::uuid::create()}), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsADuplicateKey) {
    const auto a = binary("a", 0, 8);
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a}), {a.key, a.key}), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAZeroScale) {
    auto a = binary("a", 0, 8);
    a.scale = 0;
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAFloatThatIsNot32Or64Bits) {
    auto a = binary("a", 0, 16);
    a.float_ = true;
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsATextFieldInABinaryMessage) {
    ASSERT_OCCURRED_AS_P(Plan::compile(message({delimited("a", 0)})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsABinaryFieldInATextMessage) {
    ASSERT_OCCURRED_AS_P(Plan::compile(text({binary("a", 0, 8)})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsFieldsBeyondTheMessageLength) {
    auto m = message({binary("a", 56, 16)});
    m.length = 8;
    ASSERT_OCCURRED_AS_P(Plan::compile(m), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAMultiplexorCycle) {
    auto a = binary("a", 0, 8);
    auto b = binary("b", 8, 8);
    a.multiplexor = b.key;
    a.multiplex_values = {0};
    b.multiplexor = a.key;
    b.multiplex_values = {0};
    const auto c = multiplexed(binary("c", 16, 8), a, 1);
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a, b, c})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAFieldThatMultiplexesItself) {
    auto a = binary("a", 0, 8);
    a.multiplexor = a.key;
    a.multiplex_values = {0};
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAFloatMultiplexor) {
    auto mux = binary("mux", 0, 32);
    mux.float_ = true;
    const auto a = multiplexed(binary("a", 32, 8), mux, 0);
    ASSERT_OCCURRED_AS_P(Plan::compile(message({mux, a})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAMultiplexedFieldWithNoValues) {
    const auto mux = binary("mux", 0, 8);
    auto a = binary("a", 8, 8);
    a.multiplexor = mux.key;
    ASSERT_OCCURRED_AS_P(Plan::compile(message({mux, a})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAMultiplexedTextField) {
    const auto mux = delimited("mux", 0);
    auto a = delimited("a", 1);
    a.multiplexor = mux.key;
    a.multiplex_values = {1};
    ASSERT_OCCURRED_AS_P(Plan::compile(text({mux, a})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAnEmptyTag) {
    ASSERT_OCCURRED_AS_P(Plan::compile(text({tagged("a", "")})), LAYOUT_ERROR);
}

TEST(PlanCompile, RejectsAnEmptyDelimiter) {
    auto m = text({delimited("a", 0)});
    m.delimiter = "";
    ASSERT_OCCURRED_AS_P(Plan::compile(m), LAYOUT_ERROR);
}

TEST(PlanCompile, SpansTheFieldsWhenTheMessageHasNoLength) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 20, 12)})));
    EXPECT_EQ(plan.length(), 4);
}

TEST(PlanDecode, FillsOnlyTheSelectedFieldsInKeyOrder) {
    const auto a = binary("a", 0, 8);
    const auto b = binary("b", 8, 8);
    const auto c = binary("c", 16, 8);
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a, b, c}), {c.key, a.key}));
    ASSERT_EQ(plan.size(), 2);
    EXPECT_EQ(plan.key(0), c.key);
    auto values = plan.values();
    const std::vector<std::uint8_t> payload = {1, 2, 3};
    ASSERT_NIL(plan.decode(payload, values));
    EXPECT_EQ(values.get(0), 3);
    EXPECT_EQ(values.get(1), 1);
}

TEST(PlanDecode, AppliesScaleAndOffset) {
    auto a = binary("a", 0, 16, true);
    a.scale = 0.5;
    a.offset = -10;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    const std::vector<std::uint8_t> payload = {0xFE, 0xFF};
    ASSERT_NIL(plan.decode(payload, values));
    EXPECT_EQ(values.get(0), -11);
}

TEST(PlanDecode, ResolvesAMultiplexorThatIsNotSelected) {
    const auto mux = binary("mux", 0, 8);
    const auto a = multiplexed(binary("a", 8, 8), mux, 2);
    const auto plan = ASSERT_NIL_P(Plan::compile(message({mux, a}), {a.key}));
    auto values = plan.values();
    const std::vector<std::uint8_t> selected = {2, 7};
    ASSERT_NIL(plan.decode(selected, values));
    ASSERT_TRUE(values.present(0));
    EXPECT_EQ(values.get(0), 7);
    const std::vector<std::uint8_t> other = {3, 7};
    ASSERT_NIL(plan.decode(other, values));
    EXPECT_FALSE(values.present(0));
}

TEST(PlanDecode, SelectsAFieldByAnyOfItsMultiplexValues) {
    const auto mux = binary("mux", 0, 4, true);
    auto a = binary("a", 8, 8);
    a.multiplexor = mux.key;
    a.multiplex_values = {-1, 5};
    const auto plan = ASSERT_NIL_P(Plan::compile(message({mux, a})));
    auto values = plan.values();
    for (const std::uint8_t m: {0x0F, 0x05}) {
        const std::vector<std::uint8_t> payload = {m, 9};
        ASSERT_NIL(plan.decode(payload, values));
        EXPECT_TRUE(values.present(1)) << int(m);
    }
    const std::vector<std::uint8_t> payload = {0x04, 9};
    ASSERT_NIL(plan.decode(payload, values));
    EXPECT_FALSE(values.present(1));
}

TEST(PlanDecode, RequiresEveryMultiplexorInAChain) {
    const auto outer = binary("outer", 0, 8);
    auto inner = multiplexed(binary("inner", 8, 8), outer, 1);
    const auto a = multiplexed(binary("a", 16, 8), inner, 2);
    const auto plan = ASSERT_NIL_P(Plan::compile(message({outer, inner, a}), {a.key}));
    auto values = plan.values();
    const std::vector<std::uint8_t> both = {1, 2, 42};
    ASSERT_NIL(plan.decode(both, values));
    EXPECT_TRUE(values.present(0));
    const std::vector<std::uint8_t> inner_only = {0, 2, 42};
    ASSERT_NIL(plan.decode(inner_only, values));
    EXPECT_FALSE(values.present(0));
}

TEST(PlanDecode, ReturnsShortPayloadForATruncatedField) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 8, 16)})));
    auto values = plan.values();
    const std::vector<std::uint8_t> payload = {1, 2};
    ASSERT_OCCURRED_AS(plan.decode(payload, values), SHORT_PAYLOAD_ERROR);
}

TEST(PlanDecode, IgnoresTheBytesOfAnAbsentField) {
    const auto mux = binary("mux", 0, 8);
    const auto a = multiplexed(binary("a", 32, 8), mux, 1);
    const auto plan = ASSERT_NIL_P(Plan::compile(message({mux, a})));
    auto values = plan.values();
    const std::vector<std::uint8_t> payload = {0};
    ASSERT_NIL(plan.decode(payload, values));
    EXPECT_TRUE(values.present(0));
    EXPECT_FALSE(values.present(1));
}

TEST(PlanDecode, KeepsUnsigned64BitIntegersExact) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 64)})));
    auto values = plan.values();
    const std::vector<std::uint8_t> payload = {1, 0, 0, 0, 0, 0, 0, 0xFF};
    ASSERT_NIL(plan.decode(payload, values));
    x::telem::Series s(x::telem::UINT64_T, 1);
    ASSERT_EQ(values.write(0, s), 1);
    EXPECT_EQ(s.at<std::uint64_t>(0), 0xFF00000000000001);
}

TEST(PlanDecode, ReinterpretsFloatBits) {
    auto a = binary("a", 0, 32);
    a.float_ = true;
    a.scale = 2;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    const auto bits = std::bit_cast<std::uint32_t>(1.5f);
    const std::vector<std::uint8_t> payload = {
        static_cast<std::uint8_t>(bits),
        static_cast<std::uint8_t>(bits >> 8),
        static_cast<std::uint8_t>(bits >> 16),
        static_cast<std::uint8_t>(bits >> 24),
    };
    ASSERT_NIL(plan.decode(payload, values));
    EXPECT_EQ(values.get(0), 3);
}

TEST(PlanEncode, RoundsToTheNearestRawValue) {
    auto a = binary("a", 0, 8);
    a.scale = 0.1;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    values.set(0, 1.26);
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload, std::vector<std::uint8_t>{13});
    values.set(0, 1.24);
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload, std::vector<std::uint8_t>{12});
}

TEST(PlanEncode, ClampsToTheRawRangeOfAnUnsignedField) {
    auto a = binary("a", 0, 8);
    a.offset = 0.5;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    std::vector<std::uint8_t> payload;
    values.set(0, 300.0);
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload[0], 255);
    values.set(0, -5.0);
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload[0], 0);
}

TEST(PlanEncode, ClampsToTheRawRangeOfASignedField) {
    auto a = binary("a", 0, 8, true);
    a.scale = 2;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    std::vector<std::uint8_t> payload;
    values.set(0, 1000.0);
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload[0], 0x7F);
    values.set(0, -1000.0);
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload[0], 0x80);
    values.set(0, std::numeric_limits<double>::infinity());
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload[0], 0x7F);
}

TEST(PlanEncode, ClampsExactIntegers) {
    const auto u = binary("u", 0, 8);
    const auto s = binary("s", 8, 16, true);
    const auto plan = ASSERT_NIL_P(Plan::compile(message({u, s})));
    auto values = plan.values();
    values.set(0, std::int64_t{-1});
    values.set(1, std::numeric_limits<std::uint64_t>::max());
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0x00, 0xFF, 0x7F}));
}

TEST(PlanEncode, KeepsA64BitIntegerExact) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 64, true)})));
    auto values = plan.values();
    values.set(0, std::numeric_limits<std::int64_t>::min() + 1);
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{1, 0, 0, 0, 0, 0, 0, 0x80}));
}

TEST(PlanEncode, ReturnsAnErrorOnNaN) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 8)})));
    auto values = plan.values();
    values.set(0, std::nan(""));
    std::vector<std::uint8_t> payload;
    ASSERT_OCCURRED_AS(plan.encode(values, payload), ENCODE_ERROR);
}

TEST(PlanEncode, LeavesTheOtherBitsOfThePayload) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 4, 8)})));
    auto values = plan.values();
    values.set(0, 0.0);
    std::vector<std::uint8_t> payload = {0xFF, 0xFF, 0xFF};
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0x0F, 0xF0, 0xFF}));
}

TEST(PlanEncode, SkipsAbsentFields) {
    const auto plan = ASSERT_NIL_P(
        Plan::compile(message({binary("a", 0, 8), binary("b", 8, 8)}))
    );
    auto values = plan.values();
    values.set(1, 5.0);
    std::vector<std::uint8_t> payload = {0xAA, 0xAA};
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0xAA, 5}));
}

TEST(PlanEncode, GrowsThePayloadToTheMessageLength) {
    auto m = message({binary("a", 0, 8)});
    m.length = 8;
    const auto plan = ASSERT_NIL_P(Plan::compile(m));
    auto values = plan.values();
    values.set(0, 1.0);
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{1, 0, 0, 0, 0, 0, 0, 0}));
}

TEST(PlanEncode, WritesOnlyTheFieldsTheMultiplexorSelects) {
    const auto mux = binary("mux", 0, 8);
    const auto a = multiplexed(binary("a", 8, 8), mux, 0);
    const auto b = multiplexed(binary("b", 8, 8), mux, 1);
    // The multiplexor is last in key order but must be written first.
    const auto plan = ASSERT_NIL_P(
        Plan::compile(message({mux, a, b}), {a.key, b.key, mux.key})
    );
    auto values = plan.values();
    values.set(0, 10.0);
    values.set(1, 20.0);
    values.set(2, 1.0);
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{1, 20}));
}

TEST(PlanEncode, ReadsAnUnselectedMultiplexorFromTheInitialPayload) {
    const auto mux = binary("mux", 0, 8);
    const auto a = multiplexed(binary("a", 8, 8), mux, 3);
    const auto plan = ASSERT_NIL_P(Plan::compile(message({mux, a}), {a.key}));
    auto values = plan.values();
    values.set(0, 9.0);
    std::vector<std::uint8_t> selected = {3, 0};
    ASSERT_NIL(plan.encode(values, selected));
    EXPECT_EQ(selected[1], 9);
    std::vector<std::uint8_t> other = {4, 0};
    ASSERT_NIL(plan.encode(values, other));
    EXPECT_EQ(other[1], 0);
}

TEST(PlanEncode, WritesFloatBits) {
    auto a = binary("a", 0, 64);
    a.float_ = true;
    a.offset = 1;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    values.set(0, 3.5);
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(plan.encode(values, payload));
    std::uint64_t bits = 0;
    for (int i = 7; i >= 0; i--)
        bits = (bits << 8) | payload[i];
    EXPECT_EQ(std::bit_cast<double>(bits), 2.5);
}

TEST(PlanEncode, WritesInPlaceIntoAWindowOfALargerBuffer) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 4, 8)})));
    auto values = plan.values();
    values.set(0, 0.0);
    std::vector<std::uint8_t> buffer = {0xAA, 0xFF, 0xFF, 0xAA};
    ASSERT_NIL(plan.encode(values, std::span(buffer).subspan(1, 2)));
    EXPECT_EQ(buffer, (std::vector<std::uint8_t>{0xAA, 0x0F, 0xF0, 0xAA}));
}

TEST(PlanEncode, RejectsAnInPlacePayloadShorterThanThePlan) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 4, 8)})));
    auto values = plan.values();
    values.set(0, 1.0);
    std::vector<std::uint8_t> buffer = {0, 0};
    ASSERT_OCCURRED_AS(plan.encode(values, std::span(buffer).first(1)), ENCODE_ERROR);
    EXPECT_EQ(buffer, (std::vector<std::uint8_t>{0, 0}));
}

TEST(PlanEncode, RejectsEncodingTextInPlace) {
    const auto plan = ASSERT_NIL_P(Plan::compile(text({delimited("a", 0)})));
    auto values = plan.values();
    values.set(0, 1.0);
    std::vector<std::uint8_t> buffer(4);
    ASSERT_OCCURRED_AS(plan.encode(values, std::span(buffer)), ENCODE_ERROR);
}

TEST(PlanDecode, ReadsAWordSwappedFloat) {
    auto a = binary("a", 7, 32);
    a.float_ = true;
    a.byte_order = BYTE_ORDER_BIG_ENDIAN_WORD_SWAPPED;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    const auto raw = std::bit_cast<std::uint32_t>(3.14159f);
    const std::vector<std::uint8_t> payload = {
        static_cast<std::uint8_t>(raw >> 8),
        static_cast<std::uint8_t>(raw),
        static_cast<std::uint8_t>(raw >> 24),
        static_cast<std::uint8_t>(raw >> 16),
    };
    auto values = plan.values();
    ASSERT_NIL(plan.decode(payload, values));
    EXPECT_EQ(values.get(0), static_cast<double>(3.14159f));
    std::vector<std::uint8_t> encoded;
    ASSERT_NIL(plan.encode(values, encoded));
    EXPECT_EQ(encoded, payload);
}

TEST(Values, SetsASampleValueExactly) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 32)})));
    auto values = plan.values();
    ASSERT_NIL(values.set(0, x::telem::SampleValue(std::uint32_t{4000000000})));
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0x00, 0x28, 0x6B, 0xEE}));
}

TEST(Values, RejectsAStringSample) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 8)})));
    auto values = plan.values();
    ASSERT_OCCURRED_AS(
        values.set(0, x::telem::SampleValue(std::string("x"))),
        ENCODE_ERROR
    );
}

TEST(Values, WritesNothingForAnAbsentSlot) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 8)})));
    const auto values = plan.values();
    x::telem::Series s(x::telem::FLOAT64_T, 1);
    EXPECT_EQ(values.write(0, s), 0);
    EXPECT_EQ(s.size(), 0);
}

TEST(Values, CastsToTheSeriesDataType) {
    auto a = binary("a", 0, 8);
    a.scale = 0.5;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    const std::vector<std::uint8_t> payload = {5};
    ASSERT_NIL(plan.decode(payload, values));
    x::telem::Series s(x::telem::FLOAT32_T, 1);
    ASSERT_EQ(values.write(0, s), 1);
    EXPECT_EQ(s.at<float>(0), 2.5f);
}

TEST(PlanText, DecodesDelimitedItemsByPosition) {
    auto b = delimited("b", 2);
    b.scale = 2;
    b.offset = 1;
    const auto plan = ASSERT_NIL_P(Plan::compile(text({delimited("a", 0), b})));
    auto values = plan.values();
    ASSERT_NIL(plan.decode(bytes("1.5,abc,-3e2"), values));
    EXPECT_EQ(values.get(0), 1.5);
    EXPECT_EQ(values.get(1), -599);
    EXPECT_EQ(values.invalid(), 0);
}

TEST(PlanText, DecodesTwoFieldsAtOnePosition) {
    const auto plan = ASSERT_NIL_P(
        Plan::compile(text({delimited("a", 1), delimited("b", 1)}))
    );
    auto values = plan.values();
    ASSERT_NIL(plan.decode(bytes("0,7"), values));
    EXPECT_EQ(values.get(0), 7);
    EXPECT_EQ(values.get(1), 7);
}

TEST(PlanText, DecodesTaggedItems) {
    const auto plan = ASSERT_NIL_P(
        Plan::compile(text({tagged("p", "P="), tagged("t", "T=")}))
    );
    auto values = plan.values();
    ASSERT_NIL(plan.decode(bytes("T=23.4, P=101.3"), values));
    EXPECT_EQ(values.get(0), 101.3);
    EXPECT_EQ(values.get(1), 23.4);
}

TEST(PlanText, MatchesATagOnlyAtTheStartOfAnItem) {
    const auto plan = ASSERT_NIL_P(Plan::compile(text({tagged("p", "P=")})));
    auto values = plan.values();
    ASSERT_NIL(plan.decode(bytes("XP=1,P=2"), values));
    EXPECT_EQ(values.get(0), 2);
}

TEST(PlanText, MarksAnUnparseableItemAbsentAndCountsIt) {
    const auto plan = ASSERT_NIL_P(
        Plan::compile(text({delimited("a", 0), delimited("b", 1)}))
    );
    auto values = plan.values();
    ASSERT_NIL(plan.decode(bytes("1.0,12abc"), values));
    EXPECT_TRUE(values.present(0));
    EXPECT_FALSE(values.present(1));
    EXPECT_EQ(values.invalid(), 1);
}

TEST(PlanText, CountsMissingItemsAndTags) {
    const auto plan = ASSERT_NIL_P(
        Plan::compile(text({delimited("a", 5), tagged("b", "B=")}))
    );
    auto values = plan.values();
    ASSERT_NIL(plan.decode(bytes("1,2"), values));
    EXPECT_FALSE(values.present(0));
    EXPECT_FALSE(values.present(1));
    EXPECT_EQ(values.invalid(), 2);
}

TEST(PlanText, ParsesSCPINumbersWithASignAndTrailingWhitespace) {
    const auto plan = ASSERT_NIL_P(Plan::compile(text({delimited("a", 0)})));
    auto values = plan.values();
    ASSERT_NIL(plan.decode(bytes(" +1.234E+00\r"), values));
    EXPECT_EQ(values.get(0), 1.234);
}

TEST(PlanText, SplitsOnAMultiCharacterDelimiter) {
    auto m = text({delimited("a", 1)});
    m.delimiter = ";;";
    const auto plan = ASSERT_NIL_P(Plan::compile(m));
    auto values = plan.values();
    ASSERT_NIL(plan.decode(bytes("1;;2;3"), values));
    EXPECT_FALSE(values.present(0));
    ASSERT_NIL(plan.decode(bytes("1;;2"), values));
    EXPECT_EQ(values.get(0), 2);
}

TEST(PlanText, EncodesItemsInPositionOrderThenTags) {
    auto c = delimited("c", 2);
    c.scale = 0.5;
    const auto plan = ASSERT_NIL_P(
        Plan::compile(text({c, tagged("p", "P="), delimited("a", 0)}))
    );
    auto values = plan.values();
    values.set(0, 3.5);
    values.set(1, std::int64_t{-4});
    values.set(2, 0.25);
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(str(payload), "0.25,,7,P=-4");
}

TEST(PlanText, SkipsAbsentTaggedFieldsOnEncode) {
    const auto plan = ASSERT_NIL_P(
        Plan::compile(text({tagged("a", "A="), tagged("b", "B=")}))
    );
    auto values = plan.values();
    values.set(1, std::uint64_t{18446744073709551615u});
    std::vector<std::uint8_t> payload = {'x'};
    ASSERT_NIL(plan.encode(values, payload));
    EXPECT_EQ(str(payload), "B=18446744073709551615");
}
}
