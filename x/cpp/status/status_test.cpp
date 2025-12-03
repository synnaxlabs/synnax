// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/status/status.h"

/// @brief it should correctly convert a status to its JSON representation.
TEST(StatusTest, TestToJSON) {
    status::Status<> stat{
        .key = "dog",
        .variant = status::variant::SUCCESS,
        .message = "the dog is happy",
        .description = "a longer description of the dog's status",
    };
    auto j = stat.to_json();
    ASSERT_EQ(j["key"], "dog");
    ASSERT_EQ(j["variant"], status::variant::SUCCESS);
    ASSERT_EQ(j["message"], "the dog is happy");
    ASSERT_EQ(j["description"], "a longer description of the dog's status");
    ASSERT_GT(j["time"], 0);
}

/// @brief it should correctly parse a status from its JSON representation.
TEST(StatusTest, TestParse) {
    json j = {
        {"key", "cat"},
        {"variant", status::variant::ERR},
        {"message", "the cat is angry"},
        {"description", "a longer description of the cat's status"},
        {"time", telem::TimeStamp(telem::SECOND).nanoseconds()}
    };
    xjson::Parser p(j);
    auto stat = status::Status<>::parse(p);
    ASSERT_EQ(stat.key, "cat");
    ASSERT_EQ(stat.variant, status::variant::ERR);
    ASSERT_EQ(stat.message, "the cat is angry");
    ASSERT_EQ(stat.description, "a longer description of the cat's status");
    ASSERT_EQ(stat.time, telem::TimeStamp(telem::SECOND));
}
