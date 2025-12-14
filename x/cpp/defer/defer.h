// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <iostream>

namespace x {
/**
 * @brief A utility class that executes a function when it goes out of scope.
 *
 * The defer class provides a mechanism similar to Go's defer statement, allowing
 * for cleanup code to be specified at the beginning of a scope but executed when
 * the scope is exited, regardless of how the scope is exited (normal execution,
 * return, or exception).
 *
 * Example usage:
 * ```
 * {
 *     auto resource = acquire_resource();
 *     x::defer d([&resource]() { release_resource(resource); });
 *
 *     // Use resource...
 *     // When scope ends, release_resource will be called automatically
 * }
 * ```
 *
 * Keep in mind that this function CANNOT be used to modify return values.
 */
// NOLINTBEGIN(readability-identifier-naming)
class defer {
    std::function<void()> fn;

public:
    /**
     * @brief Destructor that executes the deferred function.
     */
    ~defer() noexcept {
        try {
            fn();
        } catch (const std::exception &e) {
            std::cerr << "Exception in deferred function: " << e.what() << '\n';
        } catch (...) { std::cerr << "Unknown exception in deferred function\n"; }
    }

    /**
     * @brief Constructs a defer object with the function to be executed on
     * destruction.
     *
     * @param fn The function to execute when this object is destroyed.
     */
    [[nodiscard]] explicit defer(const std::function<void()> &fn): fn(fn) {}

    // Prevent copying to avoid multiple executions of the deferred function
    defer(const defer &) = delete;
    defer &operator=(const defer &) = delete;
};
// NOLINTEND(readability-identifier-naming)
}
