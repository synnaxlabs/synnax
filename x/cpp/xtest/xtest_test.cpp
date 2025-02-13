// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <atomic>
/// external
#include "gtest/gtest.h"

/// internal
#include "x/cpp/xtest/xtest.h"

class XTestTest : public ::testing::Test {
protected:
    std::atomic<int> counter{0};
    
    void incrementCounter() {
        counter++;
    }

    void SetUp() override {
        counter = 0;
    }
};

TEST_F(XTestTest, TestEventuallyEQ) {
    // Start a thread that increments counter to 5
    std::thread t([this]() {
        for (int i = 0; i < 5; i++) {
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            incrementCounter();
        }
    });

    // Should eventually equal 5
    ASSERT_EVENTUALLY_EQ(counter.load(), 5);
    t.join();
}

TEST_F(XTestTest, TestEventuallyGE) {
    std::thread t([this]() {
        for (int i = 0; i < 10; i++) {
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            incrementCounter();
        }
    });

    // Should eventually be greater than or equal to 5
    ASSERT_EVENTUALLY_GE(counter.load(), 5);
    t.join();
}

TEST_F(XTestTest, TestEventuallyLE) {
    counter = 10;
    std::thread t([this]() {
        for (int i = 0; i < 5; i++) {
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            counter--;
        }
    });

    // Should eventually be less than or equal to 5
    ASSERT_EVENTUALLY_LE(counter.load(), 5);
    t.join();
}

TEST_F(XTestTest, TestEventuallyEQWithCustomTimeout) {
    std::thread t([this]() {
        std::this_thread::sleep_for(std::chrono::milliseconds(150));
        counter = 5;
    });

    // Should fail with default timeout (1s), but succeed with 2s timeout
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        counter.load(), 
        5,
        std::chrono::milliseconds(200),
        std::chrono::milliseconds(10)
    );
    t.join();
}

TEST_F(XTestTest, TestEventuallyGEWithCustomTimeout) {
    std::thread t([this]() {
        std::this_thread::sleep_for(std::chrono::milliseconds(150));
        counter = 10;
    });

    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        counter.load(),
        5,
        std::chrono::milliseconds(200),
        std::chrono::milliseconds(10)
    );
    t.join();
}

TEST_F(XTestTest, TestEventuallyLEWithCustomTimeout) {
    counter = 10;
    std::thread t([this]() {
        std::this_thread::sleep_for(std::chrono::milliseconds(150));
        counter = 3;
    });

    ASSERT_EVENTUALLY_LE_WITH_TIMEOUT(
        counter.load(),
        5,
        std::chrono::milliseconds(200),
        std::chrono::milliseconds(10)
    );
    t.join();
}
