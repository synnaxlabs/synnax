// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <optional>
#include <utility>
#include <vector>

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

/// @brief it should create a not found error that describes the filter.
TEST(ErrorsTest, NotFoundError) {
    const auto err = not_found_error("channel", "key = 123");
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(err.message(), "[sy.query.not_found] Found no channel where key = 123");
}

/// @brief it should leave out an empty filter description.
TEST(ErrorsTest, NotFoundErrorNoFilter) {
    const auto err = not_found_error("task", "");
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(err.message(), "[sy.query.not_found] Found no task");
}

/// @brief it should create a multiple results error that describes the filter.
TEST(ErrorsTest, MultipleFoundError) {
    const auto err = multiple_found_error("range", "key = 1", 2);
    ASSERT_MATCHES(err, x::errors::MULTIPLE_RESULTS);
    EXPECT_EQ(
        err.message(),
        "[sy.query.multiple_results] Expected one range where key = 1, but found 2"
    );
}

/// @brief it should leave out an empty filter description.
TEST(ErrorsTest, MultipleFoundErrorNoFilter) {
    const auto err = multiple_found_error("range", "", 3);
    ASSERT_MATCHES(err, x::errors::MULTIPLE_RESULTS);
    EXPECT_EQ(
        err.message(),
        "[sy.query.multiple_results] Expected one range, but found 3"
    );
}

/// @brief it should match only the errors that clear without a restart.
TEST(ErrorsTest, IsTemporarilyUnavailable) {
    EXPECT_TRUE(is_temporarily_unavailable(freighter::UNREACHABLE));
    EXPECT_TRUE(is_temporarily_unavailable(LICENSE_MISSING));
    EXPECT_TRUE(is_temporarily_unavailable(LICENSE_EXPIRED));
    EXPECT_FALSE(is_temporarily_unavailable(LICENSE));
    EXPECT_FALSE(is_temporarily_unavailable(LICENSE.sub("invalid")));
    EXPECT_FALSE(is_temporarily_unavailable(x::errors::UNKNOWN));
}

namespace {
/// @brief a filter that selects the record with key 12.
struct KeyWhere {
    [[nodiscard]] x::json::json to_json() const { return {{"key", {{"eq", 12}}}}; }
};

const std::optional WHERE = KeyWhere{};
}

/// @brief it should return the one record that a retrieve found.
TEST(ErrorsTest, ExactlyOne) {
    const auto record = ASSERT_NIL_P(
        exactly_one("range", WHERE, std::pair{std::vector{7}, x::errors::NIL})
    );
    EXPECT_EQ(record, 7);
}

/// @brief it should return the error of the retrieve.
TEST(ErrorsTest, ExactlyOneRetrieveError) {
    ASSERT_OCCURRED_AS_P(
        exactly_one(
            "range",
            WHERE,
            std::pair{std::vector<int>{}, freighter::UNREACHABLE}
        ),
        freighter::UNREACHABLE
    );
}

/// @brief it should return a not found error that describes the filter.
TEST(ErrorsTest, ExactlyOneNotFound) {
    const auto [record, err] = exactly_one(
        "range",
        WHERE,
        std::pair{std::vector<int>{}, x::errors::NIL}
    );
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    EXPECT_EQ(err.data, "Found no range where key = 12");
}

/// @brief it should return a multiple results error that counts the records.
TEST(ErrorsTest, ExactlyOneMultipleFound) {
    const auto [record, err] = exactly_one(
        "range",
        WHERE,
        std::pair{std::vector{7, 8}, x::errors::NIL}
    );
    ASSERT_MATCHES(err, x::errors::MULTIPLE_RESULTS);
    EXPECT_EQ(err.data, "Expected one range where key = 12, but found 2");
}
}
