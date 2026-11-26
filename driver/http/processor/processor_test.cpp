// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <chrono>
#include <cstddef>
#include <map>
#include <memory>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "gmock/gmock.h"
#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/http/errors/errors.h"
#include "driver/http/processor/processor.h"
#include "driver/http/testutil/server.h"

namespace driver::http {
namespace {
using ::testing::Contains;
using ::testing::Pair;

// Only bounds a hang. A loaded runner can stall a loopback request for seconds, so a
// test that wants a timeout sets a short one itself.
const auto TIMEOUT = 5 * x::telem::SECOND;

/// @brief starts a mock server with routes. The server stops when it is destroyed.
std::unique_ptr<testutil::Server> serve(std::vector<testutil::Route> routes) {
    auto server = std::make_unique<testutil::Server>(
        testutil::ServerConfig{.routes = std::move(routes)}
    );
    EXPECT_NIL(server->start());
    return server;
}

/// @brief returns a GET route on path that answers 200 "ok" after delay.
testutil::Route
route(const std::string &path, const x::telem::TimeSpan delay = x::telem::TimeSpan()) {
    return {
        .path = path,
        .response_body = "ok",
        .content_type = "text/plain",
        .delay = delay,
    };
}

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

Request make_gated_request(
    const std::string &base_url,
    const std::string &path,
    const std::size_t cap
) {
    auto req = make_request(base_url, path);
    req.base_url = base_url;
    req.max_concurrent_requests = cap;
    return req;
}
}

struct MethodCase {
    Method method;
    /// @brief body of the request.
    std::string body;
    /// @brief body that the server receives.
    std::string received;
};

class MethodTest : public ::testing::TestWithParam<MethodCase> {};

/// @brief it should send each method, and a body only with a method that takes one.
TEST_P(MethodTest, SendsMethodAndBody) {
    const auto &[method, body, received] = GetParam();
    const auto server = serve({{
        .method = method,
        .path = "/r",
        .status_code = 201,
        .response_body = "ok",
    }});
    auto req = make_request(server->base_url(), "/r", method);
    req.body = body;

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 201);
    EXPECT_EQ(resp.body, "ok");
    EXPECT_GT(resp.time_range.end, resp.time_range.start);
    const auto reqs = server->received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].method, method);
    EXPECT_EQ(reqs[0].body, received);
}

INSTANTIATE_TEST_SUITE_P(
    Methods,
    MethodTest,
    ::testing::Values(
        MethodCase{Method::GET, "{}", ""},
        MethodCase{Method::DEL, "{}", ""},
        MethodCase{Method::POST, "{}", "{}"},
        MethodCase{Method::POST, "", ""},
        MethodCase{Method::PUT, "{}", "{}"},
        MethodCase{Method::PATCH, "{}", "{}"},
        MethodCase{Method::OPTIONS, "", ""}
    )
);

/// @brief it should send the headers and the URL query as given.
TEST(ProcessorTest, SendsHeadersAndQueryAsGiven) {
    const auto server = serve({route("/r")});
    auto req = make_request(server->base_url(), "/r?key=secret&q=a%20b&tag=a%26b%3Dc");
    req.headers = {"Authorization: Bearer token", "X-API-Key: secret"};

    Processor proc;
    ASSERT_NIL_P(proc.execute(req));
    const auto reqs = server->received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_THAT(reqs[0].headers, Contains(Pair("Authorization", "Bearer token")));
    EXPECT_THAT(reqs[0].headers, Contains(Pair("X-API-Key", "secret")));
    EXPECT_EQ(
        reqs[0].query_params,
        (std::multimap<std::string, std::string>{
            {"key", "secret"},
            {"q", "a b"},
            {"tag", "a&b=c"},
        })
    );
}

/// @brief it should fail a request that runs past its timeout as unreachable.
TEST(ProcessorTest, TimeoutError) {
    const auto server = serve({route("/slow", 2 * x::telem::SECOND)});
    auto req = make_request(server->base_url(), "/slow");
    req.timeout = x::telem::MILLISECOND;

    Processor proc;
    ASSERT_OCCURRED_AS_P(proc.execute(req), errors::UNREACHABLE_ERROR);
}

/// @brief it should fail a request to a host that does not answer as unreachable.
TEST(ProcessorTest, UnreachableError) {
    const Request req{
        .url = "http://192.0.2.1:1/",
        .timeout = 100 * x::telem::MILLISECOND,
    };

    Processor proc;
    ASSERT_OCCURRED_AS_P(proc.execute(req), errors::UNREACHABLE_ERROR);
}

/// @brief it should return the results of a batch in order, and an error status as a
/// response.
TEST(ProcessorTest, BatchReturnsResultsInOrder) {
    const auto server = serve({
        {.path = "/ok", .status_code = 200, .response_body = "A"},
        {.path = "/missing", .status_code = 404, .response_body = "B"},
        {.path = "/error", .status_code = 500, .response_body = "C"},
    });
    const std::vector<Request> reqs = {
        make_request(server->base_url(), "/ok"),
        make_request(server->base_url(), "/missing"),
        make_request(server->base_url(), "/error"),
    };
    const std::vector<std::pair<int, std::string>> expected = {
        {200, "A"},
        {404, "B"},
        {500, "C"},
    };

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), expected.size());
    for (std::size_t i = 0; i < expected.size(); i++) {
        const auto &resp = ASSERT_NIL_P(results[i]);
        EXPECT_EQ(resp.status_code, expected[i].first);
        EXPECT_EQ(resp.body, expected[i].second);
    }
}

/// @brief it should return the other results of a batch when one request times out.
TEST(ProcessorTest, ParallelFirstTimesOutSecondSucceeds) {
    const auto server = serve({route("/slow", 2 * x::telem::SECOND), route("/fast")});
    auto slow = make_request(server->base_url(), "/slow");
    slow.timeout = 100 * x::telem::MILLISECOND;
    const std::vector<Request> reqs = {slow, make_request(server->base_url(), "/fast")};

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);
    const auto &slow_resp = ASSERT_OCCURRED_AS_P(results[0], errors::UNREACHABLE_ERROR);
    EXPECT_EQ(slow_resp.status_code, 0);
    const auto &fast_resp = ASSERT_NIL_P(results[1]);
    EXPECT_EQ(fast_resp.status_code, 200);
    EXPECT_EQ(fast_resp.body, "ok");
}

/// @brief it should stay usable after a server error and send the body of each request.
TEST(ProcessorTest, SerialRequestsAfterServerError) {
    const auto server = serve({{
        .method = Method::POST,
        .path = "/r",
        .status_code = 500,
        .response_body = "broken",
    }});
    auto req = make_request(server->base_url(), "/r", Method::POST);
    const std::vector<std::string> bodies = {"1", "2", "3"};

    Processor proc;
    for (const auto &body: bodies) {
        req.body = body;
        const auto resp = ASSERT_NIL_P(proc.execute(req));
        EXPECT_EQ(resp.status_code, 500);
        EXPECT_EQ(resp.body, "broken");
    }
    const auto received = server->received_requests();
    ASSERT_EQ(received.size(), bodies.size());
    for (std::size_t i = 0; i < bodies.size(); i++)
        EXPECT_EQ(received[i].body, bodies[i]);
}

/// @brief it should refuse a self-signed certificate unless verify_ssl is false.
TEST(ProcessorTest, VerifySSL) {
    testutil::Server server({
        .routes = {route("/r")},
        .secure = true,
        .cert_path = "driver/http/testutil/test_cert.pem",
        .key_path = "driver/http/testutil/test_key.pem",
    });
    ASSERT_NIL(server.start());
    auto req = make_request(server.base_url(), "/r");

    Processor proc;
    req.verify_ssl = true;
    ASSERT_OCCURRED_AS_P(proc.execute(req), errors::CRITICAL_ERROR);
    req.verify_ssl = false;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.body, "ok");
}

/// @brief it should give each response of a batch its own time range.
TEST(ProcessorTest, ParallelPerResponseTimeRanges) {
    const auto server = serve(
        {route("/fast"), route("/slow", 50 * x::telem::MILLISECOND)}
    );
    const std::vector<Request> reqs = {
        make_request(server->base_url(), "/fast"),
        make_request(server->base_url(), "/slow"),
    };

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);
    const auto &fast_resp = ASSERT_NIL_P(results[0]);
    const auto &slow_resp = ASSERT_NIL_P(results[1]);
    // The Processor creates handles one at a time, so the starts differ a little.
    const auto start_diff = slow_resp.time_range.start > fast_resp.time_range.start
                              ? slow_resp.time_range.start - fast_resp.time_range.start
                              : fast_resp.time_range.start - slow_resp.time_range.start;
    EXPECT_LT(start_diff, 10 * x::telem::MILLISECOND);
    EXPECT_GT(slow_resp.time_range.end, fast_resp.time_range.end);
}

/// @brief it should return no results for an empty batch.
TEST(ProcessorTest, EmptyBatchExecute) {
    Processor proc;
    EXPECT_TRUE(proc.execute(std::vector<Request>{}).empty());
}

/// @brief it should keep every byte of a body that spans several reads, null bytes
/// included.
TEST(ProcessorTest, ResponseBodyKeepsEveryByte) {
    // Spans several curl write callbacks (16 KiB each) but fits the 128 KiB loopback
    // send buffer, so the mock server never blocks mid-body on flow control.
    std::string body(std::size_t{64} * 1024, 'X');
    body[100] = '\0';
    body.back() = '\0';
    const auto server = serve({{
        .path = "/r",
        .response_body = body,
        .content_type = "application/octet-stream",
    }});

    Processor proc;
    const auto resp = ASSERT_NIL_P(
        proc.execute(make_request(server->base_url(), "/r"))
    );
    EXPECT_EQ(resp.body, body);
}

/// @brief it should follow redirects within one origin and keep the headers.
TEST(ProcessorTest, RedirectFollowsSameOrigin) {
    const auto server = serve({
        {
            .path = "/hop1",
            .status_code = 301,
            .redirect_to = "/hop2",
        },
        {
            .path = "/hop2",
            .status_code = 302,
            .redirect_to = "/hop3",
        },
        route("/hop3"),
    });
    auto req = make_request(server->base_url(), "/hop1");
    req.headers = {"X-API-Key: secret"};

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.status_code, 200);
    EXPECT_EQ(resp.body, "ok");
    const auto reqs = server->received_requests();
    ASSERT_EQ(reqs.size(), 3);
    for (std::size_t i = 0; i < reqs.size(); i++) {
        EXPECT_EQ(reqs[i].path, "/hop" + std::to_string(i + 1));
        EXPECT_THAT(reqs[i].headers, Contains(Pair("X-API-Key", "secret")));
    }
}

/// @brief it should refuse a redirect to another origin without sending the API key.
TEST(ProcessorTest, RedirectToAnotherOriginDoesNotSendAPIKey) {
    const auto target = serve({route("/steal")});
    // curl follows a Location with a space in it, which a strict URL parse rejects.
    const std::vector<std::string> locations = {"/steal?api_key=secret123", "/a b"};
    std::vector<testutil::Route> routes;
    routes.reserve(locations.size());
    for (std::size_t i = 0; i < locations.size(); i++)
        routes.push_back({
            .path = "/old" + std::to_string(i),
            .status_code = 302,
            .redirect_to = target->base_url() + locations[i],
        });
    const auto server = serve(routes);

    Processor proc;
    for (std::size_t i = 0; i < locations.size(); i++) {
        auto req = make_request(
            server->base_url(),
            "/old" + std::to_string(i) + "?api_key=secret123"
        );
        req.headers.emplace_back("X-API-Key: secret123");
        const auto [resp, err] = proc.execute(req);
        ASSERT_OCCURRED_AS(err, errors::CRITICAL_ERROR);
        EXPECT_EQ(
            err.data,
            "refused redirect to " + target->base_url() +
                ", a different scheme, host, or port than " + server->base_url()
        );
    }
    EXPECT_EQ(server->received_requests().size(), locations.size());
    EXPECT_TRUE(target->received_requests().empty());
}

/// @brief it should serve single and batch calls from many threads at once.
TEST(ProcessorTest, ConcurrentExecuteFromManyThreads) {
    const auto server = serve({
        route("/a"),
        {
            .method = Method::POST,
            .path = "/b",
            .status_code = 201,
            .response_body = "B",
        },
    });
    const auto base_url = server->base_url();

    Processor proc;
    constexpr int num_threads = 6;
    constexpr int iters_per_thread = 10;
    std::atomic<int> success_count{0};
    std::vector<std::thread> threads;
    threads.reserve(num_threads);
    for (int t = 0; t < num_threads; t++) {
        threads.emplace_back([&, t] {
            for (int i = 0; i < iters_per_thread; i++) {
                if (t % 3 == 0) {
                    const std::vector<Request> reqs = {
                        make_request(base_url, "/a"),
                        make_request(base_url, "/a"),
                    };
                    for (const auto &[resp, err]: proc.execute(reqs))
                        if (!err && resp.status_code == 200) success_count++;
                } else if (t % 3 == 1) {
                    auto req = make_request(base_url, "/b", Method::POST);
                    req.body = std::to_string(i);
                    const auto [resp, err] = proc.execute(req);
                    if (!err && resp.status_code == 201) success_count++;
                } else {
                    const auto [resp, err] = proc.execute(make_request(base_url, "/a"));
                    if (!err && resp.status_code == 200) success_count++;
                }
            }
        });
    }
    for (auto &t: threads)
        t.join();
    // Batch threads send 2 requests per iteration, single threads 1.
    EXPECT_EQ(success_count.load(), 80);
}

/// @brief it should send requests over the cap of a base URL one after another, in
/// order.
TEST(ProcessorTest, GateRunsRequestsAboveCapInOrder) {
    const auto delay = 30 * x::telem::MILLISECOND;
    const auto server = serve(
        {route("/a", delay), route("/b", delay), route("/c", delay)}
    );
    const std::vector<Request> reqs = {
        make_gated_request(server->base_url(), "/a", 1),
        make_gated_request(server->base_url(), "/b", 1),
        make_gated_request(server->base_url(), "/c", 1),
    };

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 3);
    for (const auto &result: results)
        ASSERT_NIL(result.second);
    EXPECT_GE(results[1].first.time_range.start, results[0].first.time_range.end);
    EXPECT_GE(results[2].first.time_range.start, results[1].first.time_range.end);
    const auto received = server->received_requests();
    ASSERT_EQ(received.size(), 3);
    EXPECT_EQ(received[0].path, "/a");
    EXPECT_EQ(received[1].path, "/b");
    EXPECT_EQ(received[2].path, "/c");
}

/// @brief it should send up to the cap of a base URL at once.
TEST(ProcessorTest, GateRunsUpToCapInParallel) {
    const auto delay = 50 * x::telem::MILLISECOND;
    const auto server = serve(
        {route("/a", delay), route("/b", delay), route("/c", delay)}
    );
    const std::vector<Request> reqs = {
        make_gated_request(server->base_url(), "/a", 2),
        make_gated_request(server->base_url(), "/b", 2),
        make_gated_request(server->base_url(), "/c", 2),
    };

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 3);
    for (const auto &result: results)
        ASSERT_NIL(result.second);
    const auto &a = results[0].first.time_range;
    const auto &b = results[1].first.time_range;
    const auto &c = results[2].first.time_range;
    EXPECT_LT(b.start, a.end);
    EXPECT_GE(c.start, std::min(a.end, b.end));
}

/// @brief it should keep a separate cap for each base URL.
TEST(ProcessorTest, GateIsPerBaseURL) {
    const auto delay = 50 * x::telem::MILLISECOND;
    const auto server_a = serve({route("/a", delay)});
    const auto server_b = serve({route("/b", delay)});
    const std::vector<Request> reqs = {
        make_gated_request(server_a->base_url(), "/a", 1),
        make_gated_request(server_b->base_url(), "/b", 1),
    };

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);
    ASSERT_NIL(results[0].second);
    ASSERT_NIL(results[1].second);
    EXPECT_LT(results[1].first.time_range.start, results[0].first.time_range.end);
}

/// @brief it should keep the cap of the first request to a base URL.
TEST(ProcessorTest, GateKeepsCapOfFirstRequest) {
    const auto delay = 50 * x::telem::MILLISECOND;
    const auto server = serve(
        {route("/a", delay), route("/b", delay), route("/c", delay)}
    );
    const std::vector<Request> reqs = {
        make_gated_request(server->base_url(), "/a", 1),
        make_gated_request(server->base_url(), "/b", 3),
        make_gated_request(server->base_url(), "/c", 3),
    };

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 3);
    for (const auto &result: results)
        ASSERT_NIL(result.second);
    const auto &a = results[0].first.time_range;
    const auto &b = results[1].first.time_range;
    const auto &c = results[2].first.time_range;
    EXPECT_GE(b.start, a.end);
    EXPECT_GE(c.start, b.end);
}

/// @brief it should fill every slot of a gate that frees in one batch of completions.
TEST(ProcessorTest, GateRefillsEverySlotFreedInOneBatch) {
    const auto delay = 50 * x::telem::MILLISECOND;
    const auto server = serve({
        route("/a", delay),
        route("/b", delay),
        route("/c", delay),
        route("/d", delay),
    });
    const std::vector<Request> reqs = {
        make_gated_request(server->base_url(), "/a", 2),
        make_gated_request(server->base_url(), "/b", 2),
        make_gated_request(server->base_url(), "/c", 2),
        make_gated_request(server->base_url(), "/d", 2),
    };

    Processor proc;
    const auto start = std::chrono::steady_clock::now();
    const auto results = proc.execute(reqs);
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(300));
    ASSERT_EQ(results.size(), 4);
    for (const auto &result: results)
        ASSERT_NIL(result.second);
    const auto first_end = std::min(
        results[0].first.time_range.end,
        results[1].first.time_range.end
    );
    EXPECT_GE(results[2].first.time_range.start, first_end);
    EXPECT_GE(results[3].first.time_range.start, first_end);
}

/// @brief it should skip the waiting requests of a base URL that is unreachable.
TEST(ProcessorTest, UnreachableSkipsWaitingRequests) {
    const auto server = serve(
        {route("/slow", 2 * x::telem::SECOND), route("/a"), route("/b")}
    );
    std::vector<Request> reqs = {
        make_gated_request(server->base_url(), "/slow", 1),
        make_gated_request(server->base_url(), "/a", 1),
        make_gated_request(server->base_url(), "/b", 1),
    };
    reqs[0].timeout = 100 * x::telem::MILLISECOND;

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 3);
    ASSERT_OCCURRED_AS_P(results[0], errors::UNREACHABLE_ERROR);
    ASSERT_OCCURRED_AS_P(results[1], errors::SKIPPED_ERROR);
    ASSERT_OCCURRED_AS_P(results[2], errors::SKIPPED_ERROR);
    EXPECT_EQ(server->received_requests().size(), 1);
    const auto resp = ASSERT_NIL_P(
        proc.execute(make_gated_request(server->base_url(), "/a", 1))
    );
    EXPECT_EQ(resp.status_code, 200);
}

/// @brief it should keep the waiting requests when the device answers another request.
TEST(ProcessorTest, UnreachableKeepsWaitingRequestsWhenDeviceResponds) {
    const auto server = serve({
        route("/slow", 2 * x::telem::SECOND),
        route("/fast", 50 * x::telem::MILLISECOND),
    });
    auto slow = make_gated_request(server->base_url(), "/slow", 2);
    slow.timeout = 300 * x::telem::MILLISECOND;
    std::vector<Request> reqs = {slow};
    for (int i = 0; i < 10; i++)
        reqs.push_back(make_gated_request(server->base_url(), "/fast", 2));

    Processor proc;
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 11);
    ASSERT_OCCURRED_AS_P(results[0], errors::UNREACHABLE_ERROR);
    for (std::size_t i = 1; i < results.size(); i++) {
        const auto &resp = ASSERT_NIL_P(results[i]);
        EXPECT_EQ(resp.status_code, 200);
    }
    EXPECT_EQ(server->received_requests().size(), 11);
}

/// @brief it should fail the active and the waiting requests when it shuts down.
TEST(ProcessorTest, ShutdownFailsActiveAndWaitingRequests) {
    const auto server = serve({route("/slow", 2 * x::telem::SECOND)});
    const std::vector<Request> reqs(
        3,
        make_gated_request(server->base_url(), "/slow", 1)
    );

    auto proc = std::make_unique<Processor>();
    std::vector<std::pair<Response, x::errors::Error>> results;
    std::thread caller([&, p = proc.get()] { results = p->execute(reqs); });
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        server->received_requests().size(),
        1,
        std::chrono::seconds(10),
        std::chrono::milliseconds(1)
    );
    // The wait only records a failure, so keep proc alive until the caller ends.
    if (HasFailure()) {
        caller.join();
        return;
    }
    proc.reset();
    caller.join();

    ASSERT_EQ(results.size(), 3);
    for (const auto &result: results)
        ASSERT_OCCURRED_AS_P(result, errors::CRITICAL_ERROR);
}

/// @brief it should end the poll wait when it shuts down.
TEST(ProcessorTest, ShutdownWakesActivePoll) {
    const auto server = serve({route("/slow", 500 * x::telem::MILLISECOND)});
    const auto req = make_request(server->base_url(), "/slow");

    auto proc = std::make_unique<Processor>(1 * x::telem::SECOND);
    std::pair<Response, x::errors::Error> result;
    std::thread caller([&, p = proc.get()] { result = p->execute(req); });
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        server->received_requests().size(),
        1,
        std::chrono::seconds(10),
        std::chrono::milliseconds(1)
    );
    // The wait only records a failure, so keep proc alive until the caller ends.
    if (HasFailure()) {
        caller.join();
        return;
    }
    const auto start = std::chrono::steady_clock::now();
    proc.reset();
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(300));

    caller.join();
    ASSERT_OCCURRED_AS_P(result, errors::CRITICAL_ERROR);
}

/// @brief it should time out a request without waiting for the poll timeout.
TEST(ProcessorTest, PollTimeoutDoesNotDelayRequestTimeout) {
    const auto server = serve({route("/slow", 500 * x::telem::MILLISECOND)});
    auto req = make_request(server->base_url(), "/slow");
    req.timeout = 100 * x::telem::MILLISECOND;

    Processor proc(1 * x::telem::SECOND);
    const auto start = std::chrono::steady_clock::now();
    ASSERT_OCCURRED_AS_P(proc.execute(req), errors::UNREACHABLE_ERROR);
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(500));
}

/// @brief it should return a response without waiting for the poll timeout.
TEST(ProcessorTest, PollTimeoutDoesNotDelayCompletion) {
    const auto server = serve({route("/fast", 50 * x::telem::MILLISECOND)});

    Processor proc(1 * x::telem::SECOND);
    const auto start = std::chrono::steady_clock::now();
    ASSERT_NIL_P(proc.execute(make_request(server->base_url(), "/fast")));
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(500));
}

/// @brief it should send a new request while another is in flight without waiting for
/// the poll timeout.
TEST(ProcessorTest, PollTimeoutDoesNotDelaySubmissionWhileActive) {
    const auto server = serve(
        {route("/slow", 500 * x::telem::MILLISECOND), route("/fast")}
    );
    const auto slow = make_request(server->base_url(), "/slow");

    Processor proc(1 * x::telem::SECOND);
    std::pair<Response, x::errors::Error> slow_result;
    std::thread caller([&] { slow_result = proc.execute(slow); });
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    const auto start = std::chrono::steady_clock::now();
    ASSERT_NIL_P(proc.execute(make_request(server->base_url(), "/fast")));
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(300));

    caller.join();
    ASSERT_NIL(slow_result.second);
}

/// @brief it should send the next request of a gate without waiting for the poll
/// timeout.
TEST(ProcessorTest, PollTimeoutDoesNotDelayGateDispatch) {
    const auto delay = 50 * x::telem::MILLISECOND;
    const auto server = serve({route("/a", delay), route("/b", delay)});
    const std::vector<Request> reqs = {
        make_gated_request(server->base_url(), "/a", 1),
        make_gated_request(server->base_url(), "/b", 1),
    };

    Processor proc(1 * x::telem::SECOND);
    const auto start = std::chrono::steady_clock::now();
    const auto results = proc.execute(reqs);
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(500));
    ASSERT_EQ(results.size(), 2);
    for (const auto &result: results)
        ASSERT_NIL(result.second);
    EXPECT_GE(results[1].first.time_range.start, results[0].first.time_range.end);
}

/// @brief it should skip the waiting requests without waiting for the poll timeout.
TEST(ProcessorTest, PollTimeoutDoesNotDelaySkip) {
    const auto server = serve({route("/slow", 500 * x::telem::MILLISECOND)});
    std::vector<Request> reqs(2, make_gated_request(server->base_url(), "/slow", 1));
    reqs[0].timeout = 100 * x::telem::MILLISECOND;

    Processor proc(1 * x::telem::SECOND);
    const auto start = std::chrono::steady_clock::now();
    const auto results = proc.execute(reqs);
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(500));
    ASSERT_EQ(results.size(), 2);
    ASSERT_OCCURRED_AS_P(results[0], errors::UNREACHABLE_ERROR);
    ASSERT_OCCURRED_AS_P(results[1], errors::SKIPPED_ERROR);
}

/// @brief it should return the response of a fast server before a slow one ends.
TEST(ProcessorTest, PollTimeoutDoesNotDelayFasterServer) {
    const auto slow_server = serve({route("/slow", 500 * x::telem::MILLISECOND)});
    const auto fast_server = serve({route("/fast")});
    const std::vector<Request> reqs = {
        make_request(slow_server->base_url(), "/slow"),
        make_request(fast_server->base_url(), "/fast"),
    };

    Processor proc(1 * x::telem::SECOND);
    const auto start = x::telem::TimeStamp::now();
    const auto results = proc.execute(reqs);
    ASSERT_EQ(results.size(), 2);
    for (const auto &result: results)
        ASSERT_NIL(result.second);
    EXPECT_LT(results[1].first.time_range.end - start, 300 * x::telem::MILLISECOND);
    EXPECT_GE(results[0].first.time_range.end - start, 500 * x::telem::MILLISECOND);
}

/// @brief it should wake the idle event loop when a request comes in.
TEST(ProcessorTest, SubmissionWakesIdleLoop) {
    const auto server = serve({route("/fast")});
    const auto req = make_request(server->base_url(), "/fast");

    Processor proc(1 * x::telem::SECOND);
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    const auto start = std::chrono::steady_clock::now();
    for (int i = 0; i < 5; i++)
        ASSERT_NIL_P(proc.execute(req));
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(500));
}

/// @brief it should fail a body larger than max_response_size, chunked or not.
TEST(ProcessorTest, ResponseLargerThanMaxResponseSizeErrors) {
    const std::string body(2048, 'a');
    const auto server = serve({
        {
            .path = "/sized",
            .response_body = body,
        },
        {
            .path = "/chunked",
            .response_body = body,
            .body_delay = x::telem::MILLISECOND,
        },
    });

    Processor proc;
    for (const auto *path: {"/sized", "/chunked"}) {
        auto req = make_request(server->base_url(), path);
        req.max_response_size = 1024;
        const auto [resp, err] = proc.execute(req);
        ASSERT_OCCURRED_AS(err, errors::CRITICAL_ERROR);
        EXPECT_EQ(
            err.data,
            "response body is larger than the max_response_size of the device"
        );
    }
}

/// @brief it should accept a body exactly max_response_size bytes long.
TEST(ProcessorTest, ResponseAtMaxResponseSizeSucceeds) {
    const auto server = serve({{
        .path = "/exact",
        .response_body = std::string(1024, 'a'),
    }});
    auto req = make_request(server->base_url(), "/exact");
    req.max_response_size = 1024;

    Processor proc;
    const auto resp = ASSERT_NIL_P(proc.execute(req));
    EXPECT_EQ(resp.body.size(), 1024);
}
}
