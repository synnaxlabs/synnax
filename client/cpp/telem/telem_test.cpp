// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/telem/telem.h"

#include "gtest/gtest.h"

using namespace synnax;

/// @brief - it should initialize a timestamp from a long.
TEST(TimeStampTests, testContructor) {
    auto ts = TimeStamp(5);
    ASSERT_EQ(ts.value, 5);
}

TEST(TimeSpanTests, testPeriod) {
    auto r = Rate(1);
    ASSERT_EQ(r.period().value, synnax::SECOND.value);
    auto r2 = Rate(2);
    ASSERT_EQ(r2.period().value, synnax::SECOND.value / 2);
}

TEST(TimeSpanTests, testConstructor) {
    auto ts = TimeSpan(5);
    ASSERT_EQ(ts.value, 5);
}

TEST(TimeSpanTests, testAddition) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(5);
    auto ts3 = ts + ts2;
    ASSERT_EQ(ts3.value, 10);
}

TEST(TimeSpanTests, testSubtraction) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(5);
    auto ts3 = ts - ts2;
    ASSERT_EQ(ts3.value, 0);
}

TEST(TimeSpanTests, testMultiplication) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(5);
    auto ts3 = ts * ts2;
    ASSERT_EQ(ts3.value, 25);

    auto ts4 = TimeSpan(5);
    auto ts5 = ts4 * 5;
    ASSERT_EQ(ts5.value, 25);

    auto ts6 = TimeSpan(5);
    auto ts7 = 5 * ts6;
    ASSERT_EQ(ts7.value, 25);

    auto ts8 = TimeSpan(5);
    auto ts9 = ts8 * 5.0;
    ASSERT_EQ(ts9.value, 25);

    auto ts10 = TimeSpan(5);
    auto ts11 = ts10 * 5.0f;
    ASSERT_EQ(ts11.value, 25);
}

TEST(TimeSpanTests, testDivision) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(5);
    auto ts3 = ts / ts2;
    ASSERT_EQ(ts3.value, 1);

    auto ts4 = TimeSpan(5);
    auto ts5 = ts4 / 5;
    ASSERT_EQ(ts5.value, 1);

    auto ts6 = TimeSpan(5);
    auto ts7 = 5 / ts6;
    ASSERT_EQ(ts7.value, 1);
}


TEST(TimeSpanTests, testEquality) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts == ts2);
}

TEST(TimeSpanTests, testInequality) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(6);
    ASSERT_TRUE(ts != ts2);
}

TEST(TimeSpanTests, testLessThan) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(6);
    ASSERT_TRUE(ts < ts2);
}

TEST(TimeSpanTests, testLessThanEqual) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts <= ts2);
}

TEST(TimeSpanTests, testGreaterThan) {
    auto ts = TimeSpan(6);
    auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts > ts2);
}

TEST(TimeSpanTests, testGreaterThanEqual) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts >= ts2);
}

TEST(TimeSpanTests, testModulo) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(2);
    auto ts3 = ts % ts2;
    ASSERT_EQ(ts3.value, 1);

    auto ts4 = TimeSpan(5);
    auto ts5 = 2 % ts4;
    ASSERT_EQ(ts5.value, 2);

    auto ts6 = TimeSpan(5);
    auto ts7 = ts6 % 2;
    ASSERT_EQ(ts7.value, 1);
}

TEST(TimeSpanTests, testTruncate) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(2);
    auto ts3 = ts.truncate(ts2);
    ASSERT_EQ(ts3.value, 4);
}

TEST(TimeSpanTests, testDelta) {
    auto ts = TimeSpan(5);
    auto ts2 = TimeSpan(2);
    auto ts3 = ts.delta(ts2);
    ASSERT_EQ(ts3.value, 3);
}

TEST(TimeStampTests, testAddition) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    auto ts3 = ts + ts2;
    ASSERT_EQ(ts3.value, 10);
}

TEST(TimeStampTests, testSubtraction) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    auto ts3 = ts - ts2;
    ASSERT_EQ(ts3.value, 0);
}

TEST(TimeStampTests, testMultiplication) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    auto ts3 = ts * ts2;
    ASSERT_EQ(ts3.value, 25);
}

TEST(TimeStampTests, testDivision) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    auto ts3 = ts / ts2;
    ASSERT_EQ(ts3.value, 1);
}

TEST(TimeStampTests, testEquality) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts == ts2);
}

TEST(TimeStampTests, testInequality) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(6);
    ASSERT_TRUE(ts != ts2);
}

TEST(TimeStampTests, testLessThan) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(6);
    ASSERT_TRUE(ts < ts2);
}

TEST(TimeStampTests, testLessThanEqual) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts <= ts2);
}

TEST(TimeStampTests, testGreaterThan) {
    auto ts = TimeStamp(6);
    auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts > ts2);
}

TEST(TimeStampTests, testGreaterThanEqual) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts >= ts2);
}

TEST(TimeStampTests, testModulo) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(2);
    auto ts3 = ts % ts2;
    ASSERT_EQ(ts3.value, 1);
}

// Test assignment compound operators like +=, -=, *=, and /=.

TEST(TimeStampTests, testAdditionAssignment) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    ts += ts2;
    ASSERT_EQ(ts.value, 10);
}

TEST(TimeStampTests, testSubtractionAssignment) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    ts -= ts2;
    ASSERT_EQ(ts.value, 0);
}

TEST(TimeStampTests, testMultiplicationAssignment) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    ts *= ts2;
    ASSERT_EQ(ts.value, 25);
}

TEST(TimeStampTests, testDivisionAssignment) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(5);
    ts /= ts2;
    ASSERT_EQ(ts.value, 1);
}

TEST(TimeStampTests, testModuloAssignment) {
    auto ts = TimeStamp(5);
    auto ts2 = TimeStamp(2);
    ts %= ts2;
    ASSERT_EQ(ts.value, 1);
}


TEST(TimeRangeTests, testContains) {
    auto tr = TimeRange(5, 10);
    auto ts = TimeStamp(7);
    ASSERT_TRUE(tr.contains(ts));
}

TEST(TimeRangeTests, testContainsRange) {
    auto tr = TimeRange(5, 10);
    auto tr2 = TimeRange(6, 9);
    ASSERT_TRUE(tr.contains(tr2));
}

TEST(TimeRangeTests, testEquality) {
    auto tr = TimeRange(5, 10);
    auto tr2 = TimeRange(5, 10);
    ASSERT_TRUE(tr == tr2);
}

TEST(RateTests, testContructor) {
    auto r = Rate(5);
    ASSERT_EQ(r.value, 5);
}

TEST(RateTests, testPeriod) {
    auto r = Rate(1);
    ASSERT_EQ(r.period().value, synnax::SECOND.value);
    auto r2 = Rate(2);
    ASSERT_EQ(r2.period().value, synnax::SECOND.value / 2);
}

TEST(RateTests, testAddition) {
    auto r = Rate(5);
    auto r2 = Rate(5);
    auto r3 = r + r2;
    ASSERT_EQ(r3.value, 10);
}

TEST(RateTests, testSubtraction) {
    auto r = Rate(5);
    auto r2 = Rate(5);
    auto r3 = r - r2;
    ASSERT_EQ(r3.value, 0);
}

TEST(RateTests, testMultiplication) {
    auto r = Rate(5);
    auto r2 = Rate(5);
    auto r3 = r * r2;
    ASSERT_EQ(r3.value, 25);

    auto r4 = Rate(5);
    auto r5 = r4 * 5;
    ASSERT_EQ(r5.value, 25);

    auto r6 = Rate(5);
    auto r7 = 5 * r6;
    ASSERT_EQ(r7.value, 25);

    auto r8 = Rate(5);
    auto r9 = r8 * 5.0;
    ASSERT_EQ(r9.value, 25);

    auto r10 = Rate(5);
    auto r11 = r10 * 5.0f;
    ASSERT_EQ(r11.value, 25);

    auto r12 = Rate(5);
    auto r13 = r12 * 5.0l;
    ASSERT_EQ(r13.value, 25);
}

TEST(RateTests, testDivision) {
    auto r = Rate(5);
    auto r2 = Rate(5);
    auto r3 = r / r2;
    ASSERT_EQ(r3.value, 1);

    auto r4 = Rate(5);
    auto r5 = r4 / 5;
    ASSERT_EQ(r5.value, 1);
}

TEST(RateTests, testEquality) {
    auto r = Rate(5);
    auto r2 = Rate(5);
    ASSERT_TRUE(r == r2);
}

TEST(RateTests, testInequality) {
    auto r = Rate(5);
    auto r2 = Rate(6);
    ASSERT_TRUE(r != r2);
}

TEST(RateTests, testLessThan) {
    auto r = Rate(5);
    auto r2 = Rate(6);
    ASSERT_TRUE(r < r2);
}

TEST(RateTests, testLessThanEqual) {
    auto r = Rate(5);
    auto r2 = Rate(5);
    ASSERT_TRUE(r <= r2);
}

TEST(RateTests, testGreaterThan) {
    auto r = Rate(6);
    auto r2 = Rate(5);
    ASSERT_TRUE(r > r2);
}

TEST(RateTests, testGreaterThanEqual) {
    auto r = Rate(5);
    auto r2 = Rate(5);
    ASSERT_TRUE(r >= r2);
}
