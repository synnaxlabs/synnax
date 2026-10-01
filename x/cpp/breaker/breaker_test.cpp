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

namespace x::breaker {
void helper(Breaker &b) {
    while (b.wait("testBreakRetries breaker"))
        ;
}

/// @brief it should correctly wait for an expended number of requests.
TEST(BreakerTests, testBreaker) {
    auto b = Breaker(Config{"my-breaker", 10 * telem::MILLISECOND, 1, 1});
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
    auto b = Breaker(Config{"my-breaker", 10 * telem::MILLISECOND, 10, 1.1});
    EXPECT_TRUE(b.start());
    EXPECT_TRUE(b.running());
    while (b.wait("testBreakRetries breaker")) {}
    EXPECT_TRUE(b.stop());
    EXPECT_FALSE(b.running());
}

/// @brief it should correctly shut down before expending the max number of requests
TEST(BreakerTests, testBreakerPrematureShutdown) {
    auto b = Breaker(Config{"my-breaker", 10 * telem::MILLISECOND, 10, 1});
    EXPECT_TRUE(b.start());
    std::thread t(&helper, std::ref(b));
    std::this_thread::sleep_for(std::chrono::milliseconds(40));
    EXPECT_TRUE(b.stop());
    t.join();
}

/// @brief it should correctly shut down before expending the max number of requests
TEST(BreakerTests, testDestructorShuttingDown) {
    const auto b = std::make_unique<Breaker>(
        Config{"my-breaker", 10 * telem::MILLISECOND, 10, 1}
    );
    EXPECT_TRUE(b->start());
    EXPECT_TRUE(b->running());
    std::thread t(&helper, std::ref(*b));
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    EXPECT_TRUE(b->stop());
    EXPECT_FALSE(b->running());
    t.join();
}

/// @brief it should retry past the default maximum retry count when the breaker is
/// configured to retry infinitely.
TEST(BreakerTests, testInfiniteRetries) {
    auto b = Breaker(
        Config{"my-breaker", telem::TimeSpan::ZERO(), RETRY_INFINITELY, 1}
    );
    const auto past_default_max = static_cast<size_t>(Config{}.max_retries) + 10;
    EXPECT_TRUE(b.start());
    EXPECT_TRUE(b.running());
    size_t retries = 0;
    while (retries < past_default_max && b.wait("testInfiniteRetries breaker"))
        retries++;
    EXPECT_EQ(retries, past_default_max);
    EXPECT_TRUE(b.stop());
    EXPECT_FALSE(b.running());
}

/// @brief it should return false when attempting to start a breaker that was
/// already running.
TEST(BreakerTests, testStartAlreadyRunning) {
    Breaker b;
    EXPECT_TRUE(b.start());
    EXPECT_FALSE(b.start());
    EXPECT_TRUE(b.stop());
}

/// @brief it should return false when attempting to stop a breaker that was
/// already stopped.
TEST(BreakerTests, testStopAlreadyStopped) {
    Breaker b;
    EXPECT_TRUE(b.start());
    EXPECT_TRUE(b.stop());
    EXPECT_FALSE(b.stop());
}

/// @brief it should increment the retry count when the breaker is triggered, starting
/// at 0.
TEST(BreakerTest, testRetryCount) {
    auto b = Breaker(Config{"my-breaker", 10 * telem::MILLISECOND, 5, 1});
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

/// @brief it should describe the retry number and interval of the next call to wait().
TEST(BreakerTest, testNextRetry) {
    auto b = Breaker(Config{"my-breaker", 100 * telem::MILLISECOND, 2, 2});
    EXPECT_TRUE(b.start());
    EXPECT_EQ(b.next_retry(), "retry 1/2 in 0.1 s");
    EXPECT_TRUE(b.wait("first retry"));
    EXPECT_EQ(b.next_retry(), "retry 2/2 in 0.2 s");
    EXPECT_TRUE(b.wait("second retry"));
    EXPECT_EQ(b.next_retry(), "");
    b.reset();
    EXPECT_EQ(b.next_retry(), "retry 1/2 in 0.1 s");
    EXPECT_TRUE(b.stop());
}

/// @brief it should show an unbounded retry count when the breaker retries infinitely.
TEST(BreakerTest, testNextRetryInfinite) {
    auto b = Breaker(Config{"my-breaker", telem::SECOND, RETRY_INFINITELY, 1});
    EXPECT_EQ(b.next_retry(), "retry 1/∞ in 1.0 s");
}

/// @brief it should allow reads and resets of the retry state while another thread
/// waits on the breaker.
TEST(BreakerTest, testConcurrentRetryState) {
    auto b = Breaker(
        Config{"my-breaker", telem::TimeSpan::ZERO(), RETRY_INFINITELY, 1}
    );
    EXPECT_TRUE(b.start());
    std::thread t(&helper, std::ref(b));
    for (int i = 0; i < 1000; i++) {
        EXPECT_TRUE(b.next_retry().starts_with("retry "));
        static_cast<void>(b.retry_count());
        b.reset();
    }
    EXPECT_TRUE(b.stop());
    t.join();
}
}
