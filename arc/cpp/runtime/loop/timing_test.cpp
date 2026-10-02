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
#include <thread>
#include <vector>

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <windows.h>
#ifndef CREATE_WAITABLE_TIMER_HIGH_RESOLUTION
#define CREATE_WAITABLE_TIMER_HIGH_RESOLUTION 0x00000002
#endif
#endif

#include "gtest/gtest.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {
namespace {
/// @brief Earliest a wait may fire ahead of its deadline, as the time module allows.
const auto TOLERANCE = 100 * x::telem::MICROSECOND;
/// @brief Maximum median distance between a fire and its deadline on an idle machine.
const auto MEDIAN_BOUND = x::telem::MILLISECOND;
/// @brief Time a thread spins to find the longest stall the scheduler gives it.
const auto STALL_SPAN = 100 * x::telem::MILLISECOND;
/// @brief Waits measured for each duration.
constexpr int COUNT = 40;
/// @brief Wait durations measured on each loop.
const std::vector<x::telem::TimeSpan> DURATIONS = {
    x::telem::MILLISECOND,
    5 * x::telem::MILLISECOND,
    10 * x::telem::MILLISECOND,
    20 * x::telem::MILLISECOND,
    30 * x::telem::MILLISECOND,
    100 * x::telem::MILLISECOND,
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
    const HANDLE timer = CreateWaitableTimerExW(
        NULL,
        NULL,
        CREATE_WAITABLE_TIMER_HIGH_RESOLUTION,
        TIMER_ALL_ACCESS
    );
    if (timer == NULL) {
        std::cout << "  high-resolution timer: unavailable, error " << GetLastError()
                  << "\n";
        return;
    }
    std::cout << "  high-resolution timer: available\n";
    CloseHandle(timer);
#endif
}

/// @brief Prints the minimum, median, 90th percentile, and maximum of a set of spans.
/// @param spans_ns the spans in nanoseconds.
/// @returns the median in nanoseconds.
std::int64_t print_spread(std::vector<std::int64_t> spans_ns) {
    std::sort(spans_ns.begin(), spans_ns.end());
    const auto at = [&spans_ns](const size_t percent) {
        return spans_ns[(spans_ns.size() - 1) * percent / 100];
    };
    std::cout << std::fixed << std::setprecision(1) << "min " << at(0) / 1e3 << ", p50 "
              << at(50) / 1e3 << ", p90 " << at(90) / 1e3 << ", max " << at(100) / 1e3
              << " us";
    return at(50);
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

/// @brief Holds COUNT waits of duration back to back and prints how late they fired.
/// The clock skew is the largest gap between the stamp clock and the steady clock.
/// @returns the median fire error in nanoseconds.
std::int64_t measure_waits(
    Loop &loop,
    x::breaker::Breaker &breaker,
    const x::telem::TimeSpan duration
) {
    std::vector<std::int64_t> errors_ns;
    errors_ns.reserve(COUNT);
    std::int64_t skew_ns = 0;
    int wakes = 0;
    for (int i = 0; i < COUNT; i++) {
        const auto stamp = x::telem::TimeStamp::now();
        const auto sw = x::telem::Stopwatch();
        auto elapsed = x::telem::TimeSpan(0);
        while (elapsed < duration - TOLERANCE) {
            loop.wait(breaker, duration - elapsed);
            elapsed = sw.elapsed();
            wakes++;
        }
        const auto stamped_ns = x::telem::TimeStamp::now().nanoseconds() -
                                stamp.nanoseconds();
        skew_ns = std::max(skew_ns, std::abs(stamped_ns - elapsed.nanoseconds()));
        errors_ns.push_back((elapsed - duration).nanoseconds());
    }
    std::cout << "  wait " << duration << ": ";
    const auto median_ns = print_spread(errors_ns);
    std::cout << " late, " << wakes << " wakes, clock skew " << skew_ns / 1e3
              << " us\n";
    return median_ns;
}

/// @brief Measures every duration on the loop that AUTO selects for timing_interval.
/// The loop runs on its own thread, as it does in the runtime.
/// @param timing_interval the base interval, or the maximum span if there is none.
/// @returns the median fire error of each duration in nanoseconds, or nothing if the
/// loop did not start.
std::vector<std::int64_t> sweep(const x::telem::TimeSpan timing_interval) {
    std::vector<std::int64_t> medians_ns;
    std::thread thread([&medians_ns, timing_interval] {
        Config config;
        // The Driver does not pin the loop thread on Windows.
        config.cpu_affinity = CPU_AFFINITY_NONE;
        config = config.apply_defaults(timing_interval);
        const auto loop = create(config);
        if (const auto err = loop->start()) {
            std::cout << "  start failed: " << err.message() << "\n";
            return;
        }
        x::breaker::Breaker breaker;
        breaker.start();
        std::cout << config.mode << ", interval " << config.interval << "\n";
        log_timers();
        std::cout << std::fixed << std::setprecision(1) << "  longest stall in a "
                  << STALL_SPAN << " spin: " << measure_stall(STALL_SPAN).microseconds()
                  << " us\n";
        for (const auto &duration: DURATIONS)
            medians_ns.push_back(measure_waits(*loop, breaker, duration));
        breaker.stop();
    });
    thread.join();
    return medians_ns;
}

/// @brief Spins one thread on each core until destroyed.
class Load {
    std::atomic<bool> running{true};
    std::vector<std::thread> threads;

public:
    Load() {
        const auto count = std::max(1u, std::thread::hardware_concurrency());
        for (unsigned int i = 0; i < count; i++)
            this->threads.emplace_back([this] {
                while (this->running.load(std::memory_order_relaxed))
                    continue;
            });
    }

    ~Load() {
        this->running.store(false);
        for (auto &thread: this->threads)
            thread.join();
    }
};
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
    errors_ns.reserve(COUNT);
    for (int i = 0; i < COUNT; i++) {
        const auto sw = x::telem::Stopwatch();
        std::this_thread::sleep_for(duration.chrono());
        errors_ns.push_back((sw.elapsed() - duration).nanoseconds());
    }
    std::cout << "sleep " << duration << ": ";
    print_spread(errors_ns);
    std::cout << " late\n";
    const auto earliest_ns = *std::min_element(errors_ns.begin(), errors_ns.end());
    EXPECT_GE(earliest_ns, -MEDIAN_BOUND.nanoseconds());
}

/// @brief Wait timing of the loop that AUTO selects for one base interval.
class WaitTimingTest : public testing::TestWithParam<x::telem::TimeSpan> {};

/// @brief On an idle machine, each wait duration should fire on its deadline.
TEST_P(WaitTimingTest, FiresOnDeadlineWhenIdle) {
    const auto medians_ns = sweep(this->GetParam());
    ASSERT_EQ(medians_ns.size(), DURATIONS.size());
    for (const auto median_ns: medians_ns)
        EXPECT_LE(median_ns, MEDIAN_BOUND.nanoseconds());
}

/// @brief With every core busy, each wait duration should still be measured. The
/// printed spread shows what the scheduler adds.
TEST_P(WaitTimingTest, MeasuresUnderLoad) {
    const Load load;
    const auto medians_ns = sweep(this->GetParam());
    ASSERT_EQ(medians_ns.size(), DURATIONS.size());
}

INSTANTIATE_TEST_SUITE_P(
    BaseIntervals,
    WaitTimingTest,
    testing::Values(
        x::telem::TimeSpan::max(),
        10 * x::telem::MILLISECOND,
        4 * x::telem::MILLISECOND,
        x::telem::MILLISECOND
    )
);
}
