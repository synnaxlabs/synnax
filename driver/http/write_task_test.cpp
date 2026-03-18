// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <map>
#include <memory>
// Disable GCC 13 false positive warning in <regex> header
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wmaybe-uninitialized"
#endif
#include <regex>
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic pop
#endif
#
#include "gtest/gtest.h"

#include "x/cpp/defer/defer.h"
#include "x/cpp/test/test.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/mock/server.h"
#include "driver/http/write_task.h"

namespace driver::http {
namespace {
/// @brief helper to build a WriteTaskSink from config and a mock server URL.
std::pair<std::unique_ptr<WriteTaskSink>, std::shared_ptr<Processor>>
make_sink(WriteTaskConfig &cfg, const std::string &base_url) {
    auto conn_json = x::json::json{
        {"base_url", base_url},
        {"timeout_ms", 1000},
        {"verify_ssl", false},
    };
    auto conn_parser = x::json::Parser(conn_json);
    auto conn = device::ConnectionConfig(conn_parser);

    std::vector<Request> base_requests;
    base_requests.reserve(cfg.endpoints.size());
    for (const auto &ep: cfg.endpoints) {
        base_requests.push_back(device::build_request(conn, ep.request));
    }

    auto processor = std::make_shared<Processor>();
    return {
        std::make_unique<WriteTaskSink>(
            WriteTaskConfig(cfg),
            processor,
            std::move(base_requests)
        ),
        std::move(processor),
    };
}
}

/// @brief it should fail to parse config when endpoints array is empty.
TEST(HTTPWriteTask, ParseConfigEmptyEndpoints) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints", x::json::json::array()},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(WriteTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should fail to parse config when all endpoints are disabled.
TEST(HTTPWriteTask, ParseConfigAllEndpointsDisabled) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"enabled", false},
             {"method", "POST"},
             {"path", "/api/data"},
             {"channel",
              {{"pointer", "/value"}, {"json_type", "number"}, {"channel", 1}}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(WriteTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should fail when duplicate pointers exist across fields.
TEST(HTTPWriteTask, ParseConfigDuplicatePointers) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"channel",
              {{"pointer", "/value"}, {"json_type", "number"}, {"channel", 1}}},
             {"fields",
              {{
                  {"type", "static"},
                  {"pointer", "/value"},
                  {"json_type", "number"},
                  {"value", 42},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(WriteTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should fail when bare primitive has additional fields.
TEST(HTTPWriteTask, ParseConfigBarePrimitiveWithAdditionalFields) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"channel", {{"pointer", ""}, {"json_type", "number"}, {"channel", 1}}},
             {"fields",
              {{
                  {"type", "static"},
                  {"pointer", "/extra"},
                  {"json_type", "number"},
                  {"value", 42},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(WriteTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should fail when a static field has an empty pointer.
TEST(HTTPWriteTask, ParseConfigStaticFieldEmptyPointer) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"channel",
              {{"pointer", "/value"}, {"json_type", "number"}, {"channel", 1}}},
             {"fields",
              {{
                  {"type", "static"},
                  {"pointer", ""},
                  {"json_type", "number"},
                  {"value", 42},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(WriteTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should fail when a generated field has an empty pointer.
TEST(HTTPWriteTask, ParseConfigGeneratedFieldEmptyPointer) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"endpoints",
         {{
             {"method", "POST"},
             {"path", "/api/data"},
             {"channel",
              {{"pointer", "/value"}, {"json_type", "number"}, {"channel", 1}}},
             {"fields",
              {{
                  {"type", "generated"},
                  {"pointer", ""},
                  {"generator", "uuid"},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(WriteTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should POST a numeric channel value to the server.
TEST(HTTPWriteTask, POSTNumericValue) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

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

/// @brief it should PUT a string channel value to the server.
TEST(HTTPWriteTask, PUTStringValue) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::PUT,
                .path = "/api/setpoint",
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

    WriteEndpoint ep;
    ep.request.method = Method::PUT;
    ep.request.path = "/api/setpoint";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/state");
    ep.channel.json_type = x::json::Type::String;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::string("ON")));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_EQ(body["state"].get<std::string>(), "ON");
}

/// @brief it should send a bare primitive body when channel pointer is root.
TEST(HTTPWriteTask, BarePrimitiveBody) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::PUT,
                .path = "/api/setpoint",
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

    WriteEndpoint ep;
    ep.request.method = Method::PUT;
    ep.request.path = "/api/setpoint";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{99.0}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, "99.0");
}

/// @brief it should include static fields in the request body.
TEST(HTTPWriteTask, StaticFields) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;
    ep.static_fields = {{
        .pointer = x::json::json::json_pointer("/device_id"),
        .value = "sensor-01",
    }};

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_NEAR(body["value"].get<double>(), 10.0, 0.001);
    EXPECT_EQ(body["device_id"].get<std::string>(), "sensor-01");
}

/// @brief it should include a generated UUID field.
TEST(HTTPWriteTask, GeneratedUUIDField) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;
    ep.generated_fields = {{
        .pointer = x::json::json::json_pointer("/request_id"),
        .generator = GeneratorType::UUID,
    }};

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{5.0}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_TRUE(body.contains("request_id"));
    // Validate UUID v4 format.
    std::regex uuid_re("[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}");
    EXPECT_TRUE(std::regex_match(body["request_id"].get<std::string>(), uuid_re));
}

/// @brief it should return an error on 4xx responses.
TEST(HTTPWriteTask, Error4xxResponse) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{42.0}));

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_ERROR);
}

/// @brief it should return a temporary error on 5xx responses.
TEST(HTTPWriteTask, Error5xxResponse) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{42.0}));

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, errors::TEMPORARY_ERROR);
}

/// @brief it should fire multiple endpoints independently.
TEST(HTTPWriteTask, MultipleEndpointsFireIndependently) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep1;
    ep1.request.method = Method::POST;
    ep1.request.path = "/api/temp";
    ep1.request.request_content_type = "application/json";
    ep1.channel.pointer = x::json::json::json_pointer("/value");
    ep1.channel.json_type = x::json::Type::Number;
    ep1.channel.channel_key = 1;

    WriteEndpoint ep2;
    ep2.request.method = Method::PUT;
    ep2.request.path = "/api/pressure";
    ep2.request.request_content_type = "application/json";
    ep2.channel.pointer = x::json::json::json_pointer("/value");
    ep2.channel.json_type = x::json::Type::Number;
    ep2.channel.channel_key = 2;

    cfg.endpoints = {ep1, ep2};
    cfg.cmd_keys = {1, 2};

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
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep1;
    ep1.request.method = Method::POST;
    ep1.request.path = "/api/temp";
    ep1.request.request_content_type = "application/json";
    ep1.channel.pointer = x::json::json::json_pointer("/value");
    ep1.channel.json_type = x::json::Type::Number;
    ep1.channel.channel_key = 1;

    WriteEndpoint ep2;
    ep2.request.method = Method::POST;
    ep2.request.path = "/api/pressure";
    ep2.request.request_content_type = "application/json";
    ep2.channel.pointer = x::json::json::json_pointer("/value");
    ep2.channel.json_type = x::json::Type::Number;
    ep2.channel.channel_key = 2;

    WriteEndpoint ep3;
    ep3.request.method = Method::POST;
    ep3.request.path = "/api/humidity";
    ep3.request.request_content_type = "application/json";
    ep3.channel.pointer = x::json::json::json_pointer("/value");
    ep3.channel.json_type = x::json::Type::Number;
    ep3.channel.channel_key = 3;

    cfg.endpoints = {ep1, ep2, ep3};
    cfg.cmd_keys = {1, 2, 3};

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
/// still return that error.
TEST(HTTPWriteTask, ParallelBatchPartialFailure) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep1;
    ep1.request.method = Method::POST;
    ep1.request.path = "/api/good";
    ep1.request.request_content_type = "application/json";
    ep1.channel.pointer = x::json::json::json_pointer("/value");
    ep1.channel.json_type = x::json::Type::Number;
    ep1.channel.channel_key = 1;

    WriteEndpoint ep2;
    ep2.request.method = Method::POST;
    ep2.request.path = "/api/bad";
    ep2.request.request_content_type = "application/json";
    ep2.channel.pointer = x::json::json::json_pointer("/value");
    ep2.channel.json_type = x::json::Type::Number;
    ep2.channel.channel_key = 2;

    cfg.endpoints = {ep1, ep2};
    cfg.cmd_keys = {1, 2};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{10.0}));
    frame.emplace(synnax::channel::Key(2), x::telem::Series(std::vector<double>{20.0}));

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, errors::TEMPORARY_ERROR);

    // Both requests should have been sent (parallel, not short-circuit).
    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 2);
}

/// @brief it should use only the last sample value when a series has multiple
/// samples (last write wins).
TEST(HTTPWriteTask, LastWriteWins) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::vector<double>{1.0, 2.0, 3.0, 4.0, 5.0})
    );

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_NEAR(body["value"].get<double>(), 5.0, 0.001);
}

/// @brief it should handle sequential writes correctly.
TEST(HTTPWriteTask, SequentialSends) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

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
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

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

/// @brief it should return an error when a string value cannot be converted to a JSON
/// number.
TEST(HTTPWriteTask, BadConversionStringToNumber) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::string("not a number"))
    );

    auto err = sink->write(frame);
    ASSERT_OCCURRED_AS(err, x::json::CONVERSION_ERROR);

    // No request should have been sent.
    EXPECT_EQ(server.received_requests().size(), 0);
}

/// @brief it should PATCH with a boolean channel value.
TEST(HTTPWriteTask, PATCHBooleanValue) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::PATCH,
                .path = "/api/config",
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

    WriteEndpoint ep;
    ep.request.method = Method::PATCH;
    ep.request.path = "/api/config";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/enabled");
    ep.channel.json_type = x::json::Type::Boolean;
    ep.channel.channel_key = 1;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<uint8_t>{1}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_EQ(body["enabled"].get<bool>(), true);
}

/// @brief it should format a TIMESTAMP channel value as ISO8601 when time_format
/// is set.
TEST(HTTPWriteTask, TimeFormatISO8601) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/timestamp");
    ep.channel.json_type = x::json::Type::String;
    ep.channel.channel_key = 1;
    ep.channel.time_format = x::json::TimeFormat::ISO8601;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    // Send a nanosecond timestamp: 2025-01-15T00:00:00Z = 1736899200000000000 ns.
    const int64_t ts_ns = 1736899200000000000LL;
    x::telem::Frame frame;
    frame.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::vector<int64_t>{ts_ns})
    );

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);
    const auto ts_str = body["timestamp"].get<std::string>();
    EXPECT_TRUE(ts_str.find("2025-01-15") != std::string::npos);
}

/// @brief it should format a TIMESTAMP channel value as unix seconds when
/// time_format is set.
TEST(HTTPWriteTask, TimeFormatUnixSeconds) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/ts");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;
    ep.channel.time_format = x::json::TimeFormat::UnixSecond;

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    // 1736899200 seconds = 2025-01-15T00:00:00Z in nanoseconds.
    const int64_t ts_ns = 1736899200000000000LL;
    x::telem::Frame frame;
    frame.emplace(
        synnax::channel::Key(1),
        x::telem::Series(std::vector<int64_t>{ts_ns})
    );

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_NEAR(body["ts"].get<double>(), 1736899200.0, 1.0);
}

/// @brief it should include a generated timestamp field formatted as ISO8601.
TEST(HTTPWriteTask, GeneratedTimestampField) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/control";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer("/value");
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;
    ep.generated_fields = {{
        .pointer = x::json::json::json_pointer("/created_at"),
        .generator = GeneratorType::Timestamp,
        .time_format = x::json::TimeFormat::ISO8601,
    }};

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{7.5}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);
    EXPECT_NEAR(body["value"].get<double>(), 7.5, 0.001);
    ASSERT_TRUE(body.contains("created_at"));
    // ISO8601 timestamps contain 'T' and have at least 19 chars (YYYY-MM-DDTHH:MM:SS).
    const auto ts_str = body["created_at"].get<std::string>();
    EXPECT_GE(ts_str.size(), 19);
    EXPECT_NE(ts_str.find('T'), std::string::npos);
}

/// @brief it should construct deeply nested JSON bodies from pointers correctly.
TEST(HTTPWriteTask, DeeplyNestedPointer) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/data",
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

    WriteEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/data";
    ep.request.request_content_type = "application/json";
    ep.channel.pointer = x::json::json::json_pointer(
        "/payload/sensors/temperature/reading"
    );
    ep.channel.json_type = x::json::Type::Number;
    ep.channel.channel_key = 1;
    ep.static_fields = {
        {
            .pointer = x::json::json::json_pointer("/payload/sensors/temperature/unit"),
            .value = "celsius",
        },
        {
            .pointer = x::json::json::json_pointer("/metadata/source"),
            .value = "driver",
        },
    };

    cfg.endpoints = {ep};
    cfg.cmd_keys = {1};

    auto [sink, processor] = make_sink(cfg, server.base_url());

    x::telem::Frame frame;
    frame.emplace(synnax::channel::Key(1), x::telem::Series(std::vector<double>{23.5}));

    ASSERT_NIL(sink->write(frame));

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto body = x::json::json::parse(reqs[0].body);

    // Verify the deeply nested structure was constructed correctly.
    EXPECT_NEAR(
        body["payload"]["sensors"]["temperature"]["reading"].get<double>(),
        23.5,
        0.001
    );
    EXPECT_EQ(
        body["payload"]["sensors"]["temperature"]["unit"].get<std::string>(),
        "celsius"
    );
    EXPECT_EQ(body["metadata"]["source"].get<std::string>(), "driver");
}

/// @brief it should skip disabled endpoints and only send to enabled ones.
TEST(HTTPWriteTask, DisabledEndpointSkipped) {
    mock::Server server(
        mock::ServerConfig{
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

    WriteEndpoint ep1;
    ep1.enabled = true;
    ep1.request.method = Method::POST;
    ep1.request.path = "/api/active";
    ep1.request.request_content_type = "application/json";
    ep1.channel.pointer = x::json::json::json_pointer("/value");
    ep1.channel.json_type = x::json::Type::Number;
    ep1.channel.channel_key = 1;

    WriteEndpoint ep2;
    ep2.enabled = false;
    ep2.request.method = Method::POST;
    ep2.request.path = "/api/inactive";
    ep2.request.request_content_type = "application/json";
    ep2.channel.pointer = x::json::json::json_pointer("/value");
    ep2.channel.json_type = x::json::Type::Number;
    ep2.channel.channel_key = 2;

    cfg.endpoints = {ep1, ep2};
    cfg.cmd_keys = {1};

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
}
