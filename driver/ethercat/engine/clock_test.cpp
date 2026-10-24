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
constexpr int64_t MS = 1'000'000;

TEST(Clock, AnchorsTheFirstTimeToTheHost) {
    Clock clock;
    EXPECT_EQ(
        clock.map(x::telem::TimeStamp(1'000), x::telem::TimeStamp(5'000)),
        x::telem::TimeStamp(5'000)
    );
}

TEST(Clock, KeepsDCSpacingThroughHostJitter) {
    Clock clock;
    clock.map(x::telem::TimeStamp(0), x::telem::TimeStamp(1'000 * MS));
    EXPECT_EQ(
        clock.map(x::telem::TimeStamp(MS), x::telem::TimeStamp(1'001 * MS + MS / 2)),
        x::telem::TimeStamp(1'001 * MS + MS / 2'000)
    );
}

TEST(Clock, ReanchorsWhenTheHostJumps) {
    Clock clock;
    clock.map(x::telem::TimeStamp(0), x::telem::TimeStamp(1'000 * MS));
    EXPECT_EQ(
        clock.map(x::telem::TimeStamp(MS), x::telem::TimeStamp(3'000 * MS)),
        x::telem::TimeStamp(3'000 * MS)
    );
}

TEST(Clock, FollowsAHostClockThatRunsFaster) {
    Clock clock;
    int64_t host = 0;
    int64_t error = 0;
    for (int64_t i = 0; i < 20'000; ++i) {
        host = i * (MS + 100);
        error = host - clock.map(x::telem::TimeStamp(i * MS), x::telem::TimeStamp(host))
                           .nanoseconds();
    }
    EXPECT_LT(error, MS / 8);
}
}
