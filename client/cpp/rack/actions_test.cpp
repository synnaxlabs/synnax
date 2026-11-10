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

#include "client/cpp/rack/actions.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::rack {
/// @brief it should parse a create action from the JSON that it writes.
TEST(ActionsTests, testCreateRoundTrip) {
    const x::json::Parser parser(
        to_json(create({.rack = {.key = 12, .name = "Edge"}}))
    );
    const auto action = parse_action(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<CreatePayload>(action));
    const auto &payload = std::get<CreatePayload>(action);
    ASSERT_EQ(payload.rack.key, 12);
    ASSERT_EQ(payload.rack.name, "Edge");
}
}
