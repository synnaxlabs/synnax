// Copyright 2025 Synnax Labs, Inc.
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

#include "x/cpp/xjson/xjson.h"

#include "driver/modbus/scan_task.h"
#include "driver/task/common/status.h"

namespace modbus {
Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::Task task,
    std::shared_ptr<device::Manager> devices,
    const ScannerConfig cfg
):
    ctx(std::move(ctx)), task(std::move(task)), devices(std::move(devices)), cfg(cfg) {}

common::ScannerConfig Scanner::config() const {
    return common::ScannerConfig{
        .make = INTEGRATION_NAME,
        .log_prefix = SCAN_LOG_PREFIX,
    };
}

std::pair<std::vector<synnax::Device>, xerrors::Error>
Scanner::scan(const common::ScannerContext &scan_ctx) {
    std::vector<synnax::Device> devices_out;
    if (scan_ctx.devices == nullptr) return {devices_out, xerrors::NIL};
    for (auto [key, dev]: *scan_ctx.devices) {
        if (const auto err = this->check_device_health(dev); err)
            LOG(WARNING) << SCAN_LOG_PREFIX << "health check failed for " << dev.name
                         << ": " << err;
        devices_out.push_back(dev);
    }
    return {devices_out, xerrors::NIL};
}

bool Scanner::exec(
    task::Command &cmd,
    const synnax::Task &,
    const std::shared_ptr<task::Context> &
) {
    if (cmd.type == TEST_CONNECTION_CMD_TYPE) {
        this->test_connection(cmd);
        return true;
    }
    return false;
}

xerrors::Error Scanner::check_device_health(synnax::Device &dev) const {
    const auto rack_key = synnax::rack_key_from_task_key(this->task.key);
    const auto parser = xjson::Parser(dev.properties);
    const auto conn_cfg = device::ConnectionConfig(parser.child("connection"));
    if (parser.error()) {
        dev.status = synnax::DeviceStatus{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = status::variant::WARNING,
            .message = "Invalid device properties",
            .description = parser.error().message(),
            .time = telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
        return parser.error();
    }

    auto [conn, conn_err] = this->devices->acquire(conn_cfg);
    if (conn_err)
        dev.status = synnax::DeviceStatus{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = status::variant::WARNING,
            .message = "Failed to reach device",
            .description = conn_err.message(),
            .time = telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
    else
        dev.status = synnax::DeviceStatus{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = status::variant::SUCCESS,
            .message = "Device connected",
            .time = telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
    return xerrors::NIL;
}

void Scanner::test_connection(const task::Command &cmd) const {
    xjson::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::TaskStatus status{
        .key = this->task.status_key(),
        .name = this->task.name,
        .variant = status::variant::ERR,
        .details = synnax::TaskStatusDetails{
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
