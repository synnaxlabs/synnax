// Copyright 2026 Synnax Labs, Inc.
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

namespace x::queue {
TEST(SPSCQueueTest, BasicPushPop) {
    SPSC<int> queue;
    EXPECT_TRUE(queue.push(42));
    int value;
    EXPECT_TRUE(queue.pop(value));
    EXPECT_EQ(value, 42);
}

TEST(SPSCQueueTest, Empty) {
    SPSC<int> queue;
    EXPECT_TRUE(queue.empty());
    queue.push(1);
    EXPECT_FALSE(queue.empty());
    int value;
    queue.pop(value);
    EXPECT_TRUE(queue.empty());
}

TEST(SPSCQueueTest, TryPopEmpty) {
    SPSC<int> queue;
    int value;
    EXPECT_FALSE(queue.try_pop(value));
    queue.push(42);
    EXPECT_TRUE(queue.try_pop(value));
    EXPECT_EQ(value, 42);
    EXPECT_FALSE(queue.try_pop(value));
}

TEST(SPSCQueueTest, MoveSemantics) {
    SPSC<std::unique_ptr<int>> queue;
    auto ptr = std::make_unique<int>(42);
    EXPECT_TRUE(queue.push(std::move(ptr)));
    EXPECT_EQ(ptr, nullptr);
    std::unique_ptr<int> result;
    EXPECT_TRUE(queue.pop(result));
    EXPECT_NE(result, nullptr);
    EXPECT_EQ(*result, 42);
}

TEST(SPSCQueueTest, CloseQueue) {
    SPSC<int> queue;
    EXPECT_FALSE(queue.closed());
    EXPECT_TRUE(queue.push(1));
    queue.close();
    EXPECT_TRUE(queue.closed());
    EXPECT_FALSE(queue.push(2));
    int value;
    EXPECT_TRUE(queue.pop(value));
    EXPECT_EQ(value, 1);
    EXPECT_FALSE(queue.pop(value));
}

TEST(SPSCQueueTest, CloseUnblocksWaitingPop) {
    SPSC<int> queue;
    std::thread consumer([&]() {
        int value;
        EXPECT_FALSE(queue.pop(value));
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
    queue.close();
    consumer.join();
}

TEST(SPSCQueueTest, MultipleRounds) {
    SPSC<int> queue;
    for (int round = 0; round < 10; round++) {
        for (int i = 0; i < 7; i++)
            EXPECT_TRUE(queue.push(round * 100 + i));
        for (int i = 0; i < 7; i++) {
            int value;
            EXPECT_TRUE(queue.pop(value));
            EXPECT_EQ(value, round * 100 + i);
        }
        EXPECT_TRUE(queue.empty());
    }
}

TEST(SPSCQueueTest, ProducerConsumerThreads) {
    SPSC<int> queue;
    constexpr int num_items = 10000;
    std::thread producer([&]() {
        for (int i = 0; i < num_items; i++)
            queue.push(i);
    });
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
}
