// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "x/cpp/breaker/breaker.h"

void helper(breaker::Breaker &b) {
    while (b.wait("testBreakRetries breaker"));
}

/// @brief it should correctly wait for an expended number of requests.
TEST(BreakerTests, testBreaker) {
    auto b = breaker::Breaker(breaker::Config{"my-breaker", 10 * telem::MILLISECOND, 1, 1});
    b.start();
    ASSERT_TRUE(b.wait("testBreaker breaker"));
    ASSERT_FALSE(b.wait("testBreaker breaker"));
    b.stop();
}

/// @brief it should correctly expend max number of requests
TEST(BreakerTests, testBreakRetries) {
    auto b = breaker::Breaker(breaker::Config{"my-breaker", 10 * telem::MILLISECOND, 10, 1.1});
    b.start();
    while (b.wait("testBreakRetries breaker")) {}
    b.stop();
}

/// @brief it should correctly shut down before expending the max number of requests
TEST(BreakerTests, testBreakerPrematureShutdown) {
    auto b = breaker::Breaker(breaker::Config{"my-breaker", 10 * telem::MILLISECOND, 10, 1});
    b.start();
    // Create a new thread
    std::thread t(&helper, std::ref(b));
    // Sleep a couple seconds
    std::this_thread::sleep_for(std::chrono::milliseconds(40));
    b.stop();
    t.join();
}

/// @brief it should correctly shut down before expending the max number of requests
TEST(BreakerTests, testDestructorShuttingDown) {
    // create a unique pointer to a breaker
    auto b = std::make_unique<breaker::Breaker>(breaker::Config{
        "my-breaker", 10 * telem::MILLISECOND, 10, 1
    });
    b->start();
    //create a new thread
    std::thread t(&helper, std::ref(*b));
    //sleep briefly
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    // stop the breaker before destroying it
    b->stop();
    // join the thread before destroying the breaker
    t.join();
    // now safe to destroy the object
    b.reset();
}
