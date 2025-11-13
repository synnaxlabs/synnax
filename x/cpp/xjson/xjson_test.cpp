// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

/// local.
#include "x/cpp/xjson/xjson.h"

TEST(testConfig, testParserHappyPath) {
    struct MyConfig {
        std::string name;
        float dog;
    };
    MyConfig v;

    const json j = {{"name", "test"}, {"dog", 1.0}};
    xjson::Parser parser(j);
    v.name = parser.field<std::string>("name");
    v.dog = parser.field<float>("dog", 12.0f);
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(v.name, "test");
    ASSERT_EQ(v.dog, 1.0);
}

TEST(testConfig, testParserFieldDoesnNotExist) {
    struct MyConfig {
        std::string name;
        float dog{};
    };
    MyConfig v;
    const json j = {};
    xjson::Parser parser(j);
    v.name = parser.field<std::string>("name");
    v.dog = parser.field<float>("dog", 12.0f);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "name");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testParserFieldHasInvalidType) {
    struct MyConfig {
        std::string name;
        float dog{};
    };
    MyConfig v;
    const json j = {{"name", "test"}, {"dog", "cat"}};
    xjson::Parser parser(j);
    v.name = parser.field<std::string>("name");
    v.dog = parser.field<float>("dog", 12.0f);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "dog");
    EXPECT_EQ(err["message"], "expected a number, got 'cat'");
}

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
    xjson::Parser parser(j);
    auto child_parser = parser.child("child");
    v.child.name = child_parser.field<std::string>("name");
    v.child.dog = child_parser.field<float>("dog", 12.0f);
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(v.child.name, "test");
    ASSERT_EQ(v.child.dog, 1.0);
}

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
    xjson::Parser parser(j);
    auto child_parser = parser.child("child");
    v.child.name = child_parser.field<std::string>("name");
    v.child.dog = child_parser.field<float>("dog", 12.0f);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "child");
    EXPECT_EQ(err["message"], "This field is required");
}

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
    xjson::Parser parser(j);
    auto child_parser = parser.child("child");
    v.child.name = child_parser.field<std::string>("name");
    v.child.dog = child_parser.field<float>("dog", 12.0f);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "child.dog");
    EXPECT_EQ(err["message"], "expected a number, got 'cat'");
}

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
    const xjson::Parser parser(j);
    parser.iter("children", [&](xjson::Parser &child_parser) {
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
    const xjson::Parser parser(j);
    parser.iter("children", [&](xjson::Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.field<std::string>("name");
        child.dog = child_parser.field<float>("dog", 12.0f);
        v.children.push_back(child);
    });
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "children");
    EXPECT_EQ(err["message"], "This field is required");
}

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
    const xjson::Parser parser(j);
    parser.iter("children", [&](xjson::Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.field<std::string>("name");
        child.dog = child_parser.field<float>("dog", 12.0f);
        v.children.push_back(child);
    });
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "children");
    EXPECT_EQ(err["message"], "Expected an array");
}

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
    const xjson::Parser parser(j);
    parser.iter("children", [&](xjson::Parser &child_parser) {
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

TEST(testConfig, testInterpretStringAsNumber) {
    struct MyConfig {
        float dog;
    };
    const json j = {{"dog", "1.232"}};
    MyConfig v;
    xjson::Parser parser(j);
    v.dog = parser.field<float>("dog");
    EXPECT_TRUE(parser.ok());
    // assert that the value is close to the expected value.
    ASSERT_NEAR(v.dog, 1.232, 0.0001);
}

TEST(testConfig, testArray) {
    const json j = {{"array", {1, 2, 3, 4, 5}}};
    xjson::Parser parser(j);
    const auto values = parser.field<std::vector<int>>("array");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 5);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
    ASSERT_EQ(values[3], 4);
    ASSERT_EQ(values[4], 5);
}

TEST(testConfig, testArrayDoesNotExist) {
    const json j = {};
    xjson::Parser parser(j);
    auto values = parser.field<std::vector<int>>("array");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "array");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testArrayIsNotArray) {
    const json j = {{"array", 1}};
    xjson::Parser parser(j);
    auto values = parser.field<std::vector<int>>("array");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "array");
    EXPECT_EQ(err["message"], "Expected an array");
}

TEST(testConfig, testOptionalArray) {
    const json j = {{"array", {1, 2, 3, 4, 5}}};
    xjson::Parser parser(j);
    const auto values = parser.field<std::vector<int>>("array", {6, 7, 8});
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 5);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
    ASSERT_EQ(values[3], 4);
    ASSERT_EQ(values[4], 5);
}

TEST(testConfig, testNoError) {
    const json j = {};
    const xjson::Parser parser(j);
    const auto err = parser.error();
    ASSERT_FALSE(err);
}

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
    auto parser = xjson::Parser::from_file_path(test_file);
    v.name = parser.field<std::string>("name");
    v.value = parser.field<float>("value");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(v.name, "test");
    ASSERT_EQ(v.value, 42.5);

    // Clean up
    std::remove(test_file.c_str());
}

TEST(testConfig, testParseFromFileFailure) {
    auto parser = xjson::Parser::from_file_path("nonexistent_file.json");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "");
    EXPECT_EQ(err["message"], "failed to open file: nonexistent_file.json");
}

TEST(testConfig, testParseFromFileInvalidJSON) {
    // Create a temporary test file with invalid JSON
    std::string test_file = "invalid_config.json";
    std::ofstream file(test_file);
    file << R"({
        "name": "test",
        invalid json here
    })";
    file.close();

    auto parser = xjson::Parser::from_file_path(test_file);
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

TEST(testConfig, testFieldErrWithXError) {
    const json j = {};
    xjson::Parser parser(j);

    xerrors::Error custom_error(xerrors::VALIDATION, "Custom validation error");
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

// ============================================================================
// Tests for Parser-Constructible Types
// ============================================================================

// Test structs defined outside TEST for proper template instantiation
struct BasicConstructibleConfig {
    std::string name;
    int value;

    explicit BasicConstructibleConfig(xjson::Parser p):
        name(p.field<std::string>("name")), value(p.field<int>("value")) {}

    BasicConstructibleConfig(): value(0) {}
};

struct NestedInnerConfig {
    std::string type;
    float threshold;

    explicit NestedInnerConfig(xjson::Parser p):
        type(p.field<std::string>("type")), threshold(p.field<float>("threshold")) {}

    NestedInnerConfig(): threshold(0.0f) {}
};

struct NestedOuterConfig {
    std::string name;
    NestedInnerConfig inner;

    explicit NestedOuterConfig(xjson::Parser p):
        name(p.field<std::string>("name")),
        inner(p.field<NestedInnerConfig>("inner")) {}

    NestedOuterConfig() {}
};

struct MissingFieldConfig {
    std::string name;
    int value;

    explicit MissingFieldConfig(xjson::Parser p):
        name(p.field<std::string>("name")), value(p.field<int>("value")) {}

    MissingFieldConfig(): value(0) {}
};

struct InvalidTypeConfig {
    std::string name;
    int value;

    explicit InvalidTypeConfig(xjson::Parser p):
        name(p.field<std::string>("name")), value(p.field<int>("value")) {}

    InvalidTypeConfig(): value(0) {}
};

struct NestedErrorInnerConfig {
    int required_value;

    explicit NestedErrorInnerConfig(xjson::Parser p):
        required_value(p.field<int>("required_value")) {}

    NestedErrorInnerConfig(): required_value(0) {}
};

struct NestedErrorOuterConfig {
    std::string name;
    NestedErrorInnerConfig inner;

    explicit NestedErrorOuterConfig(xjson::Parser p):
        name(p.field<std::string>("name")),
        inner(p.field<NestedErrorInnerConfig>("inner")) {}

    NestedErrorOuterConfig() {}
};

struct ArrayItem {
    std::string name;
    int id;

    explicit ArrayItem(xjson::Parser p):
        name(p.field<std::string>("name")), id(p.field<int>("id")) {}

    ArrayItem(): id(0) {}
};

struct MixedInnerConfig {
    float value;

    explicit MixedInnerConfig(xjson::Parser p): value(p.field<float>("value")) {}

    MixedInnerConfig(): value(0.0f) {}
};

struct MixedOuterConfig {
    std::string name;
    int count;
    MixedInnerConfig nested;
    bool enabled;

    explicit MixedOuterConfig(xjson::Parser p):
        name(p.field<std::string>("name")),
        count(p.field<int>("count")),
        nested(p.field<MixedInnerConfig>("nested")),
        enabled(p.field<bool>("enabled", true)) {}

    MixedOuterConfig(): count(0), enabled(false) {}
};

struct ParentMissingConfig {
    std::string name;

    explicit ParentMissingConfig(xjson::Parser p): name(p.field<std::string>("name")) {}

    ParentMissingConfig() {}
};

struct ParentInvalidTypeConfig {
    std::string name;

    explicit ParentInvalidTypeConfig(xjson::Parser p):
        name(p.field<std::string>("name")) {}

    ParentInvalidTypeConfig() {}
};

struct OptionalConfig {
    std::string name;
    int value;

    explicit OptionalConfig(xjson::Parser p):
        name(p.field<std::string>("name")), value(p.field<int>("value")) {}

    OptionalConfig(): value(99) {}
};

// Verify that the trait can see our test structs
static_assert(
    xjson::is_parser_constructible_v<BasicConstructibleConfig>,
    "BasicConstructibleConfig should be detected as parser-constructible"
);
static_assert(
    xjson::is_parser_constructible_v<NestedInnerConfig>,
    "NestedInnerConfig should be detected as parser-constructible"
);

TEST(testConfig, testBasicConstructibleType) {
    const json j = {{"config", {{"name", "test"}, {"value", 42}}}};
    xjson::Parser parser(j);
    auto config = parser.field<BasicConstructibleConfig>("config");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.name, "test");
    ASSERT_EQ(config.value, 42);
}

TEST(testConfig, testNestedConstructibleTypes) {
    const json j = {
        {"config",
         {{"name", "outer"}, {"inner", {{"type", "sensor"}, {"threshold", 3.14f}}}}}
    };

    xjson::Parser parser(j);
    auto config = parser.field<NestedOuterConfig>("config");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.name, "outer");
    ASSERT_EQ(config.inner.type, "sensor");
    ASSERT_NEAR(config.inner.threshold, 3.14f, 0.0001f);
}

TEST(testConfig, testConstructibleTypeWithMissingRequiredField) {
    const json j = {{"config", {{"name", "test"}}}};
    xjson::Parser parser(j);
    auto config = parser.field<MissingFieldConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config.value");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testConstructibleTypeWithInvalidType) {
    const json j = {{"config", {{"name", "test"}, {"value", "not_a_number"}}}};
    xjson::Parser parser(j);
    auto config = parser.field<InvalidTypeConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config.value");
    EXPECT_TRUE(
        err["message"].get<std::string>().find("expected a number") != std::string::npos
    );
}

TEST(testConfig, testConstructibleTypeWithNestedError) {
    // Create an empty JSON object explicitly
    json empty_obj = json::object();
    const json j = {{"config", {{"name", "outer"}, {"inner", empty_obj}}}};

    xjson::Parser parser(j);
    auto config = parser.field<NestedErrorOuterConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config.inner.required_value");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testArrayOfConstructibleTypes) {
    const json j = {
        {"items",
         {{{"name", "item1"}, {"id", 1}},
          {{"name", "item2"}, {"id", 2}},
          {{"name", "item3"}, {"id", 3}}}}
    };

    xjson::Parser parser(j);
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

TEST(testConfig, testArrayOfConstructibleTypesWithError) {
    const json j = {
        {"items",
         {{{"name", "item1"}, {"id", 1}},
          {{"name", "item2"}}, // Missing id
          {{"name", "item3"}, {"id", 3}}}}
    };

    xjson::Parser parser(j);
    auto items = parser.field<std::vector<ArrayItem>>("items");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "items.1.id");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testMixedConstructibleAndPrimitiveTypes) {
    const json j = {
        {"config",
         {{"name", "mixed"},
          {"count", 5},
          {"nested", {{"value", 2.5f}}},
          {"enabled", false}}}
    };

    xjson::Parser parser(j);
    auto config = parser.field<MixedOuterConfig>("config");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.name, "mixed");
    ASSERT_EQ(config.count, 5);
    ASSERT_NEAR(config.nested.value, 2.5f, 0.0001f);
    ASSERT_FALSE(config.enabled);
}

TEST(testConfig, testConstructibleTypeParentMissing) {
    const json j = {};
    xjson::Parser parser(j);
    auto config = parser.field<ParentMissingConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testConstructibleTypeParentInvalidType) {
    const json j = {{"config", "not_an_object"}};
    xjson::Parser parser(j);
    auto config = parser.field<ParentInvalidTypeConfig>("config");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config");
    EXPECT_EQ(err["message"], "Expected an object or array");
}

TEST(testConfig, testOptionalConstructibleType) {
    const json j = {};
    xjson::Parser parser(j);
    OptionalConfig default_config;
    default_config.name = "default";
    default_config.value = 100;

    auto config = parser.field<OptionalConfig>("config", default_config);

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.name, "default");
    ASSERT_EQ(config.value, 100);
}

TEST(testConfig, testEmptyPathBehaviorParsesRoot) {
    // Test 1: Empty path now parses the root object as a parser-constructible type
    const json j1 = {{"name", "test"}, {"id", 42}};
    xjson::Parser parser1(j1);
    auto item1 = parser1.field<ArrayItem>(""); // Empty string means parse root
    EXPECT_TRUE(parser1.ok());
    EXPECT_EQ(item1.name, "test");
    EXPECT_EQ(item1.id, 42);

    // Test 2: Empty path parses root scalar
    const json j2 = "hello";
    xjson::Parser parser2(j2);
    auto val2 = parser2.field<std::string>("");
    EXPECT_TRUE(parser2.ok());
    EXPECT_EQ(val2, "hello");

    // Test 3: Empty path parses root array
    const json j3 = json::array({1, 2, 3});
    xjson::Parser parser3(j3);
    auto val3 = parser3.field<std::vector<int>>("");
    EXPECT_TRUE(parser3.ok());
    ASSERT_EQ(val3.size(), 3);
    EXPECT_EQ(val3[0], 1);

    // Test 4: No-args and empty string are equivalent
    const json j4 = 123;
    xjson::Parser parser4a(j4);
    xjson::Parser parser4b(j4);
    auto val4a = parser4a.field<int>();
    auto val4b = parser4b.field<int>("");
    EXPECT_TRUE(parser4a.ok());
    EXPECT_TRUE(parser4b.ok());
    EXPECT_EQ(val4a, val4b);
    EXPECT_EQ(val4a, 123);
}

// ============================================================================
// Tests for field<T>() - parsing root/current parser value
// ============================================================================

TEST(testConfig, testFieldNoArgsWithRootArray) {
    const json j = json::array({1, 2, 3, 4, 5});
    xjson::Parser parser(j);
    const auto values = parser.field<std::vector<int>>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 5);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
    ASSERT_EQ(values[3], 4);
    ASSERT_EQ(values[4], 5);
}

TEST(testConfig, testFieldNoArgsWithRootArrayStrings) {
    const json j = json::array({"a", "b", "c"});
    xjson::Parser parser(j);
    const auto values = parser.field<std::vector<std::string>>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 3);
    ASSERT_EQ(values[0], "a");
    ASSERT_EQ(values[1], "b");
    ASSERT_EQ(values[2], "c");
}

TEST(testConfig, testFieldNoArgsWithConstructibleTypes) {
    const json j = json::array(
        {{{"name", "item1"}, {"id", 1}},
         {{"name", "item2"}, {"id", 2}},
         {{"name", "item3"}, {"id", 3}}}
    );
    xjson::Parser parser(j);
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

TEST(testConfig, testFieldNoArgsRootNotArray) {
    const json j = {{"name", "test"}, {"value", 42}};
    xjson::Parser parser(j);
    const auto values = parser.field<std::vector<int>>();
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "");
    EXPECT_EQ(err["message"], "Expected an array");
}

TEST(testConfig, testFieldNoArgsWithError) {
    const json j = json::array(
        {{{"name", "item1"}, {"id", 1}},
         {{"name", "item2"}}, // Missing id
         {{"name", "item3"}, {"id", 3}}}
    );
    xjson::Parser parser(j);
    const auto items = parser.field<std::vector<ArrayItem>>();
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "1.id");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testFieldNoArgsEmpty) {
    const json j = json::array({});
    xjson::Parser parser(j);
    const auto values = parser.field<std::vector<int>>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 0);
}

TEST(testConfig, testFieldNoArgsWithRootScalar) {
    const json j = 42;
    xjson::Parser parser(j);
    const auto value = parser.field<int>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(value, 42);
}

TEST(testConfig, testFieldNoArgsWithRootObject) {
    const json j = {{"name", "test"}, {"id", 123}};
    xjson::Parser parser(j);
    const auto item = parser.field<ArrayItem>();
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(item.name, "test");
    ASSERT_EQ(item.id, 123);
}

TEST(testConfig, testFieldEmptyStringEquivalentToNoArgs) {
    const json j = json::array({1, 2, 3});
    xjson::Parser parser(j);
    const auto values = parser.field<std::vector<int>>("");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 3);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
}

// ============================================================================
// Tests for Map Support
// ============================================================================

TEST(testConfig, testMapHappyPath) {
    const json j = {{"servers", {{"host1", 8080}, {"host2", 8081}, {"host3", 8082}}}};
    xjson::Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>("servers");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(servers.size(), 3);
    ASSERT_EQ(servers.at("host1"), 8080);
    ASSERT_EQ(servers.at("host2"), 8081);
    ASSERT_EQ(servers.at("host3"), 8082);
}

TEST(testConfig, testUnorderedMapHappyPath) {
    const json j = {{"config", {{"key1", "value1"}, {"key2", "value2"}}}};
    xjson::Parser parser(j);
    const auto config = parser.field<std::unordered_map<std::string, std::string>>(
        "config"
    );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.size(), 2);
    ASSERT_EQ(config.at("key1"), "value1");
    ASSERT_EQ(config.at("key2"), "value2");
}

TEST(testConfig, testMapDoesNotExist) {
    const json j = {};
    xjson::Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>("servers");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "servers");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testMapIsNotObject) {
    const json j = {{"servers", "not an object"}};
    xjson::Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>("servers");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "servers");
    EXPECT_EQ(err["message"], "Expected an object");
}

TEST(testConfig, testOptionalMapWithDefault) {
    const json j = {};
    xjson::Parser parser(j);
    std::map<std::string, int> default_servers = {{"default", 9000}};
    const auto servers = parser.field<std::map<std::string, int>>(
        "servers",
        default_servers
    );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(servers.size(), 1);
    ASSERT_EQ(servers.at("default"), 9000);
}

TEST(testConfig, testMapWithInvalidValueType) {
    const json j = {{"servers", {{"host1", "not_a_number"}, {"host2", 8081}}}};
    xjson::Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>("servers");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "servers.host1");
    EXPECT_TRUE(
        err["message"].get<std::string>().find("expected a number") != std::string::npos
    );
}

TEST(testConfig, testNestedMaps) {
    const json j = {
        {"regions",
         {{"us-east", {{"server1", 8080}, {"server2", 8081}}},
          {"us-west", {{"server3", 9090}, {"server4", 9091}}}}}
    };
    xjson::Parser parser(j);
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

TEST(testConfig, testMapWithVectorValues) {
    const json j = {
        {"groups", {{"admin", {1, 2, 3}}, {"user", {4, 5, 6}}, {"guest", {7, 8}}}}
    };
    xjson::Parser parser(j);
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

TEST(testConfig, testMapWithConstructibleTypeValues) {
    const json j = {
        {"devices",
         {{"device1", {{"name", "sensor1"}, {"id", 100}}},
          {"device2", {{"name", "sensor2"}, {"id", 200}}}}}
    };
    xjson::Parser parser(j);
    const auto devices = parser.field<std::map<std::string, ArrayItem>>("devices");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(devices.size(), 2);
    ASSERT_EQ(devices.at("device1").name, "sensor1");
    ASSERT_EQ(devices.at("device1").id, 100);
    ASSERT_EQ(devices.at("device2").name, "sensor2");
    ASSERT_EQ(devices.at("device2").id, 200);
}

TEST(testConfig, testMapWithConstructibleTypeValuesError) {
    const json j = {
        {"devices",
         {{"device1", {{"name", "sensor1"}, {"id", 100}}},
          {"device2", {{"name", "sensor2"}}}}} // Missing id
    };
    xjson::Parser parser(j);
    const auto devices = parser.field<std::map<std::string, ArrayItem>>("devices");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "devices.device2.id");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testMapRootParsing) {
    const json j = {{"key1", 10}, {"key2", 20}, {"key3", 30}};
    xjson::Parser parser(j);
    const auto map_values = parser.field<std::map<std::string, int>>();

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(map_values.size(), 3);
    ASSERT_EQ(map_values.at("key1"), 10);
    ASSERT_EQ(map_values.at("key2"), 20);
    ASSERT_EQ(map_values.at("key3"), 30);
}

TEST(testConfig, testMapEmptyObject) {
    const json j = {{"config", json::object()}};
    xjson::Parser parser(j);
    const auto config = parser.field<std::map<std::string, int>>("config");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(config.size(), 0);
}

TEST(testConfig, testMapWithAlternativePaths) {
    const json j = {{"servers_v2", {{"host1", 8080}, {"host2", 8081}}}};
    xjson::Parser parser(j);
    const auto servers = parser.field<std::map<std::string, int>>(
        "servers",
        "servers_v2"
    );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(servers.size(), 2);
    ASSERT_EQ(servers.at("host1"), 8080);
    ASSERT_EQ(servers.at("host2"), 8081);
}

// ============================================================================
// Tests for Numeric Key Map Support
// ============================================================================

TEST(testConfig, testMapWithIntKeys) {
    const json j = {{"ports", {{"8080", "http"}, {"8443", "https"}, {"3000", "dev"}}}};
    xjson::Parser parser(j);
    const auto ports = parser.field<std::map<int, std::string>>("ports");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(ports.size(), 3);
    ASSERT_EQ(ports.at(8080), "http");
    ASSERT_EQ(ports.at(8443), "https");
    ASSERT_EQ(ports.at(3000), "dev");
}

TEST(testConfig, testMapWithSizeTKeys) {
    const json j = {{"indices", {{"0", "first"}, {"1", "second"}, {"42", "answer"}}}};
    xjson::Parser parser(j);
    const auto indices = parser.field<std::map<size_t, std::string>>("indices");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(indices.size(), 3);
    ASSERT_EQ(indices.at(0), "first");
    ASSERT_EQ(indices.at(1), "second");
    ASSERT_EQ(indices.at(42), "answer");
}

TEST(testConfig, testMapWithFloatKeys) {
    const json j = {
        {"thresholds", {{"1.5", "low"}, {"3.14", "medium"}, {"9.99", "high"}}}
    };
    xjson::Parser parser(j);
    const auto thresholds = parser.field<std::map<float, std::string>>("thresholds");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(thresholds.size(), 3);
    ASSERT_EQ(thresholds.at(1.5f), "low");
    ASSERT_NEAR(thresholds.count(3.14f), 1, 0.0001f);
    ASSERT_EQ(thresholds.at(9.99f), "high");
}

TEST(testConfig, testMapWithInvalidNumericKey) {
    const json j = {{"ports", {{"8080", "http"}, {"not_a_number", "invalid"}}}};
    xjson::Parser parser(j);
    const auto ports = parser.field<std::map<int, std::string>>("ports");

    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "ports.not_a_number");
    EXPECT_EQ(err["message"], "Invalid numeric key: 'not_a_number'");
}

TEST(testConfig, testMapWithNumericKeysAndComplexValues) {
    const json j = {
        {"items",
         {{"0", {{"name", "first"}, {"id", 100}}},
          {"1", {{"name", "second"}, {"id", 200}}},
          {"5", {{"name", "fifth"}, {"id", 500}}}}}
    };
    xjson::Parser parser(j);
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

TEST(testConfig, testMapWithNumericKeysOptional) {
    const json j = {};
    xjson::Parser parser(j);
    std::map<int, std::string> default_ports = {{80, "default_http"}};
    const auto ports = parser.field<std::map<int, std::string>>("ports", default_ports);

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(ports.size(), 1);
    ASSERT_EQ(ports.at(80), "default_http");
}

TEST(testConfig, testMapWithNumericKeysNested) {
    const json j = {
        {"regions",
         {{"0", {{"10", "server1"}, {"20", "server2"}}},
          {"1", {{"30", "server3"}, {"40", "server4"}}}}}
    };
    xjson::Parser parser(j);
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

TEST(testConfig, testMapWithNumericKeysRootParsing) {
    const json j = {{"0", 100}, {"1", 200}, {"10", 300}};
    xjson::Parser parser(j);
    const auto values = parser.field<std::map<int, int>>();

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 3);
    ASSERT_EQ(values.at(0), 100);
    ASSERT_EQ(values.at(1), 200);
    ASSERT_EQ(values.at(10), 300);
}

TEST(testConfig, testUnorderedMapWithNumericKeys) {
    const json j = {{"channels", {{"0", "red"}, {"1", "green"}, {"2", "blue"}}}};
    xjson::Parser parser(j);
    const auto channels = parser.field<std::unordered_map<int, std::string>>(
        "channels"
    );

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(channels.size(), 3);
    ASSERT_EQ(channels.at(0), "red");
    ASSERT_EQ(channels.at(1), "green");
    ASSERT_EQ(channels.at(2), "blue");
}

TEST(testConfig, testMapMixedStringAndNumericKeys) {
    // Test that we can handle string keys in one map and numeric keys in another
    const json j = {
        {"string_map", {{"host1", 8080}, {"host2", 8081}}},
        {"numeric_map", {{"0", 100}, {"1", 200}}}
    };
    xjson::Parser parser(j);
    const auto string_map = parser.field<std::map<std::string, int>>("string_map");
    const auto numeric_map = parser.field<std::map<int, int>>("numeric_map");

    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(string_map.size(), 2);
    ASSERT_EQ(string_map.at("host1"), 8080);
    ASSERT_EQ(numeric_map.size(), 2);
    ASSERT_EQ(numeric_map.at(0), 100);
    ASSERT_EQ(numeric_map.at(1), 200);
}
