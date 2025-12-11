// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <stdexcept>

#include "gtest/gtest.h"

#include "x/cpp/defer/defer.h"

/// @brief it should call the function when the scope ends.
TEST(DeferTests, BasicFunctionality) {
    bool called = false;
    {
        x::defer d([&called] { called = true; });
        ASSERT_FALSE(called);
    }
    ASSERT_TRUE(called);
}

/// @brief it should call the functions in reverse order.
TEST(DeferTests, MultipleDefers) {
    int counter = 0;
    {
        x::defer d1([&counter] { counter += 1; });
        x::defer d2([&counter] { counter += 2; });
        x::defer d3([&counter] { counter += 3; });
        ASSERT_EQ(counter, 0);
    }
    ASSERT_EQ(counter, 6);
}

/// @brief it should call a deferred function even after an early return.
TEST(DeferTests, EarlyReturn) {
    bool called_after_early_return = false;
    bool called_after_normal_return = false;

    {
        auto test_function = [](const bool early_return, bool &called_after) -> bool {
            x::defer d([&called_after] { called_after = true; });
            if (early_return) return false;
            return true;
        };
        test_function(true, called_after_early_return);
    }

    {
        auto test_function = [](bool early_return, bool &called_after) -> bool {
            x::defer d([&called_after] { called_after = true; });
            if (early_return) return false;
            return true;
        };

        test_function(false, called_after_normal_return);
    }

    ASSERT_TRUE(called_after_early_return);
    ASSERT_TRUE(called_after_normal_return);
}

/// @brief it should be called even if a standard exception occurs later in the scope.
TEST(DeferTests, ExceptionHandling) {
    bool called = false;
    try {
        x::defer d([&called] { called = true; });
        throw std::runtime_error("Test exception");
    } catch (const std::exception &) {
        // Exception caught
    }
    ASSERT_TRUE(called);
}

/// @brief it should call deferrals correctly within nested scopes.
TEST(DeferTests, NestedScopes) {
    int outer = 0;
    int inner = 0;

    {
        x::defer d_outer([&outer] { outer++; });
        {
            x::defer d_inner([&inner] { inner++; });
            ASSERT_EQ(inner, 0);
        }

        ASSERT_EQ(inner, 1);
        ASSERT_EQ(outer, 0);
    }

    ASSERT_EQ(inner, 1);
    ASSERT_EQ(outer, 1);
}

/// @brief it should be able to capture and modify variables in the outer scope.
TEST(DeferTests, ModifyingCapturedVariables) {
    int value = 5;
    {
        x::defer d([&value] { value *= 2; });
        value += 5;
        ASSERT_EQ(value, 10);
    }
    ASSERT_EQ(value, 20);
}

/// @brief it should be able to conditionally execute the deferred function.
TEST(DeferTests, ConditionalExecution) {
    bool condition = false;
    bool executed = false;

    {
        if (condition) x::defer d([&executed]() { executed = true; });
    }

    ASSERT_FALSE(executed);

    condition = true;
    {
        if (condition) x::defer d([&executed]() { executed = true; });
    }

    ASSERT_TRUE(executed);
}
