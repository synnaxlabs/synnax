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

#include "arc/cpp/stl/math/math.h"

namespace arc::stl::math {
TEST(IntPow, BasicPower) {
    EXPECT_EQ(int_pow(2, 3), 8);
    EXPECT_EQ(int_pow(3, 4), 81);
    EXPECT_EQ(int_pow(5, 3), 125);
}

TEST(IntPow, ZeroExponent) {
    EXPECT_EQ(int_pow(2, 0), 1);
    EXPECT_EQ(int_pow(0, 0), 1);
    EXPECT_EQ(int_pow(100, 0), 1);
}

TEST(IntPow, ExponentOfOne) {
    EXPECT_EQ(int_pow(7, 1), 7);
    EXPECT_EQ(int_pow(0, 1), 0);
    EXPECT_EQ(int_pow(1, 1), 1);
}

TEST(IntPow, BaseOfZero) {
    EXPECT_EQ(int_pow(0, 1), 0);
    EXPECT_EQ(int_pow(0, 5), 0);
    EXPECT_EQ(int_pow(0, 10), 0);
}

TEST(IntPow, BaseOfOne) {
    EXPECT_EQ(int_pow(1, 0), 1);
    EXPECT_EQ(int_pow(1, 100), 1);
    EXPECT_EQ(int_pow(1, 1000), 1);
}

TEST(IntPow, LargeExponent) {
    EXPECT_EQ(int_pow(2, 10), 1024);
    EXPECT_EQ(int_pow(2, 20), 1048576);
}

TEST(IntPow, Uint8) {
    EXPECT_EQ(int_pow<uint8_t>(2, 7), 128);
    EXPECT_EQ(int_pow<uint8_t>(3, 3), 27);
}

TEST(IntPow, Int32) {
    EXPECT_EQ(int_pow<int32_t>(2, 30), 1073741824);
    EXPECT_EQ(int_pow<int32_t>(-2, 3), -8);
    EXPECT_EQ(int_pow<int32_t>(-2, 4), 16);
}

TEST(IntPow, Uint64) {
    EXPECT_EQ(int_pow<uint64_t>(2, 32), 4294967296ULL);
    EXPECT_EQ(int_pow<uint64_t>(10, 9), 1000000000ULL);
}

/// @brief Negative exponents on signed integer types: the loop condition
/// `i < exp` is immediately false when exp < 0, so the function returns 1.
TEST(IntPow, NegativeExponentSignedReturnsOne) {
    EXPECT_EQ(int_pow<int32_t>(2, -1), 1);
    EXPECT_EQ(int_pow<int32_t>(5, -3), 1);
    EXPECT_EQ(int_pow<int64_t>(10, -5), 1);
}
}
