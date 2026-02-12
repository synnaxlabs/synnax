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
#include "x/cpp/errors/errors.h"
#include "x/cpp/test/test.h"

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

/// @brief it should create a multiple results error for channels.
TEST(ErrorsTest, MultipleFoundError) {
    const auto err = multiple_found_error("channels", "name test");
    ASSERT_MATCHES(err, x::errors::MULTIPLE_RESULTS);
    EXPECT_EQ(
        err.message(),
        "[sy.query.multiple_results] Multiple channels matching name test not found."
    );
}

/// @brief it should create a multiple results error for ranges.
TEST(ErrorsTest, MultipleFoundErrorRanges) {
    const auto err = multiple_found_error("ranges", "name experiment-1");
    ASSERT_MATCHES(err, x::errors::MULTIPLE_RESULTS);
    EXPECT_EQ(
        err.message(),
        "[sy.query.multiple_results] Multiple ranges matching name experiment-1 not found."
    );
}
}
