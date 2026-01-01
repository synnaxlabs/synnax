// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <condition_variable>
#include <mutex>
#include <queue>

namespace queue {
template<typename T>
class SPSC {
public:
    /// @brief Pushes a value onto the queue. Returns false if the queue is closed.
    bool push(T value) {
        {
            std::lock_guard lock(mtx_);
            if (closed_.load(std::memory_order_acquire)) return false;
            q_.push(std::move(value));
        }
        cv_.notify_one();
        return true;
    }

    /// @brief Pops a value from the queue. Blocks until a value is available or
    /// the queue is closed. Returns false if the queue is closed and empty.
    bool pop(T &value) {
        std::unique_lock lock(mtx_);
        cv_.wait(lock, [&] {
            return !q_.empty() || closed_.load(std::memory_order_acquire);
        });
        if (q_.empty()) return false;
        value = std::move(q_.front());
        q_.pop();
        return true;
    }

    bool try_pop(T &value) {
        std::lock_guard lock(mtx_);
        if (q_.empty()) return false;
        value = std::move(q_.front());
        q_.pop();
        return true;
    }

    bool empty() const {
        std::lock_guard lock(mtx_);
        return q_.empty();
    }

    /// @brief Closes the queue. Any blocked pop() calls will return false.
    /// Subsequent push() calls will return false.
    void close() {
        closed_.store(true, std::memory_order_release);
        cv_.notify_all();
    }

    bool closed() const { return closed_.load(std::memory_order_acquire); }

private:
    mutable std::mutex mtx_;
    std::queue<T> q_;
    std::condition_variable cv_;
    std::atomic<bool> closed_{false};
};
};
