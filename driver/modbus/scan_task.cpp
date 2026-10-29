// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <mutex>
#include <utility>

#include "x/cpp/json/json.h"

#include "driver/modbus/scan_task.h"

namespace driver::modbus {
Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::task::Task task,
    std::shared_ptr<device::Manager> devices
):
    ctx(std::move(ctx)), task(std::move(task)), devices(std::move(devices)) {}

common::ScannerConfig Scanner::config() const {
    return common::ScannerConfig{
        .make = MAKE,
        .log_prefix = SCAN_LOG_PREFIX,
    };
}

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &scan_ctx) {
    std::vector<synnax::device::Device> devices_out;
    if (scan_ctx.devices == nullptr) return {devices_out, x::errors::NIL};
    std::lock_guard lock(this->mu);
    std::unordered_map<std::string, Check> next;
    for (auto [key, dev]: *scan_ctx.devices) {
        this->check_device_health(dev, next);
        devices_out.push_back(dev);
    }
    this->checks = std::move(next);
    return {devices_out, x::errors::NIL};
}

bool Scanner::exec(
    synnax::task::Command &cmd,
    const synnax::task::Task &,
    const std::shared_ptr<task::Context> &
) {
    if (cmd.type == TEST_CONNECTION_CMD_TYPE) {
        this->test_connection(cmd);
        return true;
    }
    return false;
}

void Scanner::release_stale(const device::ConnectionConfig &cfg) {
    if (cfg.transport != device::Transport::RTU) return;
    std::erase_if(this->checks, [&](const auto &entry) {
        const auto &held = entry.second.dev->config();
        return held.endpoint() == cfg.endpoint() &&
               held.serial.line() != cfg.serial.line();
    });
}

void Scanner::check_device_health(
    synnax::device::Device &dev,
    std::unordered_map<std::string, Check> &next
) {
    const auto rack_key = this->task.rack;
    const auto parser = x::json::Parser(dev.properties);
    const auto props = device::Properties(parser);
    if (parser.error()) {
        dev.status = synnax::device::Status{
            .key = synnax::device::status_key(dev),
            .name = dev.name,
            .variant = synnax::status::VARIANT_WARNING,
            .message = "Invalid device properties",
            .description = common::config_err_description(parser.error()),
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
        return;
    }

    this->release_stale(props.connection);
    auto [d, err] = this->devices->acquire(props.connection, props.requests);
    synnax::device::StatusDetails details{.rack = rack_key, .device = dev.key};
    if (!err) {
        const auto prev = this->checks.find(dev.key);
        const bool replied = prev != this->checks.end() &&
                             d->last_reply() >= prev->second.time;
        if (!replied) err = d->probe();
        next[dev.key] = Check{.dev = d, .time = device::Connection::Clock::now()};
        details.data = x::json::json::object_t{{"stats", d->stats().to_json()}};
    }
    if (err)
        dev.status = synnax::device::Status{
            .key = synnax::device::status_key(dev),
            .name = dev.name,
            .variant = synnax::status::VARIANT_WARNING,
            .message = "Failed to reach device",
            .description = err.data,
            .time = x::telem::TimeStamp::now(),
            .details = details,
        };
    else
        dev.status = synnax::device::Status{
            .key = synnax::device::status_key(dev),
            .name = dev.name,
            .variant = synnax::status::VARIANT_SUCCESS,
            .message = "Device connected",
            .time = x::telem::TimeStamp::now(),
            .details = details,
        };
}

void Scanner::test_connection(const synnax::task::Command &cmd) {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::task::Status status{
        .key = synnax::task::status_key(this->task),
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .running = true,
            .cmd = cmd.key,
            .config_hash = task.config_hash,
        }
    };
    if (!parser.ok()) {
        status.message = "Failed to parse test command";
        status.description = common::config_err_description(parser.error());
        status.details.data = parser.error_json();
        ctx->set_status(status);
        return;
    }
    std::lock_guard lock(this->mu);
    this->release_stale(args.connection);
    auto [dev, err] = this->devices->acquire(args.connection);
    if (!err) err = dev->probe();
    if (err) {
        status.message = "Failed to connect to device";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = "Connected to device";
    ctx->set_status(status);
}
}
