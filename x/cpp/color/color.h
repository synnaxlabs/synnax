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

#include "x/cpp/color/json.gen.h"
#include "x/cpp/color/proto.gen.h"
#include "x/cpp/color/types.gen.h"
#include "x/cpp/errors/errors.h"

namespace x::color {

/// @brief from_hex parses a 6- or 8-character hex color, with or without a leading
/// '#'. Mirrors the Go color.FromHex.
inline std::pair<Color, x::errors::Error> from_hex(const std::string &input) {
    std::string s = input;
    if (!s.empty() && s.front() == '#') s = s.substr(1);
    const auto byte = [](const std::string &h, std::uint8_t &out) -> bool {
        if (h.size() != 2) return false;
        try {
            size_t pos = 0;
            const unsigned long v = std::stoul(h, &pos, 16);
            if (pos != 2) return false;
            out = static_cast<std::uint8_t>(v);
            return true;
        } catch (...) { return false; }
    };
    std::uint8_t r = 0, g = 0, b = 0, a = 0;
    if (s.size() == 6) {
        if (!byte(s.substr(0, 2), r) || !byte(s.substr(2, 2), g) ||
            !byte(s.substr(4, 2), b))
            return {
                Color{},
                x::errors::Error(x::errors::VALIDATION, "invalid hex color: " + input)
            };
        return {Color{.r = r, .g = g, .b = b, .a = 1}, x::errors::NIL};
    }
    if (s.size() == 8) {
        if (!byte(s.substr(0, 2), r) || !byte(s.substr(2, 2), g) ||
            !byte(s.substr(4, 2), b) || !byte(s.substr(6, 2), a))
            return {
                Color{},
                x::errors::Error(x::errors::VALIDATION, "invalid hex color: " + input)
            };
        return {
            Color{.r = r, .g = g, .b = b, .a = static_cast<double>(a) / 255.0},
            x::errors::NIL
        };
    }
    return {
        Color{},
        x::errors::Error(x::errors::VALIDATION, "invalid hex color length: " + input)
    };
}

/// @brief from_css parses a CSS color: a '#'-prefixed hex or an rgb()/rgba() value.
/// Mirrors the Go color.FromCSS.
std::pair<Color, x::errors::Error> from_css(const std::string &input);
}
