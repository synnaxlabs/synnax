// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <sstream>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/notify/notify.h"
#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/runtime/loop/loop.h"

using namespace arc::runtime::loop;

/// @brief Test timing constants.
namespace test_timing {
/// @brief Time to wait for a thread to start waiting before signaling.
const auto THREAD_STARTUP = 50 * telem::MILLISECOND;
/// @brief Small delay before wake to ensure thread is ready.
const auto SMALL_DELAY = 100 * telem::MICROSECOND;
/// @brief Expected timer bounds (lower).
const auto TIMER_LOWER_BOUND = 5 * telem::MILLISECOND;
/// @brief Expected timer bounds (upper, accounts for system jitter).
const auto TIMER_UPPER_BOUND = 50 * telem::MILLISECOND;
/// @brief Maximum wake latency (Windows ~15ms scheduler time slice, POSIX ~1ms).
#ifdef _WIN32
const auto WAKE_LATENCY = 50 * telem::MILLISECOND;
#else
const auto WAKE_LATENCY = telem::MILLISECOND;
#endif
/// @brief Maximum time for breaker stop to take effect.
const auto BREAKER_STOP_LATENCY = 10 * telem::MILLISECOND;
/// @brief Maximum time for event-driven timeout (100ms + margin).
const auto EVENT_DRIVEN_BOUND = 150 * telem::MILLISECOND;
}

/// @brief Test that Loop can be created.
TEST(LoopTest, Create) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create(config));
    ASSERT_NE(loop, nullptr);
}

/// @brief Test that Loop can be created and destroyed.
TEST(LoopTest, CreateAndDestroy) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::MILLISECOND;
    const auto loop = ASSERT_NIL_P(create(config));
    // Loop is cleaned up when it goes out of scope
}

/// @brief Test that Loop wakes up on wake() call (EVENT_DRIVEN mode).
TEST(LoopTest, Wake_EventDriven) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0); // No timer

    const auto loop = ASSERT_NIL_P(create(config));

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    // Give the waiter time to start waiting
    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());

    // Wake should unblock immediately
    loop->wake();

    waiter.join();
    ASSERT_TRUE(woke_up.load());
}

/// @brief Test that Loop wakes up on timer expiration.
TEST(LoopTest, TimerExpiration) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 10 * telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create(config));

    breaker::Breaker breaker;

    const auto sw = telem::Stopwatch();
    loop->wait(breaker);

    // Should have waited approximately 10ms (allow some jitter)
    const auto elapsed = sw.elapsed();
    EXPECT_GE(elapsed, test_timing::TIMER_LOWER_BOUND);
    EXPECT_LE(elapsed, test_timing::TIMER_UPPER_BOUND);
}

/// @brief Test BUSY_WAIT mode responds quickly to wake().
TEST(LoopTest, BusyWaitMode) {
    Config config;
    config.mode = ExecutionMode::BUSY_WAIT;
    config.interval = telem::TimeSpan(0); // No timer

    const auto loop = ASSERT_NIL_P(create(config));

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::SMALL_DELAY.chrono());

    const auto sw = telem::Stopwatch();
    loop->wake();
    waiter.join();

    EXPECT_LE(sw.elapsed(), test_timing::WAKE_LATENCY);
    ASSERT_TRUE(woke_up.load());
}

/// @brief Test HIGH_RATE mode with timer.
TEST(LoopTest, HighRateMode) {
    Config config;
    config.mode = ExecutionMode::HIGH_RATE;
    config.interval = 10 * telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create(config));

    breaker::Breaker breaker;

    const auto sw = telem::Stopwatch();
    loop->wait(breaker);

    // Should wait approximately 10ms with high-rate timer
    const auto elapsed = sw.elapsed();
    EXPECT_GE(elapsed, test_timing::TIMER_LOWER_BOUND);
    EXPECT_LE(elapsed, test_timing::TIMER_UPPER_BOUND);
}

/// @brief Test HYBRID mode behavior.
TEST(LoopTest, HybridMode) {
    Config config;
    config.mode = ExecutionMode::HYBRID;
    config.interval = telem::TimeSpan(0); // No timer
    config.spin_duration = 50 * telem::MICROSECOND; // 50us spin

    const auto loop = ASSERT_NIL_P(create(config));

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for((10 * telem::MICROSECOND).chrono());
    loop->wake();

    waiter.join();
    ASSERT_TRUE(woke_up.load());
}

/// @brief Test multiple create/destroy cycles.
TEST(LoopTest, MultipleCreateDestroy) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::MILLISECOND;

    for (int i = 0; i < 3; i++) {
        const auto loop = ASSERT_NIL_P(create(config));
        // Loop is cleaned up when it goes out of scope
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
        // Loop is cleaned up when it goes out of scope
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
    const Config cfg;
    EXPECT_EQ(cfg.mode, ExecutionMode::AUTO);
    const auto resolved = cfg.apply_defaults(10 * telem::MILLISECOND);
    EXPECT_NE(resolved.mode, ExecutionMode::AUTO);
}

TEST(ConfigTest, ApplyDefaultsSetsInterval) {
    const Config cfg;
    EXPECT_EQ(cfg.interval.nanoseconds(), 0);
    const auto resolved = cfg.apply_defaults(10 * telem::MILLISECOND);
    EXPECT_EQ(resolved.interval, 10 * telem::MILLISECOND);
}

TEST(ConfigTest, DefaultRtPriority) {
    const Config cfg;
    EXPECT_EQ(cfg.rt_priority, DEFAULT_RT_PRIORITY);
}

TEST(ConfigTest, AutoCpuAffinityPinsForRTEvent) {
    Config cfg;
    cfg.mode = ExecutionMode::RT_EVENT;
    EXPECT_EQ(cfg.cpu_affinity, CPU_AFFINITY_AUTO);
    const auto resolved = cfg.apply_defaults(500 * telem::MICROSECOND);
    if (std::thread::hardware_concurrency() > 1) {
        EXPECT_GE(resolved.cpu_affinity, 0);
    }
}

TEST(ConfigTest, AutoModeResolvesToRTEventGetsCpuPinning) {
    Config cfg;
    cfg.mode = ExecutionMode::AUTO;
    cfg.cpu_affinity = CPU_AFFINITY_AUTO;
    const auto resolved = cfg.apply_defaults(500 * telem::MICROSECOND);
    if (xthread::has_rt_support() && std::thread::hardware_concurrency() > 1) {
        EXPECT_GE(resolved.cpu_affinity, 0);
    }
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

TEST(ConfigTest, HighRateModeWithoutIntervalGetsDefault) {
    Config cfg;
    cfg.mode = ExecutionMode::HIGH_RATE;
    cfg.interval = telem::TimeSpan(0);
    const auto resolved = cfg.apply_defaults(telem::TimeSpan::max());
    EXPECT_EQ(resolved.interval, timing::HIGH_RATE_POLL_INTERVAL);
}

TEST(ConfigTest, RTEventModeWithoutIntervalGetsDefault) {
    Config cfg;
    cfg.mode = ExecutionMode::RT_EVENT;
    cfg.interval = telem::TimeSpan(0);
    const auto resolved = cfg.apply_defaults(telem::TimeSpan::max());
    EXPECT_EQ(resolved.interval, timing::HIGH_RATE_POLL_INTERVAL);
}

TEST(ConfigTest, HighRateModeWithExplicitIntervalNotOverridden) {
    Config cfg;
    cfg.mode = ExecutionMode::HIGH_RATE;
    cfg.interval = 500 * telem::MICROSECOND;
    const auto resolved = cfg.apply_defaults(telem::TimeSpan::max());
    EXPECT_EQ(resolved.interval, 500 * telem::MICROSECOND);
}

TEST(ConfigOutputTest, OutputContainsMode) {
    Config cfg;
    cfg.mode = ExecutionMode::EVENT_DRIVEN;
    std::ostringstream os;
    os << cfg;
    EXPECT_NE(os.str().find("execution mode"), std::string::npos);
    EXPECT_NE(os.str().find("EVENT_DRIVEN"), std::string::npos);
}

TEST(ConfigOutputTest, OutputContainsIntervalWhenSet) {
    Config cfg;
    cfg.mode = ExecutionMode::HIGH_RATE;
    cfg.interval = 10 * telem::MILLISECOND;
    std::ostringstream os;
    os << cfg;
    EXPECT_NE(os.str().find("interval"), std::string::npos);
}

TEST(ConfigOutputTest, OutputOmitsIntervalWhenZero) {
    Config cfg;
    cfg.mode = ExecutionMode::EVENT_DRIVEN;
    cfg.interval = telem::TimeSpan(0);
    std::ostringstream os;
    os << cfg;
    EXPECT_EQ(os.str().find("interval"), std::string::npos);
}

TEST(ConfigOutputTest, HybridModeShowsSpinDuration) {
    Config cfg;
    cfg.mode = ExecutionMode::HYBRID;
    std::ostringstream os;
    os << cfg;
    EXPECT_NE(os.str().find("spin duration"), std::string::npos);
}

TEST(ConfigOutputTest, NonHybridModeOmitsSpinDuration) {
    Config cfg;
    cfg.mode = ExecutionMode::EVENT_DRIVEN;
    std::ostringstream os;
    os << cfg;
    EXPECT_EQ(os.str().find("spin duration"), std::string::npos);
}

TEST(ConfigOutputTest, RTEventShowsRtPriorityAndLockMemory) {
    Config cfg;
    cfg.mode = ExecutionMode::RT_EVENT;
    cfg.rt_priority = 80;
    cfg.lock_memory = true;
    std::ostringstream os;
    os << cfg;
    EXPECT_NE(os.str().find("rt priority"), std::string::npos);
    EXPECT_NE(os.str().find("80"), std::string::npos);
    EXPECT_NE(os.str().find("lock memory"), std::string::npos);
    EXPECT_NE(os.str().find("yes"), std::string::npos);
}

TEST(ConfigOutputTest, NonRTEventOmitsRtPriority) {
    Config cfg;
    cfg.mode = ExecutionMode::EVENT_DRIVEN;
    std::ostringstream os;
    os << cfg;
    EXPECT_EQ(os.str().find("rt priority"), std::string::npos);
}

TEST(ConfigOutputTest, OutputContainsCpuAffinityWhenSet) {
    Config cfg;
    cfg.mode = ExecutionMode::HIGH_RATE;
    cfg.cpu_affinity = 7;
    std::ostringstream os;
    os << cfg;
    EXPECT_NE(os.str().find("cpu affinity"), std::string::npos);
    EXPECT_NE(os.str().find("7"), std::string::npos);
}

TEST(ConfigOutputTest, OutputOmitsCpuAffinityWhenAuto) {
    Config cfg;
    cfg.mode = ExecutionMode::EVENT_DRIVEN;
    cfg.cpu_affinity = CPU_AFFINITY_AUTO;
    std::ostringstream os;
    os << cfg;
    EXPECT_EQ(os.str().find("cpu affinity"), std::string::npos);
}

/// @brief watch() should return true when given a valid notifier.
TEST(WatchTest, WatchReturnsTrue_ValidNotifier) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier = notify::create();
    EXPECT_TRUE(loop->watch(*notifier));
}

/// @brief wait() should return when a watched notifier is signaled.
TEST(WatchTest, WatchWakesWait_NotifierSignaled) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier = notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    EXPECT_FALSE(woke_up.load());

    notifier->signal();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
}

/// @brief Both wake() and watched notifier should wake wait().
TEST(WatchTest, WatchAndWake_BothWork) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier = notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    breaker::Breaker breaker;

    std::atomic<int> wake_count{0};
    std::thread waiter1([&]() {
        loop->wait(breaker);
        wake_count.fetch_add(1);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    loop->wake();
    waiter1.join();

    std::thread waiter2([&]() {
        loop->wait(breaker);
        wake_count.fetch_add(1);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    notifier->signal();
    waiter2.join();

    EXPECT_EQ(wake_count.load(), 2);
}

/// @brief Timer and watch should work together.
TEST(WatchTest, WatchAndTimer_BothWork) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 50 * telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier = notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    breaker::Breaker breaker;

    const auto sw = telem::Stopwatch();
    loop->wait(breaker);
    const auto elapsed = sw.elapsed();
    EXPECT_GE(elapsed, 25 * telem::MILLISECOND);
    EXPECT_LE(elapsed, test_timing::EVENT_DRIVEN_BOUND);

    std::atomic<bool> woke_up{false};
    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for((10 * telem::MILLISECOND).chrono());
    notifier->signal();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
}

#if defined(__linux__) || defined(__APPLE__)

/// @brief Multiple notifiers should be watchable simultaneously (Linux/macOS only).
TEST(WatchTest, WatchMultipleNotifiers) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier1 = notify::create();
    auto notifier2 = notify::create();
    ASSERT_TRUE(loop->watch(*notifier1));
    ASSERT_TRUE(loop->watch(*notifier2));

    breaker::Breaker breaker;

    std::atomic<bool> woke_up{false};
    std::thread waiter1([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    notifier1->signal();
    waiter1.join();
    EXPECT_TRUE(woke_up.load());

    woke_up.store(false);
    std::thread waiter2([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    notifier2->signal();
    waiter2.join();
    EXPECT_TRUE(woke_up.load());
}

/// @brief watch() should be idempotent - calling twice with same notifier succeeds.
TEST(WatchTest, WatchSameNotifierTwice_Succeeds) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier = notify::create();
    EXPECT_TRUE(loop->watch(*notifier));
    EXPECT_TRUE(loop->watch(*notifier)); // Should succeed, not fail with EEXIST
}

/// @brief watch() called twice should still allow notifier to wake wait().
TEST(WatchTest, WatchSameNotifierTwice_StillWakes) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier = notify::create();
    ASSERT_TRUE(loop->watch(*notifier));
    ASSERT_TRUE(loop->watch(*notifier)); // Re-register

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    notifier->signal();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
}

/// @brief Simulates runtime restart: watch, use, then watch again on same notifier.
TEST(WatchTest, WatchAfterSimulatedRestart_Works) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier = notify::create();

    // First "run" - watch and use
    ASSERT_TRUE(loop->watch(*notifier));
    breaker::Breaker breaker1;
    std::thread t1([&]() { loop->wait(breaker1); });
    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    loop->wake();
    t1.join();

    // Second "run" - watch same notifier again (simulates restart scenario)
    ASSERT_TRUE(loop->watch(*notifier));
    breaker::Breaker breaker2;
    std::atomic<bool> woke{false};
    std::thread t2([&]() {
        loop->wait(breaker2);
        woke.store(true);
    });
    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    notifier->signal();
    t2.join();

    EXPECT_TRUE(woke.load());
}

#endif // defined(__linux__) || defined(__APPLE__)

#if defined(_WIN32)

/// @brief watch() should fail for a second notifier on Windows (only one supported).
TEST(WatchTest, WatchSecondNotifierFails_Windows) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier1 = notify::create();
    auto notifier2 = notify::create();
    EXPECT_TRUE(loop->watch(*notifier1));
    EXPECT_FALSE(loop->watch(*notifier2));
}

#endif // defined(_WIN32)

//
// Breaker Cancellation Tests
//

/// @brief BUSY_WAIT mode should exit quickly when breaker stops.
TEST(BreakerCancellationTest, BreakerStop_BusyWaitExits) {
    Config config;
    config.mode = ExecutionMode::BUSY_WAIT;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;
    breaker.start();

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());

    const auto sw = telem::Stopwatch();
    breaker.stop();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
    EXPECT_LE(sw.elapsed(), test_timing::BREAKER_STOP_LATENCY);
}

/// @brief HYBRID mode should exit when breaker stops during spin or block phase.
TEST(BreakerCancellationTest, BreakerStop_HybridModeExits) {
    Config config;
    config.mode = ExecutionMode::HYBRID;
    config.interval = telem::TimeSpan(0);
    config.spin_duration = 50 * telem::MICROSECOND;

    const auto loop = ASSERT_NIL_P(create(config));

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;
    breaker.start();

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());

    const auto sw = telem::Stopwatch();
    breaker.stop();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
    EXPECT_LE(sw.elapsed(), test_timing::THREAD_STARTUP);
}

/// @brief EVENT_DRIVEN mode uses 100ms timeout; wait() returns within that window.
TEST(BreakerCancellationTest, EventDriven_ReturnsWithinTimeout) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    breaker::Breaker breaker;

    const auto sw = telem::Stopwatch();
    loop->wait(breaker);

    // EVENT_DRIVEN uses 100ms timeout, allow some margin
    EXPECT_LE(sw.elapsed(), test_timing::EVENT_DRIVEN_BOUND);
}

/// @brief wake() should immediately unblock a waiting thread.
TEST(WakeTest, Wake_UnblocksWait) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    std::atomic<bool> woke_up{false};
    breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    EXPECT_FALSE(woke_up.load());

    const auto sw = telem::Stopwatch();
    loop->wake();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
    EXPECT_LE(sw.elapsed(), test_timing::THREAD_STARTUP);
}

TEST(WakeReasonTest, ReturnsTimerOnTimerFire) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 10 * telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create(config));

    breaker::Breaker breaker;
    breaker.start();

    const auto reason = loop->wait(breaker);
    ASSERT_EQ(reason, WakeReason::Timer);

    breaker.stop();
}

TEST(WakeReasonTest, ReturnsInputOnNotifierSignal) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier = notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    breaker::Breaker breaker;
    breaker.start();

    std::atomic<WakeReason> reason{WakeReason::Shutdown};
    std::thread waiter([&]() { reason.store(loop->wait(breaker)); });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    notifier->signal();
    waiter.join();

    ASSERT_EQ(reason.load(), WakeReason::Input);

    breaker.stop();
}

TEST(WakeReasonTest, DistinguishesTimerFromInputWhenBothConfigured) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = 100 * telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create(config));

    auto notifier = notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    breaker::Breaker breaker;
    breaker.start();

    std::atomic<WakeReason> reason{WakeReason::Shutdown};
    std::thread waiter([&]() { reason.store(loop->wait(breaker)); });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    notifier->signal();
    waiter.join();

    ASSERT_EQ(reason.load(), WakeReason::Input);

    reason.store(WakeReason::Shutdown);
    const auto wait_reason = loop->wait(breaker);
    ASSERT_EQ(wait_reason, WakeReason::Timer);

    breaker.stop();
}
