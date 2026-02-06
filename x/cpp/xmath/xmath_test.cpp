// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>

#include "gtest/gtest.h"
#include "x/cpp/xmath/xmath.h"


TEST(FloorDivInt64, PositiveExact) {
    ASSERT_EQ(xmath::floor_div(int64_t(10), int64_t(5)), 2);
}

TEST(FloorDivInt64, PositiveWithRemainder) {
    ASSERT_EQ(xmath::floor_div(int64_t(7), int64_t(3)), 2);
}


TEST(FloorDivInt64, NegativeExact) {
    ASSERT_EQ(xmath::floor_div(int64_t(-10), int64_t(5)), -2);
}

TEST(FloorDivInt64, NegativeWithRemainder) {
    // C++ truncation: -7/3 = -2, but floor is -3
    ASSERT_EQ(xmath::floor_div(int64_t(-7), int64_t(3)), -3);
}

TEST(FloorDivInt64, NegativeOneRemainder) {
    // -1/3 truncates to 0 in C++, but floor is -1
    ASSERT_EQ(xmath::floor_div(int64_t(-1), int64_t(3)), -1);
}


TEST(FloorDivInt64, PositiveDividedByNegative) {
    // 7 / -3 truncates to -2 in C++, but floor is -3
    ASSERT_EQ(xmath::floor_div(int64_t(7), int64_t(-3)), -3);
}


TEST(FloorDivInt64, NegativeDividedByNegative) {
    // -7 / -3 truncates to 2 in C++, floor is also 2
    ASSERT_EQ(xmath::floor_div(int64_t(-7), int64_t(-3)), 2);
}


TEST(FloorDivInt64, ZeroDividend) {
    ASSERT_EQ(xmath::floor_div(int64_t(0), int64_t(5)), 0);
}

TEST(FloorDivInt64, ZeroDividendNegativeDivisor) {
    ASSERT_EQ(xmath::floor_div(int64_t(0), int64_t(-5)), 0);
}


TEST(FloorDivInt64, DivideBySelf) {
    ASSERT_EQ(xmath::floor_div(int64_t(7), int64_t(7)), 1);
    ASSERT_EQ(xmath::floor_div(int64_t(-7), int64_t(-7)), 1);
}

TEST(FloorDivInt64, DivideByOne) {
    ASSERT_EQ(xmath::floor_div(int64_t(7), int64_t(1)), 7);
    ASSERT_EQ(xmath::floor_div(int64_t(-7), int64_t(1)), -7);
}


TEST(FloorDivInt32, PositiveWithRemainder) {
    ASSERT_EQ(xmath::floor_div(int32_t(7), int32_t(3)), 2);
}

TEST(FloorDivInt32, NegativeWithRemainder) {
    ASSERT_EQ(xmath::floor_div(int32_t(-7), int32_t(3)), -3);
}

TEST(FloorDivInt32, PositiveDividedByNegative) {
    ASSERT_EQ(xmath::floor_div(int32_t(7), int32_t(-3)), -3);
}


TEST(FloorDivInt16, NegativeWithRemainder) {
    ASSERT_EQ(xmath::floor_div(int16_t(-7), int16_t(3)), int16_t(-3));
}

TEST(FloorDivInt16, PositiveDividedByNegative) {
    ASSERT_EQ(xmath::floor_div(int16_t(7), int16_t(-3)), int16_t(-3));
}


TEST(FloorDivInt8, NegativeWithRemainder) {
    ASSERT_EQ(xmath::floor_div(int8_t(-7), int8_t(3)), int8_t(-3));
}

TEST(FloorDivInt8, PositiveDividedByNegative) {
    ASSERT_EQ(xmath::floor_div(int8_t(7), int8_t(-3)), int8_t(-3));
}
