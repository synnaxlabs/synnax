// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/notify/notify.h"
#include "x/cpp/queue/spsc.h"

namespace x::queue {
/// @brief it should push and pop a single element.
TEST(SPSCQueueTest, BasicPushPop) {
    SPSC<int> queue;
    EXPECT_TRUE(queue.push(42));
    int value;
    EXPECT_TRUE(queue.pop(value));
    EXPECT_EQ(value, 42);
}

/// @brief it should report empty correctly.
TEST(SPSCQueueTest, Empty) {
    SPSC<int> queue;
    EXPECT_TRUE(queue.empty());
    queue.push(1);
    EXPECT_FALSE(queue.empty());
    int value;
    queue.pop(value);
    EXPECT_TRUE(queue.empty());
}

/// @brief it should return false on try_pop when empty.
TEST(SPSCQueueTest, TryPopEmpty) {
    SPSC<int> queue;
    int value;
    EXPECT_FALSE(queue.try_pop(value));
    queue.push(42);
    EXPECT_TRUE(queue.try_pop(value));
    EXPECT_EQ(value, 42);
    EXPECT_FALSE(queue.try_pop(value));
}

/// @brief it should correctly move unique_ptr through the queue.
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

/// @brief it should reject pushes after close.
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

/// @brief it should unblock waiting pop when closed.
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

/// @brief it should handle multiple push/pop rounds.
TEST(SPSCQueueTest, MultipleRounds) {
    SPSC<int> queue;

    for (int round = 0; round < 10; round++) {
        for (int i = 0; i < 7; i++) {
            EXPECT_TRUE(queue.push(round * 100 + i));
        }
        for (int i = 0; i < 7; i++) {
            int value;
            EXPECT_TRUE(queue.pop(value));
            EXPECT_EQ(value, round * 100 + i);
        }
        EXPECT_TRUE(queue.empty());
    }
}

/// @brief it should handle concurrent producer and consumer threads.
TEST(SPSCQueueTest, ProducerConsumerThreads) {
    SPSC<int> queue;
    constexpr int num_items = 10000;

    std::thread producer([&]() {
        for (int i = 0; i < num_items; i++) {
            while (!queue.push(i))
                std::this_thread::yield();
        }
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

/// @brief it should honor explicit capacity.
TEST(SPSCQueueTest, ExplicitCapacity) {
    SPSC<int> queue(16);
    EXPECT_GE(queue.capacity(), 16);
    size_t pushed = 0;
    while (queue.push(static_cast<int>(pushed)))
        pushed++;
    EXPECT_GE(pushed, 16);
    int value;
    while (queue.try_pop(value)) {}
    EXPECT_TRUE(queue.empty());
}

/// @brief it should return false when queue is full.
TEST(SPSCQueueTest, FullQueueReturnsFalse) {
    SPSC<int> queue(4);
    size_t pushed = 0;
    while (queue.push(static_cast<int>(pushed)))
        pushed++;
    EXPECT_FALSE(queue.push(999));
    int value;
    EXPECT_TRUE(queue.pop(value));
    EXPECT_TRUE(queue.push(999));
}

/// @brief it should expose the notifier for external integration.
TEST(SPSCQueueTest, NotifierAccess) {
    SPSC<int> queue;
    notify::Notifier &notif = queue.notifier();
    notif.signal();
    EXPECT_TRUE(notif.poll());
}

/// @brief it should track size correctly.
TEST(SPSCQueueTest, SizeTracking) {
    SPSC<int> queue(16);
    EXPECT_EQ(queue.size(), 0);
    queue.push(1);
    EXPECT_EQ(queue.size(), 1);
    queue.push(2);
    queue.push(3);
    EXPECT_EQ(queue.size(), 3);
    int value;
    queue.pop(value);
    EXPECT_EQ(queue.size(), 2);
    queue.pop(value);
    queue.pop(value);
    EXPECT_EQ(queue.size(), 0);
}

/// @brief it should handle high-throughput lock-free operations.
TEST(SPSCQueueTest, LockFreeStressTest) {
    SPSC<int> queue(1024);
    constexpr int num_items = 100000;
    std::atomic<int> items_received{0};

    std::thread producer([&]() {
        for (int i = 0; i < num_items; i++) {
            while (!queue.push(i))
                std::this_thread::yield();
        }
    });

    std::thread consumer([&]() {
        int expected = 0;
        while (expected < num_items) {
            int value;
            if (queue.try_pop(value)) {
                EXPECT_EQ(value, expected);
                expected++;
                items_received.fetch_add(1, std::memory_order_relaxed);
            } else {
                std::this_thread::yield();
            }
        }
    });

    producer.join();
    consumer.join();
    EXPECT_EQ(items_received.load(), num_items);
    EXPECT_TRUE(queue.empty());
}
}

/// @brief it should drain items and reopen after reset.
TEST(SPSCQueueTest, ResetDrainsAndReopens) {
    queue::SPSC<int> queue;
    EXPECT_TRUE(queue.push(1));
    EXPECT_TRUE(queue.push(2));
    EXPECT_TRUE(queue.push(3));
    queue.close();
    EXPECT_TRUE(queue.closed());
    EXPECT_FALSE(queue.push(4));

    queue.reset();

    EXPECT_FALSE(queue.closed());
    EXPECT_TRUE(queue.empty());
    EXPECT_TRUE(queue.push(10));
    int value;
    EXPECT_TRUE(queue.pop(value));
    EXPECT_EQ(value, 10);
}

/// @brief it should allow multiple reset cycles.
TEST(SPSCQueueTest, MultipleResetCycles) {
    queue::SPSC<int> queue;

    for (int cycle = 0; cycle < 3; cycle++) {
        EXPECT_TRUE(queue.push(cycle * 10 + 1));
        EXPECT_TRUE(queue.push(cycle * 10 + 2));
        queue.close();
        queue.reset();
        EXPECT_TRUE(queue.empty());
        EXPECT_FALSE(queue.closed());
    }

    EXPECT_TRUE(queue.push(100));
    int value;
    EXPECT_TRUE(queue.pop(value));
    EXPECT_EQ(value, 100);
}
