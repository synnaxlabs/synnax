// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/color/color.h"
#include "x/cpp/testutil/testutil.h"

namespace x::color {

struct ParseCase {
    std::string input;
    Color expected;
};

/// @brief it should parse every form of the color string grammar.
TEST(Parse, ParsesEveryForm) {
    const std::vector<ParseCase> cases = {
        {"#ff8000", {255, 128, 0, 255}},
        {"ff8000", {255, 128, 0, 255}},
        {"#FF8000", {255, 128, 0, 255}},
        {"#ff000080", {255, 0, 0, 128}},
        {"#00000000", {0, 0, 0, 0}},
        {"#f80", {255, 136, 0, 255}},
        {"#f808", {255, 136, 0, 136}},
        {"  #f80  ", {255, 136, 0, 255}},
        {"rgb(59, 196, 84)", {59, 196, 84, 255}},
        {"rgb(59,196,84)", {59, 196, 84, 255}},
        {"rgba(59, 196, 84)", {59, 196, 84, 255}},
        {"rgb(59, 196, 84, 0.5)", {59, 196, 84, 128}},
        {"rgba(59, 196, 84, 0.5)", {59, 196, 84, 128}},
        {"rgba(0, 0, 0, 0)", {0, 0, 0, 0}},
        {"rgb(59 196 84)", {59, 196, 84, 255}},
        {"rgb(59 196 84 / 0.5)", {59, 196, 84, 128}},
        {"rgb(59 196 84/50%)", {59, 196, 84, 128}},
        {"rgb(100%, 0%, 50%)", {255, 0, 128, 255}},
        {"RGB(1, 2, 3)", {1, 2, 3, 255}},
        {"hsl(0, 100%, 50%)", {255, 0, 0, 255}},
        {"hsl(120, 100, 50)", {0, 255, 0, 255}},
        {"hsla(240, 100%, 50%, 0.25)", {0, 0, 255, 64}},
        {"hsl(240 100% 50% / 25%)", {0, 0, 255, 64}},
        {"hsl(360deg 100% 50%)", {255, 0, 0, 255}},
        {"hsl(-120, 100%, 50%)", {0, 0, 255, 255}},
        {"hsl(0, 0%, 100%)", {255, 255, 255, 255}},
    };
    for (const auto &[input, expected]: cases) {
        SCOPED_TRACE(input);
        EXPECT_EQ(ASSERT_NIL_P(parse(input)), expected);
    }
}

/// @brief it should reject a string outside the grammar or a value out of range.
TEST(Parse, RejectsInvalidStrings) {
    for (const std::string input:
         {"",
          "   ",
          "red",
          "transparent",
          "#",
          "#12345",
          "#1234567",
          "#gggggg",
          "rgb(1, 2)",
          "rgb(1, 2, 3, 4, 5)",
          "rgb(1 2 3 0.5)",
          "rgb(1 2 / 3 / 4)",
          "rgb(1, 2, 3",
          "rgb(300, 0, 0)",
          "rgb(-1, 0, 0)",
          "rgb(101%, 0%, 0%)",
          "rgba(1, 2, 3, 1.5)",
          "rgba(1, 2, 3, -0.5)",
          "rgb(1, 2, x)",
          "rgb(1px, 2, 3)",
          "hsl(0, 101%, 50%)",
          "hsl(0, 100%, -1%)",
          "hsl(0rad, 100%, 50%)",
          "hwb(0, 0%, 0%)"}) {
        SCOPED_TRACE(input);
        ASSERT_OCCURRED_AS_P(parse(input), x::errors::VALIDATION);
    }
}

/// @brief it should default to opaque black.
TEST(Color, DefaultsToOpaqueBlack) {
    EXPECT_EQ(Color{}, (Color{0, 0, 0, 255}));
}

/// @brief it should write six digits for an opaque color and eight otherwise.
TEST(Color, WritesShortHexWhenOpaque) {
    EXPECT_EQ((Color{255, 128, 1, 255}.hex()), "#ff8001");
    EXPECT_EQ((Color{255, 128, 1, 64}.hex()), "#ff800140");
    EXPECT_EQ((Color{0, 0, 0, 0}.hex()), "#00000000");
    EXPECT_EQ((Color{255, 128, 1, 64}.to_json()), x::json::json("#ff800140"));
    EXPECT_EQ(Color{}.to_json(), x::json::json("#000000"));
}

/// @brief it should parse any string form from JSON.
TEST(Color, ParsesJSONStrings) {
    x::json::Parser hex(x::json::json("#ff800080"));
    EXPECT_EQ(Color::parse(hex), (Color{255, 128, 0, 128}));
    EXPECT_TRUE(hex.ok());
    x::json::Parser rgb(x::json::json("rgb(1, 2, 3)"));
    EXPECT_EQ(Color::parse(rgb), (Color{1, 2, 3, 255}));
    EXPECT_TRUE(rgb.ok());
}

/// @brief it should reject JSON that is not a color string.
TEST(Color, RejectsNonStringJSON) {
    for (const auto &j:
         {x::json::json(nullptr),
          x::json::json(""),
          x::json::json::object(),
          x::json::json::array({122, 44, 38, 0.5}),
          x::json::json({{"r", 255}, {"g", 128}, {"b", 0}, {"a", 0.5}})}) {
        SCOPED_TRACE(j.dump());
        x::json::Parser parser(j);
        Color::parse(parser);
        EXPECT_FALSE(parser.ok());
    }
}

/// @brief it should round trip every alpha through JSON and protobuf.
TEST(Color, RoundTripsEveryAlpha) {
    for (int a = 0; a < 256; a++) {
        const Color c{.r = 100, .g = 200, .b = 50, .a = static_cast<std::uint8_t>(a)};
        SCOPED_TRACE(c.hex());
        x::json::Parser parser(c.to_json());
        EXPECT_EQ(Color::parse(parser), c);
        EXPECT_TRUE(parser.ok());
        const auto pb = ASSERT_NIL_P(c.to_proto());
        ::x::color::pb::Color decoded_pb;
        ASSERT_TRUE(decoded_pb.ParseFromString(pb.SerializeAsString()));
        EXPECT_EQ(ASSERT_NIL_P(Color::from_proto(decoded_pb)), c);
    }
}

/// @brief it should pack the color into one proto field as 0xRRGGBBAA.
TEST(Color, PacksTheProtoColorIntoOneField) {
    const Color c{.r = 0x12, .g = 0x34, .b = 0x56, .a = 0x78};
    EXPECT_EQ(ASSERT_NIL_P(c.to_proto()).rgba(), 0x12345678u);
}
}
