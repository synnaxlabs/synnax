// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>

#include "client/cpp/http/json.gen.h"
#include "x/cpp/strings/strings.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/read_task.h"

namespace driver::http {

std::pair<ReadTaskConfig, x::errors::Error> ReadTaskConfig::parse(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    auto parser = x::json::Parser(task.config);
    ReadTaskConfig cfg;
    static_cast<::synnax::http::ReadConfig &>(cfg) = ::synnax::http::ReadConfig::parse(
        parser
    );
    if (cfg.device.empty()) parser.field_err("device", "this field is required");

    std::set<synnax::channel::Key> field_keys;
    std::set<synnax::channel::Key> enabled_field_keys;

    std::vector<::synnax::http::ReadEndpoint> active;
    active.reserve(cfg.endpoints.size());
    for (auto &ep: cfg.endpoints) {
        validate_request(ep, parser);
        size_t enabled_field_count = 0;
        for (const auto &field: ep.fields) {
            validate_pointer(field.pointer, parser, "endpoints.fields.pointer");
            if (field.channel == 0)
                parser.field_err("endpoints.fields.channel", "this field is required");
            else if (!field_keys.insert(field.channel).second)
                parser.field_err(
                    "endpoints.fields.channel",
                    "channel " + std::to_string(field.channel) +
                        " is used multiple times"
                );

            if (field.time_format.has_value())
                if (auto [fmt, fmt_err] = x::json::parse_time_format(
                        *field.time_format
                    );
                    fmt_err)
                    parser.field_err("endpoints.fields.time_format", fmt_err);

            std::set<std::string> labels;
            for (const auto &entry: field.enum_values)
                if (entry.label.empty())
                    parser.field_err(
                        "endpoints.fields.enum_values.label",
                        "this field is required"
                    );
                else if (!labels.insert(entry.label).second)
                    parser.field_err(
                        "endpoints.fields.enum_values.label",
                        "duplicate enum label '" + entry.label + "'"
                    );

            if (!field.disabled) {
                enabled_field_count++;
                enabled_field_keys.insert(field.channel);
            }
        }

        if (ep.fields.empty())
            parser.field_err("endpoints.fields", "at least one field is required");
        else if (enabled_field_count > 0)
            active.push_back(std::move(ep));
    }
    cfg.endpoints = std::move(active);

    if (enabled_field_keys.empty()) {
        parser.field_err(
            "endpoints",
            "at least one endpoint with enabled fields is required"
        );
    }

    if (!parser.ok()) return {std::move(cfg), parser.error()};

    const std::vector<synnax::channel::Key> all_keys(
        enabled_field_keys.begin(),
        enabled_field_keys.end()
    );
    auto [sy_channels, ch_err] = ctx->client->channels.retrieve(all_keys);
    if (ch_err) return {{}, ch_err};

    for (const auto &ch: sy_channels)
        cfg.channels[ch.key] = ch;

    for (int ei = 0; ei < static_cast<int>(cfg.endpoints.size()); ei++) {
        for (const auto &field: cfg.endpoints[ei].fields) {
            if (field.disabled) continue;
            auto it = cfg.channels.find(field.channel);
            if (it == cfg.channels.end()) continue;
            const auto &ch = it->second;

            const auto &dt = ch.data_type;
            if (dt == x::telem::UUID_T || dt == x::telem::JSON_T ||
                dt == x::telem::BYTES_T) {
                parser.field_err(
                    "endpoints",
                    "channel " + ch.name + " has unsupported data type " + dt.name()
                );
                continue;
            }

            if (dt == x::telem::TIMESTAMP_T && !field.time_format.has_value()) {
                parser.field_err(
                    "endpoints",
                    "channel " + ch.name +
                        " is a timestamp channel but has no time_format"
                );
                continue;
            }

            if (ch.index == 0) continue;
            const auto idx_key = ch.index;
            if (enabled_field_keys.count(idx_key)) continue;
            auto [existing, inserted] = cfg.software_timed_indexes.try_emplace(
                idx_key,
                ei
            );
            if (!inserted && existing->second != ei) {
                parser.field_err(
                    "endpoints",
                    "index channel " + std::to_string(idx_key) +
                        " is referenced by fields on different endpoints"
                );
            }
        }
    }

    if (!parser.ok()) return {std::move(cfg), parser.error()};

    return {std::move(cfg), x::errors::NIL};
}

ReadTaskSource::ReadTaskSource(
    ReadTaskConfig cfg,
    std::shared_ptr<Processor> processor,
    std::vector<Request> requests
):
    cfg(std::move(cfg)),
    processor(std::move(processor)),
    requests(std::move(requests)),
    sample_clock(this->cfg.rate) {
    parsed_bodies.resize(this->cfg.endpoints.size());
    // Build sampling groups: fields sharing an index channel must be written
    // atomically. Fields with no index (ch.index == 0) get their own group.
    std::map<std::pair<size_t, synnax::channel::Key>, size_t> group_map;
    for (size_t ei = 0; ei < this->cfg.endpoints.size(); ei++) {
        for (const auto &field: this->cfg.endpoints[ei].fields) {
            if (field.disabled) continue;
            auto it = this->cfg.channels.find(field.channel);
            if (it == this->cfg.channels.end()) continue;
            this->chs.push_back(it->second);

            GroupField gf;
            gf.channel = field.channel;
            gf.pointer = x::json::json::json_pointer(field.pointer);
            if (field.time_format.has_value())
                if (auto [fmt, fmt_err] = x::json::parse_time_format(
                        *field.time_format
                    );
                    !fmt_err)
                    gf.time_format = fmt;
            for (const auto &entry: field.enum_values)
                if (!entry.label.empty())
                    gf.enum_values.emplace(entry.label, entry.value);

            const auto idx_key = it->second.index;
            if (idx_key == 0) {
                this->groups.push_back({
                    .index_key = 0,
                    .software_timed_index = false,
                    .endpoint_index = ei,
                    .fields = {std::move(gf)},
                });
                continue;
            }
            auto key = std::make_pair(ei, idx_key);
            auto [git, inserted] = group_map.try_emplace(key, this->groups.size());
            if (inserted)
                this->groups.push_back({
                    .index_key = idx_key,
                    .software_timed_index = this->cfg.software_timed_indexes.count(
                                                idx_key
                                            ) > 0,
                    .endpoint_index = ei,
                    .fields = {std::move(gf)},
                });
            else
                this->groups[git->second].fields.push_back(std::move(gf));
        }
    }
}

synnax::framer::WriterConfig ReadTaskSource::writer_config() const {
    std::vector<synnax::channel::Key> keys;
    keys.reserve(this->cfg.channels.size() + this->cfg.software_timed_indexes.size());
    for (const auto &ep: this->cfg.endpoints)
        for (const auto &field: ep.fields) {
            if (field.disabled) continue;
            keys.push_back(field.channel);
        }
    for (const auto &[key, _]: this->cfg.software_timed_indexes)
        keys.push_back(key);
    return {
        .channels = keys,
        .mode = common::data_saving_writer_mode(this->cfg.data_saving_disabled),
    };
}

std::vector<synnax::channel::Channel> ReadTaskSource::channels() const {
    return this->chs;
}

common::ReadResult
ReadTaskSource::read(x::breaker::Breaker &breaker, x::telem::Frame &fr) {
    common::ReadResult res;
    this->sample_clock.wait(breaker);

    auto results = this->processor->execute(this->requests);

    fr.reserve(this->cfg.channels.size() + this->cfg.software_timed_indexes.size());

    std::vector<std::string> warnings;

    // Parse all response bodies up front so sampling groups can reference them.
    std::vector<bool> ep_parsed(this->cfg.endpoints.size(), false);
    for (size_t ei = 0; ei < this->cfg.endpoints.size(); ei++) {
        const auto &ep = this->cfg.endpoints[ei];
        auto &[resp, req_err] = results[ei];

        if (req_err) {
            const auto &req = requests[ei];
            warnings.push_back(
                std::string(to_string(req.method)) + " " + req.url +
                " failed: " + req_err.data
            );
            continue;
        }

        if (auto status_err = errors::from_status(resp.status_code); status_err) {
            const auto &req = requests[ei];
            auto msg = std::string(to_string(req.method)) + " " + req.url +
                       " returned " + std::to_string(resp.status_code);
            if (!resp.body.empty()) msg += ": " + resp.body;
            warnings.push_back(msg);
            continue;
        }

        try {
            this->parsed_bodies[ei] = x::json::json::parse(resp.body);
            ep_parsed[ei] = true;
        } catch (const x::json::json::parse_error &e) {
            warnings.push_back(
                "failed to parse response from " + ep.path + ": " + e.what()
            );
        }
    }

    // Process each sampling group atomically: either all fields in the group succeed
    // and are written to the frame, or the entire group is skipped.
    for (const auto &group: this->groups) {
        const auto ei = group.endpoint_index;
        if (!ep_parsed[ei]) continue;

        const auto &ep = this->cfg.endpoints[ei];
        const auto &body = this->parsed_bodies[ei];
        const auto &resp = results[ei].first;

        bool group_ok = true;
        std::vector<std::pair<synnax::channel::Key, x::telem::Series>> group_data;
        group_data.reserve(group.fields.size());

        for (const auto &field: group.fields) {
            if (!body.contains(field.pointer)) {
                warnings.push_back(
                    "field " + field.pointer.to_string() +
                    " not found in response from " + ep.path
                );
                group_ok = false;
                break;
            }

            const auto &ch = this->cfg.channels.at(field.channel);
            const auto &json_val = body.at(field.pointer);

            auto tf = x::json::TimeFormat::ISO8601;
            if (field.time_format.has_value()) tf = *field.time_format;

            const auto *enum_ptr = field.enum_values.empty() ? nullptr
                                                             : &field.enum_values;
            auto [sample_val, conv_err] = x::json::to_sample_value(
                json_val,
                ch.data_type,
                tf,
                enum_ptr
            );
            if (conv_err) {
                warnings.push_back(
                    "failed to convert " + field.pointer.to_string() + " for channel " +
                    ch.name + ": " + conv_err.message()
                );
                group_ok = false;
                break;
            }

            group_data.emplace_back(field.channel, x::telem::Series(sample_val));
        }

        if (!group_ok) continue;

        for (auto &[key, series]: group_data)
            fr.emplace(key, std::move(series));

        if (group.software_timed_index) {
            auto ts = x::telem::TimeStamp::midpoint(
                resp.time_range.start,
                resp.time_range.end
            );
            auto s = x::telem::Series(x::telem::TIMESTAMP_T, 1);
            s.write(ts);
            fr.emplace(group.index_key, std::move(s));
        }
    }

    if (!warnings.empty()) res.warning = x::strings::join(warnings, "; ");
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
    auto [cfg, parse_err] = ReadTaskConfig::parse(ctx, task);
    if (parse_err) return {common::ConfigureResult{}, parse_err};

    auto [conn, conn_err] = device::retrieve_connection(
        ctx->client->devices,
        cfg.device
    );
    if (conn_err) return {common::ConfigureResult{}, conn_err};

    auto requests = build_requests(conn, cfg.endpoints);

    const bool auto_start = cfg.auto_start;
    auto source = std::make_unique<ReadTaskSource>(
        std::move(cfg),
        processor,
        std::move(requests)
    );

    auto breaker_cfg = x::breaker::Config{.name = task.name};

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
