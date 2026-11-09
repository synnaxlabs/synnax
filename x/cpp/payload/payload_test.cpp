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

#include "gtest/gtest.h"

#include "x/cpp/payload/json.gen.h"

namespace x::payload {
constexpr double INF = std::numeric_limits<double>::infinity();

/// @brief it should parse a float from a JSON number.
TEST(PayloadFloat, ParseNumber) {
    x::json::Parser parser(x::json::json{{"label", "ON"}, {"value", 1.5}});
    const auto entry = FloatEntry::parse(parser);
    ASSERT_FALSE(parser.error());
    EXPECT_EQ(entry.value.value, 1.5);
}

/// @brief it should parse a non-finite float from its text.
TEST(PayloadFloat, ParseNonFiniteText) {
    x::json::Parser parser(x::json::json{{"fill", "-Infinity"}});
    const auto read = FloatRead::parse(parser);
    ASSERT_FALSE(parser.error());
    ASSERT_TRUE(read.fill.has_value());
    EXPECT_EQ(read.fill->value, -INF);
}

/// @brief it should parse NaN from its text.
TEST(PayloadFloat, ParseNaNText) {
    x::json::Parser parser(x::json::json{{"label", "OFF"}, {"value", "NaN"}});
    const auto entry = FloatEntry::parse(parser);
    ASSERT_FALSE(parser.error());
    EXPECT_TRUE(std::isnan(entry.value.value));
}

/// @brief it should reject text that names no float.
TEST(PayloadFloat, ParseInvalidText) {
    x::json::Parser parser(x::json::json{{"label", "OFF"}, {"value", "inf"}});
    FloatEntry::parse(parser);
    EXPECT_TRUE(parser.error());
}

/// @brief it should write a float as a number when finite and as text when not.
TEST(PayloadFloat, ToJSON) {
    const FloatRead finite{.fill = Float(2.5)};
    EXPECT_EQ(finite.to_json()["fill"], 2.5);
    const FloatRead inf{.fill = Float(INF)};
    EXPECT_EQ(inf.to_json()["fill"], "Infinity");
    const FloatRead nan{.fill = Float(std::numeric_limits<double>::quiet_NaN())};
    EXPECT_EQ(nan.to_json()["fill"], "NaN");
}

/// @brief it should parse an integer from a string of decimal digits.
TEST(PayloadInteger, ParseDecimalString) {
    x::json::Parser parser(x::json::json{{"fill", "-9223372036854775808"}});
    const auto read = IntegerRead::parse(parser);
    ASSERT_FALSE(parser.error());
    EXPECT_EQ(read.fill, std::numeric_limits<std::int64_t>::min());
}

/// @brief it should parse an unsigned integer from a string of decimal digits.
TEST(PayloadUnsigned, ParseDecimalString) {
    x::json::Parser parser(x::json::json{{"fill", "18446744073709551615"}});
    const auto read = UnsignedRead::parse(parser);
    ASSERT_FALSE(parser.error());
    EXPECT_EQ(read.fill, std::numeric_limits<std::uint64_t>::max());
}
}
