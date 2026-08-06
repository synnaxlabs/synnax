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
#include "x/cpp/uuid/uuid.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/write_task.h"

namespace driver::http {
namespace {
/// @brief parses an x::json::Type from its string form.
/// @param str the type string ("number", "string", "boolean").
/// @returns the parsed type paired with an error when the string is unknown.
std::pair<x::json::Type, x::errors::Error> parse_json_type(const std::string &str) {
    if (str == ::synnax::http::JSON_TYPE_NUMBER)
        return {x::json::Type::Number, x::errors::NIL};
    if (str == ::synnax::http::JSON_TYPE_STRING)
        return {x::json::Type::String, x::errors::NIL};
    if (str == ::synnax::http::JSON_TYPE_BOOLEAN)
        return {x::json::Type::Boolean, x::errors::NIL};
    return {
        x::json::Type::Number,
        x::errors::Error(x::errors::VALIDATION, "unknown JSON type '" + str + "'"),
    };
}

/// @brief parses a GeneratorType from its string form.
/// @param str the generator string ("uuid", "timestamp").
/// @returns the parsed generator paired with an error when the string is unknown.
std::pair<GeneratorType, x::errors::Error> parse_generator(const std::string &str) {
    if (str == ::synnax::http::GENERATOR_TYPE_UUID)
        return {GeneratorType::UUID, x::errors::NIL};
    if (str == ::synnax::http::GENERATOR_TYPE_TIMESTAMP)
        return {GeneratorType::Timestamp, x::errors::NIL};
    return {
        GeneratorType::UUID,
        x::errors::Error(x::errors::VALIDATION, "unknown generator type '" + str + "'"),
    };
}
}

std::pair<WriteTaskConfig, x::errors::Error> WriteTaskConfig::parse(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    auto parser = x::json::Parser(task.config);
    WriteTaskConfig cfg;
    static_cast<::synnax::http::WriteConfig &>(
        cfg
    ) = ::synnax::http::WriteConfig::parse(parser);
    if (cfg.device.empty()) parser.field_err("device", "this field is required");

    std::set<std::string> all_pointers;

    bool some_enabled = false;
    for (const auto &ep: cfg.endpoints) {
        validate_request(ep, parser);

        all_pointers.clear();
        validate_pointer(ep.channel.pointer, parser, "endpoints.channel.pointer");
        all_pointers.insert(ep.channel.pointer);
        if (auto [t, type_err] = parse_json_type(ep.channel.json_type); type_err)
            parser.field_err("endpoints.channel.json_type", type_err);
        if (ep.channel.channel == 0)
            parser.field_err("endpoints.channel.channel", "this field is required");
        if (ep.channel.time_format.has_value())
            if (auto [fmt, fmt_err] = x::json::parse_time_format(
                    *ep.channel.time_format
                );
                fmt_err)
                parser.field_err("endpoints.channel.time_format", fmt_err);
        if (ep.channel.enum_values.has_value()) {
            std::set<double> values;
            for (const auto &entry: *ep.channel.enum_values)
                if (!values.insert(entry.value).second)
                    parser.field_err(
                        "endpoints.channel.enum_values.value",
                        "duplicate enum value " + x::json::json(entry.value).dump()
                    );
            if (!ep.channel.enum_values->empty() &&
                ep.channel.json_type != ::synnax::http::JSON_TYPE_STRING)
                parser.field_err(
                    "endpoints.channel.enum_values",
                    "enum values are only supported when json_type is 'string'"
                );
        }

        for (const auto &field: ep.fields) {
            const std::string *pointer = nullptr;
            if (const auto *sf = std::get_if<::synnax::http::WriteFieldStatic>(
                    &field
                )) {
                pointer = &sf->pointer;
                if (!sf->value.has_value())
                    parser.field_err(
                        "endpoints.fields.value",
                        "this field is required"
                    );
            } else if (
                const auto *gf = std::get_if<::synnax::http::WriteFieldGenerated>(
                    &field
                )
            ) {
                pointer = &gf->pointer;
                if (auto [g, gen_err] = parse_generator(gf->generator); gen_err)
                    parser.field_err("endpoints.fields.generator", gen_err);
                if (gf->time_format.has_value())
                    if (auto [fmt, fmt_err] = x::json::parse_time_format(
                            *gf->time_format
                        );
                        fmt_err)
                        parser.field_err("endpoints.fields.time_format", fmt_err);
            }
            if (pointer == nullptr) continue;
            validate_pointer(*pointer, parser, "endpoints.fields.pointer");
            if (pointer->empty())
                parser.field_err(
                    "endpoints.fields.pointer",
                    "field pointer cannot be empty"
                );
            else if (!all_pointers.insert(*pointer).second)
                parser.field_err(
                    "endpoints.fields.pointer",
                    "pointer '" + *pointer + "' is already used"
                );
        }

        // Validate bare primitive: if channel pointer is root, no other fields.
        if (ep.channel.pointer.empty() && !ep.fields.empty())
            parser.field_err(
                "endpoints.channel",
                "bare primitive body (root pointer) cannot have additional "
                "fields"
            );

        if (!ep.disabled) {
            some_enabled = true;
            cfg.cmd_keys.push_back(ep.channel.channel);
        }
    }

    if (!some_enabled)
        parser.field_err("endpoints", "at least one enabled endpoint is required");

    if (!parser.ok()) return {std::move(cfg), parser.error()};

    auto [sy_channels, ch_err] = ctx->client->channels.retrieve(cfg.cmd_keys);
    if (ch_err) return {{}, ch_err};

    std::map<synnax::channel::Key, synnax::channel::Channel> ch_map;
    for (const auto &ch: sy_channels)
        ch_map[ch.key] = ch;

    for (const auto &ep: cfg.endpoints) {
        if (ep.disabled) continue;
        auto it = ch_map.find(ep.channel.channel);
        if (it == ch_map.end()) {
            parser.field_err(
                "endpoints",
                "channel " + std::to_string(ep.channel.channel) + " not found"
            );
            continue;
        }
        const auto &ch = it->second;

        // Validate data type vs json_type.
        if (auto conv_err = x::json::check_from_sample_value(
                ch.data_type,
                parse_json_type(ep.channel.json_type).first
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
    this->endpoints.reserve(this->cfg.endpoints.size());
    for (size_t i = 0; i < this->cfg.endpoints.size(); i++) {
        const auto &ep = this->cfg.endpoints[i];
        Endpoint state;
        state.method = parse_method(ep.method).first;
        state.pointer = x::json::json::json_pointer(ep.channel.pointer);
        state.json_type = parse_json_type(ep.channel.json_type).first;
        if (ep.channel.time_format.has_value())
            if (auto [fmt, fmt_err] = x::json::parse_time_format(
                    *ep.channel.time_format
                );
                !fmt_err)
                state.time_format = fmt;
        if (ep.channel.enum_values.has_value())
            for (const auto &entry: *ep.channel.enum_values)
                state.enum_values.emplace(x::json::json(entry.value), entry.label);
        for (const auto &field: ep.fields)
            if (const auto *sf = std::get_if<::synnax::http::WriteFieldStatic>(
                    &field
                )) {
                StaticField state_field;
                state_field.pointer = x::json::json::json_pointer(sf->pointer);
                state_field.value = sf->value.value_or(x::json::json());
                state.static_fields.push_back(std::move(state_field));
            } else if (
                const auto *gf = std::get_if<::synnax::http::WriteFieldGenerated>(
                    &field
                )
            ) {
                GeneratedField gen_field;
                gen_field.pointer = x::json::json::json_pointer(gf->pointer);
                gen_field.generator = parse_generator(gf->generator).first;
                if (gf->time_format.has_value())
                    if (auto [fmt, fmt_err] = x::json::parse_time_format(
                            *gf->time_format
                        );
                        !fmt_err)
                        gen_field.time_format = fmt;
                state.generated_fields.push_back(std::move(gen_field));
            }
        if (!ep.disabled) this->channel_to_endpoint[ep.channel.channel] = i;
        this->endpoints.push_back(std::move(state));
    }
}

std::string
WriteTaskSink::build_body(const Endpoint &ep, const x::json::json &sample_val) {
    // Bare primitive: if channel pointer is root and no other fields, body IS the value
    // directly.
    if (ep.pointer.empty() && ep.static_fields.empty() && ep.generated_fields.empty())
        return sample_val.dump();

    x::json::json body;
    body[ep.pointer] = sample_val;

    for (const auto &sf: ep.static_fields)
        body[sf.pointer] = sf.value;

    const auto now = x::telem::TimeStamp::now();
    for (const auto &gf: ep.generated_fields) {
        if (gf.generator == GeneratorType::UUID)
            body[gf.pointer] = x::uuid::create().to_string();
        else
            body[gf.pointer] = x::json::from_timestamp(now, gf.time_format);
    }

    return body.dump();
}

x::errors::Error WriteTaskSink::write(x::telem::Frame &frame) {
    struct PendingRequest {
        Request request;
        size_t ep_idx;
        synnax::channel::Key ch_key;
        x::telem::Series series;
    };

    std::vector<PendingRequest> pending;
    for (const auto &[ch_key, series]: frame) {
        auto it = channel_to_endpoint.find(ch_key);
        if (it == channel_to_endpoint.end()) continue;
        const auto ep_idx = it->second;
        const auto &ep = this->endpoints[ep_idx];

        const auto sample_val = series.at(-1);

        const auto *enum_ptr = ep.enum_values.empty() ? nullptr : &ep.enum_values;
        auto [json_val, conv_err] = x::json::from_sample_value(
            sample_val,
            ep.json_type,
            enum_ptr
        );
        if (conv_err)
            return {
                conv_err.type,
                "failed to convert value for endpoint " +
                    this->cfg.endpoints[ep_idx].path + ": " + conv_err.data,
            };

        if (ep.time_format.has_value()) {
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
                *ep.time_format
            );
        }

        auto req = base_requests[ep_idx];
        req.body = build_body(ep, json_val);
        pending.push_back({
            .request = std::move(req),
            .ep_idx = ep_idx,
            .ch_key = ch_key,
            .series = series.shallow_copy(),
        });
    }

    if (pending.empty()) return x::errors::NIL;

    std::vector<std::string> error_msgs;
    std::string first_error_type;

    while (!pending.empty()) {
        std::vector<Request> batch;
        batch.reserve(pending.size());
        for (const auto &p: pending)
            batch.push_back(p.request);
        auto results = processor->execute(batch);

        std::vector<PendingRequest> still_pending;
        bool made_progress = false;
        x::telem::Frame success_frame;

        for (size_t i = 0; i < results.size(); i++) {
            const auto &p = pending[i];
            const auto &ep = this->endpoints[p.ep_idx];
            auto &[resp, req_err] = results[i];

            x::errors::Error err = x::errors::NIL;
            if (req_err)
                err = req_err;
            else if (
                auto status_err = errors::from_status(resp.status_code); status_err
            )
                err = status_err;

            if (!err) {
                made_progress = true;
                success_frame.emplace(p.ch_key, p.series.shallow_copy());
                continue;
            }

            if (err.matches(errors::TEMPORARY_ERROR)) {
                still_pending.push_back(std::move(pending[i]));
            } else {
                made_progress = true;
                if (first_error_type.empty()) first_error_type = err.type;
                auto msg = std::string(to_string(ep.method)) + " " +
                           base_requests[p.ep_idx].url;
                if (req_err)
                    msg += ": " + req_err.data;
                else {
                    msg += " returned " + std::to_string(resp.status_code);
                    if (!resp.body.empty()) msg += ": " + resp.body;
                }
                error_msgs.push_back(std::move(msg));
            }
        }

        if (!success_frame.empty()) this->set_state(success_frame);
        pending = std::move(still_pending);
        if (!made_progress) break;
    }

    for (const auto &p: pending) {
        if (first_error_type.empty()) first_error_type = errors::TEMPORARY_ERROR.type;
        const auto &ep = this->endpoints[p.ep_idx];
        error_msgs.push_back(
            std::string(to_string(ep.method)) + " " + base_requests[p.ep_idx].url +
            ": timed out"
        );
    }

    if (error_msgs.empty()) return x::errors::NIL;
    return {first_error_type, x::strings::join(error_msgs, "; ")};
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
    for (const auto &ep: cfg.endpoints) {
        auto req_cfg = request_config(ep);
        req_cfg.request_content_type = "application/json";
        base_requests.push_back(device::build_request(conn, req_cfg));
    }

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
