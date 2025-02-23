// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// internal
#include "x/cpp/xerrors/errors.h"

TEST(testXErrors, testErrorConstructionFromString) {
    std::string error = "sy.validation---invalid key: 1000: validation error";
    auto err = xerrors::Error(error);
}

TEST(testXErrors, testErrorEqualsExactlyEqual) {
    auto err1 = xerrors::Error("test", "");
    auto err2 = xerrors::Error("test", "");
    ASSERT_EQ(err1, err2);
}

TEST(testXErrors, testErrorHequalHasPrefix) {
    auto err1 = xerrors::Error("test", "");
    auto err2 = xerrors::Error("test-specific", "");
    ASSERT_TRUE(err2.matches(err1));
}

TEST(testXErrors, testErrorMatchesVariadic) {
    auto err = xerrors::Error("test.specific.error", "");
    ASSERT_TRUE(err.matches("other", "test", "another"));
    ASSERT_TRUE(err.matches(
        xerrors::Error("wrong", ""),
        "test.specific",
        xerrors::Error("another", "")
    ));
    ASSERT_FALSE(err.matches("wrong", "other", "another"));
}

TEST(testXErrors, testErrorMatchesVector) {
    auto err = xerrors::Error("test.specific.error", "");
    std::vector<xerrors::Error> errors = {
        xerrors::Error("wrong", ""),
        xerrors::Error("test.specific", ""),
        xerrors::Error("another", "")
    };
    ASSERT_TRUE(err.matches(errors));
    std::vector<xerrors::Error> no_matches = {
        xerrors::Error("wrong", ""),
        xerrors::Error("other", ""),
        xerrors::Error("another", "")
    };
    ASSERT_FALSE(err.matches(no_matches));
}
