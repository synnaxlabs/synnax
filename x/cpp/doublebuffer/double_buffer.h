// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
#pragma once

#include <atomic>
#include <memory>

/// @brief DoubleBuffer implements a lock-free double buffering pattern for concurrent read/write operations.
/// 
/// The double buffer maintains two internal buffers and allows concurrent read and write operations
/// by toggling between them. This is particularly useful in producer-consumer scenarios where one thread
/// writes data while another reads it, avoiding the need for mutual exclusion locks.
///
/// This double buffer can only be used by a single reader and writer.
///
/// @tparam T The type of data to be stored in the buffers
template<typename T>
class DoubleBuffer {
public:
    // Remove default constructor since we always want initialized buffers
    DoubleBuffer() = delete;

    /// @brief Constructs a DoubleBuffer with two initial values
    /// @param buffer0 First buffer value (moved)
    /// @param buffer1 Second buffer value (moved)
    DoubleBuffer(T&& buffer0, T&& buffer1): buffers{std::move(buffer0), std::move(buffer1)} {
        current.store(&buffers[0], std::memory_order_release);
    }

    // Prevent copying/moving as this contains atomic members
    DoubleBuffer(const DoubleBuffer &) = delete;
    DoubleBuffer &operator=(const DoubleBuffer &) = delete;
    DoubleBuffer(DoubleBuffer &&) = delete;
    DoubleBuffer &operator=(DoubleBuffer &&) = delete;

    /// @brief Provides access to the current readable buffer
    /// @return A pair containing a pointer to the current buffer and a boolean
    /// indicating if new data is available
    /// @details Returns nullptr and false if no new data is available since the
    /// last read. This method is atomic and thread-safe.
    std::pair<T*, bool> curr_read() {
        const uint64_t seq = current_seq.load(std::memory_order_acquire);
        if (seq == this->last_consumed_seq) return {nullptr, false};
        this->last_consumed_seq = seq;
        T* curr = this->current.load(std::memory_order_acquire);
        return {curr, true};
    }

    /// @brief Provides access to the current writable buffer
    /// @return Pointer to the buffer that can be safely written to
    /// @details this method cannot be called concurrently with other calls to curr_write()
    /// or exchange().
    T* curr_write() {
        T* curr = &this->buffers[write_idx];
        return curr;
    }

    /// @brief Swaps the read and write buffers
    /// @details This operation makes the previously writable buffer readable and vice versa.
    /// This method cannot be called with other calls to curr_write() or exchange().
    void exchange() {
        this->current.store(&this->buffers[this->write_idx], std::memory_order_release);
        this->current_seq.fetch_add(1, std::memory_order_release);
        this->write_idx = (this->write_idx + 1) % 2;
    }

private:
    /// @brief Pointer to the current readable buffer.
    std::atomic<T *> current{};
    /// @brief The two internal buffers.
    T buffers[2];
    /// @brief Index of the current writable buffer.
    size_t write_idx{0};
    /// @brief Sequence number for detecting new data.
    std::atomic<uint64_t> current_seq{0};
    /// @brief Sequence number of the last consumed data.
    uint64_t last_consumed_seq{0};
};
