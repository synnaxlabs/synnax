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

/// Internal.
#include "synnax/framer/framer.h"

/// @brief it should construct a frame with a pre-allocated size.
TEST(FramerTests, testConstruction) {
    auto f = synnax::Frame(2);
    f.add(
            65537,
            synnax::Series(std::vector<float>{1, 2, 3})
    );
    ASSERT_EQ(f.size(), 1);
}

/// @brief it should construct a frame from a proto.
TEST(FramerTests, toProto) {
    auto f = synnax::Frame(2);
    f.add(
            65537,
            synnax::Series(std::vector<float>{1, 2, 3})
    );
    auto p = new api::v1::Frame();
    f.toProto(p);
    ASSERT_EQ(p->keys_size(), 1);
    ASSERT_EQ(p->series_size(), 1);
    auto f2 = synnax::Frame(*p);
    ASSERT_EQ(f2.size(), 1);
    ASSERT_EQ(f2.columns->at(0), 65537);
    ASSERT_EQ(f2.series->at(0).float32()[0], 1);
}