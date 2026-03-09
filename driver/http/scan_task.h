// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/json/json.h"

#include "driver/common/scan_task.h"
#include "driver/http/device/device.h"
#include "driver/http/http.h"
#include "driver/http/processor/processor.h"
#include "driver/task/task.h"

namespace driver::http {
const std::string SCAN_TASK_TYPE = INTEGRATION_NAME + "_scan";
const std::string SCAN_LOG_PREFIX = "[" + INTEGRATION_NAME + ".scan_task]";
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

/// @brief configuration for the HTTP scanner.
struct ScanTaskConfig : common::ScanTaskConfig {
    [[nodiscard]] explicit ScanTaskConfig(x::json::Parser &cfg):
        common::ScanTaskConfig(cfg) {}
};

/// @brief optional expected response validation config. When set, the health check
/// validates that the JSON value at the given pointer matches the expected value.
struct ExpectedResponseConfig {
    /// @brief JSON Pointer into the response body (e.g. "/status").
    std::string pointer;
    /// @brief expected JSON value at the pointer.
    x::json::json expected_value;

    [[nodiscard]] explicit ExpectedResponseConfig(x::json::Parser parser):
        pointer(parser.field<std::string>("pointer")),
        expected_value(parser.field<x::json::json>("expected_value")) {}
};

/// @brief configurable health check request and optional response validation.
struct HealthCheckConfig {
    /// @brief request configuration (method, path, headers, query params).
    RequestConfig request;
    /// @brief optional request body.
    std::string body;
    /// @brief optional expected response validation.
    std::optional<ExpectedResponseConfig> expected_response;

    [[nodiscard]] explicit HealthCheckConfig(x::json::Parser parser):
        request{
            .method = parse_method(parser, "method"),
            .path = parser.field<std::string>("path"),
            .query_params = parser.field<std::map<std::string, std::string>>(
                "query_params",
                std::map<std::string, std::string>{}
            ),
            .headers = parser.field<std::map<std::string, std::string>>(
                "headers",
                std::map<std::string, std::string>{}
            ),
        },
        body(parser.field<std::string>("body", "")) {
        auto resp = parser.optional_child("response");
        if (resp.ok()) expected_response.emplace(ExpectedResponseConfig(resp));
    }
};

/// @brief arguments for the test_connection command.
struct ScanCommandArgs {
    /// @brief connection configuration to test.
    device::ConnectionConfig connection;
    /// @brief health check configuration.
    HealthCheckConfig health_check;

    [[nodiscard]] explicit ScanCommandArgs(const x::json::Parser &parser):
        connection(device::ConnectionConfig(parser.child("connection"))),
        health_check(HealthCheckConfig(parser.child("health_check"))) {}
};

/// @brief HTTP scanner implementing the common::Scanner interface. Handles device
/// health monitoring by pinging each HTTP device.
class Scanner final : public common::Scanner {
public:
    [[nodiscard]] Scanner(
        std::shared_ptr<task::Context> ctx,
        synnax::task::Task task,
        std::shared_ptr<Processor> processor
    );

    /// @brief returns scanner configuration for common::ScanTask.
    [[nodiscard]] common::ScannerConfig config() const override;

    /// @brief periodic scan method - checks health of all tracked devices.
    [[nodiscard]] std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &scan_ctx) override;

    /// @brief handle HTTP-specific commands (test connection).
    [[nodiscard]] bool exec(
        task::Command &cmd,
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ) override;

private:
    std::shared_ptr<task::Context> ctx;
    synnax::task::Task task;
    std::shared_ptr<Processor> processor;

    /// @brief test connection to an HTTP server.
    void test_connection(const task::Command &cmd) const;

    /// @brief holds a successfully built health check request along with the
    /// information needed to validate the response and update the device status.
    struct PreparedHealthCheck {
        /// @brief index into the devices_out vector.
        std::size_t device_index;
        /// @brief optional expected response validation config.
        std::optional<ExpectedResponseConfig> expected_response;
        /// @brief the built HTTP request.
        Request request;
    };

    /// @brief attempts to build a health check request for a device. If config parsing
    /// fails, sets the device status and returns nullopt.
    [[nodiscard]] std::optional<PreparedHealthCheck>
    prepare_health_check(synnax::device::Device &dev, std::size_t device_index) const;

    /// @brief processes a health check response and sets the device status.
    void process_health_response(
        synnax::device::Device &dev,
        const std::optional<ExpectedResponseConfig> &expected_response,
        const Response &resp,
        const x::errors::Error &err
    ) const;
};
}
