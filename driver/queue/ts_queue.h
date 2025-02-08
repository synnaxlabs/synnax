// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <queue>
#include <mutex>
#include <condition_variable>

template<typename T>
class TSQueue {
public:
    TSQueue() = default;

    void enqueue(T&& item) {
        std::unique_lock lock(m);
        
        if (queue.size() == 1) {
            queue.pop();
        }
        queue.push(std::move(item));
        
        waiting_consumers.notify_one();
    }

    void enqueue(const T& item) {
        enqueue(T(item));
    }

    std::pair<T, bool> dequeue(void) {
        std::unique_lock lock(m);
        
        if (!waiting_consumers.wait_for(lock, std::chrono::seconds(2), 
            [this] { return !queue.empty(); })) {
            return std::make_pair(T(), false);
        }

        T item = std::move(queue.front());
        queue.pop();
        return std::make_pair(std::move(item), true);
    }

    void reset() {
        std::lock_guard<std::mutex> lock(m);
        while (!queue.empty()) {
            queue.pop();
        }
    }

private:
    std::queue<T> queue;
    std::mutex m;
    std::condition_variable waiting_consumers;
};
