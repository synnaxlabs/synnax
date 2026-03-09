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
#include "x/cpp/status/status.h"

#include "driver/http/errors/errors.h"
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

/// @brief validates the response body against the health check's expected response
/// config.
/// @returns empty string on success, error message on failure.
static std::string
validate_health_response(const HealthCheckConfig &hc, const Response &resp) {
    if (!hc.expected_response.has_value() || hc.expected_response->pointer.empty())
        return "";
    const auto &er = *hc.expected_response;
    x::json::json body;
    try {
        body = x::json::json::parse(resp.body);
    } catch (const x::json::json::parse_error &e) {
        return "failed to parse response body as JSON: " + std::string(e.what());
    }
    const auto ptr = x::json::json::json_pointer(er.pointer);
    if (!body.contains(ptr))
        return "response body does not contain pointer '" + er.pointer + "'";
    if (body[ptr] == er.expected_value) return "";
    return "expected value at '" + er.pointer + "' to be " + er.expected_value.dump() +
           ", got " + body[ptr].dump();
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

    const auto hc = HealthCheckConfig(parser.child("health_check"));

    auto request = device::build_request(conn, hc.request);
    if (!hc.body.empty()) request.body = std::move(hc.body);
    auto [resp, err] = this->processor->execute(request);
    if (err) {
        dev.status = synnax::device::Status{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = x::status::VARIANT_WARNING,
            .message = "Failed to reach server",
            .description = err.message(),
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
        return;
    }
    const auto error = errors::from_status(resp.status_code);
    if (error) {
        dev.status = synnax::device::Status{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = x::status::VARIANT_ERROR,
            .message = "HTTP " + std::to_string(resp.status_code),
            .description = resp.body,
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
        return;
    }

    const auto validation_err = validate_health_response(hc, resp);
    if (!validation_err.empty()) {
        dev.status = synnax::device::Status{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = x::status::VARIANT_ERROR,
            .message = "Health check validation failed",
            .description = validation_err,
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
        return;
    }

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
        status.description = parser.error().message();
        return ctx->set_status(status);
    }
    auto request = device::build_request(args.connection, args.health_check.request);
    if (!args.health_check.body.empty())
        request.body = std::move(args.health_check.body);
    auto [resp, err] = this->processor->execute(request);
    if (err) {
        status.message = "Failed to execute HTTP request";
        status.description = err.message();
        return ctx->set_status(status);
    }
    const auto status_err = errors::from_status(resp.status_code);
    if (status_err) {
        status.message = "HTTP " + std::to_string(resp.status_code);
        status.description = resp.body;
        return ctx->set_status(status);
    }

    const auto validation_err = validate_health_response(args.health_check, resp);
    if (!validation_err.empty()) {
        status.message = "Invalid health check response";
        status.description = validation_err;
        return ctx->set_status(status);
    }
    status.variant = x::status::VARIANT_SUCCESS;
    status.message = "Connection successful";
    return ctx->set_status(status);
}
}
