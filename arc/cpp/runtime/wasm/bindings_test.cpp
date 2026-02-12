// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/wasm/bindings.h"

namespace arc::runtime::wasm {

class BindingsTest : public testing::Test {
protected:
    void SetUp() override {
        state::Config cfg{.ir = ir::IR{}, .channels = {}};
        state = std::make_shared<state::State>(cfg, errors::noop_handler);
        bindings = std::make_unique<Bindings>(state, nullptr, errors::noop_handler);
    }

    std::shared_ptr<state::State> state;
    std::unique_ptr<Bindings> bindings;
};

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

TEST_F(BindingsTest, SeriesElementRsubF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 2.0);
    bindings->series_set_element_f64(h1, 2, 3.0);

    // 10.0 - [1, 2, 3] = [9, 8, 7]
    const uint32_t h2 = bindings->series_element_rsub_f64(10.0, h1);
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
    const uint32_t h2 = bindings->series_element_rdiv_f64(10.0, h1);
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
    const uint32_t h2 = bindings->series_element_rsub_i32(100, h1);
    EXPECT_EQ(bindings->series_index_i32(h2, 0), 95);
    EXPECT_EQ(bindings->series_index_i32(h2, 1), 90);
    EXPECT_EQ(bindings->series_index_i32(h2, 2), 85);
}

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
    bindings->series_set_element_u32(h2, 1, 7);

    // Different lengths should panic
    EXPECT_THROW(bindings->series_series_mod_u32(h1, h2), std::runtime_error);
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

TEST_F(BindingsTest, SeriesSeriesAddF64DifferentLengthsLHSLonger) {
    const uint32_t h1 = bindings->series_create_empty_f64(5);
    for (uint32_t i = 0; i < 5; i++)
        bindings->series_set_element_f64(h1, i, i + 1);

    const uint32_t h2 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h2, 0, 10.0);
    bindings->series_set_element_f64(h2, 1, 20.0);
    // Different lengths should panic
    EXPECT_THROW(bindings->series_series_add_f64(h1, h2), std::runtime_error);
}

TEST_F(BindingsTest, SeriesSeriesAddF64DifferentLengthsRHSLonger) {
    const uint32_t h1 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 2.0);

    const uint32_t h2 = bindings->series_create_empty_f64(4);
    for (uint32_t i = 0; i < 4; i++)
        bindings->series_set_element_f64(h2, i, (i + 1) * 10);
    EXPECT_THROW(bindings->series_series_add_f64(h1, h2), std::runtime_error);
}

TEST_F(BindingsTest, SeriesSeriesSubI32DifferentLengths) {
    const uint32_t h1 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h1, 0, 100);
    bindings->series_set_element_i32(h1, 1, 200);
    bindings->series_set_element_i32(h1, 2, 300);
    const uint32_t h2 = bindings->series_create_empty_i32(1);
    bindings->series_set_element_i32(h2, 0, 10);
    EXPECT_THROW(bindings->series_series_sub_i32(h1, h2), std::runtime_error);
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
    EXPECT_THROW(bindings->series_series_div_f64(h1, h2), std::runtime_error);
}

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

TEST_F(BindingsTest, SeriesCompareGtDifferentLengths) {
    // lhs = [1, 2, 3, 4, 5]
    const uint32_t h1 = bindings->series_create_empty_f64(5);
    for (uint32_t i = 0; i < 5; i++)
        bindings->series_set_element_f64(h1, i, i + 1);

    // rhs = [2, 1]
    const uint32_t h2 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h2, 0, 2.0);
    bindings->series_set_element_f64(h2, 1, 1.0);

    // Different lengths should panic
    EXPECT_THROW(bindings->series_compare_gt_f64(h1, h2), std::runtime_error);
}

TEST_F(BindingsTest, SeriesLenInvalidHandle) {
    EXPECT_EQ(bindings->series_len(999), 0);
}

TEST_F(BindingsTest, SeriesLenValidHandle) {
    const uint32_t h = bindings->series_create_empty_i32(7);
    EXPECT_EQ(bindings->series_len(h), 7);
}

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

TEST_F(BindingsTest, StateLoadStoreSeriesF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 2.0);
    bindings->series_set_element_f64(h1, 2, 3.0);

    // First load returns init_handle since nothing stored yet
    const uint32_t loaded1 = bindings->state_load_series_f64(1, h1);
    EXPECT_EQ(loaded1, h1);

    // Store the series
    bindings->state_store_series_f64(1, h1);

    // Create a different series as init value
    const uint32_t h2 = bindings->series_create_empty_f64(1);
    bindings->series_set_element_f64(h2, 0, 999.0);

    // Load should return stored series, not init
    const uint32_t loaded2 = bindings->state_load_series_f64(1, h2);
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

    bindings->state_store_series_i32(2, h1);

    const uint32_t dummy = bindings->series_create_empty_i32(1);
    const uint32_t loaded = bindings->state_load_series_i32(2, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 2);
    EXPECT_EQ(bindings->series_index_i32(loaded, 0), 42);
    EXPECT_EQ(bindings->series_index_i32(loaded, 1), -42);
}

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

TEST_F(BindingsTest, SeriesScalarCompareGtF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(4);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 5.0);
    bindings->series_set_element_f64(h1, 2, 3.0);
    bindings->series_set_element_f64(h1, 3, 8.0);

    const uint32_t h2 = bindings->series_compare_gt_scalar_f64(h1, 4.0);
    EXPECT_NE(h2, 0);
    EXPECT_EQ(bindings->series_len(h2), 4);
    // 1 > 4? false, 5 > 4? true, 3 > 4? false, 8 > 4? true
    EXPECT_EQ(bindings->series_index_u8(h2, 0), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 2), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 3), 1);
}

TEST_F(BindingsTest, SeriesScalarCompareLtI32) {
    const uint32_t h1 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h1, 0, 1);
    bindings->series_set_element_i32(h1, 1, 5);
    bindings->series_set_element_i32(h1, 2, 3);

    const uint32_t h2 = bindings->series_compare_lt_scalar_i32(h1, 3);
    // 1 < 3? true, 5 < 3? false, 3 < 3? false
    EXPECT_EQ(bindings->series_index_u8(h2, 0), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 1), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 2), 0);
}

TEST_F(BindingsTest, SeriesScalarCompareGeF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    bindings->series_set_element_f64(h1, 1, 3.0);
    bindings->series_set_element_f64(h1, 2, 5.0);

    const uint32_t h2 = bindings->series_compare_ge_scalar_f64(h1, 3.0);
    // 1 >= 3? false, 3 >= 3? true, 5 >= 3? true
    EXPECT_EQ(bindings->series_index_u8(h2, 0), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 2), 1);
}

TEST_F(BindingsTest, SeriesScalarCompareLeI64) {
    const uint32_t h1 = bindings->series_create_empty_i64(3);
    bindings->series_set_element_i64(h1, 0, 1);
    bindings->series_set_element_i64(h1, 1, 3);
    bindings->series_set_element_i64(h1, 2, 5);

    const uint32_t h2 = bindings->series_compare_le_scalar_i64(h1, 3);
    // 1 <= 3? true, 3 <= 3? true, 5 <= 3? false
    EXPECT_EQ(bindings->series_index_u8(h2, 0), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 2), 0);
}

TEST_F(BindingsTest, SeriesScalarCompareEqU32) {
    const uint32_t h1 = bindings->series_create_empty_u32(4);
    bindings->series_set_element_u32(h1, 0, 1);
    bindings->series_set_element_u32(h1, 1, 3);
    bindings->series_set_element_u32(h1, 2, 3);
    bindings->series_set_element_u32(h1, 3, 5);

    const uint32_t h2 = bindings->series_compare_eq_scalar_u32(h1, 3);
    // 1 == 3? false, 3 == 3? true, 3 == 3? true, 5 == 3? false
    EXPECT_EQ(bindings->series_index_u8(h2, 0), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 2), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 3), 0);
}

TEST_F(BindingsTest, SeriesScalarCompareNeF32) {
    const uint32_t h1 = bindings->series_create_empty_f32(3);
    bindings->series_set_element_f32(h1, 0, 1.0f);
    bindings->series_set_element_f32(h1, 1, 3.0f);
    bindings->series_set_element_f32(h1, 2, 5.0f);

    const uint32_t h2 = bindings->series_compare_ne_scalar_f32(h1, 3.0f);
    // 1 != 3? true, 3 != 3? false, 5 != 3? true
    EXPECT_EQ(bindings->series_index_u8(h2, 0), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 1), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 2), 1);
}

TEST_F(BindingsTest, SeriesScalarCompareInvalidHandle) {
    EXPECT_EQ(bindings->series_compare_gt_scalar_f64(999, 1.0), 0);
    EXPECT_EQ(bindings->series_compare_lt_scalar_i32(999, 1), 0);
}

TEST_F(BindingsTest, SeriesScalarCompareAllTypes) {
    // u8
    {
        const uint32_t h = bindings->series_create_empty_u8(2);
        bindings->series_set_element_u8(h, 0, 5);
        bindings->series_set_element_u8(h, 1, 15);
        const uint32_t r = bindings->series_compare_gt_scalar_u8(h, 10);
        EXPECT_EQ(bindings->series_index_u8(r, 0), 0); // 5 > 10? false
        EXPECT_EQ(bindings->series_index_u8(r, 1), 1); // 15 > 10? true
    }
    // u16
    {
        const uint32_t h = bindings->series_create_empty_u16(2);
        bindings->series_set_element_u16(h, 0, 100);
        bindings->series_set_element_u16(h, 1, 200);
        const uint32_t r = bindings->series_compare_lt_scalar_u16(h, 150);
        EXPECT_EQ(bindings->series_index_u8(r, 0), 1); // 100 < 150? true
        EXPECT_EQ(bindings->series_index_u8(r, 1), 0); // 200 < 150? false
    }
    // u64
    {
        const uint32_t h = bindings->series_create_empty_u64(2);
        bindings->series_set_element_u64(h, 0, 1000);
        bindings->series_set_element_u64(h, 1, 1000);
        const uint32_t r = bindings->series_compare_eq_scalar_u64(h, 1000);
        EXPECT_EQ(bindings->series_index_u8(r, 0), 1);
        EXPECT_EQ(bindings->series_index_u8(r, 1), 1);
    }
    // i8
    {
        const uint32_t h = bindings->series_create_empty_i8(2);
        bindings->series_set_element_i8(h, 0, -5);
        bindings->series_set_element_i8(h, 1, 5);
        const uint32_t r = bindings->series_compare_ge_scalar_i8(h, 0);
        EXPECT_EQ(bindings->series_index_u8(r, 0), 0); // -5 >= 0? false
        EXPECT_EQ(bindings->series_index_u8(r, 1), 1); // 5 >= 0? true
    }
    // i16
    {
        const uint32_t h = bindings->series_create_empty_i16(2);
        bindings->series_set_element_i16(h, 0, -100);
        bindings->series_set_element_i16(h, 1, 100);
        const uint32_t r = bindings->series_compare_le_scalar_i16(h, 0);
        EXPECT_EQ(bindings->series_index_u8(r, 0), 1); // -100 <= 0? true
        EXPECT_EQ(bindings->series_index_u8(r, 1), 0); // 100 <= 0? false
    }
}

TEST_F(BindingsTest, SeriesNegateI8) {
    const uint32_t h1 = bindings->series_create_empty_i8(3);
    bindings->series_set_element_i8(h1, 0, 5);
    bindings->series_set_element_i8(h1, 1, -3);
    bindings->series_set_element_i8(h1, 2, 0);

    const uint32_t h2 = bindings->series_negate_i8(h1);
    EXPECT_NE(h2, 0);
    EXPECT_NE(h2, h1);
    EXPECT_EQ(bindings->series_index_i8(h2, 0), -5);
    EXPECT_EQ(bindings->series_index_i8(h2, 1), 3);
    EXPECT_EQ(bindings->series_index_i8(h2, 2), 0);

    // Original unchanged
    EXPECT_EQ(bindings->series_index_i8(h1, 0), 5);
}

TEST_F(BindingsTest, SeriesNegateI16) {
    const uint32_t h1 = bindings->series_create_empty_i16(3);
    bindings->series_set_element_i16(h1, 0, 1000);
    bindings->series_set_element_i16(h1, 1, -500);
    bindings->series_set_element_i16(h1, 2, 0);

    const uint32_t h2 = bindings->series_negate_i16(h1);
    EXPECT_NE(h2, 0);
    EXPECT_EQ(bindings->series_index_i16(h2, 0), -1000);
    EXPECT_EQ(bindings->series_index_i16(h2, 1), 500);
    EXPECT_EQ(bindings->series_index_i16(h2, 2), 0);
}

TEST_F(BindingsTest, SeriesNegateI32) {
    const uint32_t h1 = bindings->series_create_empty_i32(4);
    bindings->series_set_element_i32(h1, 0, 100000);
    bindings->series_set_element_i32(h1, 1, -50000);
    bindings->series_set_element_i32(h1, 2, 0);
    bindings->series_set_element_i32(h1, 3, 2147483647); // INT32_MAX

    const uint32_t h2 = bindings->series_negate_i32(h1);
    EXPECT_NE(h2, 0);
    EXPECT_EQ(bindings->series_index_i32(h2, 0), -100000);
    EXPECT_EQ(bindings->series_index_i32(h2, 1), 50000);
    EXPECT_EQ(bindings->series_index_i32(h2, 2), 0);
    EXPECT_EQ(bindings->series_index_i32(h2, 3), -2147483647);
}

TEST_F(BindingsTest, SeriesNegateI64) {
    const uint32_t h1 = bindings->series_create_empty_i64(3);
    bindings->series_set_element_i64(h1, 0, 10000000000LL);
    bindings->series_set_element_i64(h1, 1, -5000000000LL);
    bindings->series_set_element_i64(h1, 2, 0);

    const uint32_t h2 = bindings->series_negate_i64(h1);
    EXPECT_NE(h2, 0);
    EXPECT_EQ(bindings->series_index_i64(h2, 0), -10000000000LL);
    EXPECT_EQ(bindings->series_index_i64(h2, 1), 5000000000LL);
    EXPECT_EQ(bindings->series_index_i64(h2, 2), 0);
}

TEST_F(BindingsTest, SeriesNegateF32) {
    const uint32_t h1 = bindings->series_create_empty_f32(4);
    bindings->series_set_element_f32(h1, 0, 3.14159f);
    bindings->series_set_element_f32(h1, 1, -2.71828f);
    bindings->series_set_element_f32(h1, 2, 0.0f);
    bindings->series_set_element_f32(h1, 3, 1.0f);

    const uint32_t h2 = bindings->series_negate_f32(h1);
    EXPECT_NE(h2, 0);
    EXPECT_FLOAT_EQ(bindings->series_index_f32(h2, 0), -3.14159f);
    EXPECT_FLOAT_EQ(bindings->series_index_f32(h2, 1), 2.71828f);
    EXPECT_FLOAT_EQ(bindings->series_index_f32(h2, 2), 0.0f);
    EXPECT_FLOAT_EQ(bindings->series_index_f32(h2, 3), -1.0f);
}

TEST_F(BindingsTest, SeriesNegateF64) {
    const uint32_t h1 = bindings->series_create_empty_f64(4);
    bindings->series_set_element_f64(h1, 0, 3.14159265358979);
    bindings->series_set_element_f64(h1, 1, -2.71828182845905);
    bindings->series_set_element_f64(h1, 2, 0.0);
    bindings->series_set_element_f64(h1, 3, 1.0);

    const uint32_t h2 = bindings->series_negate_f64(h1);
    EXPECT_NE(h2, 0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 0), -3.14159265358979);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 1), 2.71828182845905);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 2), 0.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(h2, 3), -1.0);
}

TEST_F(BindingsTest, SeriesNegateInvalidHandle) {
    EXPECT_EQ(bindings->series_negate_i32(999), 0);
    EXPECT_EQ(bindings->series_negate_f64(999), 0);
}

TEST_F(BindingsTest, SeriesNegateEmpty) {
    const uint32_t h1 = bindings->series_create_empty_f64(0);
    const uint32_t h2 = bindings->series_negate_f64(h1);
    EXPECT_NE(h2, 0);
    EXPECT_EQ(bindings->series_len(h2), 0);
}

TEST_F(BindingsTest, SeriesNegateDoubleNegation) {
    const uint32_t h1 = bindings->series_create_empty_i32(3);
    bindings->series_set_element_i32(h1, 0, 10);
    bindings->series_set_element_i32(h1, 1, -20);
    bindings->series_set_element_i32(h1, 2, 30);

    const uint32_t h2 = bindings->series_negate_i32(h1);
    const uint32_t h3 = bindings->series_negate_i32(h2);

    // Double negation should return original values
    EXPECT_EQ(bindings->series_index_i32(h3, 0), 10);
    EXPECT_EQ(bindings->series_index_i32(h3, 1), -20);
    EXPECT_EQ(bindings->series_index_i32(h3, 2), 30);
}

TEST_F(BindingsTest, SeriesNotU8) {
    const uint32_t h1 = bindings->series_create_empty_u8(4);
    bindings->series_set_element_u8(h1, 0, 0x00);
    bindings->series_set_element_u8(h1, 1, 0xFF);
    bindings->series_set_element_u8(h1, 2, 0x0F);
    bindings->series_set_element_u8(h1, 3, 0xF0);

    const uint32_t h2 = bindings->series_not_u8(h1);
    EXPECT_NE(h2, 0);
    EXPECT_NE(h2, h1);
    // Logical NOT: !0 = 1, !non-zero = 0
    EXPECT_EQ(bindings->series_index_u8(h2, 0), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 1), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 2), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 3), 0);

    // Original unchanged
    EXPECT_EQ(bindings->series_index_u8(h1, 0), 0x00);
}

TEST_F(BindingsTest, SeriesNotU8BooleanValues) {
    // Test with boolean-like values (0 and 1)
    const uint32_t h1 = bindings->series_create_empty_u8(4);
    bindings->series_set_element_u8(h1, 0, 0); // false
    bindings->series_set_element_u8(h1, 1, 1); // true
    bindings->series_set_element_u8(h1, 2, 1); // true
    bindings->series_set_element_u8(h1, 3, 0); // false

    const uint32_t h2 = bindings->series_not_u8(h1);
    // Logical NOT: !0 = 1, !1 = 0
    EXPECT_EQ(bindings->series_index_u8(h2, 0), 1);
    EXPECT_EQ(bindings->series_index_u8(h2, 1), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 2), 0);
    EXPECT_EQ(bindings->series_index_u8(h2, 3), 1);
}

TEST_F(BindingsTest, SeriesNotU8InvalidHandle) {
    EXPECT_EQ(bindings->series_not_u8(999), 0);
}

TEST_F(BindingsTest, SeriesNotU8Empty) {
    const uint32_t h1 = bindings->series_create_empty_u8(0);
    const uint32_t h2 = bindings->series_not_u8(h1);
    EXPECT_NE(h2, 0);
    EXPECT_EQ(bindings->series_len(h2), 0);
}

TEST_F(BindingsTest, SeriesNotU8DoubleNot) {
    // With logical NOT, double negation normalizes to 0/1
    const uint32_t h1 = bindings->series_create_empty_u8(4);
    bindings->series_set_element_u8(h1, 0, 0); // false
    bindings->series_set_element_u8(h1, 1, 1); // true
    bindings->series_set_element_u8(h1, 2, 0); // false
    bindings->series_set_element_u8(h1, 3, 1); // true

    const uint32_t h2 = bindings->series_not_u8(h1); // 1, 0, 1, 0
    const uint32_t h3 = bindings->series_not_u8(h2); // 0, 1, 0, 1

    // Double logical NOT returns original boolean values
    EXPECT_EQ(bindings->series_index_u8(h3, 0), 0);
    EXPECT_EQ(bindings->series_index_u8(h3, 1), 1);
    EXPECT_EQ(bindings->series_index_u8(h3, 2), 0);
    EXPECT_EQ(bindings->series_index_u8(h3, 3), 1);
}

TEST_F(BindingsTest, StringLenInvalidHandle) {
    EXPECT_EQ(bindings->string_len(999), 0u);
}

TEST_F(BindingsTest, StringEqualInvalidHandles) {
    EXPECT_EQ(bindings->string_equal(999, 998), 0u);
}

TEST_F(BindingsTest, StringConcatInvalidHandles) {
    EXPECT_EQ(bindings->string_concat(999, 998), 0u);
}

TEST_F(BindingsTest, StringConcatOneInvalidHandle) {
    // Can't easily test with valid handles without WASM memory setup
    // but we can verify that one invalid handle returns 0
    EXPECT_EQ(bindings->string_concat(999, 0), 0u);
    EXPECT_EQ(bindings->string_concat(0, 999), 0u);
}

TEST_F(BindingsTest, ClearTransientHandlesClearsSeriesHandles) {
    const uint32_t h1 = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(h1, 0, 1.0);
    const uint32_t h2 = bindings->series_create_empty_i32(2);
    bindings->series_set_element_i32(h2, 0, 42);

    EXPECT_EQ(bindings->series_len(h1), 3);
    EXPECT_EQ(bindings->series_len(h2), 2);

    state->flush();

    EXPECT_EQ(bindings->series_len(h1), 0);
    EXPECT_EQ(bindings->series_len(h2), 0);
}

TEST_F(BindingsTest, ClearTransientHandlesClearsStringHandles) {
    const uint32_t h1 = bindings->string_create("hello");
    const uint32_t h2 = bindings->string_create("world");

    EXPECT_EQ(bindings->string_get(h1), "hello");
    EXPECT_EQ(bindings->string_get(h2), "world");

    state->flush();

    EXPECT_EQ(bindings->string_get(h1), "");
    EXPECT_EQ(bindings->string_get(h2), "");
}

TEST_F(BindingsTest, ClearTransientHandlesResetsCounters) {
    bindings->series_create_empty_f64(1);
    bindings->series_create_empty_f64(1);
    bindings->series_create_empty_f64(1);
    bindings->string_create("a");
    bindings->string_create("b");

    state->flush();

    const uint32_t new_series = bindings->series_create_empty_f64(1);
    const uint32_t new_string = bindings->string_create("new");

    EXPECT_EQ(new_series, 1);
    EXPECT_EQ(new_string, 1);
}

TEST_F(BindingsTest, ClearTransientHandlesPreservesStatefulSeries) {
    const uint32_t h1 = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(h1, 0, 100.0);
    bindings->series_set_element_f64(h1, 1, 200.0);
    bindings->state_store_series_f64(1, h1);

    state->flush();

    EXPECT_EQ(bindings->series_len(h1), 0);

    const uint32_t dummy = bindings->series_create_empty_f64(1);
    const uint32_t loaded = bindings->state_load_series_f64(1, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 2);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded, 0), 100.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded, 1), 200.0);
}

TEST_F(BindingsTest, ClearTransientHandlesPreservesStatefulStrings) {
    const uint32_t h1 = bindings->string_create("persistent");
    bindings->state_store_str(2, h1);

    state->flush();

    EXPECT_EQ(bindings->string_get(h1), "");

    const uint32_t dummy = bindings->string_create("dummy");
    const uint32_t loaded = bindings->state_load_str(2, dummy);

    EXPECT_EQ(bindings->string_get(loaded), "persistent");
}

TEST_F(BindingsTest, ClearTransientHandlesPreservesStatefulPrimitives) {
    bindings->state_store_f64(1, 3.14159);
    bindings->state_store_i32(2, -42);
    bindings->state_store_u64(3, 9999999999ULL);

    state->flush();

    EXPECT_DOUBLE_EQ(bindings->state_load_f64(1, 0.0), 3.14159);
    EXPECT_EQ(bindings->state_load_i32(2, 0), -42);
    EXPECT_EQ(bindings->state_load_u64(3, 0), 9999999999ULL);
}

TEST_F(BindingsTest, StateLoadStoreU8) {
    EXPECT_EQ(bindings->state_load_u8(1, 100), 100);
    bindings->state_store_u8(1, 255);
    EXPECT_EQ(bindings->state_load_u8(1, 0), 255);
}

TEST_F(BindingsTest, StateLoadStoreU16) {
    EXPECT_EQ(bindings->state_load_u16(1, 1000), 1000);
    bindings->state_store_u16(1, 65535);
    EXPECT_EQ(bindings->state_load_u16(1, 0), 65535);
}

TEST_F(BindingsTest, StateLoadStoreU32) {
    EXPECT_EQ(bindings->state_load_u32(1, 50000), 50000);
    bindings->state_store_u32(1, 4294967295U);
    EXPECT_EQ(bindings->state_load_u32(1, 0), 4294967295U);
}

TEST_F(BindingsTest, StateLoadStoreI8) {
    EXPECT_EQ(bindings->state_load_i8(1, 50), 50);
    bindings->state_store_i8(1, -128);
    EXPECT_EQ(bindings->state_load_i8(1, 0), -128);
}

TEST_F(BindingsTest, StateLoadStoreI16) {
    EXPECT_EQ(bindings->state_load_i16(1, 1000), 1000);
    bindings->state_store_i16(1, -32768);
    EXPECT_EQ(bindings->state_load_i16(1, 0), -32768);
}

TEST_F(BindingsTest, StateLoadStoreI64) {
    EXPECT_EQ(bindings->state_load_i64(1, 1000000), 1000000);
    bindings->state_store_i64(1, -9223372036854775807LL);
    EXPECT_EQ(bindings->state_load_i64(1, 0), -9223372036854775807LL);
}

TEST_F(BindingsTest, StateLoadStoreF32) {
    EXPECT_FLOAT_EQ(bindings->state_load_f32(1, 1.5f), 1.5f);
    bindings->state_store_f32(1, 3.14159f);
    EXPECT_FLOAT_EQ(bindings->state_load_f32(1, 0.0f), 3.14159f);
}

TEST_F(BindingsTest, StateLoadStoreSeriesU8) {
    const uint32_t h1 = bindings->series_create_empty_u8(3);
    bindings->series_set_element_u8(h1, 0, 100);
    bindings->series_set_element_u8(h1, 1, 200);
    bindings->series_set_element_u8(h1, 2, 255);
    bindings->state_store_series_u8(1, h1);

    const uint32_t dummy = bindings->series_create_empty_u8(1);
    const uint32_t loaded = bindings->state_load_series_u8(1, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 3);
    EXPECT_EQ(bindings->series_index_u8(loaded, 0), 100);
    EXPECT_EQ(bindings->series_index_u8(loaded, 1), 200);
    EXPECT_EQ(bindings->series_index_u8(loaded, 2), 255);
}

TEST_F(BindingsTest, StateLoadStoreSeriesU16) {
    const uint32_t h1 = bindings->series_create_empty_u16(3);
    bindings->series_set_element_u16(h1, 0, 1000);
    bindings->series_set_element_u16(h1, 1, 30000);
    bindings->series_set_element_u16(h1, 2, 65535);
    bindings->state_store_series_u16(1, h1);

    const uint32_t dummy = bindings->series_create_empty_u16(1);
    const uint32_t loaded = bindings->state_load_series_u16(1, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 3);
    EXPECT_EQ(bindings->series_index_u16(loaded, 0), 1000);
    EXPECT_EQ(bindings->series_index_u16(loaded, 1), 30000);
    EXPECT_EQ(bindings->series_index_u16(loaded, 2), 65535);
}

TEST_F(BindingsTest, StateLoadStoreSeriesU32) {
    const uint32_t h1 = bindings->series_create_empty_u32(3);
    bindings->series_set_element_u32(h1, 0, 100000);
    bindings->series_set_element_u32(h1, 1, 2000000);
    bindings->series_set_element_u32(h1, 2, 4294967295U);
    bindings->state_store_series_u32(1, h1);

    const uint32_t dummy = bindings->series_create_empty_u32(1);
    const uint32_t loaded = bindings->state_load_series_u32(1, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 3);
    EXPECT_EQ(bindings->series_index_u32(loaded, 0), 100000);
    EXPECT_EQ(bindings->series_index_u32(loaded, 1), 2000000);
    EXPECT_EQ(bindings->series_index_u32(loaded, 2), 4294967295U);
}

TEST_F(BindingsTest, StateLoadStoreSeriesU64) {
    const uint32_t h1 = bindings->series_create_empty_u64(3);
    bindings->series_set_element_u64(h1, 0, 1000000000ULL);
    bindings->series_set_element_u64(h1, 1, 5000000000ULL);
    bindings->series_set_element_u64(h1, 2, 18446744073709551615ULL);
    bindings->state_store_series_u64(1, h1);

    const uint32_t dummy = bindings->series_create_empty_u64(1);
    const uint32_t loaded = bindings->state_load_series_u64(1, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 3);
    EXPECT_EQ(bindings->series_index_u64(loaded, 0), 1000000000ULL);
    EXPECT_EQ(bindings->series_index_u64(loaded, 1), 5000000000ULL);
    EXPECT_EQ(bindings->series_index_u64(loaded, 2), 18446744073709551615ULL);
}

TEST_F(BindingsTest, StateLoadStoreSeriesI8) {
    const uint32_t h1 = bindings->series_create_empty_i8(3);
    bindings->series_set_element_i8(h1, 0, -128);
    bindings->series_set_element_i8(h1, 1, 0);
    bindings->series_set_element_i8(h1, 2, 127);
    bindings->state_store_series_i8(1, h1);

    const uint32_t dummy = bindings->series_create_empty_i8(1);
    const uint32_t loaded = bindings->state_load_series_i8(1, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 3);
    EXPECT_EQ(bindings->series_index_i8(loaded, 0), -128);
    EXPECT_EQ(bindings->series_index_i8(loaded, 1), 0);
    EXPECT_EQ(bindings->series_index_i8(loaded, 2), 127);
}

TEST_F(BindingsTest, StateLoadStoreSeriesI16) {
    const uint32_t h1 = bindings->series_create_empty_i16(3);
    bindings->series_set_element_i16(h1, 0, -32768);
    bindings->series_set_element_i16(h1, 1, 0);
    bindings->series_set_element_i16(h1, 2, 32767);
    bindings->state_store_series_i16(1, h1);

    const uint32_t dummy = bindings->series_create_empty_i16(1);
    const uint32_t loaded = bindings->state_load_series_i16(1, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 3);
    EXPECT_EQ(bindings->series_index_i16(loaded, 0), -32768);
    EXPECT_EQ(bindings->series_index_i16(loaded, 1), 0);
    EXPECT_EQ(bindings->series_index_i16(loaded, 2), 32767);
}

TEST_F(BindingsTest, StateLoadStoreSeriesI64) {
    const uint32_t h1 = bindings->series_create_empty_i64(3);
    bindings->series_set_element_i64(h1, 0, -9223372036854775807LL);
    bindings->series_set_element_i64(h1, 1, 0);
    bindings->series_set_element_i64(h1, 2, 9223372036854775807LL);
    bindings->state_store_series_i64(1, h1);

    const uint32_t dummy = bindings->series_create_empty_i64(1);
    const uint32_t loaded = bindings->state_load_series_i64(1, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 3);
    EXPECT_EQ(bindings->series_index_i64(loaded, 0), -9223372036854775807LL);
    EXPECT_EQ(bindings->series_index_i64(loaded, 1), 0);
    EXPECT_EQ(bindings->series_index_i64(loaded, 2), 9223372036854775807LL);
}

TEST_F(BindingsTest, StateLoadStoreSeriesF32) {
    const uint32_t h1 = bindings->series_create_empty_f32(3);
    bindings->series_set_element_f32(h1, 0, -3.14159f);
    bindings->series_set_element_f32(h1, 1, 0.0f);
    bindings->series_set_element_f32(h1, 2, 2.71828f);
    bindings->state_store_series_f32(1, h1);

    const uint32_t dummy = bindings->series_create_empty_f32(1);
    const uint32_t loaded = bindings->state_load_series_f32(1, dummy);

    EXPECT_EQ(bindings->series_len(loaded), 3);
    EXPECT_FLOAT_EQ(bindings->series_index_f32(loaded, 0), -3.14159f);
    EXPECT_FLOAT_EQ(bindings->series_index_f32(loaded, 1), 0.0f);
    EXPECT_FLOAT_EQ(bindings->series_index_f32(loaded, 2), 2.71828f);
}

TEST_F(BindingsTest, NodeKeyIsolatesPrimitiveState) {
    state->set_current_node_key("node_a");
    bindings->state_store_f64(1, 100.0);

    state->set_current_node_key("node_b");
    bindings->state_store_f64(1, 200.0);

    state->set_current_node_key("node_a");
    EXPECT_DOUBLE_EQ(bindings->state_load_f64(1, 0.0), 100.0);

    state->set_current_node_key("node_b");
    EXPECT_DOUBLE_EQ(bindings->state_load_f64(1, 0.0), 200.0);
}

TEST_F(BindingsTest, NodeKeyIsolatesSeriesState) {
    state->set_current_node_key("node_a");
    const uint32_t ha = bindings->series_create_empty_f64(2);
    bindings->series_set_element_f64(ha, 0, 1.0);
    bindings->series_set_element_f64(ha, 1, 2.0);
    bindings->state_store_series_f64(1, ha);

    state->set_current_node_key("node_b");
    const uint32_t hb = bindings->series_create_empty_f64(3);
    bindings->series_set_element_f64(hb, 0, 10.0);
    bindings->series_set_element_f64(hb, 1, 20.0);
    bindings->series_set_element_f64(hb, 2, 30.0);
    bindings->state_store_series_f64(1, hb);

    state->set_current_node_key("node_a");
    const uint32_t dummy = bindings->series_create_empty_f64(1);
    const uint32_t loaded_a = bindings->state_load_series_f64(1, dummy);
    EXPECT_EQ(bindings->series_len(loaded_a), 2);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded_a, 0), 1.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded_a, 1), 2.0);

    state->set_current_node_key("node_b");
    const uint32_t loaded_b = bindings->state_load_series_f64(1, dummy);
    EXPECT_EQ(bindings->series_len(loaded_b), 3);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded_b, 0), 10.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded_b, 1), 20.0);
    EXPECT_DOUBLE_EQ(bindings->series_index_f64(loaded_b, 2), 30.0);
}

TEST_F(BindingsTest, NodeKeyIsolatesStringState) {
    state->set_current_node_key("node_a");
    const uint32_t str_a = bindings->string_create("hello from node_a");
    bindings->state_store_str(1, str_a);

    state->set_current_node_key("node_b");
    const uint32_t str_b = bindings->string_create("hello from node_b");
    bindings->state_store_str(1, str_b);

    state->set_current_node_key("node_a");
    const uint32_t dummy = bindings->string_create("dummy");
    const uint32_t loaded_a = bindings->state_load_str(1, dummy);
    EXPECT_EQ(bindings->string_get(loaded_a), "hello from node_a");

    state->set_current_node_key("node_b");
    const uint32_t loaded_b = bindings->state_load_str(1, dummy);
    EXPECT_EQ(bindings->string_get(loaded_b), "hello from node_b");
}

TEST_F(BindingsTest, MultipleClearCycles) {
    for (int cycle = 0; cycle < 3; cycle++) {
        const uint32_t h1 = bindings->series_create_empty_f64(2);
        bindings->series_set_element_f64(h1, 0, static_cast<double>(cycle));
        const uint32_t h2 = bindings->string_create("cycle" + std::to_string(cycle));

        EXPECT_EQ(bindings->series_len(h1), 2);
        EXPECT_NE(bindings->string_get(h2), "");

        state->flush();
    }

    const uint32_t final_series = bindings->series_create_empty_f64(1);
    const uint32_t final_string = bindings->string_create("final");

    EXPECT_EQ(final_series, 1);
    EXPECT_EQ(final_string, 1);
}

// Test fixture with State for channel operations
class BindingsChannelTest : public ::testing::Test {
protected:
    void SetUp() override {
        state::Config cfg{.ir = ir::IR{}, .channels = {}};
        state = std::make_shared<state::State>(cfg, errors::noop_handler);
        bindings = std::make_unique<Bindings>(state, nullptr, errors::noop_handler);
    }

    std::shared_ptr<state::State> state;
    std::unique_ptr<Bindings> bindings;
};

TEST_F(BindingsChannelTest, ChannelReadNoDataReturnsDefault) {
    // No data ingested, should return default values
    EXPECT_EQ(bindings->channel_read_f64(1), 0.0);
    EXPECT_EQ(bindings->channel_read_f32(1), 0.0f);
    EXPECT_EQ(bindings->channel_read_i32(1), 0);
    EXPECT_EQ(bindings->channel_read_u64(1), 0);
    EXPECT_EQ(bindings->channel_read_u8(1), 0);
}

TEST_F(BindingsChannelTest, ChannelReadF64WithData) {
    x::telem::Frame frame(1);
    auto series = x::telem::Series(x::telem::FLOAT64_T, 3);
    series.write(1.5);
    series.write(2.5);
    series.write(3.5);
    frame.emplace(1, std::move(series));

    state->ingest(frame);

    EXPECT_DOUBLE_EQ(bindings->channel_read_f64(1), 3.5);
}

TEST_F(BindingsChannelTest, ChannelReadI32WithData) {
    x::telem::Frame frame(1);
    auto series = x::telem::Series(x::telem::INT32_T, 2);
    series.write(42);
    series.write(-100);
    frame.emplace(2, std::move(series));

    state->ingest(frame);

    EXPECT_EQ(bindings->channel_read_i32(2), -100);
}

TEST_F(BindingsChannelTest, ChannelReadU8WithData) {
    x::telem::Frame frame(1);
    auto series = x::telem::Series(x::telem::UINT8_T, 2);
    series.write(static_cast<uint8_t>(255));
    series.write(static_cast<uint8_t>(128));
    frame.emplace(3, std::move(series));

    state->ingest(frame);

    EXPECT_EQ(bindings->channel_read_u8(3), 128);
}

TEST_F(BindingsChannelTest, ChannelWriteF64) {
    bindings->channel_write_f64(10, 99.5);

    const auto writes = state->flush();
    EXPECT_EQ(writes.size(), 1);
    EXPECT_EQ(writes[0].first, 10);
    EXPECT_EQ(writes[0].second->size(), 1);
    EXPECT_DOUBLE_EQ(writes[0].second->at<double>(0), 99.5);
}

TEST_F(BindingsChannelTest, ChannelWriteI32) {
    bindings->channel_write_i32(20, -42);

    const auto writes = state->flush();
    EXPECT_EQ(writes.size(), 1);
    EXPECT_EQ(writes[0].first, 20);
    EXPECT_EQ(writes[0].second->at<int32_t>(0), -42);
}

TEST_F(BindingsChannelTest, ChannelWriteU64) {
    bindings->channel_write_u64(30, 18446744073709551615ULL);

    const auto writes = state->flush();
    EXPECT_EQ(writes.size(), 1);
    EXPECT_EQ(writes[0].first, 30);
    EXPECT_EQ(writes[0].second->at<uint64_t>(0), 18446744073709551615ULL);
}

TEST_F(BindingsChannelTest, ChannelReadDifferentChannelReturnsDefault) {
    // Ingest data for channel 1
    x::telem::Frame frame(1);
    auto series = x::telem::Series(x::telem::FLOAT64_T, 1);
    series.write(123.456);
    frame.emplace(1, std::move(series));
    state->ingest(frame);

    EXPECT_DOUBLE_EQ(bindings->channel_read_f64(1), 123.456);
    EXPECT_EQ(bindings->channel_read_f64(2), 0.0);
}

TEST_F(BindingsChannelTest, ChannelMultipleWrites) {
    bindings->channel_write_f64(1, 1.0);
    bindings->channel_write_f64(2, 2.0);
    bindings->channel_write_i32(3, 3);

    const auto writes = state->flush();
    EXPECT_EQ(writes.size(), 3);
}

TEST_F(BindingsChannelTest, ChannelReadStrNoData) {
    EXPECT_EQ(bindings->channel_read_str(1), 0);
}

TEST_F(BindingsChannelTest, ChannelWriteStr) {
    const uint32_t str_handle = bindings->string_create("hello world");

    bindings->channel_write_str(40, str_handle);

    const auto writes = state->flush();
    EXPECT_EQ(writes.size(), 1);
    EXPECT_EQ(writes[0].first, 40);
    EXPECT_EQ(writes[0].second->size(), 1);
}

TEST(BindingsNullStateTest, ChannelReadWithNullStateReturnsDefault) {
    const auto bindings = std::make_unique<Bindings>(
        nullptr,
        nullptr,
        arc::runtime::errors::noop_handler
    );

    EXPECT_EQ(bindings->channel_read_f64(1), 0.0);
    EXPECT_EQ(bindings->channel_read_i32(1), 0);
    EXPECT_EQ(bindings->channel_read_str(1), 0);
}

TEST(BindingsNullStateTest, ChannelWriteWithNullStateDoesNotCrash) {
    const auto bindings = std::make_unique<Bindings>(
        nullptr,
        nullptr,
        arc::runtime::errors::noop_handler
    );

    bindings->channel_write_f64(1, 123.0);
    bindings->channel_write_i32(2, 456);
    bindings->channel_write_str(3, 0);
}

/// @brief Test that panic calls error handler with WASM_PANIC error.
TEST(BindingsPanicTest, PanicCallsErrorHandler) {
    std::vector<x::errors::Error> reported_errors;
    auto error_handler = [&reported_errors](const x::errors::Error &err) {
        reported_errors.push_back(err);
    };

    auto bindings = std::make_unique<Bindings>(nullptr, nullptr, error_handler);

    bindings->panic(0, 0);

    ASSERT_EQ(reported_errors.size(), 1);
    ASSERT_MATCHES(reported_errors[0], arc::runtime::errors::WASM_PANIC);
}

/// @brief Test that multiple panics each call error handler.
TEST(BindingsPanicTest, MultiplePanicsCallErrorHandlerMultipleTimes) {
    std::vector<x::errors::Error> reported_errors;
    auto error_handler = [&reported_errors](const x::errors::Error &err) {
        reported_errors.push_back(err);
    };

    auto bindings = std::make_unique<Bindings>(nullptr, nullptr, error_handler);

    bindings->panic(0, 0);
    bindings->panic(0, 0);
    bindings->panic(0, 0);

    ASSERT_EQ(reported_errors.size(), 3);
    for (const auto &err: reported_errors)
        ASSERT_MATCHES(err, arc::runtime::errors::WASM_PANIC);
}

}
