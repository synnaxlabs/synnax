// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/env/env.h"
#include "x/cpp/test/test.h"

namespace x::env {
class XEnvTest : public ::testing::Test {
protected:
    void SetUp() override {
        set("TEST_STRING", "hello");
        set("TEST_INT", "42");
        set("TEST_FLOAT", "3.14");
        set("TEST_INVALID_NUM", "not_a_number");
        set("TEST_UINT16", "65000");
    }

    void TearDown() override {
        unset("TEST_STRING");
        unset("TEST_INT");
        unset("TEST_FLOAT");
        unset("TEST_INVALID_NUM");
        unset("TEST_UINT16");
    }
};

/// @brief it should load string values from environment variables.
TEST_F(XEnvTest, LoadString) {
    EXPECT_EQ(load("TEST_STRING", std::string("default")), "hello");
    EXPECT_EQ(load("NONEXISTENT_VAR", std::string("default")), "default");
}

/// @brief it should load integer values from environment variables.
TEST_F(XEnvTest, LoadInt) {
    EXPECT_EQ(load("TEST_INT", 0), 42);
    EXPECT_EQ(load("NONEXISTENT_VAR", 100), 100);
    EXPECT_EQ(load("TEST_INVALID_NUM", 100), 100);
}

/// @brief it should load float values from environment variables.
TEST_F(XEnvTest, LoadFloat) {
    EXPECT_FLOAT_EQ(load("TEST_FLOAT", 0.0f), 3.14f);
    EXPECT_FLOAT_EQ(load("NONEXISTENT_VAR", 1.5f), 1.5f);
    EXPECT_FLOAT_EQ(load("TEST_INVALID_NUM", 1.5f), 1.5f);
}

/// @brief it should load double values from environment variables.
TEST_F(XEnvTest, LoadDouble) {
    EXPECT_DOUBLE_EQ(load("TEST_FLOAT", 0.0), 3.14);
    EXPECT_DOUBLE_EQ(load("NONEXISTENT_VAR", 1.5), 1.5);
    EXPECT_DOUBLE_EQ(load("TEST_INVALID_NUM", 1.5), 1.5);
}

/// @brief it should load long values from environment variables.
TEST_F(XEnvTest, LoadLong) {
    EXPECT_EQ(load("TEST_INT", 0L), 42L);
    EXPECT_EQ(load("NONEXISTENT_VAR", 100L), 100L);
    EXPECT_EQ(load("TEST_INVALID_NUM", 100L), 100L);
}

/// @brief it should load uint16 values from environment variables.
TEST_F(XEnvTest, LoadUInt16) {
    EXPECT_EQ(
        load("TEST_UINT16", static_cast<uint16_t>(0)),
        static_cast<uint16_t>(65000)
    );
    EXPECT_EQ(
        load("NONEXISTENT_VAR", static_cast<uint16_t>(100)),
        static_cast<uint16_t>(100)
    );
    EXPECT_EQ(
        load("TEST_INVALID_NUM", static_cast<uint16_t>(100)),
        static_cast<uint16_t>(100)
    );
}

/// @brief it should load boolean values with true as default.
TEST_F(XEnvTest, LoadBooleanTrueDefault) {
    set("TEST_BOOL_TRUE", "true");
    EXPECT_EQ(load("TEST_BOOL_TRUE", false), true);

    set("TEST_BOOL_ONE", "1");
    EXPECT_EQ(load("TEST_BOOL_ONE", false), true);

    set("TEST_BOOL_ZERO", "0");
    EXPECT_EQ(load("TEST_BOOL_ZERO", true), false);

    set("TEST_BOOL_FALSE", "false");
    EXPECT_EQ(load("TEST_BOOL_FALSE", true), false);
}

/// @brief it should load boolean values with false as default.
TEST_F(XEnvTest, LoadBooleanFalseDefault) {
    set("TEST_BOOL_TRUE", "true");
    EXPECT_EQ(load("TEST_BOOL_TRUE", true), true);

    set("TEST_BOOL_ONE", "1");
    EXPECT_EQ(load("TEST_BOOL_ONE", true), true);

    set("TEST_BOOL_ZERO", "0");
    EXPECT_EQ(load("TEST_BOOL_ZERO", false), false);

    set("TEST_BOOL_FALSE", "false");
    EXPECT_EQ(load("TEST_BOOL_FALSE", false), false);
}

/// @brief it should automatically convert variable names to screaming case.
TEST_F(XEnvTest, AutomaticCaseConversion) {
    set("HELLO_WORLD", "test_value");
    set("ANOTHER_TEST_VAR", "42");

    EXPECT_EQ(load("hello_world", std::string("default")), "test_value");
    EXPECT_EQ(load("another_test_var", 0), 42);

    EXPECT_EQ(load("HELLO_WORLD", std::string("default")), "test_value");
    EXPECT_EQ(load("ANOTHER_TEST_VAR", 0), 42);

    unset("HELLO_WORLD");
    unset("ANOTHER_TEST_VAR");
}

/// @brief it should handle mixed case variable names correctly.
TEST_F(XEnvTest, CaseConversionWithMixedCase) {
    set("MIXED_CASE_VALUE", "success");

    EXPECT_EQ(load("mixed_case_value", std::string("default")), "success");
    EXPECT_EQ(load("MIXED_CASE_VALUE", std::string("default")), "success");
    EXPECT_EQ(load("Mixed_Case_Value", std::string("default")), "success");

    unset("MIXED_CASE_VALUE");
}

/// @brief it should support prefixed environment variable loading.
TEST_F(XEnvTest, ParserWithPrefix) {
    set("APP_TEST_STRING", "prefixed");
    set("APP_TEST_INT", "123");

    Parser parser("app");
    EXPECT_EQ(parser.field("test_string", std::string("default")), "prefixed");
    EXPECT_EQ(parser.field("test_int", 0), 123);
    EXPECT_EQ(parser.field("nonexistent", std::string("default")), "default");

    Parser parser2("app_");
    EXPECT_EQ(parser2.field("test_string", std::string("default")), "prefixed");
    EXPECT_EQ(parser2.field("test_int", 0), 123);

    unset("APP_TEST_STRING");
    unset("APP_TEST_INT");
}

/// @brief it should handle mixed case prefixes correctly.
TEST_F(XEnvTest, ParserWithMixedCasePrefix) {
    set("MY_APP_TEST_VALUE", "mixed_case_prefix");

    Parser parser1("my_app");
    Parser parser2("MY_APP");
    Parser parser3("My_App");

    EXPECT_EQ(parser1.field("test_value", std::string("default")), "mixed_case_prefix");
    EXPECT_EQ(parser2.field("test_value", std::string("default")), "mixed_case_prefix");
    EXPECT_EQ(parser3.field("test_value", std::string("default")), "mixed_case_prefix");

    unset("MY_APP_TEST_VALUE");
}

/// @brief it should work correctly with an empty prefix.
TEST_F(XEnvTest, EmptyPrefix) {
    Parser parser("");
    EXPECT_EQ(parser.field("TEST_STRING", std::string("default")), "hello");
    EXPECT_EQ(parser.field("TEST_INT", 0), 42);
    EXPECT_EQ(parser.field("NONEXISTENT_VAR", std::string("default")), "default");
}

/// @brief it should support multiple parser instances with different prefixes.
TEST_F(XEnvTest, MultipleParserInstances) {
    set("APP1_VALUE", "first");
    set("APP2_VALUE", "second");

    Parser parser1("app1");
    Parser parser2("app2");

    EXPECT_EQ(parser1.field("value", std::string("default")), "first");
    EXPECT_EQ(parser2.field("value", std::string("default")), "second");

    unset("APP1_VALUE");
    unset("APP2_VALUE");
}

/// @brief it should return true from ok() when all conversions succeed.
TEST_F(XEnvTest, TestOk) {
    Parser parser("");
    parser.field("TEST_STRING", std::string("default"));
    parser.field("TEST_INT", 0);
    ASSERT_TRUE(parser.ok());
    ASSERT_NIL(parser.error());
}

/// @brief it should return a VALIDATION error on conversion failure.
TEST_F(XEnvTest, TestError) {
    Parser parser("");
    parser.field("TEST_INVALID_NUM", 0);
    ASSERT_FALSE(parser.ok());
    ASSERT_OCCURRED_AS(parser.error(), errors::VALIDATION);
    ASSERT_NE(parser.error().message().find("TEST_INVALID_NUM"), std::string::npos);
    ASSERT_NE(parser.error().message().find("failed to convert"), std::string::npos);
}

/// @brief it should accumulate errors via field_err with a message.
TEST_F(XEnvTest, TestFieldErr) {
    Parser parser("");
    ASSERT_TRUE(parser.ok());
    parser.field_err("host", "must not be empty");
    ASSERT_FALSE(parser.ok());
    ASSERT_OCCURRED_AS(parser.error(), errors::VALIDATION);
    ASSERT_EQ(parser.error().message(), "[sy.validation] host: must not be empty");
}

/// @brief it should accumulate errors via field_err with an existing error.
TEST_F(XEnvTest, TestFieldErrWithError) {
    Parser parser("");
    errors::Error err(errors::VALIDATION, "connection refused");
    parser.field_err("host", err);
    ASSERT_FALSE(parser.ok());
    ASSERT_OCCURRED_AS(parser.error(), errors::VALIDATION);
    ASSERT_EQ(parser.error().message(), "[sy.validation] host: connection refused");
}

/// @brief it should accumulate multiple conversion errors.
TEST_F(XEnvTest, TestMultipleConversionErrors) {
    set("BAD_INT", "not_a_number");
    set("BAD_FLOAT", "also_not_a_number");

    Parser parser("");
    parser.field("BAD_INT", 0);
    parser.field("BAD_FLOAT", 0.0);
    ASSERT_FALSE(parser.ok());
    ASSERT_EQ(parser.errors.size(), 2);

    unset("BAD_INT");
    unset("BAD_FLOAT");
}
}
