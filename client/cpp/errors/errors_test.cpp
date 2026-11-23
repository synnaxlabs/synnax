// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/auth/auth.h"
#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

#include "arc/cpp/errors/errors.h"

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
    ASSERT_NIL(check_one("range", x::json::json{{"key", {{"eq", 12}}}}, "", 1));
}

/// @brief it should describe the filter when no record matched.
TEST(ErrorsTest, CheckOneNotFound) {
    const auto err = check_one("range", x::json::json{{"key", {{"eq", 12}}}}, "", 0);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(err.data, "Found no range where key = 12");
}

/// @brief it should leave out the filter when it sets no conditions.
TEST(ErrorsTest, CheckOneNotFoundEmptyFilter) {
    const auto err = check_one("range", x::json::json::object(), "", 0);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(err.data, "Found no range");
}

/// @brief it should describe the filter and count the records when more than one
/// matched.
TEST(ErrorsTest, CheckOneMultipleFound) {
    const auto err = check_one(
        "range",
        x::json::json{{"name", {{"starts_with", "press"}}}},
        "",
        2
    );
    ASSERT_MATCHES(err, x::errors::MULTIPLE_RESULTS);
    EXPECT_EQ(
        err.data,
        R"(Expected one range where name starts with "press", but found 2)"
    );
}

/// @brief it should name the search term before the filter when no record matched.
TEST(ErrorsTest, CheckOneNotFoundSearchTerm) {
    const auto
        err = check_one("range", x::json::json{{"key", {{"eq", 12}}}}, "press", 0);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(err.data, R"(Found no range matching "press" where key = 12)");
}

/// @brief it should name the search term when more than one record matched.
TEST(ErrorsTest, CheckOneMultipleFoundSearchTerm) {
    const auto err = check_one("range", x::json::json::object(), "press", 2);
    ASSERT_MATCHES(err, x::errors::MULTIPLE_RESULTS);
    EXPECT_EQ(err.data, R"(Expected one range matching "press", but found 2)");
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

/// @brief it should classify each row of the table.
TEST(ErrorsTest, ClassifyTable) {
    const std::vector<std::pair<x::errors::Error, Kind>> cases = {
        {INTERRUPTED, Kind::INTERRUPTED},
        {freighter::UNREACHABLE, Kind::TRANSIENT},
        {freighter::TIMEOUT, Kind::TRANSIENT},
        {freighter::TRANSPORT, Kind::TRANSIENT},
        {freighter::RESOURCE_EXHAUSTED, Kind::TRANSIENT},
        {freighter::CANCELLED, Kind::TRANSIENT},
        {freighter::STREAM_CLOSED, Kind::TRANSIENT},
        {freighter::ERR_EOF, Kind::TRANSIENT},
        {LICENSE_MISSING, Kind::TRANSIENT},
        {LICENSE_EXPIRED, Kind::TRANSIENT},
        {auth::ERR_INVALID_TOKEN, Kind::TRANSIENT},
        {auth::ERR_EXPIRED_TOKEN, Kind::TRANSIENT},
        {auth::ERR_SESSION_ENDED, Kind::TRANSIENT},
        {LICENSE.sub("invalid"), Kind::REJECTED},
        {auth::ERR_INVALID_CREDENTIALS, Kind::REJECTED},
        {auth::ERR_ACCESS_DENIED, Kind::REJECTED},
        {x::errors::VALIDATION, Kind::REJECTED},
        {x::errors::NOT_FOUND, Kind::REJECTED},
        {x::errors::UNAUTHORIZED, Kind::REJECTED},
        {x::errors::NOT_SUPPORTED, Kind::REJECTED},
        {::arc::errors::BASE.sub("task"), Kind::REJECTED},
        {freighter::UNKNOWN, Kind::BUG},
        {x::errors::UNKNOWN, Kind::BUG},
        {x::errors::INTERNAL, Kind::BUG},
        {x::errors::UNEXPECTED, Kind::BUG},
    };
    for (const auto &[err, kind]: cases)
        EXPECT_EQ(classify(err), kind) << err;
}

/// @brief it should classify an error that the Core sent as a payload by its type.
TEST(ErrorsTest, ClassifyPayload) {
    EXPECT_EQ(
        classify(x::errors::Error("sy.license.missing---no license")),
        Kind::TRANSIENT
    );
    EXPECT_EQ(classify(x::errors::Error("sy.validation---bad key")), Kind::REJECTED);
}

/// @brief it should classify a type that only shares a prefix with a row as a bug.
TEST(ErrorsTest, ClassifyPrefixWithoutDot) {
    EXPECT_EQ(classify(x::errors::Error("sy.query_x", "")), Kind::BUG);
}
}
