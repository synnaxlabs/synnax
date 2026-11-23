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
#include <limits>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/payload/json.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace x::payload {
constexpr double INF = std::numeric_limits<double>::infinity();
constexpr std::int64_t INT64_MIN_VALUE = std::numeric_limits<std::int64_t>::min();
constexpr std::uint64_t UINT64_MAX_VALUE = std::numeric_limits<std::uint64_t>::max();

/// @brief it should parse a float from a JSON number.
TEST(PayloadFloat, ParseNumber) {
    const x::json::Parser parser(x::json::json{{"label", "ON"}, {"value", 1.5}});
    const auto entry = FloatEntry::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(entry.value.value, 1.5);
}

/// @brief it should parse a non-finite float from its text.
TEST(PayloadFloat, ParseNonFiniteText) {
    const x::json::Parser parser(
        x::json::json{{"label", "LOW"}, {"value", "-Infinity"}}
    );
    const auto entry = FloatEntry::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(entry.value.value, -INF);
}

/// @brief it should parse NaN from its text.
TEST(PayloadFloat, ParseNaNText) {
    const x::json::Parser parser(x::json::json{{"label", "OFF"}, {"value", "NaN"}});
    const auto entry = FloatEntry::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_TRUE(std::isnan(entry.value.value));
}

/// @brief it should reject text that names no float.
TEST(PayloadFloat, ParseInvalidText) {
    const x::json::Parser parser(x::json::json{{"label", "OFF"}, {"value", "inf"}});
    FloatEntry::parse(parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
}

/// @brief it should write a float as a number when finite and as text when not.
TEST(PayloadFloat, ToJSON) {
    EXPECT_EQ(FloatEntry{.value = Float{2.5}}.to_json()["value"], 2.5);
    EXPECT_EQ(FloatEntry{.value = Float{INF}}.to_json()["value"], "Infinity");
    EXPECT_EQ(FloatEntry{.value = Float{-INF}}.to_json()["value"], "-Infinity");
    const auto nan = std::numeric_limits<double>::quiet_NaN();
    EXPECT_EQ(FloatEntry{.value = Float{nan}}.to_json()["value"], "NaN");
}

/// @brief it should parse an integer from a string of decimal digits or a number.
TEST(PayloadInteger, Parse) {
    const x::json::Parser parser(
        x::json::json{{"label", "LOW"}, {"value", "-9223372036854775808"}}
    );
    const auto entry = IntegerEntry::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(entry.value.value, INT64_MIN_VALUE);
    const x::json::Parser number_parser(x::json::json{{"label", "L"}, {"value", -12}});
    const auto number = IntegerEntry::parse(number_parser);
    ASSERT_NIL(number_parser.error());
    EXPECT_EQ(number.value.value, -12);
}

/// @brief it should reject an integer that is out of range or not decimal digits.
TEST(PayloadInteger, ParseInvalid) {
    for (const auto &value: std::vector<x::json::json>{
             "9223372036854775808",
             "+1",
             "1.5",
             " 1",
             "",
             9223372036854775808ULL,
             true,
         }) {
        SCOPED_TRACE(value.dump());
        const x::json::Parser parser(x::json::json{{"label", "L"}, {"value", value}});
        IntegerEntry::parse(parser);
        ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
        EXPECT_EQ(
            parser.errors->at(0)["message"],
            "cannot parse an integer from " + value.dump()
        );
    }
}

/// @brief it should write every digit of an integer as a string.
TEST(PayloadInteger, ToJSON) {
    const IntegerEntry low{.label = "LOW", .value = Integer{INT64_MIN_VALUE}};
    EXPECT_EQ(low.to_json()["value"], "-9223372036854775808");
    const IntegerEntry on{.label = "ON", .value = Integer{1}};
    EXPECT_EQ(on.to_json()["value"], "1");
}

/// @brief it should parse an unsigned integer from a string of decimal digits.
TEST(PayloadUnsigned, Parse) {
    const x::json::Parser parser(
        x::json::json{{"label", "MAX"}, {"value", "18446744073709551615"}}
    );
    const auto entry = UnsignedEntry::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(entry.value.value, UINT64_MAX_VALUE);
}

/// @brief it should parse an unsigned integer from a positive signed JSON number.
TEST(PayloadUnsigned, ParseSignedNumber) {
    const x::json::Parser parser(x::json::json{{"label", "L"}, {"value", 5}});
    const auto entry = UnsignedEntry::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(entry.value.value, 5);
}

/// @brief it should reject a negative or out of range unsigned integer.
TEST(PayloadUnsigned, ParseInvalid) {
    for (const auto &value:
         std::vector<x::json::json>{"-1", "18446744073709551616", -1, "+1"}) {
        SCOPED_TRACE(value.dump());
        const x::json::Parser parser(x::json::json{{"label", "L"}, {"value", value}});
        UnsignedEntry::parse(parser);
        ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
        EXPECT_EQ(
            parser.errors->at(0)["message"],
            "cannot parse an unsigned integer from " + value.dump()
        );
    }
}

/// @brief it should write every digit of an unsigned integer as a string.
TEST(PayloadUnsigned, ToJSON) {
    const UnsignedEntry entry{.label = "MAX", .value = Unsigned{UINT64_MAX_VALUE}};
    EXPECT_EQ(entry.to_json()["value"], "18446744073709551615");
}

/// @brief it should keep a non-finite float through its protobuf form.
TEST(PayloadFloat, RoundTripsNonFiniteProto) {
    EXPECT_EQ(Float::from_proto(Float{-INF}.to_proto()).value, -INF);
    EXPECT_TRUE(
        std::isnan(
            Float::from_proto(
                Float{std::numeric_limits<double>::quiet_NaN()}.to_proto()
            )
                .value
        )
    );
}

/// @brief it should keep the least signed integer through its protobuf form.
TEST(PayloadInteger, RoundTripsLimitProto) {
    EXPECT_EQ(
        Integer::from_proto(Integer{INT64_MIN_VALUE}.to_proto()).value,
        INT64_MIN_VALUE
    );
}

/// @brief it should keep the greatest unsigned integer through its protobuf form.
TEST(PayloadUnsigned, RoundTripsLimitProto) {
    EXPECT_EQ(
        Unsigned::from_proto(Unsigned{UINT64_MAX_VALUE}.to_proto()).value,
        UINT64_MAX_VALUE
    );
}
}
