// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

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

/// @brief builds a synnax device with the given key and location.
/// check_device_health constructs base_url from "http://" + dev.location,
/// so set location to the mock server's host:port.
synnax::device::Device make_device(
    const std::string &key,
    const std::string &location,
    const x::json::json &extra_props = x::json::json::object()
) {
    synnax::device::Device dev;
    dev.key = key;
    dev.name = key;
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

////////////////////// HealthCheckConfig parsing //////////////////////

/// @brief it should parse a health check config with all fields.
TEST(HTTPScanTask, HealthCheckConfigParsesAllFields) {
    auto j = x::json::json{
        {"method", "POST"},
        {"path", "/api/status"},
        {"query_params", {{"key", "val"}}},
        {"headers", {{"X-Custom", "abc"}}},
        {"body", R"({"ping": true})"},
        {"response", {
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
    EXPECT_EQ(hc.response_pointer, "/status");
    EXPECT_EQ(hc.expected_value, "ok");
}

/// @brief it should use defaults when optional fields are omitted.
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
    EXPECT_TRUE(hc.response_pointer.empty());
}

////////////////////// Scanner::scan — device health checks //////////////////////

/// @brief it should mark a healthy device as connected.
TEST(HTTPScanTask, ScanHealthyDevice) {
    mock::Server server(mock::ServerConfig{
        .routes = {{.method = Method::GET, .path = "/health", .response_body = "{}"}},
    });
    ASSERT_NIL(server.start());

    auto dev = make_device("dev-1", host_port_from_url(server.base_url()));

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices["dev-1"] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(result[0].status.message, "Device connected");

    server.stop();
}

/// @brief it should mark device as warning when server is unreachable.
TEST(HTTPScanTask, ScanUnreachableDevice) {
    auto dev = make_device("dev-1", "127.0.0.1:1", {{"timeout_ms", 500}});

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices["dev-1"] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(result[0].status.message, "Failed to reach device");
}

/// @brief it should mark device as warning when response validation fails.
TEST(HTTPScanTask, ScanHealthCheckValidationFailure) {
    mock::Server server(mock::ServerConfig{
        .routes = {{
            .method = Method::GET,
            .path = "/health",
            .response_body = R"({"status": "degraded"})",
        }},
    });
    ASSERT_NIL(server.start());

    auto dev = make_device("dev-1", host_port_from_url(server.base_url()), {
        {"health_check", {
            {"method", "GET"},
            {"path", "/health"},
            {"response", {
                {"pointer", "/status"},
                {"expected_value", "ok"},
            }},
        }},
    });

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices["dev-1"] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(result[0].status.message, "Health check validation failed");

    server.stop();
}

/// @brief it should succeed validation when response matches expected value.
TEST(HTTPScanTask, ScanHealthCheckValidationSuccess) {
    mock::Server server(mock::ServerConfig{
        .routes = {{
            .method = Method::GET,
            .path = "/health",
            .response_body = R"({"status": "ok"})",
        }},
    });
    ASSERT_NIL(server.start());

    auto dev = make_device("dev-1", host_port_from_url(server.base_url()), {
        {"health_check", {
            {"method", "GET"},
            {"path", "/health"},
            {"response", {
                {"pointer", "/status"},
                {"expected_value", "ok"},
            }},
        }},
    });

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices["dev-1"] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(result[0].status.message, "Device connected");

    server.stop();
}

/// @brief it should return an empty list when no devices are tracked.
TEST(HTTPScanTask, ScanNoDevices) {
    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    common::ScannerContext scan_ctx{.devices = nullptr};
    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    EXPECT_TRUE(result.empty());
}

/// @brief it should scan multiple devices and report individual statuses.
TEST(HTTPScanTask, ScanMultipleDevices) {
    mock::Server server(mock::ServerConfig{
        .routes = {{.method = Method::GET, .path = "/health", .response_body = "{}"}},
    });
    ASSERT_NIL(server.start());

    auto healthy_dev = make_device(
        "dev-healthy",
        host_port_from_url(server.base_url())
    );
    auto bad_dev = make_device("dev-bad", "127.0.0.1:1", {{"timeout_ms", 500}});

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices["dev-healthy"] = healthy_dev;
    devices["dev-bad"] = bad_dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(result.size(), 2);

    for (const auto &dev : result) {
        if (dev.key == "dev-healthy") {
            EXPECT_EQ(dev.status.variant, x::status::VARIANT_SUCCESS);
        } else if (dev.key == "dev-bad") {
            EXPECT_EQ(dev.status.variant, x::status::VARIANT_WARNING);
        } else {
            FAIL() << "Unexpected device key: " << dev.key;
        }
    }

    server.stop();
}

/// @brief it should warn when device properties are invalid JSON config.
TEST(HTTPScanTask, ScanInvalidDeviceProperties) {
    synnax::device::Device dev;
    dev.key = "dev-bad";
    dev.name = "Bad Device";
    dev.make = INTEGRATION_NAME;
    dev.location = "127.0.0.1:0";
    dev.properties = x::json::json{{"secure", false}};

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices["dev-bad"] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_WARNING);
}

////////////////////// Scanner::exec — test_connection command //////////////////////

/// @brief it should succeed test_connection when server is healthy.
TEST(HTTPScanTask, TestConnectionSuccess) {
    mock::Server server(mock::ServerConfig{
        .routes = {{.method = Method::GET, .path = "/health", .response_body = "{}"}},
    });
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
        {"connection", {
            {"base_url", server.base_url()},
            {"timeout_ms", 1000},
            {"verify_ssl", false},
        }},
        {"health_check", {
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

/// @brief it should fail test_connection when server is unreachable.
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
        {"connection", {
            {"base_url", "http://127.0.0.1:1"},
            {"timeout_ms", 500},
            {"verify_ssl", false},
        }},
        {"health_check", {
            {"method", "GET"},
            {"path", "/health"},
        }},
    };

    EXPECT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_FALSE(ctx->statuses.empty());
    EXPECT_EQ(ctx->statuses.back().variant, x::status::VARIANT_ERROR);
}

/// @brief it should fail test_connection when response validation fails.
TEST(HTTPScanTask, TestConnectionValidationFailure) {
    mock::Server server(mock::ServerConfig{
        .routes = {{
            .method = Method::GET,
            .path = "/health",
            .response_body = R"({"status": "bad"})",
        }},
    });
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
        {"connection", {
            {"base_url", server.base_url()},
            {"timeout_ms", 1000},
            {"verify_ssl", false},
        }},
        {"health_check", {
            {"method", "GET"},
            {"path", "/health"},
            {"response", {
                {"pointer", "/status"},
                {"expected_value", "ok"},
            }},
        }},
    };

    EXPECT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_FALSE(ctx->statuses.empty());
    EXPECT_EQ(ctx->statuses.back().variant, x::status::VARIANT_ERROR);

    server.stop();
}

/// @brief it should not handle unknown command types.
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

/// @brief it should fail test_connection with invalid args.
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
}

////////////////////// Scanner::config //////////////////////

/// @brief it should return the correct scanner config.
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

////////////////////// Health check with POST body //////////////////////

/// @brief it should send a POST body for health check when configured.
TEST(HTTPScanTask, ScanWithPOSTHealthCheck) {
    mock::Server server(mock::ServerConfig{
        .routes = {{
            .method = Method::POST,
            .path = "/ping",
            .response_body = R"({"alive": true})",
        }},
    });
    ASSERT_NIL(server.start());

    auto dev = make_device("dev-post", host_port_from_url(server.base_url()), {
        {"health_check", {
            {"method", "POST"},
            {"path", "/ping"},
            {"body", R"({"check": "heartbeat"})"},
            {"response", {
                {"pointer", "/alive"},
                {"expected_value", true},
            }},
        }},
    });

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices["dev-post"] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_SUCCESS);

    auto received = server.received_requests();
    ASSERT_FALSE(received.empty());
    EXPECT_EQ(received[0].method, Method::POST);
    EXPECT_EQ(received[0].body, R"({"check": "heartbeat"})");

    server.stop();
}

/// @brief it should warn when response body is not valid JSON and pointer is set.
TEST(HTTPScanTask, ScanHealthCheckNonJSONResponse) {
    mock::Server server(mock::ServerConfig{
        .routes = {{
            .method = Method::GET,
            .path = "/health",
            .response_body = "not json",
            .content_type = "text/plain",
        }},
    });
    ASSERT_NIL(server.start());

    auto dev = make_device("dev-1", host_port_from_url(server.base_url()), {
        {"health_check", {
            {"method", "GET"},
            {"path", "/health"},
            {"response", {
                {"pointer", "/status"},
                {"expected_value", "ok"},
            }},
        }},
    });

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices["dev-1"] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(result[0].status.message, "Health check validation failed");

    server.stop();
}

/// @brief it should warn when response JSON doesn't contain the expected pointer.
TEST(HTTPScanTask, ScanHealthCheckMissingPointer) {
    mock::Server server(mock::ServerConfig{
        .routes = {{
            .method = Method::GET,
            .path = "/health",
            .response_body = R"({"other": "value"})",
        }},
    });
    ASSERT_NIL(server.start());

    auto dev = make_device("dev-1", host_port_from_url(server.base_url()), {
        {"health_check", {
            {"method", "GET"},
            {"path", "/health"},
            {"response", {
                {"pointer", "/status"},
                {"expected_value", "ok"},
            }},
        }},
    });

    std::unordered_map<std::string, synnax::device::Device> devices;
    devices["dev-1"] = dev;
    common::ScannerContext scan_ctx{.devices = &devices};

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 100);
    task.name = "HTTP Scanner";

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto processor = std::make_shared<Processor>();
    Scanner scanner(ctx, task, processor);

    auto [result, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(result.size(), 1);
    EXPECT_EQ(result[0].status.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(result[0].status.message, "Health check validation failed");

    server.stop();
}
}
