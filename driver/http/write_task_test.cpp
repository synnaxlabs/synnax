// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <chrono>
#include <limits>
#include <map>
#include <memory>
#include <span>
#include <string>
#include <variant>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/icd/types.gen.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/errors/errors.h"
#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/testutil/server.h"
#include "driver/http/write_task.h"
#include "driver/icd/writer.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::http {
using ::synnax::http::WriteEndpoint;

namespace {
/// @brief the ICD of each endpoint that endpoint returns. Its one message, Command,
/// holds the float field value at /value.
const synnax::icd::ICD &command_icd() {
    static const synnax::icd::ICD ic = [] {
        synnax::icd::FloatJSONField value;
        value.key = x::uuid::create();
        value.name = "value";
        value.pointer = "/value";
        synnax::icd::Message m;
        m.key = x::uuid::create();
        m.name = "Command";
        m.payload = synnax::icd::JSONPayload{.fields = {value}};
        synnax::icd::ICD ic;
        ic.key = x::uuid::create();
        ic.name = "Plant";
        ic.messages = {m};
        return ic;
    }();
    return ic;
}

/// @returns an endpoint that sends the commands of float64 channel ch as the message
/// of command_icd.
WriteEndpoint endpoint(
    const std::string &method,
    const std::string &path,
    const synnax::channel::Key ch
) {
    const auto &m = command_icd().messages[0];
    synnax::icd::ChannelWriteBinding b;
    b.field = synnax::icd::base(std::get<synnax::icd::JSONPayload>(m.payload).fields[0])
                  .key;
    b.channel = ch;
    WriteEndpoint ep;
    ep.key = path;
    ep.method = method;
    ep.path = path;
    ep.icd = command_icd().key;
    ep.message = synnax::icd::WriteMessage{.message = m.key, .bindings = {b}};
    return ep;
}

/// @brief helper to build a WriteTaskSink for the enabled endpoints of cfg, which
/// endpoint returns, and a mock server URL.
std::pair<std::unique_ptr<WriteTaskSink>, std::shared_ptr<Processor>>
make_sink(WriteTaskConfig &cfg, const std::string &base_url) {
    auto conn_json = x::json::json{
        {"base_url", base_url},
        {"timeout_ms", 1000},
        {"verify_ssl", false},
    };
    auto conn_parser = x::json::Parser(conn_json);
    auto conn = device::ConnectionConfig(conn_parser);

    std::vector<synnax::channel::Key> cmd_keys;
    std::vector<Request> requests;
    std::vector<driver::icd::Writer> writers;
    for (const auto &ep: cfg.endpoints) {
        if (ep.disabled) continue;
        auto req_cfg = request_config(ep);
        req_cfg.request_content_type = "application/json";
        requests.push_back(device::build_request(conn, req_cfg));
        const auto ch = std::get<synnax::icd::ChannelWriteBinding>(
                            ep.message->bindings[0]
        )
                            .channel;
        cmd_keys.push_back(ch);
        std::map<synnax::channel::Key, synnax::channel::Channel> channels{
            {ch, synnax::channel::Channel{.key = ch, .data_type = x::telem::FLOAT64_T}},
        };
        writers.push_back(ASSERT_NIL_P(
            driver::icd::Writer::create(*ep.message, command_icd(), channels)
        ));
    }

    auto processor = std::make_shared<Processor>();
    return {
        std::make_unique<WriteTaskSink>(
            std::move(cmd_keys),
            processor,
            std::move(requests),
            std::move(writers)
        ),
        std::move(processor),
    };
}

/// @brief helper to count requests matching a given path.
size_t count_requests_for_path(
    const std::vector<testutil::ReceivedRequest> &reqs,
    const std::string &path
) {
    size_t count = 0;
    for (const auto &r: reqs)
        if (r.path == path) count++;
    return count;
}

/// @brief config for one POST endpoint at /api/control that sends channel 1.
WriteTaskConfig single_endpoint_cfg() {
    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.endpoints = {endpoint("POST", "/api/control", 1)};
    return cfg;
}
}

/// @brief it should fail to parse config when endpoints array is empty.
TEST(HTTPWriteTask, ParseConfigEmptyEndpoints) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints", x::json::json::array()},
    };
    ASSERT_OCCURRED_AS_P(WriteTaskConfig::parse(task), x::errors::VALIDATION);
}

/// @brief it should fail to parse config when all endpoints are disabled.
TEST(HTTPWriteTask, ParseConfigAllEndpointsDisabled) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"disabled", true},
             {"method", "POST"},
             {"path", "/api/data"},
         }}},
    };
    ASSERT_OCCURRED_AS_P(WriteTaskConfig::parse(task), x::errors::VALIDATION);
}

/// @brief it should fail when a header entry is missing the name field.
TEST(HTTPWriteTask, ParseConfigHeaderMissingNameErrors) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"headers", {{{"value", "abc"}}}},
         }}},
    };
    auto [_1, err1] = WriteTaskConfig::parse(task);
    ASSERT_TRUE(err1.matches(x::errors::VALIDATION));
    EXPECT_NE(err1.data.find("name"), std::string::npos);
}

/// @brief it should fail when duplicate header names exist.
TEST(HTTPWriteTask, ParseConfigDuplicateHeaderErrors) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"headers",
              {
                  {{"name", "X-Key"}, {"value", "a"}},
                  {{"name", "X-Key"}, {"value", "b"}},
              }},
         }}},
    };
    auto [_2, err2] = WriteTaskConfig::parse(task);
    ASSERT_TRUE(err2.matches(x::errors::VALIDATION));
    EXPECT_NE(err2.data.find("duplicate header"), std::string::npos);
}

/// @brief a header entry may omit the value field; it defaults to an empty string.
TEST(HTTPWriteTask, ParseConfigHeaderMissingValueDefaultsEmpty) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"headers", {{{"name", "X-Key"}}}},
         }}},
    };
    auto cfg = ASSERT_NIL_P(WriteTaskConfig::parse(task));
    ASSERT_EQ(cfg.endpoints[0].headers.size(), 1);
    EXPECT_EQ(cfg.endpoints[0].headers.at(0).name, "X-Key");
    EXPECT_EQ(cfg.endpoints[0].headers.at(0).value, "");
}

/// @brief it should fail when a query_params entry is missing the parameter field.
TEST(HTTPWriteTask, ParseConfigQueryParamMissingParameterErrors) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"query_params", {{{"value", "10"}}}},
         }}},
    };
    auto [_4, err4] = WriteTaskConfig::parse(task);
    ASSERT_TRUE(err4.matches(x::errors::VALIDATION));
    EXPECT_NE(err4.data.find("parameter"), std::string::npos);
}

/// @brief a query_params entry may omit the value field; it defaults to an empty
/// string.
TEST(HTTPWriteTask, ParseConfigQueryParamMissingValueDefaultsEmpty) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"query_params", {{{"parameter", "limit"}}}},
         }}},
    };
    auto cfg = ASSERT_NIL_P(WriteTaskConfig::parse(task));
    ASSERT_EQ(cfg.endpoints[0].query_params.size(), 1);
    EXPECT_EQ(cfg.endpoints[0].query_params.at(0).parameter, "limit");
    EXPECT_EQ(cfg.endpoints[0].query_params.at(0).value, "");
}

/// @brief it should fail when duplicate query parameter names exist.
TEST(HTTPWriteTask, ParseConfigDuplicateQueryParamErrors) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"query_params",
              {
                  {{"parameter", "key"}, {"value", "a"}},
                  {{"parameter", "key"}, {"value", "b"}},
              }},
         }}},
    };
    auto [_6, err6] = WriteTaskConfig::parse(task);
    ASSERT_TRUE(err6.matches(x::errors::VALIDATION));
    EXPECT_NE(err6.data.find("duplicate query parameter"), std::string::npos);
}

/// @brief it should POST a numeric channel value to the server.
TEST(HTTPWriteTask, POSTNumericValue) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/control",
                .status_code = 200,
                .response_body = R"({"status":"ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep = endpoint("POST", "/api/control", 1);

    cfg.endpoints = {ep};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{42.5}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].method, Method::POST);

    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_NEAR(body["value"].get<double>(), 42.5, 0.001);
}

/// @brief it should return an error on 4xx responses.
TEST(HTTPWriteTask, Error4xxResponse) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/control",
                .status_code = 400,
                .response_body = R"({"error":"bad request"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep = endpoint("POST", "/api/control", 1);

    cfg.endpoints = {ep};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{42.0}));

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_ERROR);
}

/// @brief it should return a temporary error on 5xx responses.
TEST(HTTPWriteTask, Error5xxResponse) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/control",
                .status_code = 500,
                .response_body = R"({"error":"internal error"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep = endpoint("POST", "/api/control", 1);

    cfg.endpoints = {ep};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{42.0}));

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, errors::TEMPORARY_ERROR);
}

/// @brief it should fire multiple endpoints independently.
TEST(HTTPWriteTask, MultipleEndpointsFireIndependently) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/temp",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                },
                {
                    .method = Method::PUT,
                    .path = "/api/pressure",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep1 = endpoint("POST", "/api/temp", 1);

    auto ep2 = endpoint("PUT", "/api/pressure", 2);

    cfg.endpoints = {ep1, ep2};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    // Send command to only the first endpoint.
    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{25.0}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].path, "/api/temp");
}

/// @brief it should send requests to multiple endpoints in parallel when a single
/// frame contains commands for all of them.
TEST(HTTPWriteTask, ParallelMultipleEndpoints) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/temp",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                    .delay = 100 * x::telem::MILLISECOND,
                },
                {
                    .method = Method::POST,
                    .path = "/api/pressure",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                    .delay = 100 * x::telem::MILLISECOND,
                },
                {
                    .method = Method::POST,
                    .path = "/api/humidity",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                    .delay = 100 * x::telem::MILLISECOND,
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep1 = endpoint("POST", "/api/temp", 1);

    auto ep2 = endpoint("POST", "/api/pressure", 2);

    auto ep3 = endpoint("POST", "/api/humidity", 3);

    cfg.endpoints = {ep1, ep2, ep3};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{25.0}));
    frame.emplace(
        synnax::channel::Key(2),
        x::telem::Series(std::vector<double>{101.3})
    );
    frame.emplace(synnax::channel::Key(3), x::telem::Series(std::vector<double>{60.0}));

    // Each endpoint has a 100ms delay. If serial, total >= 300ms. If parallel, ~100ms.
    const auto before = std::chrono::steady_clock::now();
    ASSERT_NIL(sink->write(frame));
    const auto elapsed_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                                std::chrono::steady_clock::now() - before
    )
                                .count();

    // Parallel execution should complete well under 300ms.
    EXPECT_LT(elapsed_ms, 250);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 3);

    // Verify all three endpoint bodies arrived with the correct values.
    std::map<std::string, double> path_values;
    for (const auto &r: reqs) {
        auto body = x::json::json::parse(r.body);
        path_values[r.path] = body["value"].get<double>();
    }
    EXPECT_NEAR(path_values["/api/temp"], 25.0, 0.001);
    EXPECT_NEAR(path_values["/api/pressure"], 101.3, 0.001);
    EXPECT_NEAR(path_values["/api/humidity"], 60.0, 0.001);
}

/// @brief when one endpoint in a parallel batch returns an error, the write should
/// still return that error with the failing endpoint's details.
TEST(HTTPWriteTask, ParallelBatchPartialFailure) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/good",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                },
                {
                    .method = Method::POST,
                    .path = "/api/bad",
                    .status_code = 500,
                    .response_body = R"({"error":"internal"})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep1 = endpoint("POST", "/api/good", 1);

    auto ep2 = endpoint("POST", "/api/bad", 2);

    cfg.endpoints = {ep1, ep2};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector<double>{20.0}));

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, errors::TEMPORARY_ERROR);
    EXPECT_NE(err.data.find("/api/bad"), std::string::npos)
        << "Expected failing endpoint path in error. Got: " << err.data;

    // The good endpoint succeeds on round 1 (progress), so the bad endpoint is retried.
    // Round 2 has only the bad endpoint which fails again (no progress), so the retry
    // stops. Total: good=1, bad=2.
    auto reqs = server.received_requests();
    EXPECT_EQ(count_requests_for_path(reqs, "/api/good"), 1);
    EXPECT_EQ(count_requests_for_path(reqs, "/api/bad"), 2);
}

/// @brief when multiple endpoints in a parallel batch return errors, the combined
/// error should contain details from all failing endpoints.
TEST(HTTPWriteTask, ParallelBatchAllFailures) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/client-err",
                    .status_code = 400,
                    .response_body = R"({"error":"bad request"})",
                },
                {
                    .method = Method::POST,
                    .path = "/api/server-err",
                    .status_code = 500,
                    .response_body = R"({"error":"internal"})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep1 = endpoint("POST", "/api/client-err", 1);

    auto ep2 = endpoint("POST", "/api/server-err", 2);

    cfg.endpoints = {ep1, ep2};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector<double>{20.0}));

    auto err = sink->write(frame);
    // The critical error (400) is the first error type encountered.
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_ERROR);
    EXPECT_NE(err.data.find("/api/client-err"), std::string::npos)
        << "Expected critical endpoint in error. Got: " << err.data;
    EXPECT_NE(err.data.find("/api/server-err"), std::string::npos)
        << "Expected temporary endpoint in error. Got: " << err.data;

    // The critical error on client-err counts as progress, so server-err is retried
    // once. Round 2: server-err fails again (no progress), retry stops.
    auto reqs = server.received_requests();
    EXPECT_EQ(count_requests_for_path(reqs, "/api/client-err"), 1);
    EXPECT_EQ(count_requests_for_path(reqs, "/api/server-err"), 2);
}

/// @brief it should send one request per sample of a series, in sample order.
TEST(HTTPWriteTask, OneRequestPerSampleInOrder) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/control",
                .status_code = 200,
                .response_body = R"({"status":"ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep = endpoint("POST", "/api/control", 1);

    cfg.endpoints = {ep};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::vector<double>{1.0, 2.0, 3.0, 4.0, 5.0})
    );

    ASSERT_NIL(sink->write(frame));

    std::vector<std::string> bodies;
    for (const auto &r: server.received_requests())
        bodies.push_back(r.body);
    EXPECT_EQ(
        bodies,
        (std::vector<std::string>{
            R"({"value":1})",
            R"({"value":2})",
            R"({"value":3})",
            R"({"value":4})",
            R"({"value":5})",
        })
    );
}

/// @brief it should handle sequential writes correctly.
TEST(HTTPWriteTask, SequentialSends) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/control",
                .status_code = 200,
                .response_body = R"({"status":"ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep = endpoint("POST", "/api/control", 1);

    cfg.endpoints = {ep};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    for (int i = 1; i <= 5; i++) {
        x::telem::Frame frame;
        frame.emplace(
            synnax::channel::Key(1),
            x::telem::Series(std::vector<double>{static_cast<double>(i * 10)})
        );
        ASSERT_NIL(sink->write(frame));
    }

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 5);
    for (int i = 0; i < 5; i++) {
        auto body = x::json::json::parse(reqs[i].body);
        EXPECT_NEAR(body["value"].get<double>(), (i + 1) * 10.0, 0.001);
    }
}

/// @brief it should recover from a temporary error and continue writing.
TEST(HTTPWriteTask, RecoverFromError) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/control",
                .status_code = 200,
                .response_body = R"({"status":"ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep = endpoint("POST", "/api/control", 1);

    cfg.endpoints = {ep};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    // First write should succeed.
    x::telem::Frame frame1;
    frame1.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::vector<double>{10.0})
    );
    ASSERT_NIL(sink->write(frame1));

    // Stop the server to simulate an unreachable error.
    server.stop();

    x::telem::Frame frame2;
    frame2.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::vector<double>{20.0})
    );
    auto err = sink->write(frame2);
    ASSERT_OCCURRED_AS(err, errors::TEMPORARY_ERROR);

    // Restart the same server on the same port and verify recovery.
    server.clear_requests();
    ASSERT_NIL(server.start());

    x::telem::Frame frame3;
    frame3.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::vector<double>{30.0})
    );
    ASSERT_NIL(sink->write(frame3));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_NEAR(body["value"].get<double>(), 30.0, 0.001);
}

/// @brief it should send the other endpoints in a frame when one value is NaN.
TEST(HTTPWriteTask, NaNSkipsOnlyItsEndpoint) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/other",
                .status_code = 200,
                .response_body = R"({"status":"ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    auto cfg = single_endpoint_cfg();
    cfg.endpoints.push_back(endpoint("POST", "/api/other", 2));
    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::vector{std::numeric_limits<double>::quiet_NaN()})
    );
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector{2.5}));
    ASSERT_NIL(sink->write(frame));
    EXPECT_EQ(
        sink->take_warning().description,
        "Dropped a command for POST " + server.base_url() +
            "/api/control: message Command: field value: NaN is not a JSON number."
    );
    const auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].path, "/api/other");
}

/// @brief it should report a NaN command as a task warning and keep running.
TEST(HTTPWriteTask, NaNReportsWarningAndKeepsRunning) {
    x::breaker::Breaker cancel;
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/control",
                .status_code = 200,
                .response_body = R"({"status":"ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    auto cfg = single_endpoint_cfg();
    auto [sink, processor] = make_sink(cfg, server.base_url());

    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame bad;
    bad.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::vector{std::numeric_limits<double>::quiet_NaN()})
    );
    reads->push_back(std::move(bad));
    x::telem::Frame good;
    good.emplace(synnax::channel::Key(1), x::telem::Series(std::vector{3.5}));
    reads->push_back(std::move(good));

    synnax::task::Task task;
    task.key = x::uuid::create();
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    // A breaker wait would outlast the timeout below, so the test fails if the skipped
    // command engages the breaker.
    common::WriteTask write_task(
        task,
        ctx,
        x::breaker::Config{.name = "test", .base_interval = 30 * x::telem::SECOND},
        std::move(sink),
        std::make_shared<pipeline::testutil::WriterFactory>(),
        pipeline::testutil::simple_streamer_factory({1}, reads)
    );
    ASSERT_TRUE(write_task.start("start", cancel));

    ASSERT_EVENTUALLY_EQ_F_WITH_TIMEOUT(
        [&] { return server.received_requests().size(); },
        size_t(1),
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    EXPECT_EQ(x::json::json::parse(server.received_requests()[0].body)["value"], 3.5);
    const auto statuses = ctx->snapshot();
    const auto warning = std::ranges::find_if(statuses, [](const auto &s) {
        return s.variant == synnax::status::VARIANT_WARNING;
    });
    ASSERT_NE(warning, statuses.end());
    EXPECT_EQ(warning->message, "Skipped invalid command");
    EXPECT_EQ(
        warning->description,
        "Dropped a command for POST " + server.base_url() +
            "/api/control: message Command: field value: NaN is not a JSON number."
    );
    ASSERT_EVENTUALLY_EQ_F_WITH_TIMEOUT(
        [&] {
            const auto statuses = ctx->snapshot();
            return statuses.empty() ? "" : statuses.back().message;
        },
        std::string("Task running"),
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    EXPECT_TRUE(ctx->snapshot().back().details.running);
    ASSERT_TRUE(write_task.stop("stop", true));
}

/// @brief it should skip disabled endpoints and only send to enabled ones.
TEST(HTTPWriteTask, DisabledEndpointSkipped) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/active",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                },
                {
                    .method = Method::POST,
                    .path = "/api/inactive",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep1 = endpoint("POST", "/api/active", 1);
    auto ep2 = endpoint("POST", "/api/inactive", 2);
    ep2.disabled = true;

    cfg.endpoints = {ep1, ep2};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    // Send commands for both channels.
    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector<double>{20.0}));

    ASSERT_NIL(sink->write(frame));

    // Only the enabled endpoint should have received a request.
    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].path, "/api/active");
    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_NEAR(body["value"].get<double>(), 10.0, 0.001);
}

/// @brief it should include per-endpoint headers in the HTTP request.
TEST(HTTPWriteTask, EndpointHeaders) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/control",
                .status_code = 200,
                .response_body = R"({"status":"ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep = endpoint("POST", "/api/control", 1);
    ep.headers = std::vector<::synnax::http::Header>{{"X-Custom", "test-val"}};

    cfg.endpoints = {ep};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{42.5}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto hdr = reqs[0].headers.find("X-Custom");
    ASSERT_NE(hdr, reqs[0].headers.end());
    EXPECT_EQ(hdr->second, "test-val");
}

/// @brief it should include per-endpoint query parameters in the HTTP request URL.
TEST(HTTPWriteTask, EndpointQueryParams) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/control",
                .status_code = 200,
                .response_body = R"({"status":"ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep = endpoint("POST", "/api/control", 1);
    ep.query_params = std::vector<::synnax::http::QueryParam>{
        {"key", "abc"},
        {"verbose", "true"}
    };

    cfg.endpoints = {ep};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{42.5}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto key_param = reqs[0].query_params.find("key");
    ASSERT_NE(key_param, reqs[0].query_params.end());
    EXPECT_EQ(key_param->second, "abc");
    auto verbose_param = reqs[0].query_params.find("verbose");
    ASSERT_NE(verbose_param, reqs[0].query_params.end());
    EXPECT_EQ(verbose_param->second, "true");
}

/// @brief when one endpoint always succeeds and another fails then succeeds, only the
/// failing endpoint should be retried.
TEST(HTTPWriteTask, PartialSuccessRetryOnly) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/good",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                },
                {
                    .method = Method::POST,
                    .path = "/api/flaky",
                    .status_code = 500,
                    .response_body = R"({"error":"internal"})",
                    .remaining_failures = 1,
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep1 = endpoint("POST", "/api/good", 1);

    auto ep2 = endpoint("POST", "/api/flaky", 2);

    cfg.endpoints = {ep1, ep2};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector<double>{20.0}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    EXPECT_EQ(count_requests_for_path(reqs, "/api/good"), 1);
    EXPECT_EQ(count_requests_for_path(reqs, "/api/flaky"), 2);
}

/// @brief when all endpoints fail with temporary errors and no progress is made, retry
/// should stop after one round.
TEST(HTTPWriteTask, NoProgressStopsRetry) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/a",
                    .status_code = 500,
                    .response_body = R"({"error":"internal"})",
                },
                {
                    .method = Method::POST,
                    .path = "/api/b",
                    .status_code = 500,
                    .response_body = R"({"error":"internal"})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep1 = endpoint("POST", "/api/a", 1);

    auto ep2 = endpoint("POST", "/api/b", 2);

    cfg.endpoints = {ep1, ep2};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector<double>{20.0}));

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, errors::TEMPORARY_ERROR);

    auto reqs = server.received_requests();
    EXPECT_EQ(reqs.size(), 2);
}

/// @brief progress-driven retry across multiple rounds: C always succeeds, A fails
/// once, B fails twice. Each round makes progress until all resolve.
TEST(HTTPWriteTask, ProgressDrivenMultipleRounds) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/a",
                    .status_code = 500,
                    .response_body = R"({"error":"internal"})",
                    .remaining_failures = 1,
                },
                {
                    .method = Method::POST,
                    .path = "/api/b",
                    .status_code = 500,
                    .response_body = R"({"error":"internal"})",
                    .remaining_failures = 2,
                },
                {
                    .method = Method::POST,
                    .path = "/api/c",
                    .status_code = 200,
                    .response_body = R"({"status":"ok"})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep_a = endpoint("POST", "/api/a", 1);

    auto ep_b = endpoint("POST", "/api/b", 2);

    auto ep_c = endpoint("POST", "/api/c", 3);

    cfg.endpoints = {ep_a, ep_b, ep_c};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector<double>{20.0}));
    frame.emplace(synnax::channel::Key(3), x::telem::Series(std::vector<double>{30.0}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    EXPECT_EQ(count_requests_for_path(reqs, "/api/a"), 2);
    EXPECT_EQ(count_requests_for_path(reqs, "/api/b"), 3);
    EXPECT_EQ(count_requests_for_path(reqs, "/api/c"), 1);
}

/// @brief a critical error (400) on one endpoint should not prevent temporary failures
/// on other endpoints from being retried.
TEST(HTTPWriteTask, CriticalErrorDoesNotPreventTemporaryRetry) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/critical",
                    .status_code = 400,
                    .response_body = R"({"error":"bad request"})",
                },
                {
                    .method = Method::POST,
                    .path = "/api/flaky",
                    .status_code = 500,
                    .response_body = R"({"error":"internal"})",
                    .remaining_failures = 1,
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep1 = endpoint("POST", "/api/critical", 1);

    auto ep2 = endpoint("POST", "/api/flaky", 2);

    cfg.endpoints = {ep1, ep2};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector<double>{20.0}));

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_ERROR);
    EXPECT_NE(err.data.find("/api/critical"), std::string::npos);

    auto reqs = server.received_requests();
    EXPECT_EQ(count_requests_for_path(reqs, "/api/critical"), 1);
    EXPECT_EQ(count_requests_for_path(reqs, "/api/flaky"), 2);
}

/// @brief when all endpoints return critical errors, no retry should occur.
TEST(HTTPWriteTask, AllCriticalNoRetry) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::POST,
                    .path = "/api/a",
                    .status_code = 400,
                    .response_body = R"({"error":"bad request"})",
                },
                {
                    .method = Method::POST,
                    .path = "/api/b",
                    .status_code = 404,
                    .response_body = R"({"error":"not found"})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    WriteTaskConfig cfg;
    cfg.device = "test-device";
    cfg.auto_start = false;

    auto ep1 = endpoint("POST", "/api/a", 1);

    auto ep2 = endpoint("POST", "/api/b", 2);

    cfg.endpoints = {ep1, ep2};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector<double>{20.0}));

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_ERROR);

    auto reqs = server.received_requests();
    EXPECT_EQ(reqs.size(), 2);
}

namespace {
/// @brief creates a virtual float64 command channel.
synnax::channel::Channel create_command_channel(const synnax::Synnax &client) {
    synnax::channel::Channel ch{
        .name = make_unique_channel_name("cmd"),
        .data_type = x::telem::FLOAT64_T,
        .is_virtual = true,
    };
    EXPECT_NIL(client.channels.create(ch));
    return ch;
}

/// @brief creates an ICD whose message Setpoint holds a float field v at /v.
/// @returns a POST endpoint at path that sends the commands of ch as v.
::synnax::http::WriteEndpoint create_icd_endpoint(
    const synnax::Synnax &client,
    const std::string &path,
    const synnax::channel::Channel &ch
) {
    synnax::icd::FloatJSONField v;
    v.key = x::uuid::create();
    v.name = "v";
    v.pointer = "/v";
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = "Setpoint";
    m.payload = synnax::icd::JSONPayload{.fields = {v}};
    synnax::icd::ICD ic;
    ic.key = x::uuid::create();
    ic.name = "Plant";
    ic.messages = {m};
    EXPECT_NIL(client.icds.create(ic));
    synnax::icd::ChannelWriteBinding b;
    b.field = v.key;
    b.channel = ch.key;
    ::synnax::http::WriteEndpoint ep;
    ep.key = path;
    ep.path = path;
    ep.icd = ic.key;
    ep.message = synnax::icd::WriteMessage{.message = m.key, .bindings = {b}};
    return ep;
}

/// @brief returns the payloads that w sends for fr.
std::vector<std::string> sent(driver::icd::Writer &w, const x::telem::Frame &fr) {
    std::vector<std::string> out;
    const auto [error, refused] = w.write(fr, [&](std::span<const std::uint8_t> p) {
        out.emplace_back(p.begin(), p.end());
        return x::errors::NIL;
    });
    EXPECT_NIL(error);
    EXPECT_NIL(refused);
    return out;
}
}

/// @brief it should build the writer of each endpoint from the message of its ICD.
TEST(HTTPWriteTask, BuildWritersSendsEachEndpoint) {
    const auto client = new_test_client();
    const auto first = create_command_channel(client);
    const auto second = create_command_channel(client);
    auto writers = ASSERT_NIL_P(build_writers(
        client,
        {create_icd_endpoint(client, "/a", first),
         create_icd_endpoint(client, "/b", second)}
    ));
    ASSERT_EQ(writers.size(), 2);
    const x::telem::Frame fr(second.key, x::telem::Series(2.5));
    EXPECT_EQ(sent(writers[1], fr), (std::vector<std::string>{R"({"v":2.5})"}));
    EXPECT_TRUE(sent(writers[0], fr).empty());
}

/// @brief it should build no writer for a disabled endpoint and skip its checks.
TEST(HTTPWriteTask, BuildWritersSkipsDisabledEndpoint) {
    const auto client = new_test_client();
    const auto ch = create_command_channel(client);
    auto disabled = create_icd_endpoint(client, "/b", ch);
    disabled.disabled = true;
    disabled.icd = x::uuid::create();
    auto writers = ASSERT_NIL_P(
        build_writers(client, {create_icd_endpoint(client, "/a", ch), disabled})
    );
    ASSERT_EQ(writers.size(), 1);
    EXPECT_EQ(
        sent(writers[0], x::telem::Frame(ch.key, x::telem::Series(1.5))),
        (std::vector<std::string>{R"({"v":1.5})"})
    );
}

/// @brief it should reject an endpoint with no ICD or no message, and name it.
TEST(HTTPWriteTask, BuildWritersRejectsEndpointWithNoMessage) {
    const auto client = new_test_client();
    ::synnax::http::WriteEndpoint no_icd;
    no_icd.path = "/a";
    no_icd.message = synnax::icd::WriteMessage{.message = x::uuid::create()};
    auto no_message = create_icd_endpoint(client, "/a", create_command_channel(client));
    no_message.message.reset();
    for (const auto &ep: {no_icd, no_message}) {
        const auto err = build_writers(client, {ep}).second;
        ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
        EXPECT_EQ(
            err.data,
            "endpoint POST /a sends no message. Open the task and pick a message for "
            "it."
        );
    }
}

/// @brief it should refuse to configure a task with an endpoint that sends no message.
TEST(HTTPWriteTask, ConfigureRejectsEndpointWithNoMessage) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints", {{{"method", "PUT"}, {"path", "/api/setpoint"}}}},
    };
    const auto ctx = std::make_shared<task::MockContext>(
        std::make_shared<synnax::Synnax>(new_test_client())
    );
    const auto err = configure_write(ctx, task, std::make_shared<Processor>()).second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    EXPECT_EQ(
        err.data,
        "endpoint PUT /api/setpoint sends no message. Open the task and pick a "
        "message for it."
    );
}

/// @brief it should reject an endpoint whose ICD does not exist.
TEST(HTTPWriteTask, BuildWritersRejectsMissingICD) {
    const auto client = new_test_client();
    ::synnax::http::WriteEndpoint ep;
    ep.path = "/a";
    ep.icd = x::uuid::create();
    ep.message = synnax::icd::WriteMessage{.message = x::uuid::create()};
    const auto err = build_writers(client, {ep}).second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    EXPECT_EQ(
        err.data,
        "endpoint POST /a: ICD " + ep.icd->to_string() + " does not exist"
    );
}

/// @brief it should name the endpoint in an error that the writer of its message
/// returns, such as for a channel that does not exist.
TEST(HTTPWriteTask, BuildWritersNamesEndpointOfMessageError) {
    const auto client = new_test_client();
    synnax::channel::Channel missing;
    missing.key = 1;
    const auto err = build_writers(client, {create_icd_endpoint(client, "/a", missing)})
                         .second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    EXPECT_EQ(err.data, "endpoint POST /a: message Setpoint: channel 1 does not exist");
}
}
