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
#include <optional>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/codec/plan.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

icd::BinaryField binary(
    const std::string &name,
    const std::uint16_t start_bit,
    const std::uint8_t bit_length,
    const bool signed_ = false
) {
    icd::BinaryField f;
    f.key = x::uuid::create();
    f.name = name;
    f.start_bit = start_bit;
    f.bit_length = bit_length;
    f.signed_ = signed_;
    return f;
}

icd::BinaryField multiplexed(
    icd::BinaryField f,
    const icd::BinaryField &mux,
    const std::int32_t value
) {
    f.multiplexor = mux.key;
    f.multiplex_values = {value};
    return f;
}

icd::DelimitedTextField
delimited(const std::string &name, const std::uint32_t pos) {
    icd::DelimitedTextField f;
    f.key = x::uuid::create();
    f.name = name;
    f.position = pos;
    return f;
}

icd::TaggedTextField tagged(const std::string &name, const std::string &tag) {
    icd::TaggedTextField f;
    f.key = x::uuid::create();
    f.name = name;
    f.tag = tag;
    return f;
}

icd::MessageEntry message(
    std::vector<icd::BinaryField> fields,
    const std::optional<std::uint16_t> length = std::nullopt
) {
    icd::MessageEntry m;
    m.key = x::uuid::create();
    m.name = "message";
    m.payload = icd::BinaryPayload{.length = length, .fields = std::move(fields)};
    return m;
}

icd::MessageEntry
text(std::vector<icd::TextField> fields, const std::string &delimiter = ",") {
    icd::MessageEntry m;
    m.key = x::uuid::create();
    m.name = "message";
    m.payload = icd::TextPayload{
        .delimiter = delimiter,
        .fields = std::move(fields),
    };
    return m;
}

std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

std::string str(const std::vector<std::uint8_t> &b) {
    return {b.begin(), b.end()};
}
}

TEST(PlanCompile, RejectsAKeyThatIsNotInTheMessage) {
    const auto m = message({binary("a", 0, 8)});
    ASSERT_OCCURRED_AS_P(Plan::compile(m, {x::uuid::create()}), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsADuplicateKey) {
    const auto a = binary("a", 0, 8);
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a}), {a.key, a.key}), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsAZeroScale) {
    auto a = binary("a", 0, 8);
    a.scale = 0;
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a})), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsAFloatThatIsNot32Or64Bits) {
    auto a = binary("a", 0, 16);
    a.float_ = true;
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a})), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsFieldsBeyondTheMessageLength) {
    ASSERT_OCCURRED_AS_P(
        Plan::compile(message({binary("a", 56, 16)}, 8)),
        CONFIG_ERROR
    );
}

TEST(PlanCompile, RejectsAMultiplexorCycle) {
    auto a = binary("a", 0, 8);
    auto b = binary("b", 8, 8);
    a.multiplexor = b.key;
    a.multiplex_values = {0};
    b.multiplexor = a.key;
    b.multiplex_values = {0};
    const auto c = multiplexed(binary("c", 16, 8), a, 1);
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a, b, c})), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsAFieldThatMultiplexesItself) {
    auto a = binary("a", 0, 8);
    a.multiplexor = a.key;
    a.multiplex_values = {0};
    ASSERT_OCCURRED_AS_P(Plan::compile(message({a})), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsAFloatMultiplexor) {
    auto mux = binary("mux", 0, 32);
    mux.float_ = true;
    const auto a = multiplexed(binary("a", 32, 8), mux, 0);
    ASSERT_OCCURRED_AS_P(Plan::compile(message({mux, a})), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsAMultiplexedFieldWithNoValues) {
    const auto mux = binary("mux", 0, 8);
    auto a = binary("a", 8, 8);
    a.multiplexor = mux.key;
    ASSERT_OCCURRED_AS_P(Plan::compile(message({mux, a})), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsAMultiplexedTextField) {
    const auto mux = delimited("mux", 0);
    auto a = delimited("a", 1);
    a.multiplexor = mux.key;
    a.multiplex_values = {1};
    ASSERT_OCCURRED_AS_P(Plan::compile(text({mux, a})), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsAnEmptyTag) {
    ASSERT_OCCURRED_AS_P(Plan::compile(text({tagged("a", "")})), CONFIG_ERROR);
}

TEST(PlanCompile, RejectsAnEmptyDelimiter) {
    ASSERT_OCCURRED_AS_P(Plan::compile(text({delimited("a", 0)}, "")), CONFIG_ERROR);
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
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(1.26)));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    EXPECT_EQ(payload, std::vector<std::uint8_t>{13});
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(1.24)));
    plan.encode(values, payload);
    EXPECT_EQ(payload, std::vector<std::uint8_t>{12});
}

TEST(PlanEncode, WritesTheEndsOfTheRawRange) {
    auto u = binary("u", 0, 8);
    u.offset = 0.5;
    auto s = binary("s", 8, 8, true);
    s.scale = 2;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({u, s})));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(255.5)));
    ASSERT_NIL(plan.set(values, 1, x::telem::SampleValue(-256.0)));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0xFF, 0x80}));
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(0.5)));
    ASSERT_NIL(plan.set(values, 1, x::telem::SampleValue(254.0)));
    plan.encode(values, payload);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0x00, 0x7F}));
}

TEST(PlanEncode, WritesTheEndsOfExactIntegerFields) {
    const auto u = binary("u", 0, 8);
    const auto s = binary("s", 8, 16, true);
    const auto plan = ASSERT_NIL_P(Plan::compile(message({u, s})));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(std::uint8_t{255})));
    ASSERT_NIL(plan.set(values, 1, x::telem::SampleValue(std::int64_t{-32768})));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0xFF, 0x00, 0x80}));
}

TEST(PlanEncode, KeepsA64BitIntegerExact) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 64, true)})));
    auto values = plan.values();
    ASSERT_NIL(plan.set(
        values,
        0,
        x::telem::SampleValue(std::numeric_limits<std::int64_t>::min() + 1)
    ));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{1, 0, 0, 0, 0, 0, 0, 0x80}));
}

TEST(PlanEncode, WritesTheLargestDoubleBelowTheEndOfAn64BitField) {
    auto a = binary("a", 0, 64);
    a.scale = 2;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    const auto below = std::nextafter(0x1p65, 0.0);
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(below)));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    EXPECT_EQ(
        payload,
        (std::vector<std::uint8_t>{0, 0xF8, 255, 255, 255, 255, 255, 255})
    );
}

TEST(PlanEncode, LeavesTheOtherBitsOfThePayload) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 4, 8)})));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(0.0)));
    std::vector<std::uint8_t> payload = {0xFF, 0xFF, 0xFF};
    plan.encode(values, payload);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0x0F, 0xF0, 0xFF}));
}

TEST(PlanEncode, SkipsAbsentFields) {
    const auto plan = ASSERT_NIL_P(
        Plan::compile(message({binary("a", 0, 8), binary("b", 8, 8)}))
    );
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 1, x::telem::SampleValue(5.0)));
    std::vector<std::uint8_t> payload = {0xAA, 0xAA};
    plan.encode(values, payload);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0xAA, 5}));
}

TEST(PlanEncode, GrowsThePayloadToTheMessageLength) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 8)}, 8)));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(1.0)));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
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
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(10.0)));
    ASSERT_NIL(plan.set(values, 1, x::telem::SampleValue(20.0)));
    ASSERT_NIL(plan.set(values, 2, x::telem::SampleValue(1.0)));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{1, 20}));
}

TEST(PlanEncode, ReadsAnUnselectedMultiplexorFromTheInitialPayload) {
    const auto mux = binary("mux", 0, 8);
    const auto a = multiplexed(binary("a", 8, 8), mux, 3);
    const auto plan = ASSERT_NIL_P(Plan::compile(message({mux, a}), {a.key}));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(9.0)));
    std::vector<std::uint8_t> selected = {3, 0};
    plan.encode(values, selected);
    EXPECT_EQ(selected[1], 9);
    std::vector<std::uint8_t> other = {4, 0};
    plan.encode(values, other);
    EXPECT_EQ(other[1], 0);
}

TEST(PlanEncode, WritesFloatBits) {
    auto a = binary("a", 0, 64);
    a.float_ = true;
    a.offset = 1;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(3.5)));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    std::uint64_t bits = 0;
    for (int i = 7; i >= 0; i--)
        bits = (bits << 8) | payload[i];
    EXPECT_EQ(std::bit_cast<double>(bits), 2.5);
}

TEST(PlanSet, SetsASampleValueExactly) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 32)})));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(std::uint32_t{4000000000})));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0x00, 0x28, 0x6B, 0xEE}));
}

TEST(PlanSet, RefusesAStringSample) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 8)})));
    auto values = plan.values();
    ASSERT_OCCURRED_AS(
        plan.set(values, 0, x::telem::SampleValue(std::string("x"))),
        ENCODE_ERROR
    );
}

TEST(PlanSet, RefusesAValueOutsideTheBitsOfAnUnsignedField) {
    auto a = binary("a", 0, 8);
    a.offset = 0.5;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    const auto err = plan.set(values, 0, x::telem::SampleValue(300.0));
    ASSERT_OCCURRED_AS(err, RANGE_ERROR);
    EXPECT_EQ(err.data, "300 is outside the range 0.5 to 255.5");
    EXPECT_FALSE(values.present(0));
    ASSERT_OCCURRED_AS(plan.set(values, 0, x::telem::SampleValue(-5.0)), RANGE_ERROR);
    ASSERT_OCCURRED_AS(plan.set(values, 0, x::telem::SampleValue(256.0)), RANGE_ERROR);
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(255.9)));
}

TEST(PlanSet, RefusesAValueOutsideTheBitsOfASignedFieldWithANegativeScale) {
    auto a = binary("a", 0, 8, true);
    a.scale = -2;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    const auto err = plan.set(values, 0, x::telem::SampleValue(257.0));
    ASSERT_OCCURRED_AS(err, RANGE_ERROR);
    EXPECT_EQ(err.data, "257 is outside the range -254 to 256");
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(256.0)));
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(-254.0)));
}

TEST(PlanSet, RefusesNaN) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 8)})));
    auto values = plan.values();
    const auto err = plan.set(values, 0, x::telem::SampleValue(std::nan("")));
    ASSERT_OCCURRED_AS(err, RANGE_ERROR);
    EXPECT_EQ(err.data, "NaN is outside the range 0 to 255");
}

TEST(PlanSet, NarrowsTheRangeByTheFieldMinAndMax) {
    auto a = binary("a", 0, 8);
    a.min = 10;
    a.max = 20.5;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    const auto low = plan.set(values, 0, x::telem::SampleValue(std::int64_t{9}));
    ASSERT_OCCURRED_AS(low, RANGE_ERROR);
    EXPECT_EQ(low.data, "9 is outside the range 10 to 20.5");
    ASSERT_OCCURRED_AS(plan.set(values, 0, x::telem::SampleValue(21.0)), RANGE_ERROR);
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(std::int64_t{10})));
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(20.5)));
}

TEST(PlanSet, KeepsTheLastAcceptedValue) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 8)})));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(5.0)));
    ASSERT_OCCURRED_AS(plan.set(values, 0, x::telem::SampleValue(300.0)), RANGE_ERROR);
    EXPECT_EQ(values.get(0), 5);
}

TEST(PlanSet, ChecksExactIntegersWithoutRounding) {
    const auto u = binary("u", 0, 8);
    const auto s = binary("s", 8, 16, true);
    const auto plan = ASSERT_NIL_P(Plan::compile(message({u, s})));
    auto values = plan.values();
    const auto negative = plan.set(values, 0, x::telem::SampleValue(std::int64_t{-1}));
    ASSERT_OCCURRED_AS(negative, RANGE_ERROR);
    EXPECT_EQ(negative.data, "-1 is outside the range 0 to 255");
    const auto huge = plan.set(
        values,
        1,
        x::telem::SampleValue(std::numeric_limits<std::uint64_t>::max())
    );
    ASSERT_OCCURRED_AS(huge, RANGE_ERROR);
    EXPECT_EQ(huge.data, "18446744073709551615 is outside the range -32768 to 32767");
}

TEST(PlanSet, RefusesTheFirstValueBeyondAn64BitField) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 64)})));
    auto values = plan.values();
    const auto err = plan.set(values, 0, x::telem::SampleValue(0x1p64));
    ASSERT_OCCURRED_AS(err, RANGE_ERROR);
    EXPECT_EQ(
        err.data,
        "18446744073709551616 is outside the range 0 to 18446744073709551615"
    );
    ASSERT_NIL(plan.set(
        values,
        0,
        x::telem::SampleValue(std::numeric_limits<std::uint64_t>::max())
    ));
}

TEST(PlanSet, BoundsAFloat32FieldByTheLargestFloat) {
    auto a = binary("a", 0, 32);
    a.float_ = true;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    ASSERT_OCCURRED_AS(plan.set(values, 0, x::telem::SampleValue(1e39)), RANGE_ERROR);
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(1e38)));
}

TEST(PlanSet, EncodesAnIntegerSampleOnAFloatFieldAsAFloat) {
    auto a = binary("a", 0, 32);
    a.float_ = true;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(std::int64_t{3})));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    std::uint32_t bits = 0;
    for (int i = 3; i >= 0; i--)
        bits = (bits << 8) | payload[i];
    EXPECT_EQ(std::bit_cast<float>(bits), 3.0f);
}

TEST(PlanSet, BoundsATextFieldByItsMinAndMaxAlone) {
    auto a = delimited("a", 0);
    a.max = 10;
    const auto plan = ASSERT_NIL_P(Plan::compile(text({a, delimited("b", 1)})));
    auto values = plan.values();
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(-1e300)));
    ASSERT_OCCURRED_AS(plan.set(values, 0, x::telem::SampleValue(11.0)), RANGE_ERROR);
    const auto inf = plan.set(
        values,
        1,
        x::telem::SampleValue(std::numeric_limits<double>::infinity())
    );
    ASSERT_OCCURRED_AS(inf, RANGE_ERROR);
    EXPECT_EQ(
        inf.data,
        "inf is outside the range -1.7976931348623157e+308 to "
        "1.7976931348623157e+308"
    );
}

TEST(PlanSetRaw, SetsThePhysicalValueOfARawValue) {
    auto a = binary("a", 0, 8);
    a.scale = 2;
    a.offset = 100;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({a})));
    auto values = plan.values();
    plan.set_raw(values, 0, 0);
    EXPECT_EQ(values.get(0), 100);
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    EXPECT_EQ(payload, std::vector<std::uint8_t>{0});
    plan.set_raw(values, 0, 7);
    plan.encode(values, payload);
    EXPECT_EQ(payload, std::vector<std::uint8_t>{7});
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

TEST(Values, FitsAnIntegerTypeOnlyWhenTheTruncatedValueIsInRange) {
    const auto plan = ASSERT_NIL_P(Plan::compile(text({delimited("a", 0)})));
    auto values = plan.values();
    const auto fits = [&](const std::string &line, const x::telem::DataType &type) {
        EXPECT_FALSE(plan.decode(bytes(line), values));
        return values.fits(0, type);
    };
    EXPECT_TRUE(fits("127.9", x::telem::INT8_T));
    EXPECT_FALSE(fits("128", x::telem::INT8_T));
    EXPECT_TRUE(fits("-128.9", x::telem::INT8_T));
    EXPECT_FALSE(fits("-129", x::telem::INT8_T));
    EXPECT_TRUE(fits("-0.5", x::telem::UINT8_T));
    EXPECT_FALSE(fits("-1", x::telem::UINT8_T));
    EXPECT_FALSE(fits("256", x::telem::UINT8_T));
    EXPECT_FALSE(fits("nan", x::telem::INT32_T));
    EXPECT_FALSE(fits("inf", x::telem::INT64_T));
    EXPECT_FALSE(fits("9223372036854775808", x::telem::INT64_T));
    EXPECT_TRUE(fits("-9223372036854775808", x::telem::INT64_T));
    EXPECT_FALSE(fits("18446744073709551616", x::telem::UINT64_T));
}

TEST(Values, FitsAFloatTypeUnlessAFiniteValueOverflowsIt) {
    const auto plan = ASSERT_NIL_P(Plan::compile(text({delimited("a", 0)})));
    auto values = plan.values();
    const auto fits = [&](const std::string &line, const x::telem::DataType &type) {
        EXPECT_FALSE(plan.decode(bytes(line), values));
        return values.fits(0, type);
    };
    EXPECT_TRUE(fits("nan", x::telem::FLOAT32_T));
    EXPECT_TRUE(fits("-inf", x::telem::FLOAT32_T));
    EXPECT_TRUE(fits("3e38", x::telem::FLOAT32_T));
    EXPECT_FALSE(fits("4e38", x::telem::FLOAT32_T));
    EXPECT_TRUE(fits("1e308", x::telem::FLOAT64_T));
}

TEST(Values, FitsAnExactIntegerByItsBits) {
    auto u = binary("u", 0, 64);
    auto i = binary("i", 0, 64);
    i.signed_ = true;
    const auto plan = ASSERT_NIL_P(Plan::compile(message({u, i})));
    auto values = plan.values();
    const std::vector<std::uint8_t> payload(8, 0xFF);
    ASSERT_NIL(plan.decode(payload, values));
    EXPECT_FALSE(values.fits(0, x::telem::INT64_T));
    EXPECT_TRUE(values.fits(0, x::telem::UINT64_T));
    EXPECT_TRUE(values.fits(0, x::telem::FLOAT32_T));
    EXPECT_TRUE(values.fits(1, x::telem::INT8_T));
    EXPECT_FALSE(values.fits(1, x::telem::UINT8_T));
}

TEST(Values, DoesNotFitATypeThatHoldsNoNumber) {
    const auto plan = ASSERT_NIL_P(Plan::compile(message({binary("a", 0, 8)})));
    auto values = plan.values();
    ASSERT_NIL(plan.decode(std::vector<std::uint8_t>{1}, values));
    EXPECT_FALSE(values.fits(0, x::telem::TIMESTAMP_T));
    EXPECT_FALSE(values.fits(0, x::telem::STRING_T));
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
    const auto plan = ASSERT_NIL_P(Plan::compile(text({delimited("a", 1)}, ";;")));
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
    ASSERT_NIL(plan.set(values, 0, x::telem::SampleValue(3.5)));
    ASSERT_NIL(plan.set(values, 1, x::telem::SampleValue(std::int64_t{-4})));
    ASSERT_NIL(plan.set(values, 2, x::telem::SampleValue(0.25)));
    std::vector<std::uint8_t> payload;
    plan.encode(values, payload);
    EXPECT_EQ(str(payload), "0.25,,7,P=-4");
}

TEST(PlanText, SkipsAbsentTaggedFieldsOnEncode) {
    const auto plan = ASSERT_NIL_P(
        Plan::compile(text({tagged("a", "A="), tagged("b", "B=")}))
    );
    auto values = plan.values();
    ASSERT_NIL(
        plan.set(values, 1, x::telem::SampleValue(std::uint64_t{18446744073709551615u}))
    );
    std::vector<std::uint8_t> payload = {'x'};
    plan.encode(values, payload);
    EXPECT_EQ(str(payload), "B=18446744073709551615");
}
}
