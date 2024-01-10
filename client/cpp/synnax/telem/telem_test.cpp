// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// GTest.
#include <include/gtest/gtest.h>

/// Local headers.
#include "synnax/telem/telem.h"

/// std.

using namespace synnax;

/// @brief - it should initialize a timestamp from a long.
TEST(TimeStampTests, testContructor)
{
    auto ts = TimeStamp(5);
    ASSERT_EQ(ts.value, 5);
}
