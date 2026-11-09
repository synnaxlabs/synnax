// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <string>
#include <utility>
#include <variant>

#include "x/cpp/json/json.h"

namespace x::patch {
/// @brief the member of the JSON form of a Struct that replaces the field whole.
inline constexpr auto SET_FIELD = "set";

/// @brief Change is the change that an update makes to one field: none, a new value,
/// or a clear. A default Change makes no change. Its JSON form is an absent member for
/// no change, null for a clear, and the value otherwise, so a parent reads and writes
/// it by field name.
template<typename T>
class Change {
    std::variant<std::monostate, std::nullptr_t, T> state;

    explicit Change(std::variant<std::monostate, std::nullptr_t, T> state):
        state(std::move(state)) {}

public:
    Change() = default;

    /// @brief returns the Change that sets a field to value.
    static Change to(T value) {
        return Change(decltype(state)(std::in_place_type<T>, std::move(value)));
    }

    /// @brief returns the Change that clears an optional field.
    static Change clear() { return Change(nullptr); }

    /// @brief returns true if the Change makes no change.
    [[nodiscard]] bool is_zero() const {
        return std::holds_alternative<std::monostate>(this->state);
    }

    /// @brief returns true if the Change clears the field.
    [[nodiscard]] bool cleared() const {
        return std::holds_alternative<std::nullptr_t>(this->state);
    }

    /// @brief returns the value that the Change sets, or nullptr if it sets none.
    [[nodiscard]] const T *get() const { return std::get_if<T>(&this->state); }

    /// @brief reads the Change from the member field of the object that parser holds.
    static Change parse(x::json::Parser &parser, const std::string &field) {
        if (!parser.has(field)) return {};
        if (parser.field<x::json::json>(field).is_null()) return clear();
        return to(parser.field<T>(field));
    }

    /// @brief writes the Change to the member field of j, or nothing if it makes no
    /// change.
    void to_json(x::json::json &j, const std::string &field) const {
        if (this->cleared())
            j[field] = nullptr;
        else if (const T *v = this->get(); v != nullptr)
            j[field] = x::json::to_json(*v);
    }

    bool operator==(const Change &other) const = default;
};

/// @brief Struct is the change that an update makes to a struct field: a merge of the
/// partial U, or a replacement by the whole T. A default Struct merges a default U. Its
/// JSON form is an object with only a "set" member for a replacement, and the partial
/// otherwise. A required struct field of an update holds a std::optional<Struct>, and
/// an optional one holds a Change<Struct>, whose null clears the field.
template<typename U, typename T>
class Struct {
    struct Replacement {
        T value;
        bool operator==(const Replacement &other) const = default;
    };

    std::variant<U, Replacement> state;

    explicit Struct(std::variant<U, Replacement> state): state(std::move(state)) {}

public:
    Struct() = default;

    /// @brief returns the Struct that merges update into a field.
    static Struct merge(U update) {
        return Struct(decltype(state)(std::in_place_type<U>, std::move(update)));
    }

    /// @brief returns the Struct that replaces a field with value.
    static Struct replace(T value) { return Struct(Replacement{std::move(value)}); }

    /// @brief returns the partial that the Struct merges, or nullptr if it replaces the
    /// field.
    [[nodiscard]] const U *merged() const { return std::get_if<U>(&this->state); }

    /// @brief returns the value that the Struct replaces the field with, or nullptr if
    /// it merges a partial.
    [[nodiscard]] const T *replaced() const {
        const auto *r = std::get_if<Replacement>(&this->state);
        return r == nullptr ? nullptr : &r->value;
    }

    /// @brief parses the Struct from the object that parser holds. Binds an error to
    /// the parser for a value that is not an object, or for an object that holds "set"
    /// and any other member.
    static Struct parse(x::json::Parser parser) {
        const auto j = parser.field<x::json::json>();
        if (!j.is_object()) {
            parser.field_err("", "expected an object");
            return {};
        }
        if (!j.contains(SET_FIELD)) return merge(parser.field<U>());
        if (j.size() > 1) {
            parser.field_err("", "an object that holds set cannot hold other members");
            return {};
        }
        return replace(parser.field<T>(SET_FIELD));
    }

    /// @brief returns the JSON form of the Struct.
    [[nodiscard]] x::json::json to_json() const {
        if (const T *v = this->replaced(); v != nullptr)
            return x::json::json{{SET_FIELD, x::json::to_json(*v)}};
        return x::json::to_json(*this->merged());
    }

    bool operator==(const Struct &other) const = default;
};
}
