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
namespace {

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

/// @brief normalize_alpha lifts a legacy 0-255 alpha onto the current 0-1 scale.
/// Values in (1, 2] clamp to 1 instead of dividing: a legacy alpha that small means a
/// sub-1% opacity no user sets, while a current-scale value nudged past 1 by float
/// error means opaque. An alpha above 255 fits neither scale and accumulates a
/// validation error on the parser. Mirrors the rule in the Go color decoders.
double normalize_alpha(const double a, const x::json::Parser &parser) {
    if (a <= 1) return a;
    if (a <= 2) return 1;
    if (a > 255) {
        parser.field_err("a", "alpha is above the 0-255 scale");
        return 0;
    }
    return a / 255.0;
}

/// @brief from_array parses an [R, G, B] or [R, G, B, A] JSON array, lifting a legacy
/// 0-255 alpha onto 0-1. Accumulates an error on the parser and returns the zero
/// Color when the array is not a color tuple.
Color from_array(const x::json::json &arr, const x::json::Parser &parser) {
    if (arr.size() != 3 && arr.size() != 4) {
        parser.field_err(
            "",
            "invalid color array length: " + std::to_string(arr.size())
        );
        return Color{};
    }
    for (const auto &v: arr) {
        if (!v.is_number()) {
            parser.field_err("", "color array elements must be numbers");
            return Color{};
        }
        // Range-check before the uint8 casts below: an out-of-range double to int
        // conversion is undefined behavior, not a wrap.
        if (const double d = v.get<double>(); d < 0 || d > 255) {
            parser.field_err("", "color array elements must be within [0, 255]");
            return Color{};
        }
    }
    Color c{
        .r = static_cast<std::uint8_t>(arr[0].get<double>()),
        .g = static_cast<std::uint8_t>(arr[1].get<double>()),
        .b = static_cast<std::uint8_t>(arr[2].get<double>()),
        .a = 1.0,
    };
    if (arr.size() == 4) c.a = normalize_alpha(arr[3].get<double>(), parser);
    return c;
}

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

Color Color::parse(x::json::Parser parser) {
    const auto j = parser.get_json();
    if (j.is_null()) return Color{};
    if (j.is_string()) {
        const auto s = j.get<std::string>();
        if (s.empty()) return Color{};
        auto [c, err] = from_hex(s);
        if (err) {
            parser.field_err("", err);
            return Color{};
        }
        return c;
    }
    if (j.is_array()) return from_array(j, parser);
    if (j.is_object()) {
        if (const auto it = j.find("rgba255"); it != j.end() && it->is_array())
            return from_array(*it, parser);
        return Color{
            .r = parser.field<std::uint8_t>("r"),
            .g = parser.field<std::uint8_t>("g"),
            .b = parser.field<std::uint8_t>("b"),
            .a = normalize_alpha(parser.field<double>("a"), parser),
        };
    }
    parser.field_err("", "cannot parse color from: " + j.dump());
    return Color{};
}

x::json::json Color::to_json() const {
    x::json::json j;
    j["r"] = this->r;
    j["g"] = this->g;
    j["b"] = this->b;
    j["a"] = this->a;
    return j;
}

std::pair<::x::color::pb::Color, x::errors::Error> Color::to_proto() const {
    ::x::color::pb::Color pb;
    pb.set_r(this->r);
    pb.set_g(this->g);
    pb.set_b(this->b);
    pb.set_a(this->a);
    return {pb, x::errors::NIL};
}

std::pair<Color, x::errors::Error> Color::from_proto(const ::x::color::pb::Color &pb) {
    Color cpp;
    cpp.r = pb.r();
    cpp.g = pb.g();
    cpp.b = pb.b();
    cpp.a = pb.a();
    return {cpp, x::errors::NIL};
}
}
