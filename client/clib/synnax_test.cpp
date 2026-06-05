// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>

#include "gtest/gtest.h"

#include "client/clib/synnax.h"

TEST(ClibVersion, ReturnsNonEmptyVersion) {
    char buf[64];
    const int32_t n = synnax_version(buf, sizeof(buf));
    EXPECT_GT(n, 0);
    EXPECT_EQ(static_cast<size_t>(n), std::strlen(buf));
}

TEST(ClibVersion, TruncatesToBuffer) {
    char buf[4] = {'x', 'x', 'x', 'x'};
    const int32_t n = synnax_version(buf, sizeof(buf));
    EXPECT_EQ('\0', buf[sizeof(buf) - 1]);
    EXPECT_GT(static_cast<size_t>(n), sizeof(buf) - 1);
}

TEST(ClibVersion, ReturnsLengthWithNullBuffer) {
    EXPECT_GT(synnax_version(nullptr, 0), 0);
}
