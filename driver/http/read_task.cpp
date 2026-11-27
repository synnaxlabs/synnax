// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstdint>
#include <iterator>
#include <map>
#include <set>
#include <span>
#include <utility>
#include <variant>

#include "client/cpp/http/json.gen.h"
#include "x/cpp/strings/strings.h"

#include "driver/errors/errors.h"
#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/read_task.h"
#include "driver/icd/retrieve.h"

namespace driver::http {
std::pair<ReadTaskConfig, x::errors::Error>
ReadTaskConfig::parse(const synnax::task::Task &task) {
    auto parser = x::json::Parser(task.config);
    ReadTaskConfig cfg;
    static_cast<::synnax::http::ReadConfig &>(cfg) = ::synnax::http::ReadConfig::parse(
        parser
    );
    if (cfg.device.empty()) parser.field_err("device", "this field is required");
    std::erase_if(cfg.endpoints, [](const auto &ep) { return ep.disabled; });
    for (const auto &ep: cfg.endpoints)
        validate_request(ep, parser);
    if (cfg.endpoints.empty())
        parser.field_err("endpoints", "at least one enabled endpoint is required");
    if (!parser.ok()) return {std::move(cfg), parser.error()};
    return {std::move(cfg), x::errors::NIL};
}

std::pair<Readers, x::errors::Error> build_readers(
    const synnax::Synnax &client,
    const std::vector<::synnax::http::ReadEndpoint> &endpoints
) {
    std::set<synnax::icd::Key> icd_keys;
    std::set<synnax::channel::Key> channel_keys;
    for (const auto &ep: endpoints) {
        if (ep.icd.has_value()) icd_keys.insert(*ep.icd);
        for (const auto &rm: ep.messages)
            for (const auto &b: rm.bindings)
                if (b.channel != 0) channel_keys.insert(b.channel);
    }
    auto [resources, retrieve_err] = driver::icd::retrieve(
        client,
        icd_keys,
        channel_keys
    );
    if (retrieve_err) return {Readers{}, retrieve_err};
    Readers readers{.channels = std::move(resources.channels)};
    readers.endpoints.reserve(endpoints.size());
    // Every endpoint writes into one frame, so no two may write one channel.
    std::map<synnax::channel::Key, std::string> writers;
    for (const auto &ep: endpoints) {
        const auto name = "endpoint " + ep.method + " " + ep.path;
        if (!ep.icd.has_value() || ep.messages.empty())
            return {
                Readers{},
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    name +
                        " reads no messages. Open the task and pick a message for it."
                ),
            };
        const auto ic = std::ranges::find(
            resources.icds,
            *ep.icd,
            &synnax::icd::ICD::key
        );
        if (ic == resources.icds.end())
            return {
                Readers{},
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    name + ": ICD " + ep.icd->to_string() + " does not exist"
                ),
            };
        auto [reader, err] = driver::icd::Reader::create(
            ep.messages,
            *ic,
            readers.channels
        );
        if (err) return {Readers{}, err.wrap(name)};
        for (const auto key: reader.channels()) {
            const auto [other, claimed] = writers.try_emplace(key, name);
            if (!claimed)
                return {
                    Readers{},
                    x::errors::Error(
                        driver::errors::CONFIGURATION_ERROR,
                        other->second + " and " + name + " write to the same channel " +
                            std::to_string(key)
                    ),
                };
        }
        for (const auto &rm: ep.messages) {
            const auto msg = std::ranges::find(
                ic->messages,
                rm.message,
                &synnax::icd::Message::key
            );
            const auto *payload = std::get_if<synnax::icd::JSONPayload>(&msg->payload);
            if (payload == nullptr) continue;
            for (const auto &b: rm.bindings)
                for (const auto &f: payload->fields)
                    if (const auto &base = synnax::icd::base(f); base.key == b.field)
                        readers.pointers.emplace(b.channel, base.pointer);
        }
        readers.endpoints.push_back(std::move(reader));
    }
    return {std::move(readers), x::errors::NIL};
}

ReadTaskSource::ReadTaskSource(
    ReadTaskConfig cfg,
    std::shared_ptr<Processor> processor,
    std::vector<Request> requests,
    Readers readers
):
    cfg(std::move(cfg)),
    processor(std::move(processor)),
    requests(std::move(requests)),
    readers(std::move(readers)),
    sample_clock(this->cfg.rate) {
    for (const auto &reader: this->readers.endpoints)
        std::ranges::copy(reader.channels(), std::back_inserter(this->keys));
}

synnax::framer::WriterConfig ReadTaskSource::writer_config() const {
    return {
        .channels = this->keys,
        .mode = common::data_saving_writer_mode(this->cfg.data_saving_disabled),
    };
}

std::vector<synnax::channel::Channel> ReadTaskSource::channels() const {
    std::vector<synnax::channel::Channel> chs;
    chs.reserve(this->readers.channels.size());
    for (const auto &[_, ch]: this->readers.channels)
        chs.push_back(ch);
    return chs;
}

common::ReadResult
ReadTaskSource::read(x::breaker::Breaker &breaker, x::telem::Frame &fr) {
    common::ReadResult res;
    this->sample_clock.wait(breaker);

    const auto tick_start = x::telem::TimeStamp::now();
    auto results = this->processor->execute(this->requests);
    const auto tick = x::telem::TimeStamp::now() - tick_start;

    fr.reserve(this->keys.size());

    std::vector<std::string> warnings;
    std::size_t skipped = 0;
    std::size_t failed = 0;
    std::size_t rejected = 0;
    std::size_t unreachable = 0;

    for (std::size_t i = 0; i < this->requests.size(); i++) {
        const auto &[resp, req_err] = results[i];
        const auto &req = this->requests[i];

        if (req_err.matches(errors::UNREACHABLE_ERROR)) unreachable++;
        if (req_err.matches(errors::SKIPPED_ERROR)) {
            skipped++;
            continue;
        }

        if (req_err) {
            failed++;
            warnings.push_back(
                std::string(to_string(req.method)) + " " + req.url +
                " failed: " + req_err.data
            );
            continue;
        }

        if (auto status_err = errors::from_status(resp.status_code); status_err) {
            rejected++;
            auto msg = std::string(to_string(req.method)) + " " + req.url +
                       " returned " + std::to_string(resp.status_code);
            if (!resp.body.empty()) msg += ": " + resp.body_excerpt();
            warnings.push_back(msg);
            continue;
        }

        const auto &ep = this->cfg.endpoints[i];
        auto &reader = this->readers.endpoints[i];
        const auto [msg, read_err] = reader.read(
            std::span(
                reinterpret_cast<const std::uint8_t *>(resp.body.data()),
                resp.body.size()
            ),
            x::telem::TimeStamp::midpoint(resp.time_range.start, resp.time_range.end),
            fr,
            1,
            this->cfg.nan_fill_enabled
        );
        if (read_err) {
            warnings.push_back(
                "Failed to read the response from " + ep.path + ": " + read_err.data
            );
            continue;
        }
        if (!msg.has_value()) {
            warnings.push_back("The response from " + ep.path + " matches no message");
            continue;
        }
        for (const auto &f: reader.failures()) {
            const auto &pointer = this->readers.pointers.at(f.channel);
            if (f.missing)
                warnings.push_back(
                    "Field " + pointer + " not found in response from " + ep.path
                );
            else
                warnings.push_back(
                    "Failed to convert " + pointer + " for channel " +
                    this->readers.channels.at(f.channel).name + ": " + f.error.data
                );
        }
    }
    if (skipped > 0)
        warnings.push_back(
            "The device was unreachable, so the Driver did not send " +
            std::to_string(skipped) + " requests"
        );
    // An error, not a warning, so the pipeline breaker backs off from a device that
    // rejects every request. Always temporary: a 401 during a token refresh or a 404
    // while a device reboots must not stop the task.
    if (const auto failures = skipped + failed + rejected;
        failures == this->requests.size()) {
        res.error = x::errors::Error(
            unreachable == failures ? errors::UNREACHABLE_ERROR
                                    : errors::TEMPORARY_ERROR,
            x::strings::join(warnings, "; ")
        );
        return res;
    }
    std::string overrun;
    if (failed == 0 && skipped == 0 && !this->requests.empty() &&
        this->requests.size() > this->requests.front().max_concurrent_requests &&
        tick > this->cfg.rate.period())
        overrun = "Polling " + std::to_string(this->requests.size()) + " endpoints " +
                  std::to_string(this->requests.front().max_concurrent_requests) +
                  " at a time takes longer than the " +
                  this->cfg.rate.period().to_string() +
                  " read period. Lower the rate or raise the max concurrent "
                  "requests in the device properties.";

    if (!warnings.empty()) {
        if (!overrun.empty()) warnings.push_back(overrun);
        res.warning = {
            .message = "Failed to read some data",
            .description = x::strings::join(warnings, "; "),
        };
    } else if (!overrun.empty())
        res.warning = {.message = "Falling behind poll rate", .description = overrun};
    return res;
}

std::vector<Request> build_requests(
    const device::ConnectionConfig &conn,
    const std::vector<::synnax::http::ReadEndpoint> &endpoints
) {
    std::vector<Request> requests;
    requests.reserve(endpoints.size());
    for (const auto &ep: endpoints) {
        auto req_cfg = request_config(ep);
        const std::string body = has_request_body(req_cfg.method) ? ep.body : "";
        if (!body.empty()) req_cfg.request_content_type = "application/json";
        auto req = device::build_request(conn, req_cfg);
        req.body = body;
        requests.push_back(std::move(req));
    }
    return requests;
}

std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<Processor> &processor
) {
    auto [cfg, parse_err] = ReadTaskConfig::parse(task);
    if (parse_err) return {common::ConfigureResult{}, parse_err};

    auto [conn, conn_err] = device::retrieve_connection(
        ctx->client->devices,
        cfg.device
    );
    if (conn_err) return {common::ConfigureResult{}, conn_err};

    auto [readers, readers_err] = build_readers(*ctx->client, cfg.endpoints);
    if (readers_err) return {common::ConfigureResult{}, readers_err};

    auto requests = build_requests(conn, cfg.endpoints);

    const bool auto_start = cfg.auto_start;
    auto source = std::make_unique<ReadTaskSource>(
        std::move(cfg),
        processor,
        std::move(requests),
        std::move(readers)
    );

    // About one day of backoff before the task stops.
    auto breaker_cfg = x::breaker::Config{
        .name = task.name,
        .max_retries = 761,
        .max_interval = 2 * x::telem::MINUTE,
    };

    auto read_task = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker_cfg,
        std::move(source)
    );

    return {
        common::ConfigureResult{
            .task = std::move(read_task),
            .auto_start = auto_start,
        },
        x::errors::NIL,
    };
}
}
