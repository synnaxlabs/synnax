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
#include <cstdio>
#include <limits>
#include <memory>
#include <string>

#include "wasmtime.hh"

#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/strings/state.h"

namespace arc::stl::strings {

inline constexpr const char *MODULE_NAME = "strings";

/// Formats v as the shortest round-trippable decimal, matching Go's
/// strconv.FormatFloat(v, 'g', -1, bitSize). NaN and ±Inf are emitted as
/// "NaN", "+Inf", "-Inf" to match Go's output exactly.
template<typename T>
std::string format_float(T v) {
    if (std::isnan(v)) return "NaN";
    if (std::isinf(v)) return v > 0 ? "+Inf" : "-Inf";
    char buf[32];
    const auto [end, ec] = std::to_chars(
        buf,
        buf + sizeof(buf),
        v,
        std::chars_format::general
    );
    if (ec != std::errc{}) return "";
    return {buf, end};
}

/// Parsed Go-style format spec ([flags][width][.precision][verb]).
struct FormatSpec {
    bool alt = false;
    bool plus = false;
    bool minus = false;
    bool space = false;
    bool zero = false;
    int width = -1;
    int precision = -1;
    char verb = '\0';
};

inline FormatSpec parse_format_spec(const std::string &s) {
    FormatSpec f;
    size_t i = 0;
    bool flags_done = false;
    while (i < s.size() && !flags_done) {
        switch (s[i]) {
            case '#':
                f.alt = true;
                ++i;
                break;
            case '+':
                f.plus = true;
                ++i;
                break;
            case '-':
                f.minus = true;
                ++i;
                break;
            case ' ':
                f.space = true;
                ++i;
                break;
            case '0':
                f.zero = true;
                ++i;
                break;
            default:
                flags_done = true;
                break;
        }
    }
    if (i < s.size() && s[i] >= '0' && s[i] <= '9') {
        f.width = 0;
        while (i < s.size() && s[i] >= '0' && s[i] <= '9') {
            f.width = f.width * 10 + (s[i] - '0');
            ++i;
        }
    }
    if (i < s.size() && s[i] == '.') {
        ++i;
        f.precision = 0;
        while (i < s.size() && s[i] >= '0' && s[i] <= '9') {
            f.precision = f.precision * 10 + (s[i] - '0');
            ++i;
        }
    }
    if (i < s.size()) f.verb = s[i];
    return f;
}

/// Encodes a Unicode code point as UTF-8. Mirrors Go's behavior: invalid
/// code points (negative, > U+10FFFF, or surrogates) become U+FFFD.
inline std::string utf8_encode(int64_t cp) {
    std::string out;
    if (cp < 0 || cp > 0x10FFFF || (cp >= 0xD800 && cp <= 0xDFFF)) {
        out += "\xEF\xBF\xBD";
        return out;
    }
    if (cp < 0x80) {
        out += static_cast<char>(cp);
    } else if (cp < 0x800) {
        out += static_cast<char>(0xC0 | (cp >> 6));
        out += static_cast<char>(0x80 | (cp & 0x3F));
    } else if (cp < 0x10000) {
        out += static_cast<char>(0xE0 | (cp >> 12));
        out += static_cast<char>(0x80 | ((cp >> 6) & 0x3F));
        out += static_cast<char>(0x80 | (cp & 0x3F));
    } else {
        out += static_cast<char>(0xF0 | (cp >> 18));
        out += static_cast<char>(0x80 | ((cp >> 12) & 0x3F));
        out += static_cast<char>(0x80 | ((cp >> 6) & 0x3F));
        out += static_cast<char>(0x80 | (cp & 0x3F));
    }
    return out;
}

/// Counts UTF-8 code points in s. Each byte whose top two bits are not 10
/// starts a new rune, so this also counts invalid bytes as one rune each
/// (matching Go's tolerance for non-UTF-8 input).
inline int utf8_rune_count(const std::string &s) {
    int count = 0;
    for (const unsigned char c: s)
        if ((c & 0xC0) != 0x80) ++count;
    return count;
}

/// Returns the byte prefix of s covering the first n runes. Used for Go-style
/// precision truncation on %s and %q, which counts code points, not bytes.
inline std::string utf8_truncate(const std::string &s, int n) {
    if (n <= 0) return "";
    int count = 0;
    for (size_t i = 0; i < s.size(); ++i) {
        const unsigned char c = static_cast<unsigned char>(s[i]);
        if ((c & 0xC0) != 0x80) {
            if (count == n) return s.substr(0, i);
            ++count;
        }
    }
    return s;
}

/// Converts an unsigned magnitude to a string in the given base (2, 8, 10, 16).
inline std::string convert_base(uint64_t v, int base, bool upper) {
    if (v == 0) return "0";
    const char *digits = upper ? "0123456789ABCDEF" : "0123456789abcdef";
    std::string out;
    while (v) {
        out.insert(out.begin(), digits[v % static_cast<uint64_t>(base)]);
        v /= static_cast<uint64_t>(base);
    }
    return out;
}

/// Pads and assembles a formatted value (sign + prefix + body) to the
/// requested width. Honors the `-` and `0` flags per Go semantics.
inline std::string apply_width(
    const std::string &sign,
    const std::string &prefix,
    const std::string &body,
    const FormatSpec &f
) {
    const int sz = static_cast<int>(sign.size() + prefix.size() + body.size());
    if (f.width <= 0 || sz >= f.width) return sign + prefix + body;
    const int pad = f.width - sz;
    if (f.minus) return sign + prefix + body + std::string(pad, ' ');
    // Zero flag is suppressed when precision is explicitly set (Go behavior).
    if (f.zero && f.precision < 0) return sign + prefix + std::string(pad, '0') + body;
    return std::string(pad, ' ') + sign + prefix + body;
}

/// Formats an integer value to match Go's fmt.Sprintf("%"+spec, v). When
/// is_signed is false, value is reinterpreted as uint64_t.
inline std::string
format_int_value(const std::string &spec_str, int64_t value, bool is_signed) {
    if (spec_str.empty())
        return is_signed ? std::to_string(value)
                         : std::to_string(static_cast<uint64_t>(value));
    const FormatSpec f = parse_format_spec(spec_str);
    const bool neg = is_signed && value < 0;
    uint64_t abs_v;
    if (is_signed) {
        if (value == std::numeric_limits<int64_t>::min())
            abs_v = static_cast<uint64_t>(std::numeric_limits<int64_t>::max()) + 1;
        else
            abs_v = static_cast<uint64_t>(neg ? -value : value);
    } else {
        abs_v = static_cast<uint64_t>(value);
    }
    std::string sign;
    if (neg)
        sign = "-";
    else if (f.plus)
        sign = "+";
    else if (f.space)
        sign = " ";

    std::string prefix;
    std::string body;
    int base = 10;
    bool upper = false;
    std::string alt_prefix;
    bool alt_applies = false;
    switch (f.verb) {
        case 'd':
            break;
        case 'b':
            base = 2;
            alt_prefix = "0b";
            alt_applies = true;
            break;
        case 'o':
            base = 8;
            alt_prefix = "0";
            alt_applies = true;
            break;
        case 'O':
            base = 8;
            prefix = "0o";
            break;
        case 'x':
            base = 16;
            alt_prefix = "0x";
            alt_applies = true;
            break;
        case 'X':
            base = 16;
            upper = true;
            alt_prefix = "0X";
            alt_applies = true;
            break;
        case 'c': {
            // %c outputs one rune; width is in rune count, not bytes.
            std::string body_c = utf8_encode(value);
            if (f.width <= 1) return body_c;
            const int pad = f.width - 1;
            if (f.minus) return body_c + std::string(pad, ' ');
            return std::string(pad, ' ') + body_c;
        }
        case 'q': {
            // %q on a rune mirrors Go's strconv.QuoteRune: wrap in single
            // quotes and escape control chars / DEL; printable code points
            // pass through as UTF-8.
            std::string body_q = "'";
            bool special = true;
            switch (value) {
                case '\\':
                    body_q += "\\\\";
                    break;
                case '\'':
                    body_q += "\\'";
                    break;
                case '\a':
                    body_q += "\\a";
                    break;
                case '\b':
                    body_q += "\\b";
                    break;
                case '\f':
                    body_q += "\\f";
                    break;
                case '\n':
                    body_q += "\\n";
                    break;
                case '\r':
                    body_q += "\\r";
                    break;
                case '\t':
                    body_q += "\\t";
                    break;
                case '\v':
                    body_q += "\\v";
                    break;
                default:
                    special = false;
                    break;
            }
            if (!special) {
                if (value >= 0x20 && value < 0x7F) {
                    body_q += static_cast<char>(value);
                } else if (value < 0x80) {
                    char buf[8];
                    std::snprintf(buf, sizeof(buf), "\\x%02x", static_cast<int>(value));
                    body_q += buf;
                } else {
                    body_q += utf8_encode(value);
                }
            }
            body_q += "'";
            if (f.width <= 0) return body_q;
            const int body_runes = utf8_rune_count(body_q);
            if (body_runes >= f.width) return body_q;
            const int pad = f.width - body_runes;
            if (f.minus) return body_q + std::string(pad, ' ');
            return std::string(pad, ' ') + body_q;
        }
        case 'U': {
            std::string hex = convert_base(abs_v, 16, true);
            if (hex.size() < 4) hex = std::string(4 - hex.size(), '0') + hex;
            return apply_width("", "U+", hex, f);
        }
        default: {
            std::string raw = is_signed ? std::to_string(value)
                                        : std::to_string(static_cast<uint64_t>(value));
            return "%!" + std::string(1, f.verb) + "(int=" + raw + ")";
        }
    }
    body = convert_base(abs_v, base, upper);
    // Go suppresses the '#' prefix only on `%#o` of zero, because the body
    // "0" already carries the implicit octal prefix. `%#x`, `%#X`, and
    // `%#b` keep their prefixes (0x0, 0X0, 0b0) even when the value is 0.
    if (f.alt && alt_applies && !(f.verb == 'o' && abs_v == 0)) prefix = alt_prefix;
    if (f.precision > 0 && static_cast<int>(body.size()) < f.precision)
        body = std::string(f.precision - body.size(), '0') + body;
    else if (f.precision == 0 && abs_v == 0)
        body = "";
    return apply_width(sign, prefix, body, f);
}

/// Formats a float value to match Go's fmt.Sprintf("%"+spec, v). Delegates
/// finite values to snprintf; NaN/±Inf use Go's capitalization.
inline std::string format_float_value(const std::string &spec_str, double v) {
    if (spec_str.empty()) return format_float(v);
    const FormatSpec f = parse_format_spec(spec_str);
    if (std::isnan(v)) {
        FormatSpec g = f;
        g.zero = false;
        std::string sign;
        if (f.plus)
            sign = "+";
        else if (f.space)
            sign = " ";
        return apply_width(sign, "", "NaN", g);
    }
    if (std::isinf(v)) {
        const bool neg = v < 0;
        std::string sign;
        if (neg)
            sign = "-";
        else if (f.space)
            sign = " ";
        else
            sign = "+";
        FormatSpec g = f;
        g.zero = false;
        return apply_width(sign, "", "Inf", g);
    }
    // Go's %g/%G with default precision uses shortest roundtrip. C's snprintf
    // defaults to 6 significant digits, so route those cases through
    // format_float() which uses std::to_chars(general).
    if ((f.verb == 'g' || f.verb == 'G') && f.precision < 0) {
        std::string body = format_float(v);
        if (f.verb == 'G')
            for (auto &c: body)
                if (c == 'e') {
                    c = 'E';
                    break;
                }
        std::string sign;
        if (!body.empty() && (body[0] == '-' || body[0] == '+')) {
            sign = body.substr(0, 1);
            body = body.substr(1);
        } else if (f.plus) {
            sign = "+";
        } else if (f.space) {
            sign = " ";
        }
        return apply_width(sign, "", body, f);
    }
    char buf[128];
    const int prec = f.precision >= 0 ? f.precision : 6;
    int n = 0;
    switch (f.verb) {
        case 'f':
        case 'F':
            n = f.alt ? std::snprintf(buf, sizeof(buf), "%#.*f", prec, v)
                      : std::snprintf(buf, sizeof(buf), "%.*f", prec, v);
            break;
        case 'e':
            n = f.alt ? std::snprintf(buf, sizeof(buf), "%#.*e", prec, v)
                      : std::snprintf(buf, sizeof(buf), "%.*e", prec, v);
            break;
        case 'E':
            n = f.alt ? std::snprintf(buf, sizeof(buf), "%#.*E", prec, v)
                      : std::snprintf(buf, sizeof(buf), "%.*E", prec, v);
            break;
        case 'g':
            n = f.alt ? std::snprintf(buf, sizeof(buf), "%#.*g", prec, v)
                      : std::snprintf(buf, sizeof(buf), "%.*g", prec, v);
            break;
        case 'G':
            n = f.alt ? std::snprintf(buf, sizeof(buf), "%#.*G", prec, v)
                      : std::snprintf(buf, sizeof(buf), "%.*G", prec, v);
            break;
        default:
            return "";
    }
    if (n < 0 || n >= static_cast<int>(sizeof(buf))) return "";
    std::string body(buf, n);
    std::string sign;
    if (!body.empty() && (body[0] == '-' || body[0] == '+')) {
        sign = body.substr(0, 1);
        body = body.substr(1);
    } else if (f.plus) {
        sign = "+";
    } else if (f.space) {
        sign = " ";
    }
    FormatSpec g = f;
    g.precision = -1;
    return apply_width(sign, "", body, g);
}

/// Quotes a string with Go's strconv.Quote semantics. Bytes >= 0x80 pass
/// through, so callers are responsible for UTF-8 validity.
inline std::string go_quote(const std::string &s) {
    std::string out;
    out.reserve(s.size() + 2);
    out += '"';
    for (const unsigned char c: s) {
        switch (c) {
            case '\\':
                out += "\\\\";
                break;
            case '"':
                out += "\\\"";
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
                if (c >= 0x20 && c < 0x7F) {
                    out += static_cast<char>(c);
                } else if (c >= 0x80) {
                    out += static_cast<char>(c);
                } else {
                    char hexbuf[8];
                    std::snprintf(hexbuf, sizeof(hexbuf), "\\x%02x", c);
                    out += hexbuf;
                }
                break;
        }
    }
    out += '"';
    return out;
}

inline std::string format_str_value(const std::string &spec_str, const std::string &v) {
    if (spec_str.empty()) return v;
    const FormatSpec f = parse_format_spec(spec_str);
    // Precision for %s and %q truncates the input by runes (Go semantics).
    // Precision is ignored for %x/%X, matching Go.
    std::string truncated = v;
    if (f.precision >= 0 && (f.verb == 's' || f.verb == 'q'))
        truncated = utf8_truncate(v, f.precision);
    std::string body;
    switch (f.verb) {
        case 's':
            body = truncated;
            break;
        case 'q':
            body = go_quote(truncated);
            break;
        case 'x':
        case 'X': {
            const bool upper = f.verb == 'X';
            const char *digits = upper ? "0123456789ABCDEF" : "0123456789abcdef";
            body.reserve(truncated.size() * 2);
            for (const unsigned char c: truncated) {
                body += digits[c >> 4];
                body += digits[c & 0xF];
            }
            break;
        }
        default:
            return "%!" + std::string(1, f.verb) + "(string=" + v + ")";
    }
    // Width compares against rune count for %s/%q, byte count for %x/%X.
    if (f.width <= 0) return body;
    const int body_len = (f.verb == 's' || f.verb == 'q')
                           ? utf8_rune_count(body)
                           : static_cast<int>(body.size());
    if (body_len >= f.width) return body;
    const int pad = f.width - body_len;
    if (f.minus) return body + std::string(pad, ' ');
    return std::string(pad, ' ') + body;
}

class Module : public stl::Module {
    std::shared_ptr<State> str_state;
    wasmtime::Store *store = nullptr;
    wasmtime::Memory *memory = nullptr;

    /// Reads `len` bytes at `ptr` from WASM memory. Returns empty string if
    /// memory has not been bound or the read would be out of bounds.
    std::string read_memory_string(uint32_t ptr, uint32_t len) const {
        if (!this->memory || !this->store) return "";
        const auto mem_span = this->memory->data(*this->store);
        if (static_cast<size_t>(ptr) + len > mem_span.size()) return "";
        return {reinterpret_cast<const char *>(mem_span.data() + ptr), len};
    }

public:
    explicit Module(std::shared_ptr<State> str_state):
        str_state(std::move(str_state)) {}

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override {
        // SAFETY: raw `this` capture is safe because wasm::Module owns this
        // stl::Module via cfg.modules (shared_ptr), and Store/Memory are
        // stable members of the heap-pinned wasm::Module.
        auto self = this;
        auto ss = this->str_state;
        linker
            .func_wrap(
                MODULE_NAME,
                "from_literal",
                [self, ss](uint32_t ptr, uint32_t len) -> uint32_t {
                    if (!self->memory || !self->store) {
                        std::fprintf(
                            stderr,
                            "ERROR: string_from_literal called but no memory or "
                            "store available\n"
                        );
                        return 0;
                    }
                    const auto mem_span = self->memory->data(*self->store);
                    const uint8_t *mem_data = mem_span.data();
                    if (const size_t mem_size = mem_span.size(); ptr + len > mem_size) {
                        std::fprintf(
                            stderr,
                            "ERROR: string_from_literal ptr=%u len=%u exceeds "
                            "memory size=%zu\n",
                            ptr,
                            len,
                            mem_size
                        );
                        return 0;
                    }
                    return ss->from_memory(mem_data + ptr, len);
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "concat",
                [ss](uint32_t h1, uint32_t h2) -> uint32_t {
                    const std::string s1 = ss->get(h1);
                    const std::string s2 = ss->get(h2);
                    if (s1.empty() && s2.empty()) return 0;
                    return ss->create(s1 + s2);
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "equal",
                [ss](uint32_t h1, uint32_t h2) -> uint32_t {
                    if (!ss->exists(h1) || !ss->exists(h2)) return 0;
                    return ss->get(h1) == ss->get(h2) ? 1 : 0;
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "len",
                [ss](uint32_t handle) -> uint64_t {
                    return static_cast<uint64_t>(ss->get(handle).length());
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "from_i32",
                [ss](int32_t v) -> uint32_t { return ss->create(std::to_string(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "from_u32",
                [ss](uint32_t v) -> uint32_t { return ss->create(std::to_string(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "from_i64",
                [ss](int64_t v) -> uint32_t { return ss->create(std::to_string(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "from_u64",
                [ss](uint64_t v) -> uint32_t { return ss->create(std::to_string(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "from_f32",
                [ss](float v) -> uint32_t { return ss->create(format_float(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "from_f64",
                [ss](double v) -> uint32_t { return ss->create(format_float(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "format_i32",
                [self, ss](int32_t v, uint32_t ptr, uint32_t len) -> uint32_t {
                    const std::string spec = self->read_memory_string(ptr, len);
                    return ss->create(
                        format_int_value(spec, static_cast<int64_t>(v), true)
                    );
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "format_u32",
                [self, ss](uint32_t v, uint32_t ptr, uint32_t len) -> uint32_t {
                    const std::string spec = self->read_memory_string(ptr, len);
                    return ss->create(
                        format_int_value(spec, static_cast<int64_t>(v), false)
                    );
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "format_i64",
                [self, ss](int64_t v, uint32_t ptr, uint32_t len) -> uint32_t {
                    const std::string spec = self->read_memory_string(ptr, len);
                    return ss->create(format_int_value(spec, v, true));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "format_u64",
                [self, ss](uint64_t v, uint32_t ptr, uint32_t len) -> uint32_t {
                    const std::string spec = self->read_memory_string(ptr, len);
                    return ss->create(
                        format_int_value(spec, static_cast<int64_t>(v), false)
                    );
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "format_f32",
                [self, ss](float v, uint32_t ptr, uint32_t len) -> uint32_t {
                    const std::string spec = self->read_memory_string(ptr, len);
                    return ss->create(format_float_value(spec, static_cast<double>(v)));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "format_f64",
                [self, ss](double v, uint32_t ptr, uint32_t len) -> uint32_t {
                    const std::string spec = self->read_memory_string(ptr, len);
                    return ss->create(format_float_value(spec, v));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                MODULE_NAME,
                "format_str",
                [self, ss](uint32_t handle, uint32_t ptr, uint32_t len) -> uint32_t {
                    const std::string spec = self->read_memory_string(ptr, len);
                    return ss->create(format_str_value(spec, ss->get(handle)));
                }
            )
            .unwrap();
    }

    void set_wasm_context(wasmtime::Store *store, wasmtime::Memory *memory) override {
        this->store = store;
        this->memory = memory;
    }
};

}
