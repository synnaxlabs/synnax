// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/runtime/loop/loop.h"

using namespace arc::runtime::loop;

/// @brief Test that Loop can be created.
TEST(LoopTest, Create) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create(config));
    ASSERT_NE(loop, nullptr);
}

/// @brief Test that Loop can be started and stopped.
TEST(LoopTest, StartStop) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::MILLISECOND;
    const auto loop = ASSERT_NIL_P(create(config));
    loop->stop();
}

/// @brief Test that Loop wakes up on data notification (EVENT_DRIVEN mode).
TEST(LoopTest, NotifyData_EventDriven) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0); // No timer

    const auto loop = ASSERT_NIL_P(create(config));
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
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 10 * telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create(config));
    ASSERT_NIL(loop->start());

    breaker::Breaker breaker;

    const auto start = std::chrono::steady_clock::now();
    loop->wait(breaker);
    const auto elapsed = std::chrono::steady_clock::now() - start;

    // Should have waited approximately 10ms
    // Allow some jitter (5-50ms range)
    const auto elapsed_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                                elapsed
    )
                                .count();
    EXPECT_GE(elapsed_ms, 5);
    EXPECT_LE(elapsed_ms, 50);

    loop->stop();
}

/// @brief Test BUSY_WAIT mode responds quickly to notifications.
TEST(LoopTest, BusyWaitMode) {
    Config config;
    config.mode = ExecutionMode::BUSY_WAIT;
    config.interval = telem::TimeSpan(0); // No timer

    const auto loop = ASSERT_NIL_P(create(config));
    ASSERT_NIL(loop->start());

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(std::chrono::microseconds(100));

    const auto start = std::chrono::steady_clock::now();
    loop->notify_data();

    waiter.join();

    const auto latency = std::chrono::steady_clock::now() - start;
    const auto latency_us = std::chrono::duration_cast<std::chrono::microseconds>(
                                latency
    )
                                .count();

    // Busy wait should have very low latency (< 1ms)
    EXPECT_LE(latency_us, 1000);
    ASSERT_TRUE(woke_up.load());

    loop->stop();
}

/// @brief Test HIGH_RATE mode with timer.
TEST(LoopTest, HighRateMode) {
    Config config;
    config.mode = ExecutionMode::HIGH_RATE;
    config.interval = 10 * telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create(config));
    ASSERT_NIL(loop->start());

    breaker::Breaker breaker;

    const auto start = std::chrono::steady_clock::now();
    loop->wait(breaker);
    const auto elapsed = std::chrono::steady_clock::now() - start;

    const auto elapsed_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                                elapsed
    )
                                .count();

    // Should wait approximately 10ms with high-rate timer
    EXPECT_GE(elapsed_ms, 5);
    EXPECT_LE(elapsed_ms, 50);

    loop->stop();
}

/// @brief Test HYBRID mode behavior.
TEST(LoopTest, HybridMode) {
    Config config;
    config.mode = ExecutionMode::HYBRID;
    config.interval = telem::TimeSpan(0); // No timer
    config.spin_duration = 50 * telem::MICROSECOND; // 50us spin

    const auto loop = ASSERT_NIL_P(create(config));
    ASSERT_NIL(loop->start());

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(std::chrono::microseconds(10));
    loop->notify_data();

    waiter.join();
    ASSERT_TRUE(woke_up.load());

    loop->stop();
}

/// @brief Test multiple start/stop cycles.
TEST(LoopTest, MultipleStartStop) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::MILLISECOND;

    for (int i = 0; i < 3; i++) {
        const auto loop = ASSERT_NIL_P(create(config));
        ASSERT_NIL(loop->start());
        loop->stop();
    }
}

/// @brief Test different execution modes.
TEST(LoopTest, DifferentModes) {
    ExecutionMode modes[] = {
        ExecutionMode::BUSY_WAIT,
        ExecutionMode::HIGH_RATE,
        ExecutionMode::HYBRID,
        ExecutionMode::EVENT_DRIVEN,
        ExecutionMode::RT_EVENT
    };

    for (const auto mode: modes) {
        Config config;
        config.mode = mode;
        config.interval = telem::MILLISECOND;
        const auto loop = ASSERT_NIL_P(create(config));
        ASSERT_NIL(loop->start());
        loop->stop();
    }
}

TEST(ModeSelectorTest, NoIntervals_SelectsEventDriven) {
    EXPECT_EQ(select_mode(telem::TimeSpan(0), false), ExecutionMode::EVENT_DRIVEN);
}

TEST(ModeSelectorTest, ModerateRate_SelectsHybrid) {
    EXPECT_EQ(select_mode(3 * telem::MILLISECOND, true), ExecutionMode::HYBRID);
}

TEST(ModeSelectorTest, LowRate_SelectsEventDriven) {
    EXPECT_EQ(select_mode(10 * telem::MILLISECOND, true), ExecutionMode::EVENT_DRIVEN);
}

TEST(ModeSelectorTest, NeverAutoselectsBusyWait) {
    EXPECT_NE(select_mode(10 * telem::MICROSECOND, true), ExecutionMode::BUSY_WAIT);
    EXPECT_NE(select_mode(telem::TimeSpan(0), true), ExecutionMode::BUSY_WAIT);
}

TEST(ModeSelectorTest, Boundary_AtOneMs_SelectsHybrid) {
    EXPECT_EQ(select_mode(telem::MILLISECOND, true), ExecutionMode::HYBRID);
}

TEST(ModeSelectorTest, Boundary_AtFiveMs_SelectsEventDriven) {
    EXPECT_EQ(select_mode(5 * telem::MILLISECOND, true), ExecutionMode::EVENT_DRIVEN);
}

TEST(ConfigTest, ApplyDefaultsResolvesAuto) {
    Config cfg;
    EXPECT_EQ(cfg.mode, ExecutionMode::AUTO);
    const auto resolved = cfg.apply_defaults(10 * telem::MILLISECOND);
    EXPECT_NE(resolved.mode, ExecutionMode::AUTO);
}

TEST(ConfigTest, ApplyDefaultsSetsInterval) {
    Config cfg;
    EXPECT_EQ(cfg.interval.nanoseconds(), 0);
    const auto resolved = cfg.apply_defaults(10 * telem::MILLISECOND);
    EXPECT_EQ(resolved.interval, 10 * telem::MILLISECOND);
}

TEST(ConfigTest, DefaultRtPriority) {
    Config cfg;
    EXPECT_EQ(cfg.rt_priority, timing::DEFAULT_RT_PRIORITY);
}

TEST(ConfigTest, AutoCpuAffinityPinsForRTEvent) {
    Config cfg;
    cfg.mode = ExecutionMode::RT_EVENT;
    EXPECT_EQ(cfg.cpu_affinity, timing::CPU_AFFINITY_AUTO);
    const auto resolved = cfg.apply_defaults(500 * telem::MICROSECOND);
    if (std::thread::hardware_concurrency() > 1) EXPECT_GE(resolved.cpu_affinity, 0);
}

TEST(ConfigTest, ExplicitCpuAffinityNotOverridden) {
    Config cfg;
    cfg.mode = ExecutionMode::RT_EVENT;
    cfg.cpu_affinity = 0;
    const auto resolved = cfg.apply_defaults(500 * telem::MICROSECOND);
    EXPECT_EQ(resolved.cpu_affinity, 0);
}

TEST(ConfigTest, ExplicitModeNotOverridden) {
    Config cfg;
    cfg.mode = ExecutionMode::BUSY_WAIT;
    const auto resolved = cfg.apply_defaults(10 * telem::MILLISECOND);
    EXPECT_EQ(resolved.mode, ExecutionMode::BUSY_WAIT);
}
