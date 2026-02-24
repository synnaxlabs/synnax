// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <optional>
#include <string>
#include <utility>

#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/common/common.h"
#include "driver/common/status.h"
#include "driver/http/device/device.h"
#include "driver/http/http.h"
#include "driver/pipeline/base.h"
#include "driver/task/task.h"

namespace driver::http {
const std::string SCAN_TASK_TYPE = INTEGRATION_NAME + "_scan";

/// @brief optional response body validation for a health-check endpoint.
struct ResponseConfig {
    /// @brief JSON Pointer (RFC 6901) to the field to validate.
    x::json::json::json_pointer field;
    /// @brief expected value at the field (any JSON type).
    x::json::json expected_value;

    explicit ResponseConfig(x::json::Parser parser):
        field(parser.field<std::string>("field")),
        expected_value(parser.field<x::json::json>("expected_value")) {}
};

/// @brief configuration for an HTTP scan (health-check) task.
struct ScanTaskConfig {
    /// @brief key of the device to health-check.
    std::string device;
    /// @brief whether to auto-start the task.
    bool auto_start;
    /// @brief health check frequency.
    x::telem::Rate rate;
    /// @brief endpoint path for the health check.
    std::string path;
    /// @brief optional response body validation.
    std::optional<ResponseConfig> response;

    /// @brief parses the scan task config from the task's JSON config.
    static std::pair<ScanTaskConfig, x::errors::Error>
    parse(const synnax::task::Task &task);
};

/// @brief a per-device health-check task that periodically probes an HTTP endpoint and
/// updates the device's status in the cluster.
class ScanTask final : public task::Task, private pipeline::Base {
    using pipeline::Base::stop;

public:
    ScanTask(
        std::shared_ptr<task::Context> ctx,
        synnax::task::Task task,
        ScanTaskConfig cfg,
        device::ConnectionConfig conn
    );

    /// @brief handles start and stop commands.
    void exec(task::Command &cmd) override;

    /// @brief stops the health-check loop.
    void stop(bool will_reconfigure) override;

    [[nodiscard]] std::string name() const override { return task.name; }

private:
    std::shared_ptr<task::Context> ctx;
    synnax::task::Task task;
    ScanTaskConfig cfg;
    device::ConnectionConfig conn;
    common::StatusHandler status_handler;

    /// @brief the main health-check loop.
    void run() override;

    /// @brief updates the device status in the cluster.
    void set_device_status(const std::string &variant, const std::string &message);
};

/// @brief configures a scan task from a synnax task definition.
std::pair<common::ConfigureResult, x::errors::Error> configure_scan(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
);
}
