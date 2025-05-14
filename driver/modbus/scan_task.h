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

struct ScanCommandArgs {
    device::ConnectionConfig connection;

    explicit ScanCommandArgs(
        const xjson::Parser &parser
    ): connection(device::ConnectionConfig(parser.child("connection"))) {
    }
};

class ScanTask final : public task::Task {
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    std::shared_ptr<device::Manager> devices;

    void test_connection(const task::Command &cmd) const {
        xjson::Parser parser(cmd.args);
        const ScanCommandArgs args(parser);
        task::State state;
        state.task = task.key;
        state.key = cmd.key;
        x::defer set_state([&] { this->ctx->set_state(state); });
        if (!parser.ok()) {
            state.details = parser.error_json();
            return;
        }
        auto [dev, err] = this->devices->acquire(args.connection);
        if (err) {
            state.variant = "error";
            state.details = {{"message", err.data}};
        } else {
            state.variant = "success";
            state.details = {{"message", "Connection successful"}};
        }
    }

public:
    explicit ScanTask(
        const std::shared_ptr<task::Context> &context,
        synnax::Task task,
        const std::shared_ptr<device::Manager> &devices
    ): ctx(context), task(std::move(task)), devices(devices) {
    }

    void exec(task::Command &cmd) override {
        if (cmd.type == TEST_CONNECTION_CMD_TYPE) this->test_connection(cmd);
    }

    std::string name() const override { return this->task.name; }

    void stop(bool will_reconfigure) override {
    }
};
}
