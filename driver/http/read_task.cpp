// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/http/errors/errors.h"
#include "driver/http/read_task.h"

namespace driver::http {

std::pair<ReadTaskConfig, x::errors::Error>
ReadTaskConfig::parse(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    auto parser = x::json::Parser(task.config);
    ReadTaskConfig cfg;
    cfg.device = parser.field<std::string>("device");
    cfg.data_saving = parser.field<bool>("data_saving", true);
    cfg.auto_start = parser.field<bool>("auto_start", false);
    cfg.rate = x::telem::Rate(parser.field<double>("rate"));
    cfg.strict = parser.field<bool>("strict", false);

    // Collect all channel keys for batch retrieval.
    std::vector<synnax::channel::Key> all_keys;

    parser.iter("endpoints", [&](x::json::Parser &ep) {
        ReadEndpoint endpoint;
        endpoint.request.method = parse_method(ep, "method");
        if (endpoint.request.method != Method::GET &&
            endpoint.request.method != Method::POST)
            ep.field_err(
                "method",
                "read tasks only support GET and POST methods"
            );
        endpoint.request.path = ep.field<std::string>("path");
        endpoint.request.query_params = ep.field<std::map<std::string, std::string>>(
            "query_params",
            std::map<std::string, std::string>{}
        );
        endpoint.body = ep.field<std::string>("body", "");

        ep.iter("fields", [&](x::json::Parser &fp) {
            ReadField field;
            field.pointer = x::json::json::json_pointer(
                fp.field<std::string>("pointer")
            );
            field.channel_key = fp.field<synnax::channel::Key>("channel");

            auto tf_parser = fp.optional_child("timestampFormat");
            if (tf_parser.ok()) {
                const auto str = fp.field<std::string>("timestampFormat");
                auto [fmt, fmt_err] = x::json::parse_time_format(str);
                if (fmt_err)
                    fp.field_err("timestampFormat", fmt_err.message());
                else
                    field.time_format = fmt;
            }

            auto ti_parser = fp.optional_child("timePointer");
            if (ti_parser.ok()) field.time_info.emplace(ti_parser);

            // Check for duplicate channel keys.
            if (cfg.all_channel_keys.count(field.channel_key))
                fp.field_err(
                    "channel",
                    "channel " + std::to_string(field.channel_key) +
                        " is used multiple times"
                );
            else
                cfg.all_channel_keys.insert(field.channel_key);

            all_keys.push_back(field.channel_key);
            endpoint.fields.push_back(std::move(field));
        });

        cfg.endpoints.push_back(std::move(endpoint));
    });

    if (cfg.endpoints.empty()) {
        parser.field_err("endpoints", "at least one endpoint is required");
        return {std::move(cfg), parser.error()};
    }

    if (!parser.ok()) return {std::move(cfg), parser.error()};

    // Retrieve the device connection config.
    auto [conn, conn_err] = device::retrieve_connection(
        ctx->client->devices,
        cfg.device
    );
    if (conn_err) return {{}, conn_err};

    // Fetch all referenced channels from Synnax.
    auto [sy_channels, ch_err] = ctx->client->channels.retrieve(all_keys);
    if (ch_err) return {{}, ch_err};

    // Build a key -> channel map for lookups.
    std::map<synnax::channel::Key, synnax::channel::Channel> ch_map;
    for (const auto &ch : sy_channels) ch_map[ch.key] = ch;

    auto validate_err = cfg.validate_fields(ch_map);
    if (validate_err) return {std::move(cfg), validate_err};
    return {std::move(cfg), x::errors::NIL};
}

x::errors::Error ReadTaskConfig::validate_fields(
    const std::map<synnax::channel::Key, synnax::channel::Channel> &ch_map
) {
    struct IndexEntry {
        int endpoint_index;
        std::optional<TimeInfo> time_info;
    };
    std::map<synnax::channel::Key, IndexEntry> index_entries;
    x::json::Parser validator(x::json::json::object());

    for (int ei = 0; ei < static_cast<int>(endpoints.size()); ei++) {
        auto &ep = endpoints[ei];
        for (auto &field : ep.fields) {
            auto it = ch_map.find(field.channel_key);
            if (it == ch_map.end()) continue;
            field.ch = it->second;

            const auto &dt = field.ch.data_type;
            if (dt == x::telem::UUID_T || dt == x::telem::JSON_T) {
                validator.field_err(
                    "endpoints",
                    "channel " + field.ch.name +
                        " has unsupported data type " + dt.name()
                );
                continue;
            }

            if (field.time_format.has_value() && dt != x::telem::TIMESTAMP_T) {
                validator.field_err(
                    "endpoints",
                    "channel " + field.ch.name +
                        " has timestampFormat but is not a timestamp channel"
                );
                continue;
            }

            if (dt == x::telem::TIMESTAMP_T && !field.time_format.has_value()) {
                validator.field_err(
                    "endpoints",
                    "channel " + field.ch.name +
                        " is a timestamp channel but has no timestampFormat"
                );
                continue;
            }

            if (field.ch.index == 0) continue;
            const auto idx_key = field.ch.index;

            std::optional<TimeInfo> ti;
            if (field.time_info.has_value()) ti = field.time_info;

            auto existing = index_entries.find(idx_key);
            if (existing != index_entries.end()) {
                if (existing->second.time_info.has_value() && ti.has_value()) {
                    if (existing->second.time_info->pointer != ti->pointer ||
                        existing->second.time_info->format != ti->format) {
                        validator.field_err(
                            "endpoints",
                            "conflicting timestamp sources for index channel " +
                                std::to_string(idx_key)
                        );
                    }
                }
                if (!existing->second.time_info.has_value() && ti.has_value())
                    existing->second.time_info = ti;
            } else {
                index_entries[idx_key] = IndexEntry{
                    .endpoint_index = ei,
                    .time_info = ti,
                };
            }
            index_keys.insert(idx_key);
            all_channel_keys.insert(idx_key);
        }
    }

    if (!validator.ok()) return validator.error();

    for (const auto &[key, entry] : index_entries) {
        index_sources.push_back(IndexSource{
            .index_key = key,
            .endpoint_index = entry.endpoint_index,
            .time_info = entry.time_info,
        });
    }

    return x::errors::NIL;
}

ReadTaskSource::ReadTaskSource(ReadTaskConfig cfg, device::Client client):
    cfg_(std::move(cfg)),
    client_(std::move(client)),
    read_opts_({.strict = cfg_.strict}) {
    // Build the flat list of channels for the tare transform.
    for (const auto &ep : cfg_.endpoints)
        for (const auto &field : ep.fields)
            channels_.push_back(field.ch);
}

synnax::framer::WriterConfig ReadTaskSource::writer_config() const {
    std::vector<synnax::channel::Key> keys;
    keys.reserve(cfg_.all_channel_keys.size());
    for (const auto &key : cfg_.all_channel_keys) keys.push_back(key);
    return {
        .channels = keys,
        .mode = common::data_saving_writer_mode(cfg_.data_saving),
    };
}

std::vector<synnax::channel::Channel> ReadTaskSource::channels() const {
    return channels_;
}

common::ReadResult
ReadTaskSource::read(x::breaker::Breaker &breaker, x::telem::Frame &fr) {
    common::ReadResult res;

    // Build bodies vector.
    std::vector<std::string> bodies;
    bodies.reserve(cfg_.endpoints.size());
    for (const auto &ep : cfg_.endpoints) bodies.push_back(ep.body);

    // Execute all endpoint requests in parallel.
    auto [results, batch_err] = client_.execute_requests(bodies);
    if (batch_err) {
        res.error = batch_err;
        return res;
    }

    // Parse responses per endpoint. We store the parsed JSON bodies for
    // index timestamp extraction later.
    std::vector<x::json::json> parsed_bodies(cfg_.endpoints.size());

    // Count total fields for frame reservation.
    size_t total_fields = cfg_.all_channel_keys.size();
    fr.reserve(total_fields);

    for (size_t ei = 0; ei < cfg_.endpoints.size(); ei++) {
        const auto &ep = cfg_.endpoints[ei];
        auto &[resp, req_err] = results[ei];

        if (req_err) {
            res.error = req_err;
            return res;
        }

        // Classify HTTP status code.
        if (auto status_err = device::classify_status(resp.status_code); status_err) {
            res.error = status_err;
            return res;
        }

        // Parse the response body as JSON.
        try {
            parsed_bodies[ei] = x::json::json::parse(resp.body);
        } catch (const x::json::json::parse_error &e) {
            res.error = errors::PARSE_ERROR.sub(
                "failed to parse response from " + ep.request.path + ": " + e.what()
            );
            return res;
        }

        const auto &body = parsed_bodies[ei];

        // Extract each field value.
        for (const auto &field : ep.fields) {
            if (!body.contains(field.pointer)) {
                res.error = errors::PARSE_ERROR.sub(
                    "field " + field.pointer.to_string() + " not found in response from " +
                        ep.request.path
                );
                return res;
            }

            const auto &json_val = body.at(field.pointer);

            // If this field has a custom time_format, use it for the conversion.
            auto opts = read_opts_;
            if (field.time_format.has_value()) opts.time_format = *field.time_format;

            auto [sample_val, conv_err] = x::json::to_sample_value(
                json_val,
                field.ch.data_type,
                opts
            );
            if (conv_err) {
                res.error = errors::PARSE_ERROR.sub(
                    "failed to convert " + field.pointer.to_string() +
                        " for channel " + field.ch.name + ": " + conv_err.message()
                );
                return res;
            }

            auto s = x::telem::Series(field.ch.data_type, 1);
            s.write(sample_val);
            fr.emplace(field.channel_key, std::move(s));
        }

        // Write index timestamps for index sources associated with this endpoint.
        for (const auto &idx : cfg_.index_sources) {
            if (idx.endpoint_index != static_cast<int>(ei)) continue;
            // Skip if we already wrote this index (from a previous endpoint).
            if (fr.contains(idx.index_key)) continue;

            x::telem::TimeStamp ts;
            if (idx.time_info.has_value()) {
                // Extract timestamp from response JSON.
                if (!body.contains(idx.time_info->pointer)) {
                    res.error = errors::PARSE_ERROR.sub(
                        "timestamp field " + idx.time_info->pointer.to_string() +
                            " not found in response from " + ep.request.path
                    );
                    return res;
                }
                const auto &ts_json = body.at(idx.time_info->pointer);
                auto ts_opts = x::json::ReadOptions{
                    .strict = cfg_.strict,
                    .time_format = idx.time_info->format,
                };
                auto [ts_val, ts_err] = x::json::to_sample_value(
                    ts_json,
                    x::telem::TIMESTAMP_T,
                    ts_opts
                );
                if (ts_err) {
                    res.error = errors::PARSE_ERROR.sub(
                        "failed to parse timestamp from " +
                            idx.time_info->pointer.to_string() + ": " +
                            ts_err.message()
                    );
                    return res;
                }
                ts = std::get<x::telem::TimeStamp>(ts_val);
            } else {
                // Software timing: midpoint of request time range.
                ts = x::telem::TimeStamp::midpoint(
                    resp.time_range.start, resp.time_range.end
                );
            }

            auto s = x::telem::Series(x::telem::TIMESTAMP_T, 1);
            s.write(ts);
            fr.emplace(idx.index_key, std::move(s));
        }
    }

    return res;
}

std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    auto [cfg, parse_err] = ReadTaskConfig::parse(ctx, task);
    if (parse_err) return {common::ConfigureResult{}, parse_err};

    // Retrieve connection config for the HTTP client.
    auto [conn, conn_err] = device::retrieve_connection(
        ctx->client->devices,
        cfg.device
    );
    if (conn_err) return {common::ConfigureResult{}, conn_err};

    // Build request configs for the client.
    std::vector<device::RequestConfig> request_configs;
    request_configs.reserve(cfg.endpoints.size());
    for (const auto &ep : cfg.endpoints)
        request_configs.push_back(ep.request);

    // Create the HTTP client.
    auto [client, client_err] = device::Client::create(
        std::move(conn),
        request_configs
    );
    if (client_err) return {common::ConfigureResult{}, client_err};

    const bool auto_start = cfg.auto_start;
    auto source = std::make_unique<ReadTaskSource>(
        std::move(cfg),
        std::move(client)
    );

    auto breaker_cfg = x::breaker::Config{
        .name = task.name,
        .base_interval = 1 * x::telem::SECOND,
        .max_retries = 50,
        .scale = 1.2,
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
