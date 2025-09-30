// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/loop/loop.h"
#include "gtest/gtest.h"
#include "x/cpp/telem/telem.h"

/// @brief it should correctly wait for an expended number of requests.
TEST(LoopTest, testWaitPrecise) {
    const auto rate = telem::HERTZ * 5000;
    const auto TARGET_AVG_THRESHOLD = telem::MICROSECOND * 500;
    loop::Timer timer{rate};
    std::vector<telem::TimeSpan> elapsed;
    const int count = 5e3;
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

TEST(LoopTest, testWaitLowRate) {
    const auto rate = telem::HERTZ * 10;
    const auto AVG_THRESHOLD = telem::MILLISECOND * 10;
    loop::Timer timer{rate};
    std::vector<telem::TimeSpan> elapsed;
    const int count = 10;
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

void runBreaker(breaker::Breaker &brker) {
    const auto rate = telem::HERTZ * 1;
    loop::Timer timer{rate};
    timer.wait(brker);
}

TEST(LoopTest, testWaitBreaker) {
    const auto b = breaker::Config{
        .name = "test",
        .base_interval = telem::MILLISECOND * 10,
        .max_retries = 10,
        .scale = 1.1
    };
    auto brker = breaker::Breaker(b);
    brker.start();
    const auto start = std::chrono::high_resolution_clock::now();
    std::thread t(runBreaker, std::ref(brker));
    std::this_thread::sleep_for((telem::MILLISECOND * 10).chrono());
    brker.stop();
    const auto end = std::chrono::high_resolution_clock::now();
    const auto elapsed = telem::TimeSpan(end - start);
    EXPECT_NEAR(
        elapsed.nanoseconds(),
        (telem::MILLISECOND * 10).nanoseconds(),
        (telem::MILLISECOND * 10).nanoseconds()
    );
    t.join();
}
