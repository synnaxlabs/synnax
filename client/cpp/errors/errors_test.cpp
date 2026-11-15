// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::errors {
/// @brief it should create an unexpected missing error with correct type and message.
TEST(ErrorsTest, UnexpectedMissingError) {
    const auto err = unexpected_missing_error("channel");
    ASSERT_MATCHES(err, x::errors::UNEXPECTED);
    EXPECT_EQ(
        err.message(),
        "[sy.unexpected] No channel returned from server on create. Please report this error to the Synnax team."
    );
}

/// @brief it should create a not found error with key identifier.
TEST(ErrorsTest, NotFoundError) {
    const auto err = not_found_error("channel", "key 123");
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(
        err.message(),
        "[sy.query.not_found] channel matching key 123 not found."
    );
}

/// @brief it should create a not found error with name identifier.
TEST(ErrorsTest, NotFoundErrorWithName) {
    const auto err = not_found_error("task", "name test-task");
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(
        err.message(),
        "[sy.query.not_found] task matching name test-task not found."
    );
}

/// @brief it should accept exactly one record.
TEST(ErrorsTest, CheckOneFound) {
    ASSERT_NIL(check_one("range", x::json::json{{"key", {{"eq", 12}}}}, 1));
}

/// @brief it should describe the filter when no record matched.
TEST(ErrorsTest, CheckOneNotFound) {
    const auto err = check_one("range", x::json::json{{"key", {{"eq", 12}}}}, 0);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(err.data, "Found no range where key = 12");
}

/// @brief it should leave out the filter when it sets no conditions.
TEST(ErrorsTest, CheckOneNotFoundEmptyFilter) {
    const auto err = check_one("range", x::json::json::object(), 0);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(err.data, "Found no range");
}

/// @brief it should describe the filter and count the records when more than one
/// matched.
TEST(ErrorsTest, CheckOneMultipleFound) {
    const auto err = check_one(
        "range",
        x::json::json{{"name", {{"starts_with", "press"}}}},
        2
    );
    ASSERT_MATCHES(err, x::errors::MULTIPLE_RESULTS);
    EXPECT_EQ(
        err.data,
        R"(Expected one range where name starts with "press", but found 2)"
    );
}

/// @brief it should match only the errors that clear without a restart.
TEST(ErrorsTest, IsTemporarilyUnavailable) {
    EXPECT_TRUE(is_temporarily_unavailable(freighter::UNREACHABLE));
    EXPECT_TRUE(is_temporarily_unavailable(freighter::TIMEOUT));
    EXPECT_TRUE(is_temporarily_unavailable(LICENSE_MISSING));
    EXPECT_TRUE(is_temporarily_unavailable(LICENSE_EXPIRED));
    EXPECT_FALSE(is_temporarily_unavailable(LICENSE));
    EXPECT_FALSE(is_temporarily_unavailable(LICENSE.sub("invalid")));
    EXPECT_FALSE(is_temporarily_unavailable(x::errors::UNKNOWN));
}
}
