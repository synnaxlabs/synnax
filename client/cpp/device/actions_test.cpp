// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <variant>
#include <vector>

#include <include/gtest/gtest.h>

#include "client/cpp/device/actions.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::device {
const x::json::json PROPERTIES = {{"port", 502}};

/// @brief it should write a set action in the JSON form that the Core reads.
TEST(ActionsTests, testSetToJSON) {
    const auto action = set({.path = {"properties"}, .value = PROPERTIES});
    const auto expected = x::json::json::parse(R"({
        "type": "set",
        "set": {"path": ["properties"], "value": {"port": 502}, "guards": []}
    })");
    ASSERT_EQ(to_json(action), expected);
}

/// @brief it should parse a set action from the JSON that it writes.
TEST(ActionsTests, testSetRoundTrip) {
    const x::json::Parser parser(
        to_json(set({.path = {"properties"}, .value = PROPERTIES}))
    );
    const auto action = parse_action(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<SetPayload>(action));
    const auto &payload = std::get<SetPayload>(action);
    ASSERT_EQ(payload.path, std::vector<std::string>({"properties"}));
    ASSERT_EQ(payload.value, PROPERTIES);
    ASSERT_TRUE(payload.guards.empty());
}
}
