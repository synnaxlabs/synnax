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
std::pair<std::unique_ptr<ReadTaskSource>, x::errors::Error>
make_source(const ReadTaskConfig &cfg, const std::string &base_url) {
    auto conn_parser = x::json::Parser(
        x::json::json{
            {"base_url", base_url},
            {"timeout_ms", 1000},
        }
    );
    auto conn = device::ConnectionConfig(conn_parser, false);

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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

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
        .key = 1,
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T
    };
    cfg.channels[2] = {.key = 2, .name = "humidity", .data_type = x::telem::FLOAT64_T};

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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field;
    field.pointer = x::json::json::json_pointer("/data/sensors/0/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/sensors";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "sensor_0", .data_type = x::telem::FLOAT64_T};

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

/// @brief it should return PARSE_ERROR when a JSON pointer doesn't match.
TEST(HTTPReadTask, MissingJSONField) {
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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field;
    field.pointer = x::json::json::json_pointer("/nonexistent");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "missing", .data_type = x::telem::FLOAT64_T};

    auto source = ASSERT_NIL_P(make_source(cfg, server.base_url()));

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_OCCURRED_AS(res.error, errors::PARSE_ERROR);
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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "val", .data_type = x::telem::FLOAT64_T};

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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "val", .data_type = x::telem::FLOAT64_T};

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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

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

    cfg.channels[1] = {.key = 1, .name = "active", .data_type = x::telem::UINT8_T};
    cfg.channels[2] = {.key = 2, .name = "label", .data_type = x::telem::STRING_T};
    cfg.channels[3] = {.key = 3, .name = "count", .data_type = x::telem::INT32_T};

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

/// @brief it should use software timing (midpoint) for index channels when no
/// time_pointer is provided.
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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

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
        {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T, .index = 100};
    cfg.index_sources = {IndexSource{
        .index_key = 100,
        .endpoint_index = 0,
    }};

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

/// @brief it should extract timestamps from JSON response when time_pointer is set.
TEST(HTTPReadTask, TimestampExtractionFromResponse) {
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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

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
        {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T, .index = 100};
    cfg.index_sources = {IndexSource{
        .index_key = 100,
        .endpoint_index = 0,
        .time_info = TimeInfo{
            x::json::json::json_pointer("/timestamp"),
            x::json::TimeFormat::UnixSecond,
        },
    }};

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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

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

    cfg.channels[1] = {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T};
    cfg.channels[2] = {.key = 2, .name = "pressure", .data_type = x::telem::FLOAT64_T};

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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field;
    field.pointer = x::json::json::json_pointer("/result");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/query";
    ep.body = R"({"query": "latest"})";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "result", .data_type = x::telem::FLOAT64_T};

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

/// @brief it should construct TimeInfo from a valid JSON parser.
TEST(HTTPReadTask, TimeInfoParseValid) {
    auto parser = x::json::Parser(
        x::json::json{
            {"pointer", "/timestamp"},
            {"format", "unix_sec"},
        }
    );
    TimeInfo ti(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(ti.pointer.to_string(), "/timestamp");
    EXPECT_EQ(ti.format, x::json::TimeFormat::UnixSecond);
}

/// @brief it should report an error when TimeInfo has an invalid format.
TEST(HTTPReadTask, TimeInfoParseInvalidFormat) {
    auto parser = x::json::Parser(
        x::json::json{
            {"pointer", "/timestamp"},
            {"format", "bad_format"},
        }
    );
    TimeInfo ti(parser);
    EXPECT_FALSE(parser.ok());
}

/// @brief it should report an error when TimeInfo is missing the pointer field.
TEST(HTTPReadTask, TimeInfoParseMissingPointer) {
    auto parser = x::json::Parser(
        x::json::json{
            {"format", "iso8601"},
        }
    );
    TimeInfo ti(parser);
    EXPECT_FALSE(parser.ok());
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

/// @brief test fixture for parse tests that need a real Synnax client with
/// pre-created channels and device.
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

/// @brief it should error when a TIMESTAMP_T channel has no timestampFormat.
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

/// @brief it should error when two fields for the same index have conflicting
/// time pointers.
TEST_F(HTTPReadTaskParseTest, ConflictingTimestampSources) {
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
                  {
                      {"pointer", "/temp"},
                      {"channel", ch1.key},
                      {"timePointer", {{"pointer", "/ts1"}, {"format", "unix_sec"}}},
                  },
                  {
                      {"pointer", "/humidity"},
                      {"channel", ch2.key},
                      {"timePointer", {{"pointer", "/ts2"}, {"format", "unix_sec"}}},
                  },
              }},
         }}},
    };
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should not error when two fields for the same index have identical
/// time pointers.
TEST_F(HTTPReadTaskParseTest, SameIndexSamePointerOK) {
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
                  {
                      {"pointer", "/temp"},
                      {"channel", ch1.key},
                      {"timePointer",
                       {{"pointer", "/timestamp"}, {"format", "unix_sec"}}},
                  },
                  {
                      {"pointer", "/humidity"},
                      {"channel", ch2.key},
                      {"timePointer",
                       {{"pointer", "/timestamp"}, {"format", "unix_sec"}}},
                  },
              }},
         }}},
    };
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    EXPECT_EQ(cfg.index_sources.size(), 1);
    EXPECT_EQ(cfg.index_sources[0].index_key, idx.key);
    EXPECT_TRUE(cfg.index_sources[0].time_info.has_value());
}

/// @brief it should not error when the same index is referenced by multiple fields
/// where only some have time pointers.
TEST_F(HTTPReadTaskParseTest, SameIndexPartialTimePointerOK) {
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
                  {
                      {"pointer", "/temp"},
                      {"channel", ch1.key},
                      {"timePointer",
                       {{"pointer", "/timestamp"}, {"format", "unix_sec"}}},
                  },
                  {
                      {"pointer", "/humidity"},
                      {"channel", ch2.key},
                  },
              }},
         }}},
    };
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    EXPECT_EQ(cfg.index_sources.size(), 1);
    EXPECT_TRUE(cfg.index_sources[0].time_info.has_value());
}

/// @brief it should error when timestampFormat is set on a non-timestamp channel.
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
                  {"timestampFormat", "unix_sec"},
              }}},
         }}},
    };
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
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
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

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

}
