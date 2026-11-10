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

#include "x/cpp/base64/base64.h"
#include "x/cpp/testutil/testutil.h"

namespace x::base64 {

TEST(Base64, EncodeEmpty) {
    EXPECT_EQ(encode(""), "");
}

TEST(Base64, EncodeHelloWorld) {
    EXPECT_EQ(encode("Hello, World!"), "SGVsbG8sIFdvcmxkIQ==");
}

TEST(Base64, EncodeBasicAuth) {
    EXPECT_EQ(encode("user:pass"), "dXNlcjpwYXNz");
}

TEST(Base64, DecodeEmpty) {
    EXPECT_EQ(decode(""), "");
}

TEST(Base64, DecodeHelloWorld) {
    EXPECT_EQ(decode("SGVsbG8sIFdvcmxkIQ=="), "Hello, World!");
}

TEST(Base64, RoundTrip) {
    const std::string inputs[] = {
        "",
        std::string("\0", 1),
        "Hello, World!",
        "user:pass",
        "a",
        "ab",
        "abc",
        "abcd",
        "The quick brown fox jumps over the lazy dog",
    };
    for (const auto &input: inputs)
        EXPECT_EQ(decode(encode(input)), input) << "Failed for: " << input;
}

TEST(Base64, DecodePaddedInput) {
    // One byte -> 4 chars with "==" padding
    EXPECT_EQ(decode("YQ=="), "a");
    // Two bytes -> 4 chars with "=" padding
    EXPECT_EQ(decode("YWI="), "ab");
    // Three bytes -> 4 chars with no padding
    EXPECT_EQ(decode("YWJj"), "abc");
}

/// @brief it should decode canonical, padded base64 text.
TEST(Base64, DecodeStrict) {
    EXPECT_EQ(ASSERT_NIL_P(decode_strict("")), "");
    EXPECT_EQ(ASSERT_NIL_P(decode_strict("YQ==")), "a");
    EXPECT_EQ(ASSERT_NIL_P(decode_strict("YWI=")), "ab");
    EXPECT_EQ(ASSERT_NIL_P(decode_strict("YWJj")), "abc");
    EXPECT_EQ(
        ASSERT_NIL_P(decode_strict("AP8BAg==")),
        std::string("\x00\xff\x01\x02", 4)
    );
}

/// @brief it should reject unpadded text, characters outside the alphabet, padding
/// before the end, and nonzero trailing bits.
TEST(Base64, DecodeStrictRejectsNonCanonical) {
    for (const auto *text:
         {"YQ", "YWI", "Y Q==", "YQ==\n", "Y*==", "YQ==YQ==", "YR==", "YWJ="}) {
        SCOPED_TRACE(text);
        ASSERT_OCCURRED_AS_P(decode_strict(text), INVALID);
    }
}

}
