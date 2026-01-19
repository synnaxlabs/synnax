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

/// @brief it should correctly convert a status to its JSON representation.
TEST(StatusTest, TestToJSON) {
    const status::Status<> stat{
        .key = "dog",
        .variant = status::variant::SUCCESS,
        .message = "the dog is happy",
        .description = "a longer description of the dog's status",
        .time = telem::TimeStamp::now(),
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
    const auto stat = status::Status<>::parse(p);
    ASSERT_EQ(stat.key, "cat");
    ASSERT_EQ(stat.variant, status::variant::ERR);
    ASSERT_EQ(stat.message, "the cat is angry");
    ASSERT_EQ(stat.description, "a longer description of the cat's status");
    ASSERT_EQ(stat.time, telem::TimeStamp(telem::SECOND));
}

/// @brief custom details type for testing protobuf serialization.
struct TestDetails {
    std::string field1;
    int field2 = 0;

    [[nodiscard]] json to_json() const {
        return json{{"field1", field1}, {"field2", field2}};
    }

    static TestDetails parse(xjson::Parser &parser) {
        return TestDetails{
            .field1 = parser.field<std::string>("field1", ""),
            .field2 = parser.field<int>("field2", 0),
        };
    }
};

/// @brief it should correctly serialize and deserialize details to/from protobuf.
TEST(StatusTest, TestProtobufDetailsRoundTrip) {
    status::Status<TestDetails> original{
        .key = "test-key",
        .name = "Test Status",
        .variant = status::variant::INFO,
        .message = "test message",
        .description = "test description",
        .time = telem::TimeStamp::now(),
        .details = TestDetails{.field1 = "hello", .field2 = 42},
    };

    // Convert to protobuf
    status::PBStatus pb;
    original.to_proto(&pb);

    // Verify details field is set as JSON string
    ASSERT_FALSE(pb.details().empty());
    auto details_json = json::parse(pb.details());
    ASSERT_EQ(details_json["field1"], "hello");
    ASSERT_EQ(details_json["field2"], 42);

    // Convert back from protobuf
    auto recovered = ASSERT_NIL_P(status::Status<TestDetails>::from_proto(pb));

    // Verify all fields match
    ASSERT_EQ(recovered.key, original.key);
    ASSERT_EQ(recovered.name, original.name);
    ASSERT_EQ(recovered.variant, original.variant);
    ASSERT_EQ(recovered.message, original.message);
    ASSERT_EQ(recovered.description, original.description);
    ASSERT_EQ(recovered.time, original.time);
    ASSERT_EQ(recovered.details.field1, "hello");
    ASSERT_EQ(recovered.details.field2, 42);
}

/// @brief it should correctly identify a zero/default status.
TEST(StatusTest, TestIsZero) {
    // Default-constructed status should be zero
    const status::Status<> zero_status{};
    ASSERT_TRUE(zero_status.is_zero());

    // Status with any non-default field should not be zero
    const status::Status<> with_key{.key = "test"};
    ASSERT_FALSE(with_key.is_zero());

    const status::Status<> with_name{.name = "Test"};
    ASSERT_FALSE(with_name.is_zero());

    const status::Status<> with_variant{.variant = status::variant::SUCCESS};
    ASSERT_FALSE(with_variant.is_zero());

    const status::Status<> with_message{.message = "hello"};
    ASSERT_FALSE(with_message.is_zero());

    const status::Status<> with_description{.description = "desc"};
    ASSERT_FALSE(with_description.is_zero());

    const status::Status<> with_time{.time = telem::TimeStamp(1)};
    ASSERT_FALSE(with_time.is_zero());

    // Fully populated status should not be zero
    const status::Status<> full_status{
        .key = "key",
        .name = "name",
        .variant = status::variant::INFO,
        .message = "msg",
        .description = "desc",
        .time = telem::TimeStamp::now(),
    };
    ASSERT_FALSE(full_status.is_zero());
}
