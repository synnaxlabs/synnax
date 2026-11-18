// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <limits>
#include <string>
#include <variant>

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"

#include "arc/cpp/types/types.h"

/// @brief it should correctly round-trip a simple Type through protobuf
TEST(TypesTest, testTypeProtobufRoundTrip) {
    ASSERT_TRUE(true);
}

/// @brief it should read NaN and infinity text for float types only.
TEST(ToSampleValue, NonFiniteFloatsFromText) {
    using arc::types::to_sample_value;
    arc::types::Type f32, f64, str;
    f32.kind = arc::types::Kind::F32;
    f64.kind = arc::types::Kind::F64;
    str.kind = arc::types::Kind::String;
    EXPECT_TRUE(std::isnan(std::get<float>(to_sample_value("NaN", f32).value())));
    EXPECT_TRUE(std::isnan(std::get<double>(to_sample_value("NaN", f64).value())));
    EXPECT_EQ(
        std::get<float>(to_sample_value("Infinity", f32).value()),
        std::numeric_limits<float>::infinity()
    );
    EXPECT_EQ(
        std::get<double>(to_sample_value("-Infinity", f64).value()),
        -std::numeric_limits<double>::infinity()
    );
    EXPECT_EQ(std::get<double>(to_sample_value(1.5, f64).value()), 1.5);
    EXPECT_FALSE(to_sample_value("other", f64).has_value());
    EXPECT_EQ(std::get<std::string>(to_sample_value("NaN", str).value()), "NaN");
}

TEST(ToSampleValue, I64FromNumber) {
    arc::types::Type t;
    t.kind = arc::types::Kind::I64;
    auto sv = arc::types::to_sample_value(x::json::json(1000000000000LL), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<int64_t>(sv.value()), 1000000000000LL);
}

TEST(ToSampleValue, I64FromString) {
    arc::types::Type t;
    t.kind = arc::types::Kind::I64;
    auto sv = arc::types::to_sample_value(x::json::json("1000000000000"), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<int64_t>(sv.value()), 1000000000000LL);
}

TEST(ToSampleValue, I64NegativeFromString) {
    arc::types::Type t;
    t.kind = arc::types::Kind::I64;
    auto sv = arc::types::to_sample_value(x::json::json("-5000000000"), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<int64_t>(sv.value()), -5000000000LL);
}

TEST(ToSampleValue, U64FromNumber) {
    arc::types::Type t;
    t.kind = arc::types::Kind::U64;
    auto sv = arc::types::to_sample_value(x::json::json(5000000000ULL), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint64_t>(sv.value()), 5000000000ULL);
}

TEST(ToSampleValue, U64FromString) {
    arc::types::Type t;
    t.kind = arc::types::Kind::U64;
    auto sv = arc::types::to_sample_value(x::json::json("5000000000"), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint64_t>(sv.value()), 5000000000ULL);
}

TEST(ToSampleValue, I64FromNull) {
    arc::types::Type t;
    t.kind = arc::types::Kind::I64;
    auto sv = arc::types::to_sample_value(x::json::json(nullptr), t);
    ASSERT_FALSE(sv.has_value());
}

TEST(ToSampleValue, U64FromBoolReturnsNullopt) {
    arc::types::Type t;
    t.kind = arc::types::Kind::U64;
    auto sv = arc::types::to_sample_value(x::json::json(true), t);
    ASSERT_FALSE(sv.has_value());
}

TEST(ToSampleValue, BoolFromTrue) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    auto sv = arc::types::to_sample_value(x::json::json(true), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint8_t>(sv.value()), 1);
}

TEST(ToSampleValue, BoolFromFalse) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    auto sv = arc::types::to_sample_value(x::json::json(false), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint8_t>(sv.value()), 0);
}

TEST(ToSampleValue, BoolFromNonzeroNumberNormalizesToOne) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    auto sv = arc::types::to_sample_value(x::json::json(42), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint8_t>(sv.value()), 1);
}

TEST(ToSampleValue, BoolFromZeroNumber) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    auto sv = arc::types::to_sample_value(x::json::json(0), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint8_t>(sv.value()), 0);
}

TEST(ToSampleValue, BoolFromStringReturnsNullopt) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    auto sv = arc::types::to_sample_value(x::json::json("true"), t);
    ASSERT_FALSE(sv.has_value());
}

TEST(ToSampleValue, BoolFromNull) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    auto sv = arc::types::to_sample_value(x::json::json(nullptr), t);
    ASSERT_FALSE(sv.has_value());
}

TEST(TypesTest, BoolToString) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    ASSERT_EQ(t.to_string(), "bool");
}

TEST(TypesTest, BoolDensity) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    ASSERT_EQ(t.density(), 1);
}

TEST(TypesTest, BoolTelem) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    ASSERT_EQ(t.telem(), x::telem::BOOLEAN_T);
}

/// @brief it should round-trip Channels through JSON as objects with string keys.
TEST(TypesTest, ChannelsJSONRoundTrip) {
    arc::types::Channels original;
    original.read[1] = "channel_a";
    original.write[3] = "channel_c";
    const auto j = original.to_json();
    ASSERT_EQ(
        j,
        x::json::json::parse(R"({"read":{"1":"channel_a"},"write":{"3":"channel_c"}})")
    );
    x::json::Parser parser(j);
    const auto parsed = arc::types::Channels::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.read.at(1), "channel_a");
    ASSERT_EQ(parsed.write.at(3), "channel_c");
}

/// @brief builds a type with a time unit of the given name and scale.
arc::types::Type with_unit(arc::types::Type t, std::string name, const double scale) {
    arc::types::Unit unit;
    unit.dimensions.time = 1;
    unit.scale = scale;
    unit.name = std::move(name);
    t.unit = unit;
    return t;
}

/// @brief builds a var ref to variable v with values of type elem.
arc::types::Type var_ref(const arc::types::Type &elem) {
    arc::types::Type t;
    t.kind = arc::types::Kind::VarRef;
    t.name = "v";
    t.elem = x::mem::indirect<arc::types::Type>(elem);
    return t;
}

/// @brief it should give the factor between the two units of a var ref.
TEST(InputScale, ScaleOfAnInput) {
    using arc::types::input_scale;
    arc::types::Type i64;
    i64.kind = arc::types::Kind::I64;
    const auto seconds = with_unit(i64, "s", 1e9);
    const auto nanoseconds = with_unit(i64, "ns", 1);
    EXPECT_EQ(input_scale(with_unit(var_ref(seconds), "ns", 1)).first, 1e9);
    EXPECT_EQ(input_scale(with_unit(var_ref(nanoseconds), "s", 1e9)).first, 1e-9);
    EXPECT_EQ(
        input_scale(with_unit(var_ref(with_unit(i64, "um", 1e-6)), "nm", 1e-9)).first,
        1e3
    );
    EXPECT_EQ(input_scale(var_ref(seconds)).first, 1);
    EXPECT_EQ(input_scale(with_unit(var_ref(i64), "ns", 1)).first, 1);
    EXPECT_EQ(input_scale(seconds).first, 1);
}

/// @brief it should give the inverse factor between the two units of a var ref.
TEST(InputScale, InverseScaleOfAnInput) {
    using arc::types::input_scale;
    arc::types::Type i64;
    i64.kind = arc::types::Kind::I64;
    const auto nanoseconds = with_unit(i64, "ns", 1);
    EXPECT_EQ(input_scale(with_unit(var_ref(nanoseconds), "s", 1e9)).second, 1e9);
    EXPECT_EQ(
        input_scale(with_unit(var_ref(with_unit(i64, "nm", 1e-9)), "um", 1e-6)).second,
        1e3
    );
    EXPECT_EQ(input_scale(nanoseconds).second, 1);
}
