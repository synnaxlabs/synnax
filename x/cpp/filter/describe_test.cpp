// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <optional>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/filter/describe.h"
#include "x/cpp/json/json.h"

namespace {
std::string describe(const std::string &tree) {
    return x::filter::describe(x::json::json::parse(tree));
}

struct Where {
    std::optional<std::uint32_t> key;

    [[nodiscard]] x::json::json to_json() const {
        x::json::json j = x::json::json::object();
        if (this->key.has_value()) j["key"] = {{"eq", *this->key}, {"in", nullptr}};
        return j;
    }
};

/// @brief it should describe each operator.
TEST(FilterDescribe, Operators) {
    EXPECT_EQ(describe(R"({"name":{"eq":"press"}})"), R"(name = "press")");
    EXPECT_EQ(describe(R"({"key":{"in":[12,14]}})"), "key in [12, 14]");
    EXPECT_EQ(describe(R"({"name":{"contains":"a"}})"), R"(name contains "a")");
    EXPECT_EQ(describe(R"({"name":{"starts_with":"a"}})"), R"(name starts with "a")");
    EXPECT_EQ(describe(R"({"name":{"regex":"pre.*"}})"), "name matches /pre.*/");
    EXPECT_EQ(describe(R"({"count":{"gt":3}})"), "count > 3");
    EXPECT_EQ(describe(R"({"count":{"lte":3}})"), "count <= 3");
    EXPECT_EQ(describe(R"({"count":{"gte":3,"lt":10}})"), "count >= 3 and count < 10");
    EXPECT_EQ(
        describe(R"({"tags":{"any":["a","b"]}})"),
        R"(tags has any of ["a", "b"])"
    );
    EXPECT_EQ(describe(R"({"active":{"eq":true}})"), "active = true");
}

/// @brief it should mark each operator of a case-insensitive condition.
TEST(FilterDescribe, CaseInsensitive) {
    EXPECT_EQ(
        describe(R"({"name":{"eq":"Press","case_insensitive":true}})"),
        R"(name = "Press" (ignoring case))"
    );
    EXPECT_EQ(
        describe(
            R"({"name":{"contains":"a","starts_with":"b","case_insensitive":true}})"
        ),
        R"(name contains "a" (ignoring case) and name starts with "b" (ignoring case))"
    );
    EXPECT_EQ(
        describe(R"({"name":{"eq":"a","case_insensitive":false}})"),
        R"(name = "a")"
    );
}

/// @brief it should describe a value that is not an operator object.
TEST(FilterDescribe, Value) {
    EXPECT_EQ(describe(R"({"virtual":true})"), "virtual = true");
    EXPECT_EQ(describe(R"({"labels":["a","b"]})"), R"(labels = ["a", "b"])");
    EXPECT_EQ(
        describe(R"({"span":{"start":1,"end":2}})"),
        R"(span = {"end":2,"start":1})"
    );
}

/// @brief it should escape a string.
TEST(FilterDescribe, Escape) {
    EXPECT_EQ(describe(R"({"name":{"eq":"a\"b"}})"), R"(name = "a\"b")");
}

/// @brief it should sort the fields of a node by name.
TEST(FilterDescribe, Sort) {
    EXPECT_EQ(
        describe(R"({"name":{"eq":"a"},"active":{"eq":true},"key":{"eq":1}})"),
        R"(active = true and key = 1 and name = "a")"
    );
}

/// @brief it should join the children of and.
TEST(FilterDescribe, And) {
    EXPECT_EQ(
        describe(R"({"and":[{"name":{"eq":"a"}},{"key":{"eq":1}}]})"),
        R"(name = "a" and key = 1)"
    );
}

/// @brief it should join the children of or.
TEST(FilterDescribe, Or) {
    EXPECT_EQ(
        describe(R"({"or":[{"name":{"eq":"a"}},{"not":{"active":{"eq":true}}}]})"),
        R"(name = "a" or not (active = true))"
    );
}

/// @brief it should wrap an or in parentheses beside other terms.
TEST(FilterDescribe, OrBesideTerms) {
    EXPECT_EQ(
        describe(R"({"key":{"eq":1},"or":[{"name":{"eq":"a"}},{"name":{"eq":"b"}}]})"),
        R"(key = 1 and (name = "a" or name = "b"))"
    );
}

/// @brief it should keep the grouping of a child of a single-child or.
TEST(FilterDescribe, SingleOrChildGrouping) {
    EXPECT_EQ(
        describe(
            R"({"key":{"eq":1},"or":[{"or":[)"
            R"({"name":{"eq":"a"}},{"name":{"eq":"b"}}]}]})"
        ),
        R"(key = 1 and (name = "a" or name = "b"))"
    );
}

/// @brief it should wrap a child of or that has more than one term.
TEST(FilterDescribe, OrChildTerms) {
    EXPECT_EQ(
        describe(R"({"or":[{"key":{"eq":1},"name":{"eq":"a"}},{"key":{"eq":2}}]})"),
        R"((key = 1 and name = "a") or key = 2)"
    );
}

/// @brief it should describe not.
TEST(FilterDescribe, Not) {
    EXPECT_EQ(
        describe(R"({"not":{"key":{"eq":1},"name":{"eq":"a"}}})"),
        R"(not (key = 1 and name = "a"))"
    );
}

/// @brief it should describe an empty tree as an empty string.
TEST(FilterDescribe, Empty) {
    EXPECT_EQ(describe("{}"), "");
    EXPECT_EQ(describe(R"({"and":[]})"), "");
    EXPECT_EQ(describe(R"({"name":{}})"), "");
}

/// @brief it should describe nested empty trees.
TEST(FilterDescribe, NestedEmpty) {
    EXPECT_EQ(describe(R"({"or":[{},{"key":{"eq":1}}]})"), "true or key = 1");
    EXPECT_EQ(describe(R"({"or":[{"and":[]},{"key":{"eq":1}}]})"), "true or key = 1");
    EXPECT_EQ(describe(R"({"or":[]})"), "false");
    EXPECT_EQ(describe(R"({"or":[{}]})"), "");
    EXPECT_EQ(describe(R"({"not":{}})"), "not (true)");
}

/// @brief it should describe a filter struct, and an absent one as an empty string.
TEST(FilterDescribe, Struct) {
    EXPECT_EQ(x::filter::describe(std::optional<Where>{}), "");
    EXPECT_EQ(x::filter::describe(std::optional<Where>{Where{.key = 7}}), "key = 7");
}
}
