// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <deque>
#include <map>
#include <set>
#include <span>
#include <string>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/http/json.gen.h"
#include "x/cpp/strings/strings.h"

#include "driver/errors/errors.h"
#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/write_task.h"
#include "driver/icd/retrieve.h"
#include "driver/icd/writer.h"

namespace driver::http {
namespace {
/// @returns the command channels of the enabled endpoints.
std::set<synnax::channel::Key>
command_keys(const std::vector<::synnax::http::WriteEndpoint> &endpoints) {
    std::set<synnax::channel::Key> keys;
    for (const auto &ep: endpoints) {
        if (ep.disabled || !ep.message.has_value()) continue;
        for (const auto &b: ep.message->bindings)
            if (const auto *cb = std::get_if<synnax::icd::ChannelWriteBinding>(&b))
                keys.insert(cb->channel);
    }
    return keys;
}

/// @returns the method and URL of req, to name it in a status.
std::string describe(const Request &req) {
    return std::string(to_string(req.method)) + " " + req.url;
}
}

std::pair<WriteTaskConfig, x::errors::Error>
WriteTaskConfig::parse(const synnax::task::Task &task) {
    auto parser = x::json::Parser(task.config);
    WriteTaskConfig cfg;
    static_cast<::synnax::http::WriteConfig &>(
        cfg
    ) = ::synnax::http::WriteConfig::parse(parser);
    if (cfg.device.empty()) parser.field_err("device", "this field is required");
    for (const auto &ep: cfg.endpoints)
        validate_request(ep, parser);
    if (std::ranges::all_of(cfg.endpoints, &::synnax::http::WriteEndpoint::disabled))
        parser.field_err("endpoints", "at least one enabled endpoint is required");
    if (!parser.ok()) return {std::move(cfg), parser.error()};
    return {std::move(cfg), x::errors::NIL};
}

WriteTaskSink::WriteTaskSink(
    std::vector<synnax::channel::Key> cmd_keys,
    std::shared_ptr<Processor> processor,
    std::vector<Request> requests,
    std::vector<driver::icd::Writer> writers
):
    Sink(std::move(cmd_keys)),
    processor(std::move(processor)),
    requests(std::move(requests)),
    writers(std::move(writers)) {}

x::errors::Error WriteTaskSink::write(x::telem::Frame &frame) {
    // Queue i holds the requests of endpoint i in sample order. Each round sends the
    // front of each queue, so an endpoint gets its commands in order. A failed request
    // drops the rest of its queue, so an endpoint never repeats or holds back another.
    std::vector<std::deque<Request>> queues(this->writers.size());
    std::vector<std::string> refused;
    std::vector<std::string> error_msgs;
    std::string first_error_type;
    for (std::size_t i = 0; i < this->writers.size(); i++) {
        const auto &base = this->requests[i];
        const auto result = this->writers[i].write(
            frame,
            [&](std::span<const std::uint8_t> payload) {
                auto &req = queues[i].emplace_back(base);
                req.body.assign(payload.begin(), payload.end());
                return x::errors::NIL;
            }
        );
        if (result.refused)
            refused.push_back(
                "Dropped a command for " + describe(base) + ": " + result.refused.data +
                "."
            );
        if (result.error) {
            if (first_error_type.empty()) first_error_type = result.error.type;
            error_msgs.push_back(describe(base) + ": " + result.error.data);
        }
    }

    bool sent = false;
    bool temporary = false;
    std::vector<Request> batch;
    std::vector<std::size_t> owners;
    while (true) {
        batch.clear();
        owners.clear();
        for (std::size_t i = 0; i < queues.size(); i++)
            if (!queues[i].empty()) {
                batch.push_back(queues[i].front());
                owners.push_back(i);
            }
        if (batch.empty()) break;
        auto results = this->processor->execute(batch);
        for (std::size_t j = 0; j < results.size(); j++) {
            auto &queue = queues[owners[j]];
            auto &[resp, req_err] = results[j];
            x::errors::Error err = x::errors::NIL;
            if (req_err)
                err = req_err;
            else if (
                auto status_err = errors::from_status(resp.status_code); status_err
            )
                err = status_err;
            if (!err) {
                sent = true;
                queue.pop_front();
                continue;
            }
            if (err.matches(errors::TEMPORARY_ERROR))
                temporary = true;
            else if (first_error_type.empty())
                first_error_type = err.type;
            auto msg = describe(batch[j]);
            if (req_err)
                msg += ": " + req_err.data;
            else {
                msg += " returned " + std::to_string(resp.status_code);
                if (!resp.body.empty()) msg += ": " + resp.body;
            }
            error_msgs.push_back(std::move(msg));
            queue.clear();
        }
    }
    if (sent) this->set_state(frame);
    if (first_error_type.empty() && temporary)
        first_error_type = errors::TEMPORARY_ERROR.type;

    if (!refused.empty())
        this->warn({
            .message = "Skipped invalid command",
            .description = x::strings::join(refused, " "),
        });

    if (error_msgs.empty()) return x::errors::NIL;
    return {first_error_type, x::strings::join(error_msgs, "; ")};
}

std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<Processor> &processor
) {
    auto [cfg, parse_err] = WriteTaskConfig::parse(task);
    if (parse_err) return {common::ConfigureResult{}, parse_err};

    auto [writers, writers_err] = build_writers(*ctx->client, cfg.endpoints);
    if (writers_err) return {common::ConfigureResult{}, writers_err};

    auto [conn, conn_err] = device::retrieve_connection(
        ctx->client->devices,
        cfg.device
    );
    if (conn_err) return {common::ConfigureResult{}, conn_err};

    std::vector<Request> requests;
    for (const auto &ep: cfg.endpoints) {
        if (ep.disabled) continue;
        auto req_cfg = request_config(ep);
        req_cfg.request_content_type = "application/json";
        requests.push_back(device::build_request(conn, req_cfg));
    }

    const auto keys = command_keys(cfg.endpoints);
    auto sink = std::make_unique<WriteTaskSink>(
        std::vector(keys.begin(), keys.end()),
        processor,
        std::move(requests),
        std::move(writers)
    );

    auto write_task = std::make_unique<common::WriteTask>(
        task,
        ctx,
        x::breaker::Config{.name = task.name},
        std::move(sink)
    );

    return {
        common::ConfigureResult{
            .task = std::move(write_task),
            .auto_start = cfg.auto_start,
        },
        x::errors::NIL,
    };
}

std::pair<std::vector<driver::icd::Writer>, x::errors::Error> build_writers(
    const synnax::Synnax &client,
    const std::vector<::synnax::http::WriteEndpoint> &endpoints
) {
    std::set<synnax::icd::Key> icd_keys;
    for (const auto &ep: endpoints)
        if (!ep.disabled && ep.icd.has_value()) icd_keys.insert(*ep.icd);
    const auto [resources, retrieve_err] = driver::icd::retrieve(
        client,
        icd_keys,
        command_keys(endpoints)
    );
    if (retrieve_err) return {std::vector<driver::icd::Writer>(), retrieve_err};
    std::vector<driver::icd::Writer> writers;
    writers.reserve(endpoints.size());
    for (const auto &ep: endpoints) {
        if (ep.disabled) continue;
        const auto name = "endpoint " + ep.method + " " + ep.path;
        if (!ep.icd.has_value() || !ep.message.has_value())
            return {
                std::vector<driver::icd::Writer>(),
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    name + " sends no message. Open the task and pick a message for it."
                ),
            };
        const auto ic = std::ranges::find(
            resources.icds,
            *ep.icd,
            &synnax::icd::ICD::key
        );
        if (ic == resources.icds.end())
            return {
                std::vector<driver::icd::Writer>(),
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    name + ": ICD " + ep.icd->to_string() + " does not exist"
                ),
            };
        auto [writer, err] = driver::icd::Writer::create(
            *ep.message,
            *ic,
            resources.channels
        );
        if (err)
            return {
                std::vector<driver::icd::Writer>(),
                x::errors::Error(err, name + ": " + err.data)
            };
        writers.push_back(std::move(writer));
    }
    return {std::move(writers), x::errors::NIL};
}
}
