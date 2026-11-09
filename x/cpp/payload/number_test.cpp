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
    const x::json::Parser parser(x::json::json{{"fill", "-Infinity"}});
    const auto read = FloatRead::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(read.fill.has_value());
    EXPECT_EQ(read.fill.value().value, -INF);
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
    EXPECT_EQ(FloatRead{.fill = Float{2.5}}.to_json()["fill"], 2.5);
    EXPECT_EQ(FloatRead{.fill = Float{INF}}.to_json()["fill"], "Infinity");
    EXPECT_EQ(FloatRead{.fill = Float{-INF}}.to_json()["fill"], "-Infinity");
    const auto nan = std::numeric_limits<double>::quiet_NaN();
    EXPECT_EQ(FloatRead{.fill = Float{nan}}.to_json()["fill"], "NaN");
}

/// @brief it should parse an integer from a string of decimal digits or a number.
TEST(PayloadInteger, Parse) {
    const x::json::Parser parser(
        x::json::json{{"label", "LOW"}, {"value", "-9223372036854775808"}}
    );
    const auto entry = IntegerEntry::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(entry.value.value, INT64_MIN_VALUE);
    const x::json::Parser number_parser(x::json::json{{"fill", -12}});
    const auto read = IntegerRead::parse(number_parser);
    ASSERT_NIL(number_parser.error());
    ASSERT_TRUE(read.fill.has_value());
    EXPECT_EQ(read.fill.value().value, -12);
}

/// @brief it should reject an integer that is out of range or not decimal digits.
TEST(PayloadInteger, ParseInvalid) {
    for (const auto &fill: std::vector<x::json::json>{
             "9223372036854775808",
             "+1",
             "1.5",
             " 1",
             "",
             9223372036854775808ULL,
             true,
         }) {
        SCOPED_TRACE(fill.dump());
        const x::json::Parser parser(x::json::json{{"fill", fill}});
        IntegerRead::parse(parser);
        ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    }
}

/// @brief it should write every digit of an integer as a string.
TEST(PayloadInteger, ToJSON) {
    const IntegerRead read{.fill = Integer{INT64_MIN_VALUE}};
    EXPECT_EQ(read.to_json()["fill"], "-9223372036854775808");
    const IntegerEntry entry{.label = "ON", .value = Integer{1}};
    EXPECT_EQ(entry.to_json()["value"], "1");
}

/// @brief it should parse an unsigned integer from a string of decimal digits.
TEST(PayloadUnsigned, Parse) {
    const x::json::Parser parser(x::json::json{{"fill", "18446744073709551615"}});
    const auto read = UnsignedRead::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(read.fill.has_value());
    EXPECT_EQ(read.fill.value().value, UINT64_MAX_VALUE);
}

/// @brief it should parse an unsigned integer from a positive signed JSON number.
TEST(PayloadUnsigned, ParseSignedNumber) {
    const x::json::Parser parser(x::json::json{{"fill", 5}});
    const auto read = UnsignedRead::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(read.fill.has_value());
    EXPECT_EQ(read.fill.value().value, 5);
}

/// @brief it should reject a negative or out of range unsigned integer.
TEST(PayloadUnsigned, ParseInvalid) {
    for (const auto &fill:
         std::vector<x::json::json>{"-1", "18446744073709551616", -1, "+1"}) {
        SCOPED_TRACE(fill.dump());
        const x::json::Parser parser(x::json::json{{"fill", fill}});
        UnsignedRead::parse(parser);
        ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    }
}

/// @brief it should write every digit of an unsigned integer as a string.
TEST(PayloadUnsigned, ToJSON) {
    const UnsignedRead read{.fill = Unsigned{UINT64_MAX_VALUE}};
    EXPECT_EQ(read.to_json()["fill"], "18446744073709551615");
}
}
