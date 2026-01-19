// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "nlohmann/json.hpp"
#include <gtest/gtest.h>

#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xtest/xtest.h"

#include "driver/ni/channel/scale.h"

using json = nlohmann::json;

TEST(Scale, None) {
    const json j = {{"scale", {{"type", "none"}}}};
    const xjson::Parser p(j);
    const auto scale = channel::parse_scale(p, "scale");
    ASSERT_NIL(p.error());
    ASSERT_NE(scale, nullptr);
    EXPECT_TRUE(scale->is_none());
}

TEST(Scale, Linear) {
    const json j = {
        {"scale",
         {{"type", "linear"},
          {"slope", 2.0},
          {"y_intercept", 1.0},
          {"pre_scaled_units", "Volts"},
          {"scaled_units", "Pascals"}}}
    };
    const xjson::Parser p(j);
    const auto scale = channel::parse_scale(p, "scale");
    ASSERT_NIL(p.error());
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
    const json j = {
        {"scale",
         {{"type", "map"},
          {"pre_scaled_min", 0.0},
          {"pre_scaled_max", 10.0},
          {"scaled_min", 0.0},
          {"scaled_max", 100.0},
          {"pre_scaled_units", "Volts"},
          {"scaled_units", "Pascals"}}}
    };
    const xjson::Parser p(j);
    const auto scale = channel::parse_scale(p, "scale");
    ASSERT_NIL(p.error());
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
    const json j = {
        {"scale",
         {{"type", "polynomial"},
          {"forward_coeffs", {1.0, 2.0, 3.0}},
          {"min_x", 0.0},
          {"max_x", 10.0},
          {"pre_scaled_units", "Volts"},
          {"scaled_units", "Pascals"}}}
    };
    const xjson::Parser p(j);
    const auto scale = channel::parse_scale(p, "scale");
    ASSERT_NIL(p.error());
    ASSERT_NE(scale, nullptr);
    EXPECT_FALSE(scale->is_none());

    auto *poly_scale = dynamic_cast<channel::PolynomialScale *>(scale.get());
    ASSERT_NE(poly_scale, nullptr);
    EXPECT_EQ(poly_scale->min_x, 0.0);
    EXPECT_EQ(poly_scale->max_x, 10.0);
    EXPECT_EQ(
        poly_scale->reverse_poly_order,
        channel::REVERSE_POLY_ORDER_SAME_AS_FORWARD
    );
    EXPECT_EQ(poly_scale->pre_scaled_units, DAQmx_Val_Volts);
    EXPECT_EQ(poly_scale->scaled_units, "Pascals");

    ASSERT_EQ(poly_scale->forward_coeffs.size(), 3);
    EXPECT_EQ(poly_scale->forward_coeffs[0], 1.0);
    EXPECT_EQ(poly_scale->forward_coeffs[1], 2.0);
    EXPECT_EQ(poly_scale->forward_coeffs[2], 3.0);
}

TEST(Scale, Table) {
    const json j = {
        {"scale",
         {{"type", "table"},
          {"pre_scaled", {0.0, 5.0, 10.0}},
          {"scaled", {0.0, 50.0, 100.0}},
          {"pre_scaled_units", "Volts"},
          {"scaled_units", "Pascals"}}}
    };
    const xjson::Parser p(j);
    const auto scale = channel::parse_scale(p, "scale");
    ASSERT_NIL(p.error());
    ASSERT_NE(scale, nullptr);
    EXPECT_FALSE(scale->is_none());

    auto *table_scale = dynamic_cast<channel::TableScale *>(scale.get());
    ASSERT_NE(table_scale, nullptr);
    EXPECT_EQ(table_scale->pre_scaled_units, DAQmx_Val_Volts);
    EXPECT_EQ(table_scale->scaled_units, "Pascals");
}

TEST(Scale, InvalidType) {
    const json j = {{"scale", {{"type", "invalid"}}}};
    const xjson::Parser p(j);
    const auto ptr = channel::parse_scale(p, "scale");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
    ASSERT_EQ(ptr, nullptr);
}

TEST(Scale, MissingRequiredFields) {
    const json j = {{"scale", {{"type", "linear"}, {"y_intercept", 1.0}}}};
    const xjson::Parser p(j);
    const auto scale = channel::parse_scale(p, "scale");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

TEST(Scale, DefaultUnits) {
    const json j = {
        {"scale", {{"type", "linear"}, {"slope", 2.0}, {"y_intercept", 1.0}}}
    };
    const xjson::Parser p(j);
    const auto scale = channel::parse_scale(p, "scale");
    ASSERT_NIL(p.error());
    ASSERT_NE(scale, nullptr);
    EXPECT_FALSE(scale->is_none());

    auto *linear_scale = dynamic_cast<channel::LinearScale *>(scale.get());
    ASSERT_NE(linear_scale, nullptr);
    EXPECT_EQ(linear_scale->pre_scaled_units, DAQmx_Val_Volts);
    EXPECT_EQ(linear_scale->scaled_units, "Volts");
}
