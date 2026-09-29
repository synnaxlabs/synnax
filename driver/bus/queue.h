// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <cstddef>
#include <deque>
#include <mutex>
#include <span>

#include "x/cpp/telem/telem.h"

namespace driver::bus {
/// @brief Batch is the result of one Queue::pop.
struct Batch {
    /// @brief count is the number of items written to the output.
    std::size_t count = 0;
    /// @brief dropped is the number of items the queue discarded since the last pop
    /// because it was full.
    std::size_t dropped = 0;
};

/// @brief Queue is a bounded queue that drops its oldest item when full, as a receive
/// FIFO on a bus card does. Safe for any number of producers and consumers.
template<typename T>
class Queue {
public:
    explicit Queue(const std::size_t capacity): capacity(capacity) {}

    /// @brief adds an item, dropping the oldest one when the queue is full.
    void push(T item) {
        {
            std::lock_guard lock(this->mu);
            if (this->items.size() == this->capacity) {
                this->items.pop_front();
                this->dropped++;
            }
            this->items.push_back(std::move(item));
        }
        this->cv.notify_one();
    }

    /// @brief moves up to out.size() items into out, waiting up to timeout for the
    /// first one.
    Batch pop(std::span<T> out, const x::telem::TimeSpan timeout) {
        std::unique_lock lock(this->mu);
        this->cv.wait_for(lock, timeout.chrono(), [this] {
            return !this->items.empty();
        });
        Batch b{.dropped = this->dropped};
        this->dropped = 0;
        while (b.count < out.size() && !this->items.empty()) {
            out[b.count++] = std::move(this->items.front());
            this->items.pop_front();
        }
        return b;
    }

private:
    /// @brief capacity is the most items the queue holds.
    const std::size_t capacity;
    /// @brief mu guards items and dropped.
    std::mutex mu;
    /// @brief cv wakes a waiting pop.
    std::condition_variable cv;
    /// @brief items holds the queued items, oldest first.
    std::deque<T> items;
    /// @brief dropped counts items discarded since the last pop.
    std::size_t dropped = 0;
};
}
