// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/base64/base64.h"
#include "x/cpp/test/test.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/mock/server.h"

namespace driver::http::device {
namespace {
ConnectionConfig make_config(const x::json::json &j, const bool verify_ssl = true) {
    x::json::Parser p(j);
    return ConnectionConfig(p, verify_ssl);
}
}

TEST(AuthConfigTest, ParsesAPIKey) {
    x::json::json j = {
        {"type", "api_key"},
        {"header", "X-API-Key"},
        {"key", "secret123"}
    };
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.type, "api_key");
    EXPECT_EQ(auth.header, "X-API-Key");
    EXPECT_EQ(auth.key, "secret123");
}

TEST(AuthConfigTest, ParsesBearerToken) {
    x::json::json j = {{"type", "bearer"}, {"token", "my-jwt"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.type, "bearer");
    EXPECT_EQ(auth.token, "my-jwt");
}

TEST(AuthConfigTest, ParsesBasic) {
    x::json::json j = {{"type", "basic"}, {"username", "user"}, {"password", "pass"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.type, "basic");
    EXPECT_EQ(auth.username, "user");
    EXPECT_EQ(auth.password, "pass");
}

TEST(AuthConfigTest, BearerMissingTokenErrors) {
    x::json::json j = {{"type", "bearer"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, BasicMissingFieldsErrors) {
    x::json::json j = {{"type", "basic"}, {"username", "user"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, APIKeyMissingFieldsErrors) {
    x::json::json j = {{"type", "api_key"}, {"header", "X-Key"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, UnknownTypeError) {
    x::json::json j = {{"type", "oauth2"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, NoneTypeNoErrors) {
    x::json::json j = {{"type", "none"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
}

TEST(AuthConfigTest, DefaultsToNone) {
    x::json::json j = x::json::json::object();
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.type, "none");
}

TEST(ConnectionConfigTest, FromJSON) {
    x::json::json j = {
        {"base_url", "http://192.168.1.100:8080"},
        {"timeout_ms", 5000},
        {"auth", {{"type", "bearer"}, {"token", "abc123"}}},
        {"headers", {{"X-Custom", "value"}}}
    };
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_EQ(config.base_url, "http://192.168.1.100:8080");
    EXPECT_EQ(config.timeout_ms, 5000);
    EXPECT_EQ(config.auth.type, "bearer");
    EXPECT_EQ(config.auth.token, "abc123");
    EXPECT_EQ(config.headers["X-Custom"], "value");
}

TEST(ConnectionConfigTest, DefaultsApplied) {
    x::json::json j = {{"base_url", "http://localhost"}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_EQ(config.base_url, "http://localhost");
    EXPECT_EQ(config.timeout_ms, 1000);
    EXPECT_EQ(config.auth.type, "none");
    EXPECT_TRUE(config.headers.empty());
}

TEST(ConnectionConfigTest, ToJSONRoundtrip) {
    auto config = make_config({
        {"base_url", "http://10.0.0.1:9090"},
        {"timeout_ms", 10000},
        {"auth", {{"type", "basic"}, {"username", "user"}, {"password", "pass"}}},
        {"headers", {{"Accept", "application/json"}}},
    });

    auto j = config.to_json();
    x::json::Parser parser(j);
    ConnectionConfig parsed(parser);

    EXPECT_EQ(parsed.base_url, config.base_url);
    EXPECT_EQ(parsed.timeout_ms, config.timeout_ms);
    EXPECT_EQ(parsed.auth.type, config.auth.type);
    EXPECT_EQ(parsed.auth.username, config.auth.username);
    EXPECT_EQ(parsed.auth.password, config.auth.password);
    EXPECT_EQ(parsed.headers, config.headers);
}

TEST(ConnectionConfigTest, MissingBaseURLErrors) {
    x::json::json j = {{"timeout_ms", 5000}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, InvalidAuthErrors) {
    x::json::json j = {
        {"base_url", "http://localhost"},
        {"auth", {{"type", "bearer"}}}
    };
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, ZeroTimeoutErrors) {
    x::json::json j = {{"base_url", "http://localhost"}, {"timeout_ms", 0}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, EmptyJSONErrors) {
    x::json::json j = x::json::json::object();
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ClientTest, GETRequest) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/data",
        .status_code = 200,
        .response_body = R"({"value": 42})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::GET, .path = "/api/data"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, R"({"value": 42})");
    EXPECT_GT(resp.time_range.end, resp.time_range.start);

    server.stop();
}

TEST(ClientTest, POSTRequestWithBody) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::POST,
        .path = "/api/submit",
        .status_code = 201,
        .response_body = R"({"id": 1})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::POST, .path = "/api/submit"}});

    auto results = client.request({R"({"name": "test"})"});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 201);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, R"({"name": "test"})");

    server.stop();
}

TEST(ClientTest, CustomHeaders) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/check",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({
        {"base_url", server.base_url()},
        {"headers", {{"X-Global", "global-val"}}},
    });
    Client client(
        config,
        {{
            .method = Method::GET,
            .path = "/api/check",
            .headers = {{"X-Request", "req-val"}},
        }}
    );

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    bool found_global = false;
    bool found_request = false;
    for (const auto &[k, v]: reqs[0].headers) {
        if (k == "X-Global" && v == "global-val") found_global = true;
        if (k == "X-Request" && v == "req-val") found_request = true;
    }
    EXPECT_TRUE(found_global);
    EXPECT_TRUE(found_request);

    server.stop();
}

TEST(ClientTest, BasicAuth) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/secure",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({
        {"base_url", server.base_url()},
        {"auth", {{"type", "basic"}, {"username", "user"}, {"password", "pass"}}},
    });
    Client client(config, {{.method = Method::GET, .path = "/api/secure"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    ASSERT_NIL_P(results[0]);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    const auto expected = "Basic " + x::base64::encode("user:pass");
    bool found_auth = false;
    for (const auto &[k, v]: reqs[0].headers)
        if (k == "Authorization" && v == expected) found_auth = true;
    EXPECT_TRUE(found_auth);

    server.stop();
}

TEST(ClientTest, BearerAuth) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/secure",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({
        {"base_url", server.base_url()},
        {"auth", {{"type", "bearer"}, {"token", "my-token"}}},
    });
    Client client(config, {{.method = Method::GET, .path = "/api/secure"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    ASSERT_NIL_P(results[0]);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    bool found_auth = false;
    for (const auto &[k, v]: reqs[0].headers)
        if (k == "Authorization" && v == "Bearer my-token") found_auth = true;
    EXPECT_TRUE(found_auth);

    server.stop();
}

TEST(ClientTest, APIKeyAuth) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/keyed",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({
        {"base_url", server.base_url()},
        {"auth", {{"type", "api_key"}, {"header", "X-API-Key"}, {"key", "secret123"}}},
    });
    Client client(config, {{.method = Method::GET, .path = "/api/keyed"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    ASSERT_NIL_P(results[0]);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    bool found_key = false;
    for (const auto &[k, v]: reqs[0].headers)
        if (k == "X-API-Key" && v == "secret123") found_key = true;
    EXPECT_TRUE(found_key);

    server.stop();
}

TEST(ClientTest, QueryParams) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/search",
        .status_code = 200,
        .response_body = "found",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {{
            .method = Method::GET,
            .path = "/api/search",
            .query_params = {{"q", "hello"}, {"limit", "10"}},
        }}
    );

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

TEST(ClientTest, QueryParamsPercentEncoded) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/search",
        .status_code = 200,
        .response_body = "found",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {{
            .method = Method::GET,
            .path = "/api/search",
            .query_params = {{"q", "hello world"}, {"tag", "a&b=c"}},
        }}
    );

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].query_params.find("q")->second, "hello world");
    EXPECT_EQ(reqs[0].query_params.find("tag")->second, "a&b=c");

    server.stop();
}

TEST(ClientTest, TimeoutError) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/slow",
        .status_code = 200,
        .response_body = "delayed",
        .content_type = "text/plain",
        .delay = std::chrono::milliseconds(2000),
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({
        {"base_url", server.base_url()},
        {"timeout_ms", 1},
    });
    Client client(config, {{.method = Method::GET, .path = "/api/slow"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    ASSERT_OCCURRED_AS_P(results[0], errors::UNREACHABLE_ERROR);

    server.stop();
}

TEST(ClientTest, UnreachableError) {
    auto config = make_config({{"base_url", "http://192.0.2.1:1"}});
    Client client(config, {{.method = Method::GET, .path = "/"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    ASSERT_OCCURRED_AS_P(results[0], errors::UNREACHABLE_ERROR);
}

TEST(ClientTest, ErrorStatusCodesReturnResponse) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {
            .method = Method::GET,
            .path = "/api/notfound",
            .status_code = 404,
            .response_body = R"({"error": "not found"})",
        },
        {
            .method = Method::GET,
            .path = "/api/error",
            .status_code = 500,
            .response_body = R"({"error": "internal"})",
        },
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    {
        auto config = make_config({{"base_url", server.base_url()}});
        Client client(config, {{.method = Method::GET, .path = "/api/notfound"}});
        auto results = client.request({""});
        ASSERT_EQ(results.size(), 1);
        const auto resp = ASSERT_NIL_P(results[0]);
        EXPECT_EQ(resp.status_code, 404);
    }

    {
        auto config = make_config({{"base_url", server.base_url()}});
        Client client(config, {{.method = Method::GET, .path = "/api/error"}});
        auto results = client.request({""});
        ASSERT_EQ(results.size(), 1);
        const auto resp = ASSERT_NIL_P(results[0]);
        EXPECT_EQ(resp.status_code, 500);
    }

    server.stop();
}

TEST(ClientTest, ParallelRequests) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = Method::GET,
         .path = "/api/a",
         .status_code = 200,
         .response_body = "A"},
        {.method = Method::GET,
         .path = "/api/b",
         .status_code = 200,
         .response_body = "B"},
        {.method = Method::GET,
         .path = "/api/c",
         .status_code = 200,
         .response_body = "C"},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {
            {.method = Method::GET, .path = "/api/a"},
            {.method = Method::GET, .path = "/api/b"},
            {.method = Method::GET, .path = "/api/c"},
        }
    );

    auto results = client.request({"", "", ""});
    ASSERT_EQ(results.size(), 3);
    for (auto &r: results) {
        const auto resp = ASSERT_NIL_P(r);
        EXPECT_EQ(resp.status_code, 200);
    }

    server.stop();
}

TEST(ClientTest, ParallelRequestsWithMixedStatusCodes) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = Method::GET,
         .path = "/ok",
         .status_code = 200,
         .response_body = "success"},
        {.method = Method::GET,
         .path = "/not-found",
         .status_code = 404,
         .response_body = R"({"error": "not found"})"},
        {.method = Method::GET,
         .path = "/error",
         .status_code = 500,
         .response_body = R"({"error": "internal"})"},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {
            {.method = Method::GET, .path = "/ok"},
            {.method = Method::GET, .path = "/not-found"},
            {.method = Method::GET, .path = "/error"},
        }
    );

    auto results = client.request({"", "", ""});
    ASSERT_EQ(results.size(), 3);
    for (auto &r: results)
        ASSERT_NIL(r.second);
    EXPECT_EQ(results[0].first.status_code, 200);
    EXPECT_EQ(results[0].first.body, "success");
    EXPECT_EQ(results[1].first.status_code, 404);
    EXPECT_EQ(results[1].first.body, R"({"error": "not found"})");
    EXPECT_EQ(results[2].first.status_code, 500);
    EXPECT_EQ(results[2].first.body, R"({"error": "internal"})");

    server.stop();
}

TEST(ClientTest, ParallelOneTimesOut) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = Method::GET,
         .path = "/fast",
         .status_code = 200,
         .response_body = "fast"},
        {.method = Method::GET,
         .path = "/slow",
         .status_code = 200,
         .response_body = "slow",
         .delay = std::chrono::milliseconds(2000)},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({
        {"base_url", server.base_url()},
        {"timeout_ms", 500},
    });
    Client client(
        config,
        {
            {.method = Method::GET, .path = "/fast"},
            {.method = Method::GET, .path = "/slow"},
        }
    );

    auto results = client.request({"", ""});
    ASSERT_EQ(results.size(), 2);
    const auto fast = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(fast.status_code, 200);
    EXPECT_EQ(fast.body, "fast");
    ASSERT_OCCURRED_AS_P(results[1], errors::UNREACHABLE_ERROR);
    EXPECT_EQ(results[1].first.status_code, 0);

    server.stop();
}

TEST(ClientTest, ParallelFirstTimesOutSecondSucceeds) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = Method::GET,
         .path = "/slow",
         .status_code = 200,
         .response_body = "slow",
         .delay = std::chrono::milliseconds(2000)},
        {.method = Method::GET,
         .path = "/fast",
         .status_code = 200,
         .response_body = "fast"},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({
        {"base_url", server.base_url()},
        {"timeout_ms", 500},
    });
    Client client(
        config,
        {
            {.method = Method::GET, .path = "/slow"},
            {.method = Method::GET, .path = "/fast"},
        }
    );

    auto results = client.request({"", ""});
    ASSERT_EQ(results.size(), 2);
    ASSERT_OCCURRED_AS_P(results[0], errors::UNREACHABLE_ERROR);
    EXPECT_EQ(results[0].first.status_code, 0);
    const auto fast = ASSERT_NIL_P(results[1]);
    EXPECT_EQ(fast.status_code, 200);
    EXPECT_EQ(fast.body, "fast");

    server.stop();
}

TEST(ClientTest, ParallelPerResponseTimeRanges) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = Method::GET,
         .path = "/fast",
         .status_code = 200,
         .response_body = "fast",
         .content_type = "text/plain"},
        {.method = Method::GET,
         .path = "/slow",
         .status_code = 200,
         .response_body = "slow",
         .content_type = "text/plain",
         .delay = std::chrono::milliseconds(300)},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {
            {.method = Method::GET, .path = "/fast"},
            {.method = Method::GET, .path = "/slow"},
        }
    );

    auto results = client.request({"", ""});
    ASSERT_EQ(results.size(), 2);
    const auto fast = ASSERT_NIL_P(results[0]);
    const auto slow = ASSERT_NIL_P(results[1]);

    // Both share the same start time.
    EXPECT_EQ(fast.time_range.start, slow.time_range.start);

    // The slow response should have a later end time than the fast one.
    EXPECT_GT(slow.time_range.end, fast.time_range.end);

    server.stop();
}

TEST(ClientTest, RepeatedGETRequests) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/poll",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::GET, .path = "/api/poll"}});

    for (int i = 0; i < 5; i++) {
        auto results = client.request({""});
        ASSERT_EQ(results.size(), 1);
        const auto resp = ASSERT_NIL_P(results[0]);
        EXPECT_EQ(resp.status_code, 200);
        EXPECT_EQ(resp.body, "ok");
    }

    EXPECT_EQ(server.received_requests().size(), 5);

    server.stop();
}

TEST(ClientTest, RepeatedPOSTRequests) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::POST,
        .path = "/api/send",
        .status_code = 201,
        .response_body = "created",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::POST, .path = "/api/send"}});

    for (int i = 0; i < 3; i++) {
        const auto body = R"({"i": )" + std::to_string(i) + "}";
        auto results = client.request({body});
        ASSERT_EQ(results.size(), 1);
        const auto resp = ASSERT_NIL_P(results[0]);
        EXPECT_EQ(resp.status_code, 201);
    }

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 3);
    EXPECT_EQ(reqs[0].body, R"({"i": 0})");
    EXPECT_EQ(reqs[1].body, R"({"i": 1})");
    EXPECT_EQ(reqs[2].body, R"({"i": 2})");

    server.stop();
}

TEST(ClientTest, MixedGETAndPOST) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = Method::GET,
         .path = "/api/read",
         .status_code = 200,
         .response_body = "read-ok",
         .content_type = "text/plain"},
        {.method = Method::POST,
         .path = "/api/write",
         .status_code = 201,
         .response_body = "write-ok",
         .content_type = "text/plain"},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {
            {.method = Method::GET, .path = "/api/read"},
            {.method = Method::POST, .path = "/api/write"},
        }
    );

    auto results = client.request({"", R"({"val": 1})"});
    ASSERT_EQ(results.size(), 2);
    const auto read_resp = ASSERT_NIL_P(results[0]);
    const auto write_resp = ASSERT_NIL_P(results[1]);
    EXPECT_EQ(read_resp.status_code, 200);
    EXPECT_EQ(read_resp.body, "read-ok");
    EXPECT_EQ(write_resp.status_code, 201);
    EXPECT_EQ(write_resp.body, "write-ok");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 2);

    server.stop();
}

TEST(ClientTest, POSTWithEmptyBody) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::POST,
        .path = "/api/ping",
        .status_code = 200,
        .response_body = "pong",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::POST, .path = "/api/ping"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

TEST(ClientTest, DeleteRequest) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::DELETE,
        .path = "/api/item/42",
        .status_code = 204,
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::DELETE, .path = "/api/item/42"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 204);

    server.stop();
}

TEST(ClientTest, PutRequest) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::PUT,
        .path = "/api/item/1",
        .status_code = 200,
        .response_body = R"({"updated": true})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::PUT, .path = "/api/item/1"}});

    auto results = client.request({R"({"name": "new"})"});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, R"({"name": "new"})");

    server.stop();
}

TEST(ClientTest, PathWithoutLeadingSlash) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/data",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::GET, .path = "api/data"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

TEST(ClientTest, BaseURLWithTrailingSlash) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/data",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url() + "/"}});
    Client client(config, {{.method = Method::GET, .path = "/api/data"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

TEST(ClientTest, EmptyPath) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/",
        .status_code = 200,
        .response_body = "root",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::GET}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, "root");

    server.stop();
}

TEST(ClientTest, HTTPSGETRequest) {
    mock::ServerConfig server_cfg;
    server_cfg.secure = true;
    server_cfg.cert_path = "driver/http/mock/test_cert.pem";
    server_cfg.key_path = "driver/http/mock/test_key.pem";
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/secure",
        .status_code = 200,
        .response_body = R"({"secure": true})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}}, false);
    Client client(config, {{.method = Method::GET, .path = "/api/secure"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, R"({"secure": true})");

    server.stop();
}

TEST(ClientTest, HTTPSPOSTRequestWithBody) {
    mock::ServerConfig server_cfg;
    server_cfg.secure = true;
    server_cfg.cert_path = "driver/http/mock/test_cert.pem";
    server_cfg.key_path = "driver/http/mock/test_key.pem";
    server_cfg.routes = {{
        .method = Method::POST,
        .path = "/api/submit",
        .status_code = 201,
        .response_body = R"({"id": 1})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}}, false);
    Client client(config, {{.method = Method::POST, .path = "/api/submit"}});

    auto results = client.request({R"({"name": "test"})"});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 201);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, R"({"name": "test"})");

    server.stop();
}

TEST(ClientTest, ContentTypeValidation) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/json",
        .status_code = 200,
        .response_body = R"({"ok": true})",
        .content_type = "application/json",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {{
            .method = Method::GET,
            .path = "/api/json",
            .response_content_type = "application/json",
        }}
    );

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

TEST(ClientTest, ContentTypeMismatch) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/text",
        .status_code = 200,
        .response_body = "not json",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {{
            .method = Method::GET,
            .path = "/api/text",
            .response_content_type = "application/json",
        }}
    );

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    ASSERT_OCCURRED_AS_P(results[0], errors::PARSE_ERROR);

    server.stop();
}

TEST(ClientTest, ContentTypeNotCheckedWhenEmpty) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/any",
        .status_code = 200,
        .response_body = "whatever",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::GET, .path = "/api/any"}});

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    ASSERT_NIL_P(results[0]);

    server.stop();
}

TEST(ClientTest, ContentTypeCharsetSuffix) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/charset",
        .status_code = 200,
        .response_body = R"({"ok": true})",
        .content_type = "application/json; charset=utf-8",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {{
            .method = Method::GET,
            .path = "/api/charset",
            .response_content_type = "application/json",
        }}
    );

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    const auto resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

TEST(ClientTest, RequestContentTypeHeaderSent) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::POST,
        .path = "/api/xml",
        .status_code = 200,
        .response_body = "<ok/>",
        .content_type = "application/xml",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {{
            .method = Method::POST,
            .path = "/api/xml",
            .request_content_type = "application/xml",
        }}
    );

    auto results = client.request({"<req/>"});
    ASSERT_EQ(results.size(), 1);
    ASSERT_NIL_P(results[0]);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    bool found_ct = false;
    for (const auto &[k, v]: reqs[0].headers)
        if (k == "Content-Type" && v == "application/xml") found_ct = true;
    EXPECT_TRUE(found_ct);

    server.stop();
}

TEST(ClientTest, RequestContentTypeOmittedWhenEmpty) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::POST,
        .path = "/api/raw",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(config, {{.method = Method::POST, .path = "/api/raw"}});

    auto results = client.request({"data"});
    ASSERT_EQ(results.size(), 1);
    ASSERT_NIL_P(results[0]);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    bool found_ct = false;
    for (const auto &[k, v]: reqs[0].headers)
        if (k == "Content-Type") found_ct = true;
    EXPECT_FALSE(found_ct);

    server.stop();
}

TEST(ClientTest, AcceptHeaderSent) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/accept",
        .status_code = 200,
        .response_body = R"({"ok": true})",
        .content_type = "application/json",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto config = make_config({{"base_url", server.base_url()}});
    Client client(
        config,
        {{
            .method = Method::GET,
            .path = "/api/accept",
            .response_content_type = "application/json",
        }}
    );

    auto results = client.request({""});
    ASSERT_EQ(results.size(), 1);
    ASSERT_NIL_P(results[0]);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    bool found_accept = false;
    for (const auto &[k, v]: reqs[0].headers)
        if (k == "Accept" && v == "application/json") found_accept = true;
    EXPECT_TRUE(found_accept);

    server.stop();
}
}
