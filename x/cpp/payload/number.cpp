// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <charconv>
#include <cmath>
#include <limits>
#include <optional>
#include <string>
#include <system_error>
#include <utility>

#include "x/cpp/payload/number.h"

namespace x::payload {
namespace {
template<typename T>
std::optional<T> parse_digits(const x::json::json &j) {
    if (j.is_number_integer()) {
        if (j.is_number_unsigned()) {
            const auto u = j.get<std::uint64_t>();
            if (!std::in_range<T>(u)) return std::nullopt;
            return static_cast<T>(u);
        }
        const auto s = j.get<std::int64_t>();
        if (!std::in_range<T>(s)) return std::nullopt;
        return static_cast<T>(s);
    }
    if (!j.is_string()) return std::nullopt;
    const auto &s = j.get_ref<const std::string &>();
    T v{};
    const auto *end = s.data() + s.size();
    const auto [ptr, ec] = std::from_chars(s.data(), end, v);
    if (s.empty() || ec != std::errc() || ptr != end) return std::nullopt;
    return v;
}

template<typename T>
T parse_or_err(const x::json::Parser &parser, const char *name) {
    const auto j = parser.get_json();
    if (const auto v = parse_digits<T>(j); v.has_value()) return v.value();
    parser.field_err("", std::string("cannot parse ") + name + " from " + j.dump());
    return T{};
}
}

Float Float::parse(const x::json::Parser &parser) {
    const auto j = parser.get_json();
    if (j.is_number()) return Float{j.get<double>()};
    if (j.is_string()) {
        const auto &s = j.get_ref<const std::string &>();
        if (s == "NaN") return Float{std::numeric_limits<double>::quiet_NaN()};
        if (s == "Infinity") return Float{std::numeric_limits<double>::infinity()};
        if (s == "-Infinity") return Float{-std::numeric_limits<double>::infinity()};
    }
    parser.field_err("", "cannot parse a float from " + j.dump());
    return Float{};
}

x::json::json Float::to_json() const {
    if (std::isnan(this->value)) return "NaN";
    if (std::isinf(this->value)) return this->value > 0 ? "Infinity" : "-Infinity";
    return this->value;
}

Integer Integer::parse(const x::json::Parser &parser) {
    return Integer{parse_or_err<std::int64_t>(parser, "an integer")};
}

x::json::json Integer::to_json() const {
    return std::to_string(this->value);
}

Unsigned Unsigned::parse(const x::json::Parser &parser) {
    return Unsigned{parse_or_err<std::uint64_t>(parser, "an unsigned integer")};
}

x::json::json Unsigned::to_json() const {
    return std::to_string(this->value);
}

}
