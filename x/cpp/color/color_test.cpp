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
#include "x/cpp/testutil/testutil.h"

namespace x::color {
namespace {

void expect_css(
    const std::string &input,
    std::uint8_t r,
    std::uint8_t g,
    std::uint8_t b,
    std::uint8_t a
) {
    SCOPED_TRACE(input);
    const auto c = ASSERT_NIL_P(from_css(input));
    EXPECT_EQ(c.r, r);
    EXPECT_EQ(c.g, g);
    EXPECT_EQ(c.b, b);
    EXPECT_EQ(c.a, a);
}

void expect_css_error(const std::string &input, const std::string &msg) {
    SCOPED_TRACE(input);
    auto [c, err] = from_css(input);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(err.message().find(msg), std::string::npos) << err.message();
}

Color parse_json(const x::json::json &j) {
    SCOPED_TRACE(j.dump());
    x::json::Parser parser(j);
    const auto c = Color::parse(parser);
    EXPECT_NIL(parser.error());
    return c;
}

void expect_parsed(
    const x::json::json &j,
    std::uint8_t r,
    std::uint8_t g,
    std::uint8_t b,
    std::uint8_t a
) {
    const auto c = parse_json(j);
    EXPECT_EQ(c.r, r) << j.dump();
    EXPECT_EQ(c.g, g) << j.dump();
    EXPECT_EQ(c.b, b) << j.dump();
    EXPECT_EQ(c.a, a) << j.dump();
}
}

TEST(ColorFromCSS, ParsesValidStrings) {
    expect_css("#ff0000", 255, 0, 0, 255);
    expect_css("#ff000080", 255, 0, 0, 128);
    expect_css("rgb(59,196,84)", 59, 196, 84, 255);
    expect_css("rgb(255, 0, 0)", 255, 0, 0, 255);
    expect_css("rgba(59,196,84,0.5)", 59, 196, 84, 128);
    expect_css("rgba(0,0,0,1)", 0, 0, 0, 255);
    expect_css("rgba(0,0,0,0)", 0, 0, 0, 0);
    expect_css("  rgb(1,2,3)  ", 1, 2, 3, 255);
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

TEST(ColorParse, ParsesHexString) {
    expect_parsed("#ff8000", 255, 128, 0, 255);
    expect_parsed("#ff000080", 255, 0, 0, 128);
    expect_parsed("#00000000", 0, 0, 0, 0);
}

/// @brief it should default to opaque black.
TEST(Color, DefaultsToOpaqueBlack) {
    EXPECT_EQ(Color{}.hex(), "#000000ff");
}

/// @brief it should reject anything but a hex string.
TEST(ColorParse, RejectsNonHexInput) {
    for (const auto &j:
         {x::json::json(nullptr),
          x::json::json(""),
          x::json::json::object(),
          x::json::json::array({122, 44, 38, 0.5}),
          x::json::json({{"r", 255}, {"g", 128}, {"b", 0}, {"a", 0.5}}),
          x::json::json({{"rgba255", {122, 44, 38, 0.5}}})}) {
        SCOPED_TRACE(j.dump());
        x::json::Parser parser(j);
        Color::parse(parser);
        EXPECT_FALSE(parser.ok());
    }
    x::json::Parser bad_hex(x::json::json("#xyz"));
    Color::parse(bad_hex);
    EXPECT_FALSE(bad_hex.ok());
    EXPECT_EQ(bad_hex.errors->at(0)["message"], "invalid hex color length: #xyz");
}

/// @brief it should write a color as an eight digit hex string.
TEST(ColorJSON, WritesAnEightDigitHexString) {
    const Color c{.r = 255, .g = 128, .b = 1, .a = 64};
    EXPECT_EQ(c.to_json(), x::json::json("#ff800140"));
    EXPECT_EQ(Color{.a = 0}.to_json(), x::json::json("#00000000"));
}

/// @brief it should round trip every alpha through JSON.
TEST(ColorJSON, RoundTripsEveryAlpha) {
    for (int a = 0; a < 256; a++) {
        const Color c{.r = 100, .g = 200, .b = 50, .a = static_cast<std::uint8_t>(a)};
        expect_parsed(c.to_json(), 100, 200, 50, static_cast<std::uint8_t>(a));
    }
}

/// @brief it should pack a color into one proto field.
TEST(ColorProto, PacksTheColorIntoOneField) {
    const Color c{.r = 0x12, .g = 0x34, .b = 0x56, .a = 0x78};
    const auto pb = ASSERT_NIL_P(c.to_proto());
    EXPECT_EQ(pb.rgba(), 0x12345678u);
}

/// @brief it should round trip every alpha through the proto wire form.
TEST(ColorProto, RoundTripsEveryAlphaThroughTheWire) {
    for (int a = 0; a < 256; a++) {
        const Color c{.r = 10, .g = 20, .b = 30, .a = static_cast<std::uint8_t>(a)};
        const auto pb = ASSERT_NIL_P(c.to_proto());
        ::x::color::pb::Color decoded_pb;
        ASSERT_TRUE(decoded_pb.ParseFromString(pb.SerializeAsString()));
        const auto decoded = ASSERT_NIL_P(Color::from_proto(decoded_pb));
        EXPECT_EQ(decoded.rgba(), c.rgba());
    }
}
}
