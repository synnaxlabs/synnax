// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/sync/list.h"

namespace x::sync {
/// @brief it should return pushed items in order.
TEST(List, PushBackAndAt) {
    List<int> list;
    list.push_back(1);
    list.push_back(2);
    ASSERT_EQ(list.size(), 2);
    EXPECT_EQ(list.at(0), 1);
    EXPECT_EQ(list.at(1), 2);
}

/// @brief it should throw on an index past the end.
TEST(List, AtOutOfRangeThrows) {
    const List<int> list;
    EXPECT_THROW((void) list.at(0), std::out_of_range);
}

/// @brief it should keep a reference from at() valid while more items are appended.
TEST(List, ReferenceSurvivesPushBack) {
    List<int> list;
    list.push_back(7);
    const auto &first = list.at(0);
    for (int i = 0; i < 10000; i++)
        list.push_back(i);
    EXPECT_EQ(first, 7);
}

/// @brief it should hold items that are move-only.
TEST(List, MoveOnlyItems) {
    List<std::unique_ptr<int>> list;
    list.push_back(std::make_unique<int>(3));
    EXPECT_EQ(*list.at(0), 3);
}

/// @brief it should iterate the items present when snapshot() was called.
TEST(List, SnapshotExcludesLaterItems) {
    List<int> list;
    list.push_back(1);
    list.push_back(2);
    const auto snap = list.snapshot();
    list.push_back(3);
    std::vector<int> seen;
    for (const auto &v: snap)
        seen.push_back(v);
    EXPECT_EQ(seen, (std::vector{1, 2}));
}

/// @brief it should remove every item on clear().
TEST(List, Clear) {
    List<int> list;
    list.push_back(1);
    list.clear();
    EXPECT_TRUE(list.empty());
}

/// @brief it should allow one thread to append while another reads.
TEST(List, ConcurrentPushAndRead) {
    List<int> list;
    constexpr int n = 100000;
    std::thread writer([&] {
        for (int i = 0; i < n; i++)
            list.push_back(i);
    });
    size_t last = 0;
    while (last < n) {
        const auto size = list.size();
        if (size > 0) { EXPECT_EQ(list.at(size - 1), static_cast<int>(size - 1)); }
        last = size;
    }
    writer.join();
    EXPECT_EQ(list.size(), n);
}
}
