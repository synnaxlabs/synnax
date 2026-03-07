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

#include "driver/http/scan_task.h"

namespace driver::http {
Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::task::Task task,
    std::shared_ptr<Processor> processor
):
    ctx(std::move(ctx)), task(std::move(task)), processor(std::move(processor)) {}

common::ScannerConfig Scanner::config() const {
    return common::ScannerConfig{
        .make = INTEGRATION_NAME,
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
    return {devices_out, x::errors::NIL};
}

bool Scanner::exec(
    task::Command &cmd,
    const synnax::task::Task &,
    const std::shared_ptr<task::Context> &
) {
    if (cmd.type == TEST_CONNECTION_CMD_TYPE) {
        this->test_connection(cmd);
        return true;
    }
    return false;
}

void Scanner::check_device_health(synnax::device::Device &dev) const {
    const auto rack_key = synnax::task::rack_key_from_task_key(this->task.key);
    auto props = x::json::json(dev.properties);
    const bool secure = props.value("secure", true);
    const std::string protocol = secure ? "https://" : "http://";
    props["base_url"] = protocol + dev.location;
    auto parser = x::json::Parser(props);
    const auto conn = device::ConnectionConfig(parser);
    if (parser.error()) {
        dev.status = synnax::device::Status{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = x::status::VARIANT_WARNING,
            .message = "Invalid device properties",
            .description = parser.error().message(),
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
        return;
    }

    const RequestConfig req_cfg{.method = Method::GET, .path = "/health"};
    auto request = device::build_request(conn, req_cfg);
    auto [resp, err] = this->processor->execute(request);
    if (err)
        dev.status = synnax::device::Status{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = x::status::VARIANT_WARNING,
            .message = "Failed to reach device",
            .description = err.message(),
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
    else
        dev.status = synnax::device::Status{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = x::status::VARIANT_SUCCESS,
            .message = "Device connected",
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
}

void Scanner::test_connection(const task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::task::Status status{
        .key = this->task.status_key(),
        .name = this->task.name,
        .variant = x::status::VARIANT_ERROR,
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
    const RequestConfig req_cfg{.method = Method::GET, .path = "/health"};
    auto request = device::build_request(args.connection, req_cfg);
    auto [resp, err] = this->processor->execute(request);
    if (err) {
        status.message = err.data;
        return ctx->set_status(status);
    }
    status.variant = x::status::VARIANT_SUCCESS;
    status.message = "Connection successful";
    return ctx->set_status(status);
}
}
