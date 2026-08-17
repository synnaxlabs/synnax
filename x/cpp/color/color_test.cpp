// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"

#include "x/cpp/color/color.h"

namespace x::color {
namespace {

void expect_css(
    const std::string &input,
    std::uint8_t r,
    std::uint8_t g,
    std::uint8_t b,
    double a
) {
    auto [c, err] = from_css(input);
    ASSERT_FALSE(err) << input << ": " << err.message();
    EXPECT_EQ(c.r, r) << input;
    EXPECT_EQ(c.g, g) << input;
    EXPECT_EQ(c.b, b) << input;
    EXPECT_DOUBLE_EQ(c.a, a) << input;
}

void expect_css_error(const std::string &input, const std::string &msg) {
    auto [c, err] = from_css(input);
    ASSERT_TRUE(err) << input << ": expected error";
    EXPECT_NE(err.message().find(msg), std::string::npos)
        << input << ": got " << err.message();
}

Color parse_json(const x::json::json &j) {
    x::json::Parser parser(j);
    const auto c = Color::parse(parser);
    EXPECT_FALSE(parser.error()) << j.dump() << ": " << parser.error().message();
    return c;
}

void expect_parsed(
    const x::json::json &j,
    std::uint8_t r,
    std::uint8_t g,
    std::uint8_t b,
    double a
) {
    const auto c = parse_json(j);
    EXPECT_EQ(c.r, r) << j.dump();
    EXPECT_EQ(c.g, g) << j.dump();
    EXPECT_EQ(c.b, b) << j.dump();
    EXPECT_DOUBLE_EQ(c.a, a) << j.dump();
}
}

TEST(ColorFromCSS, ParsesValidStrings) {
    expect_css("#ff0000", 255, 0, 0, 1.0);
    expect_css("#ff000080", 255, 0, 0, 128.0 / 255.0);
    expect_css("rgb(59,196,84)", 59, 196, 84, 1.0);
    expect_css("rgb(255, 0, 0)", 255, 0, 0, 1.0);
    expect_css("rgba(59,196,84,0.5)", 59, 196, 84, 0.5);
    expect_css("rgba(0,0,0,1)", 0, 0, 0, 1.0);
    expect_css("  rgb(1,2,3)  ", 1, 2, 3, 1.0);
}

TEST(ColorFromCSS, RejectsInvalidStrings) {
    expect_css_error("#gggggg", "invalid hex");
    expect_css_error("#12345", "invalid hex");
    expect_css_error("#fff", "invalid hex");
    expect_css_error("rgb(1,2,3,0.5)", "rgb() takes 3 channels");
    expect_css_error("rgba(1,2,3)", "rgba() requires");
    expect_css_error("rgba(1,2,3,1.5)", "alpha must be 0-1");
    expect_css_error("rgb(300,0,0)", "channels must be 0-255");
    expect_css_error("00ff00", "must be a hex value");
    expect_css_error("red", "must be a hex value");
    expect_css_error("rgb(1,2)", "must be a hex value");
    expect_css_error("", "must be a hex value");
}

TEST(ColorParse, ParsesStructForm) {
    expect_parsed({{"r", 255}, {"g", 128}, {"b", 0}, {"a", 0.5}}, 255, 128, 0, 0.5);
}

TEST(ColorParse, ParsesHexString) {
    expect_parsed("#ff8000", 255, 128, 0, 1.0);
    expect_parsed("#ff000080", 255, 0, 0, 128.0 / 255.0);
}

TEST(ColorParse, ParsesArrays) {
    expect_parsed(x::json::json::array({122, 44, 38}), 122, 44, 38, 1.0);
    expect_parsed(x::json::json::array({122, 44, 38, 0.5}), 122, 44, 38, 0.5);
}

TEST(ColorParse, ParsesLegacyRGBA255Object) {
    expect_parsed({{"rgba255", {122, 44, 38, 0.5}}}, 122, 44, 38, 0.5);
}

TEST(ColorParse, ParsesNullAndEmptyStringAsZero) {
    expect_parsed(nullptr, 0, 0, 0, 0.0);
    expect_parsed("", 0, 0, 0, 0.0);
}

TEST(ColorParse, LiftsLegacyAlphaOntoUnitScale) {
    expect_parsed(x::json::json::array({28, 28, 28, 255}), 28, 28, 28, 1.0);
    expect_parsed(x::json::json::array({255, 0, 0, 128}), 255, 0, 0, 128.0 / 255.0);
    expect_parsed(x::json::json::array({255, 0, 0, 1.5}), 255, 0, 0, 1.0);
    expect_parsed(x::json::json::array({255, 0, 0, 2}), 255, 0, 0, 1.0);
    expect_parsed(x::json::json::array({255, 0, 0, 2.01}), 255, 0, 0, 2.01 / 255.0);
    expect_parsed(x::json::json::array({255, 0, 0, 1}), 255, 0, 0, 1.0);
    expect_parsed({{"r", 5}, {"g", 5}, {"b", 5}, {"a", 255}}, 5, 5, 5, 1.0);
    expect_parsed({{"rgba255", {5, 5, 5, 255}}}, 5, 5, 5, 1.0);
}

TEST(ColorParse, AccumulatesErrorOnBadInput) {
    x::json::Parser bad_length(x::json::json::array({1, 2}));
    Color::parse(bad_length);
    EXPECT_FALSE(bad_length.ok());
    EXPECT_EQ(bad_length.errors->at(0)["message"], "invalid color array length: 2");
    x::json::Parser bad_element(x::json::json::array({1, 2, "x", 1}));
    Color::parse(bad_element);
    EXPECT_FALSE(bad_element.ok());
    EXPECT_EQ(
        bad_element.errors->at(0)["message"],
        "color array elements must be numbers"
    );
    x::json::Parser out_of_range(x::json::json::array({300, 0, 0}));
    Color::parse(out_of_range);
    EXPECT_FALSE(out_of_range.ok());
    EXPECT_EQ(
        out_of_range.errors->at(0)["message"],
        "color array elements must be within [0, 255]"
    );
    x::json::Parser alpha_above_scale({{"r", 5}, {"g", 5}, {"b", 5}, {"a", 300}});
    Color::parse(alpha_above_scale);
    EXPECT_FALSE(alpha_above_scale.ok());
    EXPECT_EQ(alpha_above_scale.errors->at(0)["path"], "a");
    EXPECT_EQ(
        alpha_above_scale.errors->at(0)["message"],
        "alpha is above the 0-255 scale"
    );
    x::json::Parser bad_hex(x::json::json("#xyz"));
    Color::parse(bad_hex);
    EXPECT_FALSE(bad_hex.ok());
    EXPECT_EQ(bad_hex.errors->at(0)["message"], "invalid hex color length: #xyz");
}

TEST(ColorParse, RoundTripsToJSON) {
    const Color original{.r = 100, .g = 200, .b = 50, .a = 0.75};
    expect_parsed(original.to_json(), 100, 200, 50, 0.75);
}
}
