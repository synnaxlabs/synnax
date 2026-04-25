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
// Disable GCC 13 false positive warning in <regex> header (included by httplib.h)
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wmaybe-uninitialized"
#endif
#include "httplib.h"
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic pop
#endif

using namespace driver::http;

TEST(MockServerTest, ServesGETRoute) {
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

TEST(MockServerTest, ServesPOSTRoute) {
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

TEST(MockServerTest, ServesPUTRoute) {
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

TEST(MockServerTest, ServesDELETERoute) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::DEL,
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

TEST(MockServerTest, ServesPATCHRoute) {
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
        .delay = 200 * x::telem::MILLISECOND,
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

TEST(MockServerTest, LogsQueryParams) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::GET,
        .path = "/search",
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    httplib::Client cli(server.base_url());
    auto res = cli.Get("/search?q=hello%20world&tag=a%26b%3Dc");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 200);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].query_params.find("q")->second, "hello world");
    EXPECT_EQ(reqs[0].query_params.find("tag")->second, "a&b=c");

    server.stop();
}

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

TEST(MockServerTest, SecureServesPOSTRoute) {
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
    auto res = cli.Post("/secure-post", R"({"data": 42})", "application/json");
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

TEST(MockServerTest, ServesOptionsRoute) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::OPTIONS,
        .path = "/api/opts",
        .status_code = 204,
        .response_body = "",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    httplib::Client cli(server.base_url());
    auto res = cli.Options("/api/opts");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 204);

    server.stop();
}

TEST(MockServerTest, HEADRouteRegistrationErrors) {
    mock::ServerConfig cfg;
    cfg.routes = {{.method = Method::HEAD, .path = "/x", .response_body = "x"}};
    EXPECT_THROW(mock::Server server(cfg), std::runtime_error);
}

TEST(MockServerTest, TRACERouteRegistrationErrors) {
    mock::ServerConfig cfg;
    cfg.routes = {{.method = Method::TRACE, .path = "/x", .response_body = "x"}};
    EXPECT_THROW(mock::Server server(cfg), std::runtime_error);
}

TEST(MockServerTest, CONNECTRouteRegistrationErrors) {
    mock::ServerConfig cfg;
    cfg.routes = {{.method = Method::CONNECT, .path = "/x", .response_body = "x"}};
    EXPECT_THROW(mock::Server server(cfg), std::runtime_error);
}

/// @brief it should serve requests after a stop and restart on the same port.
TEST(MockServerTest, RestartReusesSamePort) {
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

    const auto url = server.base_url();
    httplib::Client cli(url);

    auto res1 = cli.Get("/ping");
    ASSERT_NE(res1, nullptr);
    EXPECT_EQ(res1->body, "pong");

    server.stop();

    // Restart — should bind to the same port.
    ASSERT_NIL(server.start());
    EXPECT_EQ(server.base_url(), url);

    // The same client (same URL) should still work.
    httplib::Client cli2(server.base_url());
    auto res2 = cli2.Get("/ping");
    ASSERT_NE(res2, nullptr);
    EXPECT_EQ(res2->body, "pong");

    server.stop();
}

/// @brief it should accumulate requests across restarts unless cleared.
TEST(MockServerTest, RestartAccumulatesRequests) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::POST,
        .path = "/data",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    httplib::Client cli(server.base_url());
    cli.Post("/data", "first", "text/plain");
    ASSERT_EQ(server.received_requests().size(), 1);

    server.stop();
    ASSERT_NIL(server.start());

    httplib::Client cli2(server.base_url());
    cli2.Post("/data", "second", "text/plain");

    // Both requests should be present.
    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 2);
    EXPECT_EQ(reqs[0].body, "first");
    EXPECT_EQ(reqs[1].body, "second");

    server.stop();
}

/// @brief it should clear all recorded requests.
TEST(MockServerTest, ClearRequests) {
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

    httplib::Client cli(server.base_url());
    cli.Get("/ping");
    cli.Get("/ping");
    ASSERT_EQ(server.received_requests().size(), 2);

    server.clear_requests();
    EXPECT_EQ(server.received_requests().size(), 0);

    // New requests should still be recorded after clearing.
    cli.Get("/ping");
    EXPECT_EQ(server.received_requests().size(), 1);

    server.stop();
}

/// @brief it should clear requests across a restart cycle.
TEST(MockServerTest, ClearRequestsAcrossRestart) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::POST,
        .path = "/data",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    httplib::Client cli(server.base_url());
    cli.Post("/data", "before", "text/plain");

    server.stop();
    server.clear_requests();
    ASSERT_NIL(server.start());

    httplib::Client cli2(server.base_url());
    cli2.Post("/data", "after", "text/plain");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, "after");

    server.stop();
}

TEST(MockServerTest, RedirectRoute) {
    mock::ServerConfig cfg;
    cfg.routes = {
        {
            .method = Method::GET,
            .path = "/old",
            .status_code = 302,
            .redirect_to = "/new",
        },
        {
            .method = Method::GET,
            .path = "/new",
            .status_code = 200,
            .response_body = "arrived",
            .content_type = "text/plain",
        },
    };
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    // Without following redirects, the client should see the 302.
    httplib::Client cli(server.base_url());
    cli.set_follow_location(false);
    auto res = cli.Get("/old");
    ASSERT_NE(res, nullptr);
    EXPECT_EQ(res->status, 302);
    EXPECT_EQ(res->get_header_value("Location"), "/new");

    // The redirect endpoint itself should serve normally.
    auto res2 = cli.Get("/new");
    ASSERT_NE(res2, nullptr);
    EXPECT_EQ(res2->status, 200);
    EXPECT_EQ(res2->body, "arrived");

    // Both requests should be logged.
    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 2);
    EXPECT_EQ(reqs[0].path, "/old");
    EXPECT_EQ(reqs[1].path, "/new");

    server.stop();
}

/// @brief remaining_failures should return the error status for the configured number
/// of requests, then switch to 200.
TEST(MockServerTest, RemainingFailuresCountdown) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::POST,
        .path = "/flaky",
        .status_code = 500,
        .response_body = R"({"error":"internal"})",
        .remaining_failures = 2,
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    httplib::Client cli(server.base_url());

    auto res1 = cli.Post("/flaky", "{}", "application/json");
    ASSERT_NE(res1, nullptr);
    EXPECT_EQ(res1->status, 500);

    auto res2 = cli.Post("/flaky", "{}", "application/json");
    ASSERT_NE(res2, nullptr);
    EXPECT_EQ(res2->status, 500);

    auto res3 = cli.Post("/flaky", "{}", "application/json");
    ASSERT_NE(res3, nullptr);
    EXPECT_EQ(res3->status, 200);

    auto res4 = cli.Post("/flaky", "{}", "application/json");
    ASSERT_NE(res4, nullptr);
    EXPECT_EQ(res4->status, 200);

    EXPECT_EQ(server.received_requests().size(), 4);

    server.stop();
}

/// @brief remaining_failures=0 should always use the configured status_code.
TEST(MockServerTest, RemainingFailuresZeroUsesConfiguredStatus) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::GET,
        .path = "/stable",
        .status_code = 503,
        .response_body = "unavailable",
        .content_type = "text/plain",
        .remaining_failures = 0,
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    httplib::Client cli(server.base_url());

    auto res1 = cli.Get("/stable");
    ASSERT_NE(res1, nullptr);
    EXPECT_EQ(res1->status, 503);
    EXPECT_EQ(res1->body, "unavailable");

    auto res2 = cli.Get("/stable");
    ASSERT_NE(res2, nullptr);
    EXPECT_EQ(res2->status, 503);

    server.stop();
}

/// @brief remaining_failures=1 should fail once then succeed on all subsequent
/// requests.
TEST(MockServerTest, RemainingFailuresOneFailsThenSucceeds) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::PUT,
        .path = "/once",
        .status_code = 429,
        .response_body = R"({"error":"rate limited"})",
        .remaining_failures = 1,
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    httplib::Client cli(server.base_url());

    auto res1 = cli.Put("/once", "{}", "application/json");
    ASSERT_NE(res1, nullptr);
    EXPECT_EQ(res1->status, 429);

    auto res2 = cli.Put("/once", "{}", "application/json");
    ASSERT_NE(res2, nullptr);
    EXPECT_EQ(res2->status, 200);

    auto res3 = cli.Put("/once", "{}", "application/json");
    ASSERT_NE(res3, nullptr);
    EXPECT_EQ(res3->status, 200);

    server.stop();
}

/// @brief remaining_failures counters should reset when the server is restarted.
TEST(MockServerTest, RemainingFailuresResetsOnRestart) {
    mock::ServerConfig cfg;
    cfg.routes = {{
        .method = Method::POST,
        .path = "/reset",
        .status_code = 500,
        .response_body = R"({"error":"internal"})",
        .remaining_failures = 1,
    }};
    mock::Server server(cfg);
    ASSERT_NIL(server.start());

    httplib::Client cli(server.base_url());

    auto res1 = cli.Post("/reset", "{}", "application/json");
    ASSERT_NE(res1, nullptr);
    EXPECT_EQ(res1->status, 500);

    auto res2 = cli.Post("/reset", "{}", "application/json");
    ASSERT_NE(res2, nullptr);
    EXPECT_EQ(res2->status, 200);

    server.stop();
    server.clear_requests();
    ASSERT_NIL(server.start());

    httplib::Client cli2(server.base_url());

    auto res3 = cli2.Post("/reset", "{}", "application/json");
    ASSERT_NE(res3, nullptr);
    EXPECT_EQ(res3->status, 500);

    auto res4 = cli2.Post("/reset", "{}", "application/json");
    ASSERT_NE(res4, nullptr);
    EXPECT_EQ(res4->status, 200);

    server.stop();
}
