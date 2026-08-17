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

/// @brief Color is an RGBA color with RGB as 0-255 and alpha as 0-1.
struct Color {
    /// @brief r is the red component (0-255).
    std::uint8_t r = 0;
    /// @brief g is the green component (0-255).
    std::uint8_t g = 0;
    /// @brief b is the blue component (0-255).
    std::uint8_t b = 0;
    /// @brief a is the alpha component (0-1).
    double a = 0;

    /// @brief parses a color from a JSON parser. Accepts a hex string ("#ff0000" or
    /// "#ff000080"), an [R, G, B] or [R, G, B, A] array, an {r, g, b, a} object, or a
    /// legacy {"rgba255": [R, G, B, A]} object. An alpha above 1 is lifted from the
    /// legacy 0-255 scale onto 0-1. JSON null and the empty string both parse to the
    /// zero Color. Mirrors the Go color.Color decoders.
    static Color parse(x::json::Parser parser);

    /// @brief serializes the color to its {r, g, b, a} JSON representation.
    [[nodiscard]] x::json::json to_json() const;

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
