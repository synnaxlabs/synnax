// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <charconv>
#include <optional>
#include <set>
#include <type_traits>
#include <variant>

#include "client/cpp/http/json.gen.h"
#include "x/cpp/base64/base64.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/strings/strings.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/write_task.h"

namespace driver::http {
namespace {
/// @brief parses the text of a string entry as a sample of a channel of the data type,
/// as the sample it stands for: a number for a numeric or boolean channel, and the text
/// itself otherwise.
std::pair<x::telem::SampleValue, x::errors::Error>
parse_text_entry(const std::string &text, const x::telem::DataType &dt) {
    const auto kind = kind_of(dt);
    const auto fail = [&] {
        return std::pair<x::telem::SampleValue, x::errors::Error>{
            x::telem::SampleValue{},
            x::errors::Error(
                x::errors::VALIDATION,
                "\"" + text + "\" is not a " + dt.name() + " value"
            ),
        };
    };
    if (kind == ::x::payload::KIND_FLOAT) {
        double v = 0;
        auto [end, ec] = std::from_chars(text.data(), text.data() + text.size(), v);
        if (ec != std::errc() || end != text.data() + text.size()) return fail();
        return {v, x::errors::NIL};
    }
    if (kind == ::x::payload::KIND_INTEGER) {
        int64_t v = 0;
        auto [end, ec] = std::from_chars(text.data(), text.data() + text.size(), v);
        if (ec != std::errc() || end != text.data() + text.size()) return fail();
        return {v, x::errors::NIL};
    }
    if (kind == ::x::payload::KIND_UNSIGNED || kind == ::x::payload::KIND_BOOLEAN) {
        uint64_t v = 0;
        auto [end, ec] = std::from_chars(text.data(), text.data() + text.size(), v);
        if (ec != std::errc() || end != text.data() + text.size()) return fail();
        return {v, x::errors::NIL};
    }
    return {text, x::errors::NIL};
}

/// @brief returns the labels that a numeric or string channel field writes in place of
/// the samples of a channel of the data type.
std::pair<x::json::ReverseEnumMap, x::errors::Error>
enums_of(const ::synnax::http::ChannelField &field, const x::telem::DataType &dt) {
    x::json::ReverseEnumMap out;
    if (const auto *f = std::get_if<::synnax::http::FloatChannelField>(&field))
        for (const auto &e: f->enum_values)
            out.emplace_back(e.value.value, e.label);
    else if (const auto *f = std::get_if<::synnax::http::IntegerChannelField>(&field))
        for (const auto &e: f->enum_values)
            out.emplace_back(e.value.value, e.label);
    else if (const auto *f = std::get_if<::synnax::http::UnsignedChannelField>(&field))
        for (const auto &e: f->enum_values)
            out.emplace_back(e.value.value, e.label);
    else if (const auto *f = std::get_if<::synnax::http::StringChannelField>(&field))
        for (const auto &e: f->enum_values) {
            auto [v, err] = parse_text_entry(e.value, dt);
            if (err) return {{}, err};
            out.emplace_back(std::move(v), e.label);
        }
    return {std::move(out), x::errors::NIL};
}

/// @brief returns the kind that a channel field writes its samples as.
std::string kind_of_field(const ::synnax::http::ChannelField &field) {
    return std::visit([](const auto &f) { return f.type; }, field);
}

/// @brief returns the JSON type that a numeric, string, or boolean kind writes.
x::json::Type json_type_of(const std::string &kind) {
    if (kind == ::x::payload::KIND_STRING) return x::json::Type::String;
    if (kind == ::x::payload::KIND_BOOLEAN) return x::json::Type::Boolean;
    return x::json::Type::Number;
}

/// @brief returns the JSON value that a fixed write field places in every body, or
/// nullopt for a field that generates its value.
std::pair<std::optional<x::json::json>, x::errors::Error>
fixed_value(const ::x::payload::WriteField &field) {
    using R = std::pair<std::optional<x::json::json>, x::errors::Error>;
    return std::visit(
        [](const auto &f) -> R {
            using F = std::decay_t<decltype(f)>;
            if constexpr (
                std::is_same_v<F, ::x::payload::FloatWriteField> ||
                std::is_same_v<F, ::x::payload::IntegerWriteField> ||
                std::is_same_v<F, ::x::payload::UnsignedWriteField>
            )
                return {x::json::json(f.value.value), x::errors::NIL};
            else if constexpr (std::is_same_v<F, ::x::payload::JsonWriteField>) {
                auto value = x::json::json::parse(f.value, nullptr, false);
                if (value.is_discarded())
                    return {
                        std::nullopt,
                        x::errors::Error(x::errors::VALIDATION, "value is not JSON"),
                    };
                return {std::move(value), x::errors::NIL};
            } else if constexpr (std::is_same_v<F, ::x::payload::BytesWriteField>) {
                if (auto [_, err] = x::base64::decode_strict(f.value); err)
                    return {
                        std::nullopt,
                        x::errors::Error(x::errors::VALIDATION, "value is not base64"),
                    };
                return {x::json::json(f.value), x::errors::NIL};
            } else if constexpr (
                std::is_same_v<F, ::x::payload::UuidWriteField> ||
                std::is_same_v<F, ::x::payload::TimestampWriteField>
            )
                return {std::nullopt, x::errors::NIL};
            else
                return {x::json::json(f.value), x::errors::NIL};
        },
        field
    );
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
        const auto &channel = ::synnax::http::base(ep.channel);
        validate_pointer(channel.pointer, parser, "endpoints.channel.pointer");
        all_pointers.insert(channel.pointer);
        if (channel.channel == 0)
            parser.field_err("endpoints.channel.channel", "this field is required");
        if (const auto *ts = std::get_if<::synnax::http::TimestampChannelField>(
                &ep.channel
            ))
            if (auto [fmt, fmt_err] = x::json::parse_time_format(ts->time_format);
                fmt_err)
                parser.field_err("endpoints.channel.time_format", fmt_err);

        for (const auto &field: ep.fields) {
            const std::string *pointer = &::x::payload::base(field).pointer;
            if (const auto *tf = std::get_if<::x::payload::TimestampWriteField>(&field))
                if (auto [fmt, fmt_err] = x::json::parse_time_format(tf->time_format);
                    fmt_err)
                    parser.field_err("endpoints.fields.time_format", fmt_err);
            if (auto [value, value_err] = fixed_value(field); value_err)
                parser.field_err("endpoints.fields.value", value_err);
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
        if (channel.pointer.empty() && !ep.fields.empty())
            parser.field_err(
                "endpoints.channel",
                "bare primitive body (root pointer) cannot have additional "
                "fields"
            );

        if (!ep.disabled) {
            some_enabled = true;
            cfg.cmd_keys.push_back(channel.channel);
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
        const auto key = ::synnax::http::base(ep.channel).channel;
        auto it = ch_map.find(key);
        if (it == ch_map.end()) {
            parser.field_err(
                "endpoints",
                "channel " + std::to_string(key) + " not found"
            );
            continue;
        }
        const auto &ch = it->second;
        cfg.cmd_names[ch.key] = ch.name;
        cfg.cmd_types[ch.key] = ch.data_type;
        const auto kind = kind_of_field(ep.channel);
        const auto kinds = channel_kinds(ch.data_type);
        if (std::find(kinds.begin(), kinds.end(), kind) == kinds.end()) {
            parser.field_err(
                "endpoints.channel.type",
                "channel " + ch.name + " (type " + ch.data_type.name() +
                    ") cannot be written as " + kind
            );
            continue;
        }
        if (auto [_, enum_err] = enums_of(ep.channel, ch.data_type); enum_err)
            parser.field_err("endpoints.channel.enum_values", enum_err);
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
        const auto &channel = ::synnax::http::base(ep.channel);
        state.pointer = x::json::json::json_pointer(channel.pointer);
        state.kind = kind_of_field(ep.channel);
        state.json_type = json_type_of(state.kind);
        if (const auto *ts = std::get_if<::synnax::http::TimestampChannelField>(
                &ep.channel
            ))
            state.time_format = x::json::parse_time_format(ts->time_format).first;
        if (const auto *b = std::get_if<::synnax::http::BooleanChannelField>(
                &ep.channel
            ))
            for (const auto &e: b->enum_values)
                state.boolean_labels.emplace_back(e.value, e.label);
        if (const auto dt = this->cfg.cmd_types.find(channel.channel);
            dt != this->cfg.cmd_types.end())
            state.enum_values = enums_of(ep.channel, dt->second).first;
        for (const auto &field: ep.fields) {
            const auto pointer = x::json::json::json_pointer(
                ::x::payload::base(field).pointer
            );
            if (auto value = fixed_value(field).first; value.has_value()) {
                state.static_fields.push_back({pointer, std::move(*value)});
                continue;
            }
            GeneratedField gen_field;
            gen_field.pointer = pointer;
            if (const auto *tf = std::get_if<::x::payload::TimestampWriteField>(
                    &field
                )) {
                gen_field.generator = GeneratorType::Timestamp;
                gen_field.time_format = x::json::parse_time_format(tf->time_format)
                                            .first;
            }
            state.generated_fields.push_back(std::move(gen_field));
        }
        if (!ep.disabled) this->channel_to_endpoint[channel.channel] = i;
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

std::pair<x::json::json, x::errors::Error>
WriteTaskSink::encode(const Endpoint &ep, const x::telem::Series &series) {
    if (ep.kind == ::x::payload::KIND_JSON || ep.kind == ::x::payload::KIND_UUID ||
        ep.kind == ::x::payload::KIND_BYTES)
        return x::json::from_series(series, -1);
    const auto sample = series.at(-1);
    if (ep.kind == ::x::payload::KIND_TIMESTAMP) {
        const auto *ts = std::get_if<x::telem::TimeStamp>(&sample);
        if (ts == nullptr) return {x::json::json(), x::json::CONVERSION_ERROR};
        return {x::json::from_timestamp(*ts, ep.time_format), x::errors::NIL};
    }
    if (ep.kind == ::x::payload::KIND_BOOLEAN) {
        auto [value, err] = x::json::from_sample_value(sample, x::json::Type::Boolean);
        if (err) return {std::move(value), err};
        for (const auto &[v, label]: ep.boolean_labels)
            if (v == value.get<bool>()) return {x::json::json(label), x::errors::NIL};
        return {std::move(value), x::errors::NIL};
    }
    const auto *enums = ep.enum_values.empty() ? nullptr : &ep.enum_values;
    return x::json::from_sample_value(sample, ep.json_type, enums);
}

x::errors::Error WriteTaskSink::write(x::telem::Frame &frame) {
    struct PendingRequest {
        Request request;
        size_t ep_idx;
        synnax::channel::Key ch_key;
        x::telem::Series series;
    };

    std::vector<PendingRequest> pending;
    std::vector<std::string> skipped;
    for (const auto &[ch_key, series]: frame) {
        auto it = channel_to_endpoint.find(ch_key);
        if (it == channel_to_endpoint.end()) continue;
        const auto ep_idx = it->second;
        const auto &ep = this->endpoints[ep_idx];

        auto [json_val, conv_err] = encode(ep, series);
        if (conv_err) {
            const auto reason = conv_err.data.empty()
                                  ? "the value has no JSON form for the field"
                                  : conv_err.data;
            skipped.push_back(
                "Channel " + this->cfg.cmd_names.at(ch_key) + ": " + reason +
                ". Did not send " + to_string(ep.method) + " " +
                base_requests[ep_idx].url + ". Write a finite number."
            );
            continue;
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

    if (!skipped.empty())
        this->warn({
            .message = "Skipped invalid command",
            .description = x::strings::join(skipped, " "),
        });

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
