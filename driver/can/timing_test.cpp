// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/can/can.h"
#include "driver/can/timing.h"

namespace driver::can {
namespace {
const TimingLimits PCAN_NOMINAL{
    .tseg1_max = 256,
    .tseg2_max = 128,
    .sjw_max = 128,
    .brp_max = 1024,
};

const TimingLimits PCAN_DATA{
    .tseg1_max = 32,
    .tseg2_max = 16,
    .sjw_max = 16,
    .brp_max = 1024,
};

const TimingLimits BXCAN{
    .tseg1_max = 16,
    .tseg2_max = 8,
    .sjw_max = 4,
    .brp_max = 1024,
};

double sample_point(const Timing &t) {
    return static_cast<double>(1 + t.tseg1) / (1 + t.tseg1 + t.tseg2);
}

std::uint32_t bitrate(const std::uint32_t clock, const Timing &t) {
    return clock / (t.brp * (1 + t.tseg1 + t.tseg2));
}
}

TEST(Timing, ProducesTheNominalPhaseOfA500kBusOnAn80MHzClock) {
    const auto t = ASSERT_NIL_P(compute_timing(80000000, 500000, PCAN_NOMINAL));
    EXPECT_EQ(t.brp, 1);
    EXPECT_EQ(t.tseg1, 139);
    EXPECT_EQ(t.tseg2, 20);
    EXPECT_EQ(t.sjw, 20);
}

TEST(Timing, ProducesTheDataPhaseOfA2MBusOnAn80MHzClock) {
    const auto t = ASSERT_NIL_P(compute_timing(80000000, 2000000, PCAN_DATA));
    EXPECT_EQ(t.brp, 1);
    EXPECT_EQ(t.tseg1, 29);
    EXPECT_EQ(t.tseg2, 10);
    EXPECT_EQ(t.sjw, 10);
}

TEST(Timing, PrefersAnExactSamplePointOverASmallerPrescaler) {
    const auto t = ASSERT_NIL_P(compute_timing(48000000, 500000, BXCAN));
    EXPECT_EQ(t.brp, 6);
    EXPECT_EQ(t.tseg1, 13);
    EXPECT_EQ(t.tseg2, 2);
    EXPECT_DOUBLE_EQ(sample_point(t), 0.875);
}

TEST(Timing, ProducesEveryCommonBitrateExactlyOnA48MHzClock) {
    for (const std::uint32_t rate:
         {10000, 20000, 50000, 100000, 125000, 250000, 500000, 800000, 1000000}) {
        const auto t = ASSERT_NIL_P(compute_timing(48000000, rate, BXCAN));
        EXPECT_EQ(bitrate(48000000, t), rate);
        EXPECT_LE(t.tseg1, BXCAN.tseg1_max);
        EXPECT_LE(t.tseg2, BXCAN.tseg2_max);
        EXPECT_LE(t.sjw, BXCAN.sjw_max);
    }
}

TEST(Timing, UsesASeventyFivePercentSamplePointAbove800k) {
    const auto t = ASSERT_NIL_P(compute_timing(48000000, 1000000, BXCAN));
    EXPECT_DOUBLE_EQ(sample_point(t), 0.75);
}

TEST(Timing, RejectsABitrateTheClockCannotDivideExactly) {
    auto [t, err] = compute_timing(48000000, 333333, BXCAN);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "the adapter cannot produce a bitrate of 333333 bit/s from its 48000000 Hz "
        "clock"
    );
}
}
