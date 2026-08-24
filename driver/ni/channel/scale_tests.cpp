// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "x/cpp/json/json.h"
#include "x/cpp/test/test.h"

#include "driver/ni/channel/scale.h"

namespace driver::ni::channel {
std::pair<std::unique_ptr<Scale>, x::errors::Error>
parse_test_scale(x::json::Parser &p) {
    auto child = p.child("scale");
    return channel::make_scale(::synnax::ni::parse_scale(child));
}

TEST(Scale, None) {
    const x::json::json j = {{"scale", {{"type", "none"}}}};
    x::json::Parser p(j);
    auto [scale, scale_err] = parse_test_scale(p);
    ASSERT_NIL(p.error());
    ASSERT_NIL(scale_err);
    ASSERT_NE(scale, nullptr);
    EXPECT_TRUE(scale->is_none());
}

TEST(Scale, Linear) {
    const x::json::json j = {
        {"scale",
         {{"type", "linear"},
          {"slope", 2.0},
          {"y_intercept", 1.0},
          {"pre_scaled_units", "Volts"},
          {"scaled_units", "Pascals"}}}
    };
    x::json::Parser p(j);
    auto [scale, scale_err] = parse_test_scale(p);
    ASSERT_NIL(p.error());
    ASSERT_NIL(scale_err);
    ASSERT_NE(scale, nullptr);
    EXPECT_FALSE(scale->is_none());

    auto *linear_scale = dynamic_cast<channel::LinearScale *>(scale.get());
    ASSERT_NE(linear_scale, nullptr);
    EXPECT_EQ(linear_scale->slope, 2.0);
    EXPECT_EQ(linear_scale->offset, 1.0);
    EXPECT_EQ(linear_scale->pre_scaled_units, DAQmx_Val_Volts);
    EXPECT_EQ(linear_scale->scaled_units, "Pascals");
}

TEST(Scale, Map) {
    const x::json::json j = {
        {"scale",
         {{"type", "map"},
          {"pre_scaled_min", 0.0},
          {"pre_scaled_max", 10.0},
          {"scaled_min", 0.0},
          {"scaled_max", 100.0},
          {"pre_scaled_units", "Volts"},
          {"scaled_units", "Pascals"}}}
    };
    x::json::Parser p(j);
    auto [scale, scale_err] = parse_test_scale(p);
    ASSERT_NIL(p.error());
    ASSERT_NIL(scale_err);
    EXPECT_FALSE(scale->is_none());

    auto *map_scale = dynamic_cast<channel::MapScale *>(scale.get());
    ASSERT_NE(map_scale, nullptr);
    EXPECT_EQ(map_scale->pre_scaled_min, 0.0);
    EXPECT_EQ(map_scale->pre_scaled_max, 10.0);
    EXPECT_EQ(map_scale->scaled_min, 0.0);
    EXPECT_EQ(map_scale->scaled_max, 100.0);
    EXPECT_EQ(map_scale->pre_scaled_units, DAQmx_Val_Volts);
    EXPECT_EQ(map_scale->scaled_units, "Pascals");
}

TEST(Scale, Polynomial) {
    const x::json::json j = {
        {"scale",
         {{"type", "polynomial"},
          {"forward_coeffs", {1.0, 2.0, 3.0}},
          {"reverse_coeffs", {3.0, 2.0, 1.0}},
          {"pre_scaled_units", "Volts"},
          {"scaled_units", "Pascals"}}}
    };
    x::json::Parser p(j);
    auto [scale, scale_err] = parse_test_scale(p);
    ASSERT_NIL(p.error());
    ASSERT_NIL(scale_err);
    ASSERT_NE(scale, nullptr);
    EXPECT_FALSE(scale->is_none());

    auto *poly_scale = dynamic_cast<channel::PolynomialScale *>(scale.get());
    ASSERT_NE(poly_scale, nullptr);
    EXPECT_EQ(poly_scale->pre_scaled_units, DAQmx_Val_Volts);
    EXPECT_EQ(poly_scale->scaled_units, "Pascals");

    ASSERT_EQ(poly_scale->forward_coeffs.size(), 3);
    EXPECT_EQ(poly_scale->forward_coeffs[0], 1.0);
    EXPECT_EQ(poly_scale->forward_coeffs[1], 2.0);
    EXPECT_EQ(poly_scale->forward_coeffs[2], 3.0);
    ASSERT_EQ(poly_scale->reverse_coeffs.size(), 3);
    EXPECT_EQ(poly_scale->reverse_coeffs[0], 3.0);
}

TEST(Scale, PolynomialMissingReverseCoeffs) {
    const x::json::json j = {
        {"scale",
         {{"type", "polynomial"},
          {"forward_coeffs", {1.0, 2.0, 3.0}},
          {"pre_scaled_units", "Volts"},
          {"scaled_units", "Pascals"}}}
    };
    x::json::Parser p(j);
    auto [scale, scale_err] = parse_test_scale(p);
    ASSERT_OCCURRED_AS(scale_err, x::errors::VALIDATION);
}

TEST(Scale, Table) {
    const x::json::json j = {
        {"scale",
         {{"type", "table"},
          {"pre_scaled_vals", {0.0, 5.0, 10.0}},
          {"scaled_vals", {0.0, 50.0, 100.0}},
          {"pre_scaled_units", "Volts"},
          {"scaled_units", "Pascals"}}}
    };
    x::json::Parser p(j);
    auto [scale, scale_err] = parse_test_scale(p);
    ASSERT_NIL(p.error());
    ASSERT_NIL(scale_err);
    ASSERT_NE(scale, nullptr);
    EXPECT_FALSE(scale->is_none());

    auto *table_scale = dynamic_cast<channel::TableScale *>(scale.get());
    ASSERT_NE(table_scale, nullptr);
    EXPECT_EQ(table_scale->pre_scaled.size(), 3);
    EXPECT_EQ(table_scale->scaled.size(), 3);
    EXPECT_EQ(table_scale->pre_scaled_units, DAQmx_Val_Volts);
    EXPECT_EQ(table_scale->scaled_units, "Pascals");
}

TEST(Scale, InvalidType) {
    const x::json::json j = {{"scale", {{"type", "invalid"}}}};
    x::json::Parser p(j);
    auto [scale, scale_err] = parse_test_scale(p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

TEST(Scale, MissingType) {
    const x::json::json j = {{"scale", {{"slope", 2.0}}}};
    x::json::Parser p(j);
    auto [scale, scale_err] = parse_test_scale(p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

TEST(Scale, DefaultUnits) {
    const x::json::json j = {
        {"scale", {{"type", "linear"}, {"slope", 2.0}, {"y_intercept", 1.0}}}
    };
    x::json::Parser p(j);
    auto [scale, scale_err] = parse_test_scale(p);
    ASSERT_NIL(p.error());
    ASSERT_NIL(scale_err);
    ASSERT_NE(scale, nullptr);
    EXPECT_FALSE(scale->is_none());

    auto *linear_scale = dynamic_cast<channel::LinearScale *>(scale.get());
    ASSERT_NE(linear_scale, nullptr);
    EXPECT_EQ(linear_scale->pre_scaled_units, DAQmx_Val_Volts);
    EXPECT_EQ(linear_scale->scaled_units, "Volts");
}
}
