// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/engine/clock.h"

namespace driver::ethercat::engine {
/// @brief it should map the first distributed clock time to the host time of its cycle.
TEST(Clock, AnchorsTheFirstTimeToTheHost) {
    Clock clock;
    EXPECT_EQ(
        clock.map(x::telem::TimeStamp(1'000), x::telem::TimeStamp(5'000)),
        x::telem::TimeStamp(5'000)
    );
}

/// @brief it should keep the spacing of the distributed clock when the host time
/// jitters.
TEST(Clock, KeepsDCSpacingThroughHostJitter) {
    Clock clock;
    const x::telem::TimeStamp start(x::telem::SECOND);
    const x::telem::TimeStamp dc(x::telem::MILLISECOND);
    clock.map(x::telem::TimeStamp(0), start);
    EXPECT_EQ(
        clock.map(dc, start + x::telem::MICROSECOND * 1'500),
        start + x::telem::MILLISECOND + x::telem::NANOSECOND * 500
    );
}

/// @brief it should anchor again to the host time when the host clock jumps.
TEST(Clock, ReanchorsWhenTheHostJumps) {
    Clock clock;
    const x::telem::TimeStamp start(x::telem::SECOND);
    clock.map(x::telem::TimeStamp(0), start);
    EXPECT_EQ(
        clock.map(x::telem::TimeStamp(x::telem::MILLISECOND), start + x::telem::SECOND),
        start + x::telem::SECOND
    );
}

/// @brief it should follow a host clock that runs faster than the distributed clock,
/// within a small error.
TEST(Clock, FollowsAHostClockThatRunsFaster) {
    Clock clock;
    x::telem::TimeSpan error;
    for (int64_t i = 0; i < 20'000; ++i) {
        const x::telem::TimeStamp host((x::telem::MILLISECOND + 100) * i);
        error = host - clock.map(x::telem::TimeStamp(x::telem::MILLISECOND * i), host);
    }
    EXPECT_LT(error.abs(), x::telem::MICROSECOND * 125);
}
}
