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
    ScanTaskConfig() = default;
    explicit ScanTaskConfig(x::json::Parser &cfg): common::ScanTaskConfig(cfg) {}
};

/// @brief arguments for the test_connection command.
struct ScanCommandArgs {
    /// @brief connection configuration to test.
    device::ConnectionConfig connection;

    explicit ScanCommandArgs(const x::json::Parser &parser):
        connection(device::ConnectionConfig(parser.child("connection"))) {}
};

/// @brief HTTP scanner implementing the common::Scanner interface.
/// Handles device health monitoring by pinging each HTTP device.
class Scanner final : public common::Scanner {
public:
    Scanner(
        std::shared_ptr<task::Context> ctx,
        synnax::task::Task task,
        std::shared_ptr<Processor> processor
    );

    /// @brief returns scanner configuration for common::ScanTask.
    [[nodiscard]] common::ScannerConfig config() const override;

    /// @brief periodic scan method - checks health of all tracked devices.
    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &scan_ctx) override;

    /// @brief handle HTTP-specific commands (test connection).
    bool exec(
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

    /// @brief check health of a single device by pinging it.
    void check_device_health(synnax::device::Device &dev) const;
};
}
