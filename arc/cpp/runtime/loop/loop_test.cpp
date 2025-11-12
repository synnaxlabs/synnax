// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/loop/loop.h"

#include <atomic>
#include <chrono>
#include <thread>

#include "gtest/gtest.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xtest/xtest.h"

using namespace arc::runtime::loop;

/// @brief Test that Loop can be created.
TEST(LoopTest, Create) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);
}

/// @brief Test that Loop can be configured.
TEST(LoopTest, Configure) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 1'000'000;  // 1ms

    auto err = loop->configure(config);
    ASSERT_NIL(err);
}

/// @brief Test that Loop can be started and stopped.
TEST(LoopTest, StartStop) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 1'000'000;  // 1ms

    ASSERT_NIL(loop->configure(config));
    ASSERT_NIL(loop->start());

    loop->stop();
}

/// @brief Test that Loop wakes up on data notification (EVENT_DRIVEN mode).
TEST(LoopTest, NotifyData_EventDriven) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 0;  // No timer

    ASSERT_NIL(loop->configure(config));
    ASSERT_NIL(loop->start());

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    // Start a thread that waits on the loop
    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    // Give the waiter time to start waiting
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // Notify data available
    loop->notify_data();

    // Wait for the waiter thread
    waiter.join();

    // Verify that the loop woke up
    ASSERT_TRUE(woke_up.load());

    loop->stop();
}

/// @brief Test that Loop wakes up on timer expiration.
TEST(LoopTest, TimerExpiration) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 10'000'000;  // 10ms

    ASSERT_NIL(loop->configure(config));
    ASSERT_NIL(loop->start());

    breaker::Breaker breaker;

    const auto start = std::chrono::steady_clock::now();
    loop->wait(breaker);
    const auto elapsed = std::chrono::steady_clock::now() - start;

    // Should have waited approximately 10ms
    // Allow some jitter (5-20ms range)
    const auto elapsed_ms =
        std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();
    EXPECT_GE(elapsed_ms, 5);
    EXPECT_LE(elapsed_ms, 50);

    loop->stop();
}

/// @brief Test BUSY_WAIT mode responds quickly to notifications.
TEST(LoopTest, BusyWaitMode) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    Config config;
    config.mode = ExecutionMode::BUSY_WAIT;
    config.interval = 0;  // No timer

    ASSERT_NIL(loop->configure(config));
    ASSERT_NIL(loop->start());

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    // Very short delay before notification
    std::this_thread::sleep_for(std::chrono::microseconds(100));

    const auto start = std::chrono::steady_clock::now();
    loop->notify_data();

    waiter.join();

    const auto latency = std::chrono::steady_clock::now() - start;
    const auto latency_us =
        std::chrono::duration_cast<std::chrono::microseconds>(latency).count();

    // Busy wait should have very low latency (< 1ms)
    EXPECT_LE(latency_us, 1000);
    ASSERT_TRUE(woke_up.load());

    loop->stop();
}

/// @brief Test HIGH_RATE mode with timer.
TEST(LoopTest, HighRateMode) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    Config config;
    config.mode = ExecutionMode::HIGH_RATE;
    config.interval = 10'000'000;  // 10ms

    ASSERT_NIL(loop->configure(config));
    ASSERT_NIL(loop->start());

    breaker::Breaker breaker;

    const auto start = std::chrono::steady_clock::now();
    loop->wait(breaker);
    const auto elapsed = std::chrono::steady_clock::now() - start;

    const auto elapsed_ms =
        std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count();

    // Should wait approximately 10ms with high-rate timer
    EXPECT_GE(elapsed_ms, 5);
    EXPECT_LE(elapsed_ms, 50);

    loop->stop();
}

/// @brief Test HYBRID mode behavior.
TEST(LoopTest, HybridMode) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    Config config;
    config.mode = ExecutionMode::HYBRID;
    config.interval = 0;  // No timer
    config.spin_duration_us = 50;  // 50us spin

    ASSERT_NIL(loop->configure(config));
    ASSERT_NIL(loop->start());

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    // Notify within spin window
    std::this_thread::sleep_for(std::chrono::microseconds(10));
    loop->notify_data();

    waiter.join();
    ASSERT_TRUE(woke_up.load());

    loop->stop();
}

/// @brief Test that breaker stops the wait.
TEST(LoopTest, BreakerStopsWait) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 0;  // No timer

    ASSERT_NIL(loop->configure(config));
    ASSERT_NIL(loop->start());

    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
    });

    // Give the waiter time to start waiting
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    // Break the breaker
    breaker.stop();

    // Waiter should exit promptly
    waiter.join();

    loop->stop();
}

/// @brief Test multiple start/stop cycles.
TEST(LoopTest, MultipleStartStop) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 1'000'000;  // 1ms

    for (int i = 0; i < 3; i++) {
        ASSERT_NIL(loop->configure(config));
        ASSERT_NIL(loop->start());
        loop->stop();
    }
}

/// @brief Test reconfiguration between start/stop.
TEST(LoopTest, Reconfigure) {
    auto loop = create();
    ASSERT_NE(loop, nullptr);

    // First configuration
    Config config1;
    config1.mode = ExecutionMode::EVENT_DRIVEN;
    config1.interval = 1'000'000;  // 1ms

    ASSERT_NIL(loop->configure(config1));
    ASSERT_NIL(loop->start());
    loop->stop();

    // Second configuration
    Config config2;
    config2.mode = ExecutionMode::HIGH_RATE;
    config2.interval = 5'000'000;  // 5ms

    ASSERT_NIL(loop->configure(config2));
    ASSERT_NIL(loop->start());
    loop->stop();
}
