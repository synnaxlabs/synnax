// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <iomanip>
#include <iostream>

#include "gtest/gtest.h"

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

namespace x::loop {
/// @brief Allowed relative error of a measured rate.
constexpr double RATE_TOLERANCE = 0;
/// @brief Wall time each rate is measured over.
constexpr int SPAN_SECONDS = 1;

/// @brief Calls wait count times and returns the rate measured from the first return
/// to the last, the same way the integration tests measure a task's sample rate.
template<typename Wait>
double measure_rate(Wait &wait, const int count) {
    wait();
    const auto first = hs_clock::now();
    for (int i = 1; i < count; i++)
        wait();
    return (count - 1) / telem::TimeSpan(hs_clock::now() - first).seconds();
}

/// @brief Logs the rate wait holds at rate_hz and checks it against the tolerance.
template<typename Wait>
void expect_rate(Wait wait, const int rate_hz) {
    const double measured = measure_rate(wait, rate_hz * SPAN_SECONDS);
    const double error = (measured - rate_hz) / rate_hz * 100;
    std::cout << std::fixed << std::setprecision(1);
    std::cout << rate_hz << " Hz: " << measured << " Hz measured (" << error << "%)\n";
    EXPECT_NEAR(measured, rate_hz, rate_hz * RATE_TOLERANCE)
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
    for (const int rate_hz: {50, 100, 150, 200, 250, 400, 450, 500, 550, 1000, 2000}) {
        Timer timer{telem::Rate(rate_hz)};
        expect_rate([&] { timer.wait(); }, rate_hz);
    }
}

/// @brief it should hold each configured rate over time on the breaker path.
TEST(LoopTest, testWaitBreakerHoldsRate) {
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
