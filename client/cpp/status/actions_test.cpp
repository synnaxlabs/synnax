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

#include "client/cpp/status/actions.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::status {
/// @brief it should parse a set action from the JSON that it writes.
TEST(ActionsTests, testSetRoundTrip) {
    const x::json::Parser parser(to_json(
        set({.status = {.key = "pump", .variant = "success", .message = "Started"}})
    ));
    const auto action = parse_action(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<SetPayload>(action));
    const auto &payload = std::get<SetPayload>(action);
    ASSERT_EQ(payload.status.key, "pump");
    ASSERT_EQ(payload.status.message, "Started");
}
}
