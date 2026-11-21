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
#include <type_traits>
#include <unordered_map>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/number.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/json/text.h"

#include "driver/codec/errors.h"
#include "driver/codec/json.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

/// @brief maps key to message in m. It returns the message that already held key.
template<typename Map, typename Key>
std::optional<std::size_t> claim(Map &m, const Key &key, const std::size_t message) {
    const auto [it, inserted] = m.try_emplace(key, message);
    if (inserted) return std::nullopt;
    return it->second;
}
}

double JSON::Integer::to_float64() const {
    if (this->negative)
        return static_cast<double>(static_cast<std::int64_t>(this->bits));
    return static_cast<double>(this->bits);
}

std::string JSON::Integer::to_string() const {
    if (this->negative) return std::to_string(static_cast<std::int64_t>(this->bits));
    return std::to_string(this->bits);
}

std::pair<JSON::Router, x::errors::Error>
JSON::Router::create(const std::vector<JSONMessage> &messages) {
    Router r;
    std::optional<std::size_t> first;
    std::string first_pointer;
    std::unordered_map<double, Integer> rounded;
    for (std::size_t i = 0; i < messages.size(); i++) {
        const auto &m = messages[i];
        if (!m.payload.match.has_value()) {
            if (r.fallback.has_value())
                return {
                    Router{},
                    x::errors::Error(
                        CONFIG_ERROR,
                        "messages " + messages[*r.fallback].name + " and " + m.name +
                            " both have no match rule"
                    ),
                };
            r.fallback = i;
            continue;
        }
        const auto &match = *m.payload.match;
        auto [pointer, err] = x::json::Pointer::parse(icd::base(match).pointer);
        if (err)
            return {
                Router{},
                x::errors::Error(
                    CONFIG_ERROR,
                    "message " + m.name + ": match: " + err.data
                ),
            };
        if (!first.has_value()) {
            first = i;
            first_pointer = icd::base(match).pointer;
            r.pointer = std::move(pointer);
        } else if (pointer != r.pointer)
            return {
                Router{},
                x::errors::Error(
                    CONFIG_ERROR,
                    "messages " + messages[*first].name + " and " + m.name +
                        " match at different pointers " +
                        x::json::quote(first_pointer) + " and " +
                        x::json::quote(icd::base(match).pointer)
                ),
            };
        const auto [taken, value] = r.claim_match(match, i, rounded);
        if (taken.has_value())
            return {
                Router{},
                x::errors::Error(
                    CONFIG_ERROR,
                    "messages " + messages[*taken].name + " and " + m.name +
                        " match the same value " + value
                ),
            };
    }
    r.ruled = first.has_value();
    return {std::move(r), x::errors::NIL};
}

std::pair<std::optional<std::size_t>, std::string> JSON::Router::claim_match(
    const icd::JSONMatch &match,
    const std::size_t message,
    std::unordered_map<double, Integer> &rounded
) {
    return std::visit(
        [&]<typename M>(const M &v)
            -> std::pair<std::optional<std::size_t>, std::string> {
            if constexpr (std::is_same_v<M, icd::StringJSONMatch>)
                return {
                    claim(this->strings, v.value, message),
                    x::json::quote(v.value),
                };
            else if constexpr (std::is_same_v<M, icd::BooleanJSONMatch>) {
                auto &held = this->booleans[v.value ? 1 : 0];
                if (held.has_value()) return {held, v.value ? "true" : "false"};
                held = message;
                return {std::nullopt, v.value ? "true" : "false"};
            } else if constexpr (std::is_same_v<M, icd::IntegerJSONMatch>) {
                const Integer k{
                    .bits = static_cast<std::uint64_t>(v.value.value),
                    .negative = v.value.value < 0,
                };
                return {this->claim_integer(k, message, rounded), k.to_string()};
            } else if constexpr (std::is_same_v<M, icd::UnsignedJSONMatch>) {
                const Integer k{.bits = v.value.value};
                return {this->claim_integer(k, message, rounded), k.to_string()};
            } else {
                static_assert(std::is_same_v<M, icd::FloatJSONMatch>);
                const double f = v.value.value;
                const auto taken = claim(this->floats, f, message);
                if (const auto k = rounded.find(f); k != rounded.end() && !taken)
                    return {this->integers.at(k->second), k->second.to_string()};
                return {taken, x::json::Number(f).to_string()};
            }
        },
        match
    );
}

std::optional<std::size_t> JSON::Router::claim_integer(
    const Integer k,
    const std::size_t message,
    std::unordered_map<double, Integer> &rounded
) {
    auto taken = claim(this->integers, k, message);
    if (const auto f = this->floats.find(k.to_float64());
        f != this->floats.end() && !taken)
        taken = f->second;
    rounded.try_emplace(k.to_float64(), k);
    return taken;
}

std::pair<std::optional<std::size_t>, x::errors::Error>
JSON::Router::find(const std::optional<std::string_view> &value) {
    std::optional<std::size_t> message;
    if (value.has_value()) {
        const char kind = value->front();
        if (kind == '"') {
            if (const auto err = x::json::unquote(*value, this->text))
                return {std::nullopt, err};
            if (const auto it = this->strings.find(this->text);
                it != this->strings.end())
                message = it->second;
        } else if (kind == 't' || kind == 'f')
            message = this->booleans[kind == 't' ? 1 : 0];
        // With no number rules, any number is no match, even one that does not parse.
        else if (
            kind != 'n' && kind != '{' && kind != '[' &&
            (!this->integers.empty() || !this->floats.empty())
        ) {
            const auto [n, err] = x::json::Number::parse(*value);
            if (err)
                return {
                    std::nullopt,
                    x::errors::Error(err, "match: " + err.data),
                };
            std::optional<Integer> k;
            if (const auto i = n.to_int64(); i.has_value())
                k = Integer{.bits = static_cast<std::uint64_t>(*i), .negative = *i < 0};
            else if (const auto u = n.to_uint64(); u.has_value())
                k = Integer{.bits = *u};
            if (k.has_value())
                if (const auto it = this->integers.find(*k); it != this->integers.end())
                    message = it->second;
            if (!message.has_value())
                if (const auto it = this->floats.find(n.to_float64());
                    it != this->floats.end())
                    message = it->second;
        }
    }
    if (!message.has_value()) message = this->fallback;
    return {message, x::errors::NIL};
}
}
