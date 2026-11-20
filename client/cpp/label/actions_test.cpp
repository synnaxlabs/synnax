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

#include "client/cpp/label/actions.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::label {
/// @brief it should parse a create action from the JSON that it writes.
TEST(ActionsTests, testCreateRoundTrip) {
    const x::json::Parser parser(to_json(create({.label = {.name = "Hot"}})));
    const auto action = parse_action(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<CreatePayload>(action));
    ASSERT_EQ(std::get<CreatePayload>(action).label.name, "Hot");
}
}
