// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <bit>
#include <charconv>
#include <cmath>
#include <cstdint>
#include <limits>
#include <random>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/strings/strings.h"

TEST(Put, Width2Zero) {
    char buf[2];
    x::strings::write_number(buf, 0, 2);
    EXPECT_EQ(std::string(buf, 2), "00");
}

TEST(Put, Width2SingleDigit) {
    char buf[2];
    x::strings::write_number(buf, 7, 2);
    EXPECT_EQ(std::string(buf, 2), "07");
}

TEST(Put, Width2TwoDigits) {
    char buf[2];
    x::strings::write_number(buf, 42, 2);
    EXPECT_EQ(std::string(buf, 2), "42");
}

TEST(Put, Width2Max) {
    char buf[2];
    x::strings::write_number(buf, 99, 2);
    EXPECT_EQ(std::string(buf, 2), "99");
}

TEST(Put, Width4Zero) {
    char buf[4];
    x::strings::write_number(buf, 0, 4);
    EXPECT_EQ(std::string(buf, 4), "0000");
}

TEST(Put, Width4Year) {
    char buf[4];
    x::strings::write_number(buf, 2026, 4);
    EXPECT_EQ(std::string(buf, 4), "2026");
}

TEST(Put, Width4SmallValue) {
    char buf[4];
    x::strings::write_number(buf, 5, 4);
    EXPECT_EQ(std::string(buf, 4), "0005");
}

TEST(Put, Width4Max) {
    char buf[4];
    x::strings::write_number(buf, 9999, 4);
    EXPECT_EQ(std::string(buf, 4), "9999");
}

TEST(Put, Width9Zero) {
    char buf[9];
    x::strings::write_number(buf, 0, 9);
    EXPECT_EQ(std::string(buf, 9), "000000000");
}

TEST(Put, Width9One) {
    char buf[9];
    x::strings::write_number(buf, 1, 9);
    EXPECT_EQ(std::string(buf, 9), "000000001");
}

TEST(Put, Width9Nanoseconds) {
    char buf[9];
    x::strings::write_number(buf, 123456789, 9);
    EXPECT_EQ(std::string(buf, 9), "123456789");
}

TEST(Put, Width9Max) {
    char buf[9];
    x::strings::write_number(buf, 999999999, 9);
    EXPECT_EQ(std::string(buf, 9), "999999999");
}

TEST(Put, Width1) {
    char buf[1];
    x::strings::write_number(buf, 5, 1);
    EXPECT_EQ(buf[0], '5');
}

TEST(Join, Empty) {
    EXPECT_EQ(x::strings::join({}, "; "), "");
}

TEST(Join, Single) {
    EXPECT_EQ(x::strings::join({"hello"}, "; "), "hello");
}

TEST(Join, Multiple) {
    EXPECT_EQ(x::strings::join({"a", "b", "c"}, "; "), "a; b; c");
}

TEST(Join, CommaSeparator) {
    EXPECT_EQ(x::strings::join({"x", "y"}, ", "), "x, y");
}

TEST(Join, EmptyStringsInParts) {
    EXPECT_EQ(x::strings::join({"", "b", ""}, "; "), "; b; ");
}

TEST(Trim, RemovesSurroundingWhitespace) {
    EXPECT_EQ(x::strings::trim("  hello \t"), "hello");
    EXPECT_EQ(x::strings::trim("hello"), "hello");
    EXPECT_EQ(x::strings::trim("value\r"), "value");
}

TEST(Trim, KeepsInteriorWhitespace) {
    EXPECT_EQ(x::strings::trim("  two words  "), "two words");
}

TEST(Trim, AllWhitespaceIsEmpty) {
    EXPECT_EQ(x::strings::trim(" \t\r"), "");
    EXPECT_EQ(x::strings::trim(""), "");
}

TEST(Unquote, RemovesOneLayer) {
    EXPECT_EQ(x::strings::unquote("\"hello\""), "hello");
    EXPECT_EQ(x::strings::unquote("'hello'"), "hello");
    EXPECT_EQ(x::strings::unquote("\"\"hello\"\""), "\"hello\"");
}

TEST(Unquote, KeepsUnquoted) {
    EXPECT_EQ(x::strings::unquote("hello"), "hello");
    EXPECT_EQ(x::strings::unquote("\"hello"), "\"hello");
    EXPECT_EQ(x::strings::unquote("'hello\""), "'hello\"");
    EXPECT_EQ(x::strings::unquote("\""), "\"");
    EXPECT_EQ(x::strings::unquote(""), "");
}

/// @brief a string without line breaks should come back unchanged.
TEST(EscapeLineBreaks, KeepsSingleLine) {
    EXPECT_EQ(x::strings::escape_line_breaks("hello  world\t"), "hello  world\t");
    EXPECT_EQ(x::strings::escape_line_breaks(""), "");
}

/// @brief every '\n' and '\r' should become its two-character escape.
TEST(EscapeLineBreaks, EscapesEveryLineBreak) {
    EXPECT_EQ(
        x::strings::escape_line_breaks("<html>\n  <body>\r\n</body>\n"),
        "<html>\\n  <body>\\r\\n</body>\\n"
    );
}

/// @brief it should write the shortest text in fixed notation inside its range.
TEST(FormatFloat, FixedNotation) {
    EXPECT_EQ(x::strings::format_float(-1.5), "-1.5");
    EXPECT_EQ(x::strings::format_float(1e10), "10000000000");
    EXPECT_EQ(x::strings::format_float(1e20), "100000000000000000000");
    EXPECT_EQ(x::strings::format_float(1e-6), "0.000001");
    EXPECT_EQ(x::strings::format_float(1000000000000000128.0), "1000000000000000100");
    EXPECT_EQ(x::strings::format_float(1e20f), "100000000000000000000");
    EXPECT_EQ(x::strings::format_float(0.1f), "0.1");
    EXPECT_EQ(x::strings::format_float(1e-6f), "0.000001");
    EXPECT_EQ(x::strings::format_float(0.000244140625f), "0.00024414062");
}

/// @brief it should write exponent notation below 1e-6 and from 1e21.
TEST(FormatFloat, ExponentNotation) {
    EXPECT_EQ(x::strings::format_float(1e21), "1e+21");
    EXPECT_EQ(x::strings::format_float(1.5e-7), "1.5e-7");
    EXPECT_EQ(x::strings::format_float(5e-324), "5e-324");
    EXPECT_EQ(
        x::strings::format_float(std::numeric_limits<double>::max()),
        "1.7976931348623157e+308"
    );
    EXPECT_EQ(x::strings::format_float(1e30f), "1e+30");
    EXPECT_EQ(x::strings::format_float(1.5e-7f), "1.5e-7");
}

/// @brief it should write zero, NaN, and the infinities.
TEST(FormatFloat, SpecialValues) {
    EXPECT_EQ(x::strings::format_float(0.0), "0");
    EXPECT_EQ(x::strings::format_float(std::copysign(0.0, -1.0)), "-0");
    EXPECT_EQ(
        x::strings::format_float(std::numeric_limits<double>::quiet_NaN()),
        "NaN"
    );
    EXPECT_EQ(x::strings::format_float(std::numeric_limits<float>::quiet_NaN()), "NaN");
    EXPECT_EQ(
        x::strings::format_float(std::numeric_limits<double>::infinity()),
        "Infinity"
    );
    EXPECT_EQ(
        x::strings::format_float(-std::numeric_limits<float>::infinity()),
        "-Infinity"
    );
}

/// @brief it should write text that parses back to the same bits.
TEST(FormatFloat, RoundTrip) {
    // NOLINTNEXTLINE(bugprone-random-generator-seed): a repeatable set of values.
    std::mt19937_64 rng(1);
    for (int i = 0; i < 100000; ++i) {
        const std::uint64_t bits64 = rng();
        const auto v64 = std::bit_cast<double>(bits64);
        const auto bits32 = static_cast<std::uint32_t>(rng());
        const auto v32 = std::bit_cast<float>(bits32);
        if (std::isnan(v64) || std::isnan(v32)) continue;
        const std::string t64 = x::strings::format_float(v64);
        double p64 = 0;
        std::from_chars(t64.data(), t64.data() + t64.size(), p64);
        if (!std::isinf(v64)) {
            ASSERT_EQ(std::bit_cast<std::uint64_t>(p64), bits64) << t64;
        }
        const std::string t32 = x::strings::format_float(v32);
        float p32 = 0;
        std::from_chars(t32.data(), t32.data() + t32.size(), p32);
        if (!std::isinf(v32)) {
            ASSERT_EQ(std::bit_cast<std::uint32_t>(p32), bits32) << t32;
        }
    }
}
