// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cmath>
#include <cstdint>
#include <memory>
#include <optional>
#include <string>
#include <tuple>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/errors/errors.h"
#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/read_task.h"
#include "driver/http/testutil/server.h"

namespace driver::http {
using ::synnax::http::ReadEndpoint;
using x::testutil::bytes;

namespace {
/// @brief returns a JSON field of kind F at pointer, named after it.
template<typename F>
F field(const std::string &pointer) {
    F f;
    f.key = x::uuid::create();
    f.name = pointer.substr(1);
    f.pointer = pointer;
    return f;
}

/// @brief Endpoint is a read endpoint and the channel of each field of its message.
struct Endpoint {
    ReadEndpoint endpoint;
    /// @brief index indexes every channel of the message.
    synnax::channel::Channel index;
    std::vector<synnax::channel::Channel> channels;
};

/// @brief creates a GET endpoint at path whose ICD holds one JSON message of fields.
/// Each field writes a new channel of the data type beside it, and one new index
/// channel indexes them all.
Endpoint create_endpoint(
    const synnax::Synnax &client,
    const std::string &path,
    const std::vector<std::pair<synnax::icd::JSONField, x::telem::DataType>> &fields,
    const std::optional<synnax::icd::JSONMatch> &match = std::nullopt
) {
    Endpoint ep;
    ep.index = ASSERT_NIL_P(
        client.channels
            .create(make_unique_channel_name("idx"), x::telem::TIMESTAMP_T, 0, true)
    );
    synnax::icd::JSONPayload payload{.match = match};
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = "Status";
    synnax::icd::ReadMessage rm{.message = m.key};
    for (const auto &[f, data_type]: fields) {
        const auto &base = synnax::icd::base(f);
        auto ch = ASSERT_NIL_P(client.channels.create(
            make_unique_channel_name(base.name),
            data_type,
            ep.index.key,
            false
        ));
        payload.fields.push_back(f);
        rm.bindings.push_back({.field = base.key, .channel = ch.key});
        ep.channels.push_back(std::move(ch));
    }
    m.payload = std::move(payload);
    synnax::icd::ICD ic;
    ic.key = x::uuid::create();
    ic.name = "Plant";
    ic.messages = {m};
    EXPECT_NIL(client.icds.create(ic));
    ep.endpoint.key = path;
    ep.endpoint.method = "GET";
    ep.endpoint.path = path;
    ep.endpoint.icd = ic.key;
    ep.endpoint.messages = {rm};
    return ep;
}

/// @brief creates a GET endpoint at path that reads the float at pointer.
Endpoint create_float_endpoint(
    const synnax::Synnax &client,
    const std::string &path,
    const std::string &pointer = "/value"
) {
    return create_endpoint(
        client,
        path,
        {{field<synnax::icd::FloatJSONField>(pointer), x::telem::FLOAT64_T}}
    );
}

/// @brief returns the config of a task that reads endpoints at rate.
ReadTaskConfig create_config(
    const std::vector<Endpoint> &endpoints,
    const x::telem::Rate rate = x::telem::Rate(10000)
) {
    ReadTaskConfig cfg;
    cfg.device = "test-device";
    cfg.data_saving_disabled = true;
    cfg.rate = rate;
    for (const auto &ep: endpoints)
        cfg.endpoints.push_back(ep.endpoint);
    return cfg;
}

/// @brief builds the source of cfg against the server at base_url the way
/// configure_read does. Each call creates its own Processor.
std::pair<std::unique_ptr<ReadTaskSource>, std::shared_ptr<Processor>> make_source(
    const synnax::Synnax &client,
    ReadTaskConfig cfg,
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
    auto requests = build_requests(
        device::ConnectionConfig(conn_parser),
        cfg.endpoints
    );
    auto readers = ASSERT_NIL_P(build_readers(client, cfg.endpoints));
    auto processor = std::make_shared<Processor>();
    return {
        std::make_unique<ReadTaskSource>(
            std::move(cfg),
            processor,
            std::move(requests),
            std::move(readers)
        ),
        std::move(processor),
    };
}

/// @brief polls cfg once against the server at base_url.
std::pair<x::telem::Frame, common::ReadResult> read_once(
    const synnax::Synnax &client,
    ReadTaskConfig cfg,
    const std::string &base_url,
    const x::json::json &conn_extra = x::json::json::object()
) {
    auto [source, processor] = make_source(
        client,
        std::move(cfg),
        base_url,
        conn_extra
    );
    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    x::telem::Frame fr;
    auto res = source->read(breaker, fr);
    breaker.stop();
    return {std::move(fr), std::move(res)};
}

/// @brief returns a GET route at path that answers with status_code and body after
/// delay.
testutil::Route route(
    const std::string &path,
    const std::string &body,
    const int status_code = 200,
    const x::telem::TimeSpan delay = x::telem::TimeSpan::ZERO()
) {
    return {
        .method = Method::GET,
        .path = path,
        .status_code = status_code,
        .response_body = body,
        .delay = delay,
    };
}

/// @brief returns a GET route at path that answers {"value": 1.0} after delay.
testutil::Route value_route(const std::string &path, const x::telem::TimeSpan delay) {
    return route(path, R"({"value": 1.0})", 200, delay);
}

/// @brief returns a task whose config reads endpoints from device.
synnax::task::Task
create_task(const x::json::json &endpoints, const std::string &device = "dev-001") {
    synnax::task::Task task;
    task.name = "read";
    task.config = {{"device", device}, {"rate", 1.0}, {"endpoints", endpoints}};
    return task;
}
}

/// @brief it should fail to parse config when endpoints array is empty.
TEST(HTTPReadTask, ParseConfigEmptyEndpoints) {
    auto [_, err] = ReadTaskConfig::parse(create_task(x::json::json::array()));
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(
        err.data.find("at least one enabled endpoint is required"),
        std::string::npos
    ) << err.data;
}

/// @brief it should fail to parse config when device field is missing.
TEST(HTTPReadTask, ParseConfigMissingDevice) {
    auto task = create_task({{{"method", "GET"}, {"path", "/api/data"}}});
    task.config.erase("device");
    ASSERT_OCCURRED_AS_P(ReadTaskConfig::parse(task), x::errors::VALIDATION);
}

/// @brief it should drop a disabled endpoint without checking its request.
TEST(HTTPReadTask, ParseConfigDisabledEndpointSkipped) {
    const auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(create_task({
            {{"method", "GET"}, {"path", "/api/temp"}},
            {{"method", "GET"},
             {"path", "/api/off"},
             {"disabled", true},
             {"headers",
              {{{"name", "A"}, {"value", "1"}}, {{"name", "A"}, {"value", "2"}}}}},
        }))
    );
    ASSERT_EQ(cfg.endpoints.size(), 1);
    EXPECT_EQ(cfg.endpoints[0].path, "/api/temp");
}

/// @brief it should fail to parse config when every endpoint is disabled.
TEST(HTTPReadTask, ParseConfigAllEndpointsDisabled) {
    auto [_, err] = ReadTaskConfig::parse(
        create_task({{{"method", "GET"}, {"path", "/api/off"}, {"disabled", true}}})
    );
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(
        err.data.find("at least one enabled endpoint is required"),
        std::string::npos
    ) << err.data;
}

/// @brief it should fail when a query_params entry is missing the parameter field.
TEST(HTTPReadTask, ParseConfigQueryParamMissingParameterErrors) {
    auto [_, err] = ReadTaskConfig::parse(create_task({{
        {"method", "GET"},
        {"path", "/api/data"},
        {"query_params", {{{"value", "10"}}}},
    }}));
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(err.data.find("parameter"), std::string::npos);
}

/// @brief a query_params entry may omit the value field; it defaults to an empty
/// string.
TEST(HTTPReadTask, ParseConfigQueryParamMissingValueDefaultsEmpty) {
    const auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(create_task({{
            {"method", "GET"},
            {"path", "/api/data"},
            {"query_params", {{{"parameter", "limit"}}}},
        }}))
    );
    ASSERT_EQ(cfg.endpoints[0].query_params.size(), 1);
    EXPECT_EQ(cfg.endpoints[0].query_params.at(0).parameter, "limit");
    EXPECT_EQ(cfg.endpoints[0].query_params.at(0).value, "");
}

/// @brief it should fail when duplicate header names exist.
TEST(HTTPReadTask, ParseConfigDuplicateHeaderErrors) {
    auto [_, err] = ReadTaskConfig::parse(create_task({{
        {"method", "GET"},
        {"path", "/api/data"},
        {"headers",
         {
             {{"name", "Accept"}, {"value", "text/plain"}},
             {{"name", "Accept"}, {"value", "application/json"}},
         }},
    }}));
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(err.data.find("duplicate header"), std::string::npos);
}

/// @brief it should fail when duplicate query parameter names exist.
TEST(HTTPReadTask, ParseConfigDuplicateQueryParamErrors) {
    auto [_, err] = ReadTaskConfig::parse(create_task({{
        {"method", "GET"},
        {"path", "/api/data"},
        {"query_params",
         {
             {{"parameter", "limit"}, {"value", "10"}},
             {{"parameter", "limit"}, {"value", "20"}},
         }},
    }}));
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(err.data.find("duplicate query parameter"), std::string::npos);
}

/// @brief it should parse an endpoint that omits query parameters, headers, and a
/// body, leaving it with none.
TEST(HTTPReadTask, ParseConfigOmittedRequestPartsDefaultEmpty) {
    const auto cfg = ASSERT_NIL_P(
        ReadTaskConfig::parse(create_task({{{"method", "GET"}, {"path", "/api/data"}}}))
    );
    ASSERT_EQ(cfg.endpoints.size(), 1);
    EXPECT_TRUE(cfg.endpoints[0].query_params.empty());
    EXPECT_TRUE(cfg.endpoints[0].headers.empty());
    EXPECT_TRUE(cfg.endpoints[0].body.empty());
}

/// @brief it should read the fields of a message from a single GET endpoint.
TEST(HTTPReadTask, SingleEndpointGETNumericField) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {route("/api/data", R"({"temperature": 23.5, "humidity": 80})")},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto ep = create_endpoint(
        client,
        "/api/data",
        {
            {field<synnax::icd::FloatJSONField>("/temperature"), x::telem::FLOAT64_T},
            {field<synnax::icd::FloatJSONField>("/humidity"), x::telem::FLOAT64_T},
        }
    );
    const auto [fr, res] = read_once(client, create_config({ep}), server.base_url());
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    EXPECT_EQ(fr.size(), 3);
    EXPECT_NEAR(fr.at<double>(ep.channels[0].key, 0), 23.5, 0.001);
    EXPECT_NEAR(fr.at<double>(ep.channels[1].key, 0), 80.0, 0.001);
}

/// @brief it should write the midpoint of the request to the index channel of a
/// message with no index field.
TEST(HTTPReadTask, SoftwareTimingIndex) {
    testutil::Server server(
        testutil::ServerConfig{.routes = {route("/api/data", R"({"value": 42.0})")}}
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto ep = create_float_endpoint(client, "/api/data");
    const auto before = x::telem::TimeStamp::now();
    const auto [fr, res] = read_once(client, create_config({ep}), server.base_url());
    const auto after = x::telem::TimeStamp::now();
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    const auto ts = fr.at<std::int64_t>(ep.index.key, 0);
    EXPECT_GE(ts, before.nanoseconds());
    EXPECT_LE(ts, after.nanoseconds());
}

/// @brief it should poll multiple endpoints in parallel.
TEST(HTTPReadTask, MultipleEndpoints) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                route("/api/temp", R"({"temp": 25.0})"),
                route("/api/pressure", R"({"pressure": 1013.25})"),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto temp = create_float_endpoint(client, "/api/temp", "/temp");
    const auto pressure = create_float_endpoint(client, "/api/pressure", "/pressure");
    const auto [fr, res] = read_once(
        client,
        create_config({temp, pressure}),
        server.base_url()
    );
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 4);
    EXPECT_NEAR(fr.at<double>(temp.channels[0].key, 0), 25.0, 0.001);
    EXPECT_NEAR(fr.at<double>(pressure.channels[0].key, 0), 1013.25, 0.001);
}

/// @brief it should send POST body and read the response.
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
    const auto client = new_test_client();
    auto ep = create_float_endpoint(client, "/api/query", "/result");
    ep.endpoint.method = "POST";
    ep.endpoint.body = R"({"query": "latest"})";
    const auto [fr, res] = read_once(client, create_config({ep}), server.base_url());
    ASSERT_NIL(res.error);
    EXPECT_NEAR(fr.at<double>(ep.channels[0].key, 0), 99.9, 0.001);
    const auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, R"({"query": "latest"})");
}

/// @brief it should successfully read 10 times in succession from the same endpoint.
TEST(HTTPReadTask, RepeatedReads) {
    testutil::Server server(
        testutil::ServerConfig{.routes = {route("/api/data", R"({"value": 42.0})")}}
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto ep = create_float_endpoint(client, "/api/data");
    auto [source, processor] = make_source(
        client,
        create_config({ep}),
        server.base_url()
    );
    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    for (int i = 0; i < 10; i++) {
        x::telem::Frame fr;
        auto res = source->read(breaker, fr);
        ASSERT_NIL(res.error);
        EXPECT_EQ(fr.size(), 2);
        EXPECT_NEAR(fr.at<double>(ep.channels[0].key, 0), 42.0, 0.001);
    }
    breaker.stop();
}

/// @brief it should write each channel of each endpoint, index channels included,
/// and tare only the channels of the bindings.
TEST(HTTPReadTask, WriterConfigHoldsEachChannelOfEachEndpoint) {
    const auto client = new_test_client();
    const auto a = create_float_endpoint(client, "/a");
    const auto b = create_float_endpoint(client, "/b");
    auto [source, processor] = make_source(
        client,
        create_config({a, b}),
        "http://localhost:0"
    );
    EXPECT_EQ(
        source->writer_config().channels,
        std::vector<synnax::channel::Key>(
            {a.channels[0].key, a.index.key, b.channels[0].key, b.index.key}
        )
    );
    std::vector<synnax::channel::Key> tared;
    for (const auto &ch: source->channels())
        tared.push_back(ch.key);
    std::ranges::sort(tared);
    EXPECT_EQ(
        tared,
        std::vector<synnax::channel::Key>({a.channels[0].key, b.channels[0].key})
    );
}

namespace {
/// @brief creates an endpoint at each path that reads the float at /value.
std::vector<Endpoint> create_value_endpoints(
    const synnax::Synnax &client,
    const std::vector<std::string> &paths
) {
    std::vector<Endpoint> eps;
    eps.reserve(paths.size());
    for (const auto &path: paths)
        eps.push_back(create_float_endpoint(client, path));
    return eps;
}
}

/// @brief every endpoint rejected with a 4xx status should return a temporary error
/// naming each status, so the pipeline backs off instead of stopping the task.
TEST(HTTPReadTask, AllEndpoints4xxBacksOff) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                route("/a", "unauthorized", 401),
                route("/b", "forbidden", 403),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto [fr, res] = read_once(
        client,
        create_config(create_value_endpoints(client, {"/a", "/b"})),
        server.base_url()
    );
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
                route("/a", "unavailable", 503),
                route("/b", "unavailable", 503),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto [fr, res] = read_once(
        client,
        create_config(create_value_endpoints(client, {"/a", "/b"})),
        server.base_url()
    );
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
                route("/a", "unauthorized", 401),
                route("/b", "unavailable", 503),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto [fr, res] = read_once(
        client,
        create_config(create_value_endpoints(client, {"/a", "/b"})),
        server.base_url()
    );
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
                route("/a", body, 401),
                value_route("/b", x::telem::TimeSpan::ZERO()),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto eps = create_value_endpoints(client, {"/a", "/b"});
    const auto [fr, res] = read_once(client, create_config(eps), server.base_url());
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(eps[1].channels[0].key, 0), 1.0, 0.001);
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
    const auto client = new_test_client();
    const auto [fr, res] = read_once(
        client,
        create_config(create_value_endpoints(client, {"/slow", "/a", "/b"})),
        server.base_url(),
        {{"timeout_ms", 100}, {"max_concurrent_requests", 1}}
    );
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
    const auto client = new_test_client();
    const auto [fr, res] = read_once(
        client,
        create_config(
            create_value_endpoints(client, {"/a", "/b", "/c"}),
            x::telem::Rate(100)
        ),
        server.base_url(),
        {{"max_concurrent_requests", 1}}
    );
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 6);
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
    const auto client = new_test_client();
    const auto [fr, res] = read_once(
        client,
        create_config(
            create_value_endpoints(client, {"/a", "/b", "/c"}),
            x::telem::Rate(2)
        ),
        server.base_url(),
        {{"max_concurrent_requests", 1}}
    );
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 6);
    EXPECT_TRUE(res.warning.empty());
}

namespace {
/// @brief returns the config of an HTTPS test server with routes.
testutil::ServerConfig secure(std::vector<testutil::Route> routes) {
    return {
        .routes = std::move(routes),
        .secure = true,
        .cert_path = "driver/http/testutil/test_cert.pem",
        .key_path = "driver/http/testutil/test_key.pem",
    };
}
}

/// @brief it should read from an HTTPS server with SSL verification disabled.
TEST(HTTPReadTask, HTTPSReadSingleEndpoint) {
    testutil::Server server(secure({route("/api/data", R"({"value": 77.7})")}));
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto ep = create_float_endpoint(client, "/api/data");
    const auto [fr, res] = read_once(client, create_config({ep}), server.base_url());
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(ep.channels[0].key, 0), 77.7, 0.001);
}

/// @brief it should read multiple endpoints over HTTPS.
TEST(HTTPReadTask, HTTPSMultipleEndpoints) {
    testutil::Server server(secure({
        route("/api/temp", R"({"temp": 22.5})"),
        route("/api/pressure", R"({"pressure": 1015.0})"),
    }));
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto temp = create_float_endpoint(client, "/api/temp", "/temp");
    const auto pressure = create_float_endpoint(client, "/api/pressure", "/pressure");
    const auto [fr, res] = read_once(
        client,
        create_config({temp, pressure}),
        server.base_url()
    );
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 4);
    EXPECT_NEAR(fr.at<double>(temp.channels[0].key, 0), 22.5, 0.001);
    EXPECT_NEAR(fr.at<double>(pressure.channels[0].key, 0), 1015.0, 0.001);
}

/// @brief it should read repeated times from HTTPS endpoints.
TEST(HTTPReadTask, HTTPSRepeatedReads) {
    testutil::Server server(secure({route("/api/data", R"({"value": 55.5})")}));
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto ep = create_float_endpoint(client, "/api/data");
    auto [source, processor] = make_source(
        client,
        create_config({ep}),
        server.base_url()
    );
    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    for (int i = 0; i < 5; i++) {
        x::telem::Frame fr;
        auto res = source->read(breaker, fr);
        ASSERT_NIL(res.error);
        EXPECT_EQ(fr.size(), 2);
        EXPECT_NEAR(fr.at<double>(ep.channels[0].key, 0), 55.5, 0.001);
    }
    breaker.stop();
}

/// @brief it should POST to an HTTPS endpoint and read the response.
TEST(HTTPReadTask, HTTPSPOSTWithBody) {
    testutil::Server server(secure({{
        .method = Method::POST,
        .path = "/api/query",
        .status_code = 200,
        .response_body = R"({"result": 88.8})",
    }}));
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    auto ep = create_float_endpoint(client, "/api/query", "/result");
    ep.endpoint.method = "POST";
    ep.endpoint.body = R"({"query": "latest"})";
    const auto [fr, res] = read_once(client, create_config({ep}), server.base_url());
    ASSERT_NIL(res.error);
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(ep.channels[0].key, 0), 88.8, 0.001);
}

/// @brief a timeout on every endpoint should return an unreachable error with the full
/// URL, so the pipeline backs off.
TEST(HTTPReadTask, TransportErrorTimeoutUnreachable) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {value_route("/api/slow", 2 * x::telem::SECOND)},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto [fr, res] = read_once(
        client,
        create_config({create_float_endpoint(client, "/api/slow")}),
        server.base_url(),
        {{"timeout_ms", 100}}
    );
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
                route("/api/fast", R"({"value": 42.0})"),
                value_route("/api/slow", 2 * x::telem::SECOND),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto eps = create_value_endpoints(client, {"/api/fast", "/api/slow"});
    const auto [fr, res] = read_once(
        client,
        create_config(eps),
        server.base_url(),
        {{"timeout_ms", 500}}
    );
    ASSERT_NIL(res.error);
    EXPECT_NE(res.warning.description.find("/api/slow"), std::string::npos);
    EXPECT_NE(res.warning.description.find("failed"), std::string::npos);
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(eps[0].channels[0].key, 0), 42.0, 0.001);
}

/// @brief when the first endpoint returns 5xx, it should write nothing, but the
/// second endpoint's data should still come through.
TEST(HTTPReadTask, PartialFailureFirstEndpoint5xx) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                route("/api/failing", R"({"error":"internal"})", 500),
                route("/api/ok", R"({"value": 42.0})"),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto failing = create_endpoint(
        client,
        "/api/failing",
        {{field<synnax::icd::StringJSONField>("/error"), x::telem::STRING_T}}
    );
    const auto ok = create_float_endpoint(client, "/api/ok");
    const auto [fr, res] = read_once(
        client,
        create_config({failing, ok}),
        server.base_url()
    );
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Failed to read some data");
    EXPECT_NE(res.warning.description.find("GET"), std::string::npos);
    EXPECT_NE(res.warning.description.find("/api/failing"), std::string::npos);
    EXPECT_NE(res.warning.description.find("500"), std::string::npos);
    EXPECT_NE(
        res.warning.description.find(R"({"error":"internal"})"),
        std::string::npos
    );
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(ok.channels[0].key, 0), 42.0, 0.001);
}

/// @brief when the second endpoint returns 4xx, it should write nothing, but the
/// first endpoint's data should still come through.
TEST(HTTPReadTask, PartialFailureSecondEndpointCritical4xx) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                route("/api/ok", R"({"value": 42.0})"),
                route("/api/failing", R"({"error":"bad request"})", 400),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto ok = create_float_endpoint(client, "/api/ok");
    const auto failing = create_endpoint(
        client,
        "/api/failing",
        {{field<synnax::icd::StringJSONField>("/error"), x::telem::STRING_T}}
    );
    const auto [fr, res] = read_once(
        client,
        create_config({ok, failing}),
        server.base_url()
    );
    ASSERT_NIL(res.error);
    EXPECT_NE(res.warning.description.find("GET"), std::string::npos);
    EXPECT_NE(res.warning.description.find("/api/failing"), std::string::npos);
    EXPECT_NE(res.warning.description.find("400"), std::string::npos);
    EXPECT_NE(
        res.warning.description.find(R"({"error":"bad request"})"),
        std::string::npos
    );
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(ok.channels[0].key, 0), 42.0, 0.001);
}

/// @brief when the response of one endpoint lacks a field, that endpoint should write
/// nothing and warn, and the other endpoint's data should still come through.
TEST(HTTPReadTask, PartialFailureMissingFieldInSecondEndpoint) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                route("/api/temp", R"({"temp": 25.0})"),
                route("/api/pressure", R"({"psi": 14.7})"),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto temp = create_float_endpoint(client, "/api/temp", "/temp");
    const auto pressure = create_float_endpoint(client, "/api/pressure", "/pressure");
    const auto [fr, res] = read_once(
        client,
        create_config({temp, pressure}),
        server.base_url()
    );
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Failed to read some data");
    EXPECT_EQ(
        res.warning.description,
        "Field /pressure not found in response from /api/pressure"
    );
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(temp.channels[0].key, 0), 25.0, 0.001);
}

/// @brief when one endpoint returns invalid JSON, the other endpoint's data should
/// still come through with a warning.
TEST(HTTPReadTask, PartialFailureInvalidJSONInOneEndpoint) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                route("/api/ok", R"({"value": 42.0})"),
                route("/api/broken", "not valid json{{{"),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto ok = create_float_endpoint(client, "/api/ok");
    const auto broken = create_float_endpoint(client, "/api/broken", "/data");
    const auto [fr, res] = read_once(
        client,
        create_config({ok, broken}),
        server.base_url()
    );
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.description.starts_with(
        "Failed to read the response from /api/broken: "
    )) << res.warning.description;
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(ok.channels[0].key, 0), 42.0, 0.001);
}

/// @brief when one endpoint has a value that does not convert, the other endpoint's
/// data should still come through with a warning.
TEST(HTTPReadTask, PartialFailureTypeConversionError) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {
                route("/api/good", R"({"value": 42.0})"),
                route("/api/bad", R"({"count": 3.7})"),
            },
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto good = create_float_endpoint(client, "/api/good");
    const auto bad = create_endpoint(
        client,
        "/api/bad",
        {{field<synnax::icd::IntegerJSONField>("/count"), x::telem::INT32_T}}
    );
    const auto [fr, res] = read_once(
        client,
        create_config({good, bad}),
        server.base_url()
    );
    ASSERT_NIL(res.error);
    EXPECT_EQ(
        res.warning.description,
        "Failed to convert /count for channel " + bad.channels[0].name +
            ": field count: 3.7 is not an integer"
    );
    EXPECT_EQ(fr.size(), 2);
    EXPECT_NEAR(fr.at<double>(good.channels[0].key, 0), 42.0, 0.001);
}

/// @brief the sample clock should regulate the read rate so that multiple reads
/// take at least the expected duration.
TEST(HTTPReadTask, SampleClockRegulatesRate) {
    testutil::Server server(
        testutil::ServerConfig{.routes = {route("/api/data", R"({"value": 1.0})")}}
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    auto [source, processor] = make_source(
        client,
        create_config({create_float_endpoint(client, "/api/data")}, x::telem::Rate(20)),
        server.base_url()
    );
    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    const auto start = x::telem::TimeStamp::now();
    for (int i = 0; i < 5; i++) {
        x::telem::Frame fr;
        ASSERT_NIL(source->read(breaker, fr).error);
    }
    const auto elapsed = x::telem::TimeStamp::now() - start;
    breaker.stop();
    // 5 reads at 20 Hz take at least 250ms. Allow some tolerance.
    EXPECT_GE(elapsed, 200 * x::telem::MILLISECOND);
}

/// @brief the sample clock should not prevent data from being read correctly.
TEST(HTTPReadTask, SampleClockDoesNotAffectData) {
    testutil::Server server(
        testutil::ServerConfig{.routes = {route("/api/data", R"({"value": 99.9})")}}
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    const auto ep = create_float_endpoint(client, "/api/data");
    auto [source, processor] = make_source(
        client,
        create_config({ep}, x::telem::Rate(50)),
        server.base_url()
    );
    auto breaker = x::breaker::Breaker(x::breaker::Config{.name = "test"});
    breaker.start();
    for (int i = 0; i < 3; i++) {
        x::telem::Frame fr;
        ASSERT_NIL(source->read(breaker, fr).error);
        EXPECT_EQ(fr.size(), 2);
        EXPECT_NEAR(fr.at<double>(ep.channels[0].key, 0), 99.9, 0.001);
    }
    breaker.stop();
}

/// @brief it should pass the query parameters of an endpoint to its request.
TEST(HTTPReadTask, EndpointQueryParams) {
    testutil::Server server(
        testutil::ServerConfig{.routes = {route("/api/data", R"({"value": 42.0})")}}
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    auto ep = create_float_endpoint(client, "/api/data");
    ep.endpoint.query_params = std::vector<::synnax::http::QueryParam>{
        {"api_key", "abc123"},
        {"format", "json"}
    };
    const auto [fr, res] = read_once(client, create_config({ep}), server.base_url());
    ASSERT_NIL(res.error);
    EXPECT_NEAR(fr.at<double>(ep.channels[0].key, 0), 42.0, 0.001);
    const auto requests = server.received_requests();
    ASSERT_EQ(requests.size(), 1);
    const auto &params = requests[0].query_params;
    EXPECT_EQ(params.find("api_key")->second, "abc123");
    EXPECT_EQ(params.find("format")->second, "json");
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
    const auto client = new_test_client();
    auto ep = create_float_endpoint(client, "/api/query");
    ep.endpoint.method = "POST";
    ep.endpoint.body = R"({"query": "latest"})";
    ASSERT_NIL(read_once(client, create_config({ep}), server.base_url()).second.error);
    const auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    const auto ct = reqs[0].headers.find("Content-Type");
    ASSERT_NE(ct, reqs[0].headers.end());
    EXPECT_EQ(ct->second, "application/json");
}

/// @brief it should drop a body stored on a GET endpoint rather than send it or
/// set a content type for it.
TEST(HTTPReadTask, GETDropsStoredBody) {
    testutil::Server server(
        testutil::ServerConfig{.routes = {route("/api/data", R"({"value": 42.0})")}}
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    auto ep = create_float_endpoint(client, "/api/data");
    ep.endpoint.body = R"({"query": "latest"})";
    ASSERT_NIL(read_once(client, create_config({ep}), server.base_url()).second.error);
    const auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    EXPECT_EQ(reqs[0].body, "");
    EXPECT_EQ(reqs[0].headers.count("Content-Type"), 0u);
}

/// @brief it should include per-endpoint headers in the HTTP request.
TEST(HTTPReadTask, EndpointHeaders) {
    testutil::Server server(
        testutil::ServerConfig{.routes = {route("/api/data", R"({"value": 1.0})")}}
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    auto ep = create_float_endpoint(client, "/api/data");
    ep.endpoint.headers = std::vector<::synnax::http::Header>{
        {"Accept", "application/json"}
    };
    ASSERT_NIL(read_once(client, create_config({ep}), server.base_url()).second.error);
    const auto reqs = server.received_requests();
    ASSERT_EQ(reqs.size(), 1);
    const auto accept = reqs[0].headers.find("Accept");
    ASSERT_NE(accept, reqs[0].headers.end());
    EXPECT_EQ(accept->second, "application/json");
}

namespace {
/// @brief Pressures is one read of an endpoint whose message holds a field p2 and a
/// float field p3 on one index channel.
struct Pressures {
    Endpoint endpoint;
    x::telem::Frame frame;
    common::ReadResult result;
};

/// @brief reads body once from /api/pressures, where p2 is the field beside it.
Pressures read_pressures(
    const std::string &body,
    const std::pair<synnax::icd::JSONField, x::telem::DataType> &p2,
    const bool nan_filled
) {
    testutil::Server server(
        testutil::ServerConfig{.routes = {route("/api/pressures", body)}}
    );
    if (const auto err = server.start(); err)
        return {.result = common::ReadResult{.error = err}};
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    auto ep = create_endpoint(
        client,
        "/api/pressures",
        {p2, {field<synnax::icd::FloatJSONField>("/p3"), x::telem::FLOAT64_T}}
    );
    auto cfg = create_config({ep});
    cfg.nan_fill_enabled = nan_filled;
    auto [fr, res] = read_once(client, std::move(cfg), server.base_url());
    return {
        .endpoint = std::move(ep),
        .frame = std::move(fr),
        .result = std::move(res)
    };
}

/// @brief returns a float field at /p2 on a channel of data_type.
std::pair<synnax::icd::JSONField, x::telem::DataType>
float_p2(const x::telem::DataType &data_type = x::telem::FLOAT64_T) {
    return {field<synnax::icd::FloatJSONField>("/p2"), data_type};
}
}

/// @brief it should write nothing for the index group of a float field that the
/// response lacks when NaN fill is off.
TEST(HTTPReadTask, MissingFloatDropsGroupByDefault) {
    const auto p = read_pressures(
        R"({"p3": 1.5})",
        float_p2(x::telem::FLOAT32_T),
        false
    );
    ASSERT_NIL(p.result.error);
    EXPECT_EQ(
        p.result.warning.description,
        "Field /p2 not found in response from /api/pressures"
    );
    EXPECT_EQ(p.frame.size(), 0);
}

/// @brief it should write NaN for a missing float field and keep the rest of the
/// message when NaN fill is on.
TEST(HTTPReadTask, MissingFloatFilledWithNaN) {
    const auto p = read_pressures(
        R"({"p3": 1.5})",
        float_p2(x::telem::FLOAT32_T),
        true
    );
    ASSERT_NIL(p.result.error);
    EXPECT_EQ(p.result.warning.message, "Failed to read some data");
    EXPECT_EQ(
        p.result.warning.description,
        "Field /p2 not found in response from /api/pressures"
    );
    const auto &chs = p.endpoint.channels;
    ASSERT_EQ(p.frame.size(), 3);
    EXPECT_TRUE(std::isnan(p.frame.at<float>(chs[0].key, 0)));
    EXPECT_EQ(p.frame.at<double>(chs[1].key, 0), 1.5);
    EXPECT_GT(p.frame.at<std::int64_t>(p.endpoint.index.key, 0), 0);
}

/// @brief it should write NaN for a float field that fails to convert when NaN fill
/// is on, and name the field and its channel.
TEST(HTTPReadTask, UnconvertibleFloatFilledWithNaN) {
    const auto p = read_pressures(R"({"p2": "INVALID", "p3": 1.5})", float_p2(), true);
    ASSERT_NIL(p.result.error);
    const auto &chs = p.endpoint.channels;
    EXPECT_EQ(
        p.result.warning.description,
        "Failed to convert /p2 for channel " + chs[0].name +
            R"(: field p2: invalid number "INVALID")"
    );
    ASSERT_EQ(p.frame.size(), 3);
    EXPECT_TRUE(std::isnan(p.frame.at<double>(chs[0].key, 0)));
    EXPECT_EQ(p.frame.at<double>(chs[1].key, 0), 1.5);
}

/// @brief it should write the default of a missing field with no warning.
TEST(HTTPReadTask, FieldWithDefaultDoesNotWarn) {
    auto p2 = field<synnax::icd::FloatJSONField>("/p2");
    p2.default_ = 5.0;
    const auto p = read_pressures(R"({"p3": 1.5})", {p2, x::telem::FLOAT64_T}, true);
    ASSERT_NIL(p.result.error);
    EXPECT_TRUE(p.result.warning.empty());
    EXPECT_EQ(p.frame.at<double>(p.endpoint.channels[0].key, 0), 5.0);
}

/// @brief it should write nothing for the index group of a non-float field that fails
/// to convert, even when NaN fill is on.
TEST(HTTPReadTask, InvalidIntegerDropsGroupWhenFilled) {
    const auto p = read_pressures(
        R"({"p2": "INVALID", "p3": 1.5})",
        {field<synnax::icd::IntegerJSONField>("/p2"), x::telem::INT32_T},
        true
    );
    ASSERT_NIL(p.result.error);
    EXPECT_EQ(
        p.result.warning.description,
        "Failed to convert /p2 for channel " + p.endpoint.channels[0].name +
            R"(: field p2: invalid number "INVALID")"
    );
    EXPECT_EQ(p.frame.size(), 0);
}

/// @brief it should write no NaN when a request fails, so an outage can never look
/// like a real reading.
TEST(HTTPReadTask, FailedRequestWritesNothingWhenFilled) {
    for (const auto &[status, body]: std::vector<std::pair<int, std::string>>{
             {500, "{}"},
             {200, "not json"},
         }) {
        testutil::Server server(
            testutil::ServerConfig{
                .routes = {
                    route("/api/down", body, status),
                    route("/api/up", R"({"up": 1.5})"),
                },
            }
        );
        ASSERT_NIL(server.start());
        x::defer::defer stop_server([&server] { server.stop(); });
        const auto client = new_test_client();
        const auto down = create_float_endpoint(client, "/api/down", "/speed");
        const auto up = create_float_endpoint(client, "/api/up", "/up");
        auto cfg = create_config({down, up});
        cfg.nan_fill_enabled = true;
        const auto [fr, res] = read_once(client, std::move(cfg), server.base_url());
        ASSERT_NIL(res.error);
        EXPECT_FALSE(res.warning.empty());
        EXPECT_FALSE(fr.contains(down.channels[0].key));
        EXPECT_FALSE(fr.contains(down.index.key));
        EXPECT_EQ(fr.at<double>(up.channels[0].key, 0), 1.5);
    }
}

/// @brief it should warn and write nothing for a response that matches no message.
TEST(HTTPReadTask, ResponseMatchingNoMessageWarns) {
    testutil::Server server(
        testutil::ServerConfig{
            .routes = {route("/api/data", R"({"type": "other", "value": 1.0})")},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });
    const auto client = new_test_client();
    synnax::icd::StringJSONMatch match;
    match.pointer = "/type";
    match.value = "status";
    const auto ep = create_endpoint(
        client,
        "/api/data",
        {{field<synnax::icd::FloatJSONField>("/value"), x::telem::FLOAT64_T}},
        match
    );
    const auto [fr, res] = read_once(client, create_config({ep}), server.base_url());
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Failed to read some data");
    EXPECT_EQ(
        res.warning.description,
        "The response from /api/data matches no message"
    );
    EXPECT_EQ(fr.size(), 0);
}

/// @brief it should build the reader of each endpoint from the messages of its ICD.
TEST(HTTPReadTask, BuildReadersReadsEachEndpoint) {
    const auto client = new_test_client();
    const auto a = create_float_endpoint(client, "/a", "/v");
    const auto b = create_float_endpoint(client, "/b", "/v");
    auto readers = ASSERT_NIL_P(build_readers(client, {a.endpoint, b.endpoint}));
    ASSERT_EQ(readers.endpoints.size(), 2);
    EXPECT_EQ(readers.pointers.at(b.channels[0].key), "/v");
    EXPECT_EQ(readers.channels.at(b.channels[0].key).name, b.channels[0].name);
    const x::telem::TimeStamp received(1000);
    x::telem::Frame fr;
    ASSERT_EQ(
        ASSERT_NIL_P(readers.endpoints[1].read(bytes(R"({"v": 2.5})"), received, fr)),
        0U
    );
    EXPECT_EQ(fr.at<double>(b.channels[0].key, 0), 2.5);
    EXPECT_EQ(fr.at<std::int64_t>(b.index.key, 0), received.nanoseconds());
    EXPECT_FALSE(fr.contains(a.channels[0].key));
}

/// @brief it should reject an endpoint with no ICD or no messages, and name it.
TEST(HTTPReadTask, BuildReadersRejectsEndpointWithNoMessages) {
    const auto client = new_test_client();
    ReadEndpoint no_icd;
    no_icd.path = "/a";
    auto no_messages = create_float_endpoint(client, "/a").endpoint;
    no_messages.messages.clear();
    for (const auto &ep: {no_icd, no_messages}) {
        const auto err = build_readers(client, {ep}).second;
        ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
        EXPECT_EQ(
            err.data,
            "endpoint GET /a reads no messages. Open the task and pick a message "
            "for it."
        );
    }
}

/// @brief it should reject two endpoints that write to one channel, and name both.
TEST(HTTPReadTask, BuildReadersRejectsChannelOfTwoEndpoints) {
    const auto client = new_test_client();
    const auto a = create_float_endpoint(client, "/a");
    auto b = create_float_endpoint(client, "/b");
    b.endpoint.messages[0].bindings[0].channel = a.channels[0].key;
    const auto err = build_readers(client, {a.endpoint, b.endpoint}).second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    EXPECT_EQ(
        err.data,
        "endpoint GET /a and endpoint GET /b write to the same channel " +
            std::to_string(a.channels[0].key)
    );
}

/// @brief it should reject two endpoints whose channels share one index channel, since
/// each would write it with its own arrival time.
TEST(HTTPReadTask, BuildReadersRejectsIndexOfTwoEndpoints) {
    const auto client = new_test_client();
    const auto a = create_float_endpoint(client, "/a");
    const auto beside = ASSERT_NIL_P(client.channels.create(
        make_unique_channel_name("value"),
        x::telem::FLOAT64_T,
        a.index.key,
        false
    ));
    auto b = create_float_endpoint(client, "/b");
    b.endpoint.messages[0].bindings[0].channel = beside.key;
    const auto err = build_readers(client, {a.endpoint, b.endpoint}).second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    EXPECT_EQ(
        err.data,
        "endpoint GET /a and endpoint GET /b write to the same channel " +
            std::to_string(a.index.key)
    );
}

/// @brief it should reject an endpoint whose ICD does not exist.
TEST(HTTPReadTask, BuildReadersRejectsMissingICD) {
    const auto client = new_test_client();
    ReadEndpoint ep;
    ep.path = "/a";
    ep.icd = x::uuid::create();
    ep.messages = {{.message = x::uuid::create()}};
    const auto err = build_readers(client, {ep}).second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    EXPECT_EQ(
        err.data,
        "endpoint GET /a: ICD " + ep.icd->to_string() + " does not exist"
    );
}

/// @brief it should name the endpoint in an error that the reader of its messages
/// returns, and reject a binding to a channel that does not exist.
TEST(HTTPReadTask, BuildReadersNamesEndpointOfMessageError) {
    const auto client = new_test_client();
    for (const synnax::channel::Key key: {0U, 1U}) {
        auto ep = create_float_endpoint(client, "/a").endpoint;
        ep.messages[0].bindings[0].channel = key;
        const auto err = build_readers(client, {ep}).second;
        ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
        EXPECT_EQ(
            err.data,
            "endpoint GET /a: message Status: channel " + std::to_string(key) +
                " does not exist"
        );
    }
}

/// @brief it should fail to configure a task whose endpoint reads no messages, and
/// tell the user what to do.
TEST(HTTPReadTask, ConfigureRejectsEndpointWithNoMessages) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("http_read_test_rack"))
    );
    const x::json::json props = {
        {"secure", false},
        {"timeout_ms", 1000},
        {"verify_ssl", false},
    };
    synnax::device::Device dev{
        .key = make_unique_channel_name("http_read_test_device"),
        .rack = rack.key,
        .location = "localhost:0",
        .make = "http",
        .model = "HTTP Device",
        .name = "HTTP Read Test Device",
        .properties = props.get<x::json::json::object_t>(),
    };
    ASSERT_NIL(client->devices.create(dev));
    const auto err = configure_read(
                         std::make_shared<task::MockContext>(client),
                         create_task({{{"method", "GET"}, {"path", "/a"}}}, dev.key),
                         std::make_shared<Processor>()
    )
                         .second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    EXPECT_EQ(
        err.data,
        "endpoint GET /a reads no messages. Open the task and pick a message for it."
    );
}
}
