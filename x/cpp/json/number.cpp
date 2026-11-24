// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <cerrno>
#include <charconv>
#include <cmath>
#include <cstdint>
#include <cstdlib>
#include <limits>
#include <optional>
#include <string>
#include <string_view>
#include <system_error>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/decimal.h"
#include "x/cpp/json/number.h"
#include "x/cpp/json/text.h"
#include "x/cpp/strings/strings.h"
#include "x/cpp/telem/telem.h"

namespace x::json {
namespace {
constexpr auto UINT64_MAX_VALUE = std::numeric_limits<std::uint64_t>::max();
constexpr auto INT64_MAX_VALUE = static_cast<std::uint64_t>(
    std::numeric_limits<std::int64_t>::max()
);
constexpr std::uint64_t INT64_MIN_MAGNITUDE = INT64_MAX_VALUE + 1;
/// @brief bounds the exponent parse_decimal reads. Any larger one overflows or
/// truncates to zero.
constexpr int MAX_EXPONENT = 1 << 20;
/// @brief the whitespace that parse_quoted trims from each end of its text.
constexpr std::string_view QUOTED_SPACE = " \t\n\v\f\r";

char lower(const char c) {
    return c >= 'A' && c <= 'Z' ? static_cast<char>(c | 0x20) : c;
}

/// @brief returns true when text equals word, which is lowercase, in any case.
bool equal_fold(const std::string_view text, const std::string_view word) {
    if (text.size() != word.size()) return false;
    for (std::size_t i = 0; i < text.size(); i++)
        if (lower(text[i]) != word[i]) return false;
    return true;
}

bool is_alphanumeric(const char c) {
    return (c >= '0' && c <= '9') || (lower(c) >= 'a' && lower(c) <= 'z');
}

bool is_hex_digits(const std::string_view text) {
    for (const char c: text)
        if ((c < '0' || c > '9') && (lower(c) < 'a' || lower(c) > 'f')) return false;
    return true;
}

/// @brief returns true when text is nan or nan(chars) in any case, where chars are
/// letters, digits, and underscores.
bool is_nan(const std::string_view text) {
    if (text.size() < 3 || !equal_fold(text.substr(0, 3), "nan")) return false;
    const auto rest = text.substr(3);
    if (rest.empty()) return true;
    if (rest.size() < 2 || rest.front() != '(' || rest.back() != ')') return false;
    for (const char c: rest.substr(1, rest.size() - 2))
        if (!is_alphanumeric(c) && c != '_') return false;
    return true;
}

/// @brief returns the value of digits, hex digits, or nothing when it does not fit a
/// uint64.
std::optional<std::uint64_t> hex_magnitude(const std::string_view digits) {
    std::uint64_t u = 0;
    for (const char c: digits) {
        if (u > (UINT64_MAX_VALUE >> 4)) return std::nullopt;
        const auto v = c <= '9' ? c - '0' : lower(c) - 'a' + 10;
        u = u << 4 | static_cast<std::uint64_t>(v);
    }
    return u;
}

bool is_digits(const std::string_view text) {
    for (const char c: text)
        if (c < '0' || c > '9') return false;
    return true;
}

bool has_sign(const std::string_view text) {
    return !text.empty() && (text[0] == '-' || text[0] == '+');
}

std::optional<int> parse_exponent(std::string_view text) {
    bool negative = false;
    if (has_sign(text)) {
        negative = text[0] == '-';
        text.remove_prefix(1);
    }
    if (text.empty() || !is_digits(text)) return std::nullopt;
    int e = 0;
    for (const char c: text)
        e = std::min(e * 10 + (c - '0'), MAX_EXPONENT);
    return negative ? -e : e;
}

template<typename T>
std::optional<T> parse_integer(const std::string_view text) {
    auto digits = text;
    if (!digits.empty() && digits[0] == '+') {
        digits.remove_prefix(1);
        if (!digits.empty() && digits[0] == '-') return std::nullopt;
    }
    T v{};
    const auto *end = digits.data() + digits.size();
    const auto [ptr, ec] = std::from_chars(digits.data(), end, v);
    if (ec != std::errc() || ptr != end) return std::nullopt;
    return v;
}

/// @brief parses text, which parse_decimal accepts, as the nearest double. It returns
/// nothing when the number is past the range of a double.
std::optional<double> parse_double(const std::string_view text) {
    std::array<char, 64> buf{};
    std::string long_text;
    const char *terminated = buf.data();
    if (text.size() < buf.size())
        text.copy(buf.data(), text.size());
    else {
        long_text = text;
        terminated = long_text.c_str();
    }
    errno = 0;
    char *end = nullptr;
    const double v = std::strtod(terminated, &end);
    if (end != terminated + text.size()) return std::nullopt;
    if (errno == ERANGE && std::isinf(v)) return std::nullopt;
    return v;
}

/// @brief returns the range of the integer data type dt, or nothing when dt is not an
/// integer type.
std::optional<std::pair<std::int64_t, std::uint64_t>>
integer_bounds(const telem::DataType &dt) {
    if (dt == telem::INT8_T)
        return {
            {std::numeric_limits<std::int8_t>::min(),
             std::numeric_limits<std::int8_t>::max()}
        };
    if (dt == telem::INT16_T)
        return {
            {std::numeric_limits<std::int16_t>::min(),
             std::numeric_limits<std::int16_t>::max()}
        };
    if (dt == telem::INT32_T)
        return {
            {std::numeric_limits<std::int32_t>::min(),
             std::numeric_limits<std::int32_t>::max()}
        };
    if (dt == telem::INT64_T)
        return {{std::numeric_limits<std::int64_t>::min(), INT64_MAX_VALUE}};
    if (dt == telem::UINT8_T) return {{0, std::numeric_limits<std::uint8_t>::max()}};
    if (dt == telem::UINT16_T) return {{0, std::numeric_limits<std::uint16_t>::max()}};
    if (dt == telem::UINT32_T) return {{0, std::numeric_limits<std::uint32_t>::max()}};
    if (dt == telem::UINT64_T) return {{0, UINT64_MAX_VALUE}};
    return std::nullopt;
}
}

std::optional<Decimal> parse_decimal(std::string_view text, const int scale) {
    Decimal d;
    if (has_sign(text)) {
        d.negative = text[0] == '-';
        text.remove_prefix(1);
    }
    int exp = 0;
    if (const auto i = text.find_first_of("eE"); i != std::string_view::npos) {
        const auto e = parse_exponent(text.substr(i + 1));
        if (!e.has_value()) return std::nullopt;
        exp = *e;
        text = text.substr(0, i);
    }
    auto whole = text;
    std::string_view frac;
    if (const auto i = text.find('.'); i != std::string_view::npos) {
        whole = text.substr(0, i);
        frac = text.substr(i + 1);
    }
    if (whole.size() + frac.size() == 0 || !is_digits(whole) || !is_digits(frac))
        return std::nullopt;
    // The value is the digits of whole and frac times 10^shift.
    const int count = static_cast<int>(whole.size() + frac.size());
    const int shift = exp + scale - static_cast<int>(frac.size());
    for (int i = 0; i < count; i++) {
        const auto w = static_cast<int>(whole.size());
        const auto c = static_cast<std::uint64_t>(
            (i < w ? whole[i] : frac[i - w]) - '0'
        );
        if (i >= count + shift) {
            d.fraction = d.fraction || c != 0;
            continue;
        }
        if (d.magnitude > (UINT64_MAX_VALUE - c) / 10) {
            d.overflow = true;
            return d;
        }
        d.magnitude = d.magnitude * 10 + c;
    }
    for (int i = 0; d.magnitude != 0 && i < shift; i++) {
        if (d.magnitude > UINT64_MAX_VALUE / 10) {
            d.overflow = true;
            return d;
        }
        d.magnitude *= 10;
    }
    return d;
}

errors::Error invalid_number(const std::string_view text) {
    return errors::Error(errors::VALIDATION, "invalid number " + quote(text));
}

Number::Number(const double v): kind(Kind::Float), f(v) {}

Number::Number(const std::int64_t v): i(v) {}

Number::Number(const std::uint64_t v): kind(Kind::Uint), u(v) {}

std::pair<Number, errors::Error> Number::parse(const std::string_view text) {
    Number n;
    if (text.find_first_of(".eE") == std::string_view::npos) {
        if (const auto i = parse_integer<std::int64_t>(text); i.has_value()) {
            n.i = *i;
            return {std::move(n), errors::NIL};
        }
        if (const auto u = parse_integer<std::uint64_t>(text); u.has_value()) {
            n.kind = Kind::Uint;
            n.u = *u;
            return {std::move(n), errors::NIL};
        }
    }
    const auto d = parse_decimal(text, 0);
    if (!d.has_value()) return {Number{}, invalid_number(text)};
    // A negative zero keeps its sign as a float.
    if (!d->fraction && !d->overflow && d->magnitude != 0) {
        if (!d->negative && d->magnitude <= INT64_MAX_VALUE) {
            n.i = static_cast<std::int64_t>(d->magnitude);
            return {std::move(n), errors::NIL};
        }
        if (!d->negative) {
            n.kind = Kind::Uint;
            n.u = d->magnitude;
            return {std::move(n), errors::NIL};
        }
        if (d->magnitude <= INT64_MIN_MAGNITUDE) {
            n.i = static_cast<std::int64_t>(0 - d->magnitude);
            return {std::move(n), errors::NIL};
        }
    }
    const auto v = parse_double(text);
    if (!v.has_value()) return {Number{}, invalid_number(text)};
    n.kind = Kind::Float;
    n.f = *v;
    n.fraction = d->fraction;
    n.out_of_range = !d->fraction && d->magnitude != 0;
    n.text = text;
    return {std::move(n), errors::NIL};
}

std::pair<Number, errors::Error> Number::parse_quoted(const std::string_view text) {
    auto trimmed = text;
    trimmed.remove_prefix(
        std::min(trimmed.find_first_not_of(QUOTED_SPACE), trimmed.size())
    );
    trimmed.remove_suffix(
        trimmed.size() - (trimmed.find_last_not_of(QUOTED_SPACE) + 1)
    );
    auto body = trimmed;
    const bool negative = has_sign(body) && body[0] == '-';
    if (has_sign(body)) body.remove_prefix(1);
    Number n;
    n.kind = Kind::Float;
    n.text = trimmed;
    if (equal_fold(body, "inf") || equal_fold(body, "infinity")) {
        n.f = negative ? -std::numeric_limits<double>::infinity()
                       : std::numeric_limits<double>::infinity();
        return {std::move(n), errors::NIL};
    }
    if (is_nan(body)) {
        n.f = std::numeric_limits<double>::quiet_NaN();
        return {std::move(n), errors::NIL};
    }
    if (body.size() < 2 || body[0] != '0' || lower(body[1]) != 'x')
        return parse(trimmed);
    // The text after the 0x is hex digits with an optional point, then an optional
    // binary exponent after p.
    auto mantissa = body.substr(2);
    bool has_exponent = false;
    if (const auto i = mantissa.find_first_of("pP"); i != std::string_view::npos) {
        if (!parse_exponent(mantissa.substr(i + 1)).has_value())
            return {Number{}, invalid_number(trimmed)};
        mantissa = mantissa.substr(0, i);
        has_exponent = true;
    }
    auto whole = mantissa;
    std::string_view frac;
    const auto point = mantissa.find('.');
    if (point != std::string_view::npos) {
        whole = mantissa.substr(0, point);
        frac = mantissa.substr(point + 1);
    }
    if (whole.size() + frac.size() == 0 || !is_hex_digits(whole) ||
        !is_hex_digits(frac))
        return {Number{}, invalid_number(trimmed)};
    if (point == std::string_view::npos && !has_exponent) {
        if (const auto u = hex_magnitude(whole); u.has_value()) {
            if (!negative && *u <= INT64_MAX_VALUE)
                return {Number(static_cast<std::int64_t>(*u)), errors::NIL};
            if (!negative) return {Number(*u), errors::NIL};
            if (*u <= INT64_MIN_MAGNITUDE)
                return {Number(static_cast<std::int64_t>(0 - *u)), errors::NIL};
        }
    }
    const auto v = parse_double(trimmed);
    if (!v.has_value()) return {Number{}, invalid_number(trimmed)};
    n.f = *v;
    n.out_of_range = point == std::string_view::npos && !has_exponent;
    return {std::move(n), errors::NIL};
}

std::pair<double, errors::Error> parse_float64(const std::string_view text) {
    if (!parse_decimal(text, 0).has_value()) return {0, invalid_number(text)};
    const auto v = parse_double(text);
    if (!v.has_value()) return {0, invalid_number(text)};
    return {*v, errors::NIL};
}

std::optional<std::int64_t> Number::to_int64() const {
    switch (this->kind) {
        case Kind::Int:
            return this->i;
        case Kind::Uint:
            if (this->u > INT64_MAX_VALUE) return std::nullopt;
            return static_cast<std::int64_t>(this->u);
        case Kind::Float:
            if (!this->integral() || !this->within(
                                         std::numeric_limits<std::int64_t>::min(),
                                         INT64_MAX_VALUE
                                     ))
                return std::nullopt;
            return static_cast<std::int64_t>(this->f);
    }
    return std::nullopt;
}

std::optional<std::uint64_t> Number::to_uint64() const {
    if (!this->integral() || !this->within(0, UINT64_MAX_VALUE)) return std::nullopt;
    switch (this->kind) {
        case Kind::Int:
            return static_cast<std::uint64_t>(this->i);
        case Kind::Uint:
            return this->u;
        case Kind::Float:
            return static_cast<std::uint64_t>(this->f);
    }
    return std::nullopt;
}

double Number::to_float64() const {
    switch (this->kind) {
        case Kind::Int:
            return static_cast<double>(this->i);
        case Kind::Uint:
            return static_cast<double>(this->u);
        case Kind::Float:
            return this->f;
    }
    return 0;
}

errors::Error Number::check_fits(const telem::DataType &dt) const {
    if (this->kind == Kind::Float && dt == telem::FLOAT32_T && !std::isinf(this->f) &&
        std::abs(this->f) > std::numeric_limits<float>::max())
        return errors::Error(
            errors::VALIDATION,
            this->to_string() + " overflows " + dt.name()
        );
    const auto bounds = integer_bounds(dt);
    if (!bounds.has_value()) return errors::NIL;
    if (!this->integral())
        return errors::Error(
            errors::VALIDATION,
            this->to_string() + " is not an integer"
        );
    if (!this->within(bounds->first, bounds->second))
        return errors::Error(
            errors::VALIDATION,
            this->to_string() + " is out of range for " + dt.name()
        );
    return errors::NIL;
}

std::string Number::to_string() const {
    switch (this->kind) {
        case Kind::Int:
            return std::to_string(this->i);
        case Kind::Uint:
            return std::to_string(this->u);
        case Kind::Float:
            if (!this->text.empty()) return this->text;
            return x::strings::format_float(this->f);
    }
    return "";
}

errors::Error append_float(const double v, std::string &out) {
    if (!std::isfinite(v))
        return errors::Error(
            errors::VALIDATION,
            Number(v).to_string() + " is not a JSON number"
        );
    const double abs = std::abs(v);
    const auto format = abs != 0 && (abs < 1e-6 || abs >= 1e21)
                          ? std::chars_format::scientific
                          : std::chars_format::fixed;
    std::array<char, 64> buf{};
    const auto res = std::to_chars(buf.data(), buf.data() + buf.size(), v, format);
    const std::string_view text(buf.data(), res.ptr - buf.data());
    // to_chars pads an exponent to two digits, as in e-07.
    if (const auto n = text.size(); format == std::chars_format::scientific && n >= 4 &&
                                    text.substr(n - 4, 3) == "e-0") {
        out += text.substr(0, n - 2);
        out += text.back();
        return errors::NIL;
    }
    out += text;
    return errors::NIL;
}

bool Number::integral() const {
    return this->kind != Kind::Float ||
           (!this->fraction && this->f == std::trunc(this->f));
}

bool Number::within(const std::int64_t lo, const std::uint64_t hi) const {
    switch (this->kind) {
        case Kind::Int:
            return this->i >= lo && std::cmp_less_equal(this->i, hi);
        case Kind::Uint:
            return this->u <= hi;
        case Kind::Float:
            // static_cast<double>(hi) + 1 rounds to the first value past hi for every
            // integer type.
            return !this->out_of_range && this->f >= static_cast<double>(lo) &&
                   this->f < static_cast<double>(hi) + 1;
    }
    return false;
}
}
