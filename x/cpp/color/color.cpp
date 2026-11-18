// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cctype>
#include <cmath>
#include <cstdint>
#include <cstdlib>
#include <optional>
#include <string>
#include <vector>

#include "x/cpp/color/color.h"
#include "x/cpp/color/names.h"

namespace x::color {
namespace {

x::errors::Error invalid(const std::string &input, const std::string &reason) {
    return x::errors::Error(
        x::errors::VALIDATION,
        "invalid color \"" + input + "\": " + reason
    );
}

int hex_digit(const char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    return -1;
}

/// @brief parses 3, 4, 6, or 8 lowercase hex digits. Short forms repeat each digit.
std::optional<Color> from_hex(std::string digits) {
    if (digits.size() == 3 || digits.size() == 4) {
        std::string expanded;
        for (const char c: digits)
            expanded.append(2, c);
        digits = expanded;
    }
    if (digits.size() != 6 && digits.size() != 8) return std::nullopt;
    std::uint8_t bytes[4] = {0, 0, 0, 255};
    for (size_t i = 0; i < digits.size(); i += 2) {
        const int hi = hex_digit(digits[i]);
        const int lo = hex_digit(digits[i + 1]);
        if (hi < 0 || lo < 0) return std::nullopt;
        bytes[i / 2] = static_cast<std::uint8_t>(hi * 16 + lo);
    }
    return Color{.r = bytes[0], .g = bytes[1], .b = bytes[2], .a = bytes[3]};
}

/// @brief Number is a CSS number with its unit suffix.
struct Number {
    double value = 0;
    std::string unit;
};

/// @brief parses a decimal number followed by an optional unit such as "%" or "deg".
std::optional<Number> parse_number(const std::string &token) {
    size_t i = 0;
    if (i < token.size() && (token[i] == '+' || token[i] == '-')) i++;
    bool has_digit = false;
    while (i < token.size() && std::isdigit(static_cast<unsigned char>(token[i]))) {
        i++;
        has_digit = true;
    }
    if (i < token.size() && token[i] == '.') {
        i++;
        while (i < token.size() && std::isdigit(static_cast<unsigned char>(token[i]))) {
            i++;
            has_digit = true;
        }
    }
    if (!has_digit) return std::nullopt;
    return Number{
        .value = std::strtod(token.substr(0, i).c_str(), nullptr),
        .unit = token.substr(i),
    };
}

bool is_space(const char c) {
    return std::isspace(static_cast<unsigned char>(c)) != 0;
}

/// @brief splits text on sep, trimming each piece. With sep ' ', runs of whitespace
/// separate pieces and no piece is empty.
std::vector<std::string> split(const std::string &text, const char sep) {
    std::vector<std::string> out;
    std::string current;
    for (const char c: text) {
        if (sep == ' ' ? is_space(c) : c == sep) {
            if (sep != ' ' || !current.empty()) out.push_back(current);
            current.clear();
        } else if (!is_space(c) || !current.empty())
            current.push_back(c);
    }
    if (sep != ' ' || !current.empty()) out.push_back(current);
    for (auto &piece: out)
        while (!piece.empty() && is_space(piece.back()))
            piece.pop_back();
    return out;
}

/// @brief splits the arguments of a CSS color function into three channels and an
/// optional alpha. Accepts "a, b, c[, d]" and "a b c[ / d]".
std::optional<std::vector<std::string>> split_args(const std::string &body) {
    std::vector<std::string> args;
    if (body.contains(',')) {
        args = split(body, ',');
        if (args.size() != 3 && args.size() != 4) return std::nullopt;
    } else {
        const auto halves = split(body, '/');
        if (halves.size() > 2) return std::nullopt;
        args = split(halves[0], ' ');
        if (args.size() != 3) return std::nullopt;
        if (halves.size() == 2) {
            const auto alpha = split(halves[1], ' ');
            if (alpha.size() != 1) return std::nullopt;
            args.push_back(alpha[0]);
        }
    }
    for (const auto &arg: args)
        if (arg.empty()) return std::nullopt;
    return args;
}

/// @brief parses a 0-1 or percentage alpha to a byte.
std::optional<std::uint8_t> parse_alpha(const std::string &token) {
    const auto n = parse_number(token);
    if (!n) return std::nullopt;
    double a = n->value;
    if (n->unit == "%")
        a /= 100;
    else if (!n->unit.empty())
        return std::nullopt;
    if (a < 0 || a > 1) return std::nullopt;
    return static_cast<std::uint8_t>(std::lround(a * 255));
}

/// @brief parses a 0-255 or percentage RGB channel to a byte.
std::optional<std::uint8_t> parse_channel(const std::string &token) {
    const auto n = parse_number(token);
    if (!n) return std::nullopt;
    double v = n->value;
    if (n->unit == "%") {
        if (v < 0 || v > 100) return std::nullopt;
        v = v * 255 / 100;
    } else if (!n->unit.empty() || v < 0 || v > 255)
        return std::nullopt;
    return static_cast<std::uint8_t>(std::lround(v));
}

/// @brief parses a 0-100 saturation or lightness, with or without '%', to a fraction.
std::optional<double> parse_percent(const std::string &token) {
    const auto n = parse_number(token);
    if (!n || (!n->unit.empty() && n->unit != "%")) return std::nullopt;
    if (n->value < 0 || n->value > 100) return std::nullopt;
    return n->value / 100;
}

/// @brief parses a hue in degrees, with or without "deg".
std::optional<double> parse_hue(const std::string &token) {
    const auto n = parse_number(token);
    if (!n || (!n->unit.empty() && n->unit != "deg")) return std::nullopt;
    return n->value;
}

/// @brief converts a hue in degrees and 0-1 saturation and lightness to RGB bytes.
/// The hue wraps, as it does in CSS.
Color hsl_to_rgb(double h, const double s, const double l, const std::uint8_t a) {
    h = std::fmod(h, 360);
    if (h < 0) h += 360;
    const double c = (1 - std::abs(2 * l - 1)) * s;
    const double x = c * (1 - std::abs(std::fmod(h / 60, 2) - 1));
    const double m = l - c / 2;
    double r = 0, g = 0, b = 0;
    if (h < 60)
        r = c, g = x;
    else if (h < 120)
        r = x, g = c;
    else if (h < 180)
        g = c, b = x;
    else if (h < 240)
        g = x, b = c;
    else if (h < 300)
        r = x, b = c;
    else
        r = c, b = x;
    const auto channel = [m](const double v) {
        return static_cast<std::uint8_t>(std::lround((v + m) * 255));
    };
    return Color{.r = channel(r), .g = channel(g), .b = channel(b), .a = a};
}

std::optional<Color> from_function(const std::string &name, const std::string &body) {
    const auto args = split_args(body);
    if (!args) return std::nullopt;
    std::uint8_t a = 255;
    if (args->size() == 4) {
        const auto alpha = parse_alpha((*args)[3]);
        if (!alpha) return std::nullopt;
        a = *alpha;
    }
    if (name == "rgb" || name == "rgba") {
        const auto r = parse_channel((*args)[0]);
        const auto g = parse_channel((*args)[1]);
        const auto b = parse_channel((*args)[2]);
        if (!r || !g || !b) return std::nullopt;
        return Color{.r = *r, .g = *g, .b = *b, .a = a};
    }
    const auto h = parse_hue((*args)[0]);
    const auto s = parse_percent((*args)[1]);
    const auto l = parse_percent((*args)[2]);
    if (!h || !s || !l) return std::nullopt;
    return hsl_to_rgb(*h, *s, *l, a);
}

std::uint32_t pack(const Color &c) {
    return static_cast<std::uint32_t>(c.r) << 24 |
           static_cast<std::uint32_t>(c.g) << 16 |
           static_cast<std::uint32_t>(c.b) << 8 | static_cast<std::uint32_t>(c.a);
}

Color unpack(const std::uint32_t v) {
    return Color{
        .r = static_cast<std::uint8_t>(v >> 24),
        .g = static_cast<std::uint8_t>(v >> 16),
        .b = static_cast<std::uint8_t>(v >> 8),
        .a = static_cast<std::uint8_t>(v),
    };
}
}

std::pair<Color, x::errors::Error> parse(const std::string &input) {
    std::string s = input;
    s.erase(0, s.find_first_not_of(" \t\n\r\f\v"));
    s.erase(s.find_last_not_of(" \t\n\r\f\v") + 1);
    std::ranges::transform(s, s.begin(), [](const unsigned char c) {
        return static_cast<char>(std::tolower(c));
    });
    if (s.empty()) return {Color{}, invalid(input, "the string is empty")};
    const auto i = static_cast<std::size_t>(
        std::ranges::lower_bound(NAMES, s, {}, &NamedColor::name) - NAMES.begin()
    );
    if (i < NAMES.size() && NAMES[i].name == s) {
        const auto &n = NAMES[i];
        return {Color{.r = n.r, .g = n.g, .b = n.b, .a = n.a}, x::errors::NIL};
    }
    if (const auto open = s.find('('); open != std::string::npos) {
        const std::string name = s.substr(0, open);
        if (s.back() != ')' ||
            (name != "rgb" && name != "rgba" && name != "hsl" && name != "hsla"))
            return {
                Color{},
                invalid(input, "expected rgb(), rgba(), hsl(), or hsla()")
            };
        const auto c = from_function(name, s.substr(open + 1, s.size() - open - 2));
        if (!c)
            return {Color{}, invalid(input, "an argument is missing or out of range")};
        return {*c, x::errors::NIL};
    }
    if (s.front() == '#') s.erase(0, 1);
    const auto c = from_hex(s);
    if (!c) return {Color{}, invalid(input, "expected 3, 4, 6, or 8 hex digits")};
    return {*c, x::errors::NIL};
}

Color Color::parse(const x::json::Parser &parser) {
    const auto j = parser.get_json();
    if (!j.is_string()) {
        parser.field_err("", "color must be a string: " + j.dump());
        return Color{};
    }
    auto [c, err] = x::color::parse(j.get<std::string>());
    if (err) parser.field_err("", err);
    return c;
}

std::string Color::hex() const {
    static constexpr char digits[] = "0123456789abcdef";
    std::string out = "#";
    const auto put = [&](const std::uint8_t c) {
        out += digits[c >> 4];
        out += digits[c & 0xf];
    };
    put(this->r);
    put(this->g);
    put(this->b);
    if (this->a != 255) put(this->a);
    return out;
}

x::json::json Color::to_json() const {
    return this->hex();
}

std::pair<::x::color::pb::Color, x::errors::Error> Color::to_proto() const {
    ::x::color::pb::Color pb;
    pb.set_rgba(pack(*this));
    return {pb, x::errors::NIL};
}

std::pair<Color, x::errors::Error> Color::from_proto(const ::x::color::pb::Color &pb) {
    return {unpack(pb.rgba()), x::errors::NIL};
}
}
