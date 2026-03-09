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
#include "x/cpp/test/test.h"

#include "driver/http/mock/server.h"
#include "driver/http/scan_task.h"

namespace driver::http {
namespace {
/// @brief extracts the host:port portion from a URL like "http://127.0.0.1:8080".
std::string host_port_from_url(const std::string &url) {
    auto pos = url.find("://");
    if (pos == std::string::npos) return url;
    return url.substr(pos + 3);
}

/// @brief builds a synnax device with an auto-generated unique key.
/// check_device_health constructs base_url from "http://" + dev.location, so set
/// location to the mock server's host:port.
synnax::device::Device make_device(
    const std::string &location,
    const x::json::json &extra_props = x::json::json::object()
) {
    synnax::device::Device dev;
    dev.key = make_unique_channel_name("dev");
    dev.name = dev.key;
    dev.make = INTEGRATION_NAME;
    dev.location = location;
    auto props = x::json::json{
        {"timeout_ms", 1000},
        {"verify_ssl", false},
        {"secure", false},
    };
    if (!extra_props.contains("health_check"))
        props["health_check"] = {{"method", "GET"}, {"path", "/health"}};
    props.update(extra_props);
    dev.properties = props;
    return dev;
}
}

TEST(HTTPScanTask, ExpectedResponseConfigParsesFields) {
    auto j = x::json::json{{"pointer", "/data/ok"}, {"expected_value", 42}};
    auto parser = x::json::Parser(j);
    const ExpectedResponseConfig cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(cfg.pointer, "/data/ok");
    EXPECT_EQ(cfg.expected_value, 42);
}

TEST(HTTPScanTask, ExpectedResponseConfigBooleanValue) {
    auto j = x::json::json{{"pointer", "/alive"}, {"expected_value", true}};
    auto parser = x::json::Parser(j);
    const ExpectedResponseConfig cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(cfg.expected_value, true);
}

TEST(HTTPScanTask, ExpectedResponseConfigObjectValue) {
    auto expected = x::json::json{{"nested", "value"}};
    auto j = x::json::json{{"pointer", "/obj"}, {"expected_value", expected}};
    auto parser = x::json::Parser(j);
    const ExpectedResponseConfig cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(cfg.expected_value, expected);
}

TEST(HTTPScanTask, ExpectedResponseConfigMissingPointer) {
    auto j = x::json::json{{"expected_value", "ok"}};
    auto parser = x::json::Parser(j);
    const ExpectedResponseConfig cfg(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, ExpectedResponseConfigMissingExpectedValue) {
    auto j = x::json::json{{"pointer", "/status"}};
    auto parser = x::json::Parser(j);
    const ExpectedResponseConfig cfg(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, ExpectedResponseConfigMissingBoth) {
    auto j = x::json::json::object();
    auto parser = x::json::Parser(j);
    const ExpectedResponseConfig cfg(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, HealthCheckConfigParsesAllFields) {
    auto j = x::json::json{
        {"method", "POST"},
        {"path", "/api/status"},
        {"query_params", {{"key", "val"}}},
        {"headers", {{"X-Custom", "abc"}}},
        {"body", R"({"ping": true})"},
        {"response",
         {
             {"pointer", "/status"},
             {"expected_value", "ok"},
         }},
    };
    auto parser = x::json::Parser(j);
    const HealthCheckConfig hc(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(hc.request.method, Method::POST);
    EXPECT_EQ(hc.request.path, "/api/status");
    EXPECT_EQ(hc.request.query_params.at("key"), "val");
    EXPECT_EQ(hc.request.headers.at("X-Custom"), "abc");
    EXPECT_EQ(hc.body, R"({"ping": true})");
    ASSERT_TRUE(hc.expected_response.has_value());
    EXPECT_EQ(hc.expected_response->pointer, "/status");
    EXPECT_EQ(hc.expected_response->expected_value, "ok");
}

TEST(HTTPScanTask, HealthCheckConfigDefaults) {
    auto j = x::json::json{
        {"method", "GET"},
        {"path", "/health"},
    };
    auto parser = x::json::Parser(j);
    const HealthCheckConfig hc(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(hc.request.method, Method::GET);
    EXPECT_EQ(hc.request.path, "/health");
    EXPECT_TRUE(hc.request.query_params.empty());
    EXPECT_TRUE(hc.request.headers.empty());
    EXPECT_TRUE(hc.body.empty());
    EXPECT_FALSE(hc.expected_response.has_value());
}

TEST(HTTPScanTask, HealthCheckConfigMissingMethod) {
    auto j = x::json::json{{"path", "/health"}};
    auto parser = x::json::Parser(j);
    const HealthCheckConfig hc(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, HealthCheckConfigMissingPath) {
    auto j = x::json::json{{"method", "GET"}};
    auto parser = x::json::Parser(j);
    const HealthCheckConfig hc(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, HealthCheckConfigInvalidExpectedResponse) {
    auto j = x::json::json{
        {"method", "GET"},
        {"path", "/health"},
        {"response", {{"pointer", "/status"}}},
    };
    auto parser = x::json::Parser(j);
    const HealthCheckConfig hc(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, ScanCommandArgsParsesValid) {
    auto j = x::json::json{
        {"connection",
         {
             {"base_url", "http://127.0.0.1:8080"},
             {"timeout_ms", 2000},
             {"verify_ssl", false},
         }},
        {"health_check",
         {
             {"method", "GET"},
             {"path", "/health"},
         }},
    };
    auto parser = x::json::Parser(j);
    const ScanCommandArgs args(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(args.connection.base_url, "http://127.0.0.1:8080");
    EXPECT_EQ(args.health_check.request.method, Method::GET);
    EXPECT_EQ(args.health_check.request.path, "/health");
}

TEST(HTTPScanTask, ScanCommandArgsMissingConnection) {
    auto j = x::json::json{
        {"health_check", {{"method", "GET"}, {"path", "/health"}}},
    };
    auto parser = x::json::Parser(j);
    const ScanCommandArgs args(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, ScanCommandArgsMissingHealthCheck) {
    auto j = x::json::json{
        {"connection",
         {
             {"base_url", "http://127.0.0.1:8080"},
             {"timeout_ms", 1000},
             {"verify_ssl", false},
         }},
    };
    auto parser = x::json::Parser(j);
    const ScanCommandArgs args(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, ScanCommandArgsInvalidConnection) {
    auto j = x::json::json{
        {"connection", x::json::json::object()},
        {"health_check", {{"method", "GET"}, {"path", "/health"}}},
    };
    auto parser = x::json::Parser(j);
    const ScanCommandArgs args(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, ScanCommandArgsInvalidHealthCheck) {
    auto j = x::json::json{
        {"connection",
         {
             {"base_url", "http://127.0.0.1:8080"},
             {"timeout_ms", 1000},
             {"verify_ssl", false},
         }},
        {"health_check", x::json::json::object()},
    };
    auto parser = x::json::Parser(j);
    const ScanCommandArgs args(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, ScanCommandArgsEmptyArgs) {
    auto j = x::json::json::object();
    auto parser = x::json::Parser(j);
    const ScanCommandArgs args(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(HTTPScanTask, ScannerConfig) {
    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto cfg = scanner.config();
    EXPECT_EQ(cfg.make, INTEGRATION_NAME);
    EXPECT_EQ(cfg.log_prefix, SCAN_LOG_PREFIX);
}

TEST(HTTPScanTask, ScanHealthyDevice) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {.method = Method::GET, .path = "/health", .response_body = "{}"}
            },
        }
    );
    ASSERT_NIL(server.start());

    auto dev = make_device(host_port_from_url(server.base_url()));

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(result[0].status.message, "Device connected");

    server.stop();
}

TEST(HTTPScanTask, ScanSuccessOnHTTP200) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .status_code = 200,
                .response_body = "anything",
            }},
        }
    );
    ASSERT_NIL(server.start());

    auto dev = make_device(host_port_from_url(server.base_url()));

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(result[0].status.message, "Device connected");

    server.stop();
}

TEST(HTTPScanTask, ScanFailsOnNon2xxStatus) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .status_code = 503,
                .response_body = "Service Unavailable",
            }},
        }
    );
    ASSERT_NIL(server.start());

    auto dev = make_device(host_port_from_url(server.base_url()));

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(result[0].status.message, "HTTP 503");
    EXPECT_EQ(result[0].status.description, "Service Unavailable");

    server.stop();
}

TEST(HTTPScanTask, ScanRepeatedScans) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {.method = Method::GET, .path = "/health", .response_body = "{}"}
            },
        }
    );
    ASSERT_NIL(server.start());

    auto dev = make_device(host_port_from_url(server.base_url()));

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    for (int i = 0; i < 3; i++) {
        const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
        ASSERT_EQ(result.size(), 1);
        EXPECT_EQ(result[0].status.variant, x::status::VARIANT_SUCCESS);
        EXPECT_EQ(result[0].status.message, "Device connected");
    }

    // Stop server and verify device becomes unreachable on next scan.
    server.stop();

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(result[0].status.message, "Failed to reach server");
}

TEST(HTTPScanTask, ScanUnreachableDevice) {
    auto dev = make_device("127.0.0.1:1", {{"timeout_ms", 500}});

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(result[0].status.message, "Failed to reach server");
    EXPECT_NE(
        result[0].status.description.find("Could not connect to server"),
        std::string::npos
    );
}

TEST(HTTPScanTask, ScanHealthCheckValidationFailure) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .response_body = R"({"status": "degraded"})",
            }},
        }
    );
    ASSERT_NIL(server.start());

    auto dev = make_device(
        host_port_from_url(server.base_url()),
        {
            {"health_check",
             {
                 {"method", "GET"},
                 {"path", "/health"},
                 {"response",
                  {
                      {"pointer", "/status"},
                      {"expected_value", "ok"},
                  }},
             }},
        }
    );

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(result[0].status.message, "Health check validation failed");
    EXPECT_EQ(
        result[0].status.description,
        "expected value at '/status' to be \"ok\", got \"degraded\""
    );

    server.stop();
}

TEST(HTTPScanTask, ScanHealthCheckValidationSuccess) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .response_body = R"({"status": "ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());

    auto dev = make_device(
        host_port_from_url(server.base_url()),
        {
            {"health_check",
             {
                 {"method", "GET"},
                 {"path", "/health"},
                 {"response",
                  {
                      {"pointer", "/status"},
                      {"expected_value", "ok"},
                  }},
             }},
        }
    );

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(result[0].status.message, "Device connected");

    server.stop();
}

TEST(HTTPScanTask, ScanNoDevices) {
    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    common::ScannerContext scan_ctx{.devices = nullptr};
    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    EXPECT_TRUE(result.empty());
}

TEST(HTTPScanTask, ScanMultipleDevices) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {.method = Method::GET, .path = "/health", .response_body = "{}"}
            },
        }
    );
    ASSERT_NIL(server.start());

    auto healthy_dev = make_device(host_port_from_url(server.base_url()));
    auto bad_dev = make_device("127.0.0.1:1", {{"timeout_ms", 500}});

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[healthy_dev.key] = healthy_dev;
    devices[bad_dev.key] = bad_dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 2);

    for (const auto &dev: result) {
        if (dev.key == healthy_dev.key) {
            EXPECT_EQ(dev.status.variant, x::status::VARIANT_SUCCESS);
        } else if (dev.key == bad_dev.key) {
            EXPECT_EQ(dev.status.variant, x::status::VARIANT_WARNING);
        } else {
            FAIL() << "Unexpected device key: " << dev.key;
        }
    }

    server.stop();
}

TEST(HTTPScanTask, ScanInvalidHealthCheck) {
    synnax::device::Device dev;
    dev.key = make_unique_channel_name("dev");
    dev.name = dev.key;
    dev.make = INTEGRATION_NAME;
    dev.location = "127.0.0.1:8080";
    dev.properties = x::json::json{
        {"timeout_ms", 1000},
        {"verify_ssl", false},
        {"secure", false},
        {"base_url", "http://127.0.0.1:8080"},
        {"health_check", x::json::json::object()},
    };

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(result[0].status.message, "Invalid device properties");
    EXPECT_NE(result[0].status.description.find("health_check"), std::string::npos);
}

TEST(HTTPScanTask, ScanInvalidDeviceProperties) {
    synnax::device::Device dev;
    dev.key = make_unique_channel_name("dev");
    dev.name = dev.key;
    dev.make = INTEGRATION_NAME;
    dev.properties = x::json::json{{"secure", false}};

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(result[0].status.message, "Invalid device properties");
    EXPECT_NE(result[0].status.description.find("base_url"), std::string::npos);
}

TEST(HTTPScanTask, ScanWithPOSTHealthCheck) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::POST,
                .path = "/ping",
                .response_body = R"({"alive": true})",
            }},
        }
    );
    ASSERT_NIL(server.start());

    auto dev = make_device(
        host_port_from_url(server.base_url()),
        {
            {"health_check",
             {
                 {"method", "POST"},
                 {"path", "/ping"},
                 {"body", R"({"check": "heartbeat"})"},
                 {"response",
                  {
                      {"pointer", "/alive"},
                      {"expected_value", true},
                  }},
             }},
        }
    );

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_SUCCESS);

    auto received = server.received_requests();
    ASSERT_FALSE(received.empty());
    EXPECT_EQ(received[0].method, Method::POST);
    EXPECT_EQ(received[0].body, R"({"check": "heartbeat"})");

    server.stop();
}

TEST(HTTPScanTask, ScanHealthCheckNonJSONResponse) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .response_body = "not json",
                .content_type = "text/plain",
            }},
        }
    );
    ASSERT_NIL(server.start());

    auto dev = make_device(
        host_port_from_url(server.base_url()),
        {
            {"health_check",
             {
                 {"method", "GET"},
                 {"path", "/health"},
                 {"response",
                  {
                      {"pointer", "/status"},
                      {"expected_value", "ok"},
                  }},
             }},
        }
    );

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(result[0].status.message, "Health check validation failed");
    EXPECT_NE(
        result[0].status.description.find("failed to parse response body as JSON"),
        std::string::npos
    );

    server.stop();
}

TEST(HTTPScanTask, ScanHealthCheckMissingPointer) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .response_body = R"({"other": "value"})",
            }},
        }
    );
    ASSERT_NIL(server.start());

    auto dev = make_device(
        host_port_from_url(server.base_url()),
        {
            {"health_check",
             {
                 {"method", "GET"},
                 {"path", "/health"},
                 {"response",
                  {
                      {"pointer", "/status"},
                      {"expected_value", "ok"},
                  }},
             }},
        }
    );

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices[dev.key] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(result[0].status.message, "Health check validation failed");
    EXPECT_EQ(
        result[0].status.description,
        "response body does not contain pointer '/status'"
    );

    server.stop();
}

TEST(HTTPScanTask, TestConnectionSuccess) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {
                {.method = Method::GET, .path = "/health", .response_body = "{}"}
            },
        }
    );
    ASSERT_NIL(server.start());

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    task::Command cmd;
    cmd.type = TEST_CONNECTION_CMD_TYPE;
    cmd.key = "cmd-1";
    cmd.args = {
        {"connection",
         {
             {"base_url", server.base_url()},
             {"timeout_ms", 1000},
             {"verify_ssl", false},
         }},
        {"health_check",
         {
             {"method", "GET"},
             {"path", "/health"},
         }},
    };

    EXPECT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_FALSE(ctx->statuses.empty());
    EXPECT_EQ(ctx->statuses.back().variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(ctx->statuses.back().message, "Connection successful");

    server.stop();
}

TEST(HTTPScanTask, TestConnectionUnreachable) {
    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    task::Command cmd;
    cmd.type = TEST_CONNECTION_CMD_TYPE;
    cmd.key = "cmd-1";
    cmd.args = {
        {"connection",
         {
             {"base_url", "http://127.0.0.1:1"},
             {"timeout_ms", 500},
             {"verify_ssl", false},
         }},
        {"health_check",
         {
             {"method", "GET"},
             {"path", "/health"},
         }},
    };

    EXPECT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_FALSE(ctx->statuses.empty());
    EXPECT_EQ(ctx->statuses.back().variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses.back().message, "Failed to execute HTTP request");
    EXPECT_NE(
        ctx->statuses.back().description.find("Could not connect to server"),
        std::string::npos
    );
}

TEST(HTTPScanTask, TestConnectionValidationFailure) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .response_body = R"({"status": "bad"})",
            }},
        }
    );
    ASSERT_NIL(server.start());

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    task::Command cmd;
    cmd.type = TEST_CONNECTION_CMD_TYPE;
    cmd.key = "cmd-1";
    cmd.args = {
        {"connection",
         {
             {"base_url", server.base_url()},
             {"timeout_ms", 1000},
             {"verify_ssl", false},
         }},
        {"health_check",
         {
             {"method", "GET"},
             {"path", "/health"},
             {"response",
              {
                  {"pointer", "/status"},
                  {"expected_value", "ok"},
              }},
         }},
    };

    EXPECT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_FALSE(ctx->statuses.empty());
    EXPECT_EQ(ctx->statuses.back().variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses.back().message, "Invalid health check response");
    EXPECT_EQ(
        ctx->statuses.back().description,
        "expected value at '/status' to be \"ok\", got \"bad\""
    );

    server.stop();
}

TEST(HTTPScanTask, TestConnectionNon2xxStatus) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .status_code = 503,
                .response_body = "Service Unavailable",
            }},
        }
    );
    ASSERT_NIL(server.start());

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    task::Command cmd;
    cmd.type = TEST_CONNECTION_CMD_TYPE;
    cmd.key = "cmd-1";
    cmd.args = {
        {"connection",
         {
             {"base_url", server.base_url()},
             {"timeout_ms", 1000},
             {"verify_ssl", false},
         }},
        {"health_check",
         {
             {"method", "GET"},
             {"path", "/health"},
         }},
    };

    EXPECT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_FALSE(ctx->statuses.empty());
    EXPECT_EQ(ctx->statuses.back().variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses.back().message, "HTTP 503");
    EXPECT_EQ(ctx->statuses.back().description, "Service Unavailable");

    server.stop();
}

TEST(HTTPScanTask, ExecUnknownCommand) {
    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    task::Command cmd;
    cmd.type = "unknown_command";
    EXPECT_FALSE(scanner.exec(cmd, task, ctx));
}

TEST(HTTPScanTask, ScanExecutesHealthChecksInParallel) {
    // Each server has a 300ms delay. With 5 servers, serial execution would take
    // ~1500ms. Parallel execution should complete well under 1000ms.
    constexpr int NUM_SERVERS = 5;
    const auto DELAY = x::telem::MILLISECOND * 300;
    constexpr int MAX_PARALLEL_MS = 1000;

    std::vector<std::unique_ptr<mock::Server>> servers;
    std::unordered_map<std::string, synnax::device::Device> devices;

    for (int i = 0; i < NUM_SERVERS; i++) {
        auto server = std::make_unique<mock::Server>(mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .response_body = "{}",
                .delay = DELAY,
            }},
        });
        ASSERT_NIL(server->start());
        auto dev = make_device(host_port_from_url(server->base_url()));
        devices[dev.key] = dev;
        servers.push_back(std::move(server));
    }

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    common::ScannerContext scan_ctx{.devices = &devices};

    const auto start = x::telem::TimeStamp::now();
    const auto result = ASSERT_NIL_P(scanner.scan(scan_ctx));
    const auto elapsed = x::telem::TimeStamp::now() - start;

    ASSERT_EQ(result.size(), NUM_SERVERS);
    for (const auto &dev: result)
        EXPECT_EQ(dev.status.variant, x::status::VARIANT_SUCCESS);

    EXPECT_LT(elapsed, x::telem::MILLISECOND * MAX_PARALLEL_MS)
        << "Scan took " << elapsed.milliseconds()
        << "ms — requests are likely executing in series";

    for (auto &s: servers)
        s->stop();
}

TEST(HTTPScanTask, TestConnectionInvalidArgs) {
    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    task::Command cmd;
    cmd.type = TEST_CONNECTION_CMD_TYPE;
    cmd.key = "cmd-1";
    cmd.args = x::json::json::object();

    EXPECT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_FALSE(ctx->statuses.empty());
    EXPECT_EQ(ctx->statuses.back().variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses.back().message, "Failed to parse test command");
    EXPECT_NE(ctx->statuses.back().description.find("connection"), std::string::npos);
    EXPECT_NE(ctx->statuses.back().description.find("health_check"), std::string::npos);
}

}
