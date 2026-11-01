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
#include <string>
#include <utility>

#include "x/cpp/json/json.h"

#include "driver/modbus/scan_task.h"

namespace driver::modbus {
namespace {
/// @returns whether two serial settings open separate connections: by another path to
/// the port, or with other line settings.
bool separate_connections(const device::Serial &a, const device::Serial &b) {
    return device::port_name(a.port) != device::port_name(b.port) ||
           a.line() != b.line();
}
}

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
    for (auto [key, dev]: *scan_ctx.devices) {
        this->check_device_health(dev);
        devices_out.push_back(dev);
    }
    std::lock_guard lock(this->mu);
    std::erase_if(this->checks, [&](const auto &entry) {
        return !scan_ctx.devices->contains(entry.first);
    });
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

void Scanner::release_stale(
    std::unique_lock<std::mutex> &lock,
    const device::ConnectionConfig &cfg
) {
    if (cfg.transport != device::Transport::RTU) return;
    const auto endpoint = cfg.endpoint();
    this->probed.wait(lock, [&] { return !this->probing.contains(endpoint); });
    std::erase_if(this->checks, [&](const auto &entry) {
        const auto &held = entry.second.dev->config();
        return held.endpoint() == endpoint &&
               separate_connections(held.serial, cfg.serial);
    });
}

void Scanner::check_device_health(synnax::device::Device &dev) {
    const auto rack_key = this->task.rack;
    const auto parser = x::json::Parser(dev.properties);
    const auto props = device::Properties(parser);
    std::unique_lock lock(this->mu);
    if (parser.error()) {
        this->checks.erase(dev.key);
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

    // The device's old check would hold its port through the old path or line settings.
    if (const auto held = this->checks.find(dev.key); held != this->checks.end()) {
        const auto &old = held->second.dev->config().serial;
        if (separate_connections(old, props.connection.serial))
            this->checks.erase(held);
    }
    auto [d, err] = this->devices->acquire(props.connection, props.requests);
    synnax::device::StatusDetails details{.rack = rack_key, .device = dev.key};
    if (err)
        this->checks.erase(dev.key);
    else {
        const auto prev = this->checks.find(dev.key);
        const bool replied = prev != this->checks.end() &&
                             d->last_reply() >= prev->second.time;
        if (!replied) {
            const auto endpoint = props.connection.endpoint();
            this->probing.insert(endpoint);
            lock.unlock();
            err = d->probe();
            lock.lock();
            this->probing.erase(endpoint);
            this->probed.notify_all();
        }
        this->checks[dev.key] = Check{
            .dev = d,
            .time = device::Connection::Clock::now()
        };
        details.data = x::json::json::object_t{{"stats", d->stats().to_json()}};
    }
    // A connection test that releases the check must also release the connection.
    d.reset();
    lock.unlock();
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
    auto [dev, err] = [&] {
        std::unique_lock lock(this->mu);
        this->release_stale(lock, args.connection);
        return this->devices->acquire(args.connection, args.requests);
    }();
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
