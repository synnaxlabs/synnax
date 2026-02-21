// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"

#include "x/cpp/loop/loop.h"

#include "driver/http/scan_task.h"

namespace driver::http {
namespace {
const std::string LOG_PREFIX = "[http.scan] ";

std::pair<std::string, std::string>
check_health(device::Client &client, const std::optional<ResponseConfig> &response) {
    auto [results, batch_err] = client.execute_requests({""});
    if (batch_err)
        return {
            x::status::VARIANT_WARNING,
            "Failed to reach device: " + batch_err.message(),
        };

    if (results.empty()) return {x::status::VARIANT_WARNING, "Failed to reach device"};

    auto &[resp, req_err] = results[0];
    if (req_err)
        return {
            x::status::VARIANT_WARNING,
            "Failed to reach device: " + req_err.message(),
        };

    if (resp.status_code < 200 || resp.status_code >= 300)
        return {
            x::status::VARIANT_WARNING,
            "Device returned HTTP " + std::to_string(resp.status_code),
        };

    if (response.has_value()) {
        try {
            auto body = x::json::json::parse(resp.body);
            auto ptr = x::json::json::json_pointer(response->field);
            if (!body.contains(ptr))
                return {
                    x::status::VARIANT_WARNING,
                    "Unexpected health response: field '" + response->field +
                        "' not found",
                };
            const auto &actual = body.at(ptr);
            if (actual != response->expected_value)
                return {
                    x::status::VARIANT_WARNING,
                    "Unexpected health response: expected " +
                        response->expected_value.dump() + ", got " + actual.dump(),
                };
        } catch (const x::json::json::parse_error &) {
            return {
                x::status::VARIANT_WARNING,
                "Unexpected health response: invalid JSON body",
            };
        }
    }

    return {x::status::VARIANT_SUCCESS, "Device connected"};
}
}

std::pair<ScanTaskConfig, x::errors::Error>
ScanTaskConfig::parse(const synnax::task::Task &task) {
    auto parser = x::json::Parser(task.config);
    const auto device = parser.field<std::string>("device");
    const auto auto_start = parser.field<bool>("auto_start", false);
    const auto rate_hz = parser.field<double>("rate");
    const auto path = parser.field<std::string>("path");

    std::optional<ResponseConfig> response;
    auto response_parser = parser.optional_child("response");
    if (response_parser.ok()) response.emplace(response_parser);

    if (!parser.ok()) return {{}, parser.error()};

    return {
        ScanTaskConfig{
            .device = device,
            .auto_start = auto_start,
            .rate = x::telem::Rate(rate_hz),
            .path = path,
            .response = std::move(response),
        },
        x::errors::NIL,
    };
}

ScanTask::ScanTask(
    std::shared_ptr<task::Context> ctx,
    synnax::task::Task task,
    ScanTaskConfig cfg,
    device::ConnectionConfig conn
):
    pipeline::Base(
        x::breaker::Config{
            .name = task.name,
            .max_retries = x::breaker::RETRY_INFINITELY,
        },
        task.name
    ),
    ctx_(std::move(ctx)),
    task_(std::move(task)),
    cfg_(std::move(cfg)),
    conn_(std::move(conn)),
    status_handler_(ctx_, task_) {
    this->key = task_.key;
}

void ScanTask::exec(task::Command &cmd) {
    if (cmd.type == common::START_CMD_TYPE) {
        pipeline::Base::start();
        this->status_handler_.send_start(cmd.key);
    } else if (cmd.type == common::STOP_CMD_TYPE) {
        pipeline::Base::stop();
        this->status_handler_.send_stop(cmd.key);
    }
}

void ScanTask::stop(bool will_reconfigure) {
    pipeline::Base::stop();
}

void ScanTask::run() {
    device::RequestConfig req_cfg{
        .method = Method::GET,
        .path = cfg_.path,
    };
    auto [client, err] = device::Client::create(conn_, {req_cfg});
    if (err) {
        LOG(ERROR) << LOG_PREFIX << "failed to create client: " << err;
        this->status_handler_.send_error(err);
        return;
    }

    auto timer = x::loop::Timer(cfg_.rate);
    while (this->breaker.running()) {
        auto [variant, message] = check_health(client, cfg_.response);
        this->set_device_status(variant, message);
        if (variant == x::status::VARIANT_WARNING) {
            this->status_handler_.send_warning(message);
        } else {
            this->status_handler_.status.variant = x::status::VARIANT_SUCCESS;
            this->status_handler_.status.message = message;
            this->status_handler_.status.key = task_.status_key();
            this->ctx_->set_status(this->status_handler_.status);
        }
        timer.wait(this->breaker);
    }
}

void ScanTask::set_device_status(
    const std::string &variant,
    const std::string &message
) {
    if (ctx_->client == nullptr) return;
    synnax::device::Status dev_status;
    dev_status.key = synnax::device::ontology_id(cfg_.device).string();
    dev_status.variant = variant;
    dev_status.message = message;
    dev_status.time = x::telem::TimeStamp::now();
    dev_status.details.device = cfg_.device;
    if (const auto err = ctx_->client->statuses.set<synnax::device::StatusDetails>(
            dev_status
        );
        err)
        LOG(ERROR) << LOG_PREFIX << "failed to set device status: " << err;
}

std::pair<common::ConfigureResult, x::errors::Error> configure_scan(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    auto [cfg, err] = ScanTaskConfig::parse(task);
    if (err) return {{}, err};

    auto [conn, conn_err] = device::retrieve_connection(
        ctx->client->devices,
        cfg.device
    );
    if (conn_err) return {{}, conn_err};

    const bool auto_start = cfg.auto_start;
    auto scan_task = std::make_unique<ScanTask>(
        ctx,
        task,
        std::move(cfg),
        std::move(conn)
    );
    return {
        common::ConfigureResult{
            .task = std::move(scan_task),
            .auto_start = auto_start,
        },
        x::errors::NIL,
    };
}
}
