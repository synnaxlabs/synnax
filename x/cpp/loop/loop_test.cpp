// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <iomanip>
#include <iostream>
#include <vector>

#ifdef _WIN32
#include <windows.h>
#endif

#include "gtest/gtest.h"

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

namespace x::loop {
/// @brief Allowed relative error of a measured rate.
constexpr double RATE_TOLERANCE = 0.05;
/// @brief Wall time each rate is measured over.
constexpr int SPAN_SECONDS = 1;

/// @brief Logs the timer resolution a sleep on this machine rounds up to. Windows
/// defaults to 15.625 ms, but any process raising it changes what every other process
/// measures, so the rates below only mean something next to this number.
void log_timer_resolution() {
#ifdef _WIN32
    using Query = LONG(WINAPI *)(PULONG, PULONG, PULONG);
    const auto ntdll = GetModuleHandleW(L"ntdll.dll");
// C4191 fires on every GetProcAddress cast; the signature is fixed by the ntdll ABI.
#pragma warning(push)
#pragma warning(disable : 4191)
    const auto query = reinterpret_cast<Query>(
        GetProcAddress(ntdll, "NtQueryTimerResolution")
    );
#pragma warning(pop)
    if (query == nullptr) {
        std::cout << "timer resolution: NtQueryTimerResolution unavailable\n";
        return;
    }
    ULONG min_100ns = 0, max_100ns = 0, current_100ns = 0;
    query(&min_100ns, &max_100ns, &current_100ns);
    std::cout << std::fixed << std::setprecision(4);
    std::cout << "timer resolution: " << current_100ns / 10000.0 << " ms current, "
              << max_100ns / 10000.0 << " ms best, " << min_100ns / 10000.0
              << " ms default\n";
#endif
}

/// @brief Calls wait count times and returns the period it held on each call.
/// @param wait the wait to measure.
/// @param count how many times to call wait.
/// @returns one period per call after the first, which primes the timer.
template<typename Wait>
std::vector<telem::TimeSpan> measure_periods(Wait wait, const int count) {
    std::vector<telem::TimeSpan> periods;
    periods.reserve(count - 1);
    wait();
    for (int i = 1; i < count; i++) {
        const auto start = hs_clock::now();
        wait();
        periods.emplace_back(hs_clock::now() - start);
    }
    return periods;
}

/// @brief Reduces spans to their median. A stall on a loaded runner adds one long span
/// and one near-zero one, which the median rejects. A mean cannot reject either: the
/// timer re-anchors after an overrun instead of catching up on the time the stall took.
/// @param spans the spans to reduce. Must hold at least one span.
/// @returns the median span.
telem::TimeSpan median(std::vector<telem::TimeSpan> spans) {
    const auto mid = spans.begin() + spans.size() / 2;
    std::nth_element(spans.begin(), mid, spans.end());
    return *mid;
}

/// @brief Calls wait count times and returns how far a typical period fell from target.
/// @param wait the wait to measure.
/// @param target the period the timer holds.
/// @param count how many times to call wait.
/// @returns the median distance between a held period and target.
template<typename Wait>
telem::TimeSpan
measure_deviation(Wait wait, const telem::TimeSpan &target, const int count) {
    auto periods = measure_periods(wait, count);
    for (auto &period: periods)
        period = period.delta(target);
    return median(periods);
}

/// @brief Logs the rate wait holds at rate_hz and checks it against the tolerance.
/// @param wait the wait to measure.
/// @param rate_hz the rate the timer holds.
template<typename Wait>
void expect_rate(Wait wait, const int rate_hz) {
    const auto period = median(measure_periods(wait, rate_hz * SPAN_SECONDS));
    const double measured = 1 / period.seconds();
    const double error = (measured - rate_hz) / rate_hz * 100;
    std::cout << std::fixed << std::setprecision(1);
    std::cout << rate_hz << " Hz: " << measured << " Hz measured (" << error << "%)\n";
    EXPECT_NEAR(measured, rate_hz, rate_hz * RATE_TOLERANCE)
        << "at " << rate_hz << " Hz";
}

/// @brief it should hold its period on each wait at a high rate.
TEST(LoopTest, testWaitPrecise) {
    const auto rate = telem::HERTZ * 5000;
    Timer timer{rate};
    const auto deviation = measure_deviation(
        [&] { timer.wait(); },
        rate.period(),
        5000
    );
    EXPECT_LT(deviation, telem::MICROSECOND * 500);
}

/// @brief it should hold its period on each wait at a low rate.
TEST(LoopTest, testWaitLowRate) {
    const auto rate = telem::HERTZ * 10;
    Timer timer{rate};
    const auto deviation = measure_deviation([&] { timer.wait(); }, rate.period(), 10);
    EXPECT_LT(deviation, telem::MILLISECOND * 10);
}

void runBreaker(breaker::Breaker &brk) {
    const auto rate = telem::HERTZ * 1;
    Timer timer{rate};
    timer.wait(brk);
}

/// @brief it should correctly interrupt wait when breaker is stopped.
TEST(LoopTest, testWaitBreaker) {
    const auto b = breaker::Config{
        .name = "test",
        .base_interval = telem::MILLISECOND * 10,
        .max_retries = 10,
        .scale = 1.1
    };
    auto brk = breaker::Breaker(b);
    brk.start();
    const auto start = std::chrono::high_resolution_clock::now();
    std::thread t(runBreaker, std::ref(brk));
    std::this_thread::sleep_for((telem::MILLISECOND * 10).chrono());
    brk.stop();
    const auto end = std::chrono::high_resolution_clock::now();
    const auto elapsed = telem::TimeSpan(end - start);
    EXPECT_NEAR(
        elapsed.nanoseconds(),
        (telem::MILLISECOND * 10).nanoseconds(),
        (telem::MILLISECOND * 10).nanoseconds()
    );
    t.join();
}

/// @brief it should hold each configured rate over time on the sleep path.
TEST(LoopTest, testWaitHoldsRate) {
    log_timer_resolution();
    // 199 through 201 bracket the 5 ms high_rate() threshold, where a late wake used
    // to cost 20% of the rate on one side and nothing on the other.
    for (const int rate_hz: {50, 100, 199, 200, 201, 500, 1000}) {
        Timer timer{telem::Rate(rate_hz)};
        expect_rate([&] { timer.wait(); }, rate_hz);
    }
}

/// @brief it should hold each configured rate over time on the breaker path.
TEST(LoopTest, testWaitBreakerHoldsRate) {
    log_timer_resolution();
    auto brk = breaker::Breaker(breaker::default_config("test"));
    brk.start();
    for (const int rate_hz: {5, 10, 20, 50}) {
        Timer timer{telem::Rate(rate_hz)};
        expect_rate([&] { timer.wait(brk); }, rate_hz);
    }
    brk.stop();
}

/// @brief it should not stretch the next wait after the breaker interrupts a sleep.
TEST(LoopTest, testWaitBreakerEarlyWake) {
    auto brk = breaker::Breaker(breaker::default_config("test"));
    brk.start();
    Timer timer{telem::Rate(2)};
    std::thread t([&] { timer.wait(brk); });
    std::this_thread::sleep_for((telem::MILLISECOND * 100).chrono());
    brk.stop();
    t.join();
    brk.start();
    const auto start = hs_clock::now();
    timer.wait(brk);
    const auto elapsed = telem::TimeSpan(hs_clock::now() - start);
    brk.stop();
    EXPECT_GT(elapsed, telem::MILLISECOND * 250);
    EXPECT_LT(elapsed, telem::MILLISECOND * 700);
}

/// @brief it should re-anchor after an overrun instead of catching up on the missed
/// periods.
TEST(LoopTest, testWaitOverrunNoCatchUp) {
    Timer timer{telem::Rate(50)};
    timer.wait();
    std::this_thread::sleep_for((telem::MILLISECOND * 50).chrono());
    const auto [elapsed, on_time] = timer.wait();
    EXPECT_FALSE(on_time);
    EXPECT_GE(elapsed, telem::MILLISECOND * 50);
    const auto start = hs_clock::now();
    timer.wait();
    const auto next = telem::TimeSpan(hs_clock::now() - start);
    EXPECT_GT(next, telem::MILLISECOND * 15);
    EXPECT_LT(next, telem::MILLISECOND * 60);
}
}
