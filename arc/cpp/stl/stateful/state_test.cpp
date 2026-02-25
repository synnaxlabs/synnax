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

#include "arc/cpp/stl/stateful/state.h"

namespace arc::stl::stateful {
TEST(Variables, LoadReturnsInitialValueOnFirstAccess) {
    Variables vars;
    vars.set_current_node_key("node1");
    EXPECT_EQ(vars.load_i32(0, 42), 42);
    EXPECT_EQ(vars.load_f64(1, 3.14), 3.14);
    EXPECT_EQ(vars.load_u8(2, 255), 255);
}

TEST(Variables, StoreAndLoad) {
    Variables vars;
    vars.set_current_node_key("node1");
    vars.load_i32(0, 0);
    vars.store_i32(0, 100);
    EXPECT_EQ(vars.load_i32(0, 0), 100);
}

TEST(Variables, LoadReturnsPreviouslyStoredValue) {
    Variables vars;
    vars.set_current_node_key("node1");
    vars.load_i64(0, 0);
    vars.store_i64(0, int64_t{999});
    EXPECT_EQ(vars.load_i64(0, int64_t{0}), int64_t{999});
}

TEST(Variables, NodeKeyIsolation) {
    Variables vars;
    vars.set_current_node_key("node_a");
    vars.load_i32(0, 0);
    vars.store_i32(0, 10);

    vars.set_current_node_key("node_b");
    EXPECT_EQ(vars.load_i32(0, 99), 99);

    vars.set_current_node_key("node_a");
    EXPECT_EQ(vars.load_i32(0, 0), 10);
}

TEST(Variables, MultipleVarIds) {
    Variables vars;
    vars.set_current_node_key("node1");
    vars.load_f32(0, 0.0f);
    vars.load_f32(1, 0.0f);
    vars.store_f32(0, 1.5f);
    vars.store_f32(1, 2.5f);
    EXPECT_FLOAT_EQ(vars.load_f32(0, 0.0f), 1.5f);
    EXPECT_FLOAT_EQ(vars.load_f32(1, 0.0f), 2.5f);
}

TEST(Variables, AllPrimitiveTypes) {
    Variables vars;
    vars.set_current_node_key("node1");
    vars.store_u8(0, 1);
    vars.store_u16(0, 2);
    vars.store_u32(0, 3);
    vars.store_u64(0, 4);
    vars.store_i8(0, -1);
    vars.store_i16(0, -2);
    vars.store_i32(0, -3);
    vars.store_i64(0, -4);
    vars.store_f32(0, 1.1f);
    vars.store_f64(0, 2.2);
    EXPECT_EQ(vars.load_u8(0, 0), 1);
    EXPECT_EQ(vars.load_u16(0, 0), 2);
    EXPECT_EQ(vars.load_u32(0, 0), 3);
    EXPECT_EQ(vars.load_u64(0, 0), 4);
    EXPECT_EQ(vars.load_i8(0, 0), -1);
    EXPECT_EQ(vars.load_i16(0, 0), -2);
    EXPECT_EQ(vars.load_i32(0, 0), -3);
    EXPECT_EQ(vars.load_i64(0, 0), -4);
    EXPECT_FLOAT_EQ(vars.load_f32(0, 0), 1.1f);
    EXPECT_DOUBLE_EQ(vars.load_f64(0, 0), 2.2);
}

TEST(Variables, Reset) {
    Variables vars;
    vars.set_current_node_key("node1");
    vars.load_i32(0, 0);
    vars.store_i32(0, 42);
    vars.reset();
    EXPECT_EQ(vars.load_i32(0, 7), 7);
}

TEST(Variables, LoadStrReturnsInitialOnFirstAccess) {
    Variables vars;
    str::State ss;
    vars.set_current_node_key("node1");
    const uint32_t init_handle = ss.create("hello");
    const uint32_t result = vars.load_str(0, init_handle, ss);
    EXPECT_EQ(ss.get(result), "hello");
}

TEST(Variables, StoreAndLoadStr) {
    Variables vars;
    str::State ss;
    vars.set_current_node_key("node1");
    const uint32_t init_handle = ss.create("initial");
    vars.load_str(0, init_handle, ss);
    const uint32_t new_handle = ss.create("updated");
    vars.store_str(0, new_handle, ss);
    const uint32_t result = vars.load_str(0, init_handle, ss);
    EXPECT_EQ(ss.get(result), "updated");
}

TEST(Variables, LoadSeriesReturnsInitialOnFirstAccess) {
    Variables vars;
    series::State ss;
    vars.set_current_node_key("node1");
    auto s = x::telem::Series(x::telem::INT32_T, 2);
    s.resize(2);
    s.set(0, int32_t{10});
    s.set(1, int32_t{20});
    const uint32_t init_handle = ss.store(std::move(s));
    const uint32_t result = vars.load_series(0, init_handle, ss);
    EXPECT_EQ(result, init_handle);
}

TEST(Variables, StoreAndLoadSeries) {
    Variables vars;
    series::State ss;
    vars.set_current_node_key("node1");
    auto init = x::telem::Series(x::telem::FLOAT32_T, 1);
    init.resize(1);
    init.set(0, 0.0f);
    const uint32_t init_handle = ss.store(std::move(init));
    vars.load_series(0, init_handle, ss);
    auto updated = x::telem::Series(x::telem::FLOAT32_T, 1);
    updated.resize(1);
    updated.set(0, 42.0f);
    const uint32_t new_handle = ss.store(std::move(updated));
    vars.store_series(0, new_handle, ss);
    const uint32_t result = vars.load_series(0, init_handle, ss);
    const auto *loaded = ss.get(result);
    ASSERT_NE(loaded, nullptr);
    EXPECT_FLOAT_EQ(loaded->at<float>(0), 42.0f);
}
}
