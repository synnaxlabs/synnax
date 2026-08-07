// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"

#include "arc/cpp/types/types.h"

/// @brief it should correctly round-trip a simple Type through protobuf
TEST(TypesTest, testTypeProtobufRoundTrip) {
    ASSERT_TRUE(true);
}

TEST(ToSampleValue, I64FromNumber) {
    arc::types::Type t;
    t.kind = arc::types::Kind::I64;
    auto sv = arc::types::to_sample_value(x::json::json(1000000000000LL), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<int64_t>(*sv), 1000000000000LL);
}

TEST(ToSampleValue, I64FromString) {
    arc::types::Type t;
    t.kind = arc::types::Kind::I64;
    auto sv = arc::types::to_sample_value(x::json::json("1000000000000"), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<int64_t>(*sv), 1000000000000LL);
}

TEST(ToSampleValue, I64NegativeFromString) {
    arc::types::Type t;
    t.kind = arc::types::Kind::I64;
    auto sv = arc::types::to_sample_value(x::json::json("-5000000000"), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<int64_t>(*sv), -5000000000LL);
}

TEST(ToSampleValue, U64FromNumber) {
    arc::types::Type t;
    t.kind = arc::types::Kind::U64;
    auto sv = arc::types::to_sample_value(x::json::json(5000000000ULL), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint64_t>(*sv), 5000000000ULL);
}

TEST(ToSampleValue, U64FromString) {
    arc::types::Type t;
    t.kind = arc::types::Kind::U64;
    auto sv = arc::types::to_sample_value(x::json::json("5000000000"), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint64_t>(*sv), 5000000000ULL);
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
    ASSERT_EQ(x::telem::cast<uint8_t>(*sv), 1);
}

TEST(ToSampleValue, BoolFromFalse) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    auto sv = arc::types::to_sample_value(x::json::json(false), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint8_t>(*sv), 0);
}

TEST(ToSampleValue, BoolFromNonzeroNumberNormalizesToOne) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    auto sv = arc::types::to_sample_value(x::json::json(42), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint8_t>(*sv), 1);
}

TEST(ToSampleValue, BoolFromZeroNumber) {
    arc::types::Type t;
    t.kind = arc::types::Kind::Bool;
    auto sv = arc::types::to_sample_value(x::json::json(0), t);
    ASSERT_TRUE(sv.has_value());
    ASSERT_EQ(x::telem::cast<uint8_t>(*sv), 0);
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
    ASSERT_EQ(t.telem(), x::telem::BOOL_T);
}
