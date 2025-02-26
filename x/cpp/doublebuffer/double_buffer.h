// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <mutex>
#include <condition_variable>
#include <chrono>

/// @brief A minimal SPSC (Single Producer Single Consumer) double buffer queue
/// @tparam T The type of data to be stored in the buffers
template<typename T>
class DoubleBuffer {
public:
    /// @brief Constructs a DoubleBuffer with two initial values
    /// @param buffer0 First buffer value (moved)
    /// @param buffer1 Second buffer value (moved)
    DoubleBuffer(T&& buffer0, T&& buffer1) 
        : buffers{std::move(buffer0), std::move(buffer1)} {}

    // Prevent copying/moving
    DoubleBuffer(const DoubleBuffer&) = delete;
    DoubleBuffer& operator=(const DoubleBuffer&) = delete;
    DoubleBuffer(DoubleBuffer&&) = delete;
    DoubleBuffer& operator=(DoubleBuffer&&) = delete;

    /// @brief Get write access to the next buffer (producer)
    T* start_writing() { return &buffers[write_idx]; }

    /// @brief Signal that writing is complete and make buffer available for reading
    void stop_writing() {
        std::unique_lock<std::mutex> lock(mtx);
        cv.wait(lock, [this] { return !reading; });
        read_idx = write_idx;
        write_idx = (write_idx + 1) % 2;
        has_data = true;
        cv.notify_one();
    }

    /// @brief Wait for and get read access to the next buffer (consumer)
    /// @param timeout Maximum duration to wait for data
    /// @return Pair of (buffer pointer, success). Buffer pointer is null if timeout occurred
    template<typename Rep, typename Period>
    std::pair<T*, bool> start_reading(const std::chrono::duration<Rep, Period>& timeout) {
        std::unique_lock<std::mutex> lock(mtx);
        if (!cv.wait_for(lock, timeout, [this] { return has_data; })) {
            return {nullptr, false};  // Timeout occurred
        }
        reading = true;
        return {&buffers[read_idx], true};
    }

    /// @brief Wait indefinitely for and get read access to the next buffer (consumer)
    /// @return Pointer to the buffer that can be read from
    T* start_reading() {
        std::unique_lock<std::mutex> lock(mtx);
        cv.wait(lock, [this] { return has_data; });
        reading = true;
        return &buffers[read_idx];
    }

    /// @brief Signal that reading is complete
    void stop_reading() {
        std::unique_lock<std::mutex> lock(mtx);
        reading = false;
        has_data = false;
        cv.notify_one();
    }

private:
    T buffers[2];
    size_t read_idx{0};
    size_t write_idx{1};
    bool has_data{false};
    bool reading{false};
    std::mutex mtx;
    std::condition_variable cv;
};
