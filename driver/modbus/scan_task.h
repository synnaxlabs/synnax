// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>

#include "x/cpp/json/json.h"

#include "driver/common/scan_task.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/modbus.h"
#include "driver/task/task.h"

namespace driver::modbus {
const std::string SCAN_LOG_PREFIX = "[" + INTEGRATION_NAME + ".scan_task]";
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

/// @brief Configuration for the Modbus scanner.
struct ScanTaskConfig : common::ScanTaskConfig {
    ScanTaskConfig() = default;
    explicit ScanTaskConfig(x::json::Parser &cfg): common::ScanTaskConfig(cfg) {}
};

/// @brief Arguments for testing connection to a Modbus server.
struct ScanCommandArgs {
    /// @brief Connection parameters for the device.
    device::ConnectionConfig connection;
    /// @brief how the device paces its requests. A missing object takes every default.
    device::RequestConfig requests;

    /// @brief Parses the arguments from their JSON object representation.
    explicit ScanCommandArgs(const x::json::Parser &parser):
        connection(device::ConnectionConfig(parser.child("connection"))),
        requests(parser.optional_child("requests")) {}
};

/// @brief Modbus scanner implementing the common::Scanner interface. It checks each
/// device on the connection its tasks share, and keeps that connection open between
/// scans.
class Scanner final : public common::Scanner {
public:
    Scanner(
        std::shared_ptr<task::Context> ctx,
        synnax::task::Task task,
        std::shared_ptr<device::Manager> devices
    );

    /// @brief Returns scanner configuration for common::ScanTask.
    [[nodiscard]] common::ScannerConfig config() const override;

    /// @brief Periodic scan method - checks health of all tracked devices.
    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &scan_ctx) override;

    /// @brief Handle Modbus-specific commands (test connection).
    bool exec(
        synnax::task::Command &cmd,
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ) override;

private:
    /// @brief a device the scanner checks, held between scans.
    struct Check {
        /// @brief the device on its shared connection.
        std::shared_ptr<device::Device> dev;
        /// @brief when the scanner last checked the device.
        device::Connection::Clock::time_point time;
    };

    std::shared_ptr<task::Context> ctx;
    synnax::task::Task task;
    std::shared_ptr<device::Manager> devices;
    /// @brief guards probing and checks. Scans and connection tests run on different
    /// threads.
    std::mutex mu;
    /// @brief signals the end of a probe.
    std::condition_variable probed;
    /// @brief the endpoints that a scan probes now.
    std::unordered_set<std::string> probing;
    /// @brief the latest check of each device, by key.
    std::unordered_map<std::string, Check> checks;

    /// @brief Test connection to a Modbus server.
    void test_connection(const synnax::task::Command &cmd);

    /// @brief waits for a probe of the serial port of cfg to end, then releases the
    /// checks that hold the port open through another path or with other line
    /// settings, so that a connection test can use it. The next scan takes it back.
    /// @param lock holds mu.
    void release_stale(
        std::unique_lock<std::mutex> &lock,
        const device::ConnectionConfig &cfg
    );

    /// @brief sets the status of a device. A device that answered a task since the last
    /// scan is healthy. Otherwise the scanner probes it.
    void check_device_health(synnax::device::Device &dev);
};
}
