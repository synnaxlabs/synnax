// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/clock_skew.h"

namespace x::telem {
/// @brief it should correctly calculate clock skew from a single measurement.
TEST(ClockSkewCalculatorTest, SingleMeasurement) {
    auto now = TimeStamp(0);
    ClockSkewCalculator calc([&now] { return now; });

    // Local system starts measuring.
    calc.start(); // local_start_t = 0

    // Advance the local clock by 10 ns.
    now = TimeStamp(10);

    // Suppose the remote system's midpoint was 3 ns at the same "real" point in time.
    const TimeStamp remoteMidpoint(3);

    // Complete measurement. The local midpoint = 0 + (10 - 0)/2 = 5.
    // skew = local_midpoint - remote_midpoint = 5 - 3 = 2 ns.
    calc.end(remoteMidpoint);

    // Verify skew.
    EXPECT_EQ(calc.skew().nanoseconds(), 2);
    // Check "exceeds()" logic for a threshold of 1 and 3 ns.
    EXPECT_TRUE(calc.exceeds(TimeSpan(1)));
    EXPECT_FALSE(calc.exceeds(TimeSpan(3)));
}

/// @brief it should report zero skew when local and remote times match perfectly.
TEST(ClockSkewCalculatorTest, ZeroSkewScenario) {
    auto now = TimeStamp(0);
    ClockSkewCalculator calc([&now] { return now; });

    // Pretend local and remote times match perfectly.
    calc.start();
    // Move local time forward by 1000.
    now = TimeStamp(1000);
    // If remote time is exactly the same midpoint (500 behind us from the start),
    // that forces the skew to be zero.
    calc.end(TimeStamp(500));

    // Confirm zero skew.
    EXPECT_EQ(calc.skew().nanoseconds(), 0);
    // Confirm that 0 does not exceed any positive threshold.
    EXPECT_FALSE(calc.exceeds(TimeSpan(1)));
}
}
