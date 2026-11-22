// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/http/server/json.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::http::server {
namespace {
x::json::json valid_server() {
    return x::json::json{
        {"key", "22222222-2222-4222-8222-222222222222"},
        {"rack", 1},
        {"name", "Weather API"},
        {"host", "api.example.com"},
    };
}
}

/// @brief it should parse a server and apply the defaults.
TEST(Server, AppliesDefaults) {
    x::json::Parser parser(valid_server());
    const auto server = Server::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(server.scheme, SCHEME_HTTPS);
    ASSERT_FALSE(server.certificate_validation_disabled);
    ASSERT_EQ(server.timeout_ms, 100);
    ASSERT_EQ(server.max_concurrent_requests, 6);
    ASSERT_TRUE(std::holds_alternative<NoneAuth>(server.auth));
    ASSERT_EQ(server.health_check.method, ::synnax::http::METHOD_GET);
    ASSERT_FALSE(server.health_check.response.has_value());
    ASSERT_FALSE(server.status.has_value());
}

/// @brief it should round trip a server through JSON.
TEST(Server, RoundTrips) {
    x::json::Parser parser(valid_server());
    auto server = Server::parse(parser);
    ASSERT_NIL(parser.error());
    const auto key = x::uuid::create();
    server.auth = APIKeyAuth{.key = key, .send_as = SEND_AS_QUERY_PARAM, .name = "k"};
    server.read_channels["/weather"] = ReadChannels{
        .index = 1,
        .channels = {{"/t", 2}}
    };
    server.write_channels["/valve"] = 3;
    x::json::Parser round(server.to_json());
    const auto parsed = Server::parse(round);
    ASSERT_NIL(round.error());
    const auto &auth = std::get<APIKeyAuth>(parsed.auth);
    ASSERT_EQ(auth.key, key);
    ASSERT_EQ(auth.send_as, SEND_AS_QUERY_PARAM);
    ASSERT_EQ(parsed.read_channels.at("/weather").channels.at("/t"), 2);
    ASSERT_EQ(parsed.write_channels.at("/valve"), 3);
}

/// @brief it should reject an auth that names no secret.
TEST(Server, RequiresAuthSecret) {
    for (const auto &[auth, field]: std::vector<std::pair<x::json::json, std::string>>{
             {{{"type", "bearer"}}, "token"},
             {{{"type", "basic"}, {"username", "admin"}}, "password"},
             {{{"type", "api_key"}, {"name", "X-Key"}}, "key"},
         }) {
        auto j = valid_server();
        j["auth"] = auth;
        x::json::Parser parser(j);
        Server::parse(parser);
        ASSERT_EQ(parser.errors->size(), 1);
        ASSERT_EQ(parser.errors->at(0)["path"], "auth." + field);
    }
}
}
