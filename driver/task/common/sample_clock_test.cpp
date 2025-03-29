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
    EXPECT_GE(start, now);
    now = telem::TimeStamp::now();
    const auto end = clock.end();
    ASSERT_GE(end, now);
}

/// @brief it should correctly rely on steady sample spacing to time samples.
TEST(TestSampleClock, testHardwareTimedSampleClockNominal) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v]() { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock(
        sample_rate,
        stream_rate,
        now_f
    );
    breaker::Breaker b;

    auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);
    now_v = telem::SECOND * 1;
    auto end = clock.end();
    ASSERT_EQ(end, telem::SECOND * 1);

    start = clock.wait(b);
    ASSERT_EQ(start, telem::SECOND * 1);
    now_v = telem::SECOND * 2;
    end = clock.end();
    ASSERT_EQ(end, telem::SECOND * 2);
}

TEST(TestSampleClock, testHardwareTimedSampleClockNowIsLater) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v]() { return telem::TimeStamp(now_v); };
    const double k_p = 0.1;
    auto clock = common::HardwareTimedSampleClock(
        sample_rate,
        stream_rate,
        now_f,
        k_p, // kp
        0, // ki
        0 // kd
    );
    breaker::Breaker b;

    auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);
    now_v = telem::SECOND * 1;
    auto end = clock.end();
    ASSERT_EQ(end, telem::SECOND * 1);

    start = clock.wait(b);
    ASSERT_EQ(start, telem::SECOND * 1);

    const auto skew = telem::MILLISECOND * 250;
    now_v = telem::SECOND * 2 + skew;
    end = clock.end();
    ASSERT_EQ(
        telem::TimeSpan(end.nanoseconds()),
        telem::SECOND * 2 + skew * k_p
    );
}

TEST(TestSampleClock, testHardwareTimedSampleClockReset) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = telem::SECOND * 5; // Start at non-zero time
    auto now_f = [&now_v]() { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock(sample_rate, stream_rate, now_f);
    breaker::Breaker b;

    // First cycle
    auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);
    now_v += telem::SECOND * 1;
    auto end = clock.end();

    // Reset clock
    clock.reset();

    // Verify reset state
    start = clock.wait(b);
    ASSERT_EQ(start, now_v); // Should use new current time after reset
    now_v += telem::SECOND * 1;
    end = clock.end();
    ASSERT_EQ(end, now_v);
}

TEST(TestSampleClock, testHardwareTimedSampleClockPIDCorrection) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v]() { return telem::TimeStamp(now_v); };

    // Use explicit PID values for testing
    auto clock = common::HardwareTimedSampleClock(
        sample_rate,
        stream_rate,
        now_f,
        0.5, // kp
        0.1, // ki
        0.1 // kd
    );
    breaker::Breaker b;

    // First sample - establish baseline
    auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);

    // Simulate system running slower than expected
    now_v = telem::SECOND * 1 + telem::MILLISECOND * 100; // 100ms delay
    auto end = clock.end();

    // The PID controller should attempt to correct for the delay
    // The exact value depends on the PID parameters, but it should be less than
    // the actual system time to compensate for the delay
    ASSERT_LT(end, telem::TimeStamp(now_v.nanoseconds()));
}

TEST(TestSampleClock, testHardwareTimedSampleClockConsecutiveCycles) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v]() { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock(
        sample_rate,
        stream_rate,
        now_f
    );
    breaker::Breaker b;

    // Test multiple consecutive cycles
    for (int i = 0; i < 3; i++) {
        auto start = clock.wait(b);
        ASSERT_EQ(start, now_v);
        now_v += telem::SECOND * 1; // Advance time by exactly one period
        auto end = clock.end();
        ASSERT_EQ(end, now_v);

        // Verify that the next start time matches the previous end time
        auto next_start = clock.wait(b);
        ASSERT_EQ(next_start, end);
    }
}
