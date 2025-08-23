// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>
#include <utility>

/// module
#include "x/cpp/defer/defer.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/modbus/device/device.h"
#include "driver/task/task.h"

namespace modbus {
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

/// @brief arguments for scanning a modbus device on the network.
struct ScanCommandArgs {
    /// @brief connection parameters for the device.
    device::ConnectionConfig connection;

    /// @brief parses the arguments from their JSON object representation.
    explicit ScanCommandArgs(const xjson::Parser &parser):
        connection(device::ConnectionConfig(parser.child("connection"))) {}
};

/// @brief scans for modbus devices.
class ScanTask final : public task::Task {
    /// @param ctx the task context used to communicate state changes back to Synnax.
    std::shared_ptr<task::Context> ctx;
    /// @param the task representation in Synnax.
    synnax::Task task;
    /// @brief the device manager used to acquire connections to modbus devices.
    std::shared_ptr<device::Manager> devices;

    /// @brief tests the connection to a modbus device.
    void test_connection(const task::Command &cmd) const {
        xjson::Parser parser(cmd.args);
        const ScanCommandArgs args(parser);
        synnax::TaskStatus status;
        status.key = cmd.key;
        status.details.task = task.key;
        status.details.running = true;
        x::defer set_state([&] { this->ctx->set_status(status); });
        if (!parser.ok()) {
            status.details.data = parser.error_json();
            return;
        }
        auto [dev, err] = this->devices->acquire(args.connection);
        if (err) {
            status.variant = "error";
            status.message = err.data;
        } else {
            status.variant = "success";
            status.message = "Connection successful";
        }
    }

public:
    explicit ScanTask(
        const std::shared_ptr<task::Context> &context,
        synnax::Task task,
        const std::shared_ptr<device::Manager> &devices
    ):
        ctx(context), task(std::move(task)), devices(devices) {}

    void exec(task::Command &cmd) override {
        if (cmd.type == TEST_CONNECTION_CMD_TYPE) this->test_connection(cmd);
    }

    [[nodiscard]] std::string name() const override { return this->task.name; }

    void stop(bool will_reconfigure) override {}
};
}
