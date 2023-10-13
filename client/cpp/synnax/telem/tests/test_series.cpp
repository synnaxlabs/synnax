// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// GTest.
#include <gtest/gtest.h>

/// Local headers.
#include "synnax/telem/series.h"

/// std.
#include <iostream>

/// @brief create basic int frame
TEST(FrameTests, testString)
{
    std::vector<std::any> vals;
    vals.push_back(5);
    Synnax::Telem::Series s{vals};
    std::vector<std::any> raw_vals = s.getRaw();
    auto type_name = s.getDataType().name();
    ASSERT_EQ(type_name, "int");
}