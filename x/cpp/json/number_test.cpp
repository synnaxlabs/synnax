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
#include <optional>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/number.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"

namespace x::json {
namespace {
constexpr double INF = std::numeric_limits<double>::infinity();
constexpr double NaN = std::numeric_limits<double>::quiet_NaN();
constexpr std::uint64_t UINT64_MAX_VALUE = std::numeric_limits<std::uint64_t>::max();

/// @brief checks that text parses to a number with the given integer and float values.
void expect_number(
    const std::string &text,
    const std::optional<std::int64_t> i,
    const std::optional<std::uint64_t> u,
    const double f
) {
    const auto n = ASSERT_NIL_P(Number::parse(text));
    EXPECT_EQ(n.to_int64(), i);
    EXPECT_EQ(n.to_uint64(), u);
    EXPECT_EQ(n.to_float64(), f);
}

/// @brief returns the error that check_fits gives for the number text and dt.
errors::Error fits(const std::string &text, const telem::DataType &dt) {
    return ASSERT_NIL_P(Number::parse(text)).check_fits(dt);
}
}

/// @brief it should hold a small integer exactly.
TEST(Number, ParsesSmallInteger) {
    expect_number("3", 3, 3, 3.0);
}

/// @brief it should hold a whole decimal as an integer.
TEST(Number, ParsesWholeDecimal) {
    expect_number("3.0", 3, 3, 3.0);
}

/// @brief it should hold an integer written with an exponent as an integer.
TEST(Number, ParsesIntegerWithExponent) {
    expect_number("1e2", 100, 100, 100.0);
}

/// @brief it should hold a negative integer only as a signed integer.
TEST(Number, ParsesNegativeInteger) {
    expect_number("-3", -3, std::nullopt, -3.0);
}

/// @brief it should hold an integer past the precision of a double exactly.
TEST(Number, ParsesIntegerPastFloatPrecision) {
    expect_number(
        "9007199254740993",
        9007199254740993,
        9007199254740993,
        9007199254740992.0
    );
}

/// @brief it should keep a fraction that a double rounds away from being an integer.
TEST(Number, KeepsFractionThatFloatRoundsAway) {
    expect_number("9007199254740993.5", std::nullopt, std::nullopt, 9007199254740994.0);
}

/// @brief it should hold an integer past an int64 as a uint64.
TEST(Number, ParsesIntegerPastInt64) {
    expect_number(
        "18446744073709551615",
        std::nullopt,
        UINT64_MAX_VALUE,
        1.8446744073709552e19
    );
}

/// @brief it should hold the smallest int64 exactly.
TEST(Number, ParsesSmallestInt64) {
    expect_number(
        "-9223372036854775808",
        std::numeric_limits<std::int64_t>::min(),
        std::nullopt,
        -9223372036854775808.0
    );
}

/// @brief it should hold an integer past a uint64 as a double.
TEST(Number, ParsesIntegerPastUint64) {
    expect_number("1e20", std::nullopt, std::nullopt, 1e20);
}

/// @brief it should keep the sign of a negative zero as a double.
TEST(Number, KeepsNegativeZero) {
    const auto n = ASSERT_NIL_P(Number::parse("-0.0"));
    ASSERT_EQ(n.to_int64(), 0);
    ASSERT_EQ(n.to_uint64(), 0U);
    ASSERT_TRUE(std::signbit(n.to_float64()));
}

/// @brief it should convert a whole double to an exact integer.
TEST(Number, ConvertsWholeDoubleToInteger) {
    const Number n(3.0);
    ASSERT_EQ(n.to_int64(), 3);
    ASSERT_EQ(n.to_uint64(), 3U);
}

/// @brief it should hold a fraction only as a double.
TEST(Number, ParsesFraction) {
    expect_number("2.5", std::nullopt, std::nullopt, 2.5);
}

/// @brief it should read a number too small for a double as zero.
TEST(Number, ParsesUnderflowAsZero) {
    expect_number("1e-400", std::nullopt, std::nullopt, 0.0);
}

/// @brief it should reject text that is not a number.
TEST(Number, RejectsWord) {
    const auto [n, err] = Number::parse("x");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, R"(invalid number "x")");
}

/// @brief it should reject a sign that another sign follows.
TEST(Number, RejectsDoubleSign) {
    const auto [n, err] = Number::parse("+-5");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, R"(invalid number "+-5")");
}

/// @brief it should reject a number past the range of a double.
TEST(Number, RejectsNumberPastFloat) {
    const auto [n, err] = Number::parse("1e999");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, R"(invalid number "1e999")");
}

/// @brief it should reject a fraction for an integer data type.
TEST(NumberCheckFits, RejectsFractionForInteger) {
    const auto err = fits("1.5", telem::INT64_T);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "1.5 is not an integer");
}

/// @brief it should print the text of a fraction that a double rounds away.
TEST(NumberCheckFits, PrintsTextOfRoundedFraction) {
    const auto err = fits("9007199254740993.5", telem::INT64_T);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "9007199254740993.5 is not an integer");
}

/// @brief it should reject a uint64 past an int64 for an int64 data type.
TEST(NumberCheckFits, RejectsUint64ForInt64) {
    const auto err = fits("18446744073709551615", telem::INT64_T);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "18446744073709551615 is out of range for int64");
}

/// @brief it should reject a negative number for an unsigned data type.
TEST(NumberCheckFits, RejectsNegativeForUnsigned) {
    const auto err = fits("-1", telem::UINT64_T);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "-1 is out of range for uint64");
}

/// @brief it should reject an integer past the range of a small data type.
TEST(NumberCheckFits, RejectsIntegerPastInt8) {
    const auto err = fits("128", telem::INT8_T);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "128 is out of range for int8");
}

/// @brief it should take the bounds of a small data type.
TEST(NumberCheckFits, AcceptsInt8Bounds) {
    ASSERT_NIL(fits("-128", telem::INT8_T));
    ASSERT_NIL(fits("127", telem::INT8_T));
}

/// @brief it should take a whole double at the top of an unsigned range.
TEST(NumberCheckFits, AcceptsWholeFloatAtUint8Bound) {
    ASSERT_NIL(Number(255.0).check_fits(telem::UINT8_T));
}

/// @brief it should reject a float past the range of a float32.
TEST(NumberCheckFits, RejectsFloat32Overflow) {
    const auto err = fits("1e39", telem::FLOAT32_T);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "1e39 overflows float32");
}

/// @brief it should take any number for a float64 data type.
TEST(NumberCheckFits, AcceptsAnyNumberForFloat64) {
    ASSERT_NIL(fits("1e300", telem::FLOAT64_T));
    ASSERT_NIL(fits("9007199254740993.5", telem::FLOAT64_T));
}

/// @brief it should reject NaN for an integer data type.
TEST(NumberCheckFits, RejectsNaNForInteger) {
    const auto err = Number(NaN).check_fits(telem::INT8_T);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "NaN is not an integer");
}

/// @brief it should reject a whole float past the range of an integer data type.
TEST(NumberCheckFits, RejectsWholeFloatPastInt8) {
    const auto err = Number(300.0).check_fits(telem::INT8_T);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "300 is out of range for int8");
}

/// @brief it should print a number that is not finite by name.
TEST(NumberToString, PrintsNonFiniteByName) {
    ASSERT_EQ(Number(NaN).to_string(), "NaN");
    ASSERT_EQ(Number(INF).to_string(), "Infinity");
    ASSERT_EQ(Number(-INF).to_string(), "-Infinity");
}

/// @brief it should print a double in plain form from 1e-4 up to 1e6.
TEST(NumberToString, PrintsPlainForm) {
    ASSERT_EQ(Number(1.5).to_string(), "1.5");
    ASSERT_EQ(Number(123456.0).to_string(), "123456");
    ASSERT_EQ(Number(0.000123).to_string(), "0.000123");
    ASSERT_EQ(Number(-0.0).to_string(), "-0");
}

/// @brief it should print a double with an exponent below 1e-4 and from 1e6.
TEST(NumberToString, PrintsExponentForm) {
    ASSERT_EQ(Number(1e6).to_string(), "1e+06");
    ASSERT_EQ(Number(1234567.0).to_string(), "1.234567e+06");
    ASSERT_EQ(Number(1e-5).to_string(), "1e-05");
    ASSERT_EQ(Number(9007199254740994.0).to_string(), "9.007199254740994e+15");
    ASSERT_EQ(Number(1e100).to_string(), "1e+100");
}

/// @brief it should print an integer parsed from text without its exponent.
TEST(NumberToString, PrintsParsedIntegerExactly) {
    const auto n = ASSERT_NIL_P(Number::parse("1e2"));
    ASSERT_EQ(n.to_string(), "100");
}
}
