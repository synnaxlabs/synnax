// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>

#include "gtest/gtest.h"

#include "driver/arinc429/queue.h"

namespace driver::arinc429 {
TEST(Queue, DropsTheOldestItemWhenFull) {
    Queue<int> q(2);
    q.push(1);
    q.push(2);
    q.push(3);
    std::array<int, 4> out{};
    const auto b = q.pop(out, x::telem::TimeSpan(0));
    EXPECT_EQ(b.count, 2);
    EXPECT_EQ(b.dropped, 1);
    EXPECT_EQ(out[0], 2);
    EXPECT_EQ(out[1], 3);
}

TEST(Queue, ReturnsNothingAfterTheTimeout) {
    Queue<int> q(2);
    std::array<int, 1> out{};
    const auto b = q.pop(out, x::telem::MILLISECOND);
    EXPECT_EQ(b.count, 0);
    EXPECT_EQ(b.dropped, 0);
}
}
