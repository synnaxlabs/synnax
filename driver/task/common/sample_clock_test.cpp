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
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = 0,
        .k_i = 0,
        .k_d = 0
    });
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
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    constexpr double k_p = 0.1;
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = k_p,
        .k_i = 0,
        .k_d = 0
    });
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
    auto now_v = telem::SECOND * 5;
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = 0,
        .k_i = 0,
        .k_d = 0
    });
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
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = 0.5,
        .k_i = 0.1,
        .k_d = 0.1
    });
    breaker::Breaker b;

    // First sample - establish baseline
    const auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);

    // Simulate system running slower than expected
    now_v = telem::SECOND * 1 + telem::MILLISECOND * 100; // 100ms delay
    const auto end = clock.end();

    // The PID controller should attempt to correct for the delay
    // The exact value depends on the PID parameters, but it should be less than
    // the actual system time to compensate for the delay
    ASSERT_LT(end, telem::TimeStamp(now_v.nanoseconds()));
}

TEST(TestSampleClock, testHardwareTimedSampleClockConsecutiveCycles) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = 0,
        .k_i = 0,
        .k_d = 0
    });
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

TEST(TestSampleClock, testHardwareTimedSampleClockMaxBackCorrection) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    
    // Set a large P gain to ensure correction would exceed max if unconstrained
    constexpr double k_p = 2.0;
    constexpr double max_back_correction_factor = 0.1;  // 10% of period
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = k_p,
        .k_i = 0,
        .k_d = 0,
        .max_back_correction_factor = max_back_correction_factor
    });
    breaker::Breaker b;

    const auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);
    /// now is way way earlier than end.
    now_v = telem::MILLISECOND * 500;
    const auto end = clock.end();
    const auto expected_time = telem::TimeStamp(telem::MILLISECOND * 900);
    ASSERT_EQ(end, expected_time);
}
