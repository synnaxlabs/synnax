// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/clock_skew.h"

namespace {
// A mock global time variable to simulate custom time behavior.
static telem::TimeStamp mockTime(0);

// Custom "now()" function that returns the current mock time.
telem::TimeStamp customNow() {
    return mockTime;
}
}

TEST(ClockSkewCalculatorTest, SingleMeasurement) {
    // Reset mock time to 0 for a clean test.
    mockTime = telem::TimeStamp(0);

    // Create ClockSkewCalculator with our custom "now" function.
    telem::ClockSkewCalculator calc(customNow);

    // Local system starts measuring.
    calc.start(); // local_start_t = 0

    // Advance the local clock by 10 ns.
    mockTime = telem::TimeStamp(10);

    // Suppose the remote system's midpoint was 3 ns at the same "real" point in time.
    telem::TimeStamp remoteMidpoint(3);

    // Complete measurement. The local midpoint = 0 + (10 - 0)/2 = 5.
    // skew = local_midpoint - remote_midpoint = 5 - 3 = 2 ns.
    calc.end(remoteMidpoint);

    // Verify skew.
    EXPECT_EQ(calc.skew().nanoseconds(), 2);
    // Check "exceeds()" logic for a threshold of 1 and 3 ns.
    EXPECT_TRUE(calc.exceeds(telem::TimeSpan(1)));
    EXPECT_FALSE(calc.exceeds(telem::TimeSpan(3)));
}

TEST(ClockSkewCalculatorTest, ZeroSkewScenario) {
    // Reset mock time to 0.
    mockTime = telem::TimeStamp(0);
    telem::ClockSkewCalculator calc(customNow);

    // Pretend local and remote times match perfectly.
    calc.start();
    // Move local time forward by 1000.
    mockTime = telem::TimeStamp(1000);
    // If remote time is exactly the same midpoint (500 behind us from the start),
    // that forces the skew to be zero.
    calc.end(telem::TimeStamp(500));

    // Confirm zero skew.
    EXPECT_EQ(calc.skew().nanoseconds(), 0);
    // Confirm that 0 does not exceed any positive threshold.
    EXPECT_FALSE(calc.exceeds(telem::TimeSpan(1)));
}
