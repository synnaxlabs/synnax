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
#include <cstdint>
#include <cstdlib>
#include <iomanip>
#include <iostream>
#include <optional>
#include <thread>
#include <vector>

#ifndef _WIN32
#include <time.h>
#endif

#include "gtest/gtest.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/queue/spsc.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/loop/loop.h"
#ifdef _WIN32
#include "arc/cpp/runtime/loop/loop_windows.h"
#endif
#include "arc/cpp/runtime/testutil/timing.h"

namespace arc::runtime::loop {
namespace {
/// @brief Maximum median distance between a fire and its deadline on an idle machine.
const auto MEDIAN_BOUND = x::telem::MILLISECOND;
/// @brief Time a thread spins to find the longest stall the scheduler gives it.
const auto STALL_SPAN = 100 * x::telem::MILLISECOND;
/// @brief Sleeps measured for the reference spread.
constexpr int SLEEP_COUNT = 40;
/// @brief Most waits measured for one duration.
constexpr std::int64_t MAX_COUNT = 500;
/// @brief Fewest waits measured for one duration.
constexpr std::int64_t MIN_COUNT = 30;
/// @brief Time the waits of one duration can take in total.
const auto DURATION_BUDGET = x::telem::SECOND;
/// @brief Capacity of the queue that stands in for the input queue of the runtime.
constexpr size_t INPUT_CAPACITY = 1024;
/// @brief Wait durations measured on each loop.
const std::vector<x::telem::TimeSpan> DURATIONS = {
    x::telem::MILLISECOND,
    2 * x::telem::MILLISECOND,
    3 * x::telem::MILLISECOND,
    4 * x::telem::MILLISECOND,
    5 * x::telem::MILLISECOND,
    6 * x::telem::MILLISECOND,
    8 * x::telem::MILLISECOND,
    10 * x::telem::MILLISECOND,
    20 * x::telem::MILLISECOND,
    30 * x::telem::MILLISECOND,
    100 * x::telem::MILLISECOND,
};
/// @brief Short wait durations that find the floor of each loop.
const std::vector<x::telem::TimeSpan> SHORT_DURATIONS = {
    50 * x::telem::MICROSECOND,
    100 * x::telem::MICROSECOND,
    250 * x::telem::MICROSECOND,
    500 * x::telem::MICROSECOND,
    x::telem::MILLISECOND,
};

/// @brief Logs the Windows timer resolution, and if this machine can create the
/// high-resolution timer the loop asks for. A loop that falls back to a standard timer
/// shows here as a 1 ms current resolution.
void log_timers() {
#ifdef _WIN32
    using Query = LONG(WINAPI *)(PULONG, PULONG, PULONG);
    const auto ntdll = GetModuleHandleW(L"ntdll.dll");
// C4191 fires on every GetProcAddress cast.
#pragma warning(push)
#pragma warning(disable : 4191)
    const auto query = reinterpret_cast<Query>(
        GetProcAddress(ntdll, "NtQueryTimerResolution")
    );
#pragma warning(pop)
    if (query != nullptr) {
        ULONG min_100ns = 0, max_100ns = 0, current_100ns = 0;
        query(&min_100ns, &max_100ns, &current_100ns);
        std::cout << std::fixed << std::setprecision(4);
        std::cout << "  timer resolution: " << current_100ns / 10000.0
                  << " ms current, " << max_100ns / 10000.0 << " ms best, "
                  << min_100ns / 10000.0 << " ms default\n";
    }
    const HANDLE timer = WaitableTimerApi::create_high_resolution();
    if (timer == NULL) {
        std::cout << "  high-resolution timer: unavailable, error " << GetLastError()
                  << "\n";
        return;
    }
    std::cout << "  high-resolution timer: available\n";
    CloseHandle(timer);
#endif
}

/// @brief Spins for span and returns the longest gap between two clock reads. The gap
/// is the longest time the scheduler held the calling thread off its core.
x::telem::TimeSpan measure_stall(const x::telem::TimeSpan span) {
    const auto sw = x::telem::Stopwatch();
    auto last = sw.elapsed();
    auto longest = x::telem::TimeSpan(0);
    while (last < span) {
        const auto now = sw.elapsed();
        if (now - last > longest) longest = now - last;
        last = now;
    }
    return longest;
}

/// @brief Returns the CPU time the calling thread has used. Windows counts it in
/// scheduler ticks, so a short span reads coarse there.
x::telem::TimeSpan thread_cpu_time() {
#ifdef _WIN32
    FILETIME created, exited, kernel, user;
    GetThreadTimes(GetCurrentThread(), &created, &exited, &kernel, &user);
    const auto ticks = [](const FILETIME &t) {
        return (static_cast<std::int64_t>(t.dwHighDateTime) << 32) | t.dwLowDateTime;
    };
    return x::telem::TimeSpan((ticks(kernel) + ticks(user)) * 100);
#else
    timespec ts{};
    clock_gettime(CLOCK_THREAD_CPUTIME_ID, &ts);
    return x::telem::TimeSpan(ts.tv_sec * 1'000'000'000LL + ts.tv_nsec);
#endif
}

/// @brief Pushes to a queue once each period until destroyed, as a streamer does.
class Traffic {
    std::atomic<bool> running{true};
    std::thread thread;

public:
    Traffic(x::queue::SPSC<int> &queue, const x::telem::TimeSpan period):
        thread([this, &queue, period] {
            const auto sw = x::telem::Stopwatch();
            auto due = period;
            while (this->running.load(std::memory_order_relaxed)) {
                if (sw.elapsed() < due) {
                    std::this_thread::yield();
                    continue;
                }
                queue.push(0);
                due = due + period;
            }
        }) {}

    ~Traffic() {
        this->running.store(false);
        this->thread.join();
    }
};

/// @brief Returns how many waits of duration fit in the budget, inside the bounds.
int wait_count(const x::telem::TimeSpan duration) {
    const auto fit = DURATION_BUDGET.nanoseconds() / duration.nanoseconds();
    return static_cast<int>(std::clamp(fit, MIN_COUNT, MAX_COUNT));
}

/// @brief Holds waits of duration back to back and prints how late they fired. It
/// drains inputs after each wake, as the runtime does. The clock skew is the largest
/// gap between the stamp clock and the steady clock. The CPU share is the CPU time of
/// the loop thread over the wall time of the waits.
/// @returns the median fire error in nanoseconds.
std::int64_t measure_waits(
    Loop &loop,
    x::breaker::Breaker &breaker,
    x::queue::SPSC<int> &inputs,
    const x::telem::TimeSpan duration
) {
    const auto count = wait_count(duration);
    std::vector<std::int64_t> errors_ns;
    errors_ns.reserve(count);
    std::int64_t skew_ns = 0;
    int wakes = 0;
    int input = 0;
    const auto cpu_start = thread_cpu_time();
    const auto wall = x::telem::Stopwatch();
    for (int i = 0; i < count; i++) {
        const auto stamp = x::telem::TimeStamp::now();
        const auto sw = x::telem::Stopwatch();
        auto elapsed = x::telem::TimeSpan(0);
        while (elapsed < duration) {
            loop.wait(breaker, duration - elapsed, duration);
            while (inputs.try_pop(input))
                continue;
            elapsed = sw.elapsed();
            wakes++;
        }
        const auto stamped_ns = x::telem::TimeStamp::now().nanoseconds() -
                                stamp.nanoseconds();
        skew_ns = std::max(skew_ns, std::abs(stamped_ns - elapsed.nanoseconds()));
        errors_ns.push_back((elapsed - duration).nanoseconds());
    }
    const auto cpu_share = 100.0 * (thread_cpu_time() - cpu_start).nanoseconds() /
                           wall.elapsed().nanoseconds();
    const testutil::Spread spread(std::move(errors_ns));
    std::cout << "  wait " << duration << " x" << count << ", late: " << spread << ", "
              << wakes << " wakes, clock skew " << skew_ns / 1e3 << " us, cpu "
              << cpu_share << "%\n";
    return spread.at(50);
}

/// @brief Measures each duration on the loop that AUTO selects for shortest_span.
/// The loop runs on its own thread, as it does in the runtime.
/// @param shortest_span the shortest timer span, or the maximum span if there is none.
/// @param input_period the time between two inputs to the loop, or zero for no input.
/// @param durations the wait durations to measure.
/// @returns the median fire error of each duration in nanoseconds, or nothing if the
/// loop did not start.
std::vector<std::int64_t> sweep(
    const x::telem::TimeSpan shortest_span,
    const x::telem::TimeSpan input_period = x::telem::TimeSpan(0),
    const std::vector<x::telem::TimeSpan> &durations = DURATIONS
) {
    std::vector<std::int64_t> medians_ns;
    std::thread thread([&medians_ns, shortest_span, input_period, &durations] {
        Config config;
        // The Driver does not pin the loop thread on Windows.
        config.cpu_affinity = CPU_AFFINITY_NONE;
        config = config.apply_defaults(shortest_span);
        const auto loop = create(config);
        if (const auto err = loop->start()) {
            std::cout << "  start failed: " << err.message() << "\n";
            return;
        }
        x::queue::SPSC<int> inputs(INPUT_CAPACITY);
        if (!loop->watch(inputs.notifier())) {
            std::cout << "  watch failed\n";
            return;
        }
        x::breaker::Breaker breaker;
        breaker.start();
        std::cout << config.mode << ", interval " << config.interval;
        if (input_period.nanoseconds() > 0)
            std::cout << ", input every " << input_period;
        std::cout << "\n";
        log_timers();
        std::cout << std::fixed << std::setprecision(1) << "  longest stall in a "
                  << STALL_SPAN << " spin: " << measure_stall(STALL_SPAN).microseconds()
                  << " us\n";
        std::optional<Traffic> traffic;
        if (input_period.nanoseconds() > 0) traffic.emplace(inputs, input_period);
        for (const auto &duration: durations)
            medians_ns.push_back(measure_waits(*loop, breaker, inputs, duration));
        breaker.stop();
    });
    thread.join();
    return medians_ns;
}
}

/// @brief The clock the runtime stamps its writes from should step finer than 1 ms.
TEST(StampClockTest, StepsFinerThanOneMillisecond) {
    const auto span = 20 * x::telem::MILLISECOND;
    const auto sw = x::telem::Stopwatch();
    auto last_ns = x::telem::TimeStamp::now().nanoseconds();
    std::int64_t longest_ns = 0;
    int steps = 0;
    while (sw.elapsed() < span) {
        const auto now_ns = x::telem::TimeStamp::now().nanoseconds();
        if (now_ns == last_ns) continue;
        longest_ns = std::max(longest_ns, now_ns - last_ns);
        last_ns = now_ns;
        steps++;
    }
    std::cout << std::fixed << std::setprecision(1) << "stamp clock: " << steps
              << " steps in " << span << ", longest step " << longest_ns / 1e3
              << " us\n";
    EXPECT_GT(steps, span.nanoseconds() / x::telem::MILLISECOND.nanoseconds());
}

/// @brief A plain sleep should not return more than 1 ms early. Its spread is the
/// reference for what this machine gives a thread with no timer of its own.
TEST(SleepTest, HoldsItsDuration) {
    log_timers();
    const auto duration = 10 * x::telem::MILLISECOND;
    std::vector<std::int64_t> errors_ns;
    errors_ns.reserve(SLEEP_COUNT);
    for (int i = 0; i < SLEEP_COUNT; i++) {
        const auto sw = x::telem::Stopwatch();
        std::this_thread::sleep_for(duration.chrono());
        errors_ns.push_back((sw.elapsed() - duration).nanoseconds());
    }
    const testutil::Spread spread(std::move(errors_ns));
    std::cout << "sleep " << duration << ", late: " << spread << "\n";
    EXPECT_GE(spread.at(0), -MEDIAN_BOUND.nanoseconds());
}

/// @brief Wait timing of the loop that AUTO selects for one shortest timer span.
class WaitTimingTest : public testing::TestWithParam<x::telem::TimeSpan> {};

/// @brief On an idle machine, each wait duration should fire on its deadline.
TEST_P(WaitTimingTest, FiresOnDeadlineWhenIdle) {
    const auto medians_ns = sweep(this->GetParam());
    ASSERT_EQ(medians_ns.size(), DURATIONS.size());
    for (const auto median_ns: medians_ns)
        EXPECT_LE(median_ns, MEDIAN_BOUND.nanoseconds());
}

/// @brief On an idle machine, each short wait duration should be measured. The printed
/// spread shows the shortest wait the loop holds.
TEST_P(WaitTimingTest, MeasuresShortWaitsWhenIdle) {
    const auto medians_ns = sweep(
        this->GetParam(),
        x::telem::TimeSpan(0),
        SHORT_DURATIONS
    );
    ASSERT_EQ(medians_ns.size(), SHORT_DURATIONS.size());
}

/// @brief With every core busy, each wait duration should still be measured. The
/// printed spread shows what the scheduler adds.
TEST_P(WaitTimingTest, MeasuresUnderLoad) {
    const testutil::Load load;
    const auto medians_ns = sweep(this->GetParam());
    ASSERT_EQ(medians_ns.size(), DURATIONS.size());
}

/// @brief With input at 100 Hz, each wait duration should still be measured. The
/// printed spread shows what the input wakes add.
TEST_P(WaitTimingTest, MeasuresWithInputAt100Hz) {
    const auto medians_ns = sweep(this->GetParam(), 10 * x::telem::MILLISECOND);
    ASSERT_EQ(medians_ns.size(), DURATIONS.size());
}

/// @brief With input at 1 kHz, each wait duration should still be measured. The
/// printed spread shows what the input wakes add.
TEST_P(WaitTimingTest, MeasuresWithInputAt1kHz) {
    const auto medians_ns = sweep(this->GetParam(), x::telem::MILLISECOND);
    ASSERT_EQ(medians_ns.size(), DURATIONS.size());
}

INSTANTIATE_TEST_SUITE_P(
    ShortestSpans,
    WaitTimingTest,
    testing::Values(
        x::telem::TimeSpan::max(),
        10 * x::telem::MILLISECOND,
        4 * x::telem::MILLISECOND,
        x::telem::MILLISECOND,
        100 * x::telem::MICROSECOND
    )
);
}
