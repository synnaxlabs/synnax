// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <optional>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/filter/builder.h"
#include "x/cpp/json/json.h"
#include "x/cpp/mem/indirect.h"

namespace {
struct Where {
    std::optional<std::vector<Where>> and_;
    std::optional<std::vector<Where>> or_;
    x::mem::indirect<Where> not_;
    std::optional<x::filter::Set<std::uint32_t>> key;
    std::optional<x::filter::String> name;
    std::optional<x::filter::Bool> active;
    std::optional<x::filter::Ordered<std::int64_t>> count;
    std::optional<x::filter::Contains<std::string>> tags;
    std::optional<bool> virtual_;
    std::optional<std::vector<std::string>> labels;
    std::optional<std::int64_t> window;

    [[nodiscard]] x::json::json to_json() const {
        x::json::json j{{"name", nullptr}};
        if (this->key.has_value())
            j["key"] = {{"eq", this->key.value().eq.value()}, {"in", nullptr}};
        if (this->and_.has_value())
            for (const auto &w: *this->and_)
                j["and"].push_back(w.to_json());
        return j;
    }
};

struct Other {
    std::optional<std::vector<Other>> and_;
    std::optional<std::vector<Other>> or_;
    x::mem::indirect<Other> not_;
    std::optional<x::filter::Set<std::uint32_t>> key;
};

struct RetrieveRequest {
    std::optional<Where> where;
    std::string search_term;
    std::optional<std::uint32_t> limit;
    std::uint32_t offset = 0;
    bool include_labels = false;
    bool include_parent = false;
};

using Condition = x::filter::Condition<Where>;
using IncludeLabels = x::filter::Include<&RetrieveRequest::include_labels>;

constexpr x::filter::SetLine<&Where::key> key;
constexpr x::filter::StringLine<&Where::name> name;
constexpr x::filter::BoolLine<&Where::active> active;
constexpr x::filter::OrderedLine<&Where::count> count;
constexpr x::filter::ContainsLine<&Where::tags> tags;
constexpr x::filter::FlagLine<&Where::virtual_> virtual_;
constexpr x::filter::ListLine<&Where::labels> labels;
constexpr x::filter::ValueLine<&Where::window> window;
constexpr x::filter::SetLine<&Other::key> other_key;

template<typename L, typename V>
concept takes_eq = requires(const L line, V value) { line.eq(value); };

template<typename... Parts>
constexpr bool makes_request = x::filter::parts<Where, RetrieveRequest, Parts...>;

/// @brief it should build a condition on a set line.
TEST(FilterBuilder, SetLine) {
    const auto eq = key.eq(1);
    EXPECT_EQ(eq.tree.key.value().eq.value(), 1);
    const auto in = key.in({1, 2});
    EXPECT_EQ(in.tree.key.value().in.value(), (std::vector<std::uint32_t>{1, 2}));
}

/// @brief it should build a condition on a bool line.
TEST(FilterBuilder, BoolLine) {
    EXPECT_TRUE(active.eq(true).tree.active.value().eq.value());
}

/// @brief it should build each operator of an ordered line.
TEST(FilterBuilder, OrderedLine) {
    EXPECT_EQ(count.gt(3).tree.count.value().gt.value(), 3);
    EXPECT_EQ(count.gte(3).tree.count.value().gte.value(), 3);
    EXPECT_EQ(count.lt(3).tree.count.value().lt.value(), 3);
    EXPECT_EQ(count.lte(3).tree.count.value().lte.value(), 3);
    EXPECT_EQ(count.eq(3).tree.count.value().eq.value(), 3);
    EXPECT_EQ(
        count.in({1, 2}).tree.count.value().in.value(),
        (std::vector<std::int64_t>{1, 2})
    );
}

/// @brief it should build each operator of a string line.
TEST(FilterBuilder, StringLine) {
    EXPECT_EQ(name.eq("a").tree.name.value().eq.value(), "a");
    EXPECT_EQ(
        name.in({"a"}).tree.name.value().in.value(),
        std::vector<std::string>{"a"}
    );
    EXPECT_EQ(name.contains("a").tree.name.value().contains.value(), "a");
    EXPECT_EQ(name.starts_with("a").tree.name.value().starts_with.value(), "a");
    EXPECT_EQ(name.regex("^a").tree.name.value().regex.value(), "^a");
    EXPECT_FALSE(name.eq("a").tree.name.value().case_insensitive);
}

/// @brief it should set case_insensitive on a string condition.
TEST(FilterBuilder, StringLineCaseInsensitive) {
    const auto c = name.contains("a", {.case_insensitive = true});
    EXPECT_TRUE(c.tree.name.value().case_insensitive);
}

/// @brief it should build a condition on a contains line.
TEST(FilterBuilder, ContainsLine) {
    EXPECT_EQ(
        tags.has("a").tree.tags.value().any.value(),
        std::vector<std::string>{"a"}
    );
    const auto any = tags.has_any({"a", "b"});
    EXPECT_EQ(any.tree.tags.value().any.value(), (std::vector<std::string>{"a", "b"}));
}

/// @brief it should put a typed line's value on the line.
TEST(FilterBuilder, TypedLines) {
    EXPECT_FALSE(virtual_.eq(false).tree.virtual_.value());
    EXPECT_EQ(labels.has("a").tree.labels.value(), std::vector<std::string>{"a"});
    const auto any = labels.has_any({"a", "b"});
    EXPECT_EQ(any.tree.labels.value(), (std::vector<std::string>{"a", "b"}));
    EXPECT_EQ(window(5).tree.window.value(), 5);
}

/// @brief it should reject a value of the wrong type.
TEST(FilterBuilder, RejectWrongValueType) {
    static_assert(takes_eq<decltype(key), std::uint32_t>);
    static_assert(!takes_eq<decltype(key), std::string>);
    static_assert(!takes_eq<decltype(name), int>);
}

/// @brief it should match every child with and_ and the && operator.
TEST(FilterBuilder, And) {
    const auto c = x::filter::and_(key.eq(1), name.eq("a"), active.eq(true));
    EXPECT_EQ(c.tree.and_.value().size(), 3);
    const auto op = key.eq(1) && name.eq("a");
    EXPECT_EQ(op.tree.and_.value().at(1).name.value().eq.value(), "a");
}

/// @brief it should match any child with or_ and the || operator.
TEST(FilterBuilder, Or) {
    const auto c = x::filter::or_(key.eq(1), key.eq(2));
    EXPECT_EQ(c.tree.or_.value().at(1).key.value().eq.value(), 2);
    const auto op = key.eq(1) || key.eq(2);
    EXPECT_EQ(op.tree.or_.value().size(), 2);
}

/// @brief it should negate a child with not_ and the ! operator.
TEST(FilterBuilder, Not) {
    const auto c = x::filter::not_(key.eq(1));
    EXPECT_EQ(c.tree.not_->key.value().eq.value(), 1);
    const auto op = !key.eq(1);
    EXPECT_EQ(op.tree.not_->key.value().eq.value(), 1);
}

/// @brief it should leave the request empty when there are no parts.
TEST(FilterBuilder, RequestEmpty) {
    const auto req = x::filter::request<Where, RetrieveRequest>();
    EXPECT_FALSE(req.where.has_value());
    EXPECT_FALSE(req.limit.has_value());
}

/// @brief it should use one condition as the request's filter.
TEST(FilterBuilder, RequestOneCondition) {
    const auto req = x::filter::request<Where, RetrieveRequest>(key.eq(1));
    EXPECT_EQ(req.where.value().key.value().eq.value(), 1);
    EXPECT_FALSE(req.where.value().and_.has_value());
}

/// @brief it should join several conditions with and.
TEST(FilterBuilder, RequestJoinsConditions) {
    const auto req = x::filter::request<Where, RetrieveRequest>(
        key.eq(1),
        name.eq("a")
    );
    const auto &children = req.where.value().and_.value();
    ASSERT_EQ(children.size(), 2);
    EXPECT_EQ(children.at(0).key.value().eq.value(), 1);
    EXPECT_EQ(children.at(1).name.value().eq.value(), "a");
}

/// @brief it should set a request field for each option.
TEST(FilterBuilder, RequestOptions) {
    const auto req = x::filter::request<Where, RetrieveRequest>(
        x::filter::limit(5),
        x::filter::offset(10),
        x::filter::search("a"),
        IncludeLabels{},
        x::filter::one()
    );
    EXPECT_EQ(req.limit.value(), 5);
    EXPECT_EQ(req.offset, 10);
    EXPECT_EQ(req.search_term, "a");
    EXPECT_TRUE(req.include_labels);
    EXPECT_FALSE(req.include_parent);
}

/// @brief it should reject a condition of another entity.
TEST(FilterBuilder, RejectOtherEntity) {
    static_assert(makes_request<Condition>);
    static_assert(!makes_request<x::filter::Condition<Other>>);
    static_assert(!makes_request<decltype(other_key.eq(1))>);
}

/// @brief it should reject an option set twice.
TEST(FilterBuilder, RejectDuplicateOption) {
    static_assert(makes_request<Condition, Condition>);
    static_assert(!makes_request<x::filter::Limit, x::filter::Limit>);
    static_assert(!makes_request<IncludeLabels, IncludeLabels>);
}

/// @brief it should accept one only as the last part.
TEST(FilterBuilder, OneLast) {
    static_assert(makes_request<Condition, x::filter::One>);
    static_assert(!makes_request<x::filter::One, Condition>);
    static_assert(!makes_request<x::filter::One, x::filter::One>);
    static_assert(x::filter::returns_one<Condition, x::filter::One>);
    static_assert(!x::filter::returns_one<Condition>);
}

/// @brief it should accept distinct include options of the request.
TEST(FilterBuilder, Includes) {
    using IncludeParent = x::filter::Include<&RetrieveRequest::include_parent>;
    static_assert(x::filter::includes<RetrieveRequest>);
    static_assert(x::filter::includes<RetrieveRequest, IncludeLabels, IncludeParent>);
    static_assert(!x::filter::includes<RetrieveRequest, IncludeLabels, IncludeLabels>);
    static_assert(!x::filter::includes<RetrieveRequest, x::filter::Limit>);
}

/// @brief it should describe the filter of a request without null fields, for error
/// messages.
TEST(FilterBuilder, Describe) {
    EXPECT_EQ(x::filter::describe(std::optional<Where>{}), "any filter");
    const auto req = x::filter::request<Where, RetrieveRequest>(key.eq(7));
    EXPECT_EQ(x::filter::describe(req.where), R"(filter {"key":{"eq":7}})");
    const auto both = x::filter::request<Where, RetrieveRequest>(key.eq(7), key.eq(8));
    EXPECT_EQ(
        x::filter::describe(both.where),
        R"(filter {"and":[{"key":{"eq":7}},{"key":{"eq":8}}]})"
    );
}
}
