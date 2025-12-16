// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/queue/spsc.h"

TEST(SPSCQueueTest, BasicPushPop) {
    queue::SPSC<int, 16> queue;

    // Push element
    EXPECT_TRUE(queue.push(42));

    // Pop element
    int value;
    EXPECT_TRUE(queue.pop(value));
    EXPECT_EQ(value, 42);
}

TEST(SPSCQueueTest, Empty) {
    queue::SPSC<int, 16> queue;

    // Queue should be empty initially
    EXPECT_TRUE(queue.empty());

    // Push element
    queue.push(1);
    EXPECT_FALSE(queue.empty());

    // Pop element
    int value;
    queue.pop(value);
    EXPECT_TRUE(queue.empty());
}

TEST(SPSCQueueTest, Full) {
    queue::SPSC<int, 4> queue; // Capacity 3 (N-1)

    // Fill queue
    EXPECT_TRUE(queue.push(1));
    EXPECT_TRUE(queue.push(2));
    EXPECT_TRUE(queue.push(3));

    // Queue should be full now
    EXPECT_FALSE(queue.push(4));

    // Pop one element
    int value;
    queue.pop(value);

    // Now we can push again
    EXPECT_TRUE(queue.push(4));
}

TEST(SPSCQueueTest, MoveSemantics) {
    queue::SPSC<std::unique_ptr<int>, 16> queue;

    // Push unique_ptr
    auto ptr = std::make_unique<int>(42);
    EXPECT_TRUE(queue.push(std::move(ptr)));
    EXPECT_EQ(ptr, nullptr); // Moved from

    // Pop unique_ptr
    std::unique_ptr<int> result;
    EXPECT_TRUE(queue.pop(result));
    EXPECT_NE(result, nullptr);
    EXPECT_EQ(*result, 42);
}

TEST(SPSCQueueTest, Size) {
    queue::SPSC<int, 16> queue;

    EXPECT_EQ(queue.size(), 0);

    queue.push(1);
    EXPECT_EQ(queue.size(), 1);

    queue.push(2);
    EXPECT_EQ(queue.size(), 2);

    int value;
    queue.pop(value);
    EXPECT_EQ(queue.size(), 1);
}

TEST(SPSCQueueTest, MultipleRounds) {
    queue::SPSC<int, 8> queue;

    // Push and pop multiple rounds to test wrap-around
    for (int round = 0; round < 10; round++) {
        // Fill queue
        for (int i = 0; i < 7; i++) { // Capacity is 7 (N-1)
            EXPECT_TRUE(queue.push(round * 100 + i));
        }

        // Empty queue
        for (int i = 0; i < 7; i++) {
            int value;
            EXPECT_TRUE(queue.pop(value));
            EXPECT_EQ(value, round * 100 + i);
        }

        EXPECT_TRUE(queue.empty());
    }
}

TEST(SPSCQueueTest, ProducerConsumerThreads) {
    queue::SPSC<int, 1024> queue;
    constexpr int num_items = 10000;

    // Producer thread
    std::thread producer([&]() {
        for (int i = 0; i < num_items; i++) {
            int val = i; // Create copy for move
            while (!queue.push(std::move(val))) {
                // Busy wait if queue full (should be rare with 1024 capacity)
                val = i; // Recreate if push failed
                std::this_thread::yield();
            }
        }
    });

    // Consumer thread
    std::thread consumer([&]() {
        for (int i = 0; i < num_items; i++) {
            int value;
            while (!queue.pop(value)) {
                // Busy wait if queue empty
                std::this_thread::yield();
            }
            EXPECT_EQ(value, i);
        }
    });

    producer.join();
    consumer.join();

    EXPECT_TRUE(queue.empty());
}
