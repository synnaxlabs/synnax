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
    queue::SPSC<int> queue;

    // Push element
    EXPECT_TRUE(queue.push(42));

    // Pop element
    int value;
    EXPECT_TRUE(queue.pop(value));
    EXPECT_EQ(value, 42);
}

TEST(SPSCQueueTest, Empty) {
    queue::SPSC<int> queue;

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

TEST(SPSCQueueTest, TryPopEmpty) {
    queue::SPSC<int> queue;

    // try_pop should return false on empty queue
    int value;
    EXPECT_FALSE(queue.try_pop(value));

    // Push and try_pop should succeed
    queue.push(42);
    EXPECT_TRUE(queue.try_pop(value));
    EXPECT_EQ(value, 42);

    // Now empty again
    EXPECT_FALSE(queue.try_pop(value));
}

TEST(SPSCQueueTest, MoveSemantics) {
    queue::SPSC<std::unique_ptr<int>> queue;

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

TEST(SPSCQueueTest, CloseQueue) {
    queue::SPSC<int> queue;

    EXPECT_FALSE(queue.closed());

    // Push before close
    EXPECT_TRUE(queue.push(1));

    // Close queue
    queue.close();
    EXPECT_TRUE(queue.closed());

    // Push after close should fail
    EXPECT_FALSE(queue.push(2));

    // Pop existing element should succeed
    int value;
    EXPECT_TRUE(queue.pop(value));
    EXPECT_EQ(value, 1);

    // Pop from empty closed queue should fail
    EXPECT_FALSE(queue.pop(value));
}

TEST(SPSCQueueTest, CloseUnblocksWaitingPop) {
    queue::SPSC<int> queue;

    std::thread consumer([&]() {
        int value;
        // This will block until close() is called
        EXPECT_FALSE(queue.pop(value));
    });

    // Give consumer time to start blocking
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    // Close should unblock the consumer
    queue.close();

    consumer.join();
}

TEST(SPSCQueueTest, MultipleRounds) {
    queue::SPSC<int> queue;

    // Push and pop multiple rounds
    for (int round = 0; round < 10; round++) {
        // Push several items
        for (int i = 0; i < 7; i++) {
            EXPECT_TRUE(queue.push(round * 100 + i));
        }

        // Pop all items
        for (int i = 0; i < 7; i++) {
            int value;
            EXPECT_TRUE(queue.pop(value));
            EXPECT_EQ(value, round * 100 + i);
        }

        EXPECT_TRUE(queue.empty());
    }
}

TEST(SPSCQueueTest, ProducerConsumerThreads) {
    queue::SPSC<int> queue;
    constexpr int num_items = 10000;

    // Producer thread
    std::thread producer([&]() {
        for (int i = 0; i < num_items; i++) {
            queue.push(i);
        }
    });

    // Consumer thread
    std::thread consumer([&]() {
        for (int i = 0; i < num_items; i++) {
            int value;
            EXPECT_TRUE(queue.pop(value));
            EXPECT_EQ(value, i);
        }
    });

    producer.join();
    consumer.join();

    EXPECT_TRUE(queue.empty());
}
