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

#include "glog/logging.h"

#include "client/cpp/synnax.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/json/json.h"

#include "driver/modbus/device/device.h"
#include "driver/modbus/modbus.h"
#include "driver/task/common/scan_task.h"
#include "driver/task/common/status.h"
#include "driver/task/task.h"

namespace driver::modbus {
const std::string SCAN_LOG_PREFIX = "[" + INTEGRATION_NAME + ".scan_task]";
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

/// @brief Configuration for the Modbus scanner.
struct ScanTaskConfig : driver::task::common::ScanTaskConfig {
    ScanTaskConfig() = default;
    explicit ScanTaskConfig(x::json::Parser &cfg): driver::task::common::ScanTaskConfig(cfg) {}
};

/// @brief Arguments for testing connection to a Modbus server.
struct ScanCommandArgs {
    /// @brief Connection parameters for the device.
    device::ConnectionConfig connection;

    /// @brief Parses the arguments from their JSON object representation.
    explicit ScanCommandArgs(const x::json::Parser &parser):
        connection(device::ConnectionConfig(parser.child("connection"))) {}
};

/// @brief Modbus scanner implementing the driver::task::common::Scanner interface.
/// Handles device health monitoring for Modbus devices.
class Scanner final : public driver::task::common::Scanner {
public:
    Scanner(
        std::shared_ptr<driver::task::Context> ctx,
        synnax::task::Task task,
        std::shared_ptr<device::Manager> devices
    );

    /// @brief Returns scanner configuration for driver::task::common::ScanTask.
    [[nodiscard]] driver::task::common::ScannerConfig config() const override;

    /// @brief Periodic scan method - checks health of all tracked devices.
    std::pair<std::vector<synnax::Device>, x::errors::Error>
    scan(const driver::task::common::ScannerContext &scan_ctx) override;

    /// @brief Handle Modbus-specific commands (test connection).
    bool exec(
        driver::task::Command &cmd,
        const synnax::task::Task &task,
        const std::shared_ptr<driver::task::Context> &ctx
    ) override;

private:
    std::shared_ptr<driver::task::Context> ctx;
    synnax::task::Task task;
    std::shared_ptr<device::Manager> devices;

    /// @brief Test connection to a Modbus server.
    void test_connection(const driver::task::Command &cmd) const;

    /// @brief Check health of a single device by testing its connection.
    /// Sets dev.status based on connection result.
    void check_device_health(synnax::Device &dev) const;
};
}
