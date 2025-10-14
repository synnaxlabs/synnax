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
    v.name = parser.required<std::string>("name");
    v.dog = parser.optional<float>("dog", 12);
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
    v.name = parser.required<std::string>("name");
    v.dog = parser.optional<float>("dog", 12);
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
    v.name = parser.required<std::string>("name");
    v.dog = parser.optional<float>("dog", 12);
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
    v.child.name = child_parser.required<std::string>("name");
    v.child.dog = child_parser.optional<float>("dog", 12);
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
    v.child.name = child_parser.required<std::string>("name");
    v.child.dog = child_parser.optional<float>("dog", 12);
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
    v.child.name = child_parser.required<std::string>("name");
    v.child.dog = child_parser.optional<float>("dog", 12);
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
        child.name = child_parser.required<std::string>("name");
        child.dog = child_parser.optional<float>("dog", 12);
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
        child.name = child_parser.required<std::string>("name");
        child.dog = child_parser.optional<float>("dog", 12);
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
        child.name = child_parser.required<std::string>("name");
        child.dog = child_parser.optional<float>("dog", 12);
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
        child.name = child_parser.required<std::string>("name");
        child.dog = child_parser.optional<float>("dog", 12);
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
    v.dog = parser.required<float>("dog");
    EXPECT_TRUE(parser.ok());
    // assert that the value is close to the expected value.
    ASSERT_NEAR(v.dog, 1.232, 0.0001);
}

TEST(testConfig, testArray) {
    const json j = {{"array", {1, 2, 3, 4, 5}}};
    xjson::Parser parser(j);
    const auto values = parser.required_vec<int>("array");
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
    auto values = parser.required_vec<int>("array");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "array");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testArrayIsNotArray) {
    const json j = {{"array", 1}};
    xjson::Parser parser(j);
    auto values = parser.required_vec<int>("array");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "array");
    EXPECT_EQ(err["message"], "Expected an array");
}

TEST(testConfig, testOptionalArray) {
    const json j = {{"array", {1, 2, 3, 4, 5}}};
    xjson::Parser parser(j);
    const auto values = parser.optional_vec<int>("array", {6, 7, 8});
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
    v.name = parser.required<std::string>("name");
    v.value = parser.required<float>("value");

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
