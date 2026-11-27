// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/spatial/json.gen.h"
#include "x/cpp/spatial/spatial.h"
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

/// @brief it should parse bounds whose lower is below upper.
TEST(ClosedBounds, ParsesOrderedBounds) {
    x::json::Parser parser(x::json::json{{"lower", 0.0}, {"upper", 10.0}});
    const auto bounds = ClosedBounds::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(bounds.lower, 0);
    EXPECT_EQ(bounds.upper, 10);
}

/// @brief it should refuse bounds whose lower is not below upper.
TEST(ClosedBounds, RefusesLowerNotBelowUpper) {
    for (const auto &[lower, upper]: {std::pair{5.0, 5.0}, std::pair{10.0, 0.0}}) {
        x::json::Parser parser(x::json::json{{"lower", lower}, {"upper", upper}});
        ClosedBounds::parse(parser);
        ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    }
}

/// @brief it should return upper minus lower.
TEST(ClosedBounds, Span) {
    EXPECT_EQ(span(ClosedBounds{.lower = -10, .upper = 10}), 20);
    EXPECT_EQ(span(ClosedBounds{.lower = 2, .upper = 1}), -1);
}
}
