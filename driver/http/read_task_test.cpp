// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <limits>
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/read_task.h"
#include "driver/http/testutil/server.h"

namespace driver::http {
using ::synnax::http::BooleanReadField;
using ::synnax::http::FloatReadField;
using ::synnax::http::IntegerReadField;
using ::synnax::http::ReadEndpoint;
using ::synnax::http::StringReadField;
using ::synnax::http::TimestampReadField;
using ::synnax::http::UnsignedReadField;

namespace {
/// @brief helper to build a ReadTaskSource from config and a mock server URL.
/// Each call creates its own Processor so tests have independent lifecycles.
std::pair<std::unique_ptr<ReadTaskSource>, std::shared_ptr<Processor>> make_source(
    ReadTaskConfig &cfg,
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

    auto requests = build_requests(conn, cfg.endpoints);

    auto processor = std::make_shared<Processor>();
    return {
        std::make_unique<ReadTaskSource>(
            ReadTaskConfig(cfg),
            processor,
            std::move(requests)
        ),
        std::move(processor),
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
                  {"type", "float"},
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
                  {{"type", "float"}, {"pointer", "/temp"}, {"channel", 1}},
                  {{"type", "float"}, {"pointer", "/humidity"}, {"channel", 1}},
              }},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief it should fail when a query_params entry is missing the parameter field.
TEST(HTTPReadTask, ParseConfigQueryParamMissingParameterErrors) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"query_params", {{{"value", "10"}}}},
             {"fields",
              {{
                  {"type", "float"},
                  {"pointer", "/value"},
                  {"channel", 1},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto [_, err] = ReadTaskConfig::parse(ctx, task);
    ASSERT_TRUE(err.matches(x::errors::VALIDATION));
    EXPECT_NE(err.data.find("parameter"), std::string::npos);
}

/// @brief a query_params entry may omit the value field; it defaults to an empty
/// string.
TEST(HTTPReadTask, ParseConfigQueryParamMissingValueDefaultsEmpty) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
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
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"query_params", {{{"parameter", "limit"}}}},
             {"fields",
              {{
                  {"type", "float"},
                  {"pointer", "/value"},
                  {"channel", ch.key},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(client);
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    ASSERT_EQ(cfg.endpoints[0].query_params.size(), 1);
    EXPECT_EQ(cfg.endpoints[0].query_params.at(0).parameter, "limit");
    EXPECT_EQ(cfg.endpoints[0].query_params.at(0).value, "");
}

namespace {
/// @brief a data channel and its index on a live Core.
struct FieldChannels {
    std::shared_ptr<synnax::Synnax> client;
    synnax::channel::Channel index;
    synnax::channel::Channel data;
};

/// @brief creates an indexed channel of the given data type.
/// @param data_type the data type of the data channel.
/// @returns the client and the created channels.
FieldChannels create_field_channels(const x::telem::DataType &data_type) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto index = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("idx"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto data = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("value"), data_type, index.key, false)
    );
    return {std::move(client), std::move(index), std::move(data)};
}

/// @brief creates a read task with one field on one endpoint.
/// @param field the JSON config of the field.
/// @returns the task.
synnax::task::Task create_field_task(const x::json::json &field) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields", x::json::json::array({field})},
         }}},
    };
    return task;
}

/// @brief asserts that a task with the given field fails to parse.
/// @param client the client that the parse retrieves channels with.
/// @param field the JSON config of the field.
/// @param message text that the error must hold.
void expect_field_rejected(
    const std::shared_ptr<synnax::Synnax> &client,
    const x::json::json &field,
    const std::string &message
) {
    auto ctx = std::make_shared<task::MockContext>(client);
    auto [_, err] = ReadTaskConfig::parse(ctx, create_field_task(field));
    ASSERT_TRUE(err.matches(x::errors::VALIDATION));
    EXPECT_NE(err.data.find(message), std::string::npos) << err.data;
}
}

/// @brief it should reject a field whose type does not match the kind of the data type
/// of its channel.
TEST(HTTPReadTask, ParseConfigKindMismatchErrors) {
    const auto [client, _, data] = create_field_channels(x::telem::INT32_T);
    expect_field_rejected(
        client,
        {{"type", "float"}, {"pointer", "/value"}, {"channel", data.key}},
        "the field type must be integer, not float"
    );
}

/// @brief it should reject a fill that does not fit the channel.
TEST(HTTPReadTask, ParseConfigFillOutOfRangeErrors) {
    const auto [client, _, data] = create_field_channels(x::telem::INT8_T);
    expect_field_rejected(
        client,
        {{"type", "integer"},
         {"pointer", "/value"},
         {"channel", data.key},
         {"fill", "200"}},
        "out of bounds"
    );
}

/// @brief it should reject a fill on an index channel, because a filled timestamp
/// would corrupt the index.
TEST(HTTPReadTask, ParseConfigIndexFillErrors) {
    const auto [client, index, _] = create_field_channels(x::telem::FLOAT64_T);
    expect_field_rejected(
        client,
        {{"type", "timestamp"},
         {"pointer", "/time"},
         {"channel", index.key},
         {"time_format", "unix_sec"},
         {"fill", "0"}},
        "index channel"
    );
}

/// @brief it should reject a JSON, UUID, or bytes channel.
TEST(HTTPReadTask, ParseConfigUnsupportedDataTypeErrors) {
    for (const auto &dt: {x::telem::JSON_T, x::telem::UUID_T, x::telem::BYTES_T}) {
        const auto [client, _, data] = create_field_channels(dt);
        expect_field_rejected(
            client,
            {{"type", dt.name()}, {"pointer", "/value"}, {"channel", data.key}},
            "unsupported data type"
        );
    }
}

/// @brief it should require a value for a field with no fill.
TEST(HTTPReadTask, ParseConfigNoFillIsRequired) {
    const auto [client, _, data] = create_field_channels(x::telem::FLOAT64_T);
    auto ctx = std::make_shared<task::MockContext>(client);
    const auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(
            ctx,
            create_field_task(
                {{"type", "float"}, {"pointer", "/value"}, {"channel", data.key}}
            )
        )
    );
    EXPECT_TRUE(cfg.fill_values.empty());
}

/// @brief it should convert a float fill to the data type of the channel.
TEST(HTTPReadTask, ParseConfigFloatFillConverts) {
    const auto [client, _, data] = create_field_channels(x::telem::FLOAT32_T);
    auto ctx = std::make_shared<task::MockContext>(client);
    const auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(
            ctx,
            create_field_task(
                {{"type", "float"},
                 {"pointer", "/value"},
                 {"channel", data.key},
                 {"fill", 0}}
            )
        )
    );
    EXPECT_EQ(std::get<float>(cfg.fill_values.at(data.key)), 0.0F);
}

/// @brief it should convert a NaN fill to a NaN of the data type of the channel.
TEST(HTTPReadTask, ParseConfigNaNFillConverts) {
    const auto [client, _, data] = create_field_channels(x::telem::FLOAT32_T);
    auto ctx = std::make_shared<task::MockContext>(client);
    const auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(
            ctx,
            create_field_task(
                {{"type", "float"},
                 {"pointer", "/value"},
                 {"channel", data.key},
                 {"fill", "NaN"}}
            )
        )
    );
    EXPECT_TRUE(std::isnan(std::get<float>(cfg.fill_values.at(data.key))));
}

/// @brief it should keep every digit of an integer fill.
TEST(HTTPReadTask, ParseConfigIntegerFillIsExact) {
    const auto [client, _, data] = create_field_channels(x::telem::INT64_T);
    auto ctx = std::make_shared<task::MockContext>(client);
    const auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(
            ctx,
            create_field_task(
                {{"type", "integer"},
                 {"pointer", "/value"},
                 {"channel", data.key},
                 {"fill", "9007199254740993"}}
            )
        )
    );
    EXPECT_EQ(
        std::get<int64_t>(cfg.fill_values.at(data.key)),
        int64_t{9007199254740993}
    );
}

/// @brief it should keep a timestamp fill as it is.
TEST(HTTPReadTask, ParseConfigTimestampFill) {
    const auto [client, _, data] = create_field_channels(x::telem::TIMESTAMP_T);
    auto ctx = std::make_shared<task::MockContext>(client);
    const auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(
            ctx,
            create_field_task(
                {{"type", "timestamp"},
                 {"pointer", "/value"},
                 {"channel", data.key},
                 {"time_format", "unix_sec"},
                 {"fill", "2000000000"}}
            )
        )
    );
    EXPECT_EQ(
        std::get<x::telem::TimeStamp>(cfg.fill_values.at(data.key)),
        x::telem::TimeStamp(2 * x::telem::SECOND)
    );
}

/// @brief it should fail when an enum_values entry is missing the label field.
TEST(HTTPReadTask, ParseConfigEnumValueMissingLabelErrors) {
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
                  {"type", "float"},
                  {"pointer", "/status"},
                  {"channel", 1},
                  {"enum_values", {{{"value", 1.0}}}},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto [_1, err1] = ReadTaskConfig::parse(ctx, task);
    ASSERT_TRUE(err1.matches(x::errors::VALIDATION));
    EXPECT_NE(err1.data.find("label"), std::string::npos);
}

/// @brief it should fail when duplicate header names exist.
TEST(HTTPReadTask, ParseConfigDuplicateHeaderErrors) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"headers",
              {
                  {{"name", "Accept"}, {"value", "text/plain"}},
                  {{"name", "Accept"}, {"value", "application/json"}},
              }},
             {"fields",
              {{
                  {"type", "float"},
                  {"pointer", "/value"},
                  {"channel", 1},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto [_3, err3] = ReadTaskConfig::parse(ctx, task);
    ASSERT_TRUE(err3.matches(x::errors::VALIDATION));
    EXPECT_NE(err3.data.find("duplicate header"), std::string::npos);
}

/// @brief it should parse an endpoint that omits query parameters, leaving it
/// with none.
TEST(HTTPReadTask, ParseConfigOmittedQueryParamsDefaultsEmpty) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
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
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"type", "float"},
                  {"pointer", "/value"},
                  {"channel", ch.key},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(client);
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    ASSERT_EQ(cfg.endpoints.size(), 1);
    ASSERT_TRUE(cfg.endpoints[0].query_params.empty());
}

/// @brief it should parse an endpoint that omits headers, leaving it with none.
TEST(HTTPReadTask, ParseConfigOmittedHeadersDefaultsEmpty) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
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
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"type", "float"},
                  {"pointer", "/value"},
                  {"channel", ch.key},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(client);
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    ASSERT_EQ(cfg.endpoints.size(), 1);
    ASSERT_TRUE(cfg.endpoints[0].headers.empty());
}

/// @brief an endpoint that omits body should parse to an empty body.
TEST(HTTPReadTask, ParseConfigOmittedBodyDefaultsEmpty) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
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
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"type", "float"},
                  {"pointer", "/value"},
                  {"channel", ch.key},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(client);
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    ASSERT_EQ(cfg.endpoints.size(), 1);
    ASSERT_TRUE(cfg.endpoints[0].body.empty());
}

/// @brief an endpoint that omits index should parse to an empty index.
TEST(HTTPReadTask, ParseConfigOmittedIndexDefaultsEmpty) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
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
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"type", "float"},
                  {"pointer", "/value"},
                  {"channel", ch.key},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(client);
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    ASSERT_EQ(cfg.endpoints.size(), 1);
    ASSERT_TRUE(cfg.endpoints[0].index.empty());
}

/// @brief a field that omits enum_values should parse to an empty list.
TEST(HTTPReadTask, ParseConfigOmittedEnumValuesDefaultsEmpty) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
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
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields",
              {{
                  {"type", "float"},
                  {"pointer", "/value"},
                  {"channel", ch.key},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(client);
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    ASSERT_EQ(cfg.endpoints.size(), 1);
    ASSERT_EQ(cfg.endpoints[0].fields.size(), 1);
    ASSERT_TRUE(
        std::get<FloatReadField>(cfg.endpoints[0].fields[0]).enum_values.empty()
    );
}

/// @brief it should fail when duplicate query parameter names exist.
TEST(HTTPReadTask, ParseConfigDuplicateQueryParamErrors) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"query_params",
              {
                  {{"parameter", "limit"}, {"value", "10"}},
                  {{"parameter", "limit"}, {"value", "20"}},
              }},
             {"fields",
              {{
                  {"type", "float"},
                  {"pointer", "/value"},
                  {"channel", 1},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto [_4, err4] = ReadTaskConfig::parse(ctx, task);
    ASSERT_TRUE(err4.matches(x::errors::VALIDATION));
    EXPECT_NE(err4.data.find("duplicate query parameter"), std::string::npos);
}

/// @brief it should extract a numeric field from a single GET endpoint.
TEST(HTTPReadTask, SingleEndpointGETNumericField) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temperature";
    temp_field.channel = 1;

    FloatReadField humidity_field;
    humidity_field.pointer = "/humidity";
    humidity_field.channel = 2;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field, humidity_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .key = 1,
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
    };
    cfg.channels[2] = {.key = 2, .name = "humidity", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/data/sensors/0/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/sensors";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "sensor_0", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/nonexistent";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "missing", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    EXPECT_EQ(fr.size(), 0);
}

/// @brief it should convert JSON types correctly (bool to uint8, string to string).
TEST(HTTPReadTask, TypeConversions) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    UnsignedReadField bool_field;
    bool_field.pointer = "/active";
    bool_field.channel = 1;

    StringReadField string_field;
    string_field.pointer = "/label";
    string_field.channel = 2;

    IntegerReadField int_field;
    int_field.pointer = "/count";
    int_field.channel = 3;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {bool_field, string_field, int_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "active", .data_type = x::telem::UINT8_T};
    cfg.channels[2] = {.key = 2, .name = "label", .data_type = x::telem::STRING_T};
    cfg.channels[3] = {.key = 3, .name = "count", .data_type = x::telem::INT32_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    StringReadField name_field;
    name_field.pointer = "/name";
    name_field.channel = 1;

    StringReadField status_field;
    status_field.pointer = "/status";
    status_field.channel = 2;

    FloatReadField value_field;
    value_field.pointer = "/value";
    value_field.channel = 3;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {name_field, status_field, value_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "name", .data_type = x::telem::STRING_T};
    cfg.channels[2] = {.key = 2, .name = "status", .data_type = x::telem::STRING_T};
    cfg.channels[3] = {.key = 3, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    IntegerReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "count", .data_type = x::telem::INT32_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    UnsignedReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "count", .data_type = x::telem::UINT32_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] =
        {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T, .index = 100};
    cfg.software_timed_indexes[100] = 0;

    auto [source, processor] = make_source(cfg, server.base_url());

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
/// channel is listed as an explicit field with a time_format.
TEST(HTTPReadTask, ExplicitIndexFieldTimestamp) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField data_field;
    data_field.pointer = "/value";
    data_field.channel = 1;

    TimestampReadField index_field;
    index_field.pointer = "/timestamp";
    index_field.channel = 100;
    index_field.time_format = "unix_sec";

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {data_field, index_field};

    cfg.endpoints = {ep};

    cfg.channels[1] =
        {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T, .index = 100};
    cfg.channels[100] = {
        .key = 100,
        .name = "time",
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    // No index_sources needed — index channel is an explicit field.

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;

    FloatReadField pressure_field;
    pressure_field.pointer = "/pressure";
    pressure_field.channel = 2;

    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/temp";
    ep1.body = "";
    ep1.fields = {temp_field};

    ReadEndpoint ep2;
    ep2.method = "GET";
    ep2.path = "/api/pressure";
    ep2.body = "";
    ep2.fields = {pressure_field};

    cfg.endpoints = {ep1, ep2};

    cfg.channels[1] = {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T};
    cfg.channels[2] = {.key = 2, .name = "pressure", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/result";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "POST";
    ep.path = "/api/query";
    ep.body = R"({"query": "latest"})";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "result", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 99.9, 0.001);
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
        x::json::json props = {
            {"secure", false},
            {"timeout_ms", 1000},
            {"verify_ssl", false},
        };
        synnax::device::Device dev{
            .key = device_key,
            .rack = rack.key,
            .location = "localhost:0",
            .make = "http",
            .model = "HTTP Device",
            .name = "HTTP Read Test Device",
            .properties = props.get<x::json::json::object_t>(),
        };
        ASSERT_NIL(client->devices.create(dev));
        ctx = std::make_shared<task::MockContext>(client);
    }
};

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
                 {"fields",
                  {{{"type", "float"}, {"pointer", "/temp"}, {"channel", ch1.key}}}},
             },
             {
                 {"method", "GET"},
                 {"path", "/api/humidity"},
                 {"fields",
                  {{{"type", "float"},
                    {"pointer", "/humidity"},
                    {"channel", ch2.key}}}},
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
                  {{"type", "float"}, {"pointer", "/temp"}, {"channel", ch1.key}},
                  {{"type", "float"}, {"pointer", "/humidity"}, {"channel", ch2.key}},
                  {
                      {"type", "timestamp"},
                      {"pointer", "/timestamp"},
                      {"channel", idx.key},
                      {"time_format", "unix_sec"},
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
                  {{"type", "float"}, {"pointer", "/temp"}, {"channel", ch1.key}},
                  {{"type", "float"}, {"pointer", "/humidity"}, {"channel", ch2.key}},
              }},
         }}},
    };
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    EXPECT_EQ(cfg.software_timed_indexes.size(), 1);
    EXPECT_TRUE(cfg.software_timed_indexes.count(idx.key));
}

/// @brief parse() should drop endpoints where every field is disabled and only
/// keep endpoints that have at least one enabled field.
TEST_F(HTTPReadTaskParseTest, DisabledEndpointFilteredOut) {
    auto idx1 = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("idx1"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto idx2 = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("idx2"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto idx3 = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("idx3"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("temp"),
        x::telem::FLOAT64_T,
        idx1.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("humidity"),
        x::telem::FLOAT64_T,
        idx2.key,
        false
    ));
    auto ch3 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("pressure"),
        x::telem::FLOAT64_T,
        idx3.key,
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
                 {"fields",
                  {{{"type", "float"}, {"pointer", "/temp"}, {"channel", ch1.key}}}},
             },
             {
                 {"method", "GET"},
                 {"path", "/api/humidity"},
                 {"fields",
                  {{
                      {"type", "float"},
                      {"pointer", "/humidity"},
                      {"channel", ch2.key},
                      {"disabled", true},
                  }}},
             },
             {
                 {"method", "GET"},
                 {"path", "/api/pressure"},
                 {"fields",
                  {{{"type", "float"},
                    {"pointer", "/pressure"},
                    {"channel", ch3.key}}}},
             },
         }},
    };
    auto cfg = ASSERT_NIL_P(ReadTaskConfig::parse(ctx, task));
    EXPECT_EQ(cfg.endpoints.size(), 2);
    EXPECT_EQ(cfg.endpoints[0].path, "/api/temp");
    EXPECT_EQ(cfg.endpoints[1].path, "/api/pressure");
}

/// @brief it should successfully read 10 times in succession from the same endpoint.
TEST(HTTPReadTask, RepeatedReads) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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

namespace {
/// @brief builds a config with one numeric field per path, all on channel keys 1..N.
ReadTaskConfig make_multi_endpoint_cfg(
    const std::vector<std::string> &paths,
    const x::telem::Rate rate
) {
    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = rate;
    for (std::size_t i = 0; i < paths.size(); i++) {
        const auto key = static_cast<synnax::channel::Key>(i + 1);
        FloatReadField field;
        field.pointer = "/value";
        field.channel = key;
        ReadEndpoint ep;
        ep.method = "GET";
        ep.path = paths[i];
        ep.body = "";
        ep.fields = {field};
        cfg.endpoints.push_back(ep);
        cfg.channels[key] = {
            .key = key,
            .name = "value" + std::to_string(key),
            .data_type = x::telem::FLOAT64_T,
        };
    }
    return cfg;
}

testutil::Route value_route(const std::string &path, const x::telem::TimeSpan &delay) {
    return {
        .method = Method::GET,
        .path = path,
        .status_code = 200,
        .response_body = R"({"value": 1.0})",
        .delay = delay,
    };
}

/// @brief builds a GET route that answers with the given status and body.
/// @param path the route path.
/// @param status_code the HTTP status code to answer with.
/// @param body the response body to answer with.
/// @returns the route.
testutil::Route
status_route(const std::string &path, const int status_code, const std::string &body) {
    return {
        .method = Method::GET,
        .path = path,
        .status_code = status_code,
        .response_body = body,
    };
}

/// @brief polls the config once against the server.
common::ReadResult
read_once(ReadTaskConfig &cfg, const testutil::Server &server, x::telem::Frame &fr) {
    auto [source, processor] = make_source(cfg, server.base_url());
    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    auto res = source->read(breaker, fr);
    breaker.stop();
    return res;
}
}

/// @brief every endpoint rejected with a 4xx status should return a temporary error
/// naming each status, so the pipeline backs off instead of stopping the task.
TEST(HTTPReadTask, AllEndpoints4xxBacksOff) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                status_route("/a", 401, "unauthorized"),
                status_route("/b", 403, "forbidden"),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto cfg = make_multi_endpoint_cfg({"/a", "/b"}, x::telem::Rate(10000));
    x::telem::Frame fr;
    const auto res = read_once(cfg, server, fr);
    ASSERT_OCCURRED_AS(res.error, errors::TEMPORARY_ERROR);
    EXPECT_FALSE(res.error.matches(errors::UNREACHABLE_ERROR));
    EXPECT_FALSE(res.error.matches(errors::CRITICAL_ERROR));
    EXPECT_EQ(fr.size(), 0);
    EXPECT_NE(res.error.data.find("GET"), std::string::npos);
    EXPECT_NE(res.error.data.find("/a returned 401: unauthorized"), std::string::npos);
    EXPECT_NE(res.error.data.find("/b returned 403: forbidden"), std::string::npos);
}

/// @brief every endpoint rejected with a 5xx status should return a temporary error,
/// so the pipeline breaker backs off.
TEST(HTTPReadTask, AllEndpoints5xxBacksOff) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                status_route("/a", 503, "unavailable"),
                status_route("/b", 503, "unavailable"),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto cfg = make_multi_endpoint_cfg({"/a", "/b"}, x::telem::Rate(10000));
    x::telem::Frame fr;
    const auto res = read_once(cfg, server, fr);
    ASSERT_OCCURRED_AS(res.error, errors::TEMPORARY_ERROR);
    EXPECT_FALSE(res.error.matches(errors::UNREACHABLE_ERROR));
    EXPECT_EQ(fr.size(), 0);
    EXPECT_NE(res.error.data.find("/a returned 503"), std::string::npos);
    EXPECT_NE(res.error.data.find("/b returned 503"), std::string::npos);
}

/// @brief a mix of 4xx and 5xx rejections on every endpoint should return a temporary
/// error carrying each code.
TEST(HTTPReadTask, MixedStatusFailuresBackOff) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                status_route("/a", 401, "unauthorized"),
                status_route("/b", 503, "unavailable"),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto cfg = make_multi_endpoint_cfg({"/a", "/b"}, x::telem::Rate(10000));
    x::telem::Frame fr;
    const auto res = read_once(cfg, server, fr);
    ASSERT_OCCURRED_AS(res.error, errors::TEMPORARY_ERROR);
    EXPECT_EQ(fr.size(), 0);
    EXPECT_NE(res.error.data.find("401"), std::string::npos);
    EXPECT_NE(res.error.data.find("503"), std::string::npos);
}

/// @brief a rejected endpoint beside a working one should stay a warning carrying the
/// start of the body with its line breaks escaped, and the working endpoint's data
/// should come through.
TEST(HTTPReadTask, PartialStatusFailureWarnsWithOneLineBody) {
    std::string body = "<html>\n  <body>\n";
    for (int i = 0; i < 40; i++)
        body += "    <p>line " + std::to_string(i) + "</p>\n";
    body += "  </body>\n</html>";
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                status_route("/a", 401, body),
                value_route("/b", x::telem::TimeSpan::ZERO()),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto cfg = make_multi_endpoint_cfg({"/a", "/b"}, x::telem::Rate(10000));
    x::telem::Frame fr;
    const auto res = read_once(cfg, server, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(2, 0), 1.0, 0.001);
    EXPECT_NE(
        res.warning.description.find(
            "/a returned 401: <html>\\n  <body>\\n    <p>line 0</p>"
        ),
        std::string::npos
    );
    EXPECT_EQ(res.warning.description.find("line 39"), std::string::npos);
    EXPECT_TRUE(res.warning.description.ends_with("..."));
    EXPECT_EQ(res.warning.description.find('\n'), std::string::npos);
}

/// @brief endpoints not sent because the device was unreachable should collapse into
/// one line of the unreachable error.
TEST(HTTPReadTask, SkippedEndpointsCollapseIntoOneLine) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                value_route("/slow", 2 * x::telem::SECOND),
                value_route("/a", x::telem::TimeSpan::ZERO()),
                value_route("/b", x::telem::TimeSpan::ZERO()),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto cfg = make_multi_endpoint_cfg({"/slow", "/a", "/b"}, x::telem::Rate(10000));
    auto [source, processor] = make_source(
        cfg,
        server.base_url(),
        {{"timeout_ms", 100}, {"max_concurrent_requests", 1}}
    );

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_OCCURRED_AS(res.error, errors::UNREACHABLE_ERROR);
    EXPECT_EQ(fr.size(), 0);
    EXPECT_NE(res.error.data.find("/slow"), std::string::npos);
    EXPECT_NE(res.error.data.find("did not send 2 requests"), std::string::npos);
    EXPECT_EQ(res.error.data.find("/a"), std::string::npos);
    EXPECT_EQ(res.error.data.find("at a time"), std::string::npos);
}

/// @brief more endpoints than slots overrunning the period should warn about the
/// cap.
TEST(HTTPReadTask, CapWarningWhenPollOverrunsPeriod) {
    const auto delay = 30 * x::telem::MILLISECOND;
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                value_route("/a", delay),
                value_route("/b", delay),
                value_route("/c", delay),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto cfg = make_multi_endpoint_cfg({"/a", "/b", "/c"}, x::telem::Rate(100));
    auto [source, processor] = make_source(
        cfg,
        server.base_url(),
        {{"max_concurrent_requests", 1}}
    );

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 3);
    EXPECT_EQ(res.warning.message, "Falling behind poll rate");
    EXPECT_EQ(
        res.warning.description,
        "Polling 3 endpoints 1 at a time takes longer than the 10ms read period. "
        "Lower the rate or raise the max concurrent requests in the device properties."
    );
}

/// @brief more endpoints than slots finishing within the period should not warn.
TEST(HTTPReadTask, NoCapWarningWithinPeriod) {
    const auto delay = 30 * x::telem::MILLISECOND;
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                value_route("/a", delay),
                value_route("/b", delay),
                value_route("/c", delay),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto cfg = make_multi_endpoint_cfg({"/a", "/b", "/c"}, x::telem::Rate(2));
    auto [source, processor] = make_source(
        cfg,
        server.base_url(),
        {{"max_concurrent_requests", 1}}
    );

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 3);
    EXPECT_TRUE(res.warning.empty());
}

/// @brief it should skip disabled fields and only return enabled ones.
TEST(HTTPReadTask, DisabledFieldsSkipped) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temperature";
    temp_field.channel = 1;
    temp_field.disabled = false;

    FloatReadField humidity_field;
    humidity_field.pointer = "/humidity";
    humidity_field.channel = 2;
    humidity_field.disabled = true;

    FloatReadField pressure_field;
    pressure_field.pointer = "/pressure";
    pressure_field.channel = 3;
    pressure_field.disabled = false;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field, humidity_field, pressure_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .key = 1,
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
    };
    cfg.channels[3] = {.key = 3, .name = "pressure", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField enabled_field;
    enabled_field.pointer = "/temp";
    enabled_field.channel = 1;
    enabled_field.disabled = false;

    FloatReadField disabled_field;
    disabled_field.pointer = "/humidity";
    disabled_field.channel = 2;
    disabled_field.disabled = true;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {enabled_field, disabled_field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {
        .key = 1,
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
    };

    auto conn_parser = x::json::Parser(
        x::json::json{
            {"base_url", "http://localhost:9999"},
            {"timeout_ms", 100},
            {"verify_ssl", false},
        }
    );
    auto conn = device::ConnectionConfig(conn_parser);

    std::vector<Request> requests;
    for (const auto &e: cfg.endpoints) {
        auto req = device::build_request(conn, request_config(e));
        req.body = e.body;
        requests.push_back(std::move(req));
    }

    auto processor = std::make_shared<Processor>();
    ReadTaskSource source(std::move(cfg), processor, std::move(requests));
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
                  {"type", "float"},
                  {"pointer", "/temp"},
                  {"channel", 1},
                  {"disabled", true},
              }}},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief an endpoint with no fields at all should produce a validation error.
TEST(HTTPReadTask, ParseConfigEndpointWithNoFields) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {{
             {"method", "GET"},
             {"path", "/api/data"},
             {"fields", x::json::json::array()},
         }}},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief when all fields on every endpoint are disabled, parsing should fail
/// because the task has no enabled fields at all.
TEST(HTTPReadTask, ParseConfigAllEndpointsAllFieldsDisabled) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"rate", 1.0},
        {"endpoints",
         {
             {
                 {"method", "GET"},
                 {"path", "/api/temp"},
                 {"fields",
                  {{
                      {"type", "float"},
                      {"pointer", "/temp"},
                      {"channel", 1},
                      {"disabled", true},
                  }}},
             },
             {
                 {"method", "GET"},
                 {"path", "/api/pressure"},
                 {"fields",
                  {{
                      {"type", "float"},
                      {"pointer", "/pressure"},
                      {"channel", 2},
                      {"disabled", true},
                  }}},
             },
         }},
    };
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(ctx, task), x::errors::VALIDATION);
}

/// @brief when only the second of three endpoints has all fields disabled, the
/// disabled endpoint should be skipped and the other two should be requested.
TEST(HTTPReadTask, MiddleEndpointAllFieldsDisabledSkipped) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;
    temp_field.disabled = false;

    FloatReadField pressure_field;
    pressure_field.pointer = "/pressure";
    pressure_field.channel = 3;
    pressure_field.disabled = false;

    // ep1 and ep3 have enabled fields; ep2 (middle) is entirely disabled and
    // excluded from the config — parse() would filter it out.
    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/temp";
    ep1.body = "";
    ep1.fields = {temp_field};

    ReadEndpoint ep3;
    ep3.method = "GET";
    ep3.path = "/api/pressure";
    ep3.body = "";
    ep3.fields = {pressure_field};

    cfg.endpoints = {ep1, ep3};

    cfg.channels[1] = {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T};
    cfg.channels[3] = {.key = 3, .name = "pressure", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(1, 0), 25.0, 0.001);
    EXPECT_NEAR(fr.at<double>(3, 0), 1013.25, 0.001);
}

/// @brief disabled fields should not cause a missing-field error even if the
/// JSON pointer would not match the response.
TEST(HTTPReadTask, DisabledFieldMissingPointerNoError) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField enabled_field;
    enabled_field.pointer = "/temperature";
    enabled_field.channel = 1;
    enabled_field.disabled = false;

    FloatReadField disabled_field;
    disabled_field.pointer = "/nonexistent";
    disabled_field.channel = 2;
    disabled_field.disabled = true;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {enabled_field, disabled_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .key = 1,
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
    };

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 23.5, 0.001);
}

/// @brief it should read from an HTTPS server with SSL verification disabled.
TEST(HTTPReadTask, HTTPSReadSingleEndpoint) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 77.7})",
            }},
            .secure = true,
            .cert_path = "driver/http/testutil/test_cert.pem",
            .key_path = "driver/http/testutil/test_key.pem",
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
            .cert_path = "driver/http/testutil/test_cert.pem",
            .key_path = "driver/http/testutil/test_key.pem",
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;

    FloatReadField pressure_field;
    pressure_field.pointer = "/pressure";
    pressure_field.channel = 2;

    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/temp";
    ep1.body = "";
    ep1.fields = {temp_field};

    ReadEndpoint ep2;
    ep2.method = "GET";
    ep2.path = "/api/pressure";
    ep2.body = "";
    ep2.fields = {pressure_field};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T};
    cfg.channels[2] = {.key = 2, .name = "pressure", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"value": 55.5})",
            }},
            .secure = true,
            .cert_path = "driver/http/testutil/test_cert.pem",
            .key_path = "driver/http/testutil/test_key.pem",
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/query",
                .status_code = 200,
                .response_body = R"({"result": 88.8})",
            }},
            .secure = true,
            .cert_path = "driver/http/testutil/test_cert.pem",
            .key_path = "driver/http/testutil/test_key.pem",
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/result";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "POST";
    ep.path = "/api/query";
    ep.body = R"({"query": "latest"})";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "result", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 88.8, 0.001);
}

/// @brief a timeout on every endpoint should return an unreachable error with the full
/// URL, so the pipeline backs off.
TEST(HTTPReadTask, TransportErrorTimeoutUnreachable) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/slow",
                .status_code = 200,
                .response_body = R"({"value": 1.0})",
                .delay = 2 * x::telem::SECOND,
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/slow";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "val", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(
        cfg,
        server.base_url(),
        {{"timeout_ms", 100}}
    );

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_OCCURRED_AS(res.error, errors::UNREACHABLE_ERROR);
    EXPECT_NE(res.error.data.find("GET"), std::string::npos);
    EXPECT_NE(res.error.data.find("/api/slow"), std::string::npos);
    EXPECT_NE(res.error.data.find("failed"), std::string::npos);
    EXPECT_EQ(fr.size(), 0);
}

/// @brief when one endpoint times out but another responds, the successful endpoint's
/// data should come through with a warning for the timeout.
TEST(HTTPReadTask, TransportErrorPartialTimeout) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                {
                    .method = Method::GET,
                    .path = "/api/fast",
                    .status_code = 200,
                    .response_body = R"({"value": 42.0})",
                },
                {
                    .method = Method::GET,
                    .path = "/api/slow",
                    .status_code = 200,
                    .response_body = R"({"value": 1.0})",
                    .delay = 2 * x::telem::SECOND,
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField fast_field;
    fast_field.pointer = "/value";
    fast_field.channel = 1;

    FloatReadField slow_field;
    slow_field.pointer = "/value";
    slow_field.channel = 2;

    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/fast";
    ep1.body = "";
    ep1.fields = {fast_field};

    ReadEndpoint ep2;
    ep2.method = "GET";
    ep2.path = "/api/slow";
    ep2.body = "";
    ep2.fields = {slow_field};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.key = 1, .name = "fast", .data_type = x::telem::FLOAT64_T};
    cfg.channels[2] = {.key = 2, .name = "slow", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(
        cfg,
        server.base_url(),
        {{"timeout_ms", 500}}
    );

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_NE(res.warning.description.find("/api/slow"), std::string::npos);
    EXPECT_NE(res.warning.description.find("failed"), std::string::npos);
    // Fast endpoint succeeded, slow endpoint timed out.
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 42.0, 0.001);
}

/// @brief when the first endpoint returns 5xx, its groups should be skipped but the
/// second endpoint's data should still come through.
TEST(HTTPReadTask, PartialFailureFirstEndpoint5xx) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    StringReadField field1;
    field1.pointer = "/error";
    field1.channel = 1;

    FloatReadField field2;
    field2.pointer = "/value";
    field2.channel = 2;

    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/failing";
    ep1.body = "";
    ep1.fields = {field1};

    ReadEndpoint ep2;
    ep2.method = "GET";
    ep2.path = "/api/ok";
    ep2.body = "";
    ep2.fields = {field2};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.key = 1, .name = "error_msg", .data_type = x::telem::STRING_T};
    cfg.channels[2] = {.key = 2, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Failed to read some data");
    EXPECT_NE(res.warning.description.find("GET"), std::string::npos);
    EXPECT_NE(res.warning.description.find("/api/failing"), std::string::npos);
    EXPECT_NE(res.warning.description.find("500"), std::string::npos);
    EXPECT_NE(
        res.warning.description.find(R"({"error":"internal"})"),
        std::string::npos
    );
    // Second endpoint succeeded.
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(2, 0), 42.0, 0.001);
}

/// @brief when the second endpoint returns 4xx, its groups should be skipped but the
/// first endpoint's data should still come through.
TEST(HTTPReadTask, PartialFailureSecondEndpointCritical4xx) {
    testutil::Server server(
        testutil::ServerConfig{
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
                    .status_code = 400,
                    .response_body = R"({"error":"bad request"})",
                },
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field1;
    field1.pointer = "/value";
    field1.channel = 1;

    StringReadField field2;
    field2.pointer = "/error";
    field2.channel = 2;

    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/ok";
    ep1.body = "";
    ep1.fields = {field1};

    ReadEndpoint ep2;
    ep2.method = "GET";
    ep2.path = "/api/failing";
    ep2.body = "";
    ep2.fields = {field2};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};
    cfg.channels[2] = {.key = 2, .name = "error_msg", .data_type = x::telem::STRING_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_NE(res.warning.description.find("GET"), std::string::npos);
    EXPECT_NE(res.warning.description.find("/api/failing"), std::string::npos);
    EXPECT_NE(res.warning.description.find("400"), std::string::npos);
    EXPECT_NE(
        res.warning.description.find(R"({"error":"bad request"})"),
        std::string::npos
    );
    // First endpoint succeeded.
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 42.0, 0.001);
}

/// @brief when one endpoint has a missing field pointer, the other endpoint's
/// data should still come through with a warning.
TEST(HTTPReadTask, PartialFailureMissingFieldInSecondEndpoint) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;

    FloatReadField pressure_field;
    // Pointer that doesn't exist in the response.
    pressure_field.pointer = "/pressure";
    pressure_field.channel = 2;

    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/temp";
    ep1.body = "";
    ep1.fields = {temp_field};

    ReadEndpoint ep2;
    ep2.method = "GET";
    ep2.path = "/api/pressure";
    ep2.body = "";
    ep2.fields = {pressure_field};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T};
    cfg.channels[2] = {.key = 2, .name = "pressure", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field1;
    field1.pointer = "/value";
    field1.channel = 1;

    FloatReadField field2;
    field2.pointer = "/data";
    field2.channel = 2;

    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/ok";
    ep1.body = "";
    ep1.fields = {field1};

    ReadEndpoint ep2;
    ep2.method = "GET";
    ep2.path = "/api/broken";
    ep2.body = "";
    ep2.fields = {field2};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};
    cfg.channels[2] = {.key = 2, .name = "data", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field1;
    field1.pointer = "/value";
    field1.channel = 1;

    IntegerReadField field2;
    field2.pointer = "/count";
    field2.channel = 2;

    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/good";
    ep1.body = "";
    ep1.fields = {field1};

    ReadEndpoint ep2;
    ep2.method = "GET";
    ep2.path = "/api/bad";
    ep2.body = "";
    ep2.fields = {field2};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};
    // INT32_T will fail on 3.7 (decimal truncation) — produces warning.
    cfg.channels[2] = {.key = 2, .name = "count", .data_type = x::telem::INT32_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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

/// @brief the sample clock should regulate the read rate so that multiple reads
/// take at least the expected duration.
TEST(HTTPReadTask, SampleClockRegulatesRate) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(20); // 20 Hz → 50ms period

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(50); // 50 Hz → 20ms period

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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

/// @brief it should pass connection-level query parameters to every request.
TEST(HTTPReadTask, EndpointQueryParams) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.query_params = std::vector<::synnax::http::QueryParam>{
        {"api_key", "abc123"},
        {"format", "json"}
    };
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField power_field;
    power_field.pointer = "/power";
    power_field.channel = 1;
    power_field.enum_values = std::vector<::x::payload::FloatEntry>{
        {"ON", 1.0},
        {"OFF", 0.0}
    };

    IntegerReadField mode_field;
    mode_field.pointer = "/mode";
    mode_field.channel = 2;
    mode_field.enum_values = std::vector<::x::payload::IntegerEntry>{
        {"AUTO", 0},
        {"MANUAL", 1},
        {"STANDBY", 2}
    };

    FloatReadField temp_field;
    temp_field.pointer = "/temperature";
    temp_field.channel = 3;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/device";
    ep.body = "";
    ep.fields = {power_field, mode_field, temp_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .key = 1,
        .name = "power",
        .data_type = x::telem::FLOAT64_T,
    };
    cfg.channels[2] = {
        .key = 2,
        .name = "mode",
        .data_type = x::telem::INT32_T,
    };
    cfg.channels[3] = {
        .key = 3,
        .name = "temperature",
        .data_type = x::telem::FLOAT64_T,
    };

    auto [source, processor] = make_source(cfg, server.base_url());

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
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField power_field;
    power_field.pointer = "/power";
    power_field.channel = 1;
    power_field.enum_values = std::vector<::x::payload::FloatEntry>{
        {"ON", 1.0},
        {"OFF", 0.0}
    };

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/device";
    ep.body = "";
    ep.fields = {power_field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .key = 1,
        .name = "power",
        .data_type = x::telem::FLOAT64_T,
    };

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    EXPECT_EQ(fr.size(), 0);
}

namespace {
/// @brief reads once from an endpoint that serves the given body.
/// @param cfg the config of the task, with no rate or device set.
/// @param body the JSON body the endpoint returns.
/// @returns the frame and the read result.
std::pair<x::telem::Frame, common::ReadResult>
read_once(ReadTaskConfig cfg, const std::string &body) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = body,
            }},
        }
    );
    if (const auto err = server.start(); err)
        return {x::telem::Frame(), common::ReadResult{.error = err}};
    x::defer::defer stop_server([&server] { server.stop(); });
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.rate = x::telem::Rate(10000);
    for (auto &ep: cfg.endpoints) {
        ep.method = "GET";
        ep.path = "/api/data";
    }
    auto [source, processor] = make_source(cfg, server.base_url());
    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    return {std::move(fr), std::move(res)};
}
}

/// @brief it should map an enum label to the exact value of an unsigned entry.
TEST(HTTPReadTask, UnsignedEnumLabel) {
    UnsignedReadField field;
    field.pointer = "/level";
    field.channel = 1;
    field.enum_values = std::vector<::x::payload::UnsignedEntry>{
        {"MAX", std::numeric_limits<uint64_t>::max()},
        {"MIN", 0},
    };
    ReadEndpoint ep;
    ep.fields = {field};
    ReadTaskConfig cfg;
    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "level", .data_type = x::telem::UINT64_T};

    auto [fr, res] = read_once(cfg, R"({"level": "MAX"})");
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(fr.at<uint64_t>(1, 0), std::numeric_limits<uint64_t>::max());
}

/// @brief it should write the value of a string field's entry, and the response text
/// when no entry has the label.
TEST(HTTPReadTask, StringFieldWithEntries) {
    StringReadField mapped;
    mapped.pointer = "/state";
    mapped.channel = 1;
    mapped.enum_values = std::vector<::x::payload::StringEntry>{
        {"1", "running"},
        {"0", "stopped"},
    };
    StringReadField unmapped = mapped;
    unmapped.pointer = "/other";
    unmapped.channel = 2;
    ReadEndpoint ep;
    ep.fields = {mapped, unmapped};
    ReadTaskConfig cfg;
    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "state", .data_type = x::telem::STRING_T};
    cfg.channels[2] = {.key = 2, .name = "other", .data_type = x::telem::STRING_T};

    auto [fr, res] = read_once(cfg, R"({"state": "1", "other": "idle"})");
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(std::get<std::string>(fr.at(1, 0)), "running");
    EXPECT_EQ(std::get<std::string>(fr.at(2, 0)), "idle");
}

/// @brief it should read a boolean channel from enum labels and JSON booleans.
TEST(HTTPReadTask, BooleanFieldWithEntries) {
    BooleanReadField on;
    on.pointer = "/on";
    on.channel = 1;
    on.enum_values = std::vector<::x::payload::BooleanEntry>{
        {"OPEN", true},
        {"CLOSED", false},
    };
    BooleanReadField off = on;
    off.pointer = "/off";
    off.channel = 2;
    BooleanReadField raw;
    raw.pointer = "/raw";
    raw.channel = 3;
    ReadEndpoint ep;
    ep.fields = {on, off, raw};
    ReadTaskConfig cfg;
    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "on", .data_type = x::telem::BOOLEAN_T};
    cfg.channels[2] = {.key = 2, .name = "off", .data_type = x::telem::BOOLEAN_T};
    cfg.channels[3] = {.key = 3, .name = "raw", .data_type = x::telem::BOOLEAN_T};

    auto [fr, res] = read_once(cfg, R"({"on": "OPEN", "off": "CLOSED", "raw": true})");
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    ASSERT_EQ(fr.size(), 3);
    EXPECT_EQ(fr.at<uint8_t>(1, 0), 1);
    EXPECT_EQ(fr.at<uint8_t>(2, 0), 0);
    EXPECT_EQ(fr.at<uint8_t>(3, 0), 1);
}

/// @brief it should read any number that is not 0 as true on a boolean channel.
TEST(HTTPReadTask, BooleanFieldReadsNumbersNotZeroAsTrue) {
    BooleanReadField big;
    big.pointer = "/big";
    big.channel = 1;
    BooleanReadField negative = big;
    negative.pointer = "/negative";
    negative.channel = 2;
    BooleanReadField zero = big;
    zero.pointer = "/zero";
    zero.channel = 3;
    ReadEndpoint ep;
    ep.fields = {big, negative, zero};
    ReadTaskConfig cfg;
    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "big", .data_type = x::telem::BOOLEAN_T};
    cfg.channels[2] = {.key = 2, .name = "negative", .data_type = x::telem::BOOLEAN_T};
    cfg.channels[3] = {.key = 3, .name = "zero", .data_type = x::telem::BOOLEAN_T};

    auto [fr, res] = read_once(cfg, R"({"big": 256, "negative": -1, "zero": 0})");
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(fr.at<uint8_t>(1, 0), 1);
    EXPECT_EQ(fr.at<uint8_t>(2, 0), 1);
    EXPECT_EQ(fr.at<uint8_t>(3, 0), 0);
}

/// @brief it should write the exact typed integer fill of a parsed config when the
/// value is absent.
TEST(HTTPReadTask, ParsedIntegerFillWritesExactly) {
    const auto [client, _, data] = create_field_channels(x::telem::INT64_T);
    auto ctx = std::make_shared<task::MockContext>(client);
    auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(
            ctx,
            create_field_task(
                {{"type", "integer"},
                 {"pointer", "/value"},
                 {"channel", data.key},
                 {"fill", "9007199254740993"}}
            )
        )
    );
    auto [fr, res] = read_once(cfg, R"({"value": null})");
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(fr.at<int64_t>(data.key, 0), int64_t{9007199254740993});
}

/// @brief it should write a parsed NaN fill with no warning when the value is absent.
TEST(HTTPReadTask, ParsedNaNFillWritesNaN) {
    const auto [client, _, data] = create_field_channels(x::telem::FLOAT32_T);
    auto ctx = std::make_shared<task::MockContext>(client);
    auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(
            ctx,
            create_field_task(
                {{"type", "float"},
                 {"pointer", "/value"},
                 {"channel", data.key},
                 {"fill", "NaN"}}
            )
        )
    );
    auto [fr, res] = read_once(cfg, R"({})");
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_TRUE(std::isnan(fr.at<float>(data.key, 0)));
}

/// @brief fields without enum_values should still parse numeric strings normally.
TEST(HTTPReadTask, EnumValuesEmptyMapFallsBackToNumericParsing) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;
    // no enum_values set — should use normal string-to-numeric

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {field};

    cfg.endpoints = {ep};

    cfg.channels[1] = {
        .key = 1,
        .name = "value",
        .data_type = x::telem::FLOAT64_T,
    };

    auto [source, processor] = make_source(cfg, server.base_url());

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

/// @brief when two fields share the same index channel (same sampling group)
/// and one field has a missing pointer, neither field should be written.
TEST(HTTPReadTask, SamplingGroupAtomicityMissingPointer) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"temp": 25.0})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;

    FloatReadField missing_field;
    missing_field.pointer = "/humidity";
    missing_field.channel = 2;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field, missing_field};

    cfg.endpoints = {ep};
    // Both channels share the same index — same sampling group.
    cfg.channels[1] =
        {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T, .index = 10};
    cfg.channels[2] = {
        .key = 2,
        .name = "humidity",
        .data_type = x::telem::FLOAT64_T,
        .index = 10,
    };
    cfg.software_timed_indexes[10] = 0;

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    // Entire group skipped — no data fields, no index timestamp.
    EXPECT_EQ(fr.size(), 0);
}

/// @brief when two fields share an index and one has a type conversion error, the
/// entire sampling group should be skipped.
TEST(HTTPReadTask, SamplingGroupAtomicityConversionError) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"temp": 25.0, "count": 3.7})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;

    IntegerReadField count_field;
    count_field.pointer = "/count";
    count_field.channel = 2;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field, count_field};

    cfg.endpoints = {ep};
    cfg.channels[1] =
        {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T, .index = 10};
    // INT32_T will fail on 3.7 — conversion error.
    cfg.channels[2] =
        {.key = 2, .name = "count", .data_type = x::telem::INT32_T, .index = 10};
    cfg.software_timed_indexes[10] = 0;

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    // Entire group skipped — temp was OK but count failed, so both are dropped.
    EXPECT_EQ(fr.size(), 0);
}

/// @brief two sampling groups on the same endpoint: if one group fails, the other
/// should still succeed independently.
TEST(HTTPReadTask, SamplingGroupIndependentOnSameEndpoint) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"temp": 25.0, "pressure": 1013.25})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    // Group A: temp + missing humidity (index 10) — will fail.
    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;

    FloatReadField humidity_field;
    humidity_field.pointer = "/humidity";
    humidity_field.channel = 2;

    // Group B: pressure (index 20) — will succeed.
    FloatReadField pressure_field;
    pressure_field.pointer = "/pressure";
    pressure_field.channel = 3;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field, humidity_field, pressure_field};

    cfg.endpoints = {ep};
    cfg.channels[1] =
        {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T, .index = 10};
    cfg.channels[2] = {
        .key = 2,
        .name = "humidity",
        .data_type = x::telem::FLOAT64_T,
        .index = 10,
    };
    cfg.channels[3] = {
        .key = 3,
        .name = "pressure",
        .data_type = x::telem::FLOAT64_T,
        .index = 20,
    };
    cfg.software_timed_indexes[10] = 0;
    cfg.software_timed_indexes[20] = 0;

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    // Group A (temp + humidity) skipped due to missing humidity pointer.
    // Group B (pressure) succeeded with its software-timed index.
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(3, 0), 1013.25, 0.001);
    // Index 20 should have a timestamp.
    EXPECT_GT(fr.at<int64_t>(20, 0), 0);
}

/// @brief two sampling groups on different endpoints: if one endpoint's group fails,
/// the other endpoint's group should still succeed.
TEST(HTTPReadTask, SamplingGroupIndependentAcrossEndpoints) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;

    // Pointer doesn't exist in the response — this group will fail.
    FloatReadField pressure_field;
    pressure_field.pointer = "/pressure";
    pressure_field.channel = 2;

    ReadEndpoint ep1;
    ep1.method = "GET";
    ep1.path = "/api/temp";
    ep1.body = "";
    ep1.fields = {temp_field};

    ReadEndpoint ep2;
    ep2.method = "GET";
    ep2.path = "/api/pressure";
    ep2.body = "";
    ep2.fields = {pressure_field};

    cfg.endpoints = {ep1, ep2};
    cfg.channels[1] = {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T};
    cfg.channels[2] = {.key = 2, .name = "pressure", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    // ep1's group succeeded, ep2's group failed (wrong pointer).
    EXPECT_EQ(fr.size(), 1);
    EXPECT_NEAR(fr.at<double>(1, 0), 25.0, 0.001);
}

/// @brief when a sampling group fails, its software-timed index timestamp should NOT be
/// written to the frame.
TEST(HTTPReadTask, SamplingGroupFailureNoIndexTimestamp) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"temp": "not_a_number"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field};

    cfg.endpoints = {ep};
    cfg.channels[1] =
        {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T, .index = 10};
    cfg.software_timed_indexes[10] = 0;

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    // Field conversion failed — no data and no index timestamp.
    EXPECT_EQ(fr.size(), 0);
}

/// @brief when one sampling group succeeds and another fails on the same endpoint, only
/// the successful group's software-timed index should be written.
TEST(HTTPReadTask, SamplingGroupPartialIndexTimestamp) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/data",
                .status_code = 200,
                .response_body = R"({"temp": 25.0, "count": "not_a_number"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField temp_field;
    temp_field.pointer = "/temp";
    temp_field.channel = 1;

    FloatReadField count_field;
    count_field.pointer = "/count";
    count_field.channel = 2;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = "";
    ep.fields = {temp_field, count_field};

    cfg.endpoints = {ep};
    // Group A: temp with index 10 — will succeed.
    cfg.channels[1] =
        {.key = 1, .name = "temp", .data_type = x::telem::FLOAT64_T, .index = 10};
    // Group B: count with index 20 — conversion will fail.
    cfg.channels[2] =
        {.key = 2, .name = "count", .data_type = x::telem::FLOAT64_T, .index = 20};
    cfg.software_timed_indexes[10] = 0;
    cfg.software_timed_indexes[20] = 0;

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_FALSE(res.warning.empty());
    // Group A succeeded: temp data + index 10 timestamp = 2 entries.
    // Group B failed: no count data, no index 20 timestamp.
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(1, 0), 25.0, 0.001);
    EXPECT_GT(fr.at<int64_t>(10, 0), 0);
}

/// @brief POST requests should include Content-Type: application/json.
TEST(HTTPReadTask, POSTSetsContentTypeJSON) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/api/query",
                .status_code = 200,
                .response_body = R"({"value": 42.0})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "POST";
    ep.path = "/api/query";
    ep.body = R"({"query": "latest"})";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto ct = reqs[0].headers.find("Content-Type");
    ASSERT_NE(ct, reqs[0].headers.end());
    EXPECT_EQ(ct->second, "application/json");
}

/// @brief it should drop a body stored on a GET endpoint rather than send it or
/// set a content type for it.
TEST(HTTPReadTask, GETDropsStoredBody) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.body = R"({"query": "latest"})";
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, "");
    EXPECT_EQ(reqs[0].headers.count("Content-Type"), 0u);
}

/// @brief it should include per-endpoint headers in the HTTP request.
TEST(HTTPReadTask, EndpointHeaders) {
    testutil::Server server(
        testutil::ServerConfig{
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
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);

    FloatReadField field;
    field.pointer = "/value";
    field.channel = 1;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/data";
    ep.headers = std::vector<::synnax::http::Header>{{"Accept", "application/json"}};
    ep.fields = {field};

    cfg.endpoints = {ep};
    cfg.channels[1] = {.key = 1, .name = "value", .data_type = x::telem::FLOAT64_T};

    auto [source, processor] = make_source(cfg, server.base_url());

    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);

    auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    auto accept = reqs[0].headers.find("Accept");
    ASSERT_NE(accept, reqs[0].headers.end());
    EXPECT_EQ(accept->second, "application/json");
}

namespace {
/// @brief reads once from an endpoint whose p2 and p3 fields share a software-timed
/// index channel.
/// @param body the JSON body the endpoint returns.
/// @param p2_type the data type of the p2 channel.
/// @param filled whether NaN fill is enabled.
/// @param fill the fill of the p2 field.
/// @returns the frame and the read result.
std::pair<x::telem::Frame, common::ReadResult> read_shared_index_group(
    const std::string &body,
    const x::telem::DataType &p2_type,
    const bool filled,
    const std::optional<x::telem::SampleValue> &fill = std::nullopt
) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/api/pressures",
                .status_code = 200,
                .response_body = body,
            }},
        }
    );
    if (const auto err = server.start(); err)
        return {x::telem::Frame(), common::ReadResult{.error = err}};
    x::defer::defer stop_server([&server] { server.stop(); });

    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.auto_start = false;
    cfg.rate = x::telem::Rate(10000);
    cfg.nan_fill_enabled = filled;

    FloatReadField p2;
    p2.pointer = "/p2";
    p2.channel = 2;
    FloatReadField p3;
    p3.pointer = "/p3";
    p3.channel = 3;

    ReadEndpoint ep;
    ep.method = "GET";
    ep.path = "/api/pressures";
    ep.fields = {p2, p3};
    cfg.endpoints = {ep};

    cfg.channels[2] = {.key = 2, .name = "p2", .data_type = p2_type, .index = 100};
    cfg.channels[3] =
        {.key = 3, .name = "p3", .data_type = x::telem::FLOAT64_T, .index = 100};
    cfg.software_timed_indexes[100] = 0;
    if (fill.has_value()) cfg.fill_values[2] = *fill;

    auto [source, processor] = make_source(cfg, server.base_url());
    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    return {std::move(fr), std::move(res)};
}
}

/// @brief it should skip the whole group when a float field is missing and invalid
/// floats are not filled.
TEST(HTTPReadTask, MissingFloatSkipsGroupByDefault) {
    auto [fr, res] = read_shared_index_group(
        R"({"p3": 1.5})",
        x::telem::FLOAT32_T,
        false
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.description.starts_with("Field /p2 not found"));
    EXPECT_EQ(fr.size(), 0);
}

/// @brief it should write NaN for a missing float field and keep the rest of the
/// group when NaN fill is enabled.
TEST(HTTPReadTask, MissingFloatFilledWithNaN) {
    auto [fr, res] = read_shared_index_group(
        R"({"p3": 1.5})",
        x::telem::FLOAT32_T,
        true
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.description.starts_with("Field /p2 not found"));
    ASSERT_EQ(fr.size(), 3);
    EXPECT_TRUE(std::isnan(fr.at<float>(2, 0)));
    EXPECT_EQ(fr.at<double>(3, 0), 1.5);
    EXPECT_GT(fr.at<int64_t>(100, 0), 0);
}

/// @brief it should write NaN for a float field that fails to convert when NaN fill
/// is enabled.
TEST(HTTPReadTask, UnconvertibleFloatFilledWithNaN) {
    auto [fr, res] = read_shared_index_group(
        R"({"p2": "INVALID", "p3": 1.5})",
        x::telem::FLOAT64_T,
        true
    );
    ASSERT_NIL(res.error);
    EXPECT_NE(
        res.warning.description.find(
            "Failed to convert /p2 for channel p2: cannot convert \"INVALID\" to "
            "float64"
        ),
        std::string::npos
    );
    ASSERT_EQ(fr.size(), 3);
    EXPECT_TRUE(std::isnan(fr.at<double>(2, 0)));
    EXPECT_EQ(fr.at<double>(3, 0), 1.5);
}

/// @brief it should write a NaN fill with no warning when the value is missing, null,
/// or an empty string, and keep the rest of the group.
TEST(HTTPReadTask, AbsentFieldWritesNaNFill) {
    for (const auto *body:
         {R"({"p3": 1.5})", R"({"p2": null, "p3": 1.5})", R"({"p2": "", "p3": 1.5})"}) {
        auto [fr, res] = read_shared_index_group(
            body,
            x::telem::FLOAT32_T,
            false,
            std::numeric_limits<float>::quiet_NaN()
        );
        ASSERT_NIL(res.error);
        EXPECT_TRUE(res.warning.empty());
        ASSERT_EQ(fr.size(), 3);
        EXPECT_TRUE(std::isnan(fr.at<float>(2, 0)));
        EXPECT_EQ(fr.at<double>(3, 0), 1.5);
    }
}

/// @brief it should write the value of a field with a NaN fill when it is present.
TEST(HTTPReadTask, PresentFieldIgnoresNaNFill) {
    auto [fr, res] = read_shared_index_group(
        R"({"p2": 2.5, "p3": 1.5})",
        x::telem::FLOAT32_T,
        false,
        std::numeric_limits<float>::quiet_NaN()
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(fr.at<float>(2, 0), 2.5F);
}

/// @brief it should still warn and skip the group when a field with a NaN fill holds a
/// value that does not convert.
TEST(HTTPReadTask, InvalidFieldWithNaNFillWarns) {
    auto [fr, res] = read_shared_index_group(
        R"({"p2": "INVALID", "p3": 1.5})",
        x::telem::FLOAT32_T,
        false,
        std::numeric_limits<float>::quiet_NaN()
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.description.starts_with("Failed to convert /p2"));
    EXPECT_EQ(fr.size(), 0);
}

/// @brief it should write the fill with no warning when the value is missing, null, or
/// an empty string.
TEST(HTTPReadTask, AbsentFieldWritesFill) {
    for (const auto *body:
         {R"({"p3": 1.5})", R"({"p2": null, "p3": 1.5})", R"({"p2": "", "p3": 1.5})"}) {
        auto [fr, res] = read_shared_index_group(body, x::telem::FLOAT64_T, false, 0.0);
        ASSERT_NIL(res.error);
        EXPECT_TRUE(res.warning.empty());
        ASSERT_EQ(fr.size(), 3);
        EXPECT_EQ(fr.at<double>(2, 0), 0.0);
        EXPECT_EQ(fr.at<double>(3, 0), 1.5);
    }
}

/// @brief it should prefer a value that is present over the fill value.
TEST(HTTPReadTask, PresentFieldIgnoresFill) {
    auto [fr, res] = read_shared_index_group(
        R"({"p2": 2.5, "p3": 1.5})",
        x::telem::FLOAT64_T,
        false,
        0.0
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(fr.at<double>(2, 0), 2.5);
}

/// @brief it should still warn and skip the group when a field with a fill value holds
/// a value that does not convert.
TEST(HTTPReadTask, InvalidFieldWithFillValueWarns) {
    auto [fr, res] = read_shared_index_group(
        R"({"p2": "INVALID", "p3": 1.5})",
        x::telem::FLOAT64_T,
        false,
        0.0
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.description.starts_with("Failed to convert /p2"));
    EXPECT_EQ(fr.size(), 0);
}

/// @brief it should keep an empty string on a string channel as a real value.
TEST(HTTPReadTask, EmptyStringIsPresentOnStringChannel) {
    auto [fr, res] = read_shared_index_group(
        R"({"p2": "", "p3": 1.5})",
        x::telem::STRING_T,
        false,
        std::string("off")
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    ASSERT_EQ(fr.size(), 3);
    EXPECT_EQ(std::get<std::string>(fr.at(2, 0)), "");
}

/// @brief it should write the fill value of a string channel when the value is missing
/// or null.
TEST(HTTPReadTask, AbsentStringFieldWritesFillValue) {
    for (const auto *body: {R"({"p3": 1.5})", R"({"p2": null, "p3": 1.5})"}) {
        auto [fr, res] = read_shared_index_group(
            body,
            x::telem::STRING_T,
            false,
            std::string("off")
        );
        ASSERT_NIL(res.error);
        EXPECT_TRUE(res.warning.empty());
        ASSERT_EQ(fr.size(), 3);
        EXPECT_EQ(std::get<std::string>(fr.at(2, 0)), "off");
    }
}

/// @brief it should write the fill value of an integer channel when the value is
/// missing.
TEST(HTTPReadTask, AbsentIntegerFieldWritesFillValue) {
    auto [fr, res] = read_shared_index_group(
        R"({"p3": 1.5})",
        x::telem::INT32_T,
        false,
        static_cast<int32_t>(7)
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    ASSERT_EQ(fr.size(), 3);
    EXPECT_EQ(fr.at<int32_t>(2, 0), 7);
}

/// @brief it should write the fill value of a timestamp channel that is not an index
/// when the value is missing.
TEST(HTTPReadTask, AbsentTimestampFieldWritesFillValue) {
    auto [fr, res] = read_shared_index_group(
        R"({"p3": 1.5})",
        x::telem::TIMESTAMP_T,
        false,
        x::telem::TimeStamp(5)
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    ASSERT_EQ(fr.size(), 3);
    EXPECT_EQ(fr.at<int64_t>(2, 0), 5);
}

/// @brief it should write no fill value when a request fails, so an outage can never
/// look like a real reading.
TEST(HTTPReadTask, FailedRequestSkipsFillValue) {
    for (const auto &[status, body]: std::vector<std::pair<int, std::string>>{
             {500, "{}"},
             {200, "not json"},
         }) {
        testutil::Server server(
            testutil::ServerConfig{
                .routes = {
                    {
                        .method = Method::GET,
                        .path = "/api/down",
                        .status_code = status,
                        .response_body = body,
                    },
                    {
                        .method = Method::GET,
                        .path = "/api/up",
                        .status_code = 200,
                        .response_body = R"({"up": 1.5})",
                    },
                },
            }
        );
        ASSERT_NIL(server.start());
        x::defer::defer stop_server([&server] { server.stop(); });

        ReadTaskConfig cfg;
        cfg.device = "test-device";
        cfg.data_saving_disabled = true;
        cfg.auto_start = false;
        cfg.rate = x::telem::Rate(10000);

        FloatReadField down;
        down.pointer = "/speed";
        down.channel = 2;
        ReadEndpoint down_ep;
        down_ep.method = "GET";
        down_ep.path = "/api/down";
        down_ep.fields = {down};

        FloatReadField up;
        up.pointer = "/up";
        up.channel = 3;
        ReadEndpoint up_ep;
        up_ep.method = "GET";
        up_ep.path = "/api/up";
        up_ep.fields = {up};
        cfg.endpoints = {down_ep, up_ep};

        cfg.channels[2] = {
            .key = 2,
            .name = "speed",
            .data_type = x::telem::FLOAT64_T,
            .index = 100,
        };
        cfg.channels[3] = {
            .key = 3,
            .name = "up",
            .data_type = x::telem::FLOAT64_T,
            .index = 101,
        };
        cfg.software_timed_indexes[100] = 0;
        cfg.software_timed_indexes[101] = 1;
        cfg.fill_values[2] = 0.0;

        auto [source, processor] = make_source(cfg, server.base_url());
        auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
        breaker.start();
        x::telem::Frame fr;
        auto res = source->read(breaker, fr);
        breaker.stop();
        ASSERT_NIL(res.error);
        EXPECT_FALSE(res.warning.empty());
        EXPECT_FALSE(fr.contains(2));
        EXPECT_EQ(fr.at<double>(3, 0), 1.5);
    }
}

/// @brief it should skip the whole group when a non-float field is invalid, even when
/// NaN fill is enabled.
TEST(HTTPReadTask, InvalidIntegerSkipsGroupWhenFilled) {
    auto [fr, res] = read_shared_index_group(
        R"({"p2": "INVALID", "p3": 1.5})",
        x::telem::INT32_T,
        true
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.description.starts_with("Failed to convert /p2"));
    EXPECT_EQ(fr.size(), 0);
}
}
