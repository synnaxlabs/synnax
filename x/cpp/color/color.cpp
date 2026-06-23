// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <regex>

#include "x/cpp/color/color.h"

namespace x::color {

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
