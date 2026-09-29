// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "driver/bus/schedule.h"

namespace driver::bus {
TEST(Schedule, MakesEveryPeriodicMessageDueAtTheStart) {
    Schedule s(
        {x::telem::MILLISECOND * 10, std::nullopt, x::telem::MILLISECOND * 20},
        x::telem::TimeStamp(0)
    );
    EXPECT_EQ(s.next(), x::telem::TimeStamp(0));
    std::vector<std::size_t> due;
    s.due(x::telem::TimeStamp(0), due);
    EXPECT_EQ(due, (std::vector<std::size_t>{0, 2}));
    EXPECT_EQ(s.next(), x::telem::TimeStamp(x::telem::MILLISECOND * 10));
}

TEST(Schedule, MakesEachMessageDueOnItsPeriod) {
    Schedule s(
        {x::telem::MILLISECOND * 10, x::telem::MILLISECOND * 20},
        x::telem::TimeStamp(0)
    );
    std::vector<std::size_t> due;
    s.due(x::telem::TimeStamp(0), due);
    due.clear();
    s.due(x::telem::TimeStamp(x::telem::MILLISECOND * 10), due);
    EXPECT_EQ(due, (std::vector<std::size_t>{0}));
    due.clear();
    s.due(x::telem::TimeStamp(x::telem::MILLISECOND * 20), due);
    EXPECT_EQ(due, (std::vector<std::size_t>{0, 1}));
}

TEST(Schedule, SkipsTheSendsALateMessageMissed) {
    Schedule s({x::telem::MILLISECOND * 10}, x::telem::TimeStamp(0));
    std::vector<std::size_t> due;
    s.due(x::telem::TimeStamp(x::telem::MILLISECOND * 55), due);
    EXPECT_EQ(due.size(), 1);
    EXPECT_EQ(s.next(), x::telem::TimeStamp(x::telem::MILLISECOND * 65));
}

TEST(Schedule, HasNoNextTimeWithoutPeriods) {
    const Schedule s({std::nullopt}, x::telem::TimeStamp(0));
    EXPECT_EQ(s.next(), std::nullopt);
}
}
