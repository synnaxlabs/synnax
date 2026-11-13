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

    bool operator==(const Color &) const = default;

    /// @brief parses a color from a JSON string in any form x::color::parse accepts.
    /// Accumulates an error on the parser for any other JSON value.
    static Color parse(const x::json::Parser &parser);

    /// @brief serializes the color to its hex string.
    [[nodiscard]] x::json::json to_json() const;

    /// @brief returns the color as a lowercase "#rrggbb" string when it is opaque, and
    /// as "#rrggbbaa" otherwise.
    [[nodiscard]] std::string hex() const;

    using proto_type = ::x::color::pb::Color;

    /// @brief serializes the color to its protobuf representation.
    [[nodiscard]] std::pair<::x::color::pb::Color, x::errors::Error> to_proto() const;

    /// @brief constructs a color from its protobuf representation.
    static std::pair<Color, x::errors::Error>
    from_proto(const ::x::color::pb::Color &pb);
};

/// @brief parses a color string. Accepts a CSS color name or the transparent keyword,
/// hex with an optional '#' and 3, 4, 6, or 8 digits, rgb() or rgba(), and hsl() or
/// hsla(), in comma or space syntax with an optional 0-1 or percentage alpha. A value
/// without an alpha is opaque.
/// @returns a validation error for any other string or an out-of-range value.
std::pair<Color, x::errors::Error> parse(const std::string &input);
}
