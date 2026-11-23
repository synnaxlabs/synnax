// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/json/template.h"
#include "x/cpp/testutil/testutil.h"

namespace x::json {
namespace {
/// @brief returns a field at pointer, a slot when literal is nothing.
Template::Field field(
    const std::string &pointer,
    std::optional<std::string> literal = std::nullopt,
    std::string label = ""
) {
    return {
        .label = std::move(label),
        .pointer = ASSERT_NIL_P(Pointer::parse(pointer)),
        .literal = std::move(literal),
    };
}

/// @brief fills the slot of field i with i.
errors::Error numbered(const std::size_t field, std::string &out) {
    out += std::to_string(field);
    return errors::NIL;
}

std::string render(const std::vector<Template::Field> &fields) {
    const auto t = ASSERT_NIL_P(Template::create(fields));
    std::string out;
    EXPECT_NIL(t.append(out, numbered));
    return out;
}
}

/// @brief it should render each member in the order of the fields that set it.
TEST(Template, RendersFieldsInOrder) {
    const std::vector<std::pair<std::vector<Template::Field>, std::string>> cases{
        {{field("")}, "0"},
        {{field("", R"("x")")}, R"("x")"},
        {{field("/z"), field("/a")}, R"({"z":0,"a":1})"},
        {{field("/v/x"), field("/v/y", R"("c")"), field("/t")},
         R"({"v":{"x":0,"y":"c"},"t":2})"},
        {{field("/a/0")}, R"({"a":{"0":0}})"},
        {{field(R"(/a"b/c~1d)")}, R"({"a\"b":{"c/d":0}})"},
        {{}, "{}"},
    };
    for (const auto &[fields, expected]: cases)
        EXPECT_EQ(render(fields), expected);
}

/// @brief it should append the same text on every render.
TEST(Template, RendersSameTextOnEveryAppend) {
    const auto t = ASSERT_NIL_P(Template::create({field("/a"), field("/b", "true")}));
    std::string out;
    ASSERT_NIL(t.append(out, numbered));
    ASSERT_NIL(t.append(out, numbered));
    ASSERT_EQ(out, R"({"a":0,"b":true}{"a":0,"b":true})");
}

/// @brief it should return the error of a slot, appending nothing.
TEST(Template, ReturnsErrorOfSlot) {
    const auto t = ASSERT_NIL_P(Template::create({field("/a", "1"), field("/b")}));
    std::string out = "x";
    const auto err = t.append(out, [](std::size_t, std::string &) {
        return errors::Error(errors::VALIDATION, "no value");
    });
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "no value");
    ASSERT_EQ(out, "x");
}

/// @brief it should reject pointers that are equal or where one holds the other,
/// naming both fields in the order of fields.
TEST(Template, RejectsOverlappingPointers) {
    const std::vector<std::pair<std::vector<Template::Field>, std::string>> cases{
        {{field("/a", std::nullopt, "x"), field("/a", std::nullopt, "y")},
         R"(x "/a" and y "/a" overlap)"},
        {{field("/a", std::nullopt, "x"), field("/a/b", std::nullopt, "y")},
         R"(x "/a" and y "/a/b" overlap)"},
        {{field("/a/b", std::nullopt, "x"), field("/a", std::nullopt, "y")},
         R"(x "/a/b" and y "/a" overlap)"},
        {{field("", std::nullopt, "x"), field("/a", std::nullopt, "y")},
         R"(x "" and y "/a" overlap)"},
        {{field("/a", std::nullopt, "x"), field("", std::nullopt, "y")},
         R"(x "/a" and y "" overlap)"},
        {{field("/a/b", std::nullopt, "x"),
          field("/c", std::nullopt, "y"),
          field("/a", std::nullopt, "z")},
         R"(x "/a/b" and z "/a" overlap)"},
    };
    for (const auto &[fields, message]: cases) {
        const auto err = Template::create(fields).second;
        ASSERT_OCCURRED_AS(err, errors::VALIDATION);
        EXPECT_EQ(err.data, message);
    }
}

/// @brief it should reject a pointer token that is not valid UTF-8.
TEST(Template, RejectsTokenThatIsNotUTF8) {
    const auto err = Template::create({field("/a\xff")}).second;
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, R"(pointer token "a\xff": invalid UTF-8)");
}
}
