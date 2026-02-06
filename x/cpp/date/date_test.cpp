// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/date/date.h"

TEST(CivilFromDays, Epoch) {
    const date::Date dt = date::civil_from_days(0);
    ASSERT_EQ(dt.year, 1970);
    ASSERT_EQ(dt.month, 1);
    ASSERT_EQ(dt.day, 1);
}

TEST(CivilFromDays, LeapYearFeb29) {
    // 2000-02-29 = day 11016
    const date::Date dt = date::civil_from_days(11016);
    ASSERT_EQ(dt.year, 2000);
    ASSERT_EQ(dt.month, 2);
    ASSERT_EQ(dt.day, 29);
}

TEST(CivilFromDays, LeapYearMar1) {
    // 2000-03-01 = day 11017
    const date::Date dt = date::civil_from_days(11017);
    ASSERT_EQ(dt.year, 2000);
    ASSERT_EQ(dt.month, 3);
    ASSERT_EQ(dt.day, 1);
}

TEST(CivilFromDays, NegativeDay) {
    // 1969-12-31 = day -1
    const date::Date dt = date::civil_from_days(-1);
    ASSERT_EQ(dt.year, 1969);
    ASSERT_EQ(dt.month, 12);
    ASSERT_EQ(dt.day, 31);
}

TEST(CivilFromDays, Year2100NotLeap) {
    // 2100-03-01 = day 47541
    const date::Date dt = date::civil_from_days(47541);
    ASSERT_EQ(dt.year, 2100);
    ASSERT_EQ(dt.month, 3);
    ASSERT_EQ(dt.day, 1);
}
