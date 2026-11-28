// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/list.h"

namespace driver::serial {
/// @brief it should list the ports of this computer sorted by path, with no empty path
/// or description.
TEST(List, ListsPortsOfComputer) {
    const auto entries = ASSERT_NIL_P(list());
    EXPECT_TRUE(std::ranges::is_sorted(entries, {}, &Entry::path));
    for (const auto &e: entries) {
        EXPECT_FALSE(e.path.empty());
        EXPECT_NE(e.description, "");
    }
}
}
