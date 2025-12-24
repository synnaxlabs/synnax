// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/series.h"

#include "arc/cpp/runtime/wasm/bindings.h"

namespace arc::runtime::wasm {

class BindingsTest : public ::testing::Test {
protected:
    void SetUp() override { bindings = std::make_unique<Bindings>(nullptr, nullptr); }

    std::unique_ptr<Bindings> bindings;
};

// ===== Series Creation Tests =====

TEST_F(BindingsTest, SeriesCreateEmptyF64) {
    const uint32_t handle = bindings->series_create_empty_f64(5);
    EXPECT_NE(handle, 0);
    EXPECT_EQ(bindings->series_len(handle), 5);
}

TEST_F(BindingsTest, SeriesCreateEmptyI32) {
    const uint32_t handle = bindings->series_create_empty_i32(10);
    EXPECT_NE(handle, 0);
    EXPECT_EQ(bindings->series_len(handle), 10);
}

TEST_F(BindingsTest, SeriesCreateEmptyU8) {
    const uint32_t handle = bindings->series_create_empty_u8(3);
    EXPECT_NE(handle, 0);
    EXPECT_EQ(bindings->series_len(handle), 3);
}

// ===== Element Access Tests =====

TEST_F(BindingsTest, SeriesSetAndIndexF64) {
    const uint32_t handle = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(handle, 0, 1.5);
    bindings->series_set_element_f64(handle, 1, 2.5);
    bindings->series_set_element_f64(handle, 2, 3.5);

    EXPECT_DOUBLE_EQ(bindings->series_index_f64(handle, 0), 1.5);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(handle, 1), 2.5);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(handle, 2), 3.5);
}

TEST_F(BindingsTest, SeriesSetAndIndexI32) {
    const uint32_t handle = bindings->series_create_empty_i32(4);
    bindings->series_set_element_i32(handle, 0, -10);
    bindings->series_set_element_i32(handle, 1, 0);
    bindings->series_set_element_i32(handle, 2, 10);
    bindings->series_set_element_i32(handle, 3, 100);

    EXPECT_EQ(bindings->series_index_i32(handle, 0), -10);
    EXPECT_EQ(bindings->series_index_i32(handle, 1), 0);
    EXPECT_EQ(bindings->series_index_i32(handle, 2), 10);
    EXPECT_EQ(bindings->series_index_i32(handle, 3), 100);
}

// ===== Scalar Arithmetic Tests =====

TEST_F(BindingsTest, SeriesElementAddF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 2.0);
    bindings->series_set_element_f64(h1, 2, 3.0);

    const uint32_t h2 = bindings->series_element_add_f64(h1, 10.0);
    EXPECT_NE(h2, 0);
    EXPECT_NE(h2, h1);

    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 0), 11.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 1), 12.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 2), 13.0);

    // Original unchanged
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h1, 0), 1.0);
}

TEST_F(BindingsTest, SeriesElementMulI32) {
    const uint32_t h1 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h1, 0, 2);
    bindings->series_set_element_i32(h1, 1, 3);
    bindings->series_set_element_i32(h1, 2, 4);

    const uint32_t h2 = bindings->series_element_mul_i32(h1, 5);
    EXPECT_EQ(bindings->series_index_i32(h2, 0), 10);
    EXPECT_EQ(bindings->series_index_i32(h2, 1), 15);
    EXPECT_EQ(bindings->series_index_i32(h2, 2), 20);
}

TEST_F(BindingsTest, SeriesElementSubF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h1, 0, 10.0);
    bindings->series_set_element_f64(h1, 1, 20.0);

    const uint32_t h2 = bindings->series_element_sub_f64(h1, 5.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 0), 5.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 1), 15.0);
}

TEST_F(BindingsTest, SeriesElementDivF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h1, 0, 10.0);
    bindings->series_set_element_f64(h1, 1, 20.0);

    const uint32_t h2 = bindings->series_element_div_f64(h1, 2.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 0), 5.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 1), 10.0);
}

// ===== Reverse Scalar Arithmetic Tests =====

TEST_F(BindingsTest, SeriesElementRsubF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 2.0);
    bindings->series_set_element_f64(h1, 2, 3.0);

    // 10.0 - [1, 2, 3] = [9, 8, 7]
    const uint32_t h2 = bindings->series_element_rsub_f64(h1, 10.0);
    EXPECT_NE(h2, 0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 0), 9.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 1), 8.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 2), 7.0);

    // Original unchanged
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h1, 0), 1.0);
}

TEST_F(BindingsTest, SeriesElementRdivF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 2.0);
    bindings->series_set_element_f64(h1, 1, 4.0);
    bindings->series_set_element_f64(h1, 2, 5.0);

    // 10.0 / [2, 4, 5] = [5, 2.5, 2]
    const uint32_t h2 = bindings->series_element_rdiv_f64(h1, 10.0);
    EXPECT_NE(h2, 0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 0), 5.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 1), 2.5);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 2), 2.0);
}

TEST_F(BindingsTest, SeriesElementRsubI32) {
    const uint32_t h1 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h1, 0, 5);
    bindings->series_set_element_i32(h1, 1, 10);
    bindings->series_set_element_i32(h1, 2, 15);

    // 100 - [5, 10, 15] = [95, 90, 85]
    const uint32_t h2 = bindings->series_element_rsub_i32(h1, 100);
    EXPECT_EQ(bindings->series_index_i32(h2, 0), 95);
    EXPECT_EQ(bindings->series_index_i32(h2, 1), 90);
    EXPECT_EQ(bindings->series_index_i32(h2, 2), 85);
}

// ===== Modulo Tests =====

TEST_F(BindingsTest, SeriesElementModI32) {
    const uint32_t h1 = bindings->series_create_empty_i32(4);
    bindings->series_set_element_i32(h1, 0, 10);
    bindings->series_set_element_i32(h1, 1, 15);
    bindings->series_set_element_i32(h1, 2, 20);
    bindings->series_set_element_i32(h1, 3, 7);

    // [10, 15, 20, 7] % 3 = [1, 0, 2, 1]
    const uint32_t h2 = bindings->series_element_mod_i32(h1, 3);
    EXPECT_NE(h2, 0);
    EXPECT_EQ(bindings->series_index_i32(h2, 0), 1);
    EXPECT_EQ(bindings->series_index_i32(h2, 1), 0);
    EXPECT_EQ(bindings->series_index_i32(h2, 2), 2);
    EXPECT_EQ(bindings->series_index_i32(h2, 3), 1);
}

TEST_F(BindingsTest, SeriesElementModU64) {
    const uint32_t h1 = bindings->series_create_empty_u64(3);
    bindings->series_set_element_u64(h1, 0, 100);
    bindings->series_set_element_u64(h1, 1, 250);
    bindings->series_set_element_u64(h1, 2, 17);

    // [100, 250, 17] % 7 = [2, 5, 3]
    const uint32_t h2 = bindings->series_element_mod_u64(h1, 7);
    EXPECT_EQ(bindings->series_index_u64(h2, 0), 2);
    EXPECT_EQ(bindings->series_index_u64(h2, 1), 5);
    EXPECT_EQ(bindings->series_index_u64(h2, 2), 3);
}

TEST_F(BindingsTest, SeriesElementModDivisionByZero) {
    const uint32_t h1 = bindings->series_create_empty_i32(1);
    bindings->series_set_element_i32(h1, 0, 10);

    // Should return 0 (invalid handle) for division by zero
    EXPECT_EQ(bindings->series_element_mod_i32(h1, 0), 0);
}

TEST_F(BindingsTest, SeriesSeriesModI32) {
    const uint32_t h1 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h1, 0, 10);
    bindings->series_set_element_i32(h1, 1, 15);
    bindings->series_set_element_i32(h1, 2, 23);

    const uint32_t h2 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h2, 0, 3);
    bindings->series_set_element_i32(h2, 1, 4);
    bindings->series_set_element_i32(h2, 2, 5);

    // [10, 15, 23] % [3, 4, 5] = [1, 3, 3]
    const uint32_t h3 = bindings->series_series_mod_i32(h1, h2);
    EXPECT_NE(h3, 0);
    EXPECT_EQ(bindings->series_index_i32(h3, 0), 1);
    EXPECT_EQ(bindings->series_index_i32(h3, 1), 3);
    EXPECT_EQ(bindings->series_index_i32(h3, 2), 3);
}

TEST_F(BindingsTest, SeriesSeriesModU32DifferentLengths) {
    const uint32_t h1 = bindings->series_create_empty_u32(4);
    bindings->series_set_element_u32(h1, 0, 10);
    bindings->series_set_element_u32(h1, 1, 20);
    bindings->series_set_element_u32(h1, 2, 30);
    bindings->series_set_element_u32(h1, 3, 40);

    const uint32_t h2 = bindings->series_create_empty_u32(2);
    bindings->series_set_element_u32(h2, 0, 3);
    bindings->series_set_element_u32(h2, 1, 7); // 7 repeats for remaining elements

    // [10, 20, 30, 40] % [3, 7, 7, 7] = [1, 6, 2, 5]
    const uint32_t h3 = bindings->series_series_mod_u32(h1, h2);
    EXPECT_EQ(bindings->series_len(h3), 4);
    EXPECT_EQ(bindings->series_index_u32(h3, 0), 1);
    EXPECT_EQ(bindings->series_index_u32(h3, 1), 6);
    EXPECT_EQ(bindings->series_index_u32(h3, 2), 2);
    EXPECT_EQ(bindings->series_index_u32(h3, 3), 5);
}

TEST_F(BindingsTest, SeriesElementModF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 10.5);
    bindings->series_set_element_f64(h1, 1, 7.5);
    bindings->series_set_element_f64(h1, 2, 15.0);

    // [10.5, 7.5, 15.0] % 3.0 = [1.5, 1.5, 0.0] (using std::fmod)
    const uint32_t h2 = bindings->series_element_mod_f64(h1, 3.0);
    EXPECT_NE(h2, 0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 0), 1.5);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 1), 1.5);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 2), 0.0);
}

TEST_F(BindingsTest, SeriesSeriesModF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 10.5);
    bindings->series_set_element_f64(h1, 1, 20.0);
    bindings->series_set_element_f64(h1, 2, 7.5);

    const uint32_t h2 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h2, 0, 3.0);
    bindings->series_set_element_f64(h2, 1, 6.0);
    bindings->series_set_element_f64(h2, 2, 2.5);

    // [10.5, 20.0, 7.5] % [3.0, 6.0, 2.5] = [1.5, 2.0, 0.0]
    const uint32_t h3 = bindings->series_series_mod_f64(h1, h2);
    EXPECT_NE(h3, 0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 0), 1.5);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 1), 2.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 2), 0.0);
}

// ===== Series-Series Arithmetic Tests (Same Length) =====

TEST_F(BindingsTest, SeriesSeriesAddF64SameLength) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 2.0);
    bindings->series_set_element_f64(h1, 2, 3.0);

    const uint32_t h2 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h2, 0, 10.0);
    bindings->series_set_element_f64(h2, 1, 20.0);
    bindings->series_set_element_f64(h2, 2, 30.0);

    const uint32_t h3 = bindings->series_series_add_f64(h1, h2);
    EXPECT_EQ(bindings->series_len(h3), 3);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 0), 11.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 1), 22.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 2), 33.0);
}

TEST_F(BindingsTest, SeriesSeriesMulI64SameLength) {
    const uint32_t h1 = bindings->series_create_empty_i64(2);
    bindings->series_set_element_i64(h1, 0, 3);
    bindings->series_set_element_i64(h1, 1, 4);

    const uint32_t h2 = bindings->series_create_empty_i64(2);
    bindings->series_set_element_i64(h2, 0, 5);
    bindings->series_set_element_i64(h2, 1, 6);

    const uint32_t h3 = bindings->series_series_mul_i64(h1, h2);
    EXPECT_EQ(bindings->series_index_i64(h3, 0), 15);
    EXPECT_EQ(bindings->series_index_i64(h3, 1), 24);
}

// ===== Series-Series Arithmetic Tests (Different Lengths - Last Value Repetition)
// =====

TEST_F(BindingsTest, SeriesSeriesAddF64DifferentLengthsLHSLonger) {
    // lhs = [1, 2, 3, 4, 5]
    const uint32_t h1 = bindings->series_create_empty_f64(5);
    for (uint32_t i = 0; i < 5; i++) {
        bindings->series_set_element_f64(h1, i, static_cast<double>(i + 1));
    }

    // rhs = [10, 20] -> repeats last value (20) for remaining positions
    const uint32_t h2 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h2, 0, 10.0);
    bindings->series_set_element_f64(h2, 1, 20.0);

    const uint32_t h3 = bindings->series_series_add_f64(h1, h2);
    EXPECT_EQ(bindings->series_len(h3), 5);
    // Result: [1+10, 2+20, 3+20, 4+20, 5+20] = [11, 22, 23, 24, 25]
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 0), 11.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 1), 22.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 2), 23.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 3), 24.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 4), 25.0);
}

TEST_F(BindingsTest, SeriesSeriesAddF64DifferentLengthsRHSLonger) {
    // lhs = [1, 2] -> repeats last value (2) for remaining positions
    const uint32_t h1 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 2.0);

    // rhs = [10, 20, 30, 40]
    const uint32_t h2 = bindings->series_create_empty_f64(4);
    for (uint32_t i = 0; i < 4; i++) {
        bindings->series_set_element_f64(h2, i, static_cast<double>((i + 1) * 10));
    }

    const uint32_t h3 = bindings->series_series_add_f64(h1, h2);
    EXPECT_EQ(bindings->series_len(h3), 4);
    // Result: [1+10, 2+20, 2+30, 2+40] = [11, 22, 32, 42]
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 0), 11.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 1), 22.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 2), 32.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 3), 42.0);
}

TEST_F(BindingsTest, SeriesSeriesSubI32DifferentLengths) {
    const uint32_t h1 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h1, 0, 100);
    bindings->series_set_element_i32(h1, 1, 200);
    bindings->series_set_element_i32(h1, 2, 300);

    const uint32_t h2 = bindings->series_create_empty_i32(1);
    bindings->series_set_element_i32(h2, 0, 10);

    const uint32_t h3 = bindings->series_series_sub_i32(h1, h2);
    // Result: [100-10, 200-10, 300-10] = [90, 190, 290]
    EXPECT_EQ(bindings->series_len(h3), 3);
    EXPECT_EQ(bindings->series_index_i32(h3, 0), 90);
    EXPECT_EQ(bindings->series_index_i32(h3, 1), 190);
    EXPECT_EQ(bindings->series_index_i32(h3, 2), 290);
}

TEST_F(BindingsTest, SeriesSeriesDivF64DifferentLengths) {
    const uint32_t h1 = bindings->series_create_empty_f64(4);
    bindings->series_set_element_f64(h1, 0, 10.0);
    bindings->series_set_element_f64(h1, 1, 20.0);
    bindings->series_set_element_f64(h1, 2, 30.0);
    bindings->series_set_element_f64(h1, 3, 40.0);

    const uint32_t h2 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h2, 0, 2.0);
    bindings->series_set_element_f64(h2, 1, 4.0);

    const uint32_t h3 = bindings->series_series_div_f64(h1, h2);
    // Result: [10/2, 20/4, 30/4, 40/4] = [5, 5, 7.5, 10]
    EXPECT_EQ(bindings->series_len(h3), 4);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 0), 5.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 1), 5.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 2), 7.5);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 3), 10.0);
}

// ===== Comparison Tests =====

TEST_F(BindingsTest, SeriesCompareGtF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 5.0);
    bindings->series_set_element_f64(h1, 2, 3.0);

    const uint32_t h2 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h2, 0, 2.0);
    bindings->series_set_element_f64(h2, 1, 3.0);
    bindings->series_set_element_f64(h2, 2, 3.0);

    const uint32_t h3 = bindings->series_compare_gt_f64(h1, h2);
    EXPECT_EQ(bindings->series_len(h3), 3);
    // 1 > 2? false, 5 > 3? true, 3 > 3? false
    EXPECT_EQ(bindings->series_index_u8(h3, 0), 0);
    EXPECT_EQ(bindings->series_index_u8(h3, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 2), 0);
}

TEST_F(BindingsTest, SeriesCompareLtI32) {
    const uint32_t h1 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h1, 0, 1);
    bindings->series_set_element_i32(h1, 1, 5);
    bindings->series_set_element_i32(h1, 2, 3);

    const uint32_t h2 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h2, 0, 2);
    bindings->series_set_element_i32(h2, 1, 3);
    bindings->series_set_element_i32(h2, 2, 3);

    const uint32_t h3 = bindings->series_compare_lt_i32(h1, h2);
    // 1 < 2? true, 5 < 3? false, 3 < 3? false
    EXPECT_EQ(bindings->series_index_u8(h3, 0), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 1), 0);
    EXPECT_EQ(bindings->series_index_u8(h3, 2), 0);
}

TEST_F(BindingsTest, SeriesCompareGeF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 3.0);
    bindings->series_set_element_f64(h1, 2, 5.0);

    const uint32_t h2 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h2, 0, 2.0);
    bindings->series_set_element_f64(h2, 1, 3.0);
    bindings->series_set_element_f64(h2, 2, 4.0);

    const uint32_t h3 = bindings->series_compare_ge_f64(h1, h2);
    // 1 >= 2? false, 3 >= 3? true, 5 >= 4? true
    EXPECT_EQ(bindings->series_index_u8(h3, 0), 0);
    EXPECT_EQ(bindings->series_index_u8(h3, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 2), 1);
}

TEST_F(BindingsTest, SeriesCompareLeI64) {
    const uint32_t h1 = bindings->series_create_empty_i64(3);
    bindings->series_set_element_i64(h1, 0, 1);
    bindings->series_set_element_i64(h1, 1, 3);
    bindings->series_set_element_i64(h1, 2, 5);

    const uint32_t h2 = bindings->series_create_empty_i64(3);
    bindings->series_set_element_i64(h2, 0, 2);
    bindings->series_set_element_i64(h2, 1, 3);
    bindings->series_set_element_i64(h2, 2, 4);

    const uint32_t h3 = bindings->series_compare_le_i64(h1, h2);
    // 1 <= 2? true, 3 <= 3? true, 5 <= 4? false
    EXPECT_EQ(bindings->series_index_u8(h3, 0), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 2), 0);
}

TEST_F(BindingsTest, SeriesCompareEqU32) {
    const uint32_t h1 = bindings->series_create_empty_u32(3);
    bindings->series_set_element_u32(h1, 0, 1);
    bindings->series_set_element_u32(h1, 1, 3);
    bindings->series_set_element_u32(h1, 2, 5);

    const uint32_t h2 = bindings->series_create_empty_u32(3);
    bindings->series_set_element_u32(h2, 0, 1);
    bindings->series_set_element_u32(h2, 1, 3);
    bindings->series_set_element_u32(h2, 2, 4);

    const uint32_t h3 = bindings->series_compare_eq_u32(h1, h2);
    // 1 == 1? true, 3 == 3? true, 5 == 4? false
    EXPECT_EQ(bindings->series_index_u8(h3, 0), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 2), 0);
}

TEST_F(BindingsTest, SeriesCompareNeF32) {
    const uint32_t h1 = bindings->series_create_empty_f32(3);
    bindings->series_set_element_f32(h1, 0, 1.0f);
    bindings->series_set_element_f32(h1, 1, 3.0f);
    bindings->series_set_element_f32(h1, 2, 5.0f);

    const uint32_t h2 = bindings->series_create_empty_f32(3);
    bindings->series_set_element_f32(h2, 0, 1.0f);
    bindings->series_set_element_f32(h2, 1, 3.0f);
    bindings->series_set_element_f32(h2, 2, 4.0f);

    const uint32_t h3 = bindings->series_compare_ne_f32(h1, h2);
    // 1 != 1? false, 3 != 3? false, 5 != 4? true
    EXPECT_EQ(bindings->series_index_u8(h3, 0), 0);
    EXPECT_EQ(bindings->series_index_u8(h3, 1), 0);
    EXPECT_EQ(bindings->series_index_u8(h3, 2), 1);
}

// ===== Comparison with Different Lengths =====

TEST_F(BindingsTest, SeriesCompareGtDifferentLengths) {
    // lhs = [1, 2, 3, 4, 5]
    const uint32_t h1 = bindings->series_create_empty_f64(5);
    for (uint32_t i = 0; i < 5; i++) {
        bindings->series_set_element_f64(h1, i, static_cast<double>(i + 1));
    }

    // rhs = [2, 1] -> last value (1) repeats
    const uint32_t h2 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h2, 0, 2.0);
    bindings->series_set_element_f64(h2, 1, 1.0);

    const uint32_t h3 = bindings->series_compare_gt_f64(h1, h2);
    EXPECT_EQ(bindings->series_len(h3), 5);
    // 1 > 2? false, 2 > 1? true, 3 > 1? true, 4 > 1? true, 5 > 1? true
    EXPECT_EQ(bindings->series_index_u8(h3, 0), 0);
    EXPECT_EQ(bindings->series_index_u8(h3, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 2), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 3), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 4), 1);
}

// ===== Series Len Tests =====

TEST_F(BindingsTest, SeriesLenInvalidHandle) {
    EXPECT_EQ(bindings->series_len(999), 0);
}

TEST_F(BindingsTest, SeriesLenValidHandle) {
    const uint32_t h = bindings->series_create_empty_i32(7);
    EXPECT_EQ(bindings->series_len(h), 7);
}

// ===== Series Slice Tests =====

TEST_F(BindingsTest, SeriesSliceBasic) {
    const uint32_t h1 = bindings->series_create_empty_f64(5);
    for (uint32_t i = 0; i < 5; i++) {
        bindings->series_set_element_f64(h1, i, static_cast<double>(i * 10));
    }

    const uint32_t h2 = bindings->series_slice(h1, 1, 4);
    EXPECT_NE(h2, 0);
    EXPECT_EQ(bindings->series_len(h2), 3);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 0), 10.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 1), 20.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 2), 30.0);
}

TEST_F(BindingsTest, SeriesSliceInvalidBounds) {
    const uint32_t h1 = bindings->series_create_empty_f64(5);
    EXPECT_EQ(bindings->series_slice(h1, 3, 2), 0); // start > end
    EXPECT_EQ(bindings->series_slice(h1, 0, 10), 0); // end > size
    EXPECT_EQ(bindings->series_slice(h1, 10, 15), 0); // start > size
}

// ===== State Persistence Tests =====

TEST_F(BindingsTest, StateLoadStoreSeriesF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 2.0);
    bindings->series_set_element_f64(h1, 2, 3.0);

    // First load returns init_handle since nothing stored yet
    const uint32_t loaded1 = bindings->state_load_series_f64(1, 1, h1);
    EXPECT_EQ(loaded1, h1);

    // Store the series
    bindings->state_store_series_f64(1, 1, h1);

    // Create a different series as init value
    const uint32_t h2 = bindings->series_create_empty_f64(1);
    bindings->series_set_element_f64(h2, 0, 999.0);

    // Load should return stored series, not init
    const uint32_t loaded2 = bindings->state_load_series_f64(1, 1, h2);
    EXPECT_NE(loaded2, h2); // Should not be init value
    EXPECT_EQ(bindings->series_len(loaded2), 3);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded2, 0), 1.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded2, 1), 2.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded2, 2), 3.0);
}

TEST_F(BindingsTest, StateLoadStoreSeriesI32) {
    const uint32_t h1 = bindings->series_create_empty_i32(2);
    bindings->series_set_element_i32(h1, 0, 42);
    bindings->series_set_element_i32(h1, 1, -42);

    bindings->state_store_series_i32(2, 2, h1);

    const uint32_t dummy = bindings->series_create_empty_i32(1);
    const uint32_t loaded = bindings->state_load_series_i32(2, 2, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 2);
    EXPECT_EQ(bindings->series_index_i32(loaded, 0), 42);
    EXPECT_EQ(bindings->series_index_i32(loaded, 1), -42);
}

// ===== Edge Case Tests =====

TEST_F(BindingsTest, SeriesOperationsOnInvalidHandle) {
    EXPECT_EQ(bindings->series_index_f64(999, 0), 0.0);
    EXPECT_EQ(bindings->series_element_add_f64(999, 1.0), 0);
    EXPECT_EQ(bindings->series_series_add_f64(999, 888), 0);
    EXPECT_EQ(bindings->series_compare_gt_f64(999, 888), 0);
}

TEST_F(BindingsTest, SeriesEmptyOperations) {
    const uint32_t h1 = bindings->series_create_empty_f64(0);
    const uint32_t h2 = bindings->series_create_empty_f64(0);

    const uint32_t h3 = bindings->series_series_add_f64(h1, h2);
    EXPECT_EQ(bindings->series_len(h3), 0);
}

TEST_F(BindingsTest, SeriesSingleElementOperations) {
    const uint32_t h1 = bindings->series_create_empty_f64(1);
    bindings->series_set_element_f64(h1, 0, 5.0);

    const uint32_t h2 = bindings->series_create_empty_f64(1);
    bindings->series_set_element_f64(h2, 0, 3.0);

    const uint32_t h3 = bindings->series_series_add_f64(h1, h2);
    EXPECT_EQ(bindings->series_len(h3), 1);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h3, 0), 8.0);
}

// ===== All Type Tests (Smoke Tests) =====

TEST_F(BindingsTest, AllTypesCreateAndIndex) {
    // u8
    {
        const uint32_t h = bindings->series_create_empty_u8(1);
        bindings->series_set_element_u8(h, 0, 255);
        EXPECT_EQ(bindings->series_index_u8(h, 0), 255);
    }
    // u16
    {
        const uint32_t h = bindings->series_create_empty_u16(1);
        bindings->series_set_element_u16(h, 0, 65535);
        EXPECT_EQ(bindings->series_index_u16(h, 0), 65535);
    }
    // u32
    {
        const uint32_t h = bindings->series_create_empty_u32(1);
        bindings->series_set_element_u32(h, 0, 4294967295U);
        EXPECT_EQ(bindings->series_index_u32(h, 0), 4294967295U);
    }
    // u64
    {
        const uint32_t h = bindings->series_create_empty_u64(1);
        bindings->series_set_element_u64(h, 0, 18446744073709551615ULL);
        EXPECT_EQ(bindings->series_index_u64(h, 0), 18446744073709551615ULL);
    }
    // i8
    {
        const uint32_t h = bindings->series_create_empty_i8(1);
        bindings->series_set_element_i8(h, 0, -128);
        EXPECT_EQ(bindings->series_index_i8(h, 0), -128);
    }
    // i16
    {
        const uint32_t h = bindings->series_create_empty_i16(1);
        bindings->series_set_element_i16(h, 0, -32768);
        EXPECT_EQ(bindings->series_index_i16(h, 0), -32768);
    }
    // i32
    {
        const uint32_t h = bindings->series_create_empty_i32(1);
        bindings->series_set_element_i32(h, 0, -2147483647);
        EXPECT_EQ(bindings->series_index_i32(h, 0), -2147483647);
    }
    // i64
    {
        const uint32_t h = bindings->series_create_empty_i64(1);
        bindings->series_set_element_i64(h, 0, -9223372036854775807LL);
        EXPECT_EQ(bindings->series_index_i64(h, 0), -9223372036854775807LL);
    }
    // f32
    {
        const uint32_t h = bindings->series_create_empty_f32(1);
        bindings->series_set_element_f32(h, 0, 3.14159f);
        EXPECT_FLOAT_EQ(bindings->series_index_f32(h, 0), 3.14159f);
    }
    // f64
    {
        const uint32_t h = bindings->series_create_empty_f64(1);
        bindings->series_set_element_f64(h, 0, 3.14159265358979);
        EXPECT_DOUBLE_EQ(bindings->series_index_f64(h, 0), 3.14159265358979);
    }
}

// ===== String Tests =====

TEST_F(BindingsTest, StringCreate) {
    const uint32_t h = bindings->string_create("hello");
    EXPECT_NE(h, 0);
    EXPECT_EQ(bindings->string_get(h), "hello");
}

TEST_F(BindingsTest, StringLen) {
    const uint32_t h = bindings->string_create("hello");
    EXPECT_EQ(bindings->string_len(h), 5);

    const uint32_t h2 = bindings->string_create("");
    EXPECT_EQ(bindings->string_len(h2), 0);

    EXPECT_EQ(bindings->string_len(999), 0); // invalid handle
}

TEST_F(BindingsTest, StringConcat) {
    const uint32_t h1 = bindings->string_create("hello");
    const uint32_t h2 = bindings->string_create(" world");
    const uint32_t h3 = bindings->string_concat(h1, h2);

    EXPECT_NE(h3, 0);
    EXPECT_NE(h3, h1);
    EXPECT_NE(h3, h2);
    EXPECT_EQ(bindings->string_get(h3), "hello world");

    // Original strings unchanged
    EXPECT_EQ(bindings->string_get(h1), "hello");
    EXPECT_EQ(bindings->string_get(h2), " world");
}

TEST_F(BindingsTest, StringConcatEmpty) {
    const uint32_t h1 = bindings->string_create("hello");
    const uint32_t h2 = bindings->string_create("");
    const uint32_t h3 = bindings->string_concat(h1, h2);

    EXPECT_EQ(bindings->string_get(h3), "hello");
}

TEST_F(BindingsTest, StringConcatInvalidHandle) {
    const uint32_t h1 = bindings->string_create("hello");
    EXPECT_EQ(bindings->string_concat(h1, 999), 0);
    EXPECT_EQ(bindings->string_concat(999, h1), 0);
    EXPECT_EQ(bindings->string_concat(999, 888), 0);
}

TEST_F(BindingsTest, StringEqual) {
    const uint32_t h1 = bindings->string_create("hello");
    const uint32_t h2 = bindings->string_create("hello");
    const uint32_t h3 = bindings->string_create("world");

    EXPECT_EQ(bindings->string_equal(h1, h2), 1);
    EXPECT_EQ(bindings->string_equal(h1, h3), 0);
    EXPECT_EQ(bindings->string_equal(h2, h3), 0);
}

TEST_F(BindingsTest, StringEqualEmpty) {
    const uint32_t h1 = bindings->string_create("");
    const uint32_t h2 = bindings->string_create("");
    const uint32_t h3 = bindings->string_create("x");

    EXPECT_EQ(bindings->string_equal(h1, h2), 1);
    EXPECT_EQ(bindings->string_equal(h1, h3), 0);
}

TEST_F(BindingsTest, StringEqualInvalidHandle) {
    const uint32_t h1 = bindings->string_create("hello");
    EXPECT_EQ(bindings->string_equal(h1, 999), 0);
    EXPECT_EQ(bindings->string_equal(999, h1), 0);
}

TEST_F(BindingsTest, StringGetInvalidHandle) {
    EXPECT_EQ(bindings->string_get(999), "");
}

} // namespace arc::runtime::wasm
