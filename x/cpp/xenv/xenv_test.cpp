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
#ifdef _WIN32
        _putenv_s("TEST_STRING", "hello");
        _putenv_s("TEST_INT", "42");
        _putenv_s("TEST_FLOAT", "3.14");
        _putenv_s("TEST_INVALID_NUM", "not_a_number");
        _putenv_s("TEST_UINT16", "65000");
#else
        setenv("TEST_STRING", "hello", 1);
        setenv("TEST_INT", "42", 1);
        setenv("TEST_FLOAT", "3.14", 1);
        setenv("TEST_INVALID_NUM", "not_a_number", 1);
        setenv("TEST_UINT16", "65000", 1);
#endif
    }

    void TearDown() override {
#ifdef _WIN32
        _putenv_s("TEST_STRING", "");
        _putenv_s("TEST_INT", "");
        _putenv_s("TEST_FLOAT", "");
        _putenv_s("TEST_INVALID_NUM", "");
        _putenv_s("TEST_UINT16", "");
#else
        unsetenv("TEST_STRING");
        unsetenv("TEST_INT");
        unsetenv("TEST_FLOAT");
        unsetenv("TEST_INVALID_NUM");
        unsetenv("TEST_UINT16");
#endif
    }
};

TEST_F(XEnvTest, LoadString) {
    EXPECT_EQ(xenv::load("TEST_STRING", std::string("default")), "hello");
    EXPECT_EQ(xenv::load("NONEXISTENT_VAR", std::string("default")), "default");
}

TEST_F(XEnvTest, LoadInt) {
    EXPECT_EQ(xenv::load("TEST_INT", 0), 42);
    EXPECT_EQ(xenv::load("NONEXISTENT_VAR", 100), 100);
    EXPECT_EQ(xenv::load("TEST_INVALID_NUM", 100), 100);
}

TEST_F(XEnvTest, LoadFloat) {
    EXPECT_FLOAT_EQ(xenv::load("TEST_FLOAT", 0.0f), 3.14f);
    EXPECT_FLOAT_EQ(xenv::load("NONEXISTENT_VAR", 1.5f), 1.5f);
    EXPECT_FLOAT_EQ(xenv::load("TEST_INVALID_NUM", 1.5f), 1.5f);
}

TEST_F(XEnvTest, LoadDouble) {
    EXPECT_DOUBLE_EQ(xenv::load("TEST_FLOAT", 0.0), 3.14);
    EXPECT_DOUBLE_EQ(xenv::load("NONEXISTENT_VAR", 1.5), 1.5);
    EXPECT_DOUBLE_EQ(xenv::load("TEST_INVALID_NUM", 1.5), 1.5);
}

TEST_F(XEnvTest, LoadLong) {
    EXPECT_EQ(xenv::load("TEST_INT", 0L), 42L);
    EXPECT_EQ(xenv::load("NONEXISTENT_VAR", 100L), 100L);
    EXPECT_EQ(xenv::load("TEST_INVALID_NUM", 100L), 100L);
}

TEST_F(XEnvTest, LoadUInt16) {
    EXPECT_EQ(xenv::load("TEST_UINT16", uint16_t(0)), uint16_t(65000));
    EXPECT_EQ(xenv::load("NONEXISTENT_VAR", uint16_t(100)), uint16_t(100));
    EXPECT_EQ(xenv::load("TEST_INVALID_NUM", uint16_t(100)), uint16_t(100));
}

TEST_F(XEnvTest, LoadBooleanTrueDefault) {
    setenv("TEST_BOOL_TRUE", "true", 1);
    EXPECT_EQ(xenv::load("TEST_BOOL_TRUE", false), true); 

    setenv("TEST_BOOL_ONE", "1", 1);
    EXPECT_EQ(xenv::load("TEST_BOOL_ONE", false), true); 

    setenv("TEST_BOOL_ZERO", "0", 1);
    EXPECT_EQ(xenv::load("TEST_BOOL_ZERO", true), false); 

    setenv("TEST_BOOL_FALSE", "false", 1);
    EXPECT_EQ(xenv::load("TEST_BOOL_FALSE", true), false); 
}

TEST_F(XEnvTest, LoadBooleanFalseDefault) {
    setenv("TEST_BOOL_TRUE", "true", 1);
    EXPECT_EQ(xenv::load("TEST_BOOL_TRUE", true), true); 

    setenv("TEST_BOOL_ONE", "1", 1);
    EXPECT_EQ(xenv::load("TEST_BOOL_ONE", true), true); 

    setenv("TEST_BOOL_ZERO", "0", 1);
    EXPECT_EQ(xenv::load("TEST_BOOL_ZERO", false), false); 

    setenv("TEST_BOOL_FALSE", "false", 1);
    EXPECT_EQ(xenv::load("TEST_BOOL_FALSE", false), false); 
}