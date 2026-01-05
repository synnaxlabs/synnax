// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/breaker/breaker.h"

void helper(breaker::Breaker &b) {
    while (b.wait("testBreakRetries breaker"))
        ;
}

/// @brief it should correctly wait for an expended number of requests.
TEST(BreakerTests, testBreaker) {
    auto b = breaker::Breaker(
        breaker::Config{"my-breaker", 10 * telem::MILLISECOND, 1, 1}
    );
    EXPECT_TRUE(b.start());
    EXPECT_TRUE(b.running());
    EXPECT_TRUE(b.wait("testBreaker breaker"));
    EXPECT_FALSE(b.wait("testBreaker breaker"));
    EXPECT_TRUE(b.running());
    EXPECT_TRUE(b.stop());
    EXPECT_FALSE(b.running());
}

/// @brief it should correctly expend max number of requests
TEST(BreakerTests, testBreakRetries) {
    auto b = breaker::Breaker(
        breaker::Config{"my-breaker", 10 * telem::MILLISECOND, 10, 1.1}
    );
    EXPECT_TRUE(b.start());
    EXPECT_TRUE(b.running());
    while (b.wait("testBreakRetries breaker")) {}
    EXPECT_TRUE(b.stop());
    EXPECT_FALSE(b.running());
}

/// @brief it should correctly shut down before expending the max number of requests
TEST(BreakerTests, testBreakerPrematureShutdown) {
    auto b = breaker::Breaker(
        breaker::Config{"my-breaker", 10 * telem::MILLISECOND, 10, 1}
    );
    EXPECT_TRUE(b.start());
    std::thread t(&helper, std::ref(b));
    std::this_thread::sleep_for(std::chrono::milliseconds(40));
    EXPECT_TRUE(b.stop());
    t.join();
}

/// @brief it should correctly shut down before expending the max number of requests
TEST(BreakerTests, testDestructorShuttingDown) {
    const auto b = std::make_unique<breaker::Breaker>(
        breaker::Config{"my-breaker", 10 * telem::MILLISECOND, 10, 1}
    );
    EXPECT_TRUE(b->start());
    EXPECT_TRUE(b->running());
    std::thread t(&helper, std::ref(*b));
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    EXPECT_TRUE(b->stop());
    EXPECT_FALSE(b->running());
    t.join();
}

/// @brief it should correctly handle infinite retries
TEST(BreakerTests, testInfiniteRetries) {
    auto b = breaker::Breaker(
        breaker::Config{
            "my-breaker",
            10 * telem::MICROSECOND,
            breaker::RETRY_INFINITELY, // Set to infinite retries
            1.1
        }
    );
    EXPECT_TRUE(b.start());
    EXPECT_TRUE(b.running());
    int retry_count = 0;
    std::thread t([&b, &retry_count]() {
        while (b.wait("testInfiniteRetries breaker")) {
            retry_count++;
            if (retry_count >= 100) break; // Safety break to prevent infinite test
        }
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    EXPECT_TRUE(b.stop());
    EXPECT_FALSE(b.running());
    t.join();

    // Verify that we got multiple retries and didn't stop at the default max (50)
    ASSERT_GT(retry_count, 50);
}

/// @brief it should return false when attempting to start a breaker that was
/// already running.
TEST(BreakerTests, testStartAlreadyRunning) {
    breaker::Breaker b;
    EXPECT_TRUE(b.start());
    EXPECT_FALSE(b.start());
    EXPECT_TRUE(b.stop());
}

/// @brief it should return false when attempting to stop a breaker that was
/// already stopped.
TEST(BreakerTests, testStopAlreadyStopped) {
    breaker::Breaker b;
    EXPECT_TRUE(b.start());
    EXPECT_TRUE(b.stop());
    EXPECT_FALSE(b.stop());
}

/// @brief it should increment the retry count when the breaker is triggered, starting
/// at 0.
TEST(BreakerTest, testRetryCount) {
    auto b = breaker::Breaker(
        breaker::Config{"my-breaker", 10 * telem::MILLISECOND, 5, 1}
    );
    EXPECT_TRUE(b.start());
    EXPECT_EQ(b.retry_count(), 0);
    EXPECT_TRUE(b.wait("first retry"));
    EXPECT_EQ(b.retry_count(), 1);
    EXPECT_TRUE(b.wait("second retry"));
    EXPECT_EQ(b.retry_count(), 2);
    b.reset();
    EXPECT_EQ(b.retry_count(), 0);
    EXPECT_TRUE(b.stop());
}
