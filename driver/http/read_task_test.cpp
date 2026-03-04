// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/test/test.h"

#include "driver/http/errors/errors.h"
#include "driver/http/mock/server.h"
#include "driver/http/read_task.h"

namespace driver::http {
namespace {
/// @brief helper to build a ReadTaskSource from config and a mock server URL.
std::pair<std::unique_ptr<ReadTaskSource>, x::errors::Error> make_source(
    const ReadTaskConfig &cfg,
    const std::string &base_url,
    const x::json::json &conn_extra = x::json::json::object()
) {
    auto conn_json = x::json::json{
        {"base_url", base_url},
        {"timeout_ms", 1000},
        {"verify_ssl", false},
    };
    conn_json.update(conn_extra);
    auto conn_parser = x::json::Parser(conn_json);
    auto conn = device::ConnectionConfig(conn_parser);

    std::vector<device::RequestConfig> request_configs;
    request_configs.reserve(cfg.endpoints.size());
    for (const auto &ep: cfg.endpoints)
        request_configs.push_back(ep.request);

    auto [client, err] = device::Client::create(std::move(conn), request_configs);
    if (err) return {nullptr, err};
    return {
        std::make_unique<ReadTaskSource>(ReadTaskConfig(cfg), std::move(client)),
        x::errors::NIL,
    };
}
}

/// @brief it should fail to parse config when endpoints array is empty.
TEST(HTTPReadTask, ParseConfigEmptyEndpoints) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints", x::json::json::array()},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should fail to parse config when device field is missing.
TEST(HTTPReadTask, ParseConfigMissingDevice) {
    synnax::task::Task task;
    task.config = {
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"pointer", "/temp"},
                  {"channel", 1},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should fail to parse config when a channel is used multiple times.
TEST(HTTPReadTask, ParseConfigDuplicateChannel) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {
                  {{"pointer", "/temp"}, {"channel", 1}},
                  {{"pointer", "/humidity"}, {"channel", 1}},
              }},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should extract a numeric field from a single GET endpoint.
TEST(HTTPReadTask, SingleEndpointGETNumericField) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"temperature": 23.5, "humidity": 80})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField temp_field;
    temp_field.pointer = x::json::json::json_pointer("/temperature");
    temp_field.channel_key = 1;

    ReadField humidity_field;
    humidity_field.pointer = x::json::json::json_pointer("/humidity");
    humidity_field.channel_key = 2;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field, humidity_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
        .key = 1,
    };
    cfg.channels[2] = {.name = "humidity", .data_type = x::telem::FLOAT64_T, .key = 2};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(1, 0), 23.5, 0.001);
    EXPECT_NEAR(fr.at<double>(2, 0), 80.0, 0.001);
}

/// @brief it should extract nested JSON fields using JSON Pointer paths.
TEST(HTTPReadTask, NestedJSONPointerPaths) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/sensors",
                .status_code = 200,
                .response_body =
                    R"({"data":{"sensors":[{"value":42.0},{"value":99.0}]}})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/data/sensors/0/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/sensors";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.name = "sensor_0", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 42.0, 0.001);
}

/// @brief a missing JSON pointer should produce a warning, not a hard error.
TEST(HTTPReadTask, MissingJSONFieldWarning) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"temperature": 23.5})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/nonexistent");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.name = "missing", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    EXPECT_EQ(fr.size(), 0);
}

/// @brief it should return SERVER_ERROR on 5xx status codes.
TEST(HTTPReadTask, ServerErrorOn5xx) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 500,
                .response_body = R"({"error":"internal"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.name = "val", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_OCCURRED_AS(res.error, errors::SERVER_ERROR);
}

/// @brief it should return CLIENT_ERROR on 4xx status codes.
TEST(HTTPReadTask, ClientErrorOn4xx) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 404,
                .response_body = R"({"error":"not found"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.name = "val", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_OCCURRED_AS(res.error, errors::CLIENT_ERROR);
}

/// @brief it should convert JSON types correctly (bool to uint8, string to string).
TEST(HTTPReadTask, TypeConversions) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body =
                    R"({"active": true, "label": "sensor-1", "count": 42})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField bool_field;
    bool_field.pointer = x::json::json::json_pointer("/active");
    bool_field.channel_key = 1;

    ReadField string_field;
    string_field.pointer = x::json::json::json_pointer("/label");
    string_field.channel_key = 2;

    ReadField int_field;
    int_field.pointer = x::json::json::json_pointer("/count");
    int_field.channel_key = 3;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {bool_field, string_field, int_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.name = "active", .data_type = x::telem::UINT8_T, .key = 1};
    cfg.channels[2] = {.name = "label", .data_type = x::telem::STRING_T, .key = 2};
    cfg.channels[3] = {.name = "count", .data_type = x::telem::INT32_T, .key = 3};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 3);
    EXPECT_EQ(fr.at<uint8_t>(1, 0), 1);
    EXPECT_EQ(fr.at<int32_t>(3, 0), 42);
}

/// @brief it should extract a string field from a JSON response.
TEST(HTTPReadTask, StringField) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body =
                    R"({"name": "sensor-42", "status": "online", "value": 3.14})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField name_field;
    name_field.pointer = x::json::json::json_pointer("/name");
    name_field.channel_key = 1;

    ReadField status_field;
    status_field.pointer = x::json::json::json_pointer("/status");
    status_field.channel_key = 2;

    ReadField value_field;
    value_field.pointer = x::json::json::json_pointer("/value");
    value_field.channel_key = 3;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {name_field, status_field, value_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.name = "name", .data_type = x::telem::STRING_T, .key = 1};
    cfg.channels[2] = {.name = "status", .data_type = x::telem::STRING_T, .key = 2};
    cfg.channels[3] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 3};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 3);
    EXPECT_EQ(std::get<std::string>(fr.at(1, 0)), "sensor-42");
    EXPECT_EQ(std::get<std::string>(fr.at(2, 0)), "online");
    EXPECT_NEAR(fr.at<double>(3, 0), 3.14, 0.001);
}

/// @brief decimal values for integer channels should produce a warning, not a
/// hard error.
TEST(HTTPReadTask, DecimalToIntegerWarns) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 3.7})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.name = "count", .data_type = x::telem::INT32_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    EXPECT_EQ(fr.size(), 0);
}

/// @brief negative values for unsigned integer channels should produce a
/// warning, not a hard error.
TEST(HTTPReadTask, NegativeForUnsignedWarns) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": -5})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.name = "count", .data_type = x::telem::UINT32_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    EXPECT_EQ(fr.size(), 0);
}

/// @brief it should use software timing (midpoint) for index channels when the
/// index channel is not listed as a field.
TEST(HTTPReadTask, SoftwareTimingIndex) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 42.0})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] =
        {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1, .index = 100};
    cfg.software_timed_indexes[100] = 0;

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    // The index channel should have a timestamp (non-zero).
    auto ts = fr.at<int64_t>(100, 0);
    EXPECT_GT(ts, 0);
}

/// @brief it should extract timestamps from the JSON response when the index
/// channel is listed as an explicit field with a timestamp_format.
TEST(HTTPReadTask, ExplicitIndexFieldTimestamp) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 42.0, "timestamp": 1700000000})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField data_field;
    data_field.pointer = x::json::json::json_pointer("/value");
    data_field.channel_key = 1;

    ReadField index_field;
    index_field.pointer = x::json::json::json_pointer("/timestamp");
    index_field.channel_key = 100;
    index_field.time_format = x::json::TimeFormat::UnixSecond;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {data_field, index_field};

    cfg.endpoints = {ep};

    cfg.channels[1] =
        {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1, .index = 100};
    cfg.channels[100] = {
        .name = "time",
        .data_type = x::telem::TIMESTAMP_T,
        .key = 100,
        .is_index = true,
    };
    // No index_sources needed — index channel is an explicit field.

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    // 1700000000 seconds = 1700000000000000000 nanoseconds
    auto ts = fr.at<int64_t>(100, 0);
    EXPECT_EQ(ts, 1700000000000000000LL);
}

/// @brief it should poll multiple endpoints in parallel.
TEST(HTTPReadTask, MultipleEndpoints) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {
                    .method = Method::GET,
                    .path = "/api/temp",
                    .status_code = 200,
                    .response_body = R"({"temp": 25.0})",
                },
                {
                    .method = Method::GET,
                    .path = "/api/pressure",
                    .status_code = 200,
                    .response_body = R"({"pressure": 1013.25})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField temp_field;
    temp_field.pointer = x::json::json::json_pointer("/temp");
    temp_field.channel_key = 1;

    ReadField pressure_field;
    pressure_field.pointer = x::json::json::json_pointer("/pressure");
    pressure_field.channel_key = 2;

    ReadEndpoint ep1;
    ep1.request.method = Method::GET;
    ep1.request.path = "/api/temp";
    ep1.body = "";
    ep1.fields = {temp_field};

    ReadEndpoint ep2;
    ep2.request.method = Method::GET;
    ep2.request.path = "/api/pressure";
    ep2.body = "";
    ep2.fields = {pressure_field};

    cfg.endpoints = {ep1, ep2};

    cfg.channels[1] = {.name = "temp", .data_type = x::telem::FLOAT64_T, .key = 1};
    cfg.channels[2] = {.name = "pressure", .data_type = x::telem::FLOAT64_T, .key = 2};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(1, 0), 25.0, 0.001);
    EXPECT_NEAR(fr.at<double>(2, 0), 1013.25, 0.001);
}

/// @brief it should send POST body and extract response fields.
TEST(HTTPReadTask, POSTWithBody) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/query",
                .status_code = 200,
                .response_body = R"({"result": 99.9})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/result");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/query";
    ep.body = R"({"query": "latest"})";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.name = "result", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 99.9, 0.001);
}

/// @brief it should reject PUT method in read task config.
TEST(HTTPReadTask, ParseConfigRejectsPUT) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "PUT"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"pointer", "/temp"},
                  {"channel", 1},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should reject DELETE method in read task config.
TEST(HTTPReadTask, ParseConfigRejectsDELETE) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "DELETE"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"pointer", "/temp"},
                  {"channel", 1},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief test fixture for parse tests that need a real Synnax client with pre-created
/// channels and device.
class HTTPReadTaskParseTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    std::string device_key;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        auto rack = ASSERT_NIL_P(
            client->racks.create(make_unique_channel_name("http_read_test_rack"))
        );
        device_key = make_unique_channel_name("http_read_test_device");
        x::json::json props = {{"secure", false}, {"timeout_ms", 1000}};
        synnax::device::Device dev{
            .key = device_key,
            .name = "HTTP Read Test Device",
            .rack = rack.key,
            .location = "localhost:0",
            .make = "http",
            .model = "HTTP Device",
            .properties = props.get<x::json::json::object_t>(),
        };
        ASSERT_NIL(client->devices.create(dev));
        ctx = std::make_shared<task::MockContext>(client);
    }
};

/// @brief it should error when a TIMESTAMP_T channel has no timestamp_format.
TEST_F(HTTPReadTaskParseTest, TimestampChannelMissingFormat) {
    auto idx = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("idx"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto ts_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ts_data"),
        x::telem::TIMESTAMP_T,
        idx.key,
        false
    ));

    synnax::task::Task task;
    task.config = {
        {"device", device_key},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"pointer", "/ts"},
                  {"channel", ts_ch.key},
              }}},
         }}},
    };
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should error when channels on different endpoints share the same
/// index channel — we wouldn't know which endpoint's response time to use for
/// software timing.
TEST_F(HTTPReadTaskParseTest, CrossEndpointSharedIndex) {
    auto idx = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("idx"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("temp"),
        x::telem::FLOAT64_T,
        idx.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("humidity"),
        x::telem::FLOAT64_T,
        idx.key,
        false
    ));

    synnax::task::Task task;
    task.config = {
        {"device", device_key},
        {"rate", 1.0},
        {"endpoints",
         {
             {
                 {"method", "GET"},
                 {"path", "/api/temp"},
                 {"fields", {{{"pointer", "/temp"}, {"channel", ch1.key}}}},
             },
             {
                 {"method", "GET"},
                 {"path", "/api/humidity"},
                 {"fields", {{{"pointer", "/humidity"}, {"channel", ch2.key}}}},
             },
         }},
    };
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should succeed when two fields on the same endpoint share an index
/// channel and the index is listed as an explicit field.
TEST_F(HTTPReadTaskParseTest, SameEndpointSharedIndexAsField) {
    auto idx = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("idx"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("temp"),
        x::telem::FLOAT64_T,
        idx.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("humidity"),
        x::telem::FLOAT64_T,
        idx.key,
        false
    ));

    synnax::task::Task task;
    task.config = {
        {"device", device_key},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {
                  {{"pointer", "/temp"}, {"channel", ch1.key}},
                  {{"pointer", "/humidity"}, {"channel", ch2.key}},
                  {
                      {"pointer", "/timestamp"},
                      {"channel", idx.key},
                      {"timestamp_format", "unix_sec"},
                  },
              }},
         }}},
    };
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    // Index channel is a field, so no software-timed indexes needed.
    EXPECT_TRUE(cfg.software_timed_indexes.empty());
}

/// @brief it should succeed when two fields on the same endpoint share an index
/// channel and the index is NOT listed as a field (software timing).
TEST_F(HTTPReadTaskParseTest, SameEndpointSharedIndexSoftwareTiming) {
    auto idx = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("idx"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("temp"),
        x::telem::FLOAT64_T,
        idx.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("humidity"),
        x::telem::FLOAT64_T,
        idx.key,
        false
    ));

    synnax::task::Task task;
    task.config = {
        {"device", device_key},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {
                  {{"pointer", "/temp"}, {"channel", ch1.key}},
                  {{"pointer", "/humidity"}, {"channel", ch2.key}},
              }},
         }}},
    };
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    EXPECT_EQ(cfg.software_timed_indexes.size(), 1);
    EXPECT_TRUE(cfg.software_timed_indexes.count(idx.key));
}

/// @brief it should silently ignore timestamp_format on a non-timestamp channel.
TEST_F(HTTPReadTaskParseTest, TimestampFormatOnNonTimestamp) {
    auto idx = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("idx"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("value"),
        x::telem::FLOAT64_T,
        idx.key,
        false
    ));

    synnax::task::Task task;
    task.config = {
        {"device", device_key},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"pointer", "/value"},
                  {"channel", ch.key},
                  {"timestamp_format", "unix_sec"},
              }}},
         }}},
    };
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
}

/// @brief it should successfully read 10 times in succession from the same endpoint.
TEST(HTTPReadTask, RepeatedReads) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 42.0})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    for (int i = 0; i < 10; i++) {
        x::telem::Frame fr;
        auto res = source->read(breaker, fr);
        ASSERT_NIL(res.error);
        EXPECT_EQ(fr.size(), 1);
        EXPECT_NEAR(fr.at<double>(1, 0), 42.0, 0.001);
    }
    breaker.stop();
}

////////////////////////////// Disabled Channels ///////////////////////////////

/// @brief it should skip disabled fields and only return enabled ones.
TEST(HTTPReadTask, DisabledFieldsSkipped) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body =
                    R"({"temperature": 23.5, "humidity": 80, "pressure": 1013})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField temp_field;
    temp_field.pointer = x::json::json::json_pointer("/temperature");
    temp_field.channel_key = 1;
    temp_field.enabled = true;

    ReadField humidity_field;
    humidity_field.pointer = x::json::json::json_pointer("/humidity");
    humidity_field.channel_key = 2;
    humidity_field.enabled = false;

    ReadField pressure_field;
    pressure_field.pointer = x::json::json::json_pointer("/pressure");
    pressure_field.channel_key = 3;
    pressure_field.enabled = true;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field, humidity_field, pressure_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
        .key = 1
    };
    cfg.channels[3] = {.name = "pressure", .data_type = x::telem::FLOAT64_T, .key = 3};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(1, 0), 23.5, 0.001);
    EXPECT_NEAR(fr.at<double>(3, 0), 1013.0, 0.001);
}

/// @brief it should not include disabled fields in the writer config.
TEST(HTTPReadTask, DisabledFieldsExcludedFromWriterConfig) {
    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField enabled_field;
    enabled_field.pointer = x::json::json::json_pointer("/temp");
    enabled_field.channel_key = 1;
    enabled_field.enabled = true;

    ReadField disabled_field;
    disabled_field.pointer = x::json::json::json_pointer("/humidity");
    disabled_field.channel_key = 2;
    disabled_field.enabled = false;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {enabled_field, disabled_field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
        .key = 1
    };

    auto conn_parser = x::json::Parser(
        x::json::json{
            {"base_url", "http://localhost:9999"},
            {"timeout_ms", 100},
            {"verify_ssl", false},
        }
    );
    auto conn = device::ConnectionConfig(conn_parser);
    std::vector<device::RequestConfig> request_configs;
    for (const auto &e: cfg.endpoints)
        request_configs.push_back(e.request);
    auto [client, err] = device::Client::create(std::move(conn), request_configs);
    ASSERT_NIL(err);

    ReadTaskSource source(std::move(cfg), std::move(client));
    auto wc = source.writer_config();
    EXPECT_EQ(wc.channels.size(), 1);
    EXPECT_EQ(wc.channels[0], 1);
}

/// @brief it should fail to parse config when all fields are disabled.
TEST(HTTPReadTask, ParseConfigAllFieldsDisabled) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"pointer", "/temp"},
                  {"channel", 1},
                  {"enabled", false},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief disabled fields should not cause a missing-field error even if the
/// JSON pointer would not match the response.
TEST(HTTPReadTask, DisabledFieldMissingPointerNoError) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"temperature": 23.5})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField enabled_field;
    enabled_field.pointer = x::json::json::json_pointer("/temperature");
    enabled_field.channel_key = 1;
    enabled_field.enabled = true;

    ReadField disabled_field;
    disabled_field.pointer = x::json::json::json_pointer("/nonexistent");
    disabled_field.channel_key = 2;
    disabled_field.enabled = false;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {enabled_field, disabled_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
        .key = 1
    };

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 23.5, 0.001);
}

///////////////////////////////// HTTPS Tests /////////////////////////////////

/// @brief it should read from an HTTPS server with SSL verification disabled.
TEST(HTTPReadTask, HTTPSReadSingleEndpoint) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 77.7})",
            }},
            .secure = true,
            .cert_path = "driver/http/mock/test_cert.pem",
            .key_path = "driver/http/mock/test_key.pem",
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 77.7, 0.001);
}

/// @brief it should read multiple endpoints over HTTPS.
TEST(HTTPReadTask, HTTPSMultipleEndpoints) {
    mock::Server server(
        mock::ServerConfig{
            .routes =
                {
                    {
                        .method = Method::GET,
                        .path = "/api/temp",
                        .status_code = 200,
                        .response_body = R"({"temp": 22.5})",
                    },
                    {
                        .method = Method::GET,
                        .path = "/api/pressure",
                        .status_code = 200,
                        .response_body = R"({"pressure": 1015.0})",
                    },
                },
            .secure = true,
            .cert_path = "driver/http/mock/test_cert.pem",
            .key_path = "driver/http/mock/test_key.pem",
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField temp_field;
    temp_field.pointer = x::json::json::json_pointer("/temp");
    temp_field.channel_key = 1;

    ReadField pressure_field;
    pressure_field.pointer = x::json::json::json_pointer("/pressure");
    pressure_field.channel_key = 2;

    ReadEndpoint ep1;
    ep1.request.method = Method::GET;
    ep1.request.path = "/api/temp";
    ep1.body = "";
    ep1.fields = {temp_field};

    ReadEndpoint ep2;
    ep2.request.method = Method::GET;
    ep2.request.path = "/api/pressure";
    ep2.body = "";
    ep2.fields = {pressure_field};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.name = "temp", .data_type = x::telem::FLOAT64_T, .key = 1};
    cfg.channels[2] = {.name = "pressure", .data_type = x::telem::FLOAT64_T, .key = 2};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(1, 0), 22.5, 0.001);
    EXPECT_NEAR(fr.at<double>(2, 0), 1015.0, 0.001);
}

/// @brief it should read repeated times from HTTPS endpoints.
TEST(HTTPReadTask, HTTPSRepeatedReads) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 55.5})",
            }},
            .secure = true,
            .cert_path = "driver/http/mock/test_cert.pem",
            .key_path = "driver/http/mock/test_key.pem",
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    for (int i = 0; i < 5; i++) {
        x::telem::Frame fr;
        auto res = source->read(breaker, fr);
        ASSERT_NIL(res.error);
        EXPECT_EQ(fr.size(), 1);
        EXPECT_NEAR(fr.at<double>(1, 0), 55.5, 0.001);
    }
    breaker.stop();
}

/// @brief it should POST to an HTTPS endpoint and extract response fields.
TEST(HTTPReadTask, HTTPSPOSTWithBody) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/query",
                .status_code = 200,
                .response_body = R"({"result": 88.8})",
            }},
            .secure = true,
            .cert_path = "driver/http/mock/test_cert.pem",
            .key_path = "driver/http/mock/test_key.pem",
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/result");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/query";
    ep.body = R"({"query": "latest"})";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.name = "result", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 88.8, 0.001);
}

///////////////////////////// Partial Failures ////////////////////////////////

/// @brief when the first endpoint returns 5xx but the second would succeed,
/// the read should fail with SERVER_ERROR.
TEST(HTTPReadTask, PartialFailureFirstEndpoint5xx) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {
                    .method = Method::GET,
                    .path = "/api/failing",
                    .status_code = 500,
                    .response_body = R"({"error":"internal"})",
                },
                {
                    .method = Method::GET,
                    .path = "/api/ok",
                    .status_code = 200,
                    .response_body = R"({"value": 42.0})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field1;
    field1.pointer = x::json::json::json_pointer("/error");
    field1.channel_key = 1;

    ReadField field2;
    field2.pointer = x::json::json::json_pointer("/value");
    field2.channel_key = 2;

    ReadEndpoint ep1;
    ep1.request.method = Method::GET;
    ep1.request.path = "/api/failing";
    ep1.body = "";
    ep1.fields = {field1};

    ReadEndpoint ep2;
    ep2.request.method = Method::GET;
    ep2.request.path = "/api/ok";
    ep2.body = "";
    ep2.fields = {field2};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.name = "error_msg", .data_type = x::telem::STRING_T, .key = 1};
    cfg.channels[2] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 2};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_OCCURRED_AS(res.error, errors::SERVER_ERROR);
}

/// @brief when the second endpoint returns 4xx but the first succeeds,
/// the read should fail with CLIENT_ERROR.
TEST(HTTPReadTask, PartialFailureSecondEndpoint4xx) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {
                    .method = Method::GET,
                    .path = "/api/ok",
                    .status_code = 200,
                    .response_body = R"({"value": 42.0})",
                },
                {
                    .method = Method::GET,
                    .path = "/api/failing",
                    .status_code = 404,
                    .response_body = R"({"error":"not found"})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field1;
    field1.pointer = x::json::json::json_pointer("/value");
    field1.channel_key = 1;

    ReadField field2;
    field2.pointer = x::json::json::json_pointer("/error");
    field2.channel_key = 2;

    ReadEndpoint ep1;
    ep1.request.method = Method::GET;
    ep1.request.path = "/api/ok";
    ep1.body = "";
    ep1.fields = {field1};

    ReadEndpoint ep2;
    ep2.request.method = Method::GET;
    ep2.request.path = "/api/failing";
    ep2.body = "";
    ep2.fields = {field2};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1};
    cfg.channels[2] = {.name = "error_msg", .data_type = x::telem::STRING_T, .key = 2};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_OCCURRED_AS(res.error, errors::CLIENT_ERROR);
}

/// @brief when one endpoint has a missing field pointer, the other endpoint's
/// data should still come through with a warning.
TEST(HTTPReadTask, PartialFailureMissingFieldInSecondEndpoint) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {
                    .method = Method::GET,
                    .path = "/api/temp",
                    .status_code = 200,
                    .response_body = R"({"temp": 25.0})",
                },
                {
                    .method = Method::GET,
                    .path = "/api/pressure",
                    .status_code = 200,
                    .response_body = R"({"psi": 14.7})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField temp_field;
    temp_field.pointer = x::json::json::json_pointer("/temp");
    temp_field.channel_key = 1;

    ReadField pressure_field;
    // Pointer that doesn't exist in the response.
    pressure_field.pointer = x::json::json::json_pointer("/pressure");
    pressure_field.channel_key = 2;

    ReadEndpoint ep1;
    ep1.request.method = Method::GET;
    ep1.request.path = "/api/temp";
    ep1.body = "";
    ep1.fields = {temp_field};

    ReadEndpoint ep2;
    ep2.request.method = Method::GET;
    ep2.request.path = "/api/pressure";
    ep2.body = "";
    ep2.fields = {pressure_field};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.name = "temp", .data_type = x::telem::FLOAT64_T, .key = 1};
    cfg.channels[2] = {.name = "pressure", .data_type = x::telem::FLOAT64_T, .key = 2};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    // First endpoint succeeded, second endpoint's field was missing.
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 25.0, 0.001);
}

/// @brief when one endpoint returns invalid JSON, the other endpoint's data
/// should still come through with a warning.
TEST(HTTPReadTask, PartialFailureInvalidJSONInOneEndpoint) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {
                    .method = Method::GET,
                    .path = "/api/ok",
                    .status_code = 200,
                    .response_body = R"({"value": 42.0})",
                },
                {
                    .method = Method::GET,
                    .path = "/api/broken",
                    .status_code = 200,
                    .response_body = "not valid json{{{",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field1;
    field1.pointer = x::json::json::json_pointer("/value");
    field1.channel_key = 1;

    ReadField field2;
    field2.pointer = x::json::json::json_pointer("/data");
    field2.channel_key = 2;

    ReadEndpoint ep1;
    ep1.request.method = Method::GET;
    ep1.request.path = "/api/ok";
    ep1.body = "";
    ep1.fields = {field1};

    ReadEndpoint ep2;
    ep2.request.method = Method::GET;
    ep2.request.path = "/api/broken";
    ep2.body = "";
    ep2.fields = {field2};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1};
    cfg.channels[2] = {.name = "data", .data_type = x::telem::FLOAT64_T, .key = 2};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    // First endpoint parsed, second was invalid JSON.
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 42.0, 0.001);
}

/// @brief when one endpoint has a type conversion error, the other endpoint's
/// data should still come through with a warning.
TEST(HTTPReadTask, PartialFailureTypeConversionError) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {
                    .method = Method::GET,
                    .path = "/api/good",
                    .status_code = 200,
                    .response_body = R"({"value": 42.0})",
                },
                {
                    .method = Method::GET,
                    .path = "/api/bad",
                    .status_code = 200,
                    .response_body = R"({"count": 3.7})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field1;
    field1.pointer = x::json::json::json_pointer("/value");
    field1.channel_key = 1;

    ReadField field2;
    field2.pointer = x::json::json::json_pointer("/count");
    field2.channel_key = 2;

    ReadEndpoint ep1;
    ep1.request.method = Method::GET;
    ep1.request.path = "/api/good";
    ep1.body = "";
    ep1.fields = {field1};

    ReadEndpoint ep2;
    ep2.request.method = Method::GET;
    ep2.request.path = "/api/bad";
    ep2.body = "";
    ep2.fields = {field2};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1};
    // INT32_T will fail on 3.7 (decimal truncation) — produces warning.
    cfg.channels[2] = {.name = "count", .data_type = x::telem::INT32_T, .key = 2};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    // First endpoint parsed fine, second had conversion error.
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 42.0, 0.001);
}

////////////////////// Connection-Level Query Parameters ///////////////////////

//////////////////////////////// Sample Clock //////////////////////////////////

/// @brief the sample clock should regulate the read rate so that multiple reads
/// take at least the expected duration.
TEST(HTTPReadTask, SampleClockRegulatesRate) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 1.0})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(20); // 20 Hz → 50ms period

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();

    const int n_reads = 5;
    const auto start = x::telem::TimeStamp::now();
    for (int i = 0; i < n_reads; i++) {
        x::telem::Frame fr;
        auto res = source->read(breaker, fr);
        ASSERT_NIL(res.error);
    }
    const auto elapsed = x::telem::TimeStamp::now() - start;
    breaker.stop();

    // 5 reads at 20 Hz = 5 × 50ms = 250ms minimum. Allow some tolerance.
    EXPECT_GE(elapsed, 200 * x::telem::MILLISECOND);
}

/// @brief the sample clock should not prevent data from being read correctly.
TEST(HTTPReadTask, SampleClockDoesNotAffectData) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 99.9})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(50); // 50 Hz → 20ms period

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    for (int i = 0; i < 3; i++) {
        x::telem::Frame fr;
        auto res = source->read(breaker, fr);
        ASSERT_NIL(res.error);
        EXPECT_EQ(fr.size(), 1);
        EXPECT_NEAR(fr.at<double>(1, 0), 99.9, 0.001);
    }
    breaker.stop();
}

////////////////////// Connection-Level Query Parameters ///////////////////////

/// @brief it should pass connection-level query parameters to every request.
TEST(HTTPReadTask, ConnectionLevelQueryParams) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 42.0})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.name = "value", .data_type = x::telem::FLOAT64_T, .key = 1};

    auto source = ASSERT_NIL_P(make_source(
        cfg,
        server.base_url(),
        x::json::json{{"query_params", {{"api_key", "abc123"}, {"format", "json"}}}}
    ));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 42.0, 0.001);

    auto requests = server.received_requests();
    ASSERT_EQ(requests.size(), 1);
    auto &params = requests[0].query_params;
    EXPECT_EQ(params.find("api_key")->second, "abc123");
    EXPECT_EQ(params.find("format")->second, "json");
}

/// @brief string values should be converted to numbers using the enum map.
TEST(HTTPReadTask, EnumValuesMapStringsToNumbers) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/device",
                .status_code = 200,
                .response_body =
                    R"({"power": "ON", "mode": "STANDBY", "temperature": 22.5})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField power_field;
    power_field.pointer = x::json::json::json_pointer("/power");
    power_field.channel_key = 1;
    power_field.enum_values = {{"ON", 1.0}, {"OFF", 0.0}};

    ReadField mode_field;
    mode_field.pointer = x::json::json::json_pointer("/mode");
    mode_field.channel_key = 2;
    mode_field.enum_values = {{"AUTO", 0}, {"MANUAL", 1}, {"STANDBY", 2}};

    ReadField temp_field;
    temp_field.pointer = x::json::json::json_pointer("/temperature");
    temp_field.channel_key = 3;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/device";
    ep.body = "";
    ep.fields = {power_field, mode_field, temp_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .name = "power",
        .data_type = x::telem::FLOAT64_T,
        .key = 1,
    };
    cfg.channels[2] = {
        .name = "mode",
        .data_type = x::telem::INT32_T,
        .key = 2,
    };
    cfg.channels[3] = {
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
        .key = 3,
    };

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(fr.size(), 3);
    EXPECT_NEAR(fr.at<double>(1, 0), 1.0, 0.001);
    EXPECT_EQ(fr.at<int32_t>(2, 0), 2);
    EXPECT_NEAR(fr.at<double>(3, 0), 22.5, 0.001);
}

/// @brief a string not in the enum map and not numeric should produce a warning.
TEST(HTTPReadTask, EnumValuesMissingKeyWarns) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/device",
                .status_code = 200,
                .response_body = R"({"power": "UNKNOWN"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField power_field;
    power_field.pointer = x::json::json::json_pointer("/power");
    power_field.channel_key = 1;
    power_field.enum_values = {{"ON", 1.0}, {"OFF", 0.0}};

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/device";
    ep.body = "";
    ep.fields = {power_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .name = "power",
        .data_type = x::telem::FLOAT64_T,
        .key = 1,
    };

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    EXPECT_EQ(fr.size(), 0);
}

/// @brief fields without enum_values should still parse numeric strings normally.
TEST(HTTPReadTask, EnumValuesEmptyMapFallsBackToNumericParsing) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": "42.5"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving = false;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;
    // no enum_values set — should use normal string-to-numeric

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .name = "value",
        .data_type = x::telem::FLOAT64_T,
        .key = 1,
    };

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 42.5, 0.001);
}

}
