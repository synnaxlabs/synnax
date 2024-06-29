// Copyright 2024 Synnax Labs, Inc.
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

    void enqueue(const T &item) {
        std::unique_lock lock(m);

        if (queue.size() == 1) {
            queue.pop();
        }
        queue.push(item);
        
        waiting_consumers.notify_one();
    }

    std::pair<T, bool> dequeue(void) {
        std::unique_lock lock(m);

        waiting_consumers.wait_for(lock, std::chrono::seconds(2));

        if (queue.empty()) {
            // FIXME change to while?
            return std::make_pair(T(), false);
        }

        T item = queue.front();
        queue.pop();

        return std::make_pair(item, true);
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
