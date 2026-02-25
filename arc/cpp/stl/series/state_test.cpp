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

#include "arc/cpp/stl/series/state.h"

namespace arc::stl::series {
TEST(State, StoreReturnsIncrementingHandles) {
    State st;
    auto s1 = x::telem::Series(x::telem::FLOAT32_T, 1);
    auto s2 = x::telem::Series(x::telem::FLOAT32_T, 1);
    const uint32_t h1 = st.store(std::move(s1));
    const uint32_t h2 = st.store(std::move(s2));
    EXPECT_EQ(h1, 1);
    EXPECT_EQ(h2, 2);
}

TEST(State, GetReturnsStoredSeries) {
    State st;
    auto s = x::telem::Series(x::telem::INT32_T, 3);
    s.resize(3);
    s.set(0, int32_t{10});
    s.set(1, int32_t{20});
    s.set(2, int32_t{30});
    const uint32_t h = st.store(std::move(s));
    auto *retrieved = st.get(h);
    ASSERT_NE(retrieved, nullptr);
    EXPECT_EQ(retrieved->size(), 3);
    EXPECT_EQ(retrieved->at<int32_t>(0), 10);
    EXPECT_EQ(retrieved->at<int32_t>(1), 20);
    EXPECT_EQ(retrieved->at<int32_t>(2), 30);
}

TEST(State, GetReturnsNullptrForMissingHandle) {
    State st;
    EXPECT_EQ(st.get(999), nullptr);
}

TEST(State, ConstGetReturnsNullptrForMissingHandle) {
    const State st;
    EXPECT_EQ(st.get(42), nullptr);
}

TEST(State, ConstGetReturnsStoredSeries) {
    State st;
    auto s = x::telem::Series(x::telem::UINT8_T, 1);
    s.set(0, uint8_t{255});
    s.resize(1);
    const uint32_t h = st.store(std::move(s));
    const State &cst = st;
    const auto *retrieved = cst.get(h);
    ASSERT_NE(retrieved, nullptr);
    EXPECT_EQ(retrieved->at<uint8_t>(0), 255);
}

TEST(State, ClearRemovesAllHandles) {
    State st;
    const uint32_t h1 = st.store(x::telem::Series(x::telem::FLOAT64_T, 1));
    const uint32_t h2 = st.store(x::telem::Series(x::telem::FLOAT64_T, 1));
    st.clear();
    EXPECT_EQ(st.get(h1), nullptr);
    EXPECT_EQ(st.get(h2), nullptr);
}

TEST(State, ClearResetsCounter) {
    State st;
    st.store(x::telem::Series(x::telem::FLOAT32_T, 1));
    st.store(x::telem::Series(x::telem::FLOAT32_T, 1));
    st.clear();
    const uint32_t h = st.store(x::telem::Series(x::telem::FLOAT32_T, 1));
    EXPECT_EQ(h, 1);
}

TEST(State, GetReturnsMutableReference) {
    State st;
    auto s = x::telem::Series(x::telem::INT64_T, 2);
    s.set(0, int64_t{100});
    s.resize(1);
    const uint32_t h = st.store(std::move(s));
    auto *retrieved = st.get(h);
    ASSERT_NE(retrieved, nullptr);
    retrieved->set(0, int64_t{999});
    EXPECT_EQ(st.get(h)->at<int64_t>(0), 999);
}
}
