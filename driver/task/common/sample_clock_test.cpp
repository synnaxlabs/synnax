// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// internal
#include "driver/task/common/sample_clock.h"

/// @brief it should correctly use the system clock to time samples.
TEST(TestSampleClock, testSoftwareTimedSampleClock) {
    auto clock = common::SoftwareTimedSampleClock(telem::HZ * 250);
    auto now = telem::TimeStamp::now();
    breaker::Breaker b;
    const auto start = clock.wait(b);
    EXPECT_GE(start, now + telem::MILLISECOND * 2);
    now = telem::TimeStamp::now();
    const auto end = clock.end(0);
    ASSERT_GE(end, now);
}

/// @brief it should correctly rely on steady sample spacing to time samples.
TEST(TestSampleClock, testHardwareTimedSampleClock) {
    auto clock = common::HardwareTimedSampleClock(telem::HZ * 1);
    clock.reset();
    const auto now = telem::TimeStamp::now();
    breaker::Breaker b;
    const auto start = clock.wait(b);
    ASSERT_GE(start, now);
    const auto end = clock.end(5);
    ASSERT_EQ(end, start + telem::SECOND * 4);
}

/// @brief it should correctly reset the hardware timed sample clock to the current
/// time.
TEST(TestSampleclock, testHardwareTimedSampleClockReset) {
    auto clock = common::HardwareTimedSampleClock(telem::HZ * 1);
    clock.reset();
    breaker::Breaker b;
    const auto start = clock.wait(b);
    const auto end = clock.end(5);
    ASSERT_EQ(end, start + telem::SECOND * 4);
    const auto start_2 = clock.wait(b);
    ASSERT_EQ(start_2,  start + telem::SECOND * 5);
    clock.reset();
    const auto start_3 = clock.wait(b);
    ASSERT_LT(start_3, start + telem::SECOND);
}