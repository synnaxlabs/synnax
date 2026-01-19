// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <utility>

namespace xmemory {

/// @brief A non-atomic reference-counted smart pointer for single-threaded use.
///
/// local_shared provides shared ownership semantics similar to std::shared_ptr,
/// but uses non-atomic reference counting for improved performance in
/// single-threaded contexts.
///
/// WARNING: This class is NOT thread-safe. Using local_shared across multiple
/// threads will result in undefined behavior. Only use when you can guarantee
/// single-threaded access.
///
/// Performance characteristics:
/// - Copy: O(1) with a single non-atomic increment
/// - Move: O(1) with no reference count modification
/// - Destruction: O(1) with a single non-atomic decrement (+ object destruction if
/// last)
///
/// @tparam T The type of object to manage
template<typename T>
class local_shared {
    struct ControlBlock {
        T value;
        size_t ref_count;

        template<typename... Args>
        explicit ControlBlock(Args &&...args):
            value(std::forward<Args>(args)...), ref_count(1) {}
    };

    ControlBlock *ptr_;

    void add_ref() {
        if (this->ptr_) { ++this->ptr_->ref_count; }
    }

    void release() {
        if (this->ptr_) {
            --this->ptr_->ref_count;
            if (this->ptr_->ref_count == 0) { delete this->ptr_; }
            this->ptr_ = nullptr;
        }
    }

public:
    /// @brief Default constructor - creates an empty local_shared
    local_shared(): ptr_(nullptr) {}

    /// @brief Move constructor from T - wraps an existing object by moving it
    explicit local_shared(T &&value): ptr_(new ControlBlock(std::move(value))) {}

    /// @brief Constructs a local_shared managing a new object constructed with args
    /// SFINAE constraint: Don't use this constructor if Args is a local_shared
    template<
        typename... Args,
        typename = std::enable_if_t<
            !std::is_same_v<
                std::decay_t<std::tuple_element_t<0, std::tuple<Args..., void>>>,
                local_shared<T>> ||
            sizeof...(Args) != 1>>
    explicit local_shared(Args &&...args):
        ptr_(new ControlBlock(std::forward<Args>(args)...)) {}

    /// @brief Copy constructor - shares ownership with another local_shared
    local_shared(const local_shared &other): ptr_(other.ptr_) { this->add_ref(); }

    /// @brief Move constructor - transfers ownership from another local_shared
    local_shared(local_shared &&other) noexcept: ptr_(other.ptr_) {
        other.ptr_ = nullptr;
    }

    /// @brief Copy assignment - shares ownership with another local_shared
    local_shared &operator=(const local_shared &other) {
        if (this != &other) {
            this->release();
            this->ptr_ = other.ptr_;
            this->add_ref();
        }
        return *this;
    }

    /// @brief Move assignment - transfers ownership from another local_shared
    local_shared &operator=(local_shared &&other) noexcept {
        if (this != &other) {
            this->release();
            this->ptr_ = other.ptr_;
            other.ptr_ = nullptr;
        }
        return *this;
    }

    /// @brief Destructor - releases ownership
    ~local_shared() { this->release(); }

    /// @brief Returns a pointer to the managed object
    T *get() const { return this->ptr_ ? &this->ptr_->value : nullptr; }

    /// @brief Dereferences pointer to the managed object
    T &operator*() const { return this->ptr_->value; }

    /// @brief Dereferences pointer to the managed object
    T *operator->() const { return &this->ptr_->value; }

    /// @brief Checks whether the local_shared manages an object
    explicit operator bool() const { return this->ptr_ != nullptr; }

    /// @brief Returns the current reference count (for debugging/testing)
    size_t use_count() const { return this->ptr_ ? this->ptr_->ref_count : 0; }

    /// @brief Resets the local_shared to empty, releasing ownership
    void reset() { this->release(); }

    /// @brief Swaps the managed object with another local_shared
    void swap(local_shared &other) noexcept { std::swap(this->ptr_, other.ptr_); }

    /// @brief Equality comparison
    bool operator==(const local_shared &other) const {
        return this->ptr_ == other.ptr_;
    }

    /// @brief Inequality comparison
    bool operator!=(const local_shared &other) const {
        return this->ptr_ != other.ptr_;
    }

    /// @brief Null pointer comparison
    bool operator==(std::nullptr_t) const { return this->ptr_ == nullptr; }

    /// @brief Null pointer comparison
    bool operator!=(std::nullptr_t) const { return this->ptr_ != nullptr; }
};

/// @brief Constructs a local_shared managing a new object of type T
///
/// @tparam T The type of object to create
/// @tparam Args The types of arguments to pass to T's constructor
/// @param args Arguments to forward to T's constructor
/// @return A local_shared managing the newly created object
template<typename T, typename... Args>
local_shared<T> make_local_shared(Args &&...args) {
    return local_shared<T>(std::forward<Args>(args)...);
}

}
