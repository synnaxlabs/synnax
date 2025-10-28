// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <cstddef>
#include <utility>
#include <vector>

namespace queue {

/// @brief Lock-free single-producer single-consumer (SPSC) ring buffer queue.
/// @tparam T Type of elements stored in the queue.
///
/// This queue is designed for real-time thread boundaries:
/// - Lock-free: No mutexes, no priority inversion
/// - Wait-free: Both push() and pop() are O(1)
/// - Cache-line aligned: Prevents false sharing between producer/consumer
/// - Move semantics: Efficient ownership transfer
/// - Fixed capacity: Set at construction, never grows (RT-safe)
///
/// Usage:
///   queue::SPSC<Message> q(1024);  // 1024 element capacity
///   // Producer thread:
///   q.push(std::move(msg));
///   // Consumer thread:
///   Message msg;
///   if (q.pop(msg)) { /* process msg */ }
template <typename T>
class SPSC {
    // Buffer storage (fixed at construction, never grows)
    std::vector<T> buffer_;
    size_t capacity_;

    // Producer-side atomics (cache-line aligned to prevent false sharing)
    alignas(64) std::atomic<size_t> write_pos_{0};

    // Consumer-side atomics (separate cache line)
    alignas(64) std::atomic<size_t> read_pos_{0};

public:
    /// @brief Construct queue with fixed capacity.
    /// @param capacity Maximum number of elements (actual capacity is capacity-1).
    /// @note This is the only allocation - buffer never grows after construction.
    explicit SPSC(size_t capacity)
        : buffer_(capacity), capacity_(capacity) {}

    ~SPSC() = default;

    // Non-copyable, non-movable (queue is tied to its memory location)
    SPSC(const SPSC &) = delete;
    SPSC &operator=(const SPSC &) = delete;
    SPSC(SPSC &&) = delete;
    SPSC &operator=(SPSC &&) = delete;

    /// @brief Push an element onto the queue (producer side).
    /// @param value Element to push (moved into queue).
    /// @return true if successful, false if queue is full.
    /// @note Called by producer thread only. RT-safe (no allocations, no locks).
    bool push(T &&value) {
        const size_t write = write_pos_.load(std::memory_order_relaxed);
        const size_t next = (write + 1) % capacity_;

        // Check if queue is full
        if (next == read_pos_.load(std::memory_order_acquire))
            return false;

        // Move element into buffer
        buffer_[write] = std::move(value);

        // Publish write (release semantics ensures visibility to consumer)
        write_pos_.store(next, std::memory_order_release);
        return true;
    }

    /// @brief Pop an element from the queue (consumer side).
    /// @param value Output parameter to receive the popped element.
    /// @return true if successful, false if queue is empty.
    /// @note Called by consumer thread only. RT-safe (no allocations, no locks).
    bool pop(T &value) {
        const size_t read = read_pos_.load(std::memory_order_relaxed);

        // Check if queue is empty
        if (read == write_pos_.load(std::memory_order_acquire))
            return false;

        // Move element out of buffer
        value = std::move(buffer_[read]);

        // Publish read (release semantics ensures visibility to producer)
        read_pos_.store((read + 1) % capacity_, std::memory_order_release);
        return true;
    }

    /// @brief Get the current number of elements in the queue.
    /// @return Approximate size (may be stale due to concurrent access).
    /// @note This is an estimate and should not be used for synchronization.
    size_t size() const {
        const size_t write = write_pos_.load(std::memory_order_acquire);
        const size_t read = read_pos_.load(std::memory_order_acquire);
        return write >= read ? (write - read) : (capacity_ - read + write);
    }

    /// @brief Check if the queue is empty.
    /// @return true if empty (approximate).
    bool empty() const {
        return read_pos_.load(std::memory_order_acquire) ==
               write_pos_.load(std::memory_order_acquire);
    }

    /// @brief Get the capacity of the queue.
    /// @return Maximum number of elements (capacity-1, since one slot is reserved).
    size_t capacity() const { return capacity_ - 1; }
};

}  // namespace queue
