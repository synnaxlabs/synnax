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

#include "driver/http/mock/server.h"
#include "driver/http/scan_task.h"

namespace driver::http {
namespace {
std::unique_ptr<ScanTask> make_scan_task(
    const std::shared_ptr<task::MockContext> &ctx,
    const std::string &base_url,
    std::optional<ResponseConfig> response = std::nullopt
) {
    auto conn_parser = x::json::Parser(
        x::json::json{
            {"base_url", base_url},
            {"timeout_ms", 1000},
        }
    );
    auto conn = device::ConnectionConfig(conn_parser, false);
    ScanTaskConfig cfg{
        .device = "test-device",
        .auto_start = false,
        .rate = x::telem::Rate(20),
        .path = "/health",
        .response = std::move(response),
    };
    synnax::task::Task task;
    task.key = 1;
    task.name = "test-scan";
    task.type = SCAN_TASK_TYPE;
    return std::make_unique<ScanTask>(ctx, task, std::move(cfg), std::move(conn));
}

void start_task(ScanTask &t) {
    task::Command cmd(1, common::START_CMD_TYPE, {});
    t.exec(cmd);
}

bool has_warning(const std::shared_ptr<task::MockContext> &ctx) {
    for (const auto &s: ctx->statuses)
        if (s.variant == x::status::VARIANT_WARNING) return true;
    return false;
}

bool has_success(const std::shared_ptr<task::MockContext> &ctx) {
    for (const auto &s: ctx->statuses)
        if (s.variant == x::status::VARIANT_SUCCESS) return true;
    return false;
}

std::string last_warning_message(const std::shared_ptr<task::MockContext> &ctx) {
    for (auto it = ctx->statuses.rbegin(); it != ctx->statuses.rend(); ++it)
        if (it->variant == x::status::VARIANT_WARNING) return it->message;
    return "";
}
}

/// @brief it should parse a valid ResponseConfig.
TEST(ResponseConfig, testParseValid) {
    auto parser = x::json::Parser(
        x::json::json{
            {"field", "/status"},
            {"expected_value", "ok"},
        }
    );
    ResponseConfig cfg(parser);
    ASSERT_TRUE(parser.ok()) << parser.error();
    EXPECT_EQ(cfg.field, "/status");
    EXPECT_EQ(cfg.expected_value, "ok");
}

/// @brief it should parse a ResponseConfig with a numeric expected value.
TEST(ResponseConfig, testParseNumericExpectedValue) {
    auto parser = x::json::Parser(
        x::json::json{
            {"field", "/uptime"},
            {"expected_value", 42},
        }
    );
    ResponseConfig cfg(parser);
    ASSERT_TRUE(parser.ok()) << parser.error();
    EXPECT_EQ(cfg.field, "/uptime");
    EXPECT_EQ(cfg.expected_value, 42);
}

/// @brief it should fail to parse when the field is missing.
TEST(ResponseConfig, testParseMissingField) {
    auto parser = x::json::Parser(
        x::json::json{
            {"expected_value", "ok"},
        }
    );
    ResponseConfig cfg(parser);
    EXPECT_FALSE(parser.ok());
}

/// @brief it should fail to parse when expected_value is missing.
TEST(ResponseConfig, testParseMissingExpectedValue) {
    auto parser = x::json::Parser(
        x::json::json{
            {"field", "/status"},
        }
    );
    ResponseConfig cfg(parser);
    EXPECT_FALSE(parser.ok());
}

/// @brief it should fail to parse when the device field is missing.
TEST(ScanTask, testParseConfigMissingDevice) {
    synnax::task::Task task;
    task.config = {{"path", "/health"}};
    ASSERT_OCCURRED_AS_P(ScanTaskConfig::parse(task), x::errors::VALIDATION);
}

/// @brief it should fail to parse when the path field is missing.
TEST(ScanTask, testParseConfigMissingPath) {
    synnax::task::Task task;
    task.config = {{"device", "dev-001"}};
    ASSERT_OCCURRED_AS_P(ScanTaskConfig::parse(task), x::errors::VALIDATION);
}

/// @brief it should parse the task config correctly.
TEST(ScanTask, testParseConfig) {
    synnax::task::Task task;
    task.config = {
        {"device", "dev-001"},
        {"auto_start", true},
        {"rate", 0.5},
        {"path", "/api/health"},
        {"response", {{"field", "/status"}, {"expected_value", "ok"}}},
    };
    const auto cfg = ASSERT_NIL_P(ScanTaskConfig::parse(task));
    EXPECT_EQ(cfg.device, "dev-001");
    EXPECT_TRUE(cfg.auto_start);
    EXPECT_EQ(cfg.path, "/api/health");
    ASSERT_TRUE(cfg.response.has_value());
    EXPECT_EQ(cfg.response->field, "/status");
    EXPECT_EQ(cfg.response->expected_value, "ok");
}

/// @brief it should report no warnings when the health endpoint returns 200.
TEST(ScanTask, testHealthCheckSuccess) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .status_code = 200,
                .response_body = R"({"status":"ok"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto scan_task = make_scan_task(ctx, server.base_url());
    start_task(*scan_task);
    ASSERT_EVENTUALLY_TRUE(has_success(ctx));
    scan_task->stop(false);
    EXPECT_FALSE(has_warning(ctx));
}

/// @brief it should report WARNING when no server is reachable.
TEST(ScanTask, testHealthCheckUnreachable) {
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto scan_task = make_scan_task(ctx, "http://127.0.0.1:19999");
    start_task(*scan_task);
    ASSERT_EVENTUALLY_TRUE(has_warning(ctx));
    scan_task->stop(false);
    auto msg = last_warning_message(ctx);
    EXPECT_TRUE(msg.find("Failed to reach device") != std::string::npos);
}

/// @brief it should report no warnings when response validation passes.
TEST(ScanTask, testHealthCheckWithResponseValidation) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .status_code = 200,
                .response_body = R"({"status":"ok","uptime":123})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto resp_parser = x::json::Parser(
        x::json::json{
            {"field", "/status"},
            {"expected_value", "ok"},
        }
    );
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto scan_task = make_scan_task(
        ctx,
        server.base_url(),
        ResponseConfig(resp_parser)
    );
    start_task(*scan_task);
    ASSERT_EVENTUALLY_TRUE(has_success(ctx));
    scan_task->stop(false);
    EXPECT_FALSE(has_warning(ctx));
}

/// @brief it should report WARNING when response validation fails.
TEST(ScanTask, testHealthCheckWithFailedResponseValidation) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .status_code = 200,
                .response_body = R"({"status":"error"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto resp_parser = x::json::Parser(
        x::json::json{
            {"field", "/status"},
            {"expected_value", "ok"},
        }
    );
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto scan_task = make_scan_task(
        ctx,
        server.base_url(),
        ResponseConfig(resp_parser)
    );
    start_task(*scan_task);
    ASSERT_EVENTUALLY_TRUE(has_warning(ctx));
    scan_task->stop(false);
    auto msg = last_warning_message(ctx);
    EXPECT_TRUE(msg.find("Unexpected health response") != std::string::npos);
}

/// @brief it should report WARNING when the response value has the wrong JSON type
/// (e.g. expected integer 1, got string "1").
TEST(ScanTask, testHealthCheckTypeMismatch) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .status_code = 200,
                .response_body = R"({"code":"1"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto resp_parser = x::json::Parser(
        x::json::json{
            {"field", "/code"},
            {"expected_value", 1},
        }
    );
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto scan_task = make_scan_task(
        ctx,
        server.base_url(),
        ResponseConfig(resp_parser)
    );
    start_task(*scan_task);
    ASSERT_EVENTUALLY_TRUE(has_warning(ctx));
    scan_task->stop(false);
    auto msg = last_warning_message(ctx);
    EXPECT_TRUE(msg.find("Unexpected health response") != std::string::npos);
}

/// @brief it should report WARNING when the server returns a non-2xx status.
TEST(ScanTask, testHealthCheckNon2xx) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .status_code = 500,
                .response_body = R"({"error":"internal"})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto scan_task = make_scan_task(ctx, server.base_url());
    start_task(*scan_task);
    ASSERT_EVENTUALLY_TRUE(has_warning(ctx));
    scan_task->stop(false);
    auto msg = last_warning_message(ctx);
    EXPECT_EQ(msg, "Device returned HTTP 500");
}

/// @brief it should report WARNING when the response field is not found.
TEST(ScanTask, testHealthCheckMissingField) {
    mock::Server server(
        mock::ServerConfig{
            .routes = {{
                .method = Method::GET,
                .path = "/health",
                .status_code = 200,
                .response_body = R"({"uptime":123})",
            }},
        }
    );
    ASSERT_NIL(server.start());
    x::defer::defer stop_server([&server] { server.stop(); });

    auto resp_parser = x::json::Parser(
        x::json::json{
            {"field", "/status"},
            {"expected_value", "ok"},
        }
    );
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto scan_task = make_scan_task(
        ctx,
        server.base_url(),
        ResponseConfig(resp_parser)
    );
    start_task(*scan_task);
    ASSERT_EVENTUALLY_TRUE(has_warning(ctx));
    scan_task->stop(false);
    auto msg = last_warning_message(ctx);
    EXPECT_TRUE(msg.find("not found") != std::string::npos);
}
}
