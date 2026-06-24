// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <regex>

#include "x/cpp/color/color.h"

namespace x::color {

std::pair<Color, x::errors::Error> from_hex(const std::string &input) {
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

namespace {
std::pair<Color, x::errors::Error>
from_rgb_match(const std::smatch &m, const std::string &input) {
    const bool has_alpha = m[5].matched;
    if (m[1].str() == "rgb" && has_alpha)
        return {
            Color{},
            x::errors::Error(
                x::errors::VALIDATION,
                "rgb() takes 3 channels; use rgba() for alpha: " + input
            )
        };
    if (m[1].str() == "rgba" && !has_alpha)
        return {
            Color{},
            x::errors::Error(
                x::errors::VALIDATION,
                "rgba() requires a 4th alpha channel: " + input
            )
        };
    const int r = std::stoi(m[2].str());
    const int g = std::stoi(m[3].str());
    const int b = std::stoi(m[4].str());
    if (r > 255 || g > 255 || b > 255)
        return {
            Color{},
            x::errors::Error(
                x::errors::VALIDATION,
                "rgb channels must be 0-255: " + input
            )
        };
    double a = 1.0;
    if (has_alpha) {
        a = std::stod(m[5].str());
        if (a < 0 || a > 1)
            return {
                Color{},
                x::errors::Error(
                    x::errors::VALIDATION,
                    "rgba alpha must be 0-1: " + input
                )
            };
    }
    return {
        Color{
            .r = static_cast<std::uint8_t>(r),
            .g = static_cast<std::uint8_t>(g),
            .b = static_cast<std::uint8_t>(b),
            .a = a
        },
        x::errors::NIL
    };
}
}

std::pair<Color, x::errors::Error> from_css(const std::string &input) {
    const auto first = input.find_first_not_of(" \t\n\r\f\v");
    if (first == std::string::npos)
        return {
            Color{},
            x::errors::Error(
                x::errors::VALIDATION,
                "color must be a hex value (e.g. \"#3bc454\") or rgb(r,g,b): " + input
            )
        };
    const auto last = input.find_last_not_of(" \t\n\r\f\v");
    const std::string s = input.substr(first, last - first + 1);
    if (s.front() == '#') return from_hex(s);
    static const std::regex rgb_pattern(
        R"(^(rgba?)\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(?:,\s*([0-9.]+)\s*)?\)$)"
    );
    std::smatch m;
    if (std::regex_match(s, m, rgb_pattern)) return from_rgb_match(m, input);
    return {
        Color{},
        x::errors::Error(
            x::errors::VALIDATION,
            "color must be a hex value (e.g. \"#3bc454\") or rgb(r,g,b): " + input
        )
    };
}
}
