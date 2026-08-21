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
#include <thread>
#include <vector>

#ifdef _WIN32
#include <windows.h>
#endif

#include "gtest/gtest.h"

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

namespace x::loop {
/// @brief Allowed relative error of a measured rate.
constexpr double RATE_TOLERANCE = 0.01;
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

/// @brief Stats from one rate sweep.
struct RateStats {
    /// @brief Average rate from the first return to the last.
    double hz;
    /// @brief Waits that returned late.
    int overruns;
    /// @brief 99th percentile deviation of a period from the ideal.
    telem::TimeSpan p99_dev;
    /// @brief Worst deviation of a period from the ideal.
    telem::TimeSpan max_dev;
};

/// @brief Calls wait count times and measures the rate the same way the integration
/// tests measure a task's sample rate, plus per-period jitter and overruns.
template<typename Wait>
RateStats measure_rate(Wait &wait, const int count, const telem::TimeSpan period) {
    std::vector<hs_clock::time_point> wakes;
    wakes.reserve(count);
    int overruns = 0;
    wait();
    wakes.push_back(hs_clock::now());
    for (int i = 1; i < count; i++) {
        if (!wait().second) overruns++;
        wakes.push_back(hs_clock::now());
    }
    std::vector<telem::TimeSpan> devs;
    devs.reserve(count - 1);
    for (size_t i = 1; i < wakes.size(); i++)
        devs.push_back(telem::TimeSpan(wakes[i] - wakes[i - 1]).delta(period));
    std::sort(devs.begin(), devs.end());
    const auto span = telem::TimeSpan(wakes.back() - wakes.front());
    return {
        .hz = (count - 1) / span.seconds(),
        .overruns = overruns,
        .p99_dev = devs[std::min(devs.size() - 1, devs.size() * 99 / 100)],
        .max_dev = devs.back(),
    };
}

/// @brief Logs the rate, jitter, and overruns wait holds at rate_hz and checks the
/// rate against the tolerance.
template<typename Wait>
void expect_rate(Wait wait, const int rate_hz) {
    const auto period = telem::Rate(rate_hz).period();
    const auto stats = measure_rate(wait, rate_hz * SPAN_SECONDS, period);
    const double error = (stats.hz - rate_hz) / rate_hz * 100;
    std::cout << std::fixed << std::setprecision(1);
    std::cout << rate_hz << " Hz: " << stats.hz << " Hz measured (" << error << "%), ";
    std::cout << std::setprecision(3);
    std::cout << "jitter p99 " << stats.p99_dev.milliseconds() << " ms max "
              << stats.max_dev.milliseconds() << " ms, overruns " << stats.overruns
              << "\n";
    EXPECT_NEAR(stats.hz, rate_hz, rate_hz * RATE_TOLERANCE)
        << "at " << rate_hz << " Hz";
}

/// @brief it should correctly wait for an expended number of requests.
TEST(LoopTest, testWaitPrecise) {
    const auto rate = telem::HERTZ * 5000;
    const auto TARGET_AVG_THRESHOLD = telem::MICROSECOND * 500;
    Timer timer{rate};
    std::vector<telem::TimeSpan> elapsed;
    constexpr int count = 5e3;
    elapsed.reserve(count);
    for (int i = 0; i < count; i++) {
        auto start = std::chrono::high_resolution_clock::now();
        timer.wait();
        auto end = std::chrono::high_resolution_clock::now();
        elapsed.emplace_back(end - start);
    }
    auto total_delta = telem::TimeSpan::ZERO();
    for (const auto &e: elapsed) {
        const auto delta = e.delta(rate.period());
        total_delta += delta;
    }
    auto avg_delta = total_delta / count;
    EXPECT_LT(avg_delta, TARGET_AVG_THRESHOLD);
}

/// @brief it should correctly wait for low rate requests.
TEST(LoopTest, testWaitLowRate) {
    const auto rate = telem::HERTZ * 10;
    const auto AVG_THRESHOLD = telem::MILLISECOND * 10;
    Timer timer{rate};
    std::vector<telem::TimeSpan> elapsed;
    constexpr int count = 10;
    elapsed.reserve(count);
    for (int i = 0; i < count; i++) {
        auto start = std::chrono::high_resolution_clock::now();
        timer.wait();
        auto end = std::chrono::high_resolution_clock::now();
        elapsed.emplace_back(end - start);
    }
    auto total_delta = telem::TimeSpan::ZERO();
    for (const auto &e: elapsed) {
        const auto delta = e.delta(rate.period());
        total_delta += delta;
    }
    auto avg_delta = total_delta / count;
    EXPECT_LT(avg_delta, AVG_THRESHOLD);
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
    for (const int rate_hz: {25, 50, 100, 150, 199, 200, 201, 250, 500, 1000}) {
        Timer timer{telem::Rate(rate_hz)};
        expect_rate([&] { return timer.wait(); }, rate_hz);
    }
}

/// @brief it should hold each configured rate over time on the breaker path.
TEST(LoopTest, testWaitBreakerHoldsRate) {
    log_timer_resolution();
    auto brk = breaker::Breaker(breaker::default_config("test"));
    brk.start();
    for (const int rate_hz: {5, 10, 20, 50}) {
        Timer timer{telem::Rate(rate_hz)};
        expect_rate([&] { return timer.wait(brk); }, rate_hz);
    }
    brk.stop();
}

/// @brief it should land a single wait closer to its deadline than the threshold
/// that selects the sleep path. The anchor corrects the average, not one wake, so
/// each wait gets a fresh timer.
TEST(LoopTest, testWaitOvershoot) {
    log_timer_resolution();
    std::cout << std::fixed << std::setprecision(3);
    for (const int ms: {5, 10, 15}) {
        const auto period = telem::MILLISECOND * ms;
        auto min_dev = telem::TimeSpan::max();
        auto max_dev = telem::TimeSpan::ZERO();
        auto total = telem::TimeSpan::ZERO();
        constexpr int reps = 20;
        for (int i = 0; i < reps; i++) {
            Timer timer{period};
            const auto start = hs_clock::now();
            timer.wait();
            const auto dev = telem::TimeSpan(hs_clock::now() - start).delta(period);
            min_dev = std::min(min_dev, dev);
            max_dev = std::max(max_dev, dev);
            total += dev;
        }
        std::cout << ms << " ms period: min " << min_dev.milliseconds()
                  << " ms off, mean " << (total / reps).milliseconds()
                  << " ms off, max " << max_dev.milliseconds() << " ms off\n";
        EXPECT_LT(max_dev, HIGH_RES_THRESHOLD) << "at a " << ms << " ms period";
    }
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
