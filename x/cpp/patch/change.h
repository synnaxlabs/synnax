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

/// @brief Struct is the change that an update makes to a struct field: none, a merge of
/// the partial U, a replacement by the whole T, or a clear. A default Struct makes no
/// change. Its JSON form is an absent member for no change, null for a clear, an object
/// with only a "set" member for a replacement, and the partial otherwise.
template<typename U, typename T>
class Struct {
    struct Replacement {
        T value;
        bool operator==(const Replacement &other) const = default;
    };

    std::variant<std::monostate, std::nullptr_t, U, Replacement> state;

    explicit Struct(std::variant<std::monostate, std::nullptr_t, U, Replacement> state):
        state(std::move(state)) {}

public:
    Struct() = default;

    /// @brief returns the Struct that merges update into a field.
    static Struct merge(U update) {
        return Struct(decltype(state)(std::in_place_type<U>, std::move(update)));
    }

    /// @brief returns the Struct that replaces a field with value.
    static Struct replace(T value) { return Struct(Replacement{std::move(value)}); }

    /// @brief returns the Struct that clears an optional field.
    static Struct clear() { return Struct(nullptr); }

    /// @brief returns true if the Struct makes no change.
    [[nodiscard]] bool is_zero() const {
        return std::holds_alternative<std::monostate>(this->state);
    }

    /// @brief returns true if the Struct clears the field.
    [[nodiscard]] bool cleared() const {
        return std::holds_alternative<std::nullptr_t>(this->state);
    }

    /// @brief returns the partial that the Struct merges, or nullptr if it merges none.
    [[nodiscard]] const U *merged() const { return std::get_if<U>(&this->state); }

    /// @brief returns the value that the Struct replaces the field with, or nullptr if
    /// it does not replace the field.
    [[nodiscard]] const T *replaced() const {
        const auto *r = std::get_if<Replacement>(&this->state);
        return r == nullptr ? nullptr : &r->value;
    }

    /// @brief reads the Struct from the member field of the object that parser holds.
    /// Binds an error to the parser for an object that holds "set" and any other
    /// member.
    static Struct parse(x::json::Parser &parser, const std::string &field) {
        if (!parser.has(field)) return {};
        const auto j = parser.field<x::json::json>(field);
        if (j.is_null()) return clear();
        if (!j.is_object() || !j.contains(SET_FIELD))
            return merge(parser.field<U>(field));
        if (j.size() > 1) {
            parser.field_err(
                field,
                "an object that holds set cannot hold other members"
            );
            return {};
        }
        return replace(parser.child(field).field<T>(SET_FIELD));
    }

    /// @brief writes the Struct to the member field of j, or nothing if it makes no
    /// change.
    void to_json(x::json::json &j, const std::string &field) const {
        if (this->cleared())
            j[field] = nullptr;
        else if (const U *u = this->merged(); u != nullptr)
            j[field] = x::json::to_json(*u);
        else if (const T *v = this->replaced(); v != nullptr)
            j[field] = x::json::json{{SET_FIELD, x::json::to_json(*v)}};
    }

    bool operator==(const Struct &other) const = default;
};
}
