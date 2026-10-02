// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <string_view>
#include <vector>

namespace x::strings {

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

/// @brief removes leading and trailing spaces, tabs, and carriage returns.
/// @param s the string to trim.
/// @returns the trimmed string, or an empty string when s is all whitespace.
[[nodiscard]] inline std::string trim(const std::string &s) {
    const auto first = s.find_first_not_of(" \t\r");
    if (first == std::string::npos) return {};
    return s.substr(first, s.find_last_not_of(" \t\r") - first + 1);
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

}
