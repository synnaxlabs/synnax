// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/math/math.h"

TEST(FloorDiv, PositiveExact) {
    ASSERT_EQ(x::math::floor_div(10, 5), 2);
}

TEST(FloorDiv, PositiveWithRemainder) {
    ASSERT_EQ(x::math::floor_div(7, 3), 2);
}

TEST(FloorDiv, NegativeExact) {
    ASSERT_EQ(x::math::floor_div(-10, 5), -2);
}

TEST(FloorDiv, NegativeWithRemainder) {
    // C++ truncation: -7/3 = -2, but floor is -3
    ASSERT_EQ(x::math::floor_div(-7, 3), -3);
}

TEST(FloorDiv, NegativeOneRemainder) {
    // -1/3 truncates to 0 in C++, but floor is -1
    ASSERT_EQ(x::math::floor_div(-1, 3), -1);
}

TEST(FloorDiv, PositiveDividedByNegative) {
    // 7 / -3 truncates to -2 in C++, but floor is -3
    ASSERT_EQ(x::math::floor_div(7, -3), -3);
}

TEST(FloorDiv, NegativeDividedByNegative) {
    // -7 / -3 truncates to 2 in C++, floor is also 2
    ASSERT_EQ(x::math::floor_div(-7, -3), 2);
}

TEST(FloorDiv, ZeroDividend) {
    ASSERT_EQ(x::math::floor_div(0, 5), 0);
}

TEST(FloorDiv, ZeroDividendNegativeDivisor) {
    ASSERT_EQ(x::math::floor_div(0, -5), 0);
}

TEST(FloorDiv, DivideBySelf) {
    ASSERT_EQ(x::math::floor_div(7, 7), 1);
    ASSERT_EQ(x::math::floor_div(-7, -7), 1);
}

TEST(FloorDiv, DivideByOne) {
    ASSERT_EQ(x::math::floor_div(7, 1), 7);
    ASSERT_EQ(x::math::floor_div(-7, 1), -7);
}
