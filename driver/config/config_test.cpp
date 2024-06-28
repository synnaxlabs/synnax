// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external.
#include "gtest/gtest.h"

/// local.
#include "driver/config/config.h"

TEST(testConfig, testParserHappyPath) {
    struct MyConfig {
        std::string name;
        std::float_t dog;
    };
    MyConfig v;

    const json j = {
        {"name", "test"},
        {"dog", 1.0}
    };
    config::Parser parser(j);
    v.name = parser.required<std::string>("name");
    v.dog = parser.optional<std::float_t>("dog", 12);
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(v.name, "test");
    ASSERT_EQ(v.dog, 1.0);
}

TEST(testConfig, testParserFieldDoesnNotExist) {
    struct MyConfig {
        std::string name;
        std::float_t dog{};
    };
    MyConfig v;
    json j = {};
    config::Parser parser(j);
    v.name = parser.required<std::string>("name");
    v.dog = parser.optional<std::float_t>("dog", 12);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "name");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testParserFieldHasInvalidType) {
    struct MyConfig {
        std::string name;
        std::float_t dog{};
    };
    MyConfig v;
    json j = {
        {"name", "test"},
        {"dog", "1.0"}
    };
    config::Parser parser(j);
    v.name = parser.required<std::string>("name");
    v.dog = parser.optional<std::float_t>("dog", 12);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "dog");
    EXPECT_EQ(err["message"], "type must be number, but is string");
}

TEST(testConfig, testParserFieldChildHappyPath) {
    struct MyChildConfig {
        std::string name;
        std::float_t dog;
    };

    struct MyConfig {
        MyChildConfig child;
    };

    json j = {
        {
            "child", {
                {"name", "test"},
                {"dog", 1.0}
            }
        }
    };
    MyConfig v;
    config::Parser parser(j);
    auto child_parser = parser.child("child");
    v.child.name = child_parser.required<std::string>("name");
    v.child.dog = child_parser.optional<std::float_t>("dog", 12);
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(v.child.name, "test");
    ASSERT_EQ(v.child.dog, 1.0);
}

TEST(testConfig, testParserFieldChildDoesNotExist) {
    struct MyChildConfig {
        std::string name;
        std::float_t dog;
    };

    struct MyConfig {
        MyChildConfig child;
    };

    json j = {};
    MyConfig v;
    config::Parser parser(j);
    auto child_parser = parser.child("child");
    v.child.name = child_parser.required<std::string>("name");
    v.child.dog = child_parser.optional<std::float_t>("dog", 12);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "child");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testParserChildFieldInvalidType) {
    struct MyChildConfig {
        std::string name;
        std::float_t dog;
    };

    struct MyConfig {
        MyChildConfig child;
    };

    json j = {
        {
            "child", {
                {"name", "test"},
                {"dog", "1.0"}
            }
        }
    };
    MyConfig v;
    config::Parser parser(j);
    auto child_parser = parser.child("child");
    v.child.name = child_parser.required<std::string>("name");
    v.child.dog = child_parser.optional<std::float_t>("dog", 12);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "child.dog");
    EXPECT_EQ(err["message"], "type must be number, but is string");
}

TEST(testConfig, testIterHappyPath) {
    struct MyChildConfig {
        std::string name;
        std::float_t dog;
    };

    struct MyConfig {
        std::vector<MyChildConfig> children;
    };

    const json j = {
        {
            "children", {
                {
                    {"name", "test1"},
                    {"dog", 1.0}
                },
                {
                    {"name", "test2"},
                    {"dog", 2.0}
                }
            }
        }
    };

    MyConfig v;
    const config::Parser parser(j);
    parser.iter("children", [&](config::Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.required<std::string>("name");
        child.dog = child_parser.optional<std::float_t>("dog", 12);
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
        std::float_t dog;
    };

    struct MyConfig {
        std::vector<MyChildConfig> children;
    };

    const json j = {};
    MyConfig v;
    const config::Parser parser(j);
    parser.iter("children", [&](config::Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.required<std::string>("name");
        child.dog = child_parser.optional<std::float_t>("dog", 12);
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
        std::float_t dog;
    };

    struct MyConfig {
        std::vector<MyChildConfig> children;
    };

    const json j = {
        {
            "children", {
                {"name", "test1"},
                {"dog", 1.0}
            }
        }
    };
    MyConfig v;
    const config::Parser parser(j);
    parser.iter("children", [&](config::Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.required<std::string>("name");
        child.dog = child_parser.optional<std::float_t>("dog", 12);
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
        std::float_t dog;
    };

    struct MyConfig {
        std::vector<MyChildConfig> children;
    };

    const json j = {
        {
            "children", {
                {
                    {"name", "test1"},
                    {"dog", "1.0"}
                },
                {
                    {"name", "test2"},
                    {"dog", 2.0}
                }
            }
        }
    };

    MyConfig v;
    const config::Parser parser(j);
    parser.iter("children", [&](config::Parser &child_parser) {
        MyChildConfig child;
        child.name = child_parser.required<std::string>("name");
        child.dog = child_parser.optional<std::float_t>("dog", 12);
        v.children.push_back(child);
    });
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "children.0.dog");
    EXPECT_EQ(err["message"], "type must be number, but is string");
}

TEST(testConfig, testInterpretStringAsNumber) {
    struct MyConfig {
        std::float_t dog;
    };
    // const json j = {
    //     {"dog", "1.232"}
    // };

    json j;
    j["dog"] = 1.232;
    MyConfig v;
    config::Parser parser(j);
    v.dog = parser.required<std::float_t>("dog");
    EXPECT_TRUE(parser.ok());
    // assert that the value is close to the expected value.
    ASSERT_NEAR(v.dog, 1.232, 0.0001);
}

TEST(testConfig, testArray){
    json j = {
        {"array", {1, 2, 3, 4, 5}}
    };
    config::Parser parser(j);
    auto values = parser.required_array<int>("array");
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 5);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
    ASSERT_EQ(values[3], 4);
    ASSERT_EQ(values[4], 5);
}

TEST(testConfig, testArrayDoesNotExist){
    json j = {};
    config::Parser parser(j);
    auto values = parser.required_array<int>("array");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "array");
    EXPECT_EQ(err["message"], "This field is required");
}

TEST(testConfig, testArrayIsNotArray){
    json j = {
        {"array", 1}
    };
    config::Parser parser(j);
    auto values = parser.required_array<int>("array");
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "array");
    EXPECT_EQ(err["message"], "Expected an array");
}

TEST(testConfig, testOptionalArray){
    json j = {
        {"array", {1, 2, 3, 4, 5}}
    };
    config::Parser parser(j);
    auto values = parser.optional_array<int>("array", {6, 7, 8});
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(values.size(), 5);
    ASSERT_EQ(values[0], 1);
    ASSERT_EQ(values[1], 2);
    ASSERT_EQ(values[2], 3);
    ASSERT_EQ(values[3], 4);
    ASSERT_EQ(values[4], 5);
}