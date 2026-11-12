// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "x/go/color/pb/color.pb.h"

namespace x::color {

/// @brief Color is an RGBA color with every component 0-255. A default Color is opaque
/// black.
struct Color {
    /// @brief r is the red component (0-255).
    std::uint8_t r = 0;
    /// @brief g is the green component (0-255).
    std::uint8_t g = 0;
    /// @brief b is the blue component (0-255).
    std::uint8_t b = 0;
    /// @brief a is the alpha component (0-255).
    std::uint8_t a = 255;

    /// @brief parses a color from a 6 or 8 digit hex string ("#ff0000" or
    /// "#ff000080"). The alpha of a 6 digit string is 255.
    static Color parse(x::json::Parser parser);

    /// @brief serializes the color to its "#rrggbbaa" JSON string.
    [[nodiscard]] x::json::json to_json() const;

    /// @brief returns the color as an 8 digit "#rrggbbaa" string.
    [[nodiscard]] std::string hex() const;

    /// @brief returns the color packed as 0xRRGGBBAA.
    [[nodiscard]] std::uint32_t rgba() const;

    /// @brief unpacks a color from its 0xRRGGBBAA form.
    static Color from_rgba(std::uint32_t v);

    using proto_type = ::x::color::pb::Color;

    /// @brief serializes the color to its protobuf representation.
    [[nodiscard]] std::pair<::x::color::pb::Color, x::errors::Error> to_proto() const;

    /// @brief constructs a color from its protobuf representation.
    static std::pair<Color, x::errors::Error>
    from_proto(const ::x::color::pb::Color &pb);
};

/// @brief from_css parses a CSS color: a '#'-prefixed hex or an rgb()/rgba() value.
/// Mirrors the Go color.FromCSS.
std::pair<Color, x::errors::Error> from_css(const std::string &input);
}
