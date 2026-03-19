// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/status/status.h"
#include "x/cpp/test/test.h"

namespace x::status {
struct TestDetails {
    std::string field1;
    int field2 = 0;

    [[nodiscard]] json::json to_json() const {
        return json::json{{"field1", field1}, {"field2", field2}};
    }

    static TestDetails parse(json::Parser parser) {
        return TestDetails{
            .field1 = parser.field<std::string>("field1", ""),
            .field2 = parser.field<int>("field2", 0),
        };
    }
};

TEST(StatusTest, TestToJSON) {
    const Status stat{
        .key = "dog",
        .name = "dog status",
        .variant = VARIANT_SUCCESS,
        .message = "the dog is happy",
        .description = "a longer description of the dog's status",
        .time = telem::TimeStamp::now(),
        .details = TestDetails{},
        .labels = {},
    };
    auto j = stat.to_json();
    ASSERT_EQ(j["key"], "dog");
    ASSERT_EQ(j["variant"], VARIANT_SUCCESS);
    ASSERT_EQ(j["message"], "the dog is happy");
    ASSERT_EQ(j["description"], "a longer description of the dog's status");
    ASSERT_GT(j["time"], 0);
}

TEST(StatusTest, TestParse) {
    json::json j = {
        {"key", "cat"},
        {"name", "cat status"},
        {"variant", VARIANT_ERROR},
        {"message", "the cat is angry"},
        {"description", "a longer description of the cat's status"},
        {"time", telem::TimeStamp(telem::SECOND).nanoseconds()},
        {"details", {{"field1", "test"}, {"field2", 123}}},
        {"labels", json::json::array()}
    };
    const json::Parser p(j);
    const auto stat = Status<TestDetails>::parse(p);
    ASSERT_EQ(stat.key, "cat");
    ASSERT_EQ(stat.variant, VARIANT_ERROR);
    ASSERT_EQ(stat.message, "the cat is angry");
    ASSERT_EQ(stat.description, "a longer description of the cat's status");
    ASSERT_EQ(stat.time, telem::TimeStamp(telem::SECOND));
}

TEST(StatusTest, TestProtobufRoundTrip) {
    const Status original{
        .key = "test-key",
        .name = "Test Status",
        .variant = VARIANT_INFO,
        .message = "test message",
        .description = "test description",
        .time = telem::TimeStamp::now(),
        .details = json::json{{"field1", "hello"}, {"field2", 42}},
        .labels = {},
    };
    const auto pb = original.to_proto();
    ASSERT_EQ(pb.key(), "test-key");
    ASSERT_EQ(pb.name(), "Test Status");
    ASSERT_EQ(pb.message(), "test message");
}
}
