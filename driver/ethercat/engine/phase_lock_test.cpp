// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/engine/phase_lock.h"

namespace driver::ethercat::engine {
namespace {
const auto PERIOD = x::telem::MILLISECOND;
}

/// @brief it should not move a cycle whose frame passed half a period after a multiple
/// of the period from origin, before or after origin.
TEST(PhaseLock, HoldsAFrameAtTheTargetPhase) {
    const x::telem::TimeStamp origin(x::telem::SECOND);
    PhaseLock lock(PERIOD, origin);
    EXPECT_EQ(lock.update(origin + PERIOD * 7 + PERIOD / 2), x::telem::TimeSpan(0));
    EXPECT_EQ(lock.update(origin - PERIOD * 7 + PERIOD / 2), x::telem::TimeSpan(0));
    EXPECT_TRUE(lock.locked());
}

/// @brief it should delay the cycle after an early frame and advance it after a late
/// one, the short way around the period.
TEST(PhaseLock, MovesEachFrameTheShortWayToTheTargetPhase) {
    PhaseLock early(PERIOD, x::telem::TimeStamp(0));
    EXPECT_GT(early.update(x::telem::TimeStamp(PERIOD / 10)), x::telem::TimeSpan(0));
    PhaseLock late(PERIOD, x::telem::TimeStamp(0));
    EXPECT_LT(late.update(x::telem::TimeStamp(PERIOD * 0.9)), x::telem::TimeSpan(0));
}

/// @brief it should never move a cycle by more than a tenth of the period.
TEST(PhaseLock, MovesACycleAtMostATenthOfThePeriod) {
    PhaseLock lock(PERIOD, x::telem::TimeStamp(0));
    for (int i = 0; i < 10'000; ++i)
        EXPECT_LE(lock.update(x::telem::TimeStamp(1)).abs(), PERIOD / 10);
}

/// @brief it should report the lock lost once a frame passes more than a quarter
/// period from the target phase.
TEST(PhaseLock, LosesTheLockPastAQuarterPeriod) {
    PhaseLock lock(PERIOD, x::telem::TimeStamp(0));
    lock.update(x::telem::TimeStamp(PERIOD * 0.7));
    EXPECT_TRUE(lock.locked());
    lock.update(x::telem::TimeStamp(PERIOD * 0.8));
    EXPECT_FALSE(lock.locked());
}

/// @brief it should report a lost lock only after the lock held, so the frames before
/// the lock first holds are not late.
TEST(PhaseLock, ReportsALostLockOnlyAfterItHeld) {
    PhaseLock lock(PERIOD, x::telem::TimeStamp(0));
    lock.update(x::telem::TimeStamp(PERIOD * 0.9));
    EXPECT_FALSE(lock.locked());
    EXPECT_FALSE(lock.lost());
    lock.update(x::telem::TimeStamp(PERIOD / 2));
    EXPECT_FALSE(lock.lost());
    lock.update(x::telem::TimeStamp(PERIOD * 0.9));
    EXPECT_TRUE(lock.lost());
}

/// @brief it should bring frames to the target phase and hold them there while the DC
/// clock runs 100 ppm faster than the host clock.
TEST(PhaseLock, HoldsTheTargetPhaseThroughClockDrift) {
    const auto p = PERIOD.nanoseconds();
    PhaseLock lock(PERIOD, x::telem::TimeStamp(0));
    int64_t deadline = p / 10;
    int64_t sent = deadline - p;
    int64_t error = 0;
    for (int i = 0; i < 20'000; ++i) {
        const int64_t dc = sent + sent / 10'000;
        error = dc % p - p / 2;
        const auto shift = lock.update(x::telem::TimeStamp(dc));
        sent = deadline;
        deadline += p + shift.nanoseconds();
    }
    EXPECT_LT(std::abs(error), 1'000);
}
}
