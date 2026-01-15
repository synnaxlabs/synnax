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
#include <iomanip>
#include <sstream>
#include <stdexcept>
#include <string>

#include "x/cpp/color/types.gen.h"
#include "x/cpp/json/json.h"

namespace x::color {

/// @brief Returns the red component (0-255).
inline std::uint8_t r(const Color &c) {
    return c[0];
}

/// @brief Returns the green component (0-255).
inline std::uint8_t g(const Color &c) {
    return c[1];
}

/// @brief Returns the blue component (0-255).
inline std::uint8_t b(const Color &c) {
    return c[2];
}

/// @brief Returns the alpha component (0-255).
inline std::uint8_t a(const Color &c) {
    return c[3];
}

/// @brief Returns true if the color is the zero value (all components are 0).
inline bool is_zero(const Color &c) {
    return c[0] == 0 && c[1] == 0 && c[2] == 0 && c[3] == 0;
}

/// @brief Converts a Color to hex string (#RRGGBB or #RRGGBBAA).
/// Returns #RRGGBB if alpha is 255, otherwise #RRGGBBAA.
inline std::string hex(const Color &c) {
    std::ostringstream ss;
    ss << "#" << std::hex << std::setfill('0');
    ss << std::setw(2) << static_cast<int>(c[0]);
    ss << std::setw(2) << static_cast<int>(c[1]);
    ss << std::setw(2) << static_cast<int>(c[2]);
    if (c[3] != 255) { ss << std::setw(2) << static_cast<int>(c[3]); }
    return ss.str();
}

/// @brief Parses a hex color string (#RRGGBB or #RRGGBBAA) into a Color.
/// If no alpha is specified, defaults to 255 (fully opaque).
inline Color from_hex(const std::string &s) {
    std::string hex_str = s;
    if (!hex_str.empty() && hex_str[0] == '#') { hex_str = hex_str.substr(1); }

    if (hex_str.length() != 6 && hex_str.length() != 8) {
        throw std::invalid_argument("invalid hex color: must be 6 or 8 hex digits");
    }

    auto parse_byte = [](const std::string &str, size_t pos) -> std::uint8_t {
        return static_cast<std::uint8_t>(std::stoi(str.substr(pos, 2), nullptr, 16));
    };

    Color result;
    result[0] = parse_byte(hex_str, 0);
    result[1] = parse_byte(hex_str, 2);
    result[2] = parse_byte(hex_str, 4);
    result[3] = (hex_str.length() == 8) ? parse_byte(hex_str, 6) : 255;

    return result;
}

/// @brief Parses JSON into a Color.
/// Accepts BOTH hex string AND [R,G,B,A] array for backward compatibility.
inline Color Color::parse(x::json::Parser parser) {
    const auto &j = parser.get_json();

    // Try array format first [R, G, B, A]
    if (j.is_array() && j.size() == 4) {
        Color result;
        result[0] = j[0].get<std::uint8_t>();
        result[1] = j[1].get<std::uint8_t>();
        result[2] = j[2].get<std::uint8_t>();
        result[3] = j[3].get<std::uint8_t>();
        return result;
    }

    // Fall back to hex string format
    if (j.is_string()) { return from_hex(j.get<std::string>()); }

    throw std::invalid_argument("color must be [R,G,B,A] array or hex string");
}

/// @brief Converts a Color to JSON as [R, G, B, A] array.
inline x::json::json Color::to_json() const {
    return x::json::json::array({(*this)[0], (*this)[1], (*this)[2], (*this)[3]});
}

} // namespace x::color
