
/// external.
#include "gtest/gtest.h"

/// local.
#include "driver/driver/config/config.h"

TEST(testConfig, testParserHappyPath) {
    struct  MyConfig {
        std::string name;
        std::float_t dog;
    };
    MyConfig v;

    json j = {
        {"name", "test"},
        {"dog", 1.0}
    };
    config::Parser builder(j);
    v.name = builder.required<std::string>("name");
    v.dog = builder.optional<std::float_t>("dog", 12);
    EXPECT_TRUE(builder.ok());
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
    config::Parser builder(j);
    v.name = builder.required<std::string>("name");
    v.dog = builder.optional<std::float_t>("dog", 12);
    EXPECT_FALSE(builder.ok());
    EXPECT_EQ(builder.errors->size(), 1);
    auto err = builder.errors->at(0);
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
    config::Parser builder(j);
    v.name = builder.required<std::string>("name");
    v.dog = builder.optional<std::float_t>("dog", 12);
    EXPECT_FALSE(builder.ok());
    EXPECT_EQ(builder.errors->size(), 1);
    auto err = builder.errors->at(0);
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
        {"child", {
            {"name", "test"},
            {"dog", 1.0}
        }}
    };
    MyConfig v;
    config::Parser builder(j);
    auto child_builder = builder.child("child");
    v.child.name = child_builder.required<std::string>("name");
    v.child.dog = child_builder.optional<std::float_t>("dog", 12);
    EXPECT_TRUE(builder.ok());
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
    config::Parser builder(j);
    auto child_builder = builder.child("child");
    v.child.name = child_builder.required<std::string>("name");
    v.child.dog = child_builder.optional<std::float_t>("dog", 12);
    EXPECT_FALSE(builder.ok());
    EXPECT_EQ(builder.errors->size(), 1);
    auto err = builder.errors->at(0);
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
        {"child", {
            {"name", "test"},
            {"dog", "1.0"}
        }}
    };
    MyConfig v;
    config::Parser builder(j);
    auto child_builder = builder.child("child");
    v.child.name = child_builder.required<std::string>("name");
    v.child.dog = child_builder.optional<std::float_t>("dog", 12);
    EXPECT_FALSE(builder.ok());
    EXPECT_EQ(builder.errors->size(), 1);
    auto err = builder.errors->at(0);
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

    json j = {
        {"children", {
            {
                {"name", "test1"},
                {"dog", 1.0}
            },
            {
                {"name", "test2"},
                {"dog", 2.0}
            }
        }}
    };

    MyConfig v;
    config::Parser builder(j);
    builder.iter("children", [&](config::Parser& child_builder) {
        MyChildConfig child;
        child.name = child_builder.required<std::string>("name");
        child.dog = child_builder.optional<std::float_t>("dog", 12);
        v.children.push_back(child);
    });
    EXPECT_TRUE(builder.ok());
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

    json j = {};
    MyConfig v;
    config::Parser builder(j);
    builder.iter("children", [&](config::Parser& child_builder) {
        MyChildConfig child;
        child.name = child_builder.required<std::string>("name");
        child.dog = child_builder.optional<std::float_t>("dog", 12);
        v.children.push_back(child);
    });
    EXPECT_FALSE(builder.ok());
    EXPECT_EQ(builder.errors->size(), 1);
    auto err = builder.errors->at(0);
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

    json j = {
        {"children", {
            {"name", "test1"},
            {"dog", 1.0}
        }
        }
    };
    MyConfig v;
    config::Parser builder(j);
    builder.iter("children", [&](config::Parser& child_builder) {
        MyChildConfig child;
        child.name = child_builder.required<std::string>("name");
        child.dog = child_builder.optional<std::float_t>("dog", 12);
        v.children.push_back(child);
    });
    EXPECT_FALSE(builder.ok());
    EXPECT_EQ(builder.errors->size(), 1);
    auto err = builder.errors->at(0);
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

    json j = {
        {"children", {
            {
                {"name", "test1"},
                {"dog", "1.0"}
            },
            {
                {"name", "test2"},
                {"dog", 2.0}
            }
        }}
    };

    MyConfig v;
    config::Parser builder(j);
    builder.iter("children", [&](config::Parser& child_builder) {
        MyChildConfig child;
        child.name = child_builder.required<std::string>("name");
        child.dog = child_builder.optional<std::float_t>("dog", 12);
        v.children.push_back(child);
    });
    EXPECT_FALSE(builder.ok());
    EXPECT_EQ(builder.errors->size(), 1);
    auto err = builder.errors->at(0);
    EXPECT_EQ(err["path"], "children.0.dog");
    EXPECT_EQ(err["message"], "type must be number, but is string");
}

