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
#include "x/cpp/telem/telem.h"
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
    ASSERT_TRUE(std::holds_alternative<AnonymousAuth>(server.connection.auth));
    ASSERT_TRUE(std::holds_alternative<NoneSecurity>(server.connection.security));
    ASSERT_EQ(server.connection.client_timeout, x::telem::SECOND * 5);
    ASSERT_TRUE(server.reads.empty());
    ASSERT_FALSE(server.status.has_value());
}

/// @brief it should round trip a secured server with bindings through JSON.
TEST(Server, RoundTrips) {
    x::json::Parser parser(valid_server());
    auto server = Server::parse(parser);
    ASSERT_NIL(parser.error());
    UsernameAuth auth;
    auth.username = "op";
    auth.password = x::uuid::create();
    server.connection.auth = auth;
    SignAndEncryptSecurity security;
    security.policy = SECURITY_POLICY_BASIC_256_SHA_256;
    PinnedServerTrust trust;
    trust.certificate = x::uuid::create();
    security.server_trust = trust;
    server.connection.security = security;
    server.reads.push_back(
        Read{.node = "nsu=urn:plc;s=Temp", .index = 3, .channel = 4}
    );
    server.writes.push_back(Write{.node = "nsu=urn:plc;s=Valve", .channel = 5});
    x::json::Parser round(server.to_json());
    const auto parsed = Server::parse(round);
    ASSERT_NIL(round.error());
    const auto &parsed_auth = std::get<UsernameAuth>(parsed.connection.auth);
    ASSERT_EQ(parsed_auth.password, auth.password);
    const auto &parsed_security = std::get<SignAndEncryptSecurity>(
        parsed.connection.security
    );
    ASSERT_EQ(parsed_security.policy, SECURITY_POLICY_BASIC_256_SHA_256);
    ASSERT_EQ(
        std::get<PinnedServerTrust>(parsed_security.server_trust).certificate,
        trust.certificate
    );
    ASSERT_EQ(parsed.reads.at(0).index, 3);
    ASSERT_EQ(parsed.writes.at(0).node, "nsu=urn:plc;s=Valve");
}

/// @brief it should fail to parse a server without a rack.
TEST(Server, RequiresRack) {
    auto j = valid_server();
    j.erase("rack");
    x::json::Parser parser(j);
    Server::parse(parser);
    ASSERT_FALSE(parser.ok());
}

/// @brief it should fail to parse a certificate login without a certificate.
TEST(Server, RequiresLoginCertificate) {
    auto j = valid_server();
    j["connection"]["auth"] = {{"type", "certificate"}};
    x::json::Parser parser(j);
    Server::parse(parser);
    ASSERT_FALSE(parser.ok());
}

/// @brief it should fail to parse a pinned trust without a certificate.
TEST(Server, RequiresPinnedCertificate) {
    auto j = valid_server();
    j["connection"]["security"] = {
        {"mode", "sign"},
        {"policy", "Basic256Sha256"},
        {"server_trust", {{"type", "pinned"}}},
    };
    x::json::Parser parser(j);
    Server::parse(parser);
    ASSERT_FALSE(parser.ok());
}
}
