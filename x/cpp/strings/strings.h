// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cctype>
#include <charconv>
#include <cmath>
#include <concepts>
#include <cstddef>
#include <cstdint>
#include <string>
#include <string_view>
#include <vector>

namespace x::strings {

/// @brief Formats v as the shortest text that parses back to v. The text is in exponent
/// notation when the decimal exponent of its shortest digits is below -6 or from 21,
/// with an unpadded exponent (1e-7, 1e+21). NaN is "NaN", and infinities are "Infinity"
/// and "-Infinity".
template<std::floating_point T>
std::string format_float(T v) {
    if (std::isnan(v)) return "NaN";
    if (std::isinf(v)) return v > 0 ? "Infinity" : "-Infinity";
    char buf[64];
    const auto [end, ec] = std::to_chars(
        buf,
        buf + sizeof(buf),
        v,
        std::chars_format::scientific
    );
    if (ec != std::errc{}) return "";
    // Fixed-mode to_chars writes every integer digit of a large value, not the
    // shortest ones, so expand the shortest scientific digits by hand.
    const std::string sci(buf, end);
    const size_t e = sci.find('e');
    const int exp = std::stoi(sci.substr(e + 1));
    const bool neg = sci[0] == '-';
    std::string digits;
    for (size_t i = neg ? 1 : 0; i < e; ++i)
        if (sci[i] != '.') digits += sci[i];
    std::string out = neg ? "-" : "";
    if (exp < -6 || exp >= 21) {
        out += digits[0];
        if (digits.size() > 1) out += "." + digits.substr(1);
        return out + (exp < 0 ? "e-" : "e+") + std::to_string(std::abs(exp));
    }
    if (exp < 0) return out + "0." + std::string(-exp - 1, '0') + digits;
    const auto point = static_cast<size_t>(exp) + 1;
    if (digits.size() <= point)
        return out + digits + std::string(point - digits.size(), '0');
    return out + digits.substr(0, point) + "." + digits.substr(point);
}

/// @brief escapes the line breaks in s so it occupies one line.
/// @param s the string to escape.
/// @returns s with each '\n' and '\r' replaced by the two characters "\\n" and
/// "\\r". Every other byte is kept as is.
[[nodiscard]] inline std::string escape_line_breaks(const std::string &s) {
    std::string out;
    out.reserve(s.size());
    for (const char c: s)
        if (c == '\n')
            out += "\\n";
        else if (c == '\r')
            out += "\\r";
        else
            out += c;
    return out;
}

/// @brief removes every '\n' and '\r' at the end of s, such as the line break that an
/// editor adds at the end of a file.
/// @param s the string to trim.
/// @returns s without its trailing line breaks. Spaces and inner line breaks stay.
[[nodiscard]] inline std::string trim_trailing_line_breaks(std::string s) {
    s.erase(s.find_last_not_of("\r\n") + 1);
    return s;
}

/// @brief writes v as a zero-padded decimal of the given width into the buffer at p.
/// @param p pointer to at least width writable chars.
/// @param v non-negative integer value to write.
/// @param width number of digits to write (zero-padded on the left).
inline void write_number(char *p, int v, int width) noexcept {
    for (int i = width - 1; i >= 0; --i) {
        p[i] = char('0' + (v % 10));
        v /= 10;
    }
}

/// @brief the characters that trim removes from each end of a string.
constexpr std::string_view TRIMMED = " \t\r";

/// @brief removes leading and trailing spaces, tabs, and carriage returns.
/// @param s the string to trim.
/// @returns a view into s without the TRIMMED characters at its ends, or an empty view
/// when s holds only them. It is valid only while s is.
[[nodiscard]] constexpr std::string_view trim(const std::string_view s) {
    const auto first = s.find_first_not_of(TRIMMED);
    if (first == std::string_view::npos) return {};
    return s.substr(first, s.find_last_not_of(TRIMMED) - first + 1);
}

/// @brief removes one layer of matching single or double quotes.
/// @param s the string to unquote.
/// @returns the contents of the quotes, or s unchanged when it is not quoted.
[[nodiscard]] inline std::string unquote(const std::string &s) {
    if (s.size() < 2) return s;
    const char quote = s.front();
    if ((quote == '"' || quote == '\'') && s.back() == quote)
        return s.substr(1, s.size() - 2);
    return s;
}

/// @brief joins a vector of strings with the given separator.
/// @param parts the strings to join.
/// @param sep the separator to insert between each pair of adjacent strings.
/// @returns the concatenated result, or an empty string if parts is empty.
[[nodiscard]] inline std::string
join(const std::vector<std::string> &parts, const std::string_view sep) {
    if (parts.empty()) return {};
    size_t total = sep.size() * (parts.size() - 1);
    for (const auto &p: parts)
        total += p.size();
    std::string out;
    out.reserve(total);
    out += parts[0];
    for (size_t i = 1; i < parts.size(); i++) {
        out += sep;
        out += parts[i];
    }
    return out;
}

/// @returns true if s is valid UTF-8, with no overlong form, surrogate, or code point
/// past U+10FFFF.
[[nodiscard]] inline bool valid_utf8(const std::string_view s) noexcept {
    size_t i = 0;
    while (i < s.size()) {
        const auto lead = static_cast<unsigned char>(s[i]);
        if (lead < 0x80) {
            i++;
            continue;
        }
        size_t n = 0;
        uint32_t code_point = 0;
        uint32_t min = 0;
        if ((lead & 0xE0) == 0xC0) {
            n = 1;
            code_point = lead & 0x1F;
            min = 0x80;
        } else if ((lead & 0xF0) == 0xE0) {
            n = 2;
            code_point = lead & 0x0F;
            min = 0x800;
        } else if ((lead & 0xF8) == 0xF0) {
            n = 3;
            code_point = lead & 0x07;
            min = 0x10000;
        } else {
            return false;
        }
        if (i + n >= s.size()) return false;
        for (size_t k = 1; k <= n; k++) {
            const auto b = static_cast<unsigned char>(s[i + k]);
            if ((b & 0xC0) != 0x80) return false;
            code_point = code_point << 6 | (b & 0x3F);
        }
        if (code_point < min || code_point > 0x10FFFF ||
            (code_point >= 0xD800 && code_point <= 0xDFFF))
            return false;
        i += n + 1;
    }
    return true;
}

}
