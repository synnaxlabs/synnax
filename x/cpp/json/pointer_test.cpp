// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/testutil/testutil.h"

namespace x::json {
/// @brief it should reject a pointer that does not start with a slash.
TEST(Pointer, RejectsMissingLeadingSlash) {
    const auto [p, err] = Pointer::parse("temperature");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        R"(json pointer "temperature" must be empty or start with "/")"
    );
}

/// @brief it should reject a tilde that does not form an escape.
TEST(Pointer, RejectsInvalidTildeEscape) {
    const auto [p, err] = Pointer::parse("/a~2b");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        R"(json pointer "/a~2b" is malformed: "~" must be followed by "0" or "1")"
    );
}

/// @brief it should reject a trailing tilde.
TEST(Pointer, RejectsTrailingTilde) {
    const auto [p, err] = Pointer::parse("/a~");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        R"(json pointer "/a~" is malformed: "~" must be followed by "0" or "1")"
    );
}

/// @brief it should parse the empty pointer as the root.
TEST(Pointer, ParsesRoot) {
    const auto p = ASSERT_NIL_P(Pointer::parse(""));
    ASSERT_TRUE(p.tokens.empty());
    ASSERT_EQ(p.to_string(), "");
}

/// @brief it should unescape each token and round trip the source text.
TEST(Pointer, RoundTripsEscapedTokens) {
    const auto p = ASSERT_NIL_P(Pointer::parse("/nested/a~1b/m~0n/~01"));
    ASSERT_EQ(p.tokens, (std::vector<std::string>{"nested", "a/b", "m~n", "~1"}));
    ASSERT_EQ(p.to_string(), "/nested/a~1b/m~0n/~01");
}

/// @brief it should keep an empty last token.
TEST(Pointer, RoundTripsEmptyToken) {
    const auto p = ASSERT_NIL_P(Pointer::parse("/nested/"));
    ASSERT_EQ(p.tokens, (std::vector<std::string>{"nested", ""}));
    ASSERT_EQ(p.to_string(), "/nested/");
}
}
