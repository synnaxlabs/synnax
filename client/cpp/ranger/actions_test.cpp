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

#include "client/cpp/ranger/actions.gen.h"
#include "client/cpp/ranger/ranger.h"
#include "freighter/cpp/testutil/testutil.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::ranger {
/// @brief it should write a set action in the JSON form that the Core reads.
TEST(ActionsTests, testSetToJSON) {
    const auto action = set({.path = {"time_range", "end"}, .value = 100});
    const auto expected = x::json::json::parse(R"({
        "type": "set",
        "set": {"path": ["time_range", "end"], "value": 100, "guards": []}
    })");
    ASSERT_EQ(to_json(action), expected);
}

/// @brief it should parse a set action from the JSON that it writes.
TEST(ActionsTests, testSetRoundTrip) {
    const x::json::Parser parser(
        to_json(set({.path = {"time_range", "end"}, .value = 100}))
    );
    const auto action = parse_action(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<SetPayload>(action));
    const auto &payload = std::get<SetPayload>(action);
    ASSERT_EQ(payload.path, std::vector<std::string>({"time_range", "end"}));
    ASSERT_EQ(payload.value, 100);
    ASSERT_TRUE(payload.guards.empty());
}

/// @brief it should bind an error when the action type is unknown.
TEST(ActionsTests, testParseUnknownType) {
    const x::json::Parser parser(x::json::json{{"type", "explode"}});
    parse_action(parser);
    const auto err = parser.error();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_NE(err.data.find("unknown Action type: explode"), std::string::npos);
}

/// @brief it should send the end as a merge into the time range of the range.
TEST(ClientTests, testSetEnd) {
    auto dispatch = std::make_unique<freighter::testutil::UnaryClient<
        grpc::ranger::DispatchRequest,
        google::protobuf::Empty>>(google::protobuf::Empty(), x::errors::NIL);
    const auto *sent = dispatch.get();
    const Client client(nullptr, nullptr, std::move(dispatch), kv::Client());
    const auto key = x::uuid::create();
    ASSERT_NIL(client.set_end(key, x::telem::TimeStamp(1'700'000'000'123'456'789)));
    ASSERT_EQ(sent->requests.size(), 1);
    ASSERT_EQ(sent->requests[0].key(), key.to_string());
    ASSERT_EQ(sent->requests[0].actions_size(), 1);
    ASSERT_EQ(
        x::json::json::parse(sent->requests[0].actions(0)),
        x::json::json::parse(R"({
        "type": "update",
        "update": {"value": {"time_range": {"end": 1700000000123456789}}, "guards": []}
    })")
    );
}
}
