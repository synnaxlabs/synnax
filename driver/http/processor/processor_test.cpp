// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/base64/base64.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "driver/http/errors/errors.h"
#include "driver/http/mock/server.h"
#include "driver/http/processor/processor.h"

namespace driver::http {
namespace {
const auto TIMEOUT = 100 * x::telem::MILLISECOND;

Request make_request(
    const std::string &base_url,
    const std::string &path = "",
    Method method = Method::GET
) {
    return Request{
        .url = base_url + path,
        .method = method,
        .timeout = TIMEOUT,
    };
}
}

TEST(ProcessorTest, GETRequest) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/data",
        .status_code = 200,
        .response_body = R"({"value": 42})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/data");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, R"({"value": 42})");
    EXPECT_GT(resp.time_range.end, resp.time_range.start);

    server.stop();
}

TEST(ProcessorTest, POSTRequestWithBody) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::POST,
        .path = "/api/submit",
        .status_code = 201,
        .response_body = R"({"id": 1})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/submit", Method::POST);
    req.body = R"({"name": "test"})";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 201);
    EXPECT_GT(resp.time_range.end, resp.time_range.start);
    EXPECT_EQ(resp.body, R"({"id": 1})");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, R"({"name": "test"})");

    server.stop();
}

TEST(ProcessorTest, CustomHeaders) {
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

    auto req = make_request(server.base_url(), "/api/check");
    req.headers["X-Global"] = "global-val";
    req.headers["X-Request"] = "req-val";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    EXPECT_EQ(reqs[0].headers.find("X-Global")->second, "global-val");
    EXPECT_EQ(reqs[0].headers.find("X-Request")->second, "req-val");

    server.stop();
}

TEST(ProcessorTest, BasicAuth) {
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

    auto req = make_request(server.base_url(), "/api/secure");
    req.headers["Authorization"] = "Basic " + x::base64::encode("user:pass");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    const auto expected = "Basic " + x::base64::encode("user:pass");
    EXPECT_EQ(reqs[0].headers.find("Authorization")->second, expected);

    server.stop();
}

TEST(ProcessorTest, BearerAuth) {
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

    auto req = make_request(server.base_url(), "/api/secure");
    req.headers["Authorization"] = "Bearer my-token";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    EXPECT_EQ(reqs[0].headers.find("Authorization")->second, "Bearer my-token");

    server.stop();
}

TEST(ProcessorTest, APIKeyAuthAsHeader) {
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

    auto req = make_request(server.base_url(), "/api/keyed");
    req.headers["X-API-Key"] = "secret123";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].headers.find("X-API-Key")->second, "secret123");

    server.stop();
}

TEST(ProcessorTest, APIKeyAuthAsQueryParam) {
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

    auto req = make_request(server.base_url(), "/api/keyed?api_key=secret123");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].query_params.find("api_key")->second, "secret123");

    server.stop();
}

TEST(ProcessorTest, QueryParams) {
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

    auto req = make_request(server.base_url(), "/api/search?q=hello&limit=10");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

TEST(ProcessorTest, QueryParamsPercentEncoded) {
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

    auto req = make_request(
        server.base_url(),
        "/api/search?q=hello%20world&tag=a%26b%3Dc"
    );

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].query_params.find("q")->second, "hello world");
    EXPECT_EQ(reqs[0].query_params.find("tag")->second, "a&b=c");

    server.stop();
}

TEST(ProcessorTest, MergedQueryParams) {
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

    auto req = make_request(server.base_url(), "/api/data?api_key=secret&limit=10");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].query_params.find("api_key")->second, "secret");
    EXPECT_EQ(reqs[0].query_params.find("limit")->second, "10");

    server.stop();
}

TEST(ProcessorTest, TimeoutError) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/slow",
        .status_code = 200,
        .response_body = "delayed",
        .content_type = "text/plain",
        .delay = 2 * x::telem::SECOND,
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/slow");
    req.timeout = 1 * x::telem::MILLISECOND;

    Processor proc;
    ASSERT_OCCURRED_AS_P(proc.execute(req), errors::UNREACHABLE_ERROR);

    server.stop();
}

TEST(ProcessorTest, UnreachableError) {
    Request req{
        .url = "http://192.0.2.1:1/",
        .method = Method::GET,
        .timeout = TIMEOUT,
    };

    Processor proc;
    ASSERT_OCCURRED_AS_P(proc.execute(req), errors::UNREACHABLE_ERROR);
}

TEST(ProcessorTest, ErrorStatusCodesReturnResponse) {
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

    Processor proc;

    {
        auto req = make_request(server.base_url(), "/api/notfound");
        const auto resp = ASSERT_NIL_P(proc.execute(req));
        EXPECT_EQ(resp.status_code, 404);
        EXPECT_EQ(resp.body, R"({"error": "not found"})");
    }

    {
        auto req = make_request(server.base_url(), "/api/error");
        const auto resp = ASSERT_NIL_P(proc.execute(req));
        EXPECT_EQ(resp.status_code, 500);
        EXPECT_EQ(resp.body, R"({"error": "internal"})");
    }

    server.stop();
}

TEST(ProcessorTest, ParallelRequests) {
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

    std::vector<Request> reqs = {
        make_request(server.base_url(), "/api/a"),
        make_request(server.base_url(), "/api/b"),
        make_request(server.base_url(), "/api/c"),
    };

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 3);
    for (auto &[resp, err]: results) {
        ASSERT_NIL(err);
        EXPECT_EQ(resp.status_code, 200);
    }
    EXPECT_EQ(results[0].first.body, "A");
    EXPECT_EQ(results[1].first.body, "B");
    EXPECT_EQ(results[2].first.body, "C");

    server.stop();
}

TEST(ProcessorTest, ParallelRequestsWithMixedStatusCodes) {
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

    std::vector<Request> reqs = {
        make_request(server.base_url(), "/ok"),
        make_request(server.base_url(), "/not-found"),
        make_request(server.base_url(), "/error"),
    };

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 3);
    for (auto &[resp, err]: results)
        ASSERT_NIL(err);
    EXPECT_EQ(results[0].first.body, "success");
    EXPECT_EQ(results[0].first.status_code, 200);
    EXPECT_EQ(results[1].first.body, R"({"error": "not found"})");
    EXPECT_EQ(results[1].first.status_code, 404);
    EXPECT_EQ(results[2].first.body, R"({"error": "internal"})");
    EXPECT_EQ(results[2].first.status_code, 500);

    server.stop();
}

TEST(ProcessorTest, ParallelOneTimesOut) {
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
         .delay = 2 * x::telem::SECOND},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto fast = make_request(server.base_url(), "/fast");
    fast.timeout = 500 * x::telem::MILLISECOND;
    auto slow = make_request(server.base_url(), "/slow");
    slow.timeout = 500 * x::telem::MILLISECOND;
    std::vector<Request> reqs = {fast, slow};

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);
    const auto &fast_resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(fast_resp.status_code, 200);
    EXPECT_EQ(fast_resp.body, "fast");
    ASSERT_OCCURRED_AS_P(results[1], errors::UNREACHABLE_ERROR);

    server.stop();
}

TEST(ProcessorTest, RepeatedGETRequests) {
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

    auto req = make_request(server.base_url(), "/api/poll");

    Processor proc;
    for (int i = 0; i < 5; i++) {
        const auto resp = ASSERT_NIL_P(proc.execute(req));
        EXPECT_EQ(resp.status_code, 200);
        EXPECT_EQ(resp.body, "ok");
    }

    EXPECT_EQ(server.received_requests().size(), 5);

    server.stop();
}

TEST(ProcessorTest, RepeatedPOSTRequests) {
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

    auto req = make_request(server.base_url(), "/api/send", Method::POST);

    Processor proc;
    for (int i = 0; i < 3; i++) {
        req.body = R"({"i": )" + std::to_string(i) + "}";
        const auto resp = ASSERT_NIL_P(proc.execute(req));
        EXPECT_EQ(resp.status_code, 201);
        EXPECT_EQ(resp.body, "created");
    }

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 3);
    for (int i = 0; i < 3; i++) {
        EXPECT_EQ(reqs[i].body, R"({"i": )" + std::to_string(i) + "}");
    }

    server.stop();
}

TEST(ProcessorTest, DELETERequest) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::DEL,
        .path = "/api/item/42",
        .status_code = 204,
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/item/42", Method::DEL);

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 204);

    server.stop();
}

TEST(ProcessorTest, PUTRequest) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::PUT,
        .path = "/api/item/1",
        .status_code = 200,
        .response_body = R"({"updated": true})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/item/1", Method::PUT);
    req.body = R"({"name": "new"})";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, R"({"name": "new"})");

    server.stop();
}

TEST(ProcessorTest, EmptyPath) {
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

    auto req = make_request(server.base_url(), "/");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, "root");

    server.stop();
}

TEST(ProcessorTest, HTTPSGETRequest) {
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

    auto req = make_request(server.base_url(), "/api/secure");
    req.verify_ssl = false;

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, R"({"secure": true})");

    server.stop();
}

TEST(ProcessorTest, ContentTypeValidation) {
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

    auto req = make_request(server.base_url(), "/api/json");
    req.headers["Accept"] = "application/json";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, R"({"ok": true})");

    server.stop();
}

TEST(ProcessorTest, ContentTypeMismatchStillSucceeds) {
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

    auto req = make_request(server.base_url(), "/api/text");
    req.headers["Accept"] = "application/json";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, "not json");

    server.stop();
}

TEST(ProcessorTest, ContentTypeHeaderSent) {
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

    auto req = make_request(server.base_url(), "/api/xml", Method::POST);
    req.headers["Content-Type"] = "application/xml";
    req.body = "<req/>";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, "<ok/>");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    EXPECT_EQ(reqs[0].headers.find("Content-Type")->second, "application/xml");

    server.stop();
}

TEST(ProcessorTest, AcceptHeaderSent) {
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

    auto req = make_request(server.base_url(), "/api/accept");
    req.headers["Accept"] = "application/json";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    EXPECT_EQ(reqs[0].headers.find("Accept")->second, "application/json");

    server.stop();
}

TEST(ProcessorTest, GETIgnoresRequestBody) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/data",
        .status_code = 200,
        .response_body = R"({"value": 42})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/data");
    req.body = R"({"filter": true})";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, R"({"value": 42})");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_TRUE(reqs[0].body.empty());

    server.stop();
}

TEST(ProcessorTest, ConcurrentExecuteFromMultipleThreads) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/data",
        .status_code = 200,
        .response_body = R"({"ok": true})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    Processor proc;

    constexpr int num_threads = 4;
    constexpr int requests_per_thread = 5;
    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};

    for (int t = 0; t < num_threads; t++) {
        threads.emplace_back([&] {
            for (int i = 0; i < requests_per_thread; i++) {
                auto req = make_request(server.base_url(), "/api/data");
                auto [resp, err] = proc.execute(req);
                if (!err && resp.status_code == 200) success_count++;
            }
        });
    }

    for (auto &t: threads)
        t.join();
    EXPECT_EQ(success_count.load(), num_threads * requests_per_thread);

    server.stop();
}

TEST(ProcessorTest, DestructorCleansUpGracefully) {
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

    auto req = make_request(server.base_url(), "/api/data");

    {
        Processor proc;
        const auto resp = ASSERT_NIL_P(proc.execute(req));
        EXPECT_EQ(resp.status_code, 200);
    }
    // Processor destructor runs here — should not crash or hang.

    server.stop();
}

TEST(ProcessorTest, HTTPSPOSTRequestWithBody) {
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

    auto req = make_request(server.base_url(), "/api/submit", Method::POST);
    req.verify_ssl = false;
    req.body = R"({"name": "test"})";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 201);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, R"({"name": "test"})");

    server.stop();
}

TEST(ProcessorTest, ContentTypeNotCheckedWhenEmpty) {
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

    auto req = make_request(server.base_url(), "/api/any");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

TEST(ProcessorTest, ContentTypeCharsetSuffix) {
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

    auto req = make_request(server.base_url(), "/api/charset");
    req.headers["Accept"] = "application/json";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);

    server.stop();
}

TEST(ProcessorTest, RequestContentTypeOmittedWhenEmpty) {
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

    auto req = make_request(server.base_url(), "/api/raw", Method::POST);
    req.headers["Content-Type"] = "";
    req.body = "data";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);

    EXPECT_EQ(reqs[0].headers.find("Content-Type"), reqs[0].headers.end());

    server.stop();
}

TEST(ProcessorTest, DELETEIgnoresRequestBody) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::DEL,
        .path = "/api/items",
        .status_code = 200,
        .response_body = R"({"deleted": 3})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/items", Method::DEL);
    req.body = R"({"ids": [1, 2, 3]})";

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, R"({"deleted": 3})");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_TRUE(reqs[0].body.empty());

    server.stop();
}

TEST(ProcessorTest, OPTIONSRequest) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::OPTIONS,
        .path = "/api/opts",
        .status_code = 204,
        .response_body = "",
        .content_type = "text/plain",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/opts", Method::OPTIONS);

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 204);
    EXPECT_EQ(resp.body, "");

    server.stop();
}

TEST(ProcessorTest, MixedGETAndPOST) {
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

    auto get_req = make_request(server.base_url(), "/api/read");
    auto post_req = make_request(server.base_url(), "/api/write", Method::POST);
    post_req.body = R"({"val": 1})";
    std::vector<Request> reqs = {get_req, post_req};

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);
    const auto &get_resp = ASSERT_NIL_P(results[0]);
    const auto &post_resp = ASSERT_NIL_P(results[1]);
    EXPECT_EQ(get_resp.status_code, 200);
    EXPECT_EQ(get_resp.body, "read-ok");
    EXPECT_EQ(post_resp.status_code, 201);
    EXPECT_EQ(post_resp.body, "write-ok");

    server.stop();
}

TEST(ProcessorTest, POSTWithEmptyBody) {
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

    auto req = make_request(server.base_url(), "/api/ping", Method::POST);

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, "pong");

    server.stop();
}

TEST(ProcessorTest, SerialGETRequestsReuseHandles) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/data",
        .status_code = 200,
        .response_body = R"({"value": 1})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/data");
    req.headers["X-Header"] = "test-value";

    Processor proc;
    for (int i = 0; i < 3; i++) {
        const auto resp = ASSERT_NIL_P(proc.execute(req));
        EXPECT_EQ(resp.status_code, 200);
        EXPECT_EQ(resp.body, R"({"value": 1})");
    }

    auto reqs = server.received_requests();
    EXPECT_EQ(reqs.size(), 3);
    for (int i = 0; i < 3; i++) {
        EXPECT_EQ(reqs[i].headers.find("X-Header")->second, "test-value");
    }

    server.stop();
}

TEST(ProcessorTest, SerialPOSTRequestsWithDifferentBodies) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::POST,
        .path = "/api/submit",
        .status_code = 201,
        .response_body = R"({"ok": true})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/submit", Method::POST);

    const std::vector<std::string> bodies = {
        R"({"name": "first"})",
        R"({"name": "second"})",
        R"({"name": "third"})",
    };

    Processor proc;
    for (const auto &body: bodies) {
        req.body = body;
        const auto resp = ASSERT_NIL_P(proc.execute(req));
        EXPECT_EQ(resp.status_code, 201);
    }

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 3);
    for (int i = 0; i < 3; i++) {
        EXPECT_EQ(reqs[i].body, bodies[i]);
    }

    server.stop();
}

TEST(ProcessorTest, SerialRequestsRecoverFromServerError) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {
            .method = Method::GET,
            .path = "/api/ok",
            .status_code = 200,
            .response_body = R"({"status": "ok"})",
        },
        {
            .method = Method::GET,
            .path = "/api/fail",
            .status_code = 500,
            .response_body = R"({"error": "internal"})",
        },
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    std::vector<Request> reqs = {
        make_request(server.base_url(), "/api/ok"),
        make_request(server.base_url(), "/api/fail"),
    };

    Processor proc;
    for (int i = 0; i < 3; i++) {
        const auto r1 = proc.execute(reqs);
        ASSERT_EQ(r1.size(), 2);
        const auto &ok_resp = ASSERT_NIL_P(r1[0]);
        const auto &fail_resp = ASSERT_NIL_P(r1[1]);
        EXPECT_EQ(ok_resp.status_code, 200);
        EXPECT_EQ(ok_resp.body, R"({"status": "ok"})");
        EXPECT_EQ(fail_resp.status_code, 500);
        EXPECT_EQ(fail_resp.body, R"({"error": "internal"})");
    }

    auto server_reqs = server.received_requests();
    EXPECT_EQ(server_reqs.size(), 6);

    server.stop();
}

TEST(ProcessorTest, SerialSingleHandleRecoveryFromServerError) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::POST,
        .path = "/api/action",
        .status_code = 500,
        .response_body = R"({"error": "broken"})",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/action", Method::POST);

    Processor proc;
    // Repeated calls to a 500 endpoint should all return valid responses without the
    // processor becoming unusable.
    for (int i = 0; i < 3; i++) {
        req.body = R"({"attempt": )" + std::to_string(i) + "}";
        const auto resp = ASSERT_NIL_P(proc.execute(req));
        EXPECT_EQ(resp.status_code, 500);
        EXPECT_EQ(resp.body, R"({"error": "broken"})");
    }

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 3);
    for (int i = 0; i < 3; i++) {
        EXPECT_EQ(reqs[i].body, R"({"attempt": )" + std::to_string(i) + "}");
    }

    server.stop();
}

TEST(ProcessorTest, ParallelFirstTimesOutSecondSucceeds) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = Method::GET,
         .path = "/slow",
         .status_code = 200,
         .response_body = "slow",
         .delay = 2 * x::telem::SECOND},
        {.method = Method::GET,
         .path = "/fast",
         .status_code = 200,
         .response_body = "fast"},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto slow = make_request(server.base_url(), "/slow");
    slow.timeout = 500 * x::telem::MILLISECOND;
    auto fast = make_request(server.base_url(), "/fast");
    fast.timeout = 500 * x::telem::MILLISECOND;
    std::vector<Request> reqs = {slow, fast};

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);
    const auto &slow_resp = ASSERT_OCCURRED_AS_P(results[0], errors::UNREACHABLE_ERROR);
    EXPECT_EQ(slow_resp.status_code, 0);
    const auto &fast_resp = ASSERT_NIL_P(results[1]);
    EXPECT_EQ(fast_resp.status_code, 200);
    EXPECT_EQ(fast_resp.body, "fast");

    server.stop();
}

TEST(ProcessorTest, ParallelPerResponseTimeRanges) {
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
         .delay = 50 * x::telem::MILLISECOND},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    std::vector<Request> reqs = {
        make_request(server.base_url(), "/fast"),
        make_request(server.base_url(), "/slow"),
    };

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);
    ASSERT_NIL(results[0].second);
    ASSERT_NIL(results[1].second);

    const auto &fast_resp = results[0].first;
    const auto &slow_resp = results[1].first;

    // Both start times should be very close (within 10ms), though not necessarily
    // identical since the Processor creates handles sequentially.
    const auto start_diff = slow_resp.time_range.start > fast_resp.time_range.start
                              ? slow_resp.time_range.start - fast_resp.time_range.start
                              : fast_resp.time_range.start - slow_resp.time_range.start;
    EXPECT_LT(start_diff, 10 * x::telem::MILLISECOND);

    // The slow response should have a later end time than the fast one.
    EXPECT_GT(slow_resp.time_range.end, fast_resp.time_range.end);

    server.stop();
}

TEST(ProcessorTest, EmptyBatchExecute) {
    Processor proc;
    std::vector<Request> reqs;
    const auto results = proc.execute(reqs);
    EXPECT_TRUE(results.empty());
}

TEST(ProcessorTest, LargeResponseBody) {
    const std::string large_body(256 * 1024, 'X');
    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/large",
        .status_code = 200,
        .response_body = large_body,
        .content_type = "application/octet-stream",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/large");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body.size(), large_body.size());
    EXPECT_EQ(resp.body, large_body);

    server.stop();
}

TEST(ProcessorTest, BinaryNullBytesInResponse) {
    std::string binary_body = "before";
    binary_body.push_back('\0');
    binary_body.push_back('\0');
    binary_body += "after";

    mock::ServerConfig server_cfg;
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/binary",
        .status_code = 200,
        .response_body = binary_body,
        .content_type = "application/octet-stream",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/binary");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body.size(), binary_body.size());
    EXPECT_EQ(resp.body, binary_body);

    server.stop();
}

TEST(ProcessorTest, VerifySSLTrueRejectsSelfSigned) {
    mock::ServerConfig server_cfg;
    server_cfg.secure = true;
    server_cfg.cert_path = "driver/http/mock/test_cert.pem";
    server_cfg.key_path = "driver/http/mock/test_key.pem";
    server_cfg.routes = {{
        .method = Method::GET,
        .path = "/api/secure",
        .status_code = 200,
        .response_body = "ok",
    }};
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/api/secure");
    req.verify_ssl = true;

    Processor proc;
    ASSERT_OCCURRED_AS_P(proc.execute(req), errors::CRITICAL_ERROR);

    server.stop();
}

TEST(ProcessorTest, RapidInterleavedBatchAndSingle) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = Method::GET,
         .path = "/api/a",
         .status_code = 200,
         .response_body = "A"},
        {.method = Method::POST,
         .path = "/api/b",
         .status_code = 201,
         .response_body = "B"},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    Processor proc;

    constexpr int num_threads = 6;
    constexpr int iters_per_thread = 10;
    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};

    for (int t = 0; t < num_threads; t++) {
        threads.emplace_back([&, t] {
            for (int i = 0; i < iters_per_thread; i++) {
                if (t % 3 == 0) {
                    // Batch of two GETs.
                    std::vector<Request> reqs = {
                        make_request(server.base_url(), "/api/a"),
                        make_request(server.base_url(), "/api/a"),
                    };
                    auto results = proc.execute(reqs);
                    for (auto &[resp, err]: results) {
                        if (!err && resp.status_code == 200) success_count++;
                    }
                } else if (t % 3 == 1) {
                    // Single POST.
                    auto req = make_request(server.base_url(), "/api/b", Method::POST);
                    req.body = R"({"i": )" + std::to_string(i) + "}";
                    auto [resp, err] = proc.execute(req);
                    if (!err && resp.status_code == 201) success_count++;
                } else {
                    // Single GET.
                    auto req = make_request(server.base_url(), "/api/a");
                    auto [resp, err] = proc.execute(req);
                    if (!err && resp.status_code == 200) success_count++;
                }
            }
        });
    }

    for (auto &t: threads)
        t.join();

    // t%3==0: 2 threads * 10 iters * 2 requests = 40
    // t%3==1: 2 threads * 10 iters * 1 request  = 20
    // t%3==2: 2 threads * 10 iters * 1 request  = 20
    EXPECT_EQ(success_count.load(), 80);

    server.stop();
}

TEST(ProcessorTest, RedirectSingleHop) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
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
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/old");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, "arrived");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 2);
    EXPECT_EQ(reqs[0].path, "/old");
    EXPECT_EQ(reqs[1].path, "/new");

    server.stop();
}

TEST(ProcessorTest, RedirectMultipleHops) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {
            .method = Method::GET,
            .path = "/hop1",
            .status_code = 301,
            .redirect_to = "/hop2",
        },
        {
            .method = Method::GET,
            .path = "/hop2",
            .status_code = 302,
            .redirect_to = "/hop3",
        },
        {
            .method = Method::GET,
            .path = "/hop3",
            .status_code = 200,
            .response_body = "final",
            .content_type = "text/plain",
        },
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    auto req = make_request(server.base_url(), "/hop1");

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, "final");

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 3);
    EXPECT_EQ(reqs[0].path, "/hop1");
    EXPECT_EQ(reqs[1].path, "/hop2");
    EXPECT_EQ(reqs[2].path, "/hop3");

    server.stop();
}

TEST(ProcessorTest, MultipleServersBothSucceed) {
    mock::ServerConfig cfg_a;
    cfg_a.routes = {{
        .method = Method::GET,
        .path = "/api/a",
        .status_code = 200,
        .response_body = R"({"source": "A"})",
    }};
    mock::Server server_a(cfg_a);
    ASSERT_NIL(server_a.start());

    mock::ServerConfig cfg_b;
    cfg_b.routes = {{
        .method = Method::POST,
        .path = "/api/b",
        .status_code = 201,
        .response_body = R"({"source": "B"})",
    }};
    mock::Server server_b(cfg_b);
    ASSERT_NIL(server_b.start());

    std::vector<Request> reqs = {
        make_request(server_a.base_url(), "/api/a"),
        make_request(server_b.base_url(), "/api/b", Method::POST),
    };
    reqs[1].body = R"({"data": 1})";

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);

    const auto &resp_a = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(resp_a.status_code, 200);
    EXPECT_EQ(resp_a.body, R"({"source": "A"})");

    const auto &resp_b = ASSERT_NIL_P(results[1]);
    EXPECT_EQ(resp_b.status_code, 201);
    EXPECT_EQ(resp_b.body, R"({"source": "B"})");

    server_a.stop();
    server_b.stop();
}

TEST(ProcessorTest, MultipleServersOneTimesOut) {
    mock::ServerConfig cfg_fast;
    cfg_fast.routes = {{
        .method = Method::GET,
        .path = "/api/fast",
        .status_code = 200,
        .response_body = "fast",
    }};
    mock::Server server_fast(cfg_fast);
    ASSERT_NIL(server_fast.start());

    mock::ServerConfig cfg_slow;
    cfg_slow.routes = {{
        .method = Method::GET,
        .path = "/api/slow",
        .status_code = 200,
        .response_body = "slow",
        .delay = 2 * x::telem::SECOND,
    }};
    mock::Server server_slow(cfg_slow);
    ASSERT_NIL(server_slow.start());

    auto fast = make_request(server_fast.base_url(), "/api/fast");
    fast.timeout = 500 * x::telem::MILLISECOND;
    auto slow = make_request(server_slow.base_url(), "/api/slow");
    slow.timeout = 500 * x::telem::MILLISECOND;
    std::vector<Request> reqs = {fast, slow};

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);

    const auto &fast_resp = ASSERT_NIL_P(results[0]);
    EXPECT_EQ(fast_resp.status_code, 200);
    EXPECT_EQ(fast_resp.body, "fast");

    ASSERT_OCCURRED_AS_P(results[1], errors::UNREACHABLE_ERROR);

    server_fast.stop();
    server_slow.stop();
}

TEST(ProcessorTest, MultiThreadedParallelBatches) {
    mock::ServerConfig server_cfg;
    server_cfg.routes = {
        {.method = Method::GET,
         .path = "/api/a",
         .status_code = 200,
         .response_body = "A"},
        {.method = Method::POST,
         .path = "/api/b",
         .status_code = 201,
         .response_body = "B"},
    };
    mock::Server server(server_cfg);
    ASSERT_NIL(server.start());

    Processor proc;

    constexpr int num_threads = 4;
    constexpr int iters_per_thread = 3;
    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};

    for (int t = 0; t < num_threads; t++) {
        threads.emplace_back([&, t] {
            for (int i = 0; i < iters_per_thread; i++) {
                if (t % 2 == 0) {
                    // Even threads: batch execute with two GET requests.
                    std::vector<Request> reqs = {
                        make_request(server.base_url(), "/api/a"),
                        make_request(server.base_url(), "/api/a"),
                    };
                    auto results = proc.execute(reqs);
                    for (auto &[resp, err]: results) {
                        if (!err && resp.status_code == 200) success_count++;
                    }
                } else {
                    // Odd threads: single execute with POST body.
                    auto req = make_request(server.base_url(), "/api/b", Method::POST);
                    req.body = R"({"thread": )" + std::to_string(t) + "}";
                    auto [resp, err] = proc.execute(req);
                    if (!err && resp.status_code == 201) success_count++;
                }
            }
        });
    }

    for (auto &t: threads)
        t.join();

    // Even threads: 2 threads * 3 iters * 2 requests = 12.
    // Odd threads: 2 threads * 3 iters * 1 request = 6.
    EXPECT_EQ(success_count.load(), 18);

    server.stop();
}

}
