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

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/http/mock/server.h"

using namespace driver::http;


TEST(MockServerTest, ServesGetRoute) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::GET,
        .path = "/ping",
        .status_code = 200,
        .response_body = "pong",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());
    EXPECT_TRUE(server.base_url().find("http://") == 0);


    httplib::Client client(server.base_url());
    auto res = client.Get("/ping");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 200);
    EXPECT_EQ(res->body, "pong");

    server.stop();
}

TEST(MockServerTest, ServesPostRoute) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::POST,
        .path = "/submit",
        .status_code = 201,
        .response_body = R"({"id": 1})",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    auto res = cli.Post("/submit", R"({"name": "test"})", "application/json");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 201);
    EXPECT_EQ(res->body, R"({"id": 1})");

    server.stop();
}

TEST(MockServerTest, ServesPutRoute) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::PUT,
        .path = "/update",
        .status_code = 200,
        .response_body = "updated",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    auto res = cli.Put("/update", "{}", "application/json");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 200);
    EXPECT_EQ(res->body, "updated");

    server.stop();
}

TEST(MockServerTest, ServesDeleteRoute) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::DELETE,
        .path = "/remove",
        .status_code = 204,
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    auto res = cli.Delete("/remove");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 204);

    server.stop();
}

TEST(MockServerTest, ServesPatchRoute) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::PATCH,
        .path = "/patch",
        .status_code = 200,
        .response_body = "patched",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    auto res = cli.Patch("/patch", "{}", "application/json");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 200);
    EXPECT_EQ(res->body, "patched");

    server.stop();
}

TEST(MockServerTest, MultipleRoutes) {
    mock::ServerConfig cfg;
    cfg.routes = {
        {.method = Method::GET, .path = "/a", .response_body = "A"},
        {.method = Method::GET, .path = "/b", .response_body = "B"},
        {.method = Method::POST, .path = "/c", .response_body = "C"},
    };
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());

    auto a = cli.Get("/a");
    ASSERT_NE(a, nullptr);
    EXPECT_EQ(a->body, "A");

    auto b = cli.Get("/b");
    ASSERT_NE(b, nullptr);
    EXPECT_EQ(b->body, "B");

    auto c = cli.Post("/c", "", "text/plain");
    ASSERT_NE(c, nullptr);
    EXPECT_EQ(c->body, "C");

    server.stop();
}

TEST(MockServerTest, CustomStatusCode) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::GET,
        .path = "/error",
        .status_code = 503,
        .response_body = "service unavailable",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    auto res = cli.Get("/error");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 503);
    EXPECT_EQ(res->body, "service unavailable");

    server.stop();
}

TEST(MockServerTest, ResponseDelay) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::GET,
        .path = "/slow",
        .response_body = "delayed",
        .content_type = "text/plain",
        .delay = std::chrono::milliseconds(200),
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    const auto before = std::chrono::steady_clock::now();
    auto res = cli.Get("/slow");
    const auto elapsed = std::chrono::steady_clock::now() - before;

    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->body, "delayed");
    EXPECT_GE(
        std::chrono::duration_cast<std::chrono::milliseconds>(elapsed).count(),
        150
    );

    server.stop();
}

// ─── Request Logging ────────────────────────────────────────────────────── //

TEST(MockServerTest, LogsReceivedRequests) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::POST,
        .path = "/log",
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    cli.Post("/log", "hello", "text/plain");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].method, Method::POST);
    EXPECT_EQ(reqs[0].path, "/log");
    EXPECT_EQ(reqs[0].body, "hello");

    server.stop();
}

TEST(MockServerTest, ClearRequests) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::GET,
        .path = "/hit",
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    cli.Get("/hit");
    ASSERT_EQ(server.received_requests().size(), 1);

    server.clear_requests();
    EXPECT_EQ(server.received_requests().size(), 0);

    cli.Get("/hit");
    EXPECT_EQ(server.received_requests().size(), 1);

    server.stop();
}

// ─── Base URL ───────────────────────────────────────────────────────────── //

TEST(MockServerTest, BaseURLUsesHTTPScheme) {
    mock::ServerConfig cfg;
    cfg.routes = {{.path = "/x", .response_body = "x"}};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    EXPECT_TRUE(server.base_url().find("http://") == 0);
    EXPECT_TRUE(server.base_url().find("https://") == std::string::npos);

    server.stop();
}

TEST(MockServerTest, BaseURLUsesHTTPSScheme) {
    mock::ServerConfig cfg;
    cfg.secure = true;
    cfg.cert_path = "driver/http/mock/test_cert.pem";
    cfg.key_path = "driver/http/mock/test_key.pem";
    cfg.routes = {{.path = "/x", .response_body = "x"}};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    EXPECT_TRUE(server.base_url().find("https://") == 0);

    server.stop();
}

// ─── HTTPS ──────────────────────────────────────────────────────────────── //

TEST(MockServerTest, SecureServesGetRoute) {
    mock::ServerConfig cfg;
    cfg.secure = true;
    cfg.cert_path = "driver/http/mock/test_cert.pem";
    cfg.key_path = "driver/http/mock/test_key.pem";
    cfg.routes = {{
        .method = Method::GET,
        .path = "/secure",
        .status_code = 200,
        .response_body = "secure-ok",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());
    EXPECT_TRUE(server.base_url().find("https://") == 0);


    httplib::Client cli(server.base_url());
    cli.enable_server_certificate_verification(false);
    auto res = cli.Get("/secure");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 200);
    EXPECT_EQ(res->body, "secure-ok");

    server.stop();
}

TEST(MockServerTest, SecureServesPostRoute) {
    mock::ServerConfig cfg;
    cfg.secure = true;
    cfg.cert_path = "driver/http/mock/test_cert.pem";
    cfg.key_path = "driver/http/mock/test_key.pem";
    cfg.routes = {{
        .method = Method::POST,
        .path = "/secure-post",
        .status_code = 201,
        .response_body = R"({"ok": true})",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    cli.enable_server_certificate_verification(false);
    auto res = cli.Post(
        "/secure-post", R"({"data": 42})", "application/json"
    );
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 201);
    EXPECT_EQ(res->body, R"({"ok": true})");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, R"({"data": 42})");

    server.stop();
}

TEST(MockServerTest, SecureLogsRequests) {
    mock::ServerConfig cfg;
    cfg.secure = true;
    cfg.cert_path = "driver/http/mock/test_cert.pem";
    cfg.key_path = "driver/http/mock/test_key.pem";
    cfg.routes = {{
        .method = Method::GET,
        .path = "/log-secure",
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());


    httplib::Client cli(server.base_url());
    cli.enable_server_certificate_verification(false);
    cli.Get("/log-secure");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].method, Method::GET);
    EXPECT_EQ(reqs[0].path, "/log-secure");

    server.stop();
}

TEST(MockServerTest, SecureInvalidCertFailsStart) {
    mock::ServerConfig cfg;
    cfg.secure = true;
    cfg.cert_path = "nonexistent_cert.pem";
    cfg.key_path = "nonexistent_key.pem";
    cfg.routes = {{.path = "/x", .response_body = "x"}};
    mock::Server server(cfg);
    EXPECT_TRUE(server.start());
}
