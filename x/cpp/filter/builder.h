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
#include <cstddef>
#include <cstdint>
#include <optional>
#include <string>
#include <tuple>
#include <type_traits>
#include <utility>
#include <vector>

#include "x/cpp/filter/types.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/mem/indirect.h"

namespace x::filter {
/// @brief Condition is a condition on the records of the entity whose filter is W.
/// Build one from the lines of an entity, such as synnax::ranger::name.eq("Hotfire").
template<typename W>
struct Condition {
    /// @brief tree is the filter that the condition holds.
    W tree;
};

/// @brief and_ matches the records that every condition matches.
template<typename W, std::same_as<Condition<W>>... Rest>
[[nodiscard]] Condition<W> and_(Condition<W> first, Rest... rest) {
    W tree;
    tree.and_ = std::vector<W>{std::move(first.tree), std::move(rest.tree)...};
    return {std::move(tree)};
}

/// @brief or_ matches the records that any condition matches.
template<typename W, std::same_as<Condition<W>>... Rest>
[[nodiscard]] Condition<W> or_(Condition<W> first, Rest... rest) {
    W tree;
    tree.or_ = std::vector<W>{std::move(first.tree), std::move(rest.tree)...};
    return {std::move(tree)};
}

/// @brief not_ matches the records that condition does not match.
template<typename W>
[[nodiscard]] Condition<W> not_(Condition<W> condition) {
    W tree;
    tree.not_ = x::mem::indirect<W>(std::move(condition.tree));
    return {std::move(tree)};
}

/// @brief matches the records that both conditions match.
template<typename W>
[[nodiscard]] Condition<W> operator&&(Condition<W> a, Condition<W> b) {
    return and_(std::move(a), std::move(b));
}

/// @brief matches the records that either condition matches.
template<typename W>
[[nodiscard]] Condition<W> operator||(Condition<W> a, Condition<W> b) {
    return or_(std::move(a), std::move(b));
}

/// @brief matches the records that condition does not match.
template<typename W>
[[nodiscard]] Condition<W> operator!(Condition<W> condition) {
    return not_(std::move(condition));
}

namespace details {
template<typename>
struct member;

template<typename W, typename T>
struct member<std::optional<T> W::*> {
    using where = W;
    using operand = T;
};

template<auto Field>
using where_of = member<decltype(Field)>::where;

template<auto Field>
using operand_of = member<decltype(Field)>::operand;

template<auto Field>
[[nodiscard]] Condition<where_of<Field>> make(operand_of<Field> operand) {
    where_of<Field> tree;
    tree.*Field = std::move(operand);
    return {std::move(tree)};
}
}

/// @brief StringOptions are the options of a condition on a text line.
struct StringOptions {
    /// @brief case_insensitive makes the condition ignore case.
    bool case_insensitive = false;
};

/// @brief SetLine is a line of the operator type Set, where Field is the line's member
/// of the entity's filter.
template<auto Field>
struct SetLine {
    /// @brief value_type is the type of the line's values.
    using value_type = decltype(details::operand_of<Field>::eq)::value_type;

    /// @brief matches a value equal to value.
    [[nodiscard]] auto eq(value_type value) const {
        return details::make<Field>({.eq = std::move(value)});
    }

    /// @brief matches a value equal to an element of values.
    [[nodiscard]] auto in(std::vector<value_type> values) const {
        return details::make<Field>({.in = std::move(values)});
    }
};

/// @brief BoolLine is a line of the operator type Bool.
template<auto Field>
struct BoolLine {
    /// @brief matches a value equal to value.
    [[nodiscard]] auto eq(const bool value) const {
        return details::make<Field>({.eq = value});
    }
};

/// @brief OrderedLine is a line of the operator type Ordered.
template<auto Field>
struct OrderedLine {
    /// @brief value_type is the type of the line's values.
    using value_type = decltype(details::operand_of<Field>::eq)::value_type;

    /// @brief matches a value equal to value.
    [[nodiscard]] auto eq(value_type value) const {
        return details::make<Field>({.eq = std::move(value)});
    }

    /// @brief matches a value equal to an element of values.
    [[nodiscard]] auto in(std::vector<value_type> values) const {
        return details::make<Field>({.in = std::move(values)});
    }

    /// @brief matches a value greater than value.
    [[nodiscard]] auto gt(value_type value) const {
        return details::make<Field>({.gt = std::move(value)});
    }

    /// @brief matches a value greater than or equal to value.
    [[nodiscard]] auto gte(value_type value) const {
        return details::make<Field>({.gte = std::move(value)});
    }

    /// @brief matches a value less than value.
    [[nodiscard]] auto lt(value_type value) const {
        return details::make<Field>({.lt = std::move(value)});
    }

    /// @brief matches a value less than or equal to value.
    [[nodiscard]] auto lte(value_type value) const {
        return details::make<Field>({.lte = std::move(value)});
    }
};

/// @brief StringLine is a line of the operator type String. Each operator ignores case
/// when options.case_insensitive is true.
template<auto Field>
struct StringLine {
    /// @brief matches a value equal to value.
    [[nodiscard]] auto
    eq(const std::string &value, const StringOptions options = {}) const {
        return details::make<Field>({
            .eq = value,
            .case_insensitive = options.case_insensitive,
        });
    }

    /// @brief matches a value equal to an element of values.
    [[nodiscard]] auto
    in(const std::vector<std::string> &values, const StringOptions options = {}) const {
        return details::make<Field>({
            .in = values,
            .case_insensitive = options.case_insensitive,
        });
    }

    /// @brief matches a value that holds value.
    [[nodiscard]] auto
    contains(const std::string &value, const StringOptions options = {}) const {
        return details::make<Field>({
            .contains = value,
            .case_insensitive = options.case_insensitive,
        });
    }

    /// @brief matches a value that begins with value.
    [[nodiscard]] auto
    starts_with(const std::string &value, const StringOptions options = {}) const {
        return details::make<Field>({
            .starts_with = value,
            .case_insensitive = options.case_insensitive,
        });
    }

    /// @brief matches a value that the unanchored RE2 pattern matches.
    [[nodiscard]] auto
    regex(const std::string &pattern, const StringOptions options = {}) const {
        return details::make<Field>({
            .regex = pattern,
            .case_insensitive = options.case_insensitive,
        });
    }
};

/// @brief ContainsLine is a line of the operator type Contains.
template<auto Field>
struct ContainsLine {
    /// @brief value_type is the type of the elements of the line's arrays.
    using value_type = decltype(details::operand_of<
                                Field>::any)::value_type::value_type;

    /// @brief matches an array that holds value.
    [[nodiscard]] auto has(value_type value) const {
        return details::make<Field>({.any = std::vector<value_type>{std::move(value)}});
    }

    /// @brief matches an array that holds an element of values.
    [[nodiscard]] auto has_any(std::vector<value_type> values) const {
        return details::make<Field>({.any = std::move(values)});
    }
};

/// @brief FlagLine is a typed line of type bool, whose value goes on the line.
template<auto Field>
struct FlagLine {
    /// @brief matches the records that the line holds for value.
    [[nodiscard]] auto eq(const bool value) const {
        return details::make<Field>(value);
    }
};

/// @brief ListLine is a typed line of a list type, whose values go on the line.
template<auto Field>
struct ListLine {
    /// @brief value_type is the type of the elements of the line's list.
    using value_type = details::operand_of<Field>::value_type;

    /// @brief matches the records that the line holds for value.
    [[nodiscard]] auto has(value_type value) const {
        return details::make<Field>(std::vector<value_type>{std::move(value)});
    }

    /// @brief matches the records that the line holds for any element of values.
    [[nodiscard]] auto has_any(std::vector<value_type> values) const {
        return details::make<Field>(std::move(values));
    }
};

/// @brief ValueLine is a typed line, called with the value that goes on the line.
template<auto Field>
struct ValueLine {
    /// @brief matches the records that the line holds for value.
    [[nodiscard]] auto operator()(details::operand_of<Field> value) const {
        return details::make<Field>(std::move(value));
    }
};

/// @brief Limit makes a retrieve return at most value records.
struct Limit {
    /// @brief value is the most records to return.
    std::uint32_t value;
};

/// @brief Offset makes a retrieve skip the first value matching records.
struct Offset {
    /// @brief value is the number of matching records to skip.
    std::uint32_t value;
};

/// @brief Search makes a retrieve keep the records that match term, closest match
/// first.
struct Search {
    /// @brief term is the search term.
    std::string term;
};

/// @brief Include sets the flag Field of a retrieve request.
template<auto Field>
struct Include {
    /// @brief field is the request flag that the option sets.
    static constexpr auto field = Field;
};

/// @brief One makes a retrieve return one record. It goes after every other part.
struct One {};

/// @returns an option that returns at most n records.
[[nodiscard]] inline Limit limit(const std::uint32_t n) {
    return {n};
}

/// @returns an option that skips the first n matching records.
[[nodiscard]] inline Offset offset(const std::uint32_t n) {
    return {n};
}

/// @returns an option that keeps the records that match term, closest match first.
[[nodiscard]] inline Search search(std::string term) {
    return {std::move(term)};
}

/// @returns a part that makes a retrieve return one record. The retrieve returns an
/// error if no record or more than one record matches.
[[nodiscard]] inline One one() {
    return {};
}

namespace details {
template<typename P, typename... Parts>
constexpr std::size_t count = (std::size_t{std::is_same_v<P, Parts>} + ... + 0);

template<typename P, typename R>
constexpr bool is_include = false;

template<auto Field, typename R>
constexpr bool
    is_include<Include<Field>, R> = std::is_same_v<decltype(Field), bool R::*>;

template<typename P, typename W, typename R>
constexpr bool is_part = std::is_same_v<P, Condition<W>> || std::is_same_v<P, Limit> ||
                         std::is_same_v<P, Offset> || std::is_same_v<P, Search> ||
                         std::is_same_v<P, One> || is_include<P, R>;

template<typename W, typename... Parts>
constexpr bool unique =
    ((std::is_same_v<Parts, Condition<W>> || count<Parts, Parts...> == 1) && ...);

template<typename... Parts>
constexpr bool one_last = [] {
    if constexpr (sizeof...(Parts) == 0)
        return true;
    else {
        using Last = std::tuple_element_t<sizeof...(Parts) - 1, std::tuple<Parts...>>;
        return count<One, Parts...> == 0 ||
               (count<One, Parts...> == 1 && std::is_same_v<Last, One>);
    }
}();

template<typename W, typename R, typename P>
void apply(R &req, std::vector<W> &conditions, P part) {
    if constexpr (std::is_same_v<P, Condition<W>>)
        conditions.push_back(std::move(part.tree));
    else if constexpr (std::is_same_v<P, Limit>)
        req.limit = part.value;
    else if constexpr (std::is_same_v<P, Offset>)
        req.offset = part.value;
    else if constexpr (std::is_same_v<P, Search>)
        req.search_term = std::move(part.term);
    else if constexpr (is_include<P, R>)
        req.*P::field = true;
}
}

/// @brief parts is satisfied when Parts can make a retrieve request R of the entity
/// whose filter is W: conditions of W and the options of R, each option at most once,
/// and one last.
template<typename W, typename R, typename... Parts>
concept parts = (details::is_part<Parts, W, R> && ...) &&
                details::unique<W, Parts...> && details::one_last<Parts...>;

/// @brief includes is satisfied when Parts are distinct include options of the retrieve
/// request R.
template<typename R, typename... Parts>
concept includes = (details::is_include<Parts, R> && ...) &&
                   ((details::count<Parts, Parts...> == 1) && ...);

/// @brief returns_one is true when Parts make a retrieve return one record.
template<typename... Parts>
constexpr bool returns_one = details::count<One, Parts...> == 1;

/// @returns the retrieve request R that parts make. Every condition must match, and
/// each option sets its field.
template<typename W, typename R, typename... Parts>
    requires parts<W, R, Parts...>
[[nodiscard]] R request(Parts... parts) {
    R req;
    std::vector<W> conditions;
    (details::apply(req, conditions, std::move(parts)), ...);
    if (conditions.size() == 1)
        req.where = std::move(conditions.front());
    else if (conditions.size() > 1) {
        W tree;
        tree.and_ = std::move(conditions);
        req.where = std::move(tree);
    }
    return req;
}

namespace details {
inline void drop_nulls(x::json::json &j) {
    if (j.is_array())
        for (auto &e: j)
            drop_nulls(e);
    if (!j.is_object()) return;
    for (auto it = j.begin(); it != j.end();)
        if (it->is_null())
            it = j.erase(it);
        else
            drop_nulls(*it++);
}
}

/// @returns a description of the records that where selects, for error messages.
template<typename W>
[[nodiscard]] std::string describe(const std::optional<W> &where) {
    if (!where.has_value()) return "any filter";
    auto j = where->to_json();
    details::drop_nulls(j);
    return "filter " + j.dump();
}
}
