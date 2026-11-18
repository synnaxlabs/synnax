// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <cstdint>
#include <optional>
#include <string>
#include <string_view>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/literal.h"
#include "x/cpp/json/text.h"

namespace x::json {
namespace {
/// @brief returns the length of the UTF-8 encoding of the code point that text starts
/// with, or 0 when text does not start with a valid encoding. An encoding of a
/// surrogate, or a longer encoding than the code point needs, is not valid.
std::size_t rune_length(const std::string_view text) {
    if (text.empty()) return 0;
    const auto b0 = static_cast<std::uint8_t>(text[0]);
    if (b0 < 0x80) return 1;
    std::size_t n = 0;
    std::uint8_t lo = 0x80;
    std::uint8_t hi = 0xBF;
    if (b0 >= 0xC2 && b0 <= 0xDF)
        n = 2;
    else if (b0 >= 0xE0 && b0 <= 0xEF) {
        n = 3;
        if (b0 == 0xE0) lo = 0xA0;
        if (b0 == 0xED) hi = 0x9F;
    } else if (b0 >= 0xF0 && b0 <= 0xF4) {
        n = 4;
        if (b0 == 0xF0) lo = 0x90;
        if (b0 == 0xF4) hi = 0x8F;
    } else
        return 0;
    if (text.size() < n) return 0;
    const auto b1 = static_cast<std::uint8_t>(text[1]);
    if (b1 < lo || b1 > hi) return 0;
    for (std::size_t i = 2; i < n; i++) {
        const auto b = static_cast<std::uint8_t>(text[i]);
        if (b < 0x80 || b > 0xBF) return 0;
    }
    return n;
}

const char *const HEX = "0123456789abcdef";

void append_hex(std::string &dst, const std::uint32_t v, const int digits) {
    for (int shift = (digits - 1) * 4; shift >= 0; shift -= 4)
        dst += HEX[(v >> shift) & 0xF];
}

/// @brief returns the value of four hex digits, or -1 when they are not hex.
int parse_hex4(const std::string_view text) {
    if (text.size() < 4) return -1;
    int v = 0;
    for (std::size_t i = 0; i < 4; i++) {
        const char c = text[i];
        v <<= 4;
        if (c >= '0' && c <= '9')
            v |= c - '0';
        else if (c >= 'a' && c <= 'f')
            v |= c - 'a' + 10;
        else if (c >= 'A' && c <= 'F')
            v |= c - 'A' + 10;
        else
            return -1;
    }
    return v;
}

void append_utf8(std::string &dst, const std::uint32_t cp) {
    if (cp < 0x80) {
        dst += static_cast<char>(cp);
    } else if (cp < 0x800) {
        dst += static_cast<char>(0xC0 | (cp >> 6));
        dst += static_cast<char>(0x80 | (cp & 0x3F));
    } else if (cp < 0x10000) {
        dst += static_cast<char>(0xE0 | (cp >> 12));
        dst += static_cast<char>(0x80 | ((cp >> 6) & 0x3F));
        dst += static_cast<char>(0x80 | (cp & 0x3F));
    } else {
        dst += static_cast<char>(0xF0 | (cp >> 18));
        dst += static_cast<char>(0x80 | ((cp >> 12) & 0x3F));
        dst += static_cast<char>(0x80 | ((cp >> 6) & 0x3F));
        dst += static_cast<char>(0x80 | (cp & 0x3F));
    }
}

/// @brief returns the character that a one-letter JSON escape stands for.
std::optional<char> unescape(const char e) {
    switch (e) {
        case '"':
            return '"';
        case '\\':
            return '\\';
        case '/':
            return '/';
        case 'b':
            return '\b';
        case 'f':
            return '\f';
        case 'n':
            return '\n';
        case 'r':
            return '\r';
        case 't':
            return '\t';
        default:
            return std::nullopt;
    }
}
}

std::string quote(const std::string_view text) {
    std::string out = "\"";
    for (std::size_t i = 0; i < text.size();) {
        const auto c = static_cast<std::uint8_t>(text[i]);
        if (c >= 0x80) {
            const auto n = rune_length(text.substr(i));
            if (n == 0) {
                out += "\\x";
                append_hex(out, c, 2);
                i++;
                continue;
            }
            const auto next = static_cast<std::uint8_t>(text[i + 1]);
            if (c == 0xC2 && next <= 0x9F) {
                out += "\\u00";
                append_hex(out, next, 2);
            } else
                out += text.substr(i, n);
            i += n;
            continue;
        }
        switch (c) {
            case '"':
                out += "\\\"";
                break;
            case '\\':
                out += "\\\\";
                break;
            case '\a':
                out += "\\a";
                break;
            case '\b':
                out += "\\b";
                break;
            case '\f':
                out += "\\f";
                break;
            case '\n':
                out += "\\n";
                break;
            case '\r':
                out += "\\r";
                break;
            case '\t':
                out += "\\t";
                break;
            case '\v':
                out += "\\v";
                break;
            default:
                if (c < 0x20 || c == 0x7F) {
                    out += "\\x";
                    append_hex(out, c, 2);
                } else
                    out += static_cast<char>(c);
        }
        i++;
    }
    out += '"';
    return out;
}

LiteralScan scan_literal(const std::string_view text, std::string *dst) {
    std::size_t i = 1;
    const auto fail = [&i](const std::string_view problem) {
        return LiteralScan{.end = i, .problem = problem};
    };
    while (i < text.size()) {
        const auto c = static_cast<std::uint8_t>(text[i]);
        if (c == '"') return LiteralScan{.end = i + 1};
        if (c < 0x20) return fail("invalid control character");
        if (c >= 0x80) {
            const auto n = rune_length(text.substr(i));
            if (n == 0) return fail("invalid UTF-8");
            if (dst != nullptr) *dst += text.substr(i, n);
            i += n;
            continue;
        }
        if (c != '\\') {
            if (dst != nullptr) *dst += static_cast<char>(c);
            i++;
            continue;
        }
        if (i + 1 == text.size()) break;
        const auto decoded = unescape(text[i + 1]);
        if (decoded.has_value()) {
            if (dst != nullptr) *dst += *decoded;
            i += 2;
            continue;
        }
        if (text[i + 1] != 'u') return fail("invalid escape sequence");
        const int unit = parse_hex4(text.substr(i + 2));
        if (unit < 0) return fail("invalid escape sequence");
        auto cp = static_cast<std::uint32_t>(unit);
        std::size_t size = 6;
        if (cp >= 0xD800 && cp <= 0xDFFF) {
            const auto rest = text.substr(i + 6);
            const int low = cp <= 0xDBFF && rest.size() >= 6 && rest[0] == '\\' &&
                                    rest[1] == 'u'
                              ? parse_hex4(rest.substr(2))
                              : -1;
            if (low < 0xDC00 || low > 0xDFFF)
                return fail("invalid surrogate in escape sequence");
            cp = 0x10000 + ((cp - 0xD800) << 10) +
                 (static_cast<std::uint32_t>(low) - 0xDC00);
            size = 12;
        }
        if (dst != nullptr) append_utf8(*dst, cp);
        i += size;
    }
    i = text.size();
    return fail("unexpected EOF");
}

errors::Error unquote(const std::string_view raw, std::string &dst) {
    dst.clear();
    if (raw.empty() || raw[0] != '"')
        return errors::Error(errors::VALIDATION, "expected a JSON string literal");
    const auto scan = scan_literal(raw, &dst);
    if (!scan.problem.empty())
        return errors::Error(
            errors::VALIDATION,
            std::string(scan.problem) + " in string literal"
        );
    if (scan.end != raw.size())
        return errors::Error(
            errors::VALIDATION,
            "unexpected data after the string literal"
        );
    return errors::NIL;
}
}
