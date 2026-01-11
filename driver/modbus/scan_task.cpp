// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <utility>

#include "glog/logging.h"

#include "x/cpp/json/json.h"

#include "driver/modbus/scan_task.h"
#include "driver/task/common/status.h"

namespace driver::modbus {
Scanner::Scanner(
    std::shared_ptr<driver::task::Context> ctx,
    synnax::task::Task task,
    std::shared_ptr<device::Manager> devices
):
    ctx(std::move(ctx)), task(std::move(task)), devices(std::move(devices)) {}

driver::task::common::ScannerConfig Scanner::config() const {
    return driver::task::common::ScannerConfig{
        .make = INTEGRATION_NAME,
        .log_prefix = SCAN_LOG_PREFIX,
    };
}

std::pair<std::vector<synnax::Device>, x::errors::Error>
Scanner::scan(const driver::task::common::ScannerContext &scan_ctx) {
    std::vector<synnax::Device> devices_out;
    if (scan_ctx.devices == nullptr) return {devices_out, x::errors::NIL};
    for (auto [key, dev]: *scan_ctx.devices) {
        this->check_device_health(dev);
        devices_out.push_back(dev);
    }
    return {devices_out, x::errors::NIL};
}

bool Scanner::exec(
    driver::task::Command &cmd,
    const synnax::task::Task &,
    const std::shared_ptr<driver::task::Context> &
) {
    if (cmd.type == TEST_CONNECTION_CMD_TYPE) {
        this->test_connection(cmd);
        return true;
    }
    return false;
}

void Scanner::check_device_health(synnax::Device &dev) const {
    const auto rack_key = synnax::rack_key_from_task_key(this->task.key);
    const auto parser = x::json::Parser(dev.properties);
    const auto conn_cfg = device::ConnectionConfig(parser.child("connection"));
    if (parser.error()) {
        dev.status = synnax::DeviceStatus{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = status::variant::WARNING,
            .message = "Invalid device properties",
            .description = parser.error().message(),
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
        return;
    }

    auto [conn, conn_err] = this->devices->acquire(conn_cfg);
    if (conn_err)
        dev.status = synnax::DeviceStatus{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = status::variant::WARNING,
            .message = "Failed to reach device",
            .description = conn_err.message(),
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
    else
        dev.status = synnax::DeviceStatus{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = status::variant::SUCCESS,
            .message = "Device connected",
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
}

void Scanner::test_connection(const driver::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::task::Status status{
        .key = this->task.status_key(),
        .name = this->task.name,
        .variant = status::variant::ERR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .cmd = cmd.key,
            .running = true,
        }
    };
    if (!parser.ok()) {
        status.message = "Failed to parse test command";
        status.details.data = parser.error_json();
        return ctx->set_status(status);
    }
    auto [dev, err] = this->devices->acquire(args.connection);
    if (err) {
        status.message = err.data;
        return ctx->set_status(status);
    }
    status.variant = status::variant::SUCCESS;
    status.message = "Connection successful";
    return ctx->set_status(status);
}
}
