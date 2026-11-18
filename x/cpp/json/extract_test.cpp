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
#include <string_view>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/extract.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/testutil/testutil.h"

namespace x::json {
namespace {
/// @brief parses each text form into a Pointer.
std::vector<Pointer> pointers(const std::vector<std::string> &texts) {
    std::vector<Pointer> parsed;
    parsed.reserve(texts.size());
    for (const auto &text: texts) {
        parsed.push_back(ASSERT_NIL_P(Pointer::parse(text)));
    }
    return parsed;
}

/// @brief resolves the pointers in doc and returns the text of each value, with "-" for
/// a pointer that leads to no value.
std::vector<std::string>
extract(const std::string_view doc, const std::vector<std::string> &texts) {
    Extractor e(pointers(texts));
    std::vector<std::optional<std::string_view>> values;
    EXPECT_NIL(e.extract(doc, values));
    std::vector<std::string> out;
    out.reserve(values.size());
    for (const auto &v: values)
        out.emplace_back(v.has_value() ? std::string(*v) : "-");
    return out;
}

/// @brief returns the error of extracting /a from doc.
errors::Error extract_error(const std::string_view doc) {
    Extractor e(pointers({"/a"}));
    std::vector<std::optional<std::string_view>> values;
    return e.extract(doc, values);
}

using Texts = std::vector<std::string>;
}

/// @brief it should resolve each pointer to the text of its value.
TEST(Extractor, ResolvesValueText) {
    ASSERT_EQ(
        extract(
            R"({"n":9007199254740993,"s":"o\nn","t":true,"z":null,"o":{"a": [1]}})",
            {"/n", "/s", "/t", "/z", "/o"}
        ),
        (Texts{"9007199254740993", R"("o\nn")", "true", "null", R"({"a": [1]})"})
    );
}

/// @brief it should resolve the root pointer to the whole document.
TEST(Extractor, ResolvesRoot) {
    ASSERT_EQ(extract(R"( [1,"a"] )", {""}), (Texts{R"([1,"a"])"}));
}

/// @brief it should resolve a value and a value nested inside it.
TEST(Extractor, ResolvesNestedValues) {
    ASSERT_EQ(
        extract(R"({"a":{"b":[1,{"c":2}]},"d":3})", {"/a", "/a/b/1/c", "/d"}),
        (Texts{R"({"b":[1,{"c":2}]})", "2", "3"})
    );
}

/// @brief it should resolve array elements by index.
TEST(Extractor, ResolvesArrayElements) {
    ASSERT_EQ(
        extract(R"({"a":[10,20,{"b":30}]})", {"/a/1", "/a/2/b"}),
        (Texts{"20", "30"})
    );
}

/// @brief it should resolve each copy of a repeated pointer.
TEST(Extractor, ResolvesRepeatedPointer) {
    ASSERT_EQ(extract(R"({"a":1})", {"/a", "/a"}), (Texts{"1", "1"}));
}

/// @brief it should skip a member whose name holds an escape.
TEST(Extractor, SkipsEscapedMemberName) {
    ASSERT_EQ(extract(R"({"x":{"\u0062":[]},"a":1})", {"/a"}), (Texts{"1"}));
}

/// @brief it should resolve escaped tokens.
TEST(Extractor, ResolvesEscapedTokens) {
    ASSERT_EQ(extract(R"({"a/b":1,"c~d":2})", {"/a~1b", "/c~0d"}), (Texts{"1", "2"}));
}

/// @brief it should match an object member whose name holds an escape.
TEST(Extractor, MatchesEscapedMemberName) {
    ASSERT_EQ(extract(R"({"\u0061":1})", {"/a"}), (Texts{"1"}));
}

/// @brief it should resolve a member name made of digits in an object.
TEST(Extractor, ResolvesDigitMemberName) {
    ASSERT_EQ(extract(R"({"a":{"01":1}})", {"/a/01"}), (Texts{"1"}));
}

/// @brief it should find no value for a missing member.
TEST(Extractor, MissingMember) {
    ASSERT_EQ(extract(R"({"a":1})", {"/b"}), (Texts{"-"}));
}

/// @brief it should find no value for an index past the end of an array.
TEST(Extractor, IndexPastEnd) {
    ASSERT_EQ(extract("[1]", {"/1"}), (Texts{"-"}));
}

/// @brief it should find no value for an index with a leading zero.
TEST(Extractor, IndexWithLeadingZero) {
    ASSERT_EQ(extract("[1,2]", {"/01"}), (Texts{"-"}));
}

/// @brief it should find no value for a member of a scalar.
TEST(Extractor, MemberOfScalar) {
    ASSERT_EQ(extract(R"({"a":1})", {"/a/b"}), (Texts{"-"}));
}

/// @brief it should find no value for an index into an object.
TEST(Extractor, IndexIntoObject) {
    ASSERT_EQ(extract(R"({"a":{"0":1}})", {"/a/1"}), (Texts{"-"}));
}

/// @brief it should allow repeated names that no pointer addresses.
TEST(Extractor, AllowsRepeatedUnaddressedNames) {
    ASSERT_EQ(extract(R"({"x":1,"x":2,"a":3})", {"/a"}), (Texts{"3"}));
}

/// @brief it should clear the values of the last document.
TEST(Extractor, ClearsLastValues) {
    Extractor e(pointers({"/a"}));
    std::vector<std::optional<std::string_view>> values;
    ASSERT_NIL(e.extract(R"({"a":1})", values));
    ASSERT_NIL(e.extract(R"({"b":1})", values));
    ASSERT_EQ(values, (std::vector<std::optional<std::string_view>>{std::nullopt}));
}

/// @brief it should take a document nested as deep as the limit.
TEST(Extractor, AcceptsMaxDepth) {
    const auto doc = std::string(10000, '[') + std::string(10000, ']');
    ASSERT_NIL(extract_error(doc));
}

/// @brief it should reject a document nested deeper than the limit.
TEST(Extractor, RejectsPastMaxDepth) {
    const auto doc = std::string(10001, '[') + std::string(10001, ']');
    const auto err = extract_error(doc);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "exceeded max depth after offset 10000");
}

/// @brief it should reject a document that ends early.
TEST(Extractor, RejectsTruncatedDocument) {
    const auto err = extract_error(R"({"a":)");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "unexpected EOF after offset 5");
}

/// @brief it should reject an empty document.
TEST(Extractor, RejectsEmptyDocument) {
    const auto err = extract_error(" ");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "unexpected EOF after offset 1");
}

/// @brief it should reject data after the document.
TEST(Extractor, RejectsTrailingData) {
    const auto err = extract_error(R"({"a":1} 2)");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "unexpected data after the top-level value");
}

/// @brief it should reject a repeated addressed name.
TEST(Extractor, RejectsRepeatedAddressedName) {
    const auto err = extract_error(R"({"a":1,"a":2})");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "duplicate value at /a");
}

/// @brief it should reject a repeated name above an addressed value.
TEST(Extractor, RejectsRepeatedParentName) {
    Extractor e(pointers({"/a/b"}));
    std::vector<std::optional<std::string_view>> values;
    const auto err = e.extract(R"({"a":{"b":1},"a":{"b":2}})", values);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "duplicate value at /a/b");
}

/// @brief it should reject invalid UTF-8.
TEST(Extractor, RejectsInvalidUTF8) {
    const auto err = extract_error("{\"a\":\"\xff\"}");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid UTF-8 after offset 6");
}

/// @brief it should reject a misspelled literal.
TEST(Extractor, RejectsBadLiteral) {
    const auto err = extract_error("not json");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid character 'o' after offset 1");
}

/// @brief it should reject a malformed value that no pointer addresses.
TEST(Extractor, RejectsMalformedSkippedValue) {
    const auto err = extract_error(R"({"x":[1,],"a":1})");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid character ']' after offset 8");
}

/// @brief it should reject a number with a leading zero.
TEST(Extractor, RejectsLeadingZero) {
    const auto err = extract_error(R"({"a":01})");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid character '1' after offset 6");
}

/// @brief it should reject a member with no colon.
TEST(Extractor, RejectsMissingColon) {
    const auto err = extract_error(R"({"a" 1})");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid character '1' after offset 5");
}

/// @brief it should reject a control character in a string.
TEST(Extractor, RejectsControlCharacterInString) {
    const auto err = extract_error("[\"a\nb\"]");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid control character after offset 3");
}
}
