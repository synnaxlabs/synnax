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
#include <deque>
#include <mutex>
#include <ranges>
#include <utility>
#include <vector>

namespace x::sync {
/// @brief an append-only list that one thread can grow while others read it. All
/// methods are thread-safe. A reference to an item stays valid until clear().
template<typename T>
class List {
    /// @brief guards items.
    mutable std::mutex mu;
    /// @brief the items. A deque keeps references stable across push_back.
    std::deque<T> items;

public:
    /// @brief appends item.
    void push_back(T item) {
        std::lock_guard lock(this->mu);
        this->items.push_back(std::move(item));
    }

    /// @returns the number of items.
    [[nodiscard]] size_t size() const {
        std::lock_guard lock(this->mu);
        return this->items.size();
    }

    /// @returns true if the list has no items.
    [[nodiscard]] bool empty() const { return this->size() == 0; }

    /// @returns the item at index i.
    /// @throws std::out_of_range if i is not less than size().
    [[nodiscard]] const T &at(const size_t i) const {
        std::lock_guard lock(this->mu);
        return this->items.at(i);
    }

    /// @brief removes every item. Must not run while another thread holds a
    /// reference from at() or snapshot().
    void clear() {
        std::lock_guard lock(this->mu);
        this->items.clear();
    }

    /// @returns a range over the items present at the time of the call, in order.
    /// Items appended later are not in the range.
    [[nodiscard]] auto snapshot() const {
        std::vector<const T *> ptrs;
        {
            std::lock_guard lock(this->mu);
            ptrs.reserve(this->items.size());
            for (const auto &item: this->items)
                ptrs.push_back(&item);
        }
        return std::move(ptrs) |
               std::views::transform([](const T *p) -> const T & { return *p; });
    }
};
}
