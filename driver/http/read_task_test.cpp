// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

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
    const std::string &base_url
) {
    auto conn_parser = x::json::Parser(
        x::json::json{
            {"base_url", base_url},
            {"timeout_ms", 1000},
        }
    );
    auto conn = device::ConnectionConfig(conn_parser, false);

    std::vector<device::RequestConfig> request_configs;
    request_configs.reserve(cfg.endpoints.size());
    for (const auto &ep : cfg.endpoints) request_configs.push_back(ep.request);

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
        {"endpoints", {{
            {"method", "GET"},
            {"path", "/api/data"},
            {"fields", {{
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
        {"endpoints", {{
            {"method", "GET"},
            {"path", "/api/data"},
            {"fields", {
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
    temp_field.ch.key = 1;
    temp_field.ch.name = "temperature";
    temp_field.ch.data_type = x::telem::FLOAT64_T;

    ReadField humidity_field;
    humidity_field.pointer = x::json::json::json_pointer("/humidity");
    humidity_field.channel_key = 2;
    humidity_field.ch.key = 2;
    humidity_field.ch.name = "humidity";
    humidity_field.ch.data_type = x::telem::FLOAT64_T;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field, humidity_field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1, 2};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
    field.ch.key = 1;
    field.ch.name = "sensor_0";
    field.ch.data_type = x::telem::FLOAT64_T;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/sensors";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
    field.ch.key = 1;
    field.ch.name = "missing";
    field.ch.data_type = x::telem::FLOAT64_T;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
    field.ch.key = 1;
    field.ch.name = "val";
    field.ch.data_type = x::telem::FLOAT64_T;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
    field.ch.key = 1;
    field.ch.name = "val";
    field.ch.data_type = x::telem::FLOAT64_T;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
    bool_field.ch.key = 1;
    bool_field.ch.name = "active";
    bool_field.ch.data_type = x::telem::UINT8_T;

    ReadField string_field;
    string_field.pointer = x::json::json::json_pointer("/label");
    string_field.channel_key = 2;
    string_field.ch.key = 2;
    string_field.ch.name = "label";
    string_field.ch.data_type = x::telem::STRING_T;

    ReadField int_field;
    int_field.pointer = x::json::json::json_pointer("/count");
    int_field.channel_key = 3;
    int_field.ch.key = 3;
    int_field.ch.name = "count";
    int_field.ch.data_type = x::telem::INT32_T;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {bool_field, string_field, int_field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1, 2, 3};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
    field.ch.key = 1;
    field.ch.name = "value";
    field.ch.data_type = x::telem::FLOAT64_T;
    field.ch.index = 100;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1, 100};
    cfg.index_keys = {100};
    cfg.index_sources = {IndexSource{
        .index_key = 100,
        .endpoint_index = 0,
    }};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
                .response_body =
                    R"({"value": 42.0, "timestamp": 1700000000})",
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
    field.ch.key = 1;
    field.ch.name = "value";
    field.ch.data_type = x::telem::FLOAT64_T;
    field.ch.index = 100;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1, 100};
    cfg.index_keys = {100};
    cfg.index_sources = {IndexSource{
        .index_key = 100,
        .endpoint_index = 0,
        .time_info = TimeInfo{
            x::json::json::json_pointer("/timestamp"),
            x::json::TimeFormat::UnixSecond,
        },
    }};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
    temp_field.ch.key = 1;
    temp_field.ch.name = "temp";
    temp_field.ch.data_type = x::telem::FLOAT64_T;

    ReadField pressure_field;
    pressure_field.pointer = x::json::json::json_pointer("/pressure");
    pressure_field.channel_key = 2;
    pressure_field.ch.key = 2;
    pressure_field.ch.name = "pressure";
    pressure_field.ch.data_type = x::telem::FLOAT64_T;

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
    cfg.all_channel_keys = {1, 2};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
    field.ch.key = 1;
    field.ch.name = "result";
    field.ch.data_type = x::telem::FLOAT64_T;

    ReadEndpoint ep;
    ep.request.method = Method::POST;
    ep.request.path = "/api/query";
    ep.body = R"({"query": "latest"})";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
    auto parser = x::json::Parser(x::json::json{
        {"pointer", "/timestamp"},
        {"format", "unix_sec"},
    });
    TimeInfo ti(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(ti.pointer.to_string(), "/timestamp");
    EXPECT_EQ(ti.format, x::json::TimeFormat::UnixSecond);
}

/// @brief it should report an error when TimeInfo has an invalid format.
TEST(HTTPReadTask, TimeInfoParseInvalidFormat) {
    auto parser = x::json::Parser(x::json::json{
        {"pointer", "/timestamp"},
        {"format", "bad_format"},
    });
    TimeInfo ti(parser);
    EXPECT_FALSE(parser.ok());
}

/// @brief it should report an error when TimeInfo is missing the pointer field.
TEST(HTTPReadTask, TimeInfoParseMissingPointer) {
    auto parser = x::json::Parser(x::json::json{
        {"format", "iso8601"},
    });
    TimeInfo ti(parser);
    EXPECT_FALSE(parser.ok());
}

/// @brief it should reject PUT method in read task config.
TEST(HTTPReadTask, ParseConfigRejectsPUT) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints", {{
            {"method", "PUT"},
            {"path", "/api/data"},
            {"fields", {{
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
        {"endpoints", {{
            {"method", "DELETE"},
            {"path", "/api/data"},
            {"fields", {{
                {"pointer", "/temp"},
                {"channel", 1},
            }}},
        }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should error when a TIMESTAMP_T channel has no timestampFormat.
TEST(HTTPReadTask, ValidateFieldsTimestampChannelMissingFormat) {
    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field;
    field.pointer = x::json::json::json_pointer("/ts");
    field.channel_key = 1;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.fields = {field};
    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1};

    synnax::channel::Channel ch;
    ch.key = 1;
    ch.name = "timestamp_ch";
    ch.data_type = x::telem::TIMESTAMP_T;
    std::map<synnax::channel::Key, synnax::channel::Channel> ch_map = {{1, ch}};

    auto err = cfg.validate_fields(ch_map);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
}

/// @brief it should error when two fields for the same index have conflicting
/// time pointers.
TEST(HTTPReadTask, ValidateFieldsConflictingTimestampSources) {
    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field1;
    field1.pointer = x::json::json::json_pointer("/temp");
    field1.channel_key = 1;
    field1.time_info = TimeInfo{
        x::json::json::json_pointer("/ts1"),
        x::json::TimeFormat::UnixSecond,
    };

    ReadField field2;
    field2.pointer = x::json::json::json_pointer("/humidity");
    field2.channel_key = 2;
    field2.time_info = TimeInfo{
        x::json::json::json_pointer("/ts2"),
        x::json::TimeFormat::UnixSecond,
    };

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.fields = {field1, field2};
    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1, 2};

    synnax::channel::Channel ch1;
    ch1.key = 1;
    ch1.name = "temp";
    ch1.data_type = x::telem::FLOAT64_T;
    ch1.index = 100;

    synnax::channel::Channel ch2;
    ch2.key = 2;
    ch2.name = "humidity";
    ch2.data_type = x::telem::FLOAT64_T;
    ch2.index = 100;

    std::map<synnax::channel::Key, synnax::channel::Channel> ch_map = {
        {1, ch1},
        {2, ch2},
    };

    auto err = cfg.validate_fields(ch_map);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
}

/// @brief it should not error when two fields for the same index have identical
/// time pointers.
TEST(HTTPReadTask, ValidateFieldsSameIndexSamePointerOK) {
    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field1;
    field1.pointer = x::json::json::json_pointer("/temp");
    field1.channel_key = 1;
    field1.time_info = TimeInfo{
        x::json::json::json_pointer("/timestamp"),
        x::json::TimeFormat::UnixSecond,
    };

    ReadField field2;
    field2.pointer = x::json::json::json_pointer("/humidity");
    field2.channel_key = 2;
    field2.time_info = TimeInfo{
        x::json::json::json_pointer("/timestamp"),
        x::json::TimeFormat::UnixSecond,
    };

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.fields = {field1, field2};
    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1, 2};

    synnax::channel::Channel ch1;
    ch1.key = 1;
    ch1.name = "temp";
    ch1.data_type = x::telem::FLOAT64_T;
    ch1.index = 100;

    synnax::channel::Channel ch2;
    ch2.key = 2;
    ch2.name = "humidity";
    ch2.data_type = x::telem::FLOAT64_T;
    ch2.index = 100;

    std::map<synnax::channel::Key, synnax::channel::Channel> ch_map = {
        {1, ch1},
        {2, ch2},
    };

    auto err = cfg.validate_fields(ch_map);
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.index_sources.size(), 1);
    EXPECT_EQ(cfg.index_sources[0].index_key, 100u);
    EXPECT_TRUE(cfg.index_sources[0].time_info.has_value());
}

/// @brief it should not error when the same index is referenced by multiple fields
/// where only some have time pointers.
TEST(HTTPReadTask, ValidateFieldsSameIndexPartialTimePointerOK) {
    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field1;
    field1.pointer = x::json::json::json_pointer("/temp");
    field1.channel_key = 1;
    field1.time_info = TimeInfo{
        x::json::json::json_pointer("/timestamp"),
        x::json::TimeFormat::UnixSecond,
    };

    ReadField field2;
    field2.pointer = x::json::json::json_pointer("/humidity");
    field2.channel_key = 2;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.fields = {field1, field2};
    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1, 2};

    synnax::channel::Channel ch1;
    ch1.key = 1;
    ch1.name = "temp";
    ch1.data_type = x::telem::FLOAT64_T;
    ch1.index = 100;

    synnax::channel::Channel ch2;
    ch2.key = 2;
    ch2.name = "humidity";
    ch2.data_type = x::telem::FLOAT64_T;
    ch2.index = 100;

    std::map<synnax::channel::Key, synnax::channel::Channel> ch_map = {
        {1, ch1},
        {2, ch2},
    };

    auto err = cfg.validate_fields(ch_map);
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.index_sources.size(), 1);
    EXPECT_TRUE(cfg.index_sources[0].time_info.has_value());
}

/// @brief it should error when timestampFormat is set on a non-timestamp channel.
TEST(HTTPReadTask, ValidateFieldsTimestampFormatOnNonTimestamp) {
    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.rate = x::telem::Rate(10);
    cfg.strict = false;

    ReadField field;
    field.pointer = x::json::json::json_pointer("/value");
    field.channel_key = 1;
    field.time_format = x::json::TimeFormat::UnixSecond;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.fields = {field};
    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1};

    synnax::channel::Channel ch;
    ch.key = 1;
    ch.name = "value";
    ch.data_type = x::telem::FLOAT64_T;
    std::map<synnax::channel::Key, synnax::channel::Channel> ch_map = {{1, ch}};

    auto err = cfg.validate_fields(ch_map);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
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
    field.ch.key = 1;
    field.ch.name = "value";
    field.ch.data_type = x::telem::FLOAT64_T;

    ReadEndpoint ep;
    ep.request.method = Method::GET;
    ep.request.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.all_channel_keys = {1};

    auto [source, err] = make_source(cfg, server.base_url());
    ASSERT_NIL(err);

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
