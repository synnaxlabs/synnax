// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// GTest
#include <chrono>
#include <thread>
#include <functional>

/// external
#include "gtest/gtest.h"

/// @brief xtest is a testing utility package that extends Google Test with eventual assertion capabilities.
/// These assertions are particularly useful for testing asynchronous operations or conditions that may
/// take time to become true.
///
/// The package provides three main types of eventual assertions:
/// - ASSERT_EVENTUALLY_EQ: Asserts that two values will eventually become equal
/// - ASSERT_EVENTUALLY_LE: Asserts that one value will eventually become less than or equal to another
/// - ASSERT_EVENTUALLY_GE: Asserts that one value will eventually become greater than or equal to another
///
/// Each assertion has two variants:
/// 1. Basic variant: Uses default timeout (1 second) and interval (1 millisecond)
/// 2. Extended variant (_WITH_TIMEOUT): Allows custom timeout and interval values
///
/// Example usage:
/// @code
///     // Basic usage
///     ASSERT_EVENTUALLY_EQ(slow_counter.get_value(), 10);
///
///     // With custom timeout and interval
///     ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
///         slow_counter.get_value(), 
///         10, 
///         std::chrono::seconds(5),
///         std::chrono::milliseconds(100)
///     );
/// @endcode
namespace xtest {

/// @brief Core comparison function that implements the eventual assertion logic
/// @tparam T The type of values being compared
/// @param actual A function that returns the actual value to be compared
/// @param expected The expected value to compare against
/// @param comparator The comparison function to use (e.g., ==, <=, >=)
/// @param op_name The name of the operation for error messages (e.g., "EQ", "LE", "GE")
/// @param op_sep The operator symbol for error messages (e.g., "==", "<=", ">=")
/// @param timeout Maximum time to wait for the condition to become true (default: 1 second)
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @throws Testing::AssertionFailure if the condition is not met within the timeout period
template<typename T>
void eventually_compare(
    const std::function<T()>& actual,
    const T& expected,
    const std::function<bool(const T&, const T&)>& comparator,
    const std::string& op_name,
    const std::string &op_sep,
    const std::chrono::milliseconds timeout = std::chrono::seconds(1),
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1)
) {
    auto start = std::chrono::steady_clock::now();
    while (true) {
        if (comparator(actual(), expected)) return;
        
        auto now = std::chrono::steady_clock::now();
        if (now - start >= timeout) {
            FAIL() << "EVENTUALLY_" << op_name << " timed out after " <<
                std::chrono::duration_cast<std::chrono::milliseconds>(timeout).count() << 
                "ms. Expected \n" << expected << " \n" << op_sep << " \n" << actual();
        }
        std::this_thread::sleep_for(interval);
    }
}

/// @brief Asserts that two values will eventually become equal
/// @tparam T The type of values being compared
/// @param actual A function that returns the actual value to be compared
/// @param expected The expected value to compare against
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @param timeout Maximum time to wait for equality (default: 1 second)
template<typename T>
void eventually_eq(
    const std::function<T()>& actual,
    const T& expected,
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1),
    const std::chrono::milliseconds timeout = std::chrono::seconds(1)
) {
    eventually_compare<T>(
        actual, 
        expected,
        [](const T& a, const T& b) { return a == b; },
        "EQ",
        "==",
        timeout,
        interval
    );
}

/// @brief Asserts that one value will eventually become less than or equal to another
/// @tparam T The type of values being compared
/// @param actual A function that returns the actual value to be compared
/// @param expected The expected value to compare against
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @param timeout Maximum time to wait for the condition (default: 1 second)
template<typename T>
void eventually_le(
    const std::function<T()>& actual,
    const T& expected,
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1),
    const std::chrono::milliseconds timeout = std::chrono::seconds(1)
) {
    eventually_compare<T>(
        actual, 
        expected,
        [](const T& a, const T& b) { return a <= b; },
        "LE",
            "<=",
        timeout,
        interval
    );
}

/// @brief Asserts that one value will eventually become greater than or equal to another
/// @tparam T The type of values being compared
/// @param actual A function that returns the actual value to be compared
/// @param expected The expected value to compare against
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @param timeout Maximum time to wait for the condition (default: 1 second)
template<typename T>
void eventually_ge(
    const std::function<T()>& actual,
    const T& expected,
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1),
    const std::chrono::milliseconds timeout = std::chrono::seconds(1)
) {
    eventually_compare<T>(
        actual, 
        expected,
        [](const T& a, const T& b) { return a >= b; },
        "GE",
        ">=",
        timeout,
        interval
    );
}

/// @brief Macro for asserting eventual equality with default timeout and interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_EQ(actual, expected) \
    xtest::eventually_eq<decltype(actual)>([&]() { return (actual); }, (expected))

/// @brief Macro for asserting eventual equality with custom timeout and interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for equality
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(actual, expected, timeout, interval) \
    xtest::eventually_eq<decltype(actual)>([&]() { return (actual); }, (expected), (interval), (timeout))

/// @brief Macro for asserting eventual less than or equal with default timeout and interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_LE(actual, expected) \
    xtest::eventually_le<decltype(actual)>([&]() { return (actual); }, (expected))

/// @brief Macro for asserting eventual less than or equal with custom timeout and interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_LE_WITH_TIMEOUT(actual, expected, timeout, interval) \
    xtest::eventually_le<decltype(actual)>([&]() { return (actual); }, (expected), (interval), (timeout))

/// @brief Macro for asserting eventual greater than or equal with default timeout and interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_GE(actual, expected) \
    xtest::eventually_ge<decltype(actual)>([&]() { return (actual); }, (expected))

/// @brief Macro for asserting eventual greater than or equal with custom timeout and interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(actual, expected, timeout, interval) \
    xtest::eventually_ge<decltype(actual)>([&]() { return (actual); }, (expected), (interval), (timeout))

/// @brief Macro for asserting eventual equality with default timeout and interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_EQ_F(fn, expected) \
    xtest::eventually_eq<decltype((fn)())>(std::function<decltype((fn)())()>(fn), (expected))

/// @brief Macro for asserting eventual equality with custom timeout and interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for equality
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_EQ_F_WITH_TIMEOUT(fn, expected, timeout, interval) \
    xtest::eventually_eq<decltype((fn)())>(std::function<decltype((fn)())()>(fn), (expected), (interval), (timeout))

/// @brief Macro for asserting eventual less than or equal with default timeout and interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_LE_F(fn, expected) \
    xtest::eventually_le<decltype((fn)())>(std::function<decltype((fn)())()>(fn), (expected))

/// @brief Macro for asserting eventual less than or equal with custom timeout and interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_LE_F_WITH_TIMEOUT(fn, expected, timeout, interval) \
    xtest::eventually_le<decltype((fn)())>(std::function<decltype((fn)())()>(fn), (expected), (interval), (timeout))

/// @brief Macro for asserting eventual greater than or equal with default timeout and interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_GE_F(fn, expected) \
    xtest::eventually_ge<decltype((fn)())>(std::function<decltype((fn)())()>(fn), (expected))

/// @brief Macro for asserting eventual greater than or equal with custom timeout and interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_GE_F_WITH_TIMEOUT(fn, expected, timeout, interval) \
    xtest::eventually_ge<decltype((fn)())>(std::function<decltype((fn)())()>(fn), (expected), (interval), (timeout))

} // namespace xtest




