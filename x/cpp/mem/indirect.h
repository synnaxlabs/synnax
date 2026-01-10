// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <utility>

namespace x::mem {

/// @brief A value-semantic wrapper for heap-allocated objects.
/// Similar to std::optional but works with incomplete types (forward declarations).
/// Provides deep copy semantics - copying an indirect copies the underlying value.
/// Mirrors the C++26 std::indirect proposal (P1950).
///
/// Use this for:
/// - Self-referential struct fields (e.g., tree nodes)
/// - Optional fields where the type is incomplete at point of declaration
///
/// @tparam T The type to store indirectly
template<typename T>
class indirect {
    std::unique_ptr<T> ptr;

public:
    /// @brief Default constructor - creates empty indirect (no value)
    indirect() = default;

    /// @brief Nullptr constructor - creates empty indirect
    indirect(std::nullptr_t): ptr(nullptr) {}

    /// @brief Value constructor - takes ownership of value
    explicit indirect(T value): ptr(std::make_unique<T>(std::move(value))) {}

    /// @brief Copy constructor - deep copies the value if present
    indirect(const indirect &other):
        ptr(other.ptr ? std::make_unique<T>(*other.ptr) : nullptr) {}

    /// @brief Move constructor
    indirect(indirect &&) noexcept = default;

    /// @brief Copy assignment - deep copies the value if present
    indirect &operator=(const indirect &other) {
        if (this != &other) {
            ptr = other.ptr ? std::make_unique<T>(*other.ptr) : nullptr;
        }
        return *this;
    }

    /// @brief Move assignment
    indirect &operator=(indirect &&) noexcept = default;

    /// @brief Nullptr assignment - clears the value
    indirect &operator=(std::nullptr_t) {
        ptr = nullptr;
        return *this;
    }

    /// @brief Value assignment
    indirect &operator=(T value) {
        ptr = std::make_unique<T>(std::move(value));
        return *this;
    }

    /// @brief Check if value is present
    [[nodiscard]] explicit operator bool() const noexcept { return ptr != nullptr; }

    /// @brief Check if value is present
    [[nodiscard]] bool has_value() const noexcept { return ptr != nullptr; }

    /// @brief Arrow operator for member access
    [[nodiscard]] T *operator->() noexcept { return ptr.get(); }
    [[nodiscard]] const T *operator->() const noexcept { return ptr.get(); }

    /// @brief Dereference operator
    [[nodiscard]] T &operator*() & noexcept { return *ptr; }
    [[nodiscard]] const T &operator*() const & noexcept { return *ptr; }
    [[nodiscard]] T &&operator*() && noexcept { return std::move(*ptr); }

    /// @brief Get raw pointer
    [[nodiscard]] T *get() noexcept { return ptr.get(); }
    [[nodiscard]] const T *get() const noexcept { return ptr.get(); }

    /// @brief Get value or default
    [[nodiscard]] T value_or(T default_value) const {
        return ptr ? *ptr : std::move(default_value);
    }

    /// @brief Reset to empty state
    void reset() noexcept { ptr.reset(); }

    /// @brief Swap with another indirect
    void swap(indirect &other) noexcept { ptr.swap(other.ptr); }

    /// @brief Equality comparison
    [[nodiscard]] bool operator==(std::nullptr_t) const noexcept { return !ptr; }
    [[nodiscard]] bool operator!=(std::nullptr_t) const noexcept {
        return ptr != nullptr;
    }
};

/// @brief Factory function for in-place construction
template<typename T, typename... Args>
[[nodiscard]] indirect<T> make_indirect(Args &&...args) {
    indirect<T> result;
    result = T(std::forward<Args>(args)...);
    return result;
}

} // namespace x::mem
