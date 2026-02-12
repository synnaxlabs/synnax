// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/http/device/device.h"
#include "driver/http/mock/server.h"

namespace driver::http::device {
// ─── ConnectionConfig ────────────────────────────────────────────────────── //

TEST(ConnectionConfigTest, FromJsonWorks) {
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
    EXPECT_EQ(config.timeout_ms, 30000);
    EXPECT_EQ(config.auth.type, "none");
    EXPECT_TRUE(config.headers.empty());
}

TEST(ConnectionConfigTest, ToJsonRoundtrip) {
    ConnectionConfig config;
    config.base_url = "http://10.0.0.1:9090";
    config.timeout_ms = 10000;
    config.auth.type = "basic";
    config.auth.username = "user";
    config.auth.password = "pass";
    config.headers["Accept"] = "application/json";

    auto j = config.to_json();
    x::json::Parser parser(j);
    ConnectionConfig parsed(parser);

    EXPECT_EQ(parsed.base_url, config.base_url);
    EXPECT_EQ(parsed.timeout_ms, config.timeout_ms);
    EXPECT_EQ(parsed.auth.type, config.auth.type);
    EXPECT_EQ(parsed.auth.username, config.auth.username);
    EXPECT_EQ(parsed.auth.password, config.auth.password);
}

TEST(AuthConfigTest, ParsesAllTypes) {
    x::json::json j = {
        {"type", "api_key"},
        {"header", "X-API-Key"},
        {"key", "secret123"}
    };
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_EQ(auth.type, "api_key");
    EXPECT_EQ(auth.header, "X-API-Key");
    EXPECT_EQ(auth.key, "secret123");
}

// ─── Client GET ──────────────────────────────────────────────────────────── //

TEST(ClientTest, GetRequest) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = "GET",
        .path = "/api/data",
        .status_code = 200,
        .response_body = R"({"value": 42})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ConnectionConfig config;
    config.base_url = server.base_url();
    Client client(config);

    Request req;
    req.method = "GET";
    req.path = "/api/data";

    auto [resp, err] = client.request(req);
    ASSERT_NIL(err);
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, R"({"value": 42})");
    EXPECT_GT(resp.request_end, resp.request_start);

    server.stop();
}

// ─── Client POST ─────────────────────────────────────────────────────────── //

TEST(ClientTest, PostWithBody) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = "POST",
        .path = "/api/submit",
        .status_code = 201,
        .response_body = R"({"id": 1})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ConnectionConfig config;
    config.base_url = server.base_url();
    Client client(config);

    Request req;
    req.method = "POST";
    req.path = "/api/submit";
    req.body = R"({"name": "test"})";

    auto [resp, err] = client.request(req);
    ASSERT_NIL(err);
    EXPECT_EQ(resp.status_code, 201);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, R"({"name": "test"})");

    server.stop();
}

// ─── Client Custom Headers ───────────────────────────────────────────────── //

TEST(ClientTest, CustomHeaders) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = "GET",
        .path = "/api/check",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ConnectionConfig config;
    config.base_url = server.base_url();
    config.headers["X-Global"] = "global-val";
    Client client(config);

    Request req;
    req.method = "GET";
    req.path = "/api/check";
    req.headers["X-Request"] = "req-val";

    auto [resp, err] = client.request(req);
    ASSERT_NIL(err);
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

// ─── Client Auth: Bearer ─────────────────────────────────────────────────── //

TEST(ClientTest, BearerAuth) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = "GET",
        .path = "/api/secure",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ConnectionConfig config;
    config.base_url = server.base_url();
    config.auth.type = "bearer";
    config.auth.token = "my-token";
    Client client(config);

    Request req;
    req.method = "GET";
    req.path = "/api/secure";

    auto [resp, err] = client.request(req);
    ASSERT_NIL(err);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    bool found_auth = false;
    for (const auto &[k, v]: reqs[0].headers)
        if (k == "Authorization" && v == "Bearer my-token") found_auth = true;
    EXPECT_TRUE(found_auth);

    server.stop();
}

// ─── Client Auth: API Key ────────────────────────────────────────────────── //

TEST(ClientTest, ApiKeyAuth) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = "GET",
        .path = "/api/keyed",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ConnectionConfig config;
    config.base_url = server.base_url();
    config.auth.type = "api_key";
    config.auth.header = "X-API-Key";
    config.auth.key = "secret123";
    Client client(config);

    Request req;
    req.method = "GET";
    req.path = "/api/keyed";

    auto [resp, err] = client.request(req);
    ASSERT_NIL(err);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    bool found_key = false;
    for (const auto &[k, v]: reqs[0].headers)
        if (k == "X-API-Key" && v == "secret123") found_key = true;
    EXPECT_TRUE(found_key);

    server.stop();
}

// ─── Client Query Params ─────────────────────────────────────────────────── //

TEST(ClientTest, QueryParams) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = "GET",
        .path = "/api/search",
        .status_code = 200,
        .response_body = "found",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ConnectionConfig config;
    config.base_url = server.base_url();
    Client client(config);

    Request req;
    req.method = "GET";
    req.path = "/api/search";
    req.query_params["q"] = "hello";
    req.query_params["limit"] = "10";

    auto [resp, err] = client.request(req);
    ASSERT_NIL(err);
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

// ─── Client Timeout ──────────────────────────────────────────────────────── //

TEST(ClientTest, TimeoutError) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = "GET",
        .path = "/api/slow",
        .status_code = 200,
        .response_body = "delayed",
        .content_type = "text/plain",
        .delay = std::chrono::milliseconds(2000),
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ConnectionConfig config;
    config.base_url = server.base_url();
    config.timeout_ms = 500;
    Client client(config);

    Request req;
    req.method = "GET";
    req.path = "/api/slow";

    auto [resp, err] = client.request(req);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(TIMEOUT_ERR));

    server.stop();
}

// ─── Client Unreachable ──────────────────────────────────────────────────── //

TEST(ClientTest, UnreachableError) {
    ConnectionConfig config;
    config.base_url = "http://192.0.2.1:1";
    config.timeout_ms = 1000;
    Client client(config);

    Request req;
    req.method = "GET";
    req.path = "/";

    auto [resp, err] = client.request(req);
    EXPECT_TRUE(err);
}

// ─── Client 4xx/5xx returns response, not error ──────────────────────────── //

TEST(ClientTest, ErrorStatusCodesReturnResponse) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {
            .method = "GET",
            .path = "/api/notfound",
            .status_code = 404,
            .response_body = R"({"error": "not found"})",
        },
        {
            .method = "GET",
            .path = "/api/error",
            .status_code = 500,
            .response_body = R"({"error": "internal"})",
        },
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ConnectionConfig config;
    config.base_url = server.base_url();
    Client client(config);

    {
        Request req;
        req.method = "GET";
        req.path = "/api/notfound";
        auto [resp, err] = client.request(req);
        ASSERT_NIL(err);
        EXPECT_EQ(resp.status_code, 404);
    }

    {
        Request req;
        req.method = "GET";
        req.path = "/api/error";
        auto [resp, err] = client.request(req);
        ASSERT_NIL(err);
        EXPECT_EQ(resp.status_code, 500);
    }

    server.stop();
}

// ─── Parallel Requests ───────────────────────────────────────────────────── //

TEST(ClientTest, ParallelRequests) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = "GET", .path = "/api/a", .status_code = 200, .response_body = "A"},
        {.method = "GET", .path = "/api/b", .status_code = 200, .response_body = "B"},
        {.method = "GET", .path = "/api/c", .status_code = 200, .response_body = "C"},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ConnectionConfig config;
    config.base_url = server.base_url();
    Client client(config);

    std::vector<Request> requests = {
        {.method = "GET", .path = "/api/a"},
        {.method = "GET", .path = "/api/b"},
        {.method = "GET", .path = "/api/c"},
    };

    auto [responses, err] = client.request_parallel(requests);
    ASSERT_NIL(err);
    ASSERT_EQ(responses.size(), 3);
    for (const auto &resp: responses) EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

// ─── Manager ─────────────────────────────────────────────────────────────── //

TEST(ManagerTest, CreatesClient) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = "GET",
        .path = "/ping",
        .status_code = 200,
        .response_body = "pong",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    Manager manager;
    ConnectionConfig config;
    config.base_url = server.base_url();

    auto client = ASSERT_NIL_P(manager.acquire(config));
    EXPECT_NE(client, nullptr);

    Request req;
    req.method = "GET";
    req.path = "/ping";

    auto [resp, err] = client->request(req);
    ASSERT_NIL(err);
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, "pong");

    server.stop();
}

TEST(ManagerTest, ReusesSameBaseUrl) {
    Manager manager;
    ConnectionConfig config;
    config.base_url = "http://127.0.0.1:9999";

    auto c1 = ASSERT_NIL_P(manager.acquire(config));
    auto c2 = ASSERT_NIL_P(manager.acquire(config));
    EXPECT_EQ(c1.get(), c2.get());
}

TEST(ManagerTest, DifferentBaseUrlsDifferentClients) {
    Manager manager;

    ConnectionConfig config1;
    config1.base_url = "http://127.0.0.1:9991";
    auto c1 = ASSERT_NIL_P(manager.acquire(config1));

    ConnectionConfig config2;
    config2.base_url = "http://127.0.0.1:9992";
    auto c2 = ASSERT_NIL_P(manager.acquire(config2));

    EXPECT_NE(c1.get(), c2.get());
}

TEST(ManagerTest, ExpiredClientRecreated) {
    Manager manager;
    ConnectionConfig config;
    config.base_url = "http://127.0.0.1:9993";

    {
        auto c1 = ASSERT_NIL_P(manager.acquire(config));
        EXPECT_NE(c1, nullptr);
    }

    // After the shared_ptr expires, acquire should succeed with a new client.
    auto c2 = ASSERT_NIL_P(manager.acquire(config));
    EXPECT_NE(c2, nullptr);
}

TEST(ManagerTest, ConcurrentAcquireIsThreadSafe) {
    Manager manager;
    std::atomic<int> success_count{0};

    std::vector<std::thread> threads;
    for (int i = 0; i < 20; i++) {
        threads.emplace_back([&manager, &success_count, i]() {
            ConnectionConfig config;
            config.base_url =
                "http://127.0.0.1:" + std::to_string(7000 + (i % 4));
            auto [client, err] = manager.acquire(config);
            if (!err && client != nullptr) success_count++;
        });
    }

    for (auto &t: threads) t.join();
    EXPECT_EQ(success_count.load(), 20);
}
}
