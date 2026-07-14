// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <charconv>
#include <cmath>
#include <cstdint>
#include <string>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

namespace x::json {
namespace priv {
/// @brief appends f formatted per the ECMAScript Number::toString algorithm, the
/// format JSON.stringify emits. Integral doubles print without a decimal point.
inline x::errors::Error append_es6_number(std::string &out, double f) {
    if (std::isnan(f) || std::isinf(f))
        return x::errors::Error("canonical JSON: NaN and Inf are not representable");
    if (f == 0) {
        out += '0';
        return x::errors::NIL;
    }
    if (f < 0) {
        out += '-';
        f = -f;
    }
    char buf[64];
    const auto
        res = std::to_chars(buf, buf + sizeof(buf), f, std::chars_format::scientific);
    // Normalize "d[.ddd]e[+|-][0]*X" into shortest digits + base-10 exponent.
    std::string digits;
    int exp10 = 0;
    bool exp_neg = false;
    bool in_exp = false;
    for (const char *c = buf; c < res.ptr; ++c) {
        if (*c == '.') continue;
        if (*c == 'e') {
            in_exp = true;
            continue;
        }
        if (in_exp) {
            if (*c == '-')
                exp_neg = true;
            else if (*c != '+')
                exp10 = exp10 * 10 + (*c - '0');
        } else
            digits += *c;
    }
    if (exp_neg) exp10 = -exp10;
    const int k = static_cast<int>(digits.size());
    const int n = exp10 + 1;
    if (k <= n && n <= 21) {
        out += digits;
        out.append(n - k, '0');
    } else if (0 < n && n <= 21) {
        out += digits.substr(0, n);
        out += '.';
        out += digits.substr(n);
    } else if (-6 < n && n <= 0) {
        out += "0.";
        out.append(-n, '0');
        out += digits;
    } else {
        out += digits[0];
        if (k > 1) {
            out += '.';
            out += digits.substr(1);
        }
        out += 'e';
        out += n - 1 >= 0 ? '+' : '-';
        out += std::to_string(std::abs(n - 1));
    }
    return x::errors::NIL;
}

inline void append_canonical_string(std::string &out, const std::string &s) {
    constexpr char hex[] = "0123456789abcdef";
    out += '"';
    for (const char c: s)
        switch (c) {
            case '"':
                out += "\\\"";
                break;
            case '\\':
                out += "\\\\";
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
            default:
                if (static_cast<unsigned char>(c) < 0x20) {
                    out += "\\u00";
                    out += hex[static_cast<unsigned char>(c) >> 4];
                    out += hex[static_cast<unsigned char>(c) & 0xf];
                } else
                    out += c;
        }
    out += '"';
}

inline x::errors::Error append_canonical(std::string &out, const json &j) {
    switch (j.type()) {
        case json::value_t::null:
            out += "null";
            break;
        case json::value_t::boolean:
            out += j.get<bool>() ? "true" : "false";
            break;
        case json::value_t::string:
            append_canonical_string(out, j.get_ref<const std::string &>());
            break;
        case json::value_t::number_integer:
            out += std::to_string(j.get<int64_t>());
            break;
        case json::value_t::number_unsigned:
            out += std::to_string(j.get<uint64_t>());
            break;
        case json::value_t::number_float:
            return append_es6_number(out, j.get<double>());
        case json::value_t::array: {
            out += '[';
            bool first = true;
            for (const auto &elem: j) {
                if (!first) out += ',';
                first = false;
                if (const auto err = append_canonical(out, elem)) return err;
            }
            out += ']';
            break;
        }
        case json::value_t::object: {
            // nlohmann's object_t is std::map, so iteration is already sorted.
            out += '{';
            bool first = true;
            for (const auto &[key, value]: j.items()) {
                if (!first) out += ',';
                first = false;
                append_canonical_string(out, key);
                out += ':';
                if (const auto err = append_canonical(out, value)) return err;
            }
            out += '}';
            break;
        }
        default:
            return x::errors::Error(
                "canonical JSON: unsupported value type " + std::string(j.type_name())
            );
    }
    return x::errors::NIL;
}
}

/// @brief serializes j as canonical JSON: object keys sorted lexicographically,
/// compact separators, minimal string escaping, and ES6 number formatting. The
/// Go (x/go json.Canonical) and TypeScript (x json.canonicalString)
/// implementations produce identical bytes for the same value, making the
/// output suitable for cross-language hashing.
/// @param j the value to serialize.
/// @returns the canonical string and an error if j contains a value with no
/// JSON representation (NaN, Inf, binary).
inline std::pair<std::string, x::errors::Error> canonical(const json &j) {
    std::string out;
    if (const auto err = priv::append_canonical(out, j)) return {"", err};
    return {out, x::errors::NIL};
}
}
