// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

/// @brief it should remove a trailing line feed.
TEST(TrimTrailingLineBreaks, RemovesLineFeed) {
    EXPECT_EQ(x::strings::trim_trailing_line_breaks("value\n"), "value");
}

/// @brief it should remove every trailing carriage return and line feed.
TEST(TrimTrailingLineBreaks, RemovesCarriageReturnLineFeed) {
    EXPECT_EQ(x::strings::trim_trailing_line_breaks("value\r\n"), "value");
    EXPECT_EQ(x::strings::trim_trailing_line_breaks("value\r\n\n"), "value");
}

/// @brief it should keep spaces and the line breaks before the end.
TEST(TrimTrailingLineBreaks, KeepsSpacesAndInnerLineBreaks) {
    EXPECT_EQ(x::strings::trim_trailing_line_breaks(" a\nb \n"), " a\nb ");
}

/// @brief it should return an empty string for a string of only line breaks.
TEST(TrimTrailingLineBreaks, AllLineBreaksIsEmpty) {
    EXPECT_EQ(x::strings::trim_trailing_line_breaks("\r\n"), "");
    EXPECT_EQ(x::strings::trim_trailing_line_breaks(""), "");
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

/// @brief it should accept ASCII and well-formed multi-byte sequences.
TEST(ValidUTF8, AcceptsWellFormed) {
    EXPECT_TRUE(x::strings::valid_utf8(""));
    EXPECT_TRUE(x::strings::valid_utf8("plain"));
    EXPECT_TRUE(x::strings::valid_utf8("caf\xC3\xA9 \xE2\x82\xAC \xF0\x9F\x98\x80"));
}

/// @brief it should reject bad lead and continuation bytes and cut sequences.
TEST(ValidUTF8, RejectsMalformed) {
    EXPECT_FALSE(x::strings::valid_utf8("caf\xE9"));
    EXPECT_FALSE(x::strings::valid_utf8("\x80"));
    EXPECT_FALSE(x::strings::valid_utf8("\xC3"));
    EXPECT_FALSE(x::strings::valid_utf8("\xE2\x82"));
    EXPECT_FALSE(x::strings::valid_utf8("\xC3\x28"));
    EXPECT_FALSE(x::strings::valid_utf8("\xFF"));
}

/// @brief it should reject overlong forms, surrogates, and code points past U+10FFFF.
TEST(ValidUTF8, RejectsOutOfRange) {
    EXPECT_FALSE(x::strings::valid_utf8("\xC0\xAF"));
    EXPECT_FALSE(x::strings::valid_utf8("\xE0\x80\xAF"));
    EXPECT_FALSE(x::strings::valid_utf8("\xED\xA0\x80"));
    EXPECT_FALSE(x::strings::valid_utf8("\xF4\x90\x80\x80"));
}
