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
    now_v = telem::SECOND * 2 + telem::MILLISECOND * 250;
    end = clock.end();
    ASSERT_EQ(
        telem::TimeSpan(end.nanoseconds()),
        telem::SECOND * 2 + telem::MILLISECOND * 2 + telem::MICROSECOND * 500
    );
}
