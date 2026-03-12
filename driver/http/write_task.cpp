// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>

#include "x/cpp/uuid/uuid.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/write_task.h"

namespace driver::http {
namespace {
/// @brief parses a json::Type from a string ("number", "string", "boolean").
/// @param parser the JSON parser.
/// @param path the field path.
/// @returns the parsed Type.
x::json::Type parse_json_type(x::json::Parser &parser, const std::string &path) {
    const auto str = parser.field<std::string>(path);
    if (str == "number") return x::json::Type::Number;
    if (str == "string") return x::json::Type::String;
    if (str == "boolean") return x::json::Type::Boolean;
    parser.field_err(path, "unknown JSON type '" + str + "'");
    return x::json::Type::Number;
}

/// @brief parses a GeneratorType from a string ("uuid", "timestamp").
/// @param parser the JSON parser.
/// @param path the field path.
/// @returns the parsed GeneratorType.
GeneratorType parse_generator_type(x::json::Parser &parser, const std::string &path) {
    const auto str = parser.field<std::string>(path);
    if (str == "uuid") return GeneratorType::UUID;
    if (str == "timestamp") return GeneratorType::Timestamp;
    parser.field_err(path, "unknown generator type '" + str + "'");
    return GeneratorType::UUID;
}

/// @brief builds the JSON body for a write endpoint from a sample value and
/// timestamp.
/// @param ep the write endpoint configuration.
/// @param sample_val the channel value as JSON.
/// @param ts the timestamp to use for time_config and timestamp generators.
/// @returns the serialized JSON body string.
std::pair<std::string, x::errors::Error> build_body(
    const WriteEndpoint &ep,
    const x::json::json &sample_val,
    const x::telem::TimeStamp &ts
) {
    // Bare primitive: if channel pointer is root and no other fields, body IS the
    // value directly.
    if (ep.channel.pointer == x::json::json::json_pointer("") &&
        ep.static_fields.empty() && ep.generated_fields.empty() &&
        !ep.channel.time_config.has_value()) {
        return {sample_val.dump(), x::errors::NIL};
    }

    x::json::json body;

    // Place channel value.
    body[ep.channel.pointer] = sample_val;

    // Place time config if present.
    if (ep.channel.time_config.has_value()) {
        const auto &tc = *ep.channel.time_config;
        body[tc.pointer] = x::json::from_timestamp(ts, tc.time_format);
    }

    // Place static fields.
    for (const auto &sf: ep.static_fields)
        body[sf.pointer] = sf.value;

    // Place generated fields.
    for (const auto &gf: ep.generated_fields) {
        if (gf.generator == GeneratorType::UUID) {
            body[gf.pointer] = x::uuid::create().to_string();
        } else {
            body[gf.pointer] = x::json::from_timestamp(ts, gf.time_format);
        }
    }

    return {body.dump(), x::errors::NIL};
}
}

std::pair<WriteTaskConfig, x::errors::Error> WriteTaskConfig::parse(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    auto parser = x::json::Parser(task.config);
    WriteTaskConfig cfg;
    cfg.device = parser.field<std::string>("device");
    cfg.auto_start = parser.field<bool>("auto_start", false);

    std::set<std::string> all_pointers;

    parser.iter("endpoints", [&](x::json::Parser &ep) {
        WriteEndpoint endpoint;
        endpoint.request.method = parse_method(ep, "method");
        endpoint.request.path = ep.field<std::string>("path");
        endpoint.request.request_content_type = ep.field<std::string>(
            "request_content_type",
            "application/json"
        );
        endpoint.request.headers = ep.field<std::map<std::string, std::string>>(
            "headers",
            std::map<std::string, std::string>{}
        );

        const auto method = endpoint.request.method;
        if (method != Method::POST && method != Method::PUT &&
            method != Method::PATCH) {
            ep.field_err("method", "write tasks only support POST, PUT, or PATCH");
        }

        all_pointers.clear();

        // Parse channel field.
        auto ch_parser = ep.child("channel");
        endpoint.channel.pointer = x::json::json::json_pointer(
            ch_parser.field<std::string>("pointer")
        );
        endpoint.channel.json_type = parse_json_type(ch_parser, "json_type");
        endpoint.channel.channel_key = ch_parser.field<synnax::channel::Key>("channel");

        const auto ch_ptr_str = endpoint.channel.pointer.to_string();
        all_pointers.insert(ch_ptr_str);

        // Parse optional time format for timestamp channels.
        const auto tf_str = ch_parser.field<std::string>("time_format", "");
        if (!tf_str.empty()) {
            auto [fmt, fmt_err] = x::json::parse_time_format(tf_str);
            if (fmt_err)
                ch_parser.field_err("time_format", fmt_err.message());
            else
                endpoint.channel.time_format = fmt;
        }

        // Parse optional time_config.
        if (ch_parser.has("time_config")) {
            auto tc_parser = ch_parser.child("time_config");
            TimeConfig tc;
            tc.pointer = x::json::json::json_pointer(
                tc_parser.field<std::string>("pointer")
            );
            const auto tc_fmt_str = tc_parser.field<std::string>("time_format");
            auto [tc_fmt, tc_err] = x::json::parse_time_format(tc_fmt_str);
            if (tc_err)
                tc_parser.field_err("time_format", tc_err.message());
            else
                tc.time_format = tc_fmt;
            const auto tc_ptr_str = tc.pointer.to_string();
            if (!all_pointers.insert(tc_ptr_str).second)
                tc_parser.field_err(
                    "pointer",
                    "pointer '" + tc_ptr_str + "' is already used"
                );
            endpoint.channel.time_config = tc;
        }

        // Parse static fields.
        ep.iter("fields", [&](x::json::Parser &fp) {
            const auto type = fp.field<std::string>("type");
            if (type == "static") {
                StaticField sf;
                sf.pointer = x::json::json::json_pointer(
                    fp.field<std::string>("pointer")
                );
                sf.value = fp.field<x::json::json>("value");
                const auto sf_ptr_str = sf.pointer.to_string();
                if (!all_pointers.insert(sf_ptr_str).second)
                    fp.field_err(
                        "pointer",
                        "pointer '" + sf_ptr_str + "' is already used"
                    );
                endpoint.static_fields.push_back(std::move(sf));
            } else if (type == "generated") {
                GeneratedField gf;
                gf.pointer = x::json::json::json_pointer(
                    fp.field<std::string>("pointer")
                );
                gf.generator = parse_generator_type(fp, "generator");
                if (gf.generator == GeneratorType::Timestamp) {
                    const auto gf_fmt_str = fp.field<std::string>(
                        "time_format",
                        "iso8601"
                    );
                    auto [gf_fmt, gf_err] = x::json::parse_time_format(gf_fmt_str);
                    if (gf_err)
                        fp.field_err("time_format", gf_err.message());
                    else
                        gf.time_format = gf_fmt;
                }
                const auto gf_ptr_str = gf.pointer.to_string();
                if (!all_pointers.insert(gf_ptr_str).second)
                    fp.field_err(
                        "pointer",
                        "pointer '" + gf_ptr_str + "' is already used"
                    );
                endpoint.generated_fields.push_back(std::move(gf));
            } else {
                fp.field_err("type", "unknown field type '" + type + "'");
            }
        });

        // Validate bare primitive: if channel pointer is root, no other fields.
        if (endpoint.channel.pointer == x::json::json::json_pointer("") &&
            (!endpoint.static_fields.empty() || !endpoint.generated_fields.empty() ||
             endpoint.channel.time_config.has_value())) {
            ep.field_err(
                "channel",
                "bare primitive body (root pointer) cannot have additional "
                "fields"
            );
        }

        cfg.cmd_keys.push_back(endpoint.channel.channel_key);
        cfg.endpoints.push_back(std::move(endpoint));
    });

    if (cfg.endpoints.empty())
        parser.field_err("endpoints", "at least one endpoint is required");

    if (!parser.ok()) return {std::move(cfg), parser.error()};

    // Fetch channels from Synnax to validate types.
    auto [sy_channels, ch_err] = ctx->client->channels.retrieve(cfg.cmd_keys);
    if (ch_err) return {{}, ch_err};

    std::map<synnax::channel::Key, synnax::channel::Channel> ch_map;
    for (const auto &ch: sy_channels)
        ch_map[ch.key] = ch;

    for (auto &ep: cfg.endpoints) {
        auto it = ch_map.find(ep.channel.channel_key);
        if (it == ch_map.end()) {
            parser.field_err(
                "endpoints",
                "channel " + std::to_string(ep.channel.channel_key) + " not found"
            );
            continue;
        }
        const auto &ch = it->second;

        // Validate data type vs json_type.
        if (auto conv_err = x::json::check_from_sample_value(
                ch.data_type,
                ep.channel.json_type
            );
            conv_err) {
            parser.field_err(
                "endpoints",
                "channel " + ch.name + " (type " + ch.data_type.name() +
                    ") cannot be converted to the specified JSON type"
            );
        }

        // Timestamp channels require time_format.
        if (ch.data_type == x::telem::TIMESTAMP_T &&
            !ep.channel.time_format.has_value()) {
            parser.field_err(
                "endpoints",
                "channel " + ch.name + " is a timestamp channel but has no time_format"
            );
        }

        // Determine if the index is virtual.
        ep.virtual_index = (ch.index == 0);
    }

    if (!parser.ok()) return {std::move(cfg), parser.error()};

    return {std::move(cfg), x::errors::NIL};
}

WriteTaskSink::WriteTaskSink(
    WriteTaskConfig cfg,
    std::shared_ptr<Processor> processor,
    std::vector<Request> base_requests
):
    Sink(cfg.cmd_keys),
    cfg(std::move(cfg)),
    processor(std::move(processor)),
    base_requests(std::move(base_requests)) {
    for (size_t i = 0; i < this->cfg.endpoints.size(); i++)
        channel_to_endpoint[this->cfg.endpoints[i].channel.channel_key] = i;
}

x::errors::Error WriteTaskSink::write(x::telem::Frame &frame) {
    for (const auto &[ch_key, series]: frame) {
        auto it = channel_to_endpoint.find(ch_key);
        if (it == channel_to_endpoint.end()) continue;
        const auto ep_idx = it->second;
        const auto &ep = cfg.endpoints[ep_idx];

        // Get the last sample value from the series.
        const auto sample_val = series.at(-1);

        // Convert sample value to JSON.
        auto [json_val, conv_err] = x::json::from_sample_value(
            sample_val,
            ep.channel.json_type
        );
        if (conv_err)
            return {
                conv_err.type,
                "failed to convert value for endpoint " + ep.request.path + ": " +
                    conv_err.data,
            };

        // If the channel is a timestamp type, format it.
        if (ep.channel.time_format.has_value()) {
            const auto ts_val = std::visit(
                [](auto &&v) -> int64_t {
                    using T = std::decay_t<decltype(v)>;
                    if constexpr (std::is_arithmetic_v<T>)
                        return static_cast<int64_t>(v);
                    else
                        return 0;
                },
                sample_val
            );
            json_val = x::json::from_timestamp(
                x::telem::TimeStamp(ts_val),
                *ep.channel.time_format
            );
        }

        // Determine the timestamp for time_config and generated timestamp fields.
        auto ts = ep.virtual_index ? x::telem::TimeStamp::now()
                                   : x::telem::TimeStamp::now();

        // Build the request body.
        auto [body, body_err] = build_body(ep, json_val, ts);
        if (body_err) return body_err;

        // Build and execute the request.
        auto req = base_requests[ep_idx];
        req.body = std::move(body);
        auto [resp, req_err] = processor->execute(req);
        if (req_err)
            return {
                req_err.type,
                std::string(to_string(ep.request.method)) + " " +
                    base_requests[ep_idx].url + ": " + req_err.data,
            };

        if (auto status_err = errors::from_status(resp.status_code); status_err) {
            auto msg = std::string(to_string(ep.request.method)) + " " +
                       base_requests[ep_idx].url + " returned " +
                       std::to_string(resp.status_code);
            if (!resp.body.empty()) msg += ": " + resp.body;
            return {status_err.type, msg};
        }
    }
    return x::errors::NIL;
}

std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<Processor> &processor
) {
    auto [cfg, parse_err] = WriteTaskConfig::parse(ctx, task);
    if (parse_err) return {common::ConfigureResult{}, parse_err};

    auto [conn, conn_err] = device::retrieve_connection(
        ctx->client->devices,
        cfg.device
    );
    if (conn_err) return {common::ConfigureResult{}, conn_err};

    std::vector<Request> base_requests;
    base_requests.reserve(cfg.endpoints.size());
    for (const auto &ep: cfg.endpoints)
        base_requests.push_back(device::build_request(conn, ep.request));

    const bool auto_start = cfg.auto_start;
    auto sink = std::make_unique<WriteTaskSink>(
        std::move(cfg),
        processor,
        std::move(base_requests)
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
            .auto_start = auto_start,
        },
        x::errors::NIL,
    };
}
}
