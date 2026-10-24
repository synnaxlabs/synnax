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

#include "x/cpp/json/json.h"

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
    if (scan_ctx.devices == nullptr) return {{}, x::errors::NIL};

    // Phase 1: build all requests, setting status immediately for devices that
    // fail config parsing.
    std::vector<PreparedHealthCheck> prepared;
    for (const auto &[key, dev]: *scan_ctx.devices) {
        const auto idx = devices_out.size();
        devices_out.push_back(dev);
        auto hc = this->prepare_health_check(devices_out[idx], idx);
        if (hc.has_value()) prepared.push_back(std::move(*hc));
    }

    if (prepared.empty()) return {devices_out, x::errors::NIL};

    // Phase 2: execute all health check requests in parallel.
    std::vector<Request> requests;
    requests.reserve(prepared.size());
    for (const auto &p: prepared)
        requests.push_back(p.request);
    auto results = this->processor->execute(requests);

    // Phase 3: process all responses.
    for (std::size_t i = 0; i < prepared.size(); i++) {
        auto &p = prepared[i];
        auto &[resp, err] = results[i];
        this->process_health_response(
            devices_out[p.device_index],
            p.expected_response,
            resp,
            err
        );
    }

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

/// @brief describes a response the server rejected by its status code and the start of
/// its body.
static std::string rejection_description(const Response &resp) {
    const auto description = "The server responded with HTTP " +
                             std::to_string(resp.status_code);
    if (resp.body.empty()) return description + ".";
    return description + ": " + resp.body_excerpt();
}

/// @brief validates the response body against an expected response config.
/// @returns an empty string on success, or a description of the problem.
static std::string validate_health_response(
    const std::optional<ExpectedResponseConfig> &expected_response,
    const Response &resp
) {
    if (!expected_response.has_value() || expected_response->pointer.empty()) return "";
    const auto &er = *expected_response;
    x::json::json body;
    try {
        body = x::json::json::parse(resp.body);
    } catch (const x::json::json::parse_error &e) {
        return "The response body is not valid JSON: " + std::string(e.what()) + ".";
    }
    const auto ptr = x::json::json::json_pointer(er.pointer);
    if (!body.contains(ptr))
        return "The response body has no value at '" + er.pointer + "'.";
    if (body[ptr] == er.expected_value) return "";
    return "The value at '" + er.pointer + "' is " + body[ptr].dump() +
           ", but the health check expects " + er.expected_value.dump() + ".";
}

void Scanner::set_device_status(
    synnax::device::Device &dev,
    const std::string &variant,
    const std::string &message,
    const std::string &description
) const {
    dev.status = synnax::device::Status{
        .key = synnax::device::status_key(dev),
        .name = dev.name,
        .variant = variant,
        .message = message,
        .description = description,
        .time = x::telem::TimeStamp::now(),
        .details = {
            .rack = this->task.rack,
            .device = dev.key,
        },
    };
}

std::optional<Scanner::PreparedHealthCheck> Scanner::prepare_health_check(
    synnax::device::Device &dev,
    const std::size_t device_index
) const {
    auto props = x::json::json(dev.properties);
    const bool secure = props.value("secure", true);
    const std::string protocol = secure ? "https://" : "http://";
    props["base_url"] = protocol + dev.location;
    auto parser = x::json::Parser(props);
    const auto conn = device::ConnectionConfig(parser);
    auto hc = HealthCheckConfig(parser.child("health_check"));
    if (parser.error()) {
        this->set_device_status(
            dev,
            synnax::status::VARIANT_WARNING,
            "Invalid device properties",
            common::config_err_description(parser.error())
        );
        return std::nullopt;
    }
    auto request = device::build_request(conn, hc.request);
    if (!hc.body.empty()) request.body = std::move(hc.body);

    return PreparedHealthCheck{
        .device_index = device_index,
        .expected_response = std::move(hc.expected_response),
        .request = std::move(request),
    };
}

void Scanner::process_health_response(
    synnax::device::Device &dev,
    const std::optional<ExpectedResponseConfig> &expected_response,
    const Response &resp,
    const x::errors::Error &err
) const {
    if (err) {
        this->set_device_status(
            dev,
            synnax::status::VARIANT_WARNING,
            "Failed to reach server",
            err.data
        );
        return;
    }
    if (const auto status_err = errors::from_status(resp.status_code)) {
        this->set_device_status(
            dev,
            synnax::status::VARIANT_ERROR,
            "Health check failed",
            rejection_description(resp)
        );
        return;
    }
    if (const auto v_err = validate_health_response(expected_response, resp);
        !v_err.empty()) {
        this->set_device_status(
            dev,
            synnax::status::VARIANT_ERROR,
            "Health check failed",
            v_err
        );
        return;
    }
    this->set_device_status(dev, synnax::status::VARIANT_SUCCESS, "Device connected");
}

void Scanner::test_connection(const synnax::task::Command &cmd) const {
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
        ctx->set_status(status);
        return;
    }
    auto request = device::build_request(args.connection, args.health_check.request);
    if (!args.health_check.body.empty()) request.body = args.health_check.body;
    auto [resp, err] = this->processor->execute(request);
    if (err) {
        status.message = "Failed to connect to server";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    const auto status_err = errors::from_status(resp.status_code);
    if (status_err) {
        status.message = "Health check failed";
        status.description = rejection_description(resp);
        ctx->set_status(status);
        return;
    }

    const auto validation_err = validate_health_response(
        args.health_check.expected_response,
        resp
    );
    if (!validation_err.empty()) {
        status.message = "Health check failed";
        status.description = validation_err;
        ctx->set_status(status);
        return;
    }
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = "Connected to server";
    ctx->set_status(status);
}
}
