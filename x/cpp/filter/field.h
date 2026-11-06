// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <concepts>
#include <initializer_list>
#include <optional>
#include <string>
#include <type_traits>
#include <utility>
#include <vector>

#include "x/cpp/filter/types.gen.h"

namespace x::filter {
namespace details {
/// @brief Shorthand reads the operator Op from a plain value or a list.
template<typename Op>
struct Shorthand;

template<>
struct Shorthand<String> {
    using plain = std::string;
    static String value(std::string v) { return {.eq = std::move(v)}; }
    static String list(std::vector<std::string> v) { return {.in = std::move(v)}; }
};

template<typename T>
struct Shorthand<Set<T>> {
    using plain = T;
    static Set<T> value(T v) { return {.eq = std::move(v)}; }
    static Set<T> list(std::vector<T> v) { return {.in = std::move(v)}; }
};

template<typename T>
struct Shorthand<Ordered<T>> {
    using plain = T;
    static Ordered<T> value(T v) { return {.eq = std::move(v)}; }
    static Ordered<T> list(std::vector<T> v) { return {.in = std::move(v)}; }
};

template<>
struct Shorthand<Bool> {
    using plain = bool;
    static Bool value(const bool v) { return {.eq = v}; }
};

template<typename T>
struct Shorthand<Contains<T>> {
    using plain = T;
    static Contains<T> value(T v) { return {.any = std::vector<T>{std::move(v)}}; }
    static Contains<T> list(std::vector<T> v) { return {.any = std::move(v)}; }
};

/// @brief takes is true when V converts to the plain value T. Only a bool converts to
/// bool, so a pointer or a number cannot become a Bool by accident.
template<typename V, typename T>
concept takes =
    (std::same_as<T, bool> ? std::same_as<std::remove_cvref_t<V>, bool>
                           : std::convertible_to<V, T>);

template<typename Op>
concept has_list = requires(std::vector<typename Shorthand<Op>::plain> v) {
    Shorthand<Op>::list(std::move(v));
};
}

// NOLINTBEGIN(misc-explicit-constructor): a designated initializer converts each value,
// as in `.name = "x"`.

/// @brief Field is a filter line that holds the operator Op. It takes the operator, a
/// plain value, or a list, so a designated initializer can write `.name = "x"` or
/// `.key = {1, 2}`. A plain value matches records whose field equals it, and a list
/// matches records whose field equals any element. On a Contains line, both match
/// records that hold any of them. Field keeps the std::optional interface.
template<typename Op>
class Field : public std::optional<Op> {
    using plain = details::Shorthand<Op>::plain;

public:
    Field() = default;
    Field(std::nullopt_t) {}
    Field(Op op): std::optional<Op>(std::move(op)) {}
    Field(std::optional<Op> op): std::optional<Op>(std::move(op)) {}

    template<typename V>
        requires details::takes<V, plain>
    Field(V &&v):
        std::optional<Op>(details::Shorthand<Op>::value(std::forward<V>(v))) {}

    Field(std::vector<plain> list)
        requires details::has_list<Op>
        : std::optional<Op>(details::Shorthand<Op>::list(std::move(list))) {}

    Field(std::initializer_list<plain> list)
        requires details::has_list<Op>
        : Field(std::vector<plain>(list)) {}
};

/// @brief List is a filter line that takes a list of T. It also takes one value as a
/// list of one, so a designated initializer can write `.labels = label.key`. List keeps
/// the std::optional interface.
template<typename T>
class List : public std::optional<std::vector<T>> {
public:
    List() = default;
    List(std::nullopt_t) {}
    List(std::optional<std::vector<T>> values):
        std::optional<std::vector<T>>(std::move(values)) {}

    List(std::vector<T> values): std::optional<std::vector<T>>(std::move(values)) {}

    List(std::initializer_list<T> values):
        std::optional<std::vector<T>>(std::vector<T>(values)) {}

    template<typename V>
        requires details::takes<V, T>
    List(V &&value):
        std::optional<std::vector<T>>(std::vector<T>{T(std::forward<V>(value))}) {}
};
// NOLINTEND(misc-explicit-constructor)
}
