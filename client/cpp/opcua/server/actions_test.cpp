// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <variant>

#include "gtest/gtest.h"

#include "client/cpp/opcua/server/actions.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::opcua::server {
/// @brief it should parse a create action from the JSON that it writes.
TEST(Actions, CreateRoundTrip) {
    Server s;
    s.name = "PLC";
    s.connection.endpoint = "opc.tcp://plc:4840";
    const x::json::Parser parser(to_json(create({.server = s})));
    const auto action = parse_action(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<CreatePayload>(action));
    ASSERT_EQ(std::get<CreatePayload>(action).server.name, "PLC");
}

/// @brief it should parse a set action from the JSON that it writes.
TEST(Actions, SetRoundTrip) {
    const x::json::Parser parser(
        to_json(set({.path = {"connection", "endpoint"}, .value = "opc.tcp://b:4840"}))
    );
    const auto action = parse_action(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<SetPayload>(action));
    const auto &payload = std::get<SetPayload>(action);
    ASSERT_EQ(payload.path, (std::vector<std::string>{"connection", "endpoint"}));
    ASSERT_EQ(payload.value, "opc.tcp://b:4840");
}
}
