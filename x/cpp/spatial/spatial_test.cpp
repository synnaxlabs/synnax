// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/spatial/json.gen.h"
#include "x/cpp/spatial/types.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace x::spatial {
/// @brief it should fall back to the same anchor and units as a default StickyXY.
TEST(StickyXY, ParsesMissingRootAndUnits) {
    x::json::Parser parser(x::json::json{{"x", 1.0}, {"y", 2.0}});
    const auto sticky = StickyXY::parse(parser);
    ASSERT_NIL(parser.error());
    const StickyXY expected{};
    EXPECT_EQ(sticky.root.x, expected.root.x);
    EXPECT_EQ(sticky.root.y, expected.root.y);
    EXPECT_EQ(sticky.units.x, expected.units.x);
    EXPECT_EQ(sticky.units.y, expected.units.y);
}
}
