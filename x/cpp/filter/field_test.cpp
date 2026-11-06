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
#include <type_traits>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/filter/field.h"

namespace x::filter {
namespace {
struct Where {
    Field<String> name;
    Field<Set<std::uint32_t>> key;
    Field<Bool> internal;
    List<std::string> tags;
};
}

/// @brief it should hold an operator as it is.
TEST(FieldTest, testOperator) {
    const Field<String> f = String{.starts_with = "Hot"};
    ASSERT_TRUE(f.has_value());
    EXPECT_EQ(f.value().starts_with, "Hot");
    EXPECT_FALSE(f.value().eq.has_value());
}

/// @brief it should read a plain value as eq.
TEST(FieldTest, testPlainValue) {
    const Field<String> f = "pressure";
    ASSERT_TRUE(f.has_value());
    EXPECT_EQ(f.value().eq, "pressure");
    EXPECT_FALSE(f.value().in.has_value());
}

/// @brief it should read a braced list as in.
TEST(FieldTest, testBracedList) {
    const Field<Set<std::uint32_t>> f = {1, 2};
    ASSERT_TRUE(f.has_value());
    EXPECT_EQ(f.value().in, (std::vector<std::uint32_t>{1, 2}));
    EXPECT_FALSE(f.value().eq.has_value());
}

/// @brief it should read a vector as in.
TEST(FieldTest, testVector) {
    const std::vector<std::string> names{"a", "b"};
    const Field<String> f = names;
    ASSERT_TRUE(f.has_value());
    EXPECT_EQ(f.value().in, names);
}

/// @brief it should read plain values and lists on an Ordered line.
TEST(FieldTest, testOrdered) {
    const Field<Ordered<std::int64_t>> value = std::int64_t{5};
    EXPECT_EQ(value.value().eq, 5);
    const Field<Ordered<std::int64_t>> list = {1, 2};
    EXPECT_EQ(list.value().in, (std::vector<std::int64_t>{1, 2}));
}

/// @brief it should read a plain bool as eq.
TEST(FieldTest, testBool) {
    const Field<Bool> f = false;
    ASSERT_TRUE(f.has_value());
    EXPECT_EQ(f.value().eq, false);
}

/// @brief it should reject a value that only converts to bool as a Bool.
TEST(FieldTest, testBoolRejectsConversions) {
    static_assert(!std::is_convertible_v<const char *, Field<Bool>>);
    static_assert(!std::is_convertible_v<int, Field<Bool>>);
    static_assert(!std::is_convertible_v<std::vector<bool>, Field<Bool>>);
}

/// @brief it should match any of the values on a Contains line.
TEST(FieldTest, testContains) {
    const Field<Contains<std::string>> value = "anomaly";
    EXPECT_EQ(value.value().any, (std::vector<std::string>{"anomaly"}));
    const Field<Contains<std::string>> list = {"a", "b"};
    EXPECT_EQ(list.value().any, (std::vector<std::string>{"a", "b"}));
}

/// @brief it should be empty by default and from nullopt.
TEST(FieldTest, testEmpty) {
    const Field<String> by_default;
    EXPECT_FALSE(by_default.has_value());
    const Field<String> from_nullopt = std::nullopt;
    EXPECT_FALSE(from_nullopt.has_value());
}

/// @brief it should take a parsed line that may be absent.
TEST(FieldTest, testParsedOptional) {
    const Field<String> absent = std::optional<String>{};
    EXPECT_FALSE(absent.has_value());
    const Field<String> present = std::optional<String>{String{.regex = "a.*"}};
    EXPECT_EQ(present.value().regex, "a.*");
}

/// @brief it should replace the operator on assignment.
TEST(FieldTest, testAssign) {
    Field<String> f = "a";
    f = String{.contains = "b"};
    ASSERT_TRUE(f.has_value());
    EXPECT_FALSE(f.value().eq.has_value());
    EXPECT_EQ(f.value().contains, "b");
    f = std::nullopt;
    EXPECT_FALSE(f.has_value());
}

/// @brief it should read one value as a list of one.
TEST(ListTest, testOneValue) {
    const List<std::string> l = "a";
    EXPECT_EQ(l, (std::vector<std::string>{"a"}));
}

/// @brief it should take a braced list, a vector, or a parsed optional.
TEST(ListTest, testLists) {
    const List<std::string> braced = {"a", "b"};
    EXPECT_EQ(braced, (std::vector<std::string>{"a", "b"}));
    const List<std::string> vector = std::vector<std::string>{"c"};
    EXPECT_EQ(vector, (std::vector<std::string>{"c"}));
    const List<std::string> parsed = std::optional<std::vector<std::string>>{};
    EXPECT_FALSE(parsed.has_value());
}

/// @brief it should let a designated initializer write plain values and lists.
TEST(FieldTest, testDesignatedInitializer) {
    const Where w{.name = "x", .key = {1, 2}, .internal = true, .tags = "a"};
    EXPECT_EQ(w.name.value().eq, "x");
    EXPECT_EQ(w.key.value().in, (std::vector<std::uint32_t>{1, 2}));
    EXPECT_EQ(w.internal.value().eq, true);
    EXPECT_EQ(w.tags, (std::vector<std::string>{"a"}));
}
}
