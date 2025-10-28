// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include <thread>

#include "arc/cpp/runtime/scheduler/time_wheel.h"
#include "gtest/gtest.h"

using namespace arc;

TEST(TimeWheelTest, CalculateBasePeriod) {
    // Test GCD calculation
    std::vector<uint64_t> periods = {100'000'000, 250'000'000, 1'000'000'000};  // 100ms, 250ms, 1s
    uint64_t base = TimeWheel::calculate_base_period(periods);
    EXPECT_EQ(base, 50'000'000);  // GCD = 50ms

    // Test with minimum clamping
    std::vector<uint64_t> small_periods = {1'000'000, 3'000'000, 7'000'000};  // 1ms, 3ms, 7ms
    uint64_t clamped = TimeWheel::calculate_base_period(small_periods);
    EXPECT_EQ(clamped, 10'000'000);  // Clamped to 10ms minimum

    // Test empty periods
    std::vector<uint64_t> empty;
    uint64_t default_period = TimeWheel::calculate_base_period(empty);
    EXPECT_EQ(default_period, 10'000'000);  // Returns minimum

    // Test single period
    std::vector<uint64_t> single = {50'000'000};  // 50ms
    uint64_t single_result = TimeWheel::calculate_base_period(single);
    EXPECT_EQ(single_result, 50'000'000);
}

TEST(TimeWheelTest, ShouldTick) {
    // Create a TimeWheel with 50ms period
    TimeWheel tw(50'000'000);  // 50ms

    // Should not tick immediately (just constructed)
    EXPECT_FALSE(tw.should_tick());

    // Sleep for 60ms
    std::this_thread::sleep_for(std::chrono::milliseconds(60));

    // Should tick now
    EXPECT_TRUE(tw.should_tick());

    // Should not tick again immediately
    EXPECT_FALSE(tw.should_tick());

    // Sleep again
    std::this_thread::sleep_for(std::chrono::milliseconds(60));

    // Should tick again
    EXPECT_TRUE(tw.should_tick());
}

TEST(TimeWheelTest, BasePeriodGetter) {
    TimeWheel tw(123'456'789);
    EXPECT_EQ(tw.base_period_ns(), 123'456'789);
}
