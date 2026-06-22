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

} // namespace

TEST(ColorFromHex, SixCharWithHash) {
    auto [c, err] = from_hex("#ff0000");
    ASSERT_FALSE(err) << err.message();
    EXPECT_EQ(c.r, 255);
    EXPECT_EQ(c.g, 0);
    EXPECT_EQ(c.b, 0);
    EXPECT_DOUBLE_EQ(c.a, 1.0);
}

TEST(ColorFromHex, SixCharWithoutHash) {
    auto [c, err] = from_hex("00ff00");
    ASSERT_FALSE(err) << err.message();
    EXPECT_EQ(c.r, 0);
    EXPECT_EQ(c.g, 255);
    EXPECT_EQ(c.b, 0);
    EXPECT_DOUBLE_EQ(c.a, 1.0);
}

TEST(ColorFromHex, EightCharWithAlpha) {
    auto [c, err] = from_hex("#ff000080");
    ASSERT_FALSE(err) << err.message();
    EXPECT_EQ(c.r, 255);
    EXPECT_EQ(c.g, 0);
    EXPECT_EQ(c.b, 0);
    EXPECT_NEAR(c.a, 128.0 / 255.0, 0.01);
}

TEST(ColorFromHex, InvalidHexDigitsErrors) {
    EXPECT_TRUE(from_hex("#xyz").second);
    EXPECT_TRUE(from_hex("#gggggg").second);
}

TEST(ColorFromHex, WrongLengthErrors) {
    EXPECT_TRUE(from_hex("#12345").second);
    EXPECT_TRUE(from_hex("#fff").second);
    EXPECT_TRUE(from_hex("").second);
}

TEST(ColorFromCSS, ParsesValidStrings) {
    expect_css("#ff0000", 255, 0, 0, 1.0);
    expect_css("rgb(59,196,84)", 59, 196, 84, 1.0);
    expect_css("rgb(255, 0, 0)", 255, 0, 0, 1.0);
    expect_css("rgba(59,196,84,0.5)", 59, 196, 84, 0.5);
    expect_css("rgba(0,0,0,1)", 0, 0, 0, 1.0);
    expect_css("  rgb(1,2,3)  ", 1, 2, 3, 1.0);
}

TEST(ColorFromCSS, RejectsInvalidStrings) {
    expect_css_error("rgb(1,2,3,0.5)", "rgb() takes 3 channels");
    expect_css_error("rgba(1,2,3)", "rgba() requires");
    expect_css_error("rgba(1,2,3,1.5)", "alpha must be 0-1");
    expect_css_error("rgb(300,0,0)", "channels must be 0-255");
    expect_css_error("00ff00", "must be a hex value");
    expect_css_error("red", "must be a hex value");
    expect_css_error("rgb(1,2)", "must be a hex value");
    expect_css_error("", "must be a hex value");
}
}
