// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/lib/lib.h"
#include "x/cpp/test/test.h"

#include "driver/mil1553/ballard/backend.h"

namespace driver::mil1553::ballard {
TEST(Backend, FailsToOpenWithoutTheBTIDriver) {
    Backend b;
    const auto [ch, err] = b.open({});
    ASSERT_OCCURRED_AS(err, x::lib::LOAD_ERROR);
    EXPECT_EQ(err.data, "Ballard BTIDriver MIL-STD-1553 library is not installed.");
}

TEST(Backend, FailsToListWithoutTheBTIDriver) {
    Backend b;
    ASSERT_OCCURRED_AS_P(b.list(), x::lib::LOAD_ERROR);
}
}
