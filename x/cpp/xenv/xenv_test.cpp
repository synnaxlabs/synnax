// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xenv/xenv.h"

class XEnvTest : public ::testing::Test {
protected:
    void SetUp() override {
        xenv::set("TEST_STRING", "hello");
        xenv::set("TEST_INT", "42");
        xenv::set("TEST_FLOAT", "3.14");
        xenv::set("TEST_INVALID_NUM", "not_a_number");
        xenv::set("TEST_UINT16", "65000");
    }

    void TearDown() override {
        xenv::unset("TEST_STRING");
        xenv::unset("TEST_INT");
        xenv::unset("TEST_FLOAT");
        xenv::unset("TEST_INVALID_NUM");
        xenv::unset("TEST_UINT16");
    }
};

/// @brief it should load string values from environment variables.
TEST_F(XEnvTest, LoadString) {
    EXPECT_EQ(xenv::load("TEST_STRING", std::string("default")), "hello");
    EXPECT_EQ(xenv::load("NONEXISTENT_VAR", std::string("default")), "default");
}

/// @brief it should load integer values from environment variables.
TEST_F(XEnvTest, LoadInt) {
    EXPECT_EQ(xenv::load("TEST_INT", 0), 42);
    EXPECT_EQ(xenv::load("NONEXISTENT_VAR", 100), 100);
    EXPECT_EQ(xenv::load("TEST_INVALID_NUM", 100), 100);
}

/// @brief it should load float values from environment variables.
TEST_F(XEnvTest, LoadFloat) {
    EXPECT_FLOAT_EQ(xenv::load("TEST_FLOAT", 0.0f), 3.14f);
    EXPECT_FLOAT_EQ(xenv::load("NONEXISTENT_VAR", 1.5f), 1.5f);
    EXPECT_FLOAT_EQ(xenv::load("TEST_INVALID_NUM", 1.5f), 1.5f);
}

/// @brief it should load double values from environment variables.
TEST_F(XEnvTest, LoadDouble) {
    EXPECT_DOUBLE_EQ(xenv::load("TEST_FLOAT", 0.0), 3.14);
    EXPECT_DOUBLE_EQ(xenv::load("NONEXISTENT_VAR", 1.5), 1.5);
    EXPECT_DOUBLE_EQ(xenv::load("TEST_INVALID_NUM", 1.5), 1.5);
}

/// @brief it should load long values from environment variables.
TEST_F(XEnvTest, LoadLong) {
    EXPECT_EQ(xenv::load("TEST_INT", 0L), 42L);
    EXPECT_EQ(xenv::load("NONEXISTENT_VAR", 100L), 100L);
    EXPECT_EQ(xenv::load("TEST_INVALID_NUM", 100L), 100L);
}

/// @brief it should load uint16 values from environment variables.
TEST_F(XEnvTest, LoadUInt16) {
    EXPECT_EQ(
        xenv::load("TEST_UINT16", static_cast<uint16_t>(0)),
        static_cast<uint16_t>(65000)
    );
    EXPECT_EQ(
        xenv::load("NONEXISTENT_VAR", static_cast<uint16_t>(100)),
        static_cast<uint16_t>(100)
    );
    EXPECT_EQ(
        xenv::load("TEST_INVALID_NUM", static_cast<uint16_t>(100)),
        static_cast<uint16_t>(100)
    );
}

/// @brief it should load boolean values with true as default.
TEST_F(XEnvTest, LoadBooleanTrueDefault) {
    xenv::set("TEST_BOOL_TRUE", "true");
    EXPECT_EQ(xenv::load("TEST_BOOL_TRUE", false), true);

    xenv::set("TEST_BOOL_ONE", "1");
    EXPECT_EQ(xenv::load("TEST_BOOL_ONE", false), true);

    xenv::set("TEST_BOOL_ZERO", "0");
    EXPECT_EQ(xenv::load("TEST_BOOL_ZERO", true), false);

    xenv::set("TEST_BOOL_FALSE", "false");
    EXPECT_EQ(xenv::load("TEST_BOOL_FALSE", true), false);
}

/// @brief it should load boolean values with false as default.
TEST_F(XEnvTest, LoadBooleanFalseDefault) {
    xenv::set("TEST_BOOL_TRUE", "true");
    EXPECT_EQ(xenv::load("TEST_BOOL_TRUE", true), true);

    xenv::set("TEST_BOOL_ONE", "1");
    EXPECT_EQ(xenv::load("TEST_BOOL_ONE", true), true);

    xenv::set("TEST_BOOL_ZERO", "0");
    EXPECT_EQ(xenv::load("TEST_BOOL_ZERO", false), false);

    xenv::set("TEST_BOOL_FALSE", "false");
    EXPECT_EQ(xenv::load("TEST_BOOL_FALSE", false), false);
}

/// @brief it should automatically convert variable names to screaming case.
TEST_F(XEnvTest, AutomaticCaseConversion) {
    xenv::set("HELLO_WORLD", "test_value");
    xenv::set("ANOTHER_TEST_VAR", "42");

    // Should work with snake_case input
    EXPECT_EQ(xenv::load("hello_world", std::string("default")), "test_value");
    EXPECT_EQ(xenv::load("another_test_var", 0), 42);

    // Should also work with already screaming case
    EXPECT_EQ(xenv::load("HELLO_WORLD", std::string("default")), "test_value");
    EXPECT_EQ(xenv::load("ANOTHER_TEST_VAR", 0), 42);

    xenv::unset("HELLO_WORLD");
    xenv::unset("ANOTHER_TEST_VAR");
}

/// @brief it should handle mixed case variable names correctly.
TEST_F(XEnvTest, CaseConversionWithMixedCase) {
    xenv::set("MIXED_CASE_VALUE", "success");

    EXPECT_EQ(xenv::load("mixed_case_value", std::string("default")), "success");
    EXPECT_EQ(xenv::load("MIXED_CASE_VALUE", std::string("default")), "success");
    EXPECT_EQ(xenv::load("Mixed_Case_Value", std::string("default")), "success");

    xenv::unset("MIXED_CASE_VALUE");
}

/// @brief it should support prefixed environment variable loading.
TEST_F(XEnvTest, ParserWithPrefix) {
    xenv::set("APP_TEST_STRING", "prefixed");
    xenv::set("APP_TEST_INT", "123");

    // Test with prefix without underscore
    xenv::Parser parser("app");
    EXPECT_EQ(parser.field("test_string", std::string("default")), "prefixed");
    EXPECT_EQ(parser.field("test_int", 0), 123);
    EXPECT_EQ(parser.field("nonexistent", std::string("default")), "default");

    // Test with prefix with underscore
    xenv::Parser parser2("app_");
    EXPECT_EQ(parser2.field("test_string", std::string("default")), "prefixed");
    EXPECT_EQ(parser2.field("test_int", 0), 123);

    xenv::unset("APP_TEST_STRING");
    xenv::unset("APP_TEST_INT");
}

/// @brief it should handle mixed case prefixes correctly.
TEST_F(XEnvTest, ParserWithMixedCasePrefix) {
    xenv::set("MY_APP_TEST_VALUE", "mixed_case_prefix");

    // Test different prefix case styles - all should access the same env var
    const xenv::Parser parser1("my_app");
    const xenv::Parser parser2("MY_APP");
    const xenv::Parser parser3("My_App");

    EXPECT_EQ(parser1.field("test_value", std::string("default")), "mixed_case_prefix");
    EXPECT_EQ(parser2.field("test_value", std::string("default")), "mixed_case_prefix");
    EXPECT_EQ(parser3.field("test_value", std::string("default")), "mixed_case_prefix");

    xenv::unset("MY_APP_TEST_VALUE");
}

/// @brief it should work correctly with an empty prefix.
TEST_F(XEnvTest, EmptyPrefix) {
    // Ensure empty prefix works the same as the global load function
    const xenv::Parser parser("");
    EXPECT_EQ(parser.field("TEST_STRING", std::string("default")), "hello");
    EXPECT_EQ(parser.field("TEST_INT", 0), 42);
    EXPECT_EQ(parser.field("NONEXISTENT_VAR", std::string("default")), "default");
}

/// @brief it should support multiple parser instances with different prefixes.
TEST_F(XEnvTest, MultipleParserInstances) {
    xenv::set("APP1_VALUE", "first");
    xenv::set("APP2_VALUE", "second");

    const xenv::Parser parser1("app1");
    const xenv::Parser parser2("app2");

    EXPECT_EQ(parser1.field("value", std::string("default")), "first");
    EXPECT_EQ(parser2.field("value", std::string("default")), "second");

    xenv::unset("APP1_VALUE");
    xenv::unset("APP2_VALUE");
}
