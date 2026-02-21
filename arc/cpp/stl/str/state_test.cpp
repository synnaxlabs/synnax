// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <string>

#include "gtest/gtest.h"

#include "arc/cpp/stl/str/state.h"

namespace arc::stl::str {
TEST(State, CreateReturnsIncrementingHandles) {
    State st;
    const uint32_t h1 = st.create("hello");
    const uint32_t h2 = st.create("world");
    EXPECT_EQ(h1, 1);
    EXPECT_EQ(h2, 2);
}

TEST(State, CreateAndGet) {
    State st;
    const uint32_t h = st.create("hello");
    EXPECT_EQ(st.get(h), "hello");
}

TEST(State, FromMemory) {
    State st;
    const std::string data = "test string";
    const uint32_t h = st.from_memory(
        reinterpret_cast<const uint8_t *>(data.data()),
        static_cast<uint32_t>(data.size())
    );
    EXPECT_EQ(st.get(h), "test string");
}

TEST(State, FromMemoryPartialLength) {
    State st;
    const std::string data = "hello world";
    const uint32_t h = st.from_memory(
        reinterpret_cast<const uint8_t *>(data.data()),
        5
    );
    EXPECT_EQ(st.get(h), "hello");
}

TEST(State, GetReturnsEmptyForMissingHandle) {
    State st;
    EXPECT_EQ(st.get(999), "");
}

TEST(State, ExistsReturnsTrueForValidHandle) {
    State st;
    const uint32_t h = st.create("abc");
    EXPECT_TRUE(st.exists(h));
}

TEST(State, ExistsReturnsFalseForInvalidHandle) {
    State st;
    EXPECT_FALSE(st.exists(42));
}

TEST(State, ClearRemovesAllHandles) {
    State st;
    const uint32_t h1 = st.create("a");
    const uint32_t h2 = st.create("b");
    st.clear();
    EXPECT_FALSE(st.exists(h1));
    EXPECT_FALSE(st.exists(h2));
    EXPECT_EQ(st.get(h1), "");
}

TEST(State, ClearResetsCounter) {
    State st;
    st.create("first");
    st.create("second");
    st.clear();
    const uint32_t h = st.create("after clear");
    EXPECT_EQ(h, 1);
    EXPECT_EQ(st.get(h), "after clear");
}
}
