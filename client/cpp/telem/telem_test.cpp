// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/telem/telem.h"

#include "gtest/gtest.h"

using namespace synnax;

/// @brief - it should initialize a timestamp from a long.
TEST(TimeStampTests, testContructor)
{
    auto ts = TimeStamp(5);
    ASSERT_EQ(ts.value, 5);
}

TEST(TimeSpanTests, testPeriod)
{
    auto r = Rate(1);
    ASSERT_EQ(r.period().value, synnax::SECOND.value);
    auto r2  = Rate(2);
    ASSERT_EQ(r2.period().value, synnax::SECOND.value / 2);
}