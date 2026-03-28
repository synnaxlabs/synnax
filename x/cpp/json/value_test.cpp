// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "google/protobuf/struct.pb.h"
#include "gtest/gtest.h"

#include "x/cpp/json/value.h"
#include "x/cpp/test/test.h"

namespace x::json {
TEST(FromValue, StringValue) {
    google::protobuf::Value pb;
    pb.set_string_value("hello");
    const auto j = ASSERT_NIL_P(from_value(pb));
    ASSERT_EQ(j, "hello");
}

TEST(FromValue, NumberValue) {
    google::protobuf::Value pb;
    pb.set_number_value(42.5);
    const auto j = ASSERT_NIL_P(from_value(pb));
    ASSERT_DOUBLE_EQ(j.get<double>(), 42.5);
}

TEST(FromValue, BoolValue) {
    google::protobuf::Value pb;
    pb.set_bool_value(true);
    const auto j = ASSERT_NIL_P(from_value(pb));
    ASSERT_EQ(j, true);
}

TEST(FromValue, NullValue) {
    google::protobuf::Value pb;
    pb.set_null_value(google::protobuf::NULL_VALUE);
    const auto j = ASSERT_NIL_P(from_value(pb));
    ASSERT_TRUE(j.is_null());
}

TEST(ToValue, StringRoundTrip) {
    const json j = "test_string";
    const auto [pb, err] = to_value(j);
    ASSERT_NIL(err);
    const auto result = ASSERT_NIL_P(from_value(pb));
    ASSERT_EQ(result, "test_string");
}

TEST(ToValue, NumberRoundTrip) {
    const json j = 99.9;
    const auto [pb, err] = to_value(j);
    ASSERT_NIL(err);
    const auto result = ASSERT_NIL_P(from_value(pb));
    ASSERT_DOUBLE_EQ(result.get<double>(), 99.9);
}

TEST(ToValue, ObjectRoundTrip) {
    const json j = {{"key", "value"}, {"num", 123}};
    const auto [pb, err] = to_value(j);
    ASSERT_NIL(err);
    const auto result = ASSERT_NIL_P(from_value(pb));
    ASSERT_EQ(result["key"], "value");
    ASSERT_EQ(result["num"], 123);
}

TEST(ToValuePointer, PopulatesExistingValue) {
    const json j = "populated";
    google::protobuf::Value pb;
    ASSERT_NIL(to_value(j, &pb));
    const auto result = ASSERT_NIL_P(from_value(pb));
    ASSERT_EQ(result, "populated");
}
}
