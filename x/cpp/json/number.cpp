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

/// @brief formats v, which is not infinite, as the Core does: the shortest digits that
/// round trip, with an exponent below 1e-4 or from 1e6.
std::string format_shortest(const double v) {
    if (std::isnan(v)) return "NaN";
    char buf[32];
    const auto
        res = std::to_chars(buf, buf + sizeof(buf), v, std::chars_format::scientific);
    const std::string_view sci(buf, res.ptr - buf);
    const auto e = sci.find('e');
    std::string_view mantissa = sci.substr(0, e);
    auto exp_text = sci.substr(e + 1);
    if (exp_text[0] == '+') exp_text.remove_prefix(1);
    int exp = 0;
    std::from_chars(exp_text.data(), exp_text.data() + exp_text.size(), exp);
    std::string out;
    if (mantissa[0] == '-') {
        out += '-';
        mantissa.remove_prefix(1);
    }
    std::string digits(1, mantissa[0]);
    if (mantissa.size() > 2) digits += mantissa.substr(2);
    if (exp < -4 || exp >= 6) {
        out += digits[0];
        if (digits.size() > 1) out += "." + digits.substr(1);
        out += exp < 0 ? "e-" : "e+";
        const int mag = std::abs(exp);
        if (mag < 10) out += '0';
        return out + std::to_string(mag);
    }
    if (exp < 0) return out + "0." + std::string(-exp - 1, '0') + digits;
    const auto point = static_cast<std::size_t>(exp) + 1;
    if (digits.size() <= point)
        return out + digits + std::string(point - digits.size(), '0');
    return out + digits.substr(0, point) + "." + digits.substr(point);
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
    n.text = text;
    return {std::move(n), errors::NIL};
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
            if (std::isinf(this->f)) return this->f > 0 ? "Infinity" : "-Infinity";
            return format_shortest(this->f);
    }
    return "";
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
            return this->f >= static_cast<double>(lo) &&
                   this->f < static_cast<double>(hi) + 1;
    }
    return false;
}
}
