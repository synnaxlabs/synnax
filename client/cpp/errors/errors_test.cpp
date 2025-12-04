// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/errors/errors.h"
#include "x/cpp/xerrors/errors.h"

TEST(ErrorsTest, UnexpectedMissingError) {
    auto err = synnax::unexpected_missing_error("channel");
    ASSERT_TRUE(err.matches(xerrors::UNEXPECTED));
    EXPECT_EQ(
        err.message(),
        "[sy.unexpected] No channel returned from server on create. Please report this error to the Synnax team."
    );
}

TEST(ErrorsTest, NotFoundError) {
    auto err = synnax::not_found_error("channel", "key 123");
    ASSERT_TRUE(err.matches(xerrors::NOT_FOUND));
    EXPECT_EQ(
        err.message(),
        "[sy.query.not_found] channel matching key 123 not found."
    );
}

TEST(ErrorsTest, NotFoundErrorWithName) {
    auto err = synnax::not_found_error("task", "name test-task");
    ASSERT_TRUE(err.matches(xerrors::NOT_FOUND));
    EXPECT_EQ(
        err.message(),
        "[sy.query.not_found] task matching name test-task not found."
    );
}

TEST(ErrorsTest, MultipleFoundError) {
    auto err = synnax::multiple_found_error("channels", "name test");
    ASSERT_TRUE(err.matches(xerrors::MULTIPLE_RESULTS));
    EXPECT_EQ(
        err.message(),
        "[sy.query.multiple_results] Multiple channels matching name test not found."
    );
}

TEST(ErrorsTest, MultipleFoundErrorRanges) {
    auto err = synnax::multiple_found_error("ranges", "name experiment-1");
    ASSERT_TRUE(err.matches(xerrors::MULTIPLE_RESULTS));
    EXPECT_EQ(
        err.message(),
        "[sy.query.multiple_results] Multiple ranges matching name experiment-1 not found."
    );
}
