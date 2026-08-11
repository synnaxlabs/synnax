// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/json/any.h"
#include "x/cpp/json/testutil/testutil.h"
#include "x/cpp/test/test.h"

namespace x::json {
TEST(ToAny, ObjectRoundTrip) {
    const json j = {{"name", "sensor"}, {"port", 8080}};
    const auto any = ASSERT_NIL_P(to_any(j));
    ASSERT_FALSE(any.type_url().empty());
    const auto result = ASSERT_NIL_P(from_any(any));
    ASSERT_EQ(result["name"], "sensor");
    ASSERT_EQ(result["port"], 8080);
}

TEST(ToAny, NullRoundTrip) {
    const auto any = ASSERT_NIL_P(to_any(json(nullptr)));
    ASSERT_TRUE(ASSERT_NIL_P(from_any(any)).is_null());
}

TEST(ToAny, NumberRoundTrip) {
    const auto any = ASSERT_NIL_P(to_any(json(42)));
    ASSERT_EQ(ASSERT_NIL_P(from_any(any)), 42);
}

TEST(ToAny, StringRoundTrip) {
    const auto any = ASSERT_NIL_P(to_any(json("sensor")));
    ASSERT_EQ(ASSERT_NIL_P(from_any(any)), "sensor");
}

TEST(ToAny, BoolRoundTrip) {
    const auto any = ASSERT_NIL_P(to_any(json(true)));
    ASSERT_EQ(ASSERT_NIL_P(from_any(any)), true);
}

TEST(ToAny, ArrayRoundTrip) {
    const auto any = ASSERT_NIL_P(to_any(json::array({1, 2})));
    const auto result = ASSERT_NIL_P(from_any(any));
    ASSERT_TRUE(result.is_array());
    ASSERT_EQ(result[0], 1);
    ASSERT_EQ(result[1], 2);
}

TEST(ToAny, ValueConversionErrorPropagates) {
    ASSERT_OCCURRED_AS_P(to_any(deeply_nested_object()), errors::VALIDATION);
}

TEST(FromAny, EmptyAnyReturnsNull) {
    ASSERT_TRUE(ASSERT_NIL_P(from_any(google::protobuf::Any())).is_null());
}

TEST(FromAny, PackedStructUnpacks) {
    google::protobuf::Struct s;
    ASSERT_NIL(to_struct(json{{"name", "sensor"}}, &s));
    google::protobuf::Any any;
    ASSERT_TRUE(any.PackFrom(s));
    ASSERT_EQ(ASSERT_NIL_P(from_any(any))["name"], "sensor");
}

TEST(FromAny, UnknownTypeReturnsError) {
    google::protobuf::Any any;
    any.set_type_url("type.googleapis.com/unknown.Type");
    ASSERT_OCCURRED_AS_P(from_any(any), errors::VALIDATION);
}

TEST(ToAny, NestedObjectRoundTrip) {
    const json j = {{"outer", {{"inner", "value"}}}, {"list", {1, 2, 3}}};
    const auto any = ASSERT_NIL_P(to_any(j));
    const auto result = ASSERT_NIL_P(from_any(any));
    ASSERT_EQ(result["outer"]["inner"], "value");
    ASSERT_EQ(result["list"][0], 1);
    ASSERT_EQ(result["list"][1], 2);
    ASSERT_EQ(result["list"][2], 3);
}
}
