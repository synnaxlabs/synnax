// Copyright 2026 Synnax Labs, Inc.
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
#include <functional>
#include <sstream>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"

/// @brief xtest is a testing utility package that extends Google Test with eventual
/// assertion capabilities. These assertions are particularly useful for testing
/// asynchronous operations or conditions that may take time to become true.
///
/// The package provides three main types of eventual assertions:
/// - ASSERT_EVENTUALLY_EQ: Asserts that two values will eventually become equal
/// - ASSERT_EVENTUALLY_LE: Asserts that one value will eventually become less than or
/// equal to another
/// - ASSERT_EVENTUALLY_GE: Asserts that one value will eventually become greater than
/// or equal to another
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
namespace x::test {

/// @brief Core function that implements the eventual assertion logic with a generic
/// condition
/// @param condition A function that returns true when the assertion should pass
/// @param failure_message A function that returns the error message to display on
/// timeout
/// @param file The source file name (for proper stack traces)
/// @param line The source line number (for proper stack traces)
/// @param timeout Maximum time to wait for the condition to become true (default: 1
/// second)
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @throws Testing::AssertionFailure if the condition is not met within the timeout
/// period
inline void eventually(
    const std::function<bool()> &condition,
    const std::function<std::string()> &failure_message,
    const char *file,
    const int line,
    const std::chrono::milliseconds timeout = std::chrono::seconds(1),
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1)
) {
    const auto start = std::chrono::steady_clock::now();
    while (true) {
        if (condition()) return;

        auto now = std::chrono::steady_clock::now();
        if (now - start >= timeout) {
            ADD_FAILURE_AT(file, line) << failure_message();
            return;
        }
        std::this_thread::sleep_for(interval);
    }
}

/// @brief Core comparison function that implements the eventual assertion logic
/// @tparam T The type of values being compared
/// @param actual A function that returns the actual value to be compared
/// @param expected The expected value to compare against
/// @param comparator The comparison function to use (e.g., ==, <=, >=)
/// @param op_name The name of the operation for error messages (e.g., "EQ", "LE",
/// "GE")
/// @param op_sep The operator symbol for error messages (e.g., "==", "<=", ">=")
/// @param file The source file name (for proper stack traces)
/// @param line The source line number (for proper stack traces)
/// @param timeout Maximum time to wait for the condition to become true (default: 1
/// second)
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @throws Testing::AssertionFailure if the condition is not met within the timeout
/// period
template<typename T>
void eventually_compare(
    const std::function<T()> &actual,
    const T &expected,
    const std::function<bool(const T &, const T &)> &comparator,
    const std::string &op_name,
    const std::string &op_sep,
    const char *file,
    const int line,
    const std::chrono::milliseconds timeout = std::chrono::seconds(1),
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1)
) {
    T last_actual_value;
    eventually(
        [&]() {
            last_actual_value = actual();
            return comparator(last_actual_value, expected);
        },
        [&]() {
            std::stringstream ss;
            ss << "EVENTUALLY_" << op_name << " timed out after "
               << std::chrono::duration_cast<std::chrono::milliseconds>(timeout).count()
               << "ms. Expected \n"
               << expected << " \n"
               << op_sep << " \n"
               << last_actual_value;
            return ss.str();
        },
        file,
        line,
        timeout,
        interval
    );
}

/// @brief Asserts that two values will eventually become equal
/// @tparam T The type of values being compared
/// @param actual A function that returns the actual value to be compared
/// @param expected The expected value to compare against
/// @param file The source file name (for proper stack traces)
/// @param line The source line number (for proper stack traces)
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @param timeout Maximum time to wait for equality (default: 1 second)
template<typename T>
void eventually_eq(
    const std::function<T()> &actual,
    const T &expected,
    const char *file,
    const int line,
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1),
    const std::chrono::milliseconds timeout = std::chrono::seconds(1)
) {
    eventually_compare<T>(
        actual,
        expected,
        [](const T &a, const T &b) { return a == b; },
        "EQ",
        "==",
        file,
        line,
        timeout,
        interval
    );
}

/// @brief Asserts that one value will eventually become less than or equal to
/// another
/// @tparam T The type of values being compared
/// @param actual A function that returns the actual value to be compared
/// @param expected The expected value to compare against
/// @param file The source file name (for proper stack traces)
/// @param line The source line number (for proper stack traces)
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @param timeout Maximum time to wait for the condition (default: 1 second)
template<typename T>
void eventually_le(
    const std::function<T()> &actual,
    const T &expected,
    const char *file,
    const int line,
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1),
    const std::chrono::milliseconds timeout = std::chrono::seconds(1)
) {
    eventually_compare<T>(
        actual,
        expected,
        [](const T &a, const T &b) { return a <= b; },
        "LE",
        "<=",
        file,
        line,
        timeout,
        interval
    );
}

/// @brief Asserts that one value will eventually become greater than or equal to
/// another
/// @tparam T The type of values being compared
/// @param actual A function that returns the actual value to be compared
/// @param expected The expected value to compare against
/// @param file The source file name (for proper stack traces)
/// @param line The source line number (for proper stack traces)
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @param timeout Maximum time to wait for the condition (default: 1 second)
template<typename T>
void eventually_ge(
    const std::function<T()> &actual,
    const T &expected,
    const char *file,
    const int line,
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1),
    const std::chrono::milliseconds timeout = std::chrono::seconds(1)
) {
    eventually_compare<T>(
        actual,
        expected,
        [](const T &a, const T &b) { return a >= b; },
        "GE",
        ">=",
        file,
        line,
        timeout,
        interval
    );
}

inline void eventually_nil(
    const std::function<errors::Error()> &actual,
    const char *file,
    const int line,
    const std::chrono::milliseconds timeout = std::chrono::seconds(1),
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1)
) {
    errors::Error last_error;
    eventually(
        [&]() {
            last_error = actual();
            return !last_error;
        },
        [&]() {
            std::stringstream ss;
            ss << "EVENTUALLY_NIL timed out after "
               << std::chrono::duration_cast<std::chrono::milliseconds>(timeout).count()
               << "ms. Expected NIL, but got " << last_error;
            return ss.str();
        },
        file,
        line,
        timeout,
        interval
    );
}

/// @brief macro for asserting eventual equality with default timeout and interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_EQ(actual, expected)                                         \
    x::test::eventually_eq<decltype(actual)>(                                          \
        [&]() { return (actual); },                                                    \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__                                                                       \
    )

/// @brief macro for asserting eventual equality with custom timeout and interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for equality
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(actual, expected, timeout, interval)         \
    x::test::eventually_eq<decltype(actual)>(                                          \
        [&]() { return (actual); },                                                    \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__,                                                                      \
        (interval),                                                                    \
        (timeout)                                                                      \
    )

/// @brief macro for asserting eventual less than or equal with default timeout and
/// interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_LE(actual, expected)                                         \
    x::test::eventually_le<decltype(actual)>(                                          \
        [&]() { return (actual); },                                                    \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__                                                                       \
    )

/// @brief macro for asserting eventual less than or equal with custom timeout and
/// interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_LE_WITH_TIMEOUT(actual, expected, timeout, interval)         \
    x::test::eventually_le<decltype(actual)>(                                          \
        [&]() { return (actual); },                                                    \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__,                                                                      \
        (interval),                                                                    \
        (timeout)                                                                      \
    )

/// @brief macro for asserting eventual greater than or equal with default timeout and
/// interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_GE(actual, expected)                                         \
    x::test::eventually_ge<decltype(actual)>(                                          \
        [&]() { return (actual); },                                                    \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__                                                                       \
    )

/// @brief macro for asserting eventual greater than or equal with custom timeout and
/// interval
/// @param actual The actual value or expression to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(actual, expected, timeout, interval)         \
    x::test::eventually_ge<decltype(actual)>(                                          \
        [&]() { return (actual); },                                                    \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__,                                                                      \
        (interval),                                                                    \
        (timeout)                                                                      \
    )

/// @brief macro for asserting eventual equality with default timeout and interval using
/// a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_EQ_F(fn, expected)                                           \
    x::test::eventually_eq<decltype((fn) ())>(                                         \
        std::function<decltype((fn) ())()>(fn),                                        \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__                                                                       \
    )

/// @brief macro for asserting eventual equality with custom timeout and interval using
/// a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for equality
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_EQ_F_WITH_TIMEOUT(fn, expected, timeout, interval)           \
    x::test::eventually_eq<decltype((fn) ())>(                                         \
        std::function<decltype((fn) ())()>(fn),                                        \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__,                                                                      \
        (interval),                                                                    \
        (timeout)                                                                      \
    )

/// @brief macro for asserting eventual less than or equal with default timeout and
/// interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_LE_F(fn, expected)                                           \
    x::test::eventually_le<decltype((fn) ())>(                                         \
        std::function<decltype((fn) ())()>(fn),                                        \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__                                                                       \
    )

/// @brief macro for asserting eventual less than or equal with custom timeout and
/// interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_LE_F_WITH_TIMEOUT(fn, expected, timeout, interval)           \
    x::test::eventually_le<decltype((fn) ())>(                                         \
        std::function<decltype((fn) ())()>(fn),                                        \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__,                                                                      \
        (interval),                                                                    \
        (timeout)                                                                      \
    )

/// @brief macro for asserting eventual greater than or equal with default timeout and
/// interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
#define ASSERT_EVENTUALLY_GE_F(fn, expected)                                           \
    x::test::eventually_ge<decltype((fn) ())>(                                         \
        std::function<decltype((fn) ())()>(fn),                                        \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__                                                                       \
    )

/// @brief macro for asserting eventual greater than or equal with custom timeout and
/// interval using a function
/// @param fn The function to evaluate
/// @param expected The expected value to compare against
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_GE_F_WITH_TIMEOUT(fn, expected, timeout, interval)           \
    x::test::eventually_ge<decltype((fn) ())>(                                         \
        std::function<decltype((fn) ())()>(fn),                                        \
        (expected),                                                                    \
        __FILE__,                                                                      \
        __LINE__,                                                                      \
        (interval),                                                                    \
        (timeout)                                                                      \
    )

/// @brief Helper function for ASSERT_NIL_P macro that works with MSVC
/// @tparam Pair The pair type (automatically deduced)
/// @param pair_result The pair to check
/// @param file The source file name
/// @param line The source line number
/// @return The first element of the pair (the result value) if successful
template<typename Pair>
auto assert_nil_p(Pair &&pair_result, const char *file, const int line) ->
    typename std::remove_reference<decltype(pair_result.first)>::type {
    if (pair_result.second) {
        ADD_FAILURE_AT(file, line)
            << "Expected operation to succeed, but got error: " << pair_result.second;
    }
    return std::move(pair_result.first);
}

/// @brief macro for asserting that an operation returning a pair<T, errors::Error>
/// succeeded and returning the result value
/// @param pair_expr The expression returning the pair to evaluate
/// @return The first element of the pair (the result value) if successful
#define ASSERT_NIL_P(pair_expr) x::test::assert_nil_p((pair_expr), __FILE__, __LINE__)

/// @brief macro asserting that the provided errors::Error is NIL.
#define ASSERT_NIL(expr) ASSERT_FALSE(expr) << expr;

/// @brief macro asserting that the provided errors::Error is the same as the provided
/// error.
#define ASSERT_OCCURRED_AS(expr, err)                                                  \
    ASSERT_TRUE(expr) << expr;                                                         \
    ASSERT_MATCHES(expr, err);

/// @brief macro asserting that the error return as the second item in the pair is the
/// same as the provided error.
#define ASSERT_OCCURRED_AS_P(expr, err)                                                \
    ASSERT_TRUE(expr.second) << expr.second;                                           \
    ASSERT_MATCHES(expr.second, err);

/// @brief macro asserting that the provided error matches the expeced err via a call
/// to err.matches(expect).
#define ASSERT_MATCHES(err, expected)                                                  \
    ASSERT_TRUE(err.matches(expected))                                                 \
        << "Expected error to match " << expected << ", but got " << err;

/// @brief macro asserting that the provided error will eventually be NIL.
/// @param expr The expression to evaluate
#define ASSERT_EVENTUALLY_NIL(expr)                                                    \
    x::test::eventually_nil([&]() { return (expr); }, __FILE__, __LINE__)

/// @brief Asserts that a pair's error component will eventually become nil and
/// returns the value component
/// @tparam T The type of the value component in the pair
/// @tparam DurationTimeout Type of the timeout duration
/// @tparam DurationInterval Type of the interval duration
/// @param actual A function that returns the pair to be checked
/// @param file The source file name (for proper stack traces)
/// @param line The source line number (for proper stack traces)
/// @param timeout Maximum time to wait for the error to become nil (default: 1
/// second)
/// @param interval Time to wait between checks (default: 1 millisecond)
/// @return The value component of the pair once the error becomes nil
/// @throws Testing::AssertionFailure if the error does not become nil within the
/// timeout period
template<
    typename T,
    typename DurationTimeout = std::chrono::milliseconds,
    typename DurationInterval = std::chrono::milliseconds>
T eventually_nil_p(
    const std::function<std::pair<T, errors::Error>()> &actual,
    const char *file,
    const int line,
    const DurationTimeout &timeout = std::chrono::seconds(1),
    const DurationInterval &interval = std::chrono::milliseconds(1)
) {
    std::pair<T, errors::Error> result;

    // Convert timeout and interval to std::chrono::milliseconds
    const auto timeout_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
        timeout
    );
    const auto interval_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
        interval
    );

    try {
        eventually(
            [&]() {
                result = actual();
                return !result.second;
            },
            [&]() {
                std::stringstream ss;
                ss << "EVENTUALLY_NIL_P timed out after "
                   << std::chrono::duration_cast<std::chrono::milliseconds>(timeout_ms)
                          .count()
                   << "ms. Expected NIL, but got " << result.second;
                return ss.str();
            },
            file,
            line,
            timeout_ms,
            interval_ms
        );
    } catch (const ::testing::AssertionException &) {
        // Return the value part even though the error isn't nil
        // This allows the test to continue and potentially diagnose other issues
        return std::move(result.first);
    }

    return std::move(result.first);
}

/// @brief macro for asserting that a pair's error component will eventually become nil
/// with default timeout and interval
/// @param expr The expression returning the pair to evaluate
/// @return The value component of the pair once the error becomes nil
#define ASSERT_EVENTUALLY_NIL_P(expr)                                                  \
    x::test::eventually_nil_p<                                                         \
        typename std::remove_reference<decltype((expr).first)>::type>(                 \
        [&]() { return (expr); },                                                      \
        __FILE__,                                                                      \
        __LINE__                                                                       \
    )

/// @brief macro for asserting that a pair's error component will eventually become nil
/// with custom timeout and interval
/// @param expr The expression returning the pair to evaluate
/// @param timeout Maximum time to wait for the error to become nil
/// @param interval Time to wait between checks
/// @return The value component of the pair once the error becomes nil
#define ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(expr, timeout, interval)                  \
    x::test::eventually_nil_p<                                                         \
        typename std::remove_reference<decltype((expr).first)>::type>(                 \
        [&]() { return (expr); },                                                      \
        __FILE__,                                                                      \
        __LINE__,                                                                      \
        (timeout),                                                                     \
        (interval)                                                                     \
    )

/// @brief Asserts that a boolean condition will eventually become false
/// @param condition A function that returns the boolean condition to check
/// @param file The source file name (for proper stack traces)
/// @param line The source line number (for proper stack traces)
/// @param timeout Maximum time to wait for the condition (default: 1 second)
/// @param interval Time to wait between checks (default: 1 millisecond)
inline void eventually_false(
    const std::function<bool()> &condition,
    const char *file,
    const int line,
    const std::chrono::milliseconds timeout = std::chrono::seconds(1),
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1)
) {
    bool last_value;
    eventually(
        [&]() {
            last_value = condition();
            return !last_value;
        },
        [&]() {
            std::stringstream ss;
            ss << "EVENTUALLY_FALSE timed out after "
               << std::chrono::duration_cast<std::chrono::milliseconds>(timeout).count()
               << "ms. Expected FALSE, but got TRUE";
            return ss.str();
        },
        file,
        line,
        timeout,
        interval
    );
}

/// @brief Asserts that a boolean condition will eventually become true
/// @param condition A function that returns the boolean condition to check
/// @param file The source file name (for proper stack traces)
/// @param line The source line number (for proper stack traces)
/// @param timeout Maximum time to wait for the condition (default: 1 second)
/// @param interval Time to wait between checks (default: 1 millisecond)
inline void eventually_true(
    const std::function<bool()> &condition,
    const char *file,
    const int line,
    const std::chrono::milliseconds timeout = std::chrono::seconds(1),
    const std::chrono::milliseconds interval = std::chrono::milliseconds(1)
) {
    bool last_value;
    eventually(
        [&]() {
            last_value = condition();
            return last_value;
        },
        [&]() {
            std::stringstream ss;
            ss << "EVENTUALLY_TRUE timed out after "
               << std::chrono::duration_cast<std::chrono::milliseconds>(timeout).count()
               << "ms. Expected TRUE, but got FALSE";
            return ss.str();
        },
        file,
        line,
        timeout,
        interval
    );
}

/// @brief macro for asserting that a condition will eventually become false
/// @param expr The expression to evaluate
#define ASSERT_EVENTUALLY_FALSE(expr)                                                  \
    x::test::eventually_false([&]() { return (expr); }, __FILE__, __LINE__)

/// @brief macro for asserting that a condition will eventually become false with custom
/// timeout and interval
/// @param expr The expression to evaluate
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_FALSE_WITH_TIMEOUT(expr, timeout, interval)                  \
    x::test::eventually_false(                                                         \
        [&]() { return (expr); },                                                      \
        __FILE__,                                                                      \
        __LINE__,                                                                      \
        (timeout),                                                                     \
        (interval)                                                                     \
    )

/// @brief macro for asserting that a condition will eventually become true
/// @param expr The expression to evaluate
#define ASSERT_EVENTUALLY_TRUE(expr)                                                   \
    x::test::eventually_true([&]() { return (expr); }, __FILE__, __LINE__)

/// @brief macro for asserting that a condition will eventually become true with custom
/// timeout and interval
/// @param expr The expression to evaluate
/// @param timeout Maximum time to wait for the condition
/// @param interval Time to wait between checks
#define ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(expr, timeout, interval)                   \
    x::test::eventually_true(                                                          \
        [&]() { return (expr); },                                                      \
        __FILE__,                                                                      \
        __LINE__,                                                                      \
        (timeout),                                                                     \
        (interval)                                                                     \
    )
}
