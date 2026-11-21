// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/opcua/server/json.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::opcua::server {
namespace {
x::json::json valid_server() {
    return x::json::json{
        {"key", "22222222-2222-4222-8222-222222222222"},
        {"rack", 1},
        {"name", "PLC"},
        {"connection", {{"endpoint", "opc.tcp://plc:4840"}}},
    };
}
}

/// @brief it should parse a server and apply the connection defaults.
TEST(Server, AppliesDefaults) {
    x::json::Parser parser(valid_server());
    const auto server = Server::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(server.connection.endpoint, "opc.tcp://plc:4840");
    ASSERT_EQ(server.connection.security_mode, SECURITY_MODE_NONE);
    ASSERT_EQ(server.connection.client_timeout_ms, 5000);
    ASSERT_FALSE(server.connection.password.has_value());
    ASSERT_TRUE(server.read_channels.empty());
    ASSERT_FALSE(server.status.has_value());
}

/// @brief it should round trip a server through JSON.
TEST(Server, RoundTrips) {
    x::json::Parser parser(valid_server());
    auto server = Server::parse(parser);
    ASSERT_NIL(parser.error());
    server.connection.password = x::uuid::create();
    server.read_indexes = {3};
    server.read_channels["ns=2;s=Tag_One"] = 4;
    server.read_fields.push_back(
        ReadField{.node_id = "ns=2;i=7", .field = "a.b", .channel = 5}
    );
    x::json::Parser round(server.to_json());
    const auto parsed = Server::parse(round);
    ASSERT_NIL(round.error());
    ASSERT_EQ(parsed.connection.password, server.connection.password);
    ASSERT_EQ(parsed.read_indexes, server.read_indexes);
    ASSERT_EQ(parsed.read_channels.at("ns=2;s=Tag_One"), 4);
    ASSERT_EQ(parsed.read_fields.at(0).field, "a.b");
}
}
