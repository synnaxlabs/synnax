// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <variant>

#include <include/gtest/gtest.h>

#include "client/cpp/arc/actions.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::arc {
/// @brief it should parse a set node position action from the JSON that it writes.
TEST(ActionsTests, testSetNodePositionRoundTrip) {
    const x::json::Parser parser(
        to_json(set_node_position({.key = "n1", .position = {.x = 1, .y = 2}}))
    );
    const auto action = parse_action(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<SetNodePositionPayload>(action));
    const auto &payload = std::get<SetNodePositionPayload>(action);
    ASSERT_EQ(payload.key, "n1");
    ASSERT_EQ(payload.position.x, 1);
    ASSERT_EQ(payload.position.y, 2);
}
}
