// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/caseconv/caseconv.h"

TEST(CaseConvTest, SnakeToScreamBasic) {
    EXPECT_EQ(caseconv::snake_to_scream("hello_world"), "HELLO_WORLD");
    EXPECT_EQ(caseconv::snake_to_scream("simple_test"), "SIMPLE_TEST");
}

TEST(CaseConvTest, SnakeToScreamAlreadyUpper) {
    EXPECT_EQ(caseconv::snake_to_scream("HELLO_WORLD"), "HELLO_WORLD");
    EXPECT_EQ(caseconv::snake_to_scream("ALREADY_UPPER"), "ALREADY_UPPER");
}

TEST(CaseConvTest, SnakeToScreamMixedCase) {
    EXPECT_EQ(caseconv::snake_to_scream("Hello_World"), "HELLO_WORLD");
    EXPECT_EQ(caseconv::snake_to_scream("Mixed_Case_String"), "MIXED_CASE_STRING");
}

TEST(CaseConvTest, SnakeToScreamEdgeCases) {
    EXPECT_EQ(caseconv::snake_to_scream(""), "");
    EXPECT_EQ(caseconv::snake_to_scream("a"), "A");
    EXPECT_EQ(caseconv::snake_to_scream("hello__world"), "HELLO__WORLD");
    EXPECT_EQ(caseconv::snake_to_scream("hello_world_123"), "HELLO_WORLD_123");
    EXPECT_EQ(caseconv::snake_to_scream("hello-world"), "HELLO-WORLD");
    EXPECT_EQ(
        caseconv::snake_to_scream("mixed_case-with-hyphens"),
        "MIXED_CASE-WITH-HYPHENS"
    );
    EXPECT_EQ(
        caseconv::snake_to_scream("multiple--hyphens__underscores"),
        "MULTIPLE--HYPHENS__UNDERSCORES"
    );
}

TEST(CaseConvTest, SnakeToKebabBasic) {
    EXPECT_EQ(caseconv::snake_to_kebab("hello_world"), "hello-world");
    EXPECT_EQ(caseconv::snake_to_kebab("simple_test"), "simple-test");
}

TEST(CaseConvTest, SnakeToKebabAlreadyKebab) {
    EXPECT_EQ(caseconv::snake_to_kebab("hello-world"), "hello-world");
    EXPECT_EQ(caseconv::snake_to_kebab("already-kebab"), "already-kebab");
}

TEST(CaseConvTest, SnakeToKebabMixedCase) {
    EXPECT_EQ(caseconv::snake_to_kebab("Hello_World"), "Hello-World");
    EXPECT_EQ(caseconv::snake_to_kebab("Mixed_Case_String"), "Mixed-Case-String");
}

TEST(CaseConvTest, SnakeToKebabEdgeCases) {
    EXPECT_EQ(caseconv::snake_to_kebab(""), "");
    EXPECT_EQ(caseconv::snake_to_kebab("a"), "a");
    EXPECT_EQ(caseconv::snake_to_kebab("hello__world"), "hello--world");
    EXPECT_EQ(caseconv::snake_to_kebab("hello_world_123"), "hello-world-123");
    EXPECT_EQ(caseconv::snake_to_kebab("hello-world_123"), "hello-world-123");
}
