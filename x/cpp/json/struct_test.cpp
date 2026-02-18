// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "google/protobuf/struct.pb.h"
#include "x/cpp/json/struct.h"
#include "x/cpp/test/test.h"

namespace x::json {
/// @brief it should convert an empty protobuf Struct to an empty JSON object.
TEST(FromStruct, EmptyStruct) {
    google::protobuf::Struct pb;
    const auto j = ASSERT_NIL_P(from_struct(pb));
    ASSERT_TRUE(j.is_object());
    ASSERT_TRUE(j.empty());
}

/// @brief it should convert a Struct with string values to JSON.
TEST(FromStruct, StringValues) {
    google::protobuf::Struct pb;
    auto &fields = *pb.mutable_fields();
    fields["name"].set_string_value("sensor");
    fields["location"].set_string_value("lab");
    const auto j = ASSERT_NIL_P(from_struct(pb));
    ASSERT_EQ(j["name"], "sensor");
    ASSERT_EQ(j["location"], "lab");
}

/// @brief it should convert a Struct with numeric values to JSON.
TEST(FromStruct, NumericValues) {
    google::protobuf::Struct pb;
    auto &fields = *pb.mutable_fields();
    fields["port"].set_number_value(8080);
    fields["threshold"].set_number_value(3.14);
    const auto j = ASSERT_NIL_P(from_struct(pb));
    ASSERT_EQ(j["port"], 8080);
    ASSERT_DOUBLE_EQ(j["threshold"].get<double>(), 3.14);
}

/// @brief it should convert a Struct with boolean values to JSON.
TEST(FromStruct, BooleanValues) {
    google::protobuf::Struct pb;
    auto &fields = *pb.mutable_fields();
    fields["enabled"].set_bool_value(true);
    fields["debug"].set_bool_value(false);
    const auto j = ASSERT_NIL_P(from_struct(pb));
    ASSERT_EQ(j["enabled"], true);
    ASSERT_EQ(j["debug"], false);
}

/// @brief it should convert a Struct with null values to JSON.
TEST(FromStruct, NullValue) {
    google::protobuf::Struct pb;
    auto &fields = *pb.mutable_fields();
    fields["empty"].set_null_value(google::protobuf::NULL_VALUE);
    const auto j = ASSERT_NIL_P(from_struct(pb));
    ASSERT_TRUE(j["empty"].is_null());
}

/// @brief it should convert a Struct with nested object values to JSON.
TEST(FromStruct, NestedStruct) {
    google::protobuf::Struct inner;
    auto &inner_fields = *inner.mutable_fields();
    inner_fields["host"].set_string_value("localhost");
    inner_fields["port"].set_number_value(9090);

    google::protobuf::Struct pb;
    auto &fields = *pb.mutable_fields();
    fields["name"].set_string_value("device");
    *fields["connection"].mutable_struct_value() = inner;

    const auto j = ASSERT_NIL_P(from_struct(pb));
    ASSERT_EQ(j["name"], "device");
    ASSERT_EQ(j["connection"]["host"], "localhost");
    ASSERT_EQ(j["connection"]["port"], 9090);
}

/// @brief it should convert a Struct with list values to JSON.
TEST(FromStruct, ListValue) {
    google::protobuf::Struct pb;
    auto &fields = *pb.mutable_fields();
    auto *list = fields["tags"].mutable_list_value();
    list->add_values()->set_string_value("sensor");
    list->add_values()->set_string_value("active");
    list->add_values()->set_number_value(42);

    const auto j = ASSERT_NIL_P(from_struct(pb));
    ASSERT_TRUE(j["tags"].is_array());
    ASSERT_EQ(j["tags"].size(), 3);
    ASSERT_EQ(j["tags"][0], "sensor");
    ASSERT_EQ(j["tags"][1], "active");
    ASSERT_EQ(j["tags"][2], 42);
}

/// @brief it should convert a Struct with mixed value types to JSON.
TEST(FromStruct, MixedTypes) {
    google::protobuf::Struct pb;
    auto &fields = *pb.mutable_fields();
    fields["name"].set_string_value("test");
    fields["count"].set_number_value(5);
    fields["active"].set_bool_value(true);
    fields["data"].set_null_value(google::protobuf::NULL_VALUE);

    const auto j = ASSERT_NIL_P(from_struct(pb));
    ASSERT_EQ(j["name"], "test");
    ASSERT_EQ(j["count"], 5);
    ASSERT_EQ(j["active"], true);
    ASSERT_TRUE(j["data"].is_null());
}

/// @brief it should round-trip an empty JSON object through to_struct and from_struct.
TEST(ToStruct, EmptyObject) {
    const json j = json::object();
    const auto pb = ASSERT_NIL_P(to_struct(j));
    ASSERT_EQ(pb.fields_size(), 0);
}

/// @brief it should convert a JSON object with string values to a Struct.
TEST(ToStruct, StringValues) {
    const json j = {{"name", "sensor"}, {"location", "lab"}};
    const auto pb = ASSERT_NIL_P(to_struct(j));
    ASSERT_EQ(pb.fields().at("name").string_value(), "sensor");
    ASSERT_EQ(pb.fields().at("location").string_value(), "lab");
}

/// @brief it should convert a JSON object with numeric values to a Struct.
TEST(ToStruct, NumericValues) {
    const json j = {{"port", 8080}, {"threshold", 3.14}};
    const auto pb = ASSERT_NIL_P(to_struct(j));
    ASSERT_EQ(pb.fields().at("port").number_value(), 8080);
    ASSERT_DOUBLE_EQ(pb.fields().at("threshold").number_value(), 3.14);
}

/// @brief it should convert a JSON object with boolean values to a Struct.
TEST(ToStruct, BooleanValues) {
    const json j = {{"enabled", true}, {"debug", false}};
    const auto pb = ASSERT_NIL_P(to_struct(j));
    ASSERT_EQ(pb.fields().at("enabled").bool_value(), true);
    ASSERT_EQ(pb.fields().at("debug").bool_value(), false);
}

/// @brief it should convert a JSON object with null values to a Struct.
TEST(ToStruct, NullValue) {
    const json j = {{"empty", nullptr}};
    const auto pb = ASSERT_NIL_P(to_struct(j));
    ASSERT_EQ(
        pb.fields().at("empty").kind_case(),
        google::protobuf::Value::kNullValue
    );
}

/// @brief it should convert a JSON object with nested objects to a Struct.
TEST(ToStruct, NestedObject) {
    const json j = {
        {"name", "device"},
        {"connection", {{"host", "localhost"}, {"port", 9090}}}
    };
    const auto pb = ASSERT_NIL_P(to_struct(j));
    ASSERT_EQ(pb.fields().at("name").string_value(), "device");
    const auto &inner = pb.fields().at("connection").struct_value();
    ASSERT_EQ(inner.fields().at("host").string_value(), "localhost");
    ASSERT_EQ(inner.fields().at("port").number_value(), 9090);
}

/// @brief it should convert a JSON object with arrays to a Struct.
TEST(ToStruct, ArrayValues) {
    const json j = {{"tags", {"sensor", "active"}}};
    const auto pb = ASSERT_NIL_P(to_struct(j));
    const auto &list = pb.fields().at("tags").list_value();
    ASSERT_EQ(list.values_size(), 2);
    ASSERT_EQ(list.values(0).string_value(), "sensor");
    ASSERT_EQ(list.values(1).string_value(), "active");
}

/// @brief it should return a validation error for non-object JSON input.
TEST(ToStruct, NonObjectError) {
    const json j = "not an object";
    auto [pb, err] = to_struct(j);
    ASSERT_TRUE(err);
}

/// @brief it should populate a Struct pointer from a JSON object.
TEST(ToStructPointer, PopulatesStruct) {
    const json j = {{"name", "sensor"}, {"port", 8080}};
    google::protobuf::Struct pb;
    ASSERT_NIL(to_struct(j, &pb));
    ASSERT_EQ(pb.fields().at("name").string_value(), "sensor");
    ASSERT_EQ(pb.fields().at("port").number_value(), 8080);
}

/// @brief it should return a validation error for non-object JSON via pointer overload.
TEST(ToStructPointer, NonObjectError) {
    const json j = json::array({1, 2, 3});
    google::protobuf::Struct pb;
    const auto err = to_struct(j, &pb);
    ASSERT_TRUE(err);
}

/// @brief it should round-trip a complex JSON object through to_struct and from_struct.
TEST(RoundTrip, ComplexObject) {
    const json original = {
        {"name", "test_device"},
        {"port", 8080},
        {"enabled", true},
        {"tags", {"sensor", "active"}},
        {"config", {{"timeout", 30}, {"retries", 3}}}
    };
    const auto pb = ASSERT_NIL_P(to_struct(original));
    const auto result = ASSERT_NIL_P(from_struct(pb));
    ASSERT_EQ(result["name"], original["name"]);
    ASSERT_EQ(result["port"], original["port"]);
    ASSERT_EQ(result["enabled"], original["enabled"]);
    ASSERT_EQ(result["tags"], original["tags"]);
    ASSERT_EQ(result["config"]["timeout"], original["config"]["timeout"]);
    ASSERT_EQ(result["config"]["retries"], original["config"]["retries"]);
}

/// @brief it should round-trip an empty object through to_struct and from_struct.
TEST(RoundTrip, EmptyObject) {
    const json original = json::object();
    const auto pb = ASSERT_NIL_P(to_struct(original));
    const auto result = ASSERT_NIL_P(from_struct(pb));
    ASSERT_TRUE(result.is_object());
    ASSERT_TRUE(result.empty());
}
}
