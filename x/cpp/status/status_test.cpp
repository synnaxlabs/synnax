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
#include "x/cpp/xtest/xtest.h"

struct TestDetails {
    std::string field1;
    int field2 = 0;

    [[nodiscard]] json to_json() const {
        return json{{"field1", field1}, {"field2", field2}};
    }

    static TestDetails parse(xjson::Parser parser) {
        return TestDetails{
            .field1 = parser.field<std::string>("field1", ""),
            .field2 = parser.field<int>("field2", 0),
        };
    }
};

TEST(StatusTest, TestToJSON) {
    const synnax::status::Status<TestDetails> stat{
        .key = "dog",
        .name = "dog status",
        .message = "the dog is happy",
        .description = "a longer description of the dog's status",
        .time = telem::TimeStamp::now(),
        .details = TestDetails{},
        .labels = {},
        .variant = synnax::status::VARIANT_SUCCESS,
    };
    auto j = stat.to_json();
    ASSERT_EQ(j["key"], "dog");
    ASSERT_EQ(j["variant"], synnax::status::VARIANT_SUCCESS);
    ASSERT_EQ(j["message"], "the dog is happy");
    ASSERT_EQ(j["description"], "a longer description of the dog's status");
    ASSERT_GT(j["time"], 0);
}

TEST(StatusTest, TestParse) {
    json j = {
        {"key", "cat"},
        {"name", "cat status"},
        {"variant", synnax::status::VARIANT_ERROR},
        {"message", "the cat is angry"},
        {"description", "a longer description of the cat's status"},
        {"time", telem::TimeStamp(telem::SECOND).nanoseconds()},
        {"details", {{"field1", "test"}, {"field2", 123}}},
        {"labels", json::array()}
    };
    xjson::Parser p(j);
    const auto stat = synnax::status::Status<TestDetails>::parse(p);
    ASSERT_EQ(stat.key, "cat");
    ASSERT_EQ(stat.variant, synnax::status::VARIANT_ERROR);
    ASSERT_EQ(stat.message, "the cat is angry");
    ASSERT_EQ(stat.description, "a longer description of the cat's status");
    ASSERT_EQ(stat.time, telem::TimeStamp(telem::SECOND));
}

TEST(StatusTest, TestProtobufRoundTrip) {
    synnax::status::Status<json> original{
        .key = "test-key",
        .name = "Test Status",
        .message = "test message",
        .description = "test description",
        .time = telem::TimeStamp::now(),
        .details = json{{"field1", "hello"}, {"field2", 42}},
        .labels = {},
        .variant = synnax::status::VARIANT_INFO,
    };

    auto pb = original.to_proto();

    ASSERT_EQ(pb.key(), "test-key");
    ASSERT_EQ(pb.name(), "Test Status");
    ASSERT_EQ(pb.message(), "test message");
}
