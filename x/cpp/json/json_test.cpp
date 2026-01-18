// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

/// local.
#include "x/cpp/json/json.h"
#include "x/cpp/test/test.h"

namespace x::json {
/// @brief it should parse valid JSON fields successfully.
TEST(testConfig, testParserHappyPath) {
    struct MyConfig {
        std::string name;
        float dog;
    };
    MyConfig v;

    const json j = {{"name", "test"}, {"dog", 1.0}};
    Parser parser(j);
    v.name = parser.field<std::string>("name");
    v.dog = parser.field<float>("dog", 12.0f);
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(v.name, "test");
    ASSERT_EQ(v.dog, 1.0);
}

/// @brief it should report error when required field does not exist.
TEST(testConfig, testParserFieldDoesnNotExist) {
    struct MyConfig {
        std::string name;
        float dog{};
    };
    MyConfig v;
    const json j = {};
    Parser parser(j);
    v.name = parser.field<std::string>("name");
    v.dog = parser.field<float>("dog", 12.0f);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "name");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should report error when field has invalid type.
TEST(testConfig, testParserFieldHasInvalidType) {
    struct MyConfig {
        std::string name;
        float dog{};
    };
    MyConfig v;
    const json j = {{"name", "test"}, {"dog", "cat"}};
    Parser parser(j);
    v.name = parser.field<std::string>("name");
    v.dog = parser.field<float>("dog", 12.0f);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "dog");
    EXPECT_EQ(err["message"], "expected a number, got 'cat'");
}

/// @brief it should parse nested child objects successfully.
TEST(testConfig, testParserFieldChildHappyPath) {
    struct MyChildConfig {
        std::string name;
        float dog;
    };

    struct MyConfig {
        MyChildConfig child;
    };

    json j = {{"child", {{"name", "test"}, {"dog", 1.0}}}};
    MyConfig v;
    Parser parser(j);
    auto child_parser = parser.child("child");
    v.child.name = child_parser.field<std::string>("name");
    v.child.dog = child_parser.field<float>("dog", 12.0f);
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(v.child.name, "test");
    ASSERT_EQ(v.child.dog, 1.0);
}

/// @brief it should report error when child object does not exist.
TEST(testConfig, testParserFieldChildDoesNotExist) {
    struct MyChildConfig {
        std::string name;
        float dog;
    };

    struct MyConfig {
        MyChildConfig child;
    };

    json j = {};
    MyConfig v;
    Parser parser(j);
    auto child_parser = parser.child("child");
    v.child.name = child_parser.field<std::string>("name");
    v.child.dog = child_parser.field<float>("dog", 12.0f);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "child");
    EXPECT_EQ(err["message"], "this field is required");
}

/// @brief it should report error when child field has invalid type.
TEST(testConfig, testParserChildFieldInvalidType) {
    struct MyChildConfig {
        std::string name;
        float dog;
    };

    struct MyConfig {
        MyChildConfig child;
    };

    json j = {{"child", {{"name", "test"}, {"dog", "cat"}}}};
    MyConfig v;
    Parser parser(j);
    auto child_parser = parser.child("child");
    v.child.name = child_parser.field<std::string>("name");
    v.child.dog = child_parser.field<float>("dog", 12.0f);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "child.dog");
    EXPECT_EQ(err["message"], "expected a number, got 'cat'");
}

/// @brief it should iterate over array fields successfully.
TEST(testConfig, testIterHappyPath) {
    struct MyChildConfig {
        std::string name;
        float dog;
    };

    struct MyConfig {
        std::vector<MyChildConfig> children;
    };

    const json j = {
        {"children",
         {{{"name", "test1"}, {"dog", 1.0}}, {{"name", "test2"}, {"dog", 2.0}}}}
    };

    MyConfig v;
    const Parser parser(j);
    parser.iter("children", [&](Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.field<std::string>("name");
        child.dog = child_parser.field<float>("dog", 12.0f);
        v.children.push_back(child);
    });
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(v.children.size(), 2);
    ASSERT_EQ(v.children[0].name, "test1");
    ASSERT_EQ(v.children[0].dog, 1.0);
}

/// @brief it should report error when iterable field does not exist.
TEST(testConfig, testIterFieldDoesNotExist) {
    struct MyChildConfig {
        std::string name;
        float dog;
    };

    struct MyConfig {
        std::vector<MyChildConfig> children;
    };

    const json j = {};
    MyConfig v;
    const Parser parser(j);
    parser.iter("children", [&](Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.field<std::string>("name");
        child.dog = child_parser.field<float>("dog", 12.0f);
        v.children.push_back(child);
    });
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "children");
    EXPECT_EQ(err["message"], "this field is required");
}

/// @brief it should report error when iterable field is not an array.
TEST(testConfig, testIterFieldIsNotArray) {
    struct MyChildConfig {
        std::string name;
        float dog;
    };

    struct MyConfig {
        std::vector<MyChildConfig> children;
    };

    const json j = {{"children", {{"name", "test1"}, {"dog", 1.0}}}};
    MyConfig v;
    const Parser parser(j);
    parser.iter("children", [&](Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.field<std::string>("name");
        child.dog = child_parser.field<float>("dog", 12.0f);
        v.children.push_back(child);
    });
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "children");
    EXPECT_EQ(err["message"], "expected an array");
}

/// @brief it should report error when array element has invalid field type.
TEST(testConfig, testIterFieldChildFieldInvalidType) {
    struct MyChildConfig {
        std::string name;
        float dog;
    };

    struct MyConfig {
        std::vector<MyChildConfig> children;
    };

    const json j = {
        {"children",
         {{{"name", "test1"}, {"dog", "1.0"}}, {{"name", "test2"}, {"dog", "red"}}}}
    };

    MyConfig v;
    const Parser parser(j);
    parser.iter("children", [&](Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.field<std::string>("name");
        child.dog = child_parser.field<float>("dog", 12.0f);
        v.children.push_back(child);
    });
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "children.1.dog");
    EXPECT_EQ(err["message"], "expected a number, got 'red'");
}

/// @brief it should interpret string values as numbers when possible.
TEST(testConfig, testInterpretStringAsNumber) {
    struct MyConfig {
        float dog;
    };
    const json j = {{"dog", "1.232"}};
    MyConfig v;
    Parser parser(j);
    v.dog = parser.field<float>("dog");
    EXPECT_TRUE(parser.ok());
    // assert that the value is close to the expected value.
    ASSERT_NEAR(v.dog, 1.232, 0.0001);
}

/// @brief it should parse array fields successfully.
TEST(testConfig, testArray) {
    const json j = {{"array", {1, 2, 3, 4, 5}}};
    Parser parser(j);
    const auto values = parser.field<std::vector<int>>("array");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 5);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
    ASSERT_EQ(values[3], 4);
    ASSERT_EQ(values[4], 5);
}

/// @brief it should report error when required array field does not exist.
TEST(testConfig, testArrayDoesNotExist) {
    const json j = {};
    Parser parser(j);
    auto values = parser.field<std::vector<int>>("array");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "array");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should report error when array field is not an array.
TEST(testConfig, testArrayIsNotArray) {
    const json j = {{"array", 1}};
    Parser parser(j);
    auto values = parser.field<std::vector<int>>("array");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "array");
    EXPECT_EQ(err["message"], "expected an array");
}

/// @brief it should parse optional array with provided value over default.
TEST(testConfig, testOptionalArray) {
    const json j = {{"array", {1, 2, 3, 4, 5}}};
    Parser parser(j);
    const auto values = parser.field<std::vector<int>>("array", {6, 7, 8});
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 5);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
    ASSERT_EQ(values[3], 4);
    ASSERT_EQ(values[4], 5);
}

/// @brief it should report no error for valid parser with no field access.
TEST(testConfig, testNoError) {
    const json j = {};
    const Parser parser(j);
    const auto err = parser.error();
    ASSERT_NIL(err);
}

/// @brief it should parse config from a valid JSON file.
TEST(testConfig, testParseFromFileSuccess) {
    struct MyConfig {
        std::string name;
        float value;
    };

    // Create a temporary test file
    std::string test_file = "test_config.json";
    std::ofstream file(test_file);
    file << R"({
        "name": "test",
        "value": 42.5
    })";
    file.close();

    MyConfig v;
    auto parser = Parser::from_file_path(test_file);
    v.name = parser.field<std::string>("name");
    v.value = parser.field<float>("value");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(v.name, "test");
    ASSERT_EQ(v.value, 42.5);

    // Clean up
    std::remove(test_file.c_str());
}

/// @brief it should report error when parsing from nonexistent file.
TEST(testConfig, testParseFromFileFailure) {
    auto parser = Parser::from_file_path("nonexistent_file.json");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "");
    EXPECT_EQ(err["message"], "failed to open file: nonexistent_file.json");
}

/// @brief it should report error when parsing file with invalid JSON.
TEST(testConfig, testParseFromFileInvalidJSON) {
    // Create a temporary test file with invalid JSON
    std::string test_file = "invalid_config.json";
    std::ofstream file(test_file);
    file << R"({
        "name": "test",
        invalid json here
    })";
    file.close();

    auto parser = Parser::from_file_path(test_file);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "");
    EXPECT_TRUE(
        err["message"].get<std::string>().find("parse error") != std::string::npos
    );

    // Clean up
    std::remove(test_file.c_str());
}

/// @brief it should add custom xerror as field error.
TEST(testConfig, testFieldErrWithXError) {
    const json j = {};
    Parser parser(j);

    errors::Error custom_error(errors::VALIDATION, "Custom validation error");
    parser.field_err("test_field", custom_error);

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "test_field");
    EXPECT_TRUE(
        err["message"].get<std::string>().find("Custom validation error") !=
        std::string::npos
    );
}

struct BasicConstructibleConfig {
    std::string name;
    int value;

    explicit BasicConstructibleConfig(Parser p):
        name(p.field<std::string>("name")), value(p.field<int>("value")) {}

    BasicConstructibleConfig(): value(0) {}
};

struct NestedInnerConfig {
    std::string type;
    float threshold;

    explicit NestedInnerConfig(Parser p):
        type(p.field<std::string>("type")), threshold(p.field<float>("threshold")) {}

    NestedInnerConfig(): threshold(0.0f) {}
};

struct NestedOuterConfig {
    std::string name;
    NestedInnerConfig inner;

    explicit NestedOuterConfig(Parser p):
        name(p.field<std::string>("name")),
        inner(p.field<NestedInnerConfig>("inner")) {}

    NestedOuterConfig() {}
};

struct MissingFieldConfig {
    std::string name;
    int value;

    explicit MissingFieldConfig(Parser p):
        name(p.field<std::string>("name")), value(p.field<int>("value")) {}

    MissingFieldConfig(): value(0) {}
};

struct InvalidTypeConfig {
    std::string name;
    int value;

    explicit InvalidTypeConfig(Parser p):
        name(p.field<std::string>("name")), value(p.field<int>("value")) {}

    InvalidTypeConfig(): value(0) {}
};

struct NestedErrorInnerConfig {
    int required_value;

    explicit NestedErrorInnerConfig(Parser p):
        required_value(p.field<int>("required_value")) {}

    NestedErrorInnerConfig(): required_value(0) {}
};

struct NestedErrorOuterConfig {
    std::string name;
    NestedErrorInnerConfig inner;

    explicit NestedErrorOuterConfig(Parser p):
        name(p.field<std::string>("name")),
        inner(p.field<NestedErrorInnerConfig>("inner")) {}

    NestedErrorOuterConfig() {}
};

struct ArrayItem {
    std::string name;
    int id;

    explicit ArrayItem(Parser p):
        name(p.field<std::string>("name")), id(p.field<int>("id")) {}

    ArrayItem(): id(0) {}
};

struct MixedInnerConfig {
    float value;

    explicit MixedInnerConfig(Parser p): value(p.field<float>("value")) {}

    MixedInnerConfig(): value(0.0f) {}
};

struct MixedOuterConfig {
    std::string name;
    int count;
    MixedInnerConfig nested;
    bool enabled;

    explicit MixedOuterConfig(Parser p):
        name(p.field<std::string>("name")),
        count(p.field<int>("count")),
        nested(p.field<MixedInnerConfig>("nested")),
        enabled(p.field<bool>("enabled", true)) {}

    MixedOuterConfig(): count(0), enabled(false) {}
};

struct ParentMissingConfig {
    std::string name;

    explicit ParentMissingConfig(Parser p): name(p.field<std::string>("name")) {}

    ParentMissingConfig() {}
};

struct ParentInvalidTypeConfig {
    std::string name;

    explicit ParentInvalidTypeConfig(Parser p): name(p.field<std::string>("name")) {}

    ParentInvalidTypeConfig() {}
};

struct OptionalConfig {
    std::string name;
    int value;

    explicit OptionalConfig(Parser p):
        name(p.field<std::string>("name")), value(p.field<int>("value")) {}

    OptionalConfig(): value(99) {}
};

// Verify that the trait can see our test structs
static_assert(
    is_parser_constructible_v<BasicConstructibleConfig>,
    "BasicConstructibleConfig should be detected as parser-constructible"
);
static_assert(
    is_parser_constructible_v<NestedInnerConfig>,
    "NestedInnerConfig should be detected as parser-constructible"
);

/// @brief it should parse basic parser-constructible types.
TEST(testConfig, testBasicConstructibleType) {
    const json j = {{"config", {{"name", "test"}, {"value", 42}}}};
    Parser parser(j);
    auto config = parser.field<BasicConstructibleConfig>("config");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.name, "test");
    ASSERT_EQ(config.value, 42);
}

/// @brief it should parse nested parser-constructible types.
TEST(testConfig, testNestedConstructibleTypes) {
    const json j = {
        {"config",
         {{"name", "outer"}, {"inner", {{"type", "sensor"}, {"threshold", 3.14f}}}}}
    };

    Parser parser(j);
    auto config = parser.field<NestedOuterConfig>("config");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.name, "outer");
    ASSERT_EQ(config.inner.type, "sensor");
    ASSERT_NEAR(config.inner.threshold, 3.14f, 0.0001f);
}

/// @brief it should report error when constructible type has missing required field.
TEST(testConfig, testConstructibleTypeWithMissingRequiredField) {
    const json j = {{"config", {{"name", "test"}}}};
    Parser parser(j);
    auto config = parser.field<MissingFieldConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config.value");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should report error when constructible type has invalid field type.
TEST(testConfig, testConstructibleTypeWithInvalidType) {
    const json j = {{"config", {{"name", "test"}, {"value", "not_a_number"}}}};
    Parser parser(j);
    auto config = parser.field<InvalidTypeConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config.value");
    EXPECT_TRUE(
        err["message"].get<std::string>().find("expected a number") != std::string::npos
    );
}

/// @brief it should report nested error path for constructible type errors.
TEST(testConfig, testConstructibleTypeWithNestedError) {
    // Create an empty JSON object explicitly
    json empty_obj = json::object();
    const json j = {{"config", {{"name", "outer"}, {"inner", empty_obj}}}};

    Parser parser(j);
    auto config = parser.field<NestedErrorOuterConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config.inner.required_value");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should parse arrays of parser-constructible types.
TEST(testConfig, testArrayOfConstructibleTypes) {
    const json j = {
        {"items",
         {{{"name", "item1"}, {"id", 1}},
          {{"name", "item2"}, {"id", 2}},
          {{"name", "item3"}, {"id", 3}}}}
    };

    Parser parser(j);
    auto items = parser.field<std::vector<ArrayItem>>("items");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(items.size(), 3);
    ASSERT_EQ(items[0].name, "item1");
    ASSERT_EQ(items[0].id, 1);
    ASSERT_EQ(items[1].name, "item2");
    ASSERT_EQ(items[1].id, 2);
    ASSERT_EQ(items[2].name, "item3");
    ASSERT_EQ(items[2].id, 3);
}

/// @brief it should report error with correct path for array of constructible types.
TEST(testConfig, testArrayOfConstructibleTypesWithError) {
    const json j = {
        {"items",
         {{{"name", "item1"}, {"id", 1}},
          {{"name", "item2"}}, // Missing id
          {{"name", "item3"}, {"id", 3}}}}
    };

    Parser parser(j);
    auto items = parser.field<std::vector<ArrayItem>>("items");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "items.1.id");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should parse mixed constructible and primitive types together.
TEST(testConfig, testMixedConstructibleAndPrimitiveTypes) {
    const json j = {
        {"config",
         {{"name", "mixed"},
          {"count", 5},
          {"nested", {{"value", 2.5f}}},
          {"enabled", false}}}
    };

    Parser parser(j);
    auto config = parser.field<MixedOuterConfig>("config");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.name, "mixed");
    ASSERT_EQ(config.count, 5);
    ASSERT_NEAR(config.nested.value, 2.5f, 0.0001f);
    ASSERT_FALSE(config.enabled);
}

/// @brief it should report error when parent field for constructible type is missing.
TEST(testConfig, testConstructibleTypeParentMissing) {
    const json j = {};
    Parser parser(j);
    auto config = parser.field<ParentMissingConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should report error when parent field has invalid type for constructible.
TEST(testConfig, testConstructibleTypeParentInvalidType) {
    const json j = {{"config", "not_an_object"}};
    Parser parser(j);
    auto config = parser.field<ParentInvalidTypeConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config");
    EXPECT_EQ(err["message"], "expected an object or array");
}

/// @brief it should use default value for optional missing constructible type.
TEST(testConfig, testOptionalConstructibleType) {
    const json j = {};
    Parser parser(j);
    OptionalConfig default_config;
    default_config.name = "default";
    default_config.value = 100;

    auto config = parser.field<OptionalConfig>("config", default_config);

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.name, "default");
    ASSERT_EQ(config.value, 100);
}

/// @brief it should parse root object when empty path is provided.
TEST(testConfig, testEmptyPathBehaviorParsesRoot) {
    // Test 1: Empty path now parses the root object as a parser-constructible type
    const json j1 = {{"name", "test"}, {"id", 42}};
    Parser parser1(j1);
    auto item1 = parser1.field<ArrayItem>(""); // Empty string means parse root
    EXPECT_TRUE(parser1.ok());
    EXPECT_EQ(item1.name, "test");
    EXPECT_EQ(item1.id, 42);

    // Test 2: Empty path parses root scalar
    const json j2 = "hello";
    Parser parser2(j2);
    auto val2 = parser2.field<std::string>("");
    EXPECT_TRUE(parser2.ok());
    EXPECT_EQ(val2, "hello");

    // Test 3: Empty path parses root array
    const json j3 = json::array({1, 2, 3});
    Parser parser3(j3);
    auto val3 = parser3.field<std::vector<int>>("");
    EXPECT_TRUE(parser3.ok());
    ASSERT_EQ(val3.size(), 3);
    EXPECT_EQ(val3[0], 1);

    // Test 4: No-args and empty string are equivalent
    const json j4 = 123;
    Parser parser4a(j4);
    Parser parser4b(j4);
    auto val4a = parser4a.field<int>();
    auto val4b = parser4b.field<int>("");
    EXPECT_TRUE(parser4a.ok());
    EXPECT_TRUE(parser4b.ok());
    EXPECT_EQ(val4a, val4b);
    EXPECT_EQ(val4a, 123);
}

/// @brief it should parse root array when no path argument is provided.
TEST(testConfig, testFieldNoArgsWithRootArray) {
    const json j = json::array({1, 2, 3, 4, 5});
    Parser parser(j);
    const auto values = parser.field<std::vector<int>>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 5);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
    ASSERT_EQ(values[3], 4);
    ASSERT_EQ(values[4], 5);
}

/// @brief it should parse root string array when no path argument is provided.
TEST(testConfig, testFieldNoArgsWithRootArrayStrings) {
    const json j = json::array({"a", "b", "c"});
    Parser parser(j);
    const auto values = parser.field<std::vector<std::string>>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 3);
    ASSERT_EQ(values[0], "a");
    ASSERT_EQ(values[1], "b");
    ASSERT_EQ(values[2], "c");
}

/// @brief it should parse root array of constructible types with no path argument.
TEST(testConfig, testFieldNoArgsWithConstructibleTypes) {
    const json j = json::array(
        {{{"name", "item1"}, {"id", 1}},
         {{"name", "item2"}, {"id", 2}},
         {{"name", "item3"}, {"id", 3}}}
    );
    Parser parser(j);
    const auto items = parser.field<std::vector<ArrayItem>>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(items.size(), 3);
    ASSERT_EQ(items[0].name, "item1");
    ASSERT_EQ(items[0].id, 1);
    ASSERT_EQ(items[1].name, "item2");
    ASSERT_EQ(items[1].id, 2);
    ASSERT_EQ(items[2].name, "item3");
    ASSERT_EQ(items[2].id, 3);
}

/// @brief it should report error when root is not array for array field type.
TEST(testConfig, testFieldNoArgsRootNotArray) {
    const json j = {{"name", "test"}, {"value", 42}};
    Parser parser(j);
    const auto values = parser.field<std::vector<int>>();
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "");
    EXPECT_EQ(err["message"], "expected an array");
}

/// @brief it should report error with correct path for root array parsing errors.
TEST(testConfig, testFieldNoArgsWithError) {
    const json j = json::array(
        {{{"name", "item1"}, {"id", 1}},
         {{"name", "item2"}}, // Missing id
         {{"name", "item3"}, {"id", 3}}}
    );
    Parser parser(j);
    const auto items = parser.field<std::vector<ArrayItem>>();
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "1.id");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should parse empty root array successfully.
TEST(testConfig, testFieldNoArgsEmpty) {
    const json j = json::array({});
    Parser parser(j);
    const auto values = parser.field<std::vector<int>>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 0);
}

/// @brief it should parse root scalar value with no path argument.
TEST(testConfig, testFieldNoArgsWithRootScalar) {
    const json j = 42;
    Parser parser(j);
    const auto value = parser.field<int>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(value, 42);
}

/// @brief it should parse root object as constructible type with no path argument.
TEST(testConfig, testFieldNoArgsWithRootObject) {
    const json j = {{"name", "test"}, {"id", 123}};
    Parser parser(j);
    const auto item = parser.field<ArrayItem>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(item.name, "test");
    ASSERT_EQ(item.id, 123);
}

/// @brief it should treat empty string path equivalent to no path argument.
TEST(testConfig, testFieldEmptyStringEquivalentToNoArgs) {
    const json j = json::array({1, 2, 3});
    Parser parser(j);
    const auto values = parser.field<std::vector<int>>("");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 3);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
}

/// @brief it should parse map fields with string keys successfully.
TEST(testConfig, testMapHappyPath) {
    const json j = {{"servers", {{"host1", 8080}, {"host2", 8081}, {"host3", 8082}}}};
    Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>("servers");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(servers.size(), 3);
    ASSERT_EQ(servers.at("host1"), 8080);
    ASSERT_EQ(servers.at("host2"), 8081);
    ASSERT_EQ(servers.at("host3"), 8082);
}

/// @brief it should parse unordered map fields successfully.
TEST(testConfig, testUnorderedMapHappyPath) {
    const json j = {{"config", {{"key1", "value1"}, {"key2", "value2"}}}};
    Parser parser(j);
    const auto config = parser.field<std::unordered_map<std::string, std::string>>(
        "config"
    );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.size(), 2);
    ASSERT_EQ(config.at("key1"), "value1");
    ASSERT_EQ(config.at("key2"), "value2");
}

/// @brief it should report error when required map field does not exist.
TEST(testConfig, testMapDoesNotExist) {
    const json j = {};
    Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>("servers");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "servers");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should report error when map field is not an object.
TEST(testConfig, testMapIsNotObject) {
    const json j = {{"servers", "not an object"}};
    Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>("servers");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "servers");
    EXPECT_EQ(err["message"], "expected an object");
}

/// @brief it should use default value for optional missing map field.
TEST(testConfig, testOptionalMapWithDefault) {
    const json j = {};
    Parser parser(j);
    std::map<std::string, int> default_servers = {{"default", 9000}};
    const auto servers = parser.field<std::map<std::string, int>>(
        "servers",
        default_servers
    );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(servers.size(), 1);
    ASSERT_EQ(servers.at("default"), 9000);
}

/// @brief it should report error when map value has invalid type.
TEST(testConfig, testMapWithInvalidValueType) {
    const json j = {{"servers", {{"host1", "not_a_number"}, {"host2", 8081}}}};
    Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>("servers");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "servers.host1");
    EXPECT_TRUE(
        err["message"].get<std::string>().find("expected a number") != std::string::npos
    );
}

/// @brief it should parse nested maps successfully.
TEST(testConfig, testNestedMaps) {
    const json j = {
        {"regions",
         {{"us-east", {{"server1", 8080}, {"server2", 8081}}},
          {"us-west", {{"server3", 9090}, {"server4", 9091}}}}}
    };
    Parser parser(j);
    const auto regions = parser
                             .field<std::map<std::string, std::map<std::string, int>>>(
                                 "regions"
                             );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(regions.size(), 2);
    ASSERT_EQ(regions.at("us-east").size(), 2);
    ASSERT_EQ(regions.at("us-east").at("server1"), 8080);
    ASSERT_EQ(regions.at("us-east").at("server2"), 8081);
    ASSERT_EQ(regions.at("us-west").size(), 2);
    ASSERT_EQ(regions.at("us-west").at("server3"), 9090);
    ASSERT_EQ(regions.at("us-west").at("server4"), 9091);
}

/// @brief it should parse maps with vector values successfully.
TEST(testConfig, testMapWithVectorValues) {
    const json j = {
        {"groups", {{"admin", {1, 2, 3}}, {"user", {4, 5, 6}}, {"guest", {7, 8}}}}
    };
    Parser parser(j);
    const auto groups = parser.field<std::map<std::string, std::vector<int>>>("groups");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(groups.size(), 3);
    ASSERT_EQ(groups.at("admin").size(), 3);
    ASSERT_EQ(groups.at("admin")[0], 1);
    ASSERT_EQ(groups.at("admin")[1], 2);
    ASSERT_EQ(groups.at("admin")[2], 3);
    ASSERT_EQ(groups.at("user").size(), 3);
    ASSERT_EQ(groups.at("guest").size(), 2);
}

/// @brief it should parse maps with constructible type values.
TEST(testConfig, testMapWithConstructibleTypeValues) {
    const json j = {
        {"devices",
         {{"device1", {{"name", "sensor1"}, {"id", 100}}},
          {"device2", {{"name", "sensor2"}, {"id", 200}}}}}
    };
    Parser parser(j);
    const auto devices = parser.field<std::map<std::string, ArrayItem>>("devices");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(devices.size(), 2);
    ASSERT_EQ(devices.at("device1").name, "sensor1");
    ASSERT_EQ(devices.at("device1").id, 100);
    ASSERT_EQ(devices.at("device2").name, "sensor2");
    ASSERT_EQ(devices.at("device2").id, 200);
}

/// @brief it should report error for map with constructible type value errors.
TEST(testConfig, testMapWithConstructibleTypeValuesError) {
    const json j = {
        {"devices",
         {{"device1", {{"name", "sensor1"}, {"id", 100}}},
          {"device2", {{"name", "sensor2"}}}}} // Missing id
    };
    Parser parser(j);
    const auto devices = parser.field<std::map<std::string, ArrayItem>>("devices");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "devices.device2.id");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should parse root object as map with no path argument.
TEST(testConfig, testMapRootParsing) {
    const json j = {{"key1", 10}, {"key2", 20}, {"key3", 30}};
    Parser parser(j);
    const auto map_values = parser.field<std::map<std::string, int>>();

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(map_values.size(), 3);
    ASSERT_EQ(map_values.at("key1"), 10);
    ASSERT_EQ(map_values.at("key2"), 20);
    ASSERT_EQ(map_values.at("key3"), 30);
}

/// @brief it should parse empty object as empty map.
TEST(testConfig, testMapEmptyObject) {
    const json j = {{"config", json::object()}};
    Parser parser(j);
    const auto config = parser.field<std::map<std::string, int>>("config");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.size(), 0);
}

/// @brief it should parse map field using alternative paths.
TEST(testConfig, testMapWithAlternativePaths) {
    const json j = {{"servers_v2", {{"host1", 8080}, {"host2", 8081}}}};
    Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>(
        std::vector<std::string>{"servers", "servers_v2"}
    );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(servers.size(), 2);
    ASSERT_EQ(servers.at("host1"), 8080);
    ASSERT_EQ(servers.at("host2"), 8081);
}

/// @brief it should parse maps with numeric string keys as integer keys.
TEST(testConfig, testMapWithIntKeys) {
    const json j = {{"ports", {{"8080", "http"}, {"8443", "https"}, {"3000", "dev"}}}};
    Parser parser(j);
    const auto ports = parser.field<std::map<int, std::string>>("ports");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(ports.size(), 3);
    ASSERT_EQ(ports.at(8080), "http");
    ASSERT_EQ(ports.at(8443), "https");
    ASSERT_EQ(ports.at(3000), "dev");
}

/// @brief it should parse maps with size_t keys.
TEST(testConfig, testMapWithSizeTKeys) {
    const json j = {{"indices", {{"0", "first"}, {"1", "second"}, {"42", "answer"}}}};
    Parser parser(j);
    const auto indices = parser.field<std::map<size_t, std::string>>("indices");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(indices.size(), 3);
    ASSERT_EQ(indices.at(0), "first");
    ASSERT_EQ(indices.at(1), "second");
    ASSERT_EQ(indices.at(42), "answer");
}

/// @brief it should parse maps with float keys.
TEST(testConfig, testMapWithFloatKeys) {
    const json j = {
        {"thresholds", {{"1.5", "low"}, {"3.14", "medium"}, {"9.99", "high"}}}
    };
    Parser parser(j);
    const auto thresholds = parser.field<std::map<float, std::string>>("thresholds");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(thresholds.size(), 3);
    ASSERT_EQ(thresholds.at(1.5f), "low");
    ASSERT_NEAR(thresholds.count(3.14f), 1, 0.0001f);
    ASSERT_EQ(thresholds.at(9.99f), "high");
}

/// @brief it should report error for invalid numeric key in map.
TEST(testConfig, testMapWithInvalidNumericKey) {
    const json j = {{"ports", {{"8080", "http"}, {"not_a_number", "invalid"}}}};
    Parser parser(j);
    const auto ports = parser.field<std::map<int, std::string>>("ports");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "ports.not_a_number");
    EXPECT_EQ(err["message"], "Invalid numeric key: 'not_a_number'");
}

/// @brief it should parse maps with numeric keys and complex values.
TEST(testConfig, testMapWithNumericKeysAndComplexValues) {
    const json j = {
        {"items",
         {{"0", {{"name", "first"}, {"id", 100}}},
          {"1", {{"name", "second"}, {"id", 200}}},
          {"5", {{"name", "fifth"}, {"id", 500}}}}}
    };
    Parser parser(j);
    const auto items = parser.field<std::map<int, ArrayItem>>("items");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(items.size(), 3);
    ASSERT_EQ(items.at(0).name, "first");
    ASSERT_EQ(items.at(0).id, 100);
    ASSERT_EQ(items.at(1).name, "second");
    ASSERT_EQ(items.at(1).id, 200);
    ASSERT_EQ(items.at(5).name, "fifth");
    ASSERT_EQ(items.at(5).id, 500);
}

/// @brief it should use default for optional map with numeric keys.
TEST(testConfig, testMapWithNumericKeysOptional) {
    const json j = {};
    Parser parser(j);
    std::map<int, std::string> default_ports = {{80, "default_http"}};
    const auto ports = parser.field<std::map<int, std::string>>("ports", default_ports);

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(ports.size(), 1);
    ASSERT_EQ(ports.at(80), "default_http");
}

/// @brief it should parse nested maps with numeric keys.
TEST(testConfig, testMapWithNumericKeysNested) {
    const json j = {
        {"regions",
         {{"0", {{"10", "server1"}, {"20", "server2"}}},
          {"1", {{"30", "server3"}, {"40", "server4"}}}}}
    };
    Parser parser(j);
    const auto regions = parser.field<std::map<int, std::map<int, std::string>>>(
        "regions"
    );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(regions.size(), 2);
    ASSERT_EQ(regions.at(0).size(), 2);
    ASSERT_EQ(regions.at(0).at(10), "server1");
    ASSERT_EQ(regions.at(0).at(20), "server2");
    ASSERT_EQ(regions.at(1).size(), 2);
    ASSERT_EQ(regions.at(1).at(30), "server3");
    ASSERT_EQ(regions.at(1).at(40), "server4");
}

/// @brief it should parse root object as map with numeric keys.
TEST(testConfig, testMapWithNumericKeysRootParsing) {
    const json j = {{"0", 100}, {"1", 200}, {"10", 300}};
    Parser parser(j);
    const auto values = parser.field<std::map<int, int>>();

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 3);
    ASSERT_EQ(values.at(0), 100);
    ASSERT_EQ(values.at(1), 200);
    ASSERT_EQ(values.at(10), 300);
}

/// @brief it should parse unordered maps with numeric keys.
TEST(testConfig, testUnorderedMapWithNumericKeys) {
    const json j = {{"channels", {{"0", "red"}, {"1", "green"}, {"2", "blue"}}}};
    Parser parser(j);
    const auto channels = parser.field<std::unordered_map<int, std::string>>(
        "channels"
    );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(channels.size(), 3);
    ASSERT_EQ(channels.at(0), "red");
    ASSERT_EQ(channels.at(1), "green");
    ASSERT_EQ(channels.at(2), "blue");
}

/// @brief it should handle different key types in separate map fields.
TEST(testConfig, testMapMixedStringAndNumericKeys) {
    // Test that we can handle string keys in one map and numeric keys in another
    const json j = {
        {"string_map", {{"host1", 8080}, {"host2", 8081}}},
        {"numeric_map", {{"0", 100}, {"1", 200}}}
    };
    Parser parser(j);
    const auto string_map = parser.field<std::map<std::string, int>>("string_map");
    const auto numeric_map = parser.field<std::map<int, int>>("numeric_map");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(string_map.size(), 2);
    ASSERT_EQ(string_map.at("host1"), 8080);
    ASSERT_EQ(numeric_map.size(), 2);
    ASSERT_EQ(numeric_map.at(0), 100);
    ASSERT_EQ(numeric_map.at(1), 200);
}

/// @brief it should find field using multiple alternative paths.
TEST(testConfig, testAlternativePathsMultiple) {
    const json j = {{"version_v3", "latest"}};
    Parser parser(j);
    const auto version = parser.field<std::string>(
        std::vector<std::string>{"version", "version_v1", "version_v2", "version_v3"}
    );
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(version, "latest");
}

/// @brief it should use first matching alternative path.
TEST(testConfig, testAlternativePathsFirst) {
    const json j = {{"version", "v1"}};
    Parser parser(j);
    const auto version = parser.field<std::string>(
        std::vector<std::string>{"version", "version_v1", "version_v2", "version_v3"}
    );
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(version, "v1");
}

/// @brief it should use second matching alternative path when first is missing.
TEST(testConfig, testAlternativePathsSecond) {
    const json j = {{"version_v1", "v1"}};
    Parser parser(j);
    const auto version = parser.field<std::string>(
        std::vector<std::string>{"version", "version_v1", "version_v2", "version_v3"}
    );
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(version, "v1");
}

/// @brief it should report error when no alternative paths are found.
TEST(testConfig, testAlternativePathsNoneFound) {
    const json j = {};
    Parser parser(j);
    const auto version = parser.field<std::string>(
        std::vector<std::string>{"version", "version_v1", "version_v2", "version_v3"}
    );
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "version");
    EXPECT_EQ(err["message"], "this field is required");
}

/// @brief it should use default value when no alternative paths are found.
TEST(testConfig, testAlternativePathsWithDefault) {
    const json j = {};
    Parser parser(j);
    const auto version = parser.field<std::string>(
        std::vector<std::string>{"version", "version_v1", "version_v2"},
        "default_version"
    );
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(version, "default_version");
}

/// @brief it should use found value over default in alternative paths.
TEST(testConfig, testAlternativePathsWithDefaultFoundInAlternative) {
    const json j = {{"version_v2", "v2"}};
    Parser parser(j);
    const auto version = parser.field<std::string>(
        std::vector<std::string>{"version", "version_v1", "version_v2"},
        "default_version"
    );
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(version, "v2");
}

/// @brief it should report error when empty paths vector is provided.
TEST(testConfig, testAlternativePathsEmptyVector) {
    const json j = {{"name", "test"}};
    Parser parser(j);
    std::vector<std::string> empty_paths;
    const auto version = parser.field<std::string>(empty_paths);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "");
    EXPECT_EQ(err["message"], "No paths provided");
}

/// @brief it should use default when empty paths vector is provided.
TEST(testConfig, testAlternativePathsEmptyVectorWithDefault) {
    const json j = {{"name", "test"}};
    Parser parser(j);
    std::vector<std::string> empty_paths;
    const auto version = parser.field<std::string>(empty_paths, "default");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(version, "default");
}

/// @brief it should parse nested vectors successfully.
TEST(testConfig, testNestedVectors) {
    const json j = {{"matrix", {{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}}};
    Parser parser(j);
    const auto matrix = parser.field<std::vector<std::vector<int>>>("matrix");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(matrix.size(), 3);
    ASSERT_EQ(matrix[0].size(), 3);
    ASSERT_EQ(matrix[0][0], 1);
    ASSERT_EQ(matrix[0][1], 2);
    ASSERT_EQ(matrix[0][2], 3);
    ASSERT_EQ(matrix[1][0], 4);
    ASSERT_EQ(matrix[1][1], 5);
    ASSERT_EQ(matrix[1][2], 6);
    ASSERT_EQ(matrix[2][0], 7);
    ASSERT_EQ(matrix[2][1], 8);
    ASSERT_EQ(matrix[2][2], 9);
}

/// @brief it should parse empty nested vectors successfully.
TEST(testConfig, testNestedVectorsEmpty) {
    const json j = {{"matrix", json::array()}};
    Parser parser(j);
    const auto matrix = parser.field<std::vector<std::vector<int>>>("matrix");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(matrix.size(), 0);
}

/// @brief it should report error for invalid element in nested vectors.
TEST(testConfig, testNestedVectorsWithError) {
    const json j = {{"matrix", {{1, 2}, {"invalid", 5}, {7, 8}}}};
    Parser parser(j);
    const auto matrix = parser.field<std::vector<std::vector<int>>>("matrix");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "matrix.1.0");
}

/// @brief it should map array elements using callback function.
TEST(testConfig, testMapMethod) {
    const json j = {
        {"items",
         {{{"name", "item1"}, {"id", 1}},
          {{"name", "item2"}, {"id", 2}},
          {{"name", "item3"}, {"id", 3}}}}
    };
    Parser parser(j);
    const auto items = parser.map<ArrayItem>(
        "items",
        [](const Parser &p) -> std::pair<ArrayItem, bool> {
            return {ArrayItem(p), true};
        }
    );
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(items.size(), 3);
    ASSERT_EQ(items[0].name, "item1");
    ASSERT_EQ(items[0].id, 1);
    ASSERT_EQ(items[1].name, "item2");
    ASSERT_EQ(items[1].id, 2);
    ASSERT_EQ(items[2].name, "item3");
    ASSERT_EQ(items[2].id, 3);
}

/// @brief it should filter array elements using map callback.
TEST(testConfig, testMapMethodWithFilter) {
    const json j = {
        {"items",
         {{{"name", "item1"}, {"id", 1}},
          {{"name", "skip"}, {"id", 2}},
          {{"name", "item3"}, {"id", 3}}}}
    };
    const Parser parser(j);
    const auto items = parser.map<ArrayItem>(
        "items",
        [](const Parser &p) -> std::pair<ArrayItem, bool> {
            ArrayItem item(p);
            // Skip items with name "skip"
            if (item.name == "skip") return {item, false};
            return {item, true};
        }
    );
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(items.size(), 2);
    ASSERT_EQ(items[0].name, "item1");
    ASSERT_EQ(items[1].name, "item3");
}

/// @brief it should report error when map field does not exist.
TEST(testConfig, testMapMethodFieldDoesNotExist) {
    const json j = {};
    Parser parser(j);
    const auto items = parser.map<ArrayItem>(
        "items",
        [](const Parser &p) -> std::pair<ArrayItem, bool> {
            return {ArrayItem(p), true};
        }
    );
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "items");
    EXPECT_EQ(err["message"], "this field is required");
}

/// @brief it should report error when map field is not an array.
TEST(testConfig, testMapMethodFieldNotArray) {
    const json j = {{"items", "not an array"}};
    Parser parser(j);
    const auto items = parser.map<ArrayItem>(
        "items",
        [](const Parser &p) -> std::pair<ArrayItem, bool> {
            return {ArrayItem(p), true};
        }
    );
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "items");
    EXPECT_EQ(err["message"], "expected an array");
}

/// @brief it should report errors from map callback with correct path.
TEST(testConfig, testMapMethodWithErrors) {
    const json j = {
        {"items",
         {{{"name", "item1"}, {"id", 1}},
          {{"name", "item2"}}, // Missing id
          {{"name", "item3"}, {"id", 3}}}}
    };
    Parser parser(j);
    const auto items = parser.map<ArrayItem>(
        "items",
        [](const Parser &p) -> std::pair<ArrayItem, bool> {
            return {ArrayItem(p), true};
        }
    );
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "items.1.id");
    EXPECT_EQ(err["message"], "This field is required");
}

/// @brief it should return valid parser for existing optional child.
TEST(testConfig, testOptionalChildExists) {
    const json j = {{"child", {{"name", "test"}, {"value", 42}}}};
    Parser parser(j);
    auto child = parser.optional_child("child");
    const auto name = child.field<std::string>("name");
    const auto value = child.field<int>("value");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(name, "test");
    ASSERT_EQ(value, 42);
}

/// @brief it should return noop parser for missing optional child.
TEST(testConfig, testOptionalChildMissing) {
    const json j = {};
    Parser parser(j);
    auto child = parser.optional_child("child");
    // Should not accumulate error when child is missing
    EXPECT_TRUE(parser.ok());
    // Trying to use the child parser should be a noop
    const auto name = child.field<std::string>("name");
    EXPECT_TRUE(parser.ok()); // Still ok because child is noop
}

/// @brief it should report error for optional child with invalid type.
TEST(testConfig, testOptionalChildInvalidType) {
    const json j = {{"child", "not an object"}};
    Parser parser(j);
    auto child = parser.optional_child("child");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "child");
    EXPECT_EQ(err["message"], "expected an object or array");
}

/// @brief it should handle arrays in optional child parser.
TEST(testConfig, testOptionalChildArray) {
    const json j = {{"items", {{{"name", "a"}}, {{"name", "b"}}}}};
    const Parser parser(j);
    const auto items_parser = parser.optional_child("items");
    EXPECT_TRUE(parser.ok());
    // Arrays are valid for optional_child - verify we can use it
    EXPECT_TRUE(items_parser.ok());
}

/// @brief it should convert string array elements to float.
TEST(testConfig, testVectorStringToNumber) {
    const json j = {{"values", {"1.5", "2.5", "3.5"}}};
    Parser parser(j);
    const auto values = parser.field<std::vector<float>>("values");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 3);
    ASSERT_NEAR(values[0], 1.5f, 0.0001f);
    ASSERT_NEAR(values[1], 2.5f, 0.0001f);
    ASSERT_NEAR(values[2], 3.5f, 0.0001f);
}

/// @brief it should convert string array elements to int.
TEST(testConfig, testVectorStringToInt) {
    const json j = {{"ports", {"8080", "8443", "3000"}}};
    Parser parser(j);
    const auto ports = parser.field<std::vector<int>>("ports");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(ports.size(), 3);
    ASSERT_EQ(ports[0], 8080);
    ASSERT_EQ(ports[1], 8443);
    ASSERT_EQ(ports[2], 3000);
}

/// @brief it should report error for invalid string to number conversion.
TEST(testConfig, testVectorStringToNumberInvalid) {
    const json j = {{"values", {"1.5", "invalid", "3.5"}}};
    Parser parser(j);
    const auto values = parser.field<std::vector<float>>("values");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "values.1");
    EXPECT_EQ(err["message"], "expected a number, got 'invalid'");
}

/// @brief it should return true for existing fields via has method.
TEST(testConfig, testHasFieldExists) {
    const json j = {{"name", "test"}, {"value", 42}};
    const Parser parser(j);
    EXPECT_TRUE(parser.has("name"));
    EXPECT_TRUE(parser.has("value"));
    EXPECT_TRUE(parser.ok());
}

/// @brief it should return false for non-existing fields via has method.
TEST(testConfig, testHasFieldDoesNotExist) {
    const json j = {{"name", "test"}};
    const Parser parser(j);
    EXPECT_FALSE(parser.has("missing"));
    EXPECT_FALSE(parser.has("value"));
    EXPECT_TRUE(parser.ok()); // has() should not accumulate errors
}

/// @brief it should return false for any field on noop parser.
TEST(testConfig, testHasNoopParser) {
    const Parser parser; // Default constructor creates noop parser
    EXPECT_FALSE(parser.has("anything"));
    EXPECT_FALSE(parser.ok()); // noop parser is never ok
}

/// @brief it should check field existence on child parser.
TEST(testConfig, testHasOnChildParser) {
    const json j = {{"parent", {{"child_field", "value"}, {"another", 123}}}};
    Parser parser(j);
    auto child = parser.child("parent");
    EXPECT_TRUE(child.has("child_field"));
    EXPECT_TRUE(child.has("another"));
    EXPECT_FALSE(child.has("missing"));
    EXPECT_TRUE(parser.ok());
}

/// @brief it should return true for field with null value via has method.
TEST(testConfig, testHasWithNullValue) {
    const json j = {{"null_field", nullptr}, {"string_field", "test"}};
    const Parser parser(j);
    EXPECT_TRUE(parser.has("null_field")); // Field exists even if value is null
    EXPECT_TRUE(parser.has("string_field"));
    EXPECT_TRUE(parser.ok());
}

/// @brief it should return false for any field on empty object.
TEST(testConfig, testHasEmptyObject) {
    const json j = json::object();
    Parser parser(j);
    EXPECT_FALSE(parser.has("anything"));
    EXPECT_TRUE(parser.ok());
}

/// @brief it should support conditional parsing based on field existence.
TEST(testConfig, testHasConditionalParsing) {
    // Test a common use case: conditionally parse based on field existence
    const json j = {{"type", "sensor"}, {"threshold", 3.14}};
    Parser parser(j);

    float threshold = 0.0f;
    int count = 0;
    const auto type = parser.field<std::string>("type");
    if (parser.has("threshold")) { threshold = parser.field<float>("threshold"); }
    if (parser.has("count")) { count = parser.field<int>("count"); }

    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(type, "sensor");
    EXPECT_NEAR(threshold, 3.14f, 0.0001f);
    EXPECT_EQ(count, 0); // Not parsed since field doesn't exist
}
}
