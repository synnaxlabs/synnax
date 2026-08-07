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

TEST(ToAny, NullConvertsToEmptyObject) {
    const json j = nullptr;
    const auto any = ASSERT_NIL_P(to_any(j));
    const auto result = ASSERT_NIL_P(from_any(any));
    ASSERT_TRUE(result.is_object());
    ASSERT_TRUE(result.empty());
}

TEST(ToAny, NonObjectReturnsError) {
    ASSERT_OCCURRED_AS_P(to_any(json(42)), errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(to_any(json("sensor")), errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(to_any(json(true)), errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(to_any(json::array({1, 2})), errors::VALIDATION);
}

TEST(ToAny, NonObjectErrorNamesTheType) {
    const auto err = to_any(json(42)).second;
    ASSERT_EQ(err.data, "expected a JSON object, got number");
}

TEST(ToAny, StructConversionErrorPropagates) {
    json j = json::object();
    for (int i = 0; i < 150; i++)
        j = json{{"nested", j}};
    ASSERT_OCCURRED_AS_P(to_any(j), errors::VALIDATION);
}

TEST(FromAny, EmptyAnyReturnsEmptyObject) {
    google::protobuf::Any any;
    const auto result = ASSERT_NIL_P(from_any(any));
    ASSERT_TRUE(result.is_object());
    ASSERT_TRUE(result.empty());
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
