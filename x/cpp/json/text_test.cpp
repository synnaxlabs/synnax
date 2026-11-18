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

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/text.h"
#include "x/cpp/testutil/testutil.h"

namespace x::json {
/// @brief it should quote plain text as it is.
TEST(Quote, PlainText) {
    ASSERT_EQ(quote("temp é"), "\"temp é\"");
}

/// @brief it should escape quotes, backslashes, and control characters.
TEST(Quote, EscapesSpecialCharacters) {
    ASSERT_EQ(quote("a\"b\\c\n\t\x01\x7f"), R"("a\"b\\c\n\t\x01\x7f")");
}

/// @brief it should escape a C1 control character.
TEST(Quote, EscapesC1ControlCharacter) {
    ASSERT_EQ(
        quote(
            "a\xc2\x85"
            "b"
        ),
        R"("a\u0085b")"
    );
}

/// @brief it should escape bytes that are not valid UTF-8.
TEST(Quote, EscapesInvalidUTF8) {
    ASSERT_EQ(quote("a\xff"), R"("a\xff")");
}

/// @brief it should decode escapes, including a surrogate pair.
TEST(Unquote, DecodesEscapes) {
    std::string dst = "stale";
    ASSERT_NIL(unquote(R"("a\"\\\/\b\f\n\r\t\u0061\ud83d\ude00")", dst));
    ASSERT_EQ(dst, "a\"\\/\b\f\n\r\ta\xf0\x9f\x98\x80");
}

/// @brief it should keep UTF-8 text as it is.
TEST(Unquote, KeepsUTF8) {
    std::string dst;
    ASSERT_NIL(unquote("\"é\"", dst));
    ASSERT_EQ(dst, "é");
}

/// @brief it should reject a high surrogate that no low surrogate follows.
TEST(Unquote, RejectsLoneHighSurrogate) {
    std::string dst;
    const auto err = unquote(R"("\ud800")", dst);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid surrogate in escape sequence in string literal");
}

/// @brief it should reject a low surrogate on its own.
TEST(Unquote, RejectsLoneLowSurrogate) {
    std::string dst;
    const auto err = unquote(R"("\udc00\udc00")", dst);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid surrogate in escape sequence in string literal");
}

/// @brief it should reject an escape that JSON does not define.
TEST(Unquote, RejectsUnknownEscape) {
    std::string dst;
    const auto err = unquote(R"("\x41")", dst);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid escape sequence in string literal");
}

/// @brief it should reject bytes that are not valid UTF-8.
TEST(Unquote, RejectsInvalidUTF8) {
    std::string dst;
    const auto err = unquote("\"\xed\xa0\x80\"", dst);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid UTF-8 in string literal");
}

/// @brief it should reject a control character that is not escaped.
TEST(Unquote, RejectsControlCharacter) {
    std::string dst;
    const auto err = unquote("\"a\nb\"", dst);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid control character in string literal");
}

/// @brief it should reject a literal with no closing quote.
TEST(Unquote, RejectsUnterminatedLiteral) {
    std::string dst;
    const auto err = unquote(R"("abc\")", dst);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "unexpected EOF in string literal");
}

/// @brief it should reject data after the closing quote.
TEST(Unquote, RejectsTrailingData) {
    std::string dst;
    const auto err = unquote(R"("a"b)", dst);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "unexpected data after the string literal");
}

/// @brief it should reject text that is not a string literal.
TEST(Unquote, RejectsNonString) {
    std::string dst;
    const auto err = unquote("12", dst);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "expected a JSON string literal");
}
}
