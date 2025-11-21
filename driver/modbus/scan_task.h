// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>

#include "x/cpp/defer/defer.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/modbus/device/device.h"
#include "driver/task/common/status.h"
#include "driver/task/task.h"

namespace modbus {
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

/// @brief arguments for scanning a Modbus server on the network.
struct ScanCommandArgs {
    /// @brief connection parameters for the device.
    device::ConnectionConfig connection;

    /// @brief parses the arguments from their JSON object representation.
    explicit ScanCommandArgs(const xjson::Parser &parser):
        connection(device::ConnectionConfig(parser.child("connection"))) {}
};

/// @brief scans for Modbus servers.
class ScanTask final : public task::Task {
    /// @param ctx the task context used to communicate state changes back to Synnax.
    std::shared_ptr<task::Context> ctx;
    /// @param the task representation in Synnax.
    synnax::Task task;
    /// @brief the device manager used to acquire connections to Modbus servers.
    std::shared_ptr<device::Manager> devices;

    /// @brief tests the connection to a Modbus server.
    void test_connection(const task::Command &cmd) const {
        xjson::Parser parser(cmd.args);
        const ScanCommandArgs args(parser);
        synnax::TaskStatus status;
        status.key = task.status_key();
        status.details.task = task.key;
        status.details.running = true;
        status.details.cmd = cmd.key;
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
            status.variant = status::variant::SUCCESS;
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
        if (cmd.type == common::START_CMD_TYPE) {
            synnax::TaskStatus status{
                .key = this->task.status_key(),
                .name = this->task.name,
                .variant = status::variant::SUCCESS,
                .message = "Running",
                .details = synnax::TaskStatusDetails{
                    .task = task.key,
                    .running = true,
                    .cmd = cmd.key,
                }
            };
            ctx->set_status(status);
            return;
        }
        if (cmd.type == TEST_CONNECTION_CMD_TYPE) this->test_connection(cmd);
    }

    [[nodiscard]] std::string name() const override { return this->task.name; }

    void stop(bool will_reconfigure) override {}
};
}
