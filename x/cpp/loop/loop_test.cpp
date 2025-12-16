// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

/// @brief it should correctly wait for an expended number of requests.
TEST(LoopTest, testWaitPrecise) {
    const auto rate = telem::HERTZ * 5000;
    const auto TARGET_AVG_THRESHOLD = telem::MICROSECOND * 500;
    loop::Timer timer{rate};
    std::vector<telem::TimeSpan> elapsed;
    constexpr int COUNT = 5e3;
    elapsed.reserve(COUNT);
    for (int i = 0; i < COUNT; i++) {
        auto start = std::chrono::high_resolution_clock::now();
        timer.wait();
        auto end = std::chrono::high_resolution_clock::now();
        elapsed.emplace_back(end - start);
    }
    auto total_delta = telem::TimeSpan(0);
    for (const auto &e: elapsed) {
        const auto delta = e.delta(rate.period());
        total_delta += delta;
    }
    auto avg_delta = total_delta / COUNT;
    EXPECT_LT(avg_delta, TARGET_AVG_THRESHOLD);
}

/// @brief it should correctly wait for low rate requests.
TEST(LoopTest, testWaitLowRate) {
    const auto rate = telem::HERTZ * 10;
    const auto AVG_THRESHOLD = telem::MILLISECOND * 10;
    loop::Timer timer{rate};
    std::vector<telem::TimeSpan> elapsed;
    constexpr int COUNT = 10;
    elapsed.reserve(COUNT);
    for (int i = 0; i < COUNT; i++) {
        auto start = std::chrono::high_resolution_clock::now();
        timer.wait();
        auto end = std::chrono::high_resolution_clock::now();
        elapsed.emplace_back(end - start);
    }
    auto total_delta = telem::TimeSpan(0);
    for (const auto &e: elapsed) {
        const auto delta = e.delta(rate.period());
        total_delta += delta;
    }
    auto avg_delta = total_delta / COUNT;
    EXPECT_LT(avg_delta, AVG_THRESHOLD);
}

void run_breaker(breaker::Breaker &brk) {
    const auto rate = telem::HERTZ * 1;
    loop::Timer timer{rate};
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
    std::thread t(run_breaker, std::ref(brk));
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
