// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <sstream>
#include <thread>
#include <tuple>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/notify/notify.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/loop/loop.h"
#if defined(__linux__)
#include "arc/cpp/runtime/loop/loop_linux.h"
#elif defined(__APPLE__)
#include "arc/cpp/runtime/loop/loop_darwin.h"
#elif defined(_WIN32)
#include "arc/cpp/runtime/loop/loop_windows.h"
#endif

namespace arc::runtime::loop {

/// @brief Creates a loop and starts it, returning the started loop. The loop does not
/// pin the thread that starts it, so tests do not share one core.
std::pair<std::unique_ptr<Loop>, x::errors::Error> create_and_start(Config cfg) {
    cfg.cpu_affinity = CPU_AFFINITY_NONE;
    auto loop = create(cfg);
    if (auto start_err = loop->start(); start_err) return {nullptr, start_err};
    return {std::move(loop), x::errors::NIL};
}

/// @brief Test timing constants.
namespace test_timing {
/// @brief Time to wait for a thread to start waiting before signaling.
const auto THREAD_STARTUP = 50 * x::telem::MILLISECOND;
/// @brief Expected timer bounds (lower).
const auto TIMER_LOWER_BOUND = 5 * x::telem::MILLISECOND;
/// @brief Expected timer bounds (upper, accounts for system jitter).
const auto TIMER_UPPER_BOUND = 50 * x::telem::MILLISECOND;
/// @brief Maximum latency from wake() to wait() returning. Covers the scheduler putting
/// both threads back on a core, which takes a time slice or two on a loaded machine.
const auto WAKE_LATENCY = 50 * x::telem::MILLISECOND;
/// @brief Maximum time for breaker stop to take effect.
const auto BREAKER_STOP_LATENCY = 10 * x::telem::MILLISECOND;
/// @brief Maximum time for event-driven timeout (100ms + margin).
const auto EVENT_DRIVEN_BOUND = 150 * x::telem::MILLISECOND;
/// @brief Wait duration of the deadline spec.
const auto DEADLINE_DURATION = 10 * x::telem::MILLISECOND;
/// @brief Maximum median distance between a fire and its deadline.
const auto FIRE_ERROR_BOUND = x::telem::MILLISECOND;
/// @brief Maximum median distance between a fire and a deadline under 10 ms.
const auto SHORT_FIRE_ERROR_BOUND = 250 * x::telem::MICROSECOND;
/// @brief Maximum median distance between a fire and its deadline in a mode that spins
/// to the deadline.
const auto SPIN_FIRE_ERROR_BOUND = 50 * x::telem::MICROSECOND;
/// @brief Deadline of a wait that an input ends before the deadline. It is shorter than
/// the block timeout of each mode, so that a stale timer fires inside the next wait.
const auto STALE_DEADLINE = 5 * x::telem::MILLISECOND;
}

/// @brief Test that Loop can be created.
TEST(LoopTest, Create) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::MILLISECOND;

    const auto loop = create(config);
    ASSERT_NE(loop, nullptr);
}

/// @brief Test that Loop can be created and destroyed.
TEST(LoopTest, CreateAndDestroy) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::MILLISECOND;
    const auto loop = create(config);
}

/// @brief Test that Loop wakes up on wake() call (EVENT_DRIVEN mode).
TEST(LoopTest, Wake_EventDriven) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    std::atomic<bool> woke_up{false};
    x::breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());

    loop->wake();

    waiter.join();
    ASSERT_TRUE(woke_up.load());
}

/// @brief Test that Loop wakes up on timer expiration.
TEST(LoopTest, TimerExpiration) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    x::breaker::Breaker breaker;

    const auto sw = x::telem::Stopwatch();
    loop->wait(breaker, 10 * x::telem::MILLISECOND);

    const auto elapsed = sw.elapsed();
    EXPECT_GE(elapsed, test_timing::TIMER_LOWER_BOUND);
    EXPECT_LE(elapsed, test_timing::TIMER_UPPER_BOUND);
}

/// @brief Test BUSY_WAIT mode responds quickly to wake().
TEST(LoopTest, BusyWaitMode) {
    Config config;
    config.mode = ExecutionMode::BUSY_WAIT;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    // Both threads read the same stopwatch to time the wake, so the window covers
    // wake() to wait() returning and leaves out thread startup and join scheduling.
    const auto sw = x::telem::Stopwatch();
    std::atomic<bool> waiting{false};
    std::atomic<int64_t> returned_at{0};
    x::breaker::Breaker breaker;
    breaker.start();

    std::thread waiter([&]() {
        waiting.store(true);
        loop->wait(breaker);
        returned_at.store(sw.elapsed().nanoseconds());
    });

    // wake() latches, so a signal landing just before the spin loop starts still ends
    // its first iteration.
    while (!waiting.load())
        std::this_thread::yield();

    const auto woke_at = sw.elapsed();
    loop->wake();
    waiter.join();
    breaker.stop();

    ASSERT_NE(returned_at.load(), 0);
    EXPECT_LE(
        x::telem::TimeSpan(returned_at.load()) - woke_at,
        test_timing::WAKE_LATENCY
    );
}

/// @brief Test HIGH_RATE mode with timer.
TEST(LoopTest, HighRateMode) {
    Config config;
    config.mode = ExecutionMode::HIGH_RATE;
    config.interval = 10 * x::telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    x::breaker::Breaker breaker;
    breaker.start();

    const auto sw = x::telem::Stopwatch();
    loop->wait(breaker);

    const auto elapsed = sw.elapsed();
    EXPECT_GE(elapsed, test_timing::TIMER_LOWER_BOUND);
    EXPECT_LE(elapsed, test_timing::TIMER_UPPER_BOUND);
    breaker.stop();
}

/// @brief Checks that a loop in mode with interval wakes on a 10 ms deadline.
void expect_wakes_on_deadline(
    const ExecutionMode mode,
    const x::telem::TimeSpan interval
) {
    Config config;
    config.mode = mode;
    config.interval = interval;
    const auto loop = ASSERT_NIL_P(create_and_start(config));
    x::breaker::Breaker breaker;
    breaker.start();

    const auto sw = x::telem::Stopwatch();
    EXPECT_EQ(loop->wait(breaker, 10 * x::telem::MILLISECOND), WakeReason::Timer);
    const auto elapsed = sw.elapsed();
    EXPECT_GE(elapsed, 10 * x::telem::MILLISECOND);
    EXPECT_LE(elapsed, test_timing::TIMER_UPPER_BOUND);
    breaker.stop();
}

/// @brief HIGH_RATE should wake on a deadline that comes before its interval.
TEST(LoopTest, HighRateMode_WakesOnDeadlineBeforeInterval) {
    expect_wakes_on_deadline(ExecutionMode::HIGH_RATE, 100 * x::telem::MILLISECOND);
}

/// @brief BUSY_WAIT should wake on a deadline.
TEST(LoopTest, BusyWaitMode_WakesOnDeadline) {
    expect_wakes_on_deadline(ExecutionMode::BUSY_WAIT, x::telem::TimeSpan(0));
}

/// @brief Test HYBRID mode behavior.
TEST(LoopTest, HybridMode) {
    Config config;
    config.mode = ExecutionMode::HYBRID;
    config.interval = x::telem::TimeSpan(0);
    config.spin_duration = 50 * x::telem::MICROSECOND;

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    std::atomic<bool> woke_up{false};
    x::breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for((10 * x::telem::MICROSECOND).chrono());
    loop->wake();

    waiter.join();
    ASSERT_TRUE(woke_up.load());
}

/// @brief Test multiple create/destroy cycles.
TEST(LoopTest, MultipleCreateDestroy) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::MILLISECOND;

    for (int i = 0; i < 3; i++) {
        const auto loop = ASSERT_NIL_P(create_and_start(config));
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
        config.interval = x::telem::MILLISECOND;
        const auto loop = ASSERT_NIL_P(create_and_start(config));
    }
}

TEST(ModeSelectorTest, NoTimers_SelectsAuto) {
    EXPECT_EQ(select_mode(x::telem::TimeSpan::max()), ExecutionMode::AUTO);
}

TEST(ModeSelectorTest, AboveRtEventThreshold_SelectsAuto) {
    EXPECT_EQ(select_mode(timing::RT_EVENT_THRESHOLD), ExecutionMode::AUTO);
    EXPECT_EQ(select_mode(100 * x::telem::MILLISECOND), ExecutionMode::AUTO);
}

TEST(ModeSelectorTest, NeverAutoselectsBusyWait) {
    EXPECT_NE(select_mode(10 * x::telem::MICROSECOND), ExecutionMode::BUSY_WAIT);
    EXPECT_NE(select_mode(x::telem::TimeSpan(0)), ExecutionMode::BUSY_WAIT);
}

TEST(ModeSelectorTest, Boundary_AtOneMs) {
    const auto expected = x::thread::rt::has_support() ? ExecutionMode::RT_EVENT
                                                       : ExecutionMode::AUTO;
    EXPECT_EQ(select_mode(x::telem::MILLISECOND), expected);
}

TEST(ConfigTest, ApplyDefaultsResolvesAutoForSubMillisecondTimers) {
    const Config cfg;
    EXPECT_EQ(cfg.mode, ExecutionMode::AUTO);
    const auto resolved = cfg.apply_defaults(500 * x::telem::MICROSECOND);
    EXPECT_EQ(
        resolved.mode,
        x::thread::rt::has_support() ? ExecutionMode::RT_EVENT
                                     : ExecutionMode::HIGH_RATE
    );
}

TEST(ConfigTest, ApplyDefaultsKeepsAutoForLongTimers) {
    const Config cfg;
    const auto resolved = cfg.apply_defaults(10 * x::telem::MILLISECOND);
    EXPECT_EQ(resolved.mode, ExecutionMode::AUTO);
}

TEST(ConfigTest, ApplyDefaultsSetsInterval) {
    const Config cfg;
    EXPECT_EQ(cfg.interval.nanoseconds(), 0);
    const auto resolved = cfg.apply_defaults(10 * x::telem::MILLISECOND);
    EXPECT_EQ(resolved.interval, 10 * x::telem::MILLISECOND);
}

TEST(ConfigTest, DefaultRtPriority) {
    const Config cfg;
    EXPECT_EQ(cfg.rt_priority, DEFAULT_RT_PRIORITY);
}

TEST(ConfigTest, AutoCpuAffinityPinsForRTEvent) {
    Config cfg;
    cfg.mode = ExecutionMode::RT_EVENT;
    EXPECT_EQ(cfg.cpu_affinity, CPU_AFFINITY_AUTO);
    const auto resolved = cfg.apply_defaults(500 * x::telem::MICROSECOND);
    if (std::thread::hardware_concurrency() > 1) {
        EXPECT_GE(resolved.cpu_affinity, 0);
    }
}

TEST(ConfigTest, AutoModeResolvesToRTEventGetsCpuPinning) {
    Config cfg;
    cfg.mode = ExecutionMode::AUTO;
    cfg.cpu_affinity = CPU_AFFINITY_AUTO;
    const auto resolved = cfg.apply_defaults(500 * x::telem::MICROSECOND);
    if (x::thread::rt::has_support() && std::thread::hardware_concurrency() > 1) {
        EXPECT_GE(resolved.cpu_affinity, 0);
    }
}

TEST(ConfigTest, ExplicitCpuAffinityNotOverridden) {
    Config cfg;
    cfg.mode = ExecutionMode::RT_EVENT;
    cfg.cpu_affinity = 0;
    const auto resolved = cfg.apply_defaults(500 * x::telem::MICROSECOND);
    EXPECT_EQ(resolved.cpu_affinity, 0);
}

TEST(ConfigTest, ExplicitModeNotOverridden) {
    Config cfg;
    cfg.mode = ExecutionMode::BUSY_WAIT;
    const auto resolved = cfg.apply_defaults(10 * x::telem::MILLISECOND);
    EXPECT_EQ(resolved.mode, ExecutionMode::BUSY_WAIT);
}

TEST(ConfigTest, HighRateModeWithoutIntervalGetsDefault) {
    Config cfg;
    cfg.mode = ExecutionMode::HIGH_RATE;
    cfg.interval = x::telem::TimeSpan(0);
    const auto resolved = cfg.apply_defaults(x::telem::TimeSpan::max());
    EXPECT_EQ(resolved.interval, timing::HIGH_RATE_POLL_INTERVAL);
}

TEST(ConfigTest, RTEventModeWithoutIntervalGetsDefault) {
    Config cfg;
    cfg.mode = ExecutionMode::RT_EVENT;
    cfg.interval = x::telem::TimeSpan(0);
    const auto resolved = cfg.apply_defaults(x::telem::TimeSpan::max());
    EXPECT_EQ(resolved.interval, timing::HIGH_RATE_POLL_INTERVAL);
}

TEST(ConfigTest, HighRateModeWithExplicitIntervalNotOverridden) {
    Config cfg;
    cfg.mode = ExecutionMode::HIGH_RATE;
    cfg.interval = 500 * x::telem::MICROSECOND;
    const auto resolved = cfg.apply_defaults(x::telem::TimeSpan::max());
    EXPECT_EQ(resolved.interval, 500 * x::telem::MICROSECOND);
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
    cfg.interval = 10 * x::telem::MILLISECOND;
    std::ostringstream os;
    os << cfg;
    EXPECT_NE(os.str().find("interval"), std::string::npos);
}

TEST(ConfigOutputTest, OutputOmitsIntervalWhenZero) {
    Config cfg;
    cfg.mode = ExecutionMode::EVENT_DRIVEN;
    cfg.interval = x::telem::TimeSpan(0);
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

TEST(ConfigOutputTest, RTEventShowsRtPriorityAndMemoryLocked) {
    Config cfg;
    cfg.mode = ExecutionMode::RT_EVENT;
    cfg.rt_priority = 80;
    cfg.memory_locked = true;
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
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();
    EXPECT_TRUE(loop->watch(*notifier));
}

/// @brief wait() should return when a watched notifier is signaled.
TEST(WatchTest, WatchWakesWait_NotifierSignaled) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    std::atomic<bool> woke_up{false};
    x::breaker::Breaker breaker;

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
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    x::breaker::Breaker breaker;

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

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    x::breaker::Breaker breaker;

    const auto sw = x::telem::Stopwatch();
    loop->wait(breaker, 50 * x::telem::MILLISECOND);
    const auto elapsed = sw.elapsed();
    EXPECT_GE(elapsed, 25 * x::telem::MILLISECOND);
    EXPECT_LE(elapsed, test_timing::EVENT_DRIVEN_BOUND);

    std::atomic<bool> woke_up{false};
    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for((10 * x::telem::MILLISECOND).chrono());
    notifier->signal();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
}

#if defined(__linux__) || defined(__APPLE__)

/// @brief Multiple notifiers should be watchable simultaneously (Linux/macOS only).
TEST(WatchTest, WatchMultipleNotifiers) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier1 = x::notify::create();
    auto notifier2 = x::notify::create();
    ASSERT_TRUE(loop->watch(*notifier1));
    ASSERT_TRUE(loop->watch(*notifier2));

    x::breaker::Breaker breaker;

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

#endif // defined(__linux__) || defined(__APPLE__)

#if defined(_WIN32)

/// @brief watch() should fail for a second notifier on Windows (only one supported).
TEST(WatchTest, WatchSecondNotifierFails_Windows) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier1 = x::notify::create();
    auto notifier2 = x::notify::create();
    EXPECT_TRUE(loop->watch(*notifier1));
    EXPECT_FALSE(loop->watch(*notifier2));
}

#endif // defined(_WIN32)

/// @brief watch() should be idempotent - calling twice with same notifier succeeds.
TEST(WatchTest, WatchSameNotifierTwice_Succeeds) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();
    EXPECT_TRUE(loop->watch(*notifier));
    EXPECT_TRUE(loop->watch(*notifier));
}

/// @brief watch() called twice should still allow notifier to wake wait().
TEST(WatchTest, WatchSameNotifierTwice_StillWakes) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();
    ASSERT_TRUE(loop->watch(*notifier));
    ASSERT_TRUE(loop->watch(*notifier));

    std::atomic<bool> woke_up{false};
    x::breaker::Breaker breaker;

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
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();

    ASSERT_TRUE(loop->watch(*notifier));
    x::breaker::Breaker breaker1;
    std::thread t1([&]() { loop->wait(breaker1); });
    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    loop->wake();
    t1.join();

    ASSERT_TRUE(loop->watch(*notifier));
    x::breaker::Breaker breaker2;
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

/// @brief BUSY_WAIT mode should exit quickly when breaker stops.
TEST(BreakerCancellationTest, BreakerStop_BusyWaitExits) {
    Config config;
    config.mode = ExecutionMode::BUSY_WAIT;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    std::atomic<bool> woke_up{false};
    x::breaker::Breaker breaker;
    breaker.start();

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());

    const auto sw = x::telem::Stopwatch();
    breaker.stop();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
    EXPECT_LE(sw.elapsed(), test_timing::BREAKER_STOP_LATENCY);
}

/// @brief HIGH_RATE should exit when the breaker stops during a long sleep.
TEST(BreakerCancellationTest, BreakerStop_HighRateModeExits) {
    Config config;
    config.mode = ExecutionMode::HIGH_RATE;
    config.interval = 10 * x::telem::SECOND;

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    WakeReason reason = WakeReason::Timer;
    x::breaker::Breaker breaker;
    breaker.start();

    std::thread waiter([&] { reason = loop->wait(breaker, 10 * x::telem::SECOND); });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());

    const auto sw = x::telem::Stopwatch();
    breaker.stop();
    waiter.join();

    EXPECT_EQ(reason, WakeReason::Shutdown);
    EXPECT_LE(sw.elapsed(), test_timing::BREAKER_STOP_LATENCY);
}

/// @brief HYBRID mode should exit when breaker stops during spin or block phase.
TEST(BreakerCancellationTest, BreakerStop_HybridModeExits) {
    Config config;
    config.mode = ExecutionMode::HYBRID;
    config.interval = x::telem::TimeSpan(0);
    config.spin_duration = 50 * x::telem::MICROSECOND;

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    std::atomic<bool> woke_up{false};
    x::breaker::Breaker breaker;
    breaker.start();

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());

    const auto sw = x::telem::Stopwatch();
    breaker.stop();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
    EXPECT_LE(sw.elapsed(), test_timing::THREAD_STARTUP);
}

/// @brief EVENT_DRIVEN mode uses 100ms timeout; wait() returns within that window.
TEST(BreakerCancellationTest, EventDriven_ReturnsWithinTimeout) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    x::breaker::Breaker breaker;

    const auto sw = x::telem::Stopwatch();
    loop->wait(breaker);

    EXPECT_LE(sw.elapsed(), test_timing::EVENT_DRIVEN_BOUND);
}

/// @brief wake() should immediately unblock a waiting thread.
TEST(WakeTest, Wake_UnblocksWait) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    std::atomic<bool> woke_up{false};
    x::breaker::Breaker breaker;

    std::thread waiter([&]() {
        loop->wait(breaker);
        woke_up.store(true);
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    EXPECT_FALSE(woke_up.load());

    const auto sw = x::telem::Stopwatch();
    loop->wake();
    waiter.join();

    EXPECT_TRUE(woke_up.load());
    EXPECT_LE(sw.elapsed(), test_timing::THREAD_STARTUP);
}

TEST(WakeReasonTest, ReturnsTimerOnTimerFire) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    x::breaker::Breaker breaker;
    breaker.start();

    const auto reason = loop->wait(breaker, 10 * x::telem::MILLISECOND);
    ASSERT_EQ(reason, WakeReason::Timer);

    breaker.stop();
}

TEST(WakeReasonTest, ReturnsInputOnNotifierSignal) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    x::breaker::Breaker breaker;
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
    const auto deadline = 100 * x::telem::MILLISECOND;

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    x::breaker::Breaker breaker;
    breaker.start();

    std::atomic<WakeReason> reason{WakeReason::Shutdown};
    std::thread waiter([&]() { reason.store(loop->wait(breaker, deadline)); });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    notifier->signal();
    waiter.join();

    ASSERT_EQ(reason.load(), WakeReason::Input);

    reason.store(WakeReason::Shutdown);
    const auto wait_reason = loop->wait(breaker, deadline);
    ASSERT_EQ(wait_reason, WakeReason::Timer);

    breaker.stop();
}

/// @brief EVENT_DRIVEN with max_timeout should wake after max_timeout.
TEST(MaxTimeoutTest, EventDriven_WakesAfterMaxTimeout) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    x::breaker::Breaker breaker;

    const auto sw = x::telem::Stopwatch();
    const auto reason = loop->wait(breaker, 20 * x::telem::MILLISECOND);

    const auto elapsed = sw.elapsed();
    EXPECT_GE(elapsed, 15 * x::telem::MILLISECOND);
    EXPECT_LE(elapsed, test_timing::TIMER_UPPER_BOUND);
    EXPECT_EQ(reason, WakeReason::Timer);
}

/// @brief Input arriving before max_timeout should wake immediately.
TEST(MaxTimeoutTest, EventDriven_InputWakesBeforeMaxTimeout) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    auto notifier = x::notify::create();
    ASSERT_TRUE(loop->watch(*notifier));

    x::breaker::Breaker breaker;

    std::atomic<WakeReason> reason{WakeReason::Shutdown};
    std::thread waiter([&]() {
        reason.store(loop->wait(breaker, 500 * x::telem::MILLISECOND));
    });

    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    notifier->signal();
    waiter.join();

    EXPECT_EQ(reason.load(), WakeReason::Input);
}

/// @brief HYBRID mode with max_timeout should use it for the blocking phase.
TEST(MaxTimeoutTest, Hybrid_MaxTimeoutConstrainsBlockPhase) {
    Config config;
    config.mode = ExecutionMode::HYBRID;
    config.interval = x::telem::TimeSpan(0);
    config.spin_duration = 50 * x::telem::MICROSECOND;

    const auto loop = ASSERT_NIL_P(create_and_start(config));

    x::breaker::Breaker breaker;
    breaker.start();

    const auto sw = x::telem::Stopwatch();
    const auto reason = loop->wait(breaker, 20 * x::telem::MILLISECOND);

    const auto elapsed = sw.elapsed();
    EXPECT_GE(elapsed, 15 * x::telem::MILLISECOND);
    EXPECT_LE(elapsed, test_timing::TIMER_UPPER_BOUND);
    // A deadline wake is a timer on Windows and Linux and a timeout on macOS.
    EXPECT_TRUE(reason == WakeReason::Timer || reason == WakeReason::Timeout);

    breaker.stop();
}

/// @brief How far past its deadline one wait fires, and how the waits to it ended.
struct Fire {
    /// @brief how far past the deadline the last wait returned.
    x::telem::TimeSpan error;
    /// @brief how many waits did not end on an input.
    int waits = 0;
    /// @brief how many waits an input ended.
    int inputs = 0;
};

/// @brief Waits until duration passes on sw, as the runtime does for a timer of span.
Fire measure_fire(
    Loop &loop,
    x::breaker::Breaker &breaker,
    const x::telem::Stopwatch &sw,
    const x::telem::TimeSpan duration,
    const x::telem::TimeSpan span = x::telem::TimeSpan::max()
) {
    auto elapsed = sw.elapsed();
    Fire fire;
    while (elapsed < duration) {
        if (loop.wait(breaker, duration - elapsed, span) == WakeReason::Input)
            fire.inputs++;
        else
            fire.waits++;
        elapsed = sw.elapsed();
    }
    fire.error = elapsed - duration;
    return fire;
}

/// @brief Returns the median of values.
template<typename T>
T median_of(std::vector<T> values) {
    std::sort(values.begin(), values.end());
    return values[values.size() / 2];
}

/// @brief Returns the median error and wait count of 50 fires of DEADLINE_DURATION.
Fire median_fire(Loop &loop, x::breaker::Breaker &breaker) {
    constexpr int COUNT = 50;
    std::vector<x::telem::TimeSpan> errors;
    std::vector<int> waits;
    for (int i = 0; i < COUNT; i++) {
        const auto fire = measure_fire(
            loop,
            breaker,
            x::telem::Stopwatch(),
            test_timing::DEADLINE_DURATION
        );
        errors.push_back(fire.error);
        waits.push_back(fire.waits);
    }
    return {.error = median_of(errors), .waits = median_of(waits)};
}

/// @brief EVENT_DRIVEN should fire a 10 ms wait on its deadline.
TEST(DeadlineTest, EventDriven_FiresOnDeadline) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    const auto loop = ASSERT_NIL_P(create_and_start(config));
    x::breaker::Breaker breaker;

    const auto fire = median_fire(*loop, breaker);
    EXPECT_LE(fire.error, test_timing::FIRE_ERROR_BOUND);
    EXPECT_EQ(fire.waits, 1);
}

/// @brief HIGH_RATE should sleep to a deadline before its interval and fire on it.
TEST(DeadlineTest, HighRate_FiresOnDeadline) {
    Config config;
    config.mode = ExecutionMode::HIGH_RATE;
    config.interval = 100 * x::telem::MILLISECOND;
    const auto loop = ASSERT_NIL_P(create_and_start(config));
    x::breaker::Breaker breaker;
    breaker.start();

    const auto fire = median_fire(*loop, breaker);
    breaker.stop();
    EXPECT_LE(fire.error, test_timing::SPIN_FIRE_ERROR_BOUND);
    EXPECT_EQ(fire.waits, 1);
}

/// @brief HIGH_RATE should fire a 10 ms wait on its deadline with a 100 us interval.
TEST(DeadlineTest, HighRate_ShortIntervalFiresOnDeadline) {
    Config config;
    config.mode = ExecutionMode::HIGH_RATE;
    config.interval = timing::HIGH_RATE_POLL_INTERVAL;
    const auto loop = ASSERT_NIL_P(create_and_start(config));
    x::breaker::Breaker breaker;
    breaker.start();

    const auto fire = median_fire(*loop, breaker);
    breaker.stop();
    EXPECT_LE(fire.error, test_timing::SPIN_FIRE_ERROR_BOUND);
}

/// @brief A loop in the mode of the parameter.
class StaleDeadlineTest : public testing::TestWithParam<ExecutionMode> {
protected:
    std::unique_ptr<Loop> loop;
    std::unique_ptr<x::notify::Notifier> notifier;
    x::breaker::Breaker breaker;

    void SetUp() override {
        Config config;
        config.mode = this->GetParam();
        this->loop = ASSERT_NIL_P(create_and_start(config));
        this->notifier = x::notify::create();
        ASSERT_TRUE(this->loop->watch(*this->notifier));
        this->breaker.start();
    }

    void TearDown() override { this->breaker.stop(); }

    /// @brief Runs a wait with a deadline that an input ends at once.
    void end_deadline_with_input() {
        this->notifier->signal();
        ASSERT_EQ(
            this->loop->wait(this->breaker, test_timing::STALE_DEADLINE),
            WakeReason::Input
        );
    }
};

/// @brief A wait with no deadline should not wake on the deadline of an earlier wait.
TEST_P(StaleDeadlineTest, PendingDeadlineDoesNotWakeTheNextWait) {
    this->end_deadline_with_input();
    EXPECT_EQ(this->loop->wait(this->breaker), WakeReason::Timeout);
}

/// @brief A wait with no deadline should not wake on a deadline that passed before it.
TEST_P(StaleDeadlineTest, PassedDeadlineDoesNotWakeTheNextWait) {
    this->end_deadline_with_input();
    std::this_thread::sleep_for((2 * test_timing::STALE_DEADLINE).chrono());
    EXPECT_EQ(this->loop->wait(this->breaker), WakeReason::Timeout);
}

INSTANTIATE_TEST_SUITE_P(
    Modes,
    StaleDeadlineTest,
    testing::Values(
        ExecutionMode::AUTO,
        ExecutionMode::EVENT_DRIVEN,
        ExecutionMode::HYBRID
    )
);

/// @brief In AUTO, a deadline that an input ends should not wake a later wait that
/// spins to its deadline.
TEST(StaleDeadlineTest, PendingDeadlineDoesNotWakeASpinningWait) {
    Config config;
    config.mode = ExecutionMode::AUTO;
    const auto loop = ASSERT_NIL_P(create_and_start(config));
    const auto notifier = x::notify::create();
    ASSERT_TRUE(loop->watch(*notifier));
    x::breaker::Breaker breaker;
    breaker.start();

    notifier->signal();
    ASSERT_EQ(
        loop->wait(breaker, test_timing::STALE_DEADLINE, x::telem::TimeSpan::max()),
        WakeReason::Input
    );
    const auto deadline = 4 * test_timing::STALE_DEADLINE;
    const auto sw = x::telem::Stopwatch();
    loop->wait(breaker, deadline, x::telem::MILLISECOND);
    EXPECT_GE(sw.elapsed(), deadline);
    breaker.stop();
}

/// @brief A loop in the mode of the parameter, with an interval shorter than any block
/// timeout. The runtime sets the interval to the shortest timer span of a program.
class IntervalTest : public testing::TestWithParam<ExecutionMode> {
protected:
    const x::telem::TimeSpan INTERVAL = 4 * x::telem::MILLISECOND;
    std::unique_ptr<Loop> loop;
    x::breaker::Breaker breaker;

    void SetUp() override {
        Config config;
        config.mode = this->GetParam();
        config.interval = this->INTERVAL;
        this->loop = ASSERT_NIL_P(create_and_start(config));
        this->breaker.start();
    }

    void TearDown() override { this->breaker.stop(); }
};

/// @brief A wait with no deadline should not wake on the interval.
TEST_P(IntervalTest, WaitWithNoDeadlineDoesNotWakeOnTheInterval) {
    const auto sw = x::telem::Stopwatch();
    EXPECT_EQ(this->loop->wait(this->breaker), WakeReason::Timeout);
    EXPECT_GE(sw.elapsed(), 2 * this->INTERVAL);
}

/// @brief A wait with a deadline past the interval should wake on the deadline.
TEST_P(IntervalTest, WaitWakesOnTheDeadlineNotTheInterval) {
    const auto deadline = 5 * this->INTERVAL;
    const auto sw = x::telem::Stopwatch();
    this->loop->wait(this->breaker, deadline);
    EXPECT_GE(sw.elapsed(), deadline);
}

INSTANTIATE_TEST_SUITE_P(
    Modes,
    IntervalTest,
    testing::Values(
        ExecutionMode::AUTO,
        ExecutionMode::EVENT_DRIVEN,
        ExecutionMode::HYBRID
    )
);

#if defined(__linux__) || defined(__APPLE__) || defined(_WIN32)
/// @brief Checks that 20 waits of DEADLINE_DURATION each return Timer, none early, and
/// that the median fires within FIRE_ERROR_BOUND.
void expect_timer_on_deadline(Loop &loop, x::breaker::Breaker &breaker) {
    constexpr int COUNT = 20;
    std::vector<x::telem::TimeSpan> errors;
    errors.reserve(COUNT);
    for (int i = 0; i < COUNT; i++) {
        const auto sw = x::telem::Stopwatch();
        ASSERT_EQ(
            loop.wait(breaker, test_timing::DEADLINE_DURATION),
            WakeReason::Timer
        );
        errors.push_back(sw.elapsed() - test_timing::DEADLINE_DURATION);
    }
    std::sort(errors.begin(), errors.end());
    EXPECT_GE(errors.front(), x::telem::TimeSpan(0));
    EXPECT_LE(errors[COUNT / 2], test_timing::FIRE_ERROR_BOUND);
}

/// @brief Checks that a signal of notifier ends a wait to a far deadline as Input.
void expect_input_ends_wait(
    Loop &loop,
    x::breaker::Breaker &breaker,
    x::notify::Notifier &notifier
) {
    WakeReason reason = WakeReason::Timeout;
    std::thread waiter([&] { reason = loop.wait(breaker, x::telem::SECOND); });
    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    const auto sw = x::telem::Stopwatch();
    notifier.signal();
    waiter.join();
    EXPECT_EQ(reason, WakeReason::Input);
    EXPECT_LE(sw.elapsed(), test_timing::WAKE_LATENCY);
}

/// @brief Timer calls whose arm fails the way the OS call does. On Linux, the loop
/// arms a timerfd because epoll_pwait2 is missing.
struct FailingArm {
#if defined(__linux__)
    static int wait(int, epoll_event *, int, const timespec &) {
        errno = ENOSYS;
        return -1;
    }
    static int set(int, const itimerspec &) {
        errno = EBADF;
        return -1;
    }
    static bool realtime() { return true; }
#elif defined(__APPLE__)
    static int set(int, const struct kevent &) {
        errno = ENOMEM;
        return -1;
    }
#else
    static HANDLE create_high_resolution() {
        return WaitableTimerApi::create_high_resolution();
    }
    static HANDLE create() { return WaitableTimerApi::create(); }
    static BOOL set(HANDLE, const LARGE_INTEGER &) {
        SetLastError(ERROR_INVALID_HANDLE);
        return FALSE;
    }
    static void raise_tick() {}
    static void lower_tick() {}
#endif
};

#if defined(__linux__)
using FailingArmLoop = Linux<FailingArm>;
#elif defined(__APPLE__)
using FailingArmLoop = Darwin<FailingArm>;
#else
using FailingArmLoop = Windows<FailingArm>;
#endif

/// @brief A loop whose timer never arms, in the mode of the parameter.
class FailedArmTest : public testing::TestWithParam<ExecutionMode> {
protected:
    std::unique_ptr<FailingArmLoop> loop;
    std::unique_ptr<x::notify::Notifier> notifier;
    x::breaker::Breaker breaker;

    void SetUp() override {
        Config config;
        config.mode = this->GetParam();
        config.cpu_affinity = CPU_AFFINITY_NONE;
        this->loop = std::make_unique<FailingArmLoop>(config);
        ASSERT_NIL(this->loop->start());
        this->notifier = x::notify::create();
        ASSERT_TRUE(this->loop->watch(*this->notifier));
        this->breaker.start();
    }

    void TearDown() override { this->breaker.stop(); }
};

/// @brief A wait should spin to its deadline when the timer does not arm.
TEST_P(FailedArmTest, SpinsToTheDeadline) {
    expect_timer_on_deadline(*this->loop, this->breaker);
}

/// @brief An input should end the spin to a deadline.
TEST_P(FailedArmTest, ReturnsInputDuringTheSpin) {
    expect_input_ends_wait(*this->loop, this->breaker, *this->notifier);
}

/// @brief A breaker stop should end the spin to a deadline.
TEST_P(FailedArmTest, ReturnsShutdownWhenTheBreakerStops) {
    WakeReason reason = WakeReason::Timeout;
    std::thread waiter([&] {
        reason = this->loop->wait(this->breaker, x::telem::SECOND);
    });
    std::this_thread::sleep_for(test_timing::THREAD_STARTUP.chrono());
    const auto sw = x::telem::Stopwatch();
    this->breaker.stop();
    waiter.join();
    EXPECT_EQ(reason, WakeReason::Shutdown);
    EXPECT_LE(sw.elapsed(), test_timing::BREAKER_STOP_LATENCY);
}

/// @brief A wait with no deadline should block until its timeout, not spin.
TEST_P(FailedArmTest, TimesOutWithNoDeadline) {
    EXPECT_EQ(this->loop->wait(this->breaker), WakeReason::Timeout);
}

/// @brief The modes that arm a timer for a deadline. macOS HYBRID and RT_EVENT block on
/// a kevent timeout, with no timer to arm.
const std::vector<ExecutionMode> ARMED_MODES = {
    ExecutionMode::AUTO,
    ExecutionMode::EVENT_DRIVEN,
#if !defined(__APPLE__)
    ExecutionMode::HYBRID,
    ExecutionMode::RT_EVENT,
#endif
};

INSTANTIATE_TEST_SUITE_P(ArmedModes, FailedArmTest, testing::ValuesIn(ARMED_MODES));

#if defined(__linux__)
/// @brief Linux calls that count timerfd arms and latency holds, and record the timeout
/// of each epoll_pwait2 wait. realtime_thread is what the thread reports as its policy,
/// and pwait2 is whether epoll_pwait2 exists.
struct CountingEpollApi {
    bool realtime_thread = true;
    bool pwait2 = true;
    int *arms = nullptr;
    std::vector<x::telem::TimeSpan> *timeouts = nullptr;
    int *holds = nullptr;

    int wait(
        const int epfd,
        epoll_event *events,
        const int max,
        const timespec &timeout
    ) const {
        if (this->timeouts != nullptr)
            this->timeouts->push_back(
                x::telem::TimeSpan(timeout.tv_sec * 1'000'000'000 + timeout.tv_nsec)
            );
        if (!this->pwait2) {
            errno = ENOSYS;
            return -1;
        }
        return EpollApi::wait(epfd, events, max, timeout);
    }

    int set(const int fd, const itimerspec &ts) const {
        (*this->arms)++;
        return EpollApi::set(fd, ts);
    }

    bool realtime() const { return this->realtime_thread; }

    int hold_latency() const {
        if (this->holds != nullptr) (*this->holds)++;
        return open("/dev/null", O_WRONLY | O_CLOEXEC);
    }
};

/// @brief A started loop on a thread that reports a real-time policy, in the mode of
/// the parameter.
class DirectWaitTest : public testing::TestWithParam<ExecutionMode> {
protected:
    int arms = 0;
    std::unique_ptr<Linux<CountingEpollApi>> loop;
    std::unique_ptr<x::notify::Notifier> notifier;
    x::breaker::Breaker breaker;

    void SetUp() override {
        Config config;
        config.mode = this->GetParam();
        config.cpu_affinity = CPU_AFFINITY_NONE;
        this->loop = std::make_unique<Linux<CountingEpollApi>>(
            config,
            nullptr,
            CountingEpollApi{.arms = &this->arms}
        );
        ASSERT_NIL(this->loop->start());
        this->notifier = x::notify::create();
        ASSERT_TRUE(this->loop->watch(*this->notifier));
        this->breaker.start();
    }

    void TearDown() override { this->breaker.stop(); }
};

/// @brief A real-time wait should block to its deadline with no timerfd, which wakes
/// through a priority 1 softirq thread on PREEMPT_RT.
TEST_P(DirectWaitTest, FiresOnTheDeadlineWithNoTimerfd) {
    ASSERT_NO_FATAL_FAILURE(expect_timer_on_deadline(*this->loop, this->breaker));
    EXPECT_EQ(this->arms, 0);
}

/// @brief An input should end a real-time wait before its deadline.
TEST_P(DirectWaitTest, ReturnsInputBeforeTheDeadline) {
    expect_input_ends_wait(*this->loop, this->breaker, *this->notifier);
}

/// @brief A real-time wait with no deadline should block until its timeout.
TEST_P(DirectWaitTest, TimesOutWithNoDeadline) {
    EXPECT_EQ(this->loop->wait(this->breaker), WakeReason::Timeout);
    EXPECT_EQ(this->arms, 0);
}

INSTANTIATE_TEST_SUITE_P(ArmedModes, DirectWaitTest, testing::ValuesIn(ARMED_MODES));

/// @brief Starts a loop with api in EVENT_DRIVEN mode and waits to one deadline.
void wait_one_deadline(const CountingEpollApi &api) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    Linux<CountingEpollApi> loop(config, nullptr, api);
    ASSERT_NIL(loop.start());
    x::breaker::Breaker breaker;
    breaker.start();
    EXPECT_EQ(loop.wait(breaker, test_timing::DEADLINE_DURATION), WakeReason::Timer);
    breaker.stop();
}

/// @brief A loop on a thread that is not real-time should arm a timerfd, as an epoll
/// timeout adds 0.1% of the wait as slack to such a thread.
TEST(DirectWaitFallbackTest, ArmsATimerfdOnANormalThread) {
    int arms = 0;
    wait_one_deadline(CountingEpollApi{.realtime_thread = false, .arms = &arms});
    EXPECT_GT(arms, 0);
}

/// @brief A loop should arm a timerfd on a kernel before 5.11, which has no
/// epoll_pwait2.
TEST(DirectWaitFallbackTest, ArmsATimerfdWithoutEpollPwait2) {
    int arms = 0;
    wait_one_deadline(CountingEpollApi{.pwait2 = false, .arms = &arms});
    EXPECT_GT(arms, 0);
}

/// @brief The blocks and latency holds of one wait.
struct Blocks {
    /// @brief the nonzero timeouts the wait blocked on.
    std::vector<x::telem::TimeSpan> timeouts;
    /// @brief how many times the loop held the wake latency of the cores.
    int holds = 0;
};

/// @brief Starts a loop in mode on a thread that reports realtime as its policy, waits
/// once to deadline for a timer of span, and records the wait in blocks.
void wait_blocks(
    Blocks &blocks,
    const ExecutionMode mode,
    const bool realtime,
    const x::telem::TimeSpan deadline,
    const x::telem::TimeSpan span = x::telem::TimeSpan::max()
) {
    int arms = 0;
    std::vector<x::telem::TimeSpan> timeouts;
    Config config;
    config.mode = mode;
    config.cpu_affinity = CPU_AFFINITY_NONE;
    Linux<CountingEpollApi> loop(
        config,
        nullptr,
        CountingEpollApi{
            .realtime_thread = realtime,
            .arms = &arms,
            .timeouts = &timeouts,
            .holds = &blocks.holds,
        }
    );
    ASSERT_NIL(loop.start());
    x::breaker::Breaker breaker;
    breaker.start();
    EXPECT_EQ(loop.wait(breaker, deadline, span), WakeReason::Timer);
    breaker.stop();
    for (const auto &timeout: timeouts)
        if (timeout.nanoseconds() > 0) blocks.timeouts.push_back(timeout);
}

/// @brief HYBRID should block until the spin span ahead of the deadline.
TEST(DeadlineSpinTest, HybridBlocksUntilTheSpinAheadOfTheDeadline) {
    Blocks blocks;
    wait_blocks(blocks, ExecutionMode::HYBRID, true, test_timing::DEADLINE_DURATION);
    EXPECT_EQ(
        blocks.timeouts,
        std::vector{test_timing::DEADLINE_DURATION - timing::LINUX_DEADLINE_SPIN}
    );
    EXPECT_EQ(blocks.holds, 0);
}

/// @brief The spin should take at most half the span of the timer, so that a real-time
/// thread leaves its core time in each period.
TEST(DeadlineSpinTest, SpinTakesAtMostHalfTheTimerSpan) {
    const auto span = 40 * x::telem::MICROSECOND;
    Blocks blocks;
    wait_blocks(
        blocks,
        ExecutionMode::HYBRID,
        true,
        test_timing::DEADLINE_DURATION,
        span
    );
    EXPECT_EQ(blocks.timeouts, std::vector{test_timing::DEADLINE_DURATION - span / 2});
}

/// @brief A wait shorter than the spin should spin with no block.
TEST(DeadlineSpinTest, HybridSpinsAShortWaitWithNoBlock) {
    Blocks blocks;
    wait_blocks(blocks, ExecutionMode::HYBRID, true, timing::LINUX_DEADLINE_SPIN / 2);
    EXPECT_TRUE(blocks.timeouts.empty());
}

/// @brief BUSY_WAIT on a real-time thread should block before the deadline, as the
/// kernel throttles a real-time thread that never sleeps, and hold the wake latency.
TEST(DeadlineSpinTest, BusyWaitBlocksOnARealtimeThread) {
    Blocks blocks;
    wait_blocks(blocks, ExecutionMode::BUSY_WAIT, true, test_timing::DEADLINE_DURATION);
    EXPECT_EQ(
        blocks.timeouts,
        std::vector{test_timing::DEADLINE_DURATION - timing::LINUX_DEADLINE_SPIN}
    );
    EXPECT_EQ(blocks.holds, 1);
}

/// @brief BUSY_WAIT on a thread that is not real-time should spin through the wait.
TEST(DeadlineSpinTest, BusyWaitSpinsOnANormalThread) {
    Blocks blocks;
    wait_blocks(
        blocks,
        ExecutionMode::BUSY_WAIT,
        false,
        test_timing::DEADLINE_DURATION
    );
    EXPECT_TRUE(blocks.timeouts.empty());
    EXPECT_EQ(blocks.holds, 0);
}

/// @brief A HYBRID loop on a thread that reports the policy of the parameter.
class HybridFireTest : public testing::TestWithParam<bool> {};

/// @brief HYBRID should spin to its deadline, whether it blocks on an epoll_pwait2
/// timeout or a timerfd.
TEST_P(HybridFireTest, FiresOnTheDeadline) {
    int arms = 0;
    Config config;
    config.mode = ExecutionMode::HYBRID;
    config.cpu_affinity = CPU_AFFINITY_NONE;
    Linux<CountingEpollApi> loop(
        config,
        nullptr,
        CountingEpollApi{.realtime_thread = this->GetParam(), .arms = &arms}
    );
    ASSERT_NIL(loop.start());
    x::breaker::Breaker breaker;
    breaker.start();
    const auto fire = median_fire(loop, breaker);
    breaker.stop();
    EXPECT_LE(fire.error, test_timing::SPIN_FIRE_ERROR_BOUND);
    EXPECT_EQ(fire.waits, 1);
}

INSTANTIATE_TEST_SUITE_P(Policies, HybridFireTest, testing::Bool());
#endif

#if defined(_WIN32)
/// @brief The arms of the timer and the raises and lowers of the system tick.
struct Calls {
    int arms = 0;
    int raises = 0;
    int lowers = 0;
};

/// @brief Timer calls that count arms and system tick changes. A standard timer is
/// the timer of Windows before 10 1803, which has no high-resolution timer.
struct CountingTimerApi {
    bool standard = false;
    Calls *calls = nullptr;

    HANDLE create_high_resolution() const {
        if (this->standard) return NULL;
        return WaitableTimerApi::create_high_resolution();
    }
    static HANDLE create() { return WaitableTimerApi::create(); }
    BOOL set(const HANDLE timer, const LARGE_INTEGER &due) const {
        this->calls->arms++;
        return WaitableTimerApi::set(timer, due);
    }
    void raise_tick() const { this->calls->raises++; }
    void lower_tick() const { this->calls->lowers++; }
};

/// @brief A deadline outside the spin span of every mode, so each mode arms it.
const auto TICK_DEADLINE = 20 * x::telem::MILLISECOND;

/// @brief A started loop on a standard timer, in the mode of the parameter.
class StandardTimerTest : public testing::TestWithParam<ExecutionMode> {
protected:
    Calls calls;
    std::unique_ptr<Windows<CountingTimerApi>> loop;
    x::breaker::Breaker breaker;

    void SetUp() override {
        Config config;
        config.mode = this->GetParam();
        this->loop = std::make_unique<Windows<CountingTimerApi>>(
            config,
            nullptr,
            CountingTimerApi{.standard = true, .calls = &this->calls}
        );
        ASSERT_NIL(this->loop->start());
        this->breaker.start();
    }

    void TearDown() override { this->breaker.stop(); }

    /// @brief returns at once from a wait with no deadline.
    WakeReason wait_without_deadline() {
        this->loop->wake();
        return this->loop->wait(this->breaker);
    }
};

/// @brief A wait with no deadline should leave the system tick alone.
TEST_P(StandardTimerTest, LeavesTheTickAloneWithNoDeadline) {
    this->wait_without_deadline();
    EXPECT_EQ(this->calls.raises, 0);
    EXPECT_EQ(this->calls.lowers, 0);
}

/// @brief A wait should reach its deadline on a standard timer.
TEST_P(StandardTimerTest, ReachesTheDeadline) {
    const auto sw = x::telem::Stopwatch();
    EXPECT_EQ(this->loop->wait(this->breaker, TICK_DEADLINE), WakeReason::Timer);
    EXPECT_GE(sw.elapsed(), TICK_DEADLINE);
}

/// @brief A deadline under one timer step away should spin with no arm, as the timer
/// cannot arm that close.
TEST_P(StandardTimerTest, SpinsToADeadlineUnderOneTimerStep) {
    const auto deadline = 50 * x::telem::NANOSECOND;
    const auto sw = x::telem::Stopwatch();
    EXPECT_EQ(this->loop->wait(this->breaker, deadline), WakeReason::Timer);
    EXPECT_GE(sw.elapsed(), deadline);
    EXPECT_EQ(this->calls.arms, 0);
}

/// @brief Waits with deadlines should raise the tick once and keep it raised.
TEST_P(StandardTimerTest, RaisesTheTickOnceForDeadlines) {
    this->loop->wait(this->breaker, TICK_DEADLINE);
    this->loop->wait(this->breaker, TICK_DEADLINE);
    EXPECT_EQ(this->calls.raises, 1);
    EXPECT_EQ(this->calls.lowers, 0);
}

/// @brief A wait with no deadline should lower a raised tick.
TEST_P(StandardTimerTest, LowersTheTickWithNoDeadline) {
    this->loop->wait(this->breaker, TICK_DEADLINE);
    this->wait_without_deadline();
    EXPECT_EQ(this->calls.raises, 1);
    EXPECT_EQ(this->calls.lowers, 1);
}

/// @brief Closing the loop should lower a raised tick.
TEST_P(StandardTimerTest, LowersTheTickOnClose) {
    this->loop->wait(this->breaker, TICK_DEADLINE);
    this->loop.reset();
    EXPECT_EQ(this->calls.raises, 1);
    EXPECT_EQ(this->calls.lowers, 1);
}

INSTANTIATE_TEST_SUITE_P(ArmedModes, StandardTimerTest, testing::ValuesIn(ARMED_MODES));

/// @brief Timer calls whose timer cannot be created.
struct UncreatableTimerApi {
    static HANDLE create_high_resolution() { return NULL; }
    static HANDLE create() {
        SetLastError(ERROR_NOT_ENOUGH_MEMORY);
        return NULL;
    }
    static BOOL set(HANDLE, const LARGE_INTEGER &) { return FALSE; }
    static void raise_tick() {}
    static void lower_tick() {}
};

/// @brief A failed start should close what it opened, so that a second start fails
/// again instead of reporting a loop with closed handles as started.
TEST(FailedStartTest, FailsAgainOnASecondStart) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    Windows<UncreatableTimerApi> loop(config);
    const std::string message = "Failed to create waitable timer: " +
                                std::to_string(ERROR_NOT_ENOUGH_MEMORY);
    EXPECT_EQ(loop.start().type, message);
    EXPECT_EQ(loop.start().type, message);
}

/// @brief A high-resolution timer should never change the system tick.
TEST(HighResolutionTimerTest, NeverRaisesTheTick) {
    Calls calls;
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    Windows<CountingTimerApi> loop(config, nullptr, CountingTimerApi{.calls = &calls});
    ASSERT_NIL(loop.start());
    x::breaker::Breaker breaker;
    breaker.start();
    EXPECT_EQ(loop.wait(breaker, TICK_DEADLINE), WakeReason::Timer);
    breaker.stop();
    EXPECT_EQ(calls.raises, 0);
}
#endif
#endif

/// @brief The shortest timer span of a program and a wait duration. The shortest span
/// resolves AUTO.
using ShortDeadlineCase = std::tuple<x::telem::TimeSpan, x::telem::TimeSpan>;

/// @brief Deadline timing of the loop that AUTO selects, for waits under 10 ms.
class ShortDeadlineTest : public testing::TestWithParam<ShortDeadlineCase> {
protected:
    std::unique_ptr<Loop> loop;
    x::breaker::Breaker breaker;
    x::telem::TimeSpan duration;

    void SetUp() override {
        const auto config = Config().apply_defaults(std::get<0>(this->GetParam()));
        this->duration = std::get<1>(this->GetParam());
        this->loop = ASSERT_NIL_P(create_and_start(config));
        this->breaker.start();
    }

    void TearDown() override { this->breaker.stop(); }
};

/// @brief A short wait should fire on its deadline.
TEST_P(ShortDeadlineTest, FiresOnDeadline) {
    constexpr int COUNT = 20;
    std::vector<x::telem::TimeSpan> errors;
    std::vector<int> waits;
    for (int i = 0; i < COUNT; i++) {
        const auto fire = measure_fire(
            *this->loop,
            this->breaker,
            x::telem::Stopwatch(),
            this->duration,
            this->duration
        );
        errors.push_back(fire.error);
        waits.push_back(fire.waits);
    }
    EXPECT_LE(median_of(errors), test_timing::SHORT_FIRE_ERROR_BOUND);
    EXPECT_EQ(median_of(waits), 1);
}

/// @brief A wait that an input ends close to its deadline should still fire on the
/// deadline.
TEST_P(ShortDeadlineTest, FiresOnDeadlineAfterInput) {
    constexpr int COUNT = 10;
    const auto total = 5 * x::telem::MILLISECOND;
    // The input arrives with the case duration left before the deadline.
    const auto input_at = total - this->duration;
    auto notifier = x::notify::create();
    ASSERT_TRUE(this->loop->watch(*notifier));
    std::vector<x::telem::TimeSpan> errors;
    std::vector<int> waits;
    int inputs = 0;
    for (int i = 0; i < COUNT; i++) {
        const auto sw = x::telem::Stopwatch();
        std::thread sender([&sw, &notifier, input_at] {
            while (sw.elapsed() < input_at)
                continue;
            notifier->signal();
        });
        const auto fire = measure_fire(*this->loop, this->breaker, sw, total, total);
        errors.push_back(fire.error);
        waits.push_back(fire.waits);
        inputs += fire.inputs;
        sender.join();
    }
    EXPECT_GT(inputs, 0);
    EXPECT_LE(median_of(errors), test_timing::SHORT_FIRE_ERROR_BOUND);
    EXPECT_EQ(median_of(waits), 1);
}

INSTANTIATE_TEST_SUITE_P(
    ShortestSpansAndDurations,
    ShortDeadlineTest,
    testing::Combine(
        testing::Values(
            x::telem::TimeSpan::max(),
            10 * x::telem::MILLISECOND,
            4 * x::telem::MILLISECOND,
            2 * x::telem::MILLISECOND
        ),
        testing::Values(
            200 * x::telem::MICROSECOND,
            500 * x::telem::MICROSECOND,
            900 * x::telem::MICROSECOND,
            x::telem::MILLISECOND,
            1500 * x::telem::MICROSECOND,
            2 * x::telem::MILLISECOND
        )
    )
);
}
