// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <limits>
#include <optional>
#include <set>
#include <string>
#include <type_traits>
#include <utility>
#include <variant>

#include "client/cpp/http/json.gen.h"
#include "x/cpp/payload/number.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/strings/strings.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"
#include "driver/http/read_task.h"

namespace driver::http {
namespace {
bool is_absent(
    const x::json::json &body,
    const x::json::json::json_pointer &pointer,
    const x::telem::DataType &data_type
) {
    if (!body.contains(pointer)) return true;
    const auto &value = body.at(pointer);
    if (value.is_null()) return true;
    return data_type != x::telem::STRING_T && value.is_string() &&
           value.get_ref<const std::string &>().empty();
}

/// @brief returns the data type that x::json converts a value of the channel to. It
/// converts no boolean, so a boolean reads as a uint8 that casts to 0 or 1.
x::telem::DataType conversion_type(const x::telem::DataType &dt) {
    return dt == x::telem::BOOLEAN_T ? x::telem::UINT8_T : dt;
}

x::telem::SampleValue to_sample(const x::payload::Float &v) {
    return v.value;
}
x::telem::SampleValue to_sample(const x::payload::Integer &v) {
    return v.value;
}
x::telem::SampleValue to_sample(const x::payload::Unsigned &v) {
    return v.value;
}
x::telem::SampleValue to_sample(const std::string &v) {
    return v;
}
x::telem::SampleValue to_sample(const bool v) {
    return static_cast<uint8_t>(v ? 1 : 0);
}
x::telem::SampleValue to_sample(const x::telem::TimeStamp &v) {
    return v;
}

x::json::EnumMap enum_map(const ::synnax::http::ReadField &field) {
    x::json::EnumMap map;
    std::visit(
        [&map](const auto &f) {
            if constexpr (requires { f.enum_values; })
                for (const auto &entry: f.enum_values)
                    map.emplace(entry.label, to_sample(entry.value));
        },
        field
    );
    return map;
}

x::json::TimeFormat time_format(const ::synnax::http::ReadField &field) {
    if (const auto *f = std::get_if<::synnax::http::TimestampReadField>(&field))
        return x::json::parse_time_format(f->time_format).first;
    return x::json::TimeFormat::ISO8601;
}

/// @brief converts the fill of a field to the data type of its channel.
/// @returns the fill, or nothing when the field has none, paired with an error when
/// the fill does not fit the channel.
std::pair<std::optional<x::telem::SampleValue>, x::errors::Error>
fill_sample(const ::synnax::http::ReadField &field, const x::telem::DataType &dt) {
    return std::visit(
        [&dt](const auto &f)
            -> std::pair<std::optional<x::telem::SampleValue>, x::errors::Error> {
            if constexpr (requires { f.fill; }) {
                if (!f.fill.has_value()) return {std::nullopt, x::errors::NIL};
                using V = std::decay_t<decltype(*f.fill)>;
                if constexpr (
                    std::is_same_v<V, x::payload::Float> ||
                    std::is_same_v<V, x::payload::Integer> ||
                    std::is_same_v<V, x::payload::Unsigned>
                ) {
                    auto [sample, err] = x::json::to_sample_value(
                        x::json::json(f.fill->value),
                        dt
                    );
                    if (err) return {std::nullopt, err};
                    return {std::move(sample), x::errors::NIL};
                } else
                    return {to_sample(*f.fill), x::errors::NIL};
            } else
                return {std::nullopt, x::errors::NIL};
        },
        field
    );
}
}

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
            const auto &b = ::synnax::http::base(field);
            validate_pointer(b.pointer, parser, "endpoints.fields.pointer");
            if (b.channel == 0)
                parser.field_err("endpoints.fields.channel", "this field is required");
            else if (!field_keys.insert(b.channel).second)
                parser.field_err(
                    "endpoints.fields.channel",
                    "channel " + std::to_string(b.channel) + " is used multiple times"
                );
            if (!b.disabled) {
                enabled_field_count++;
                enabled_field_keys.insert(b.channel);
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

    for (int ei = 0; std::cmp_less(ei, cfg.endpoints.size()); ei++) {
        for (const auto &field: cfg.endpoints[ei].fields) {
            const auto &b = ::synnax::http::base(field);
            if (b.disabled) continue;
            auto it = cfg.channels.find(b.channel);
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

            const auto &type = std::visit(
                [](const auto &f) -> const std::string & { return f.type; },
                field
            );
            if (const auto kind = kind_of(dt); type != kind) {
                parser.field_err(
                    "endpoints.fields.type",
                    "channel " + ch.name + " has data type " + dt.name() +
                        ", so the field type must be " + kind + ", not " + type
                );
                continue;
            }

            auto [fill, fill_err] = fill_sample(field, dt);
            if (fill_err) {
                parser.field_err(
                    "endpoints.fields.fill",
                    "channel " + ch.name + ": " + fill_err.data
                );
                continue;
            }
            if (fill.has_value()) {
                if (ch.is_index) {
                    parser.field_err(
                        "endpoints.fields.fill",
                        "channel " + ch.name +
                            " is an index channel, so it cannot have a fill"
                    );
                    continue;
                }
                cfg.fill_values.emplace(b.channel, std::move(*fill));
            }

            if (ch.index == 0) continue;
            const auto idx_key = ch.index;
            if (enabled_field_keys.contains(idx_key)) continue;
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
            const auto &b = ::synnax::http::base(field);
            if (b.disabled) continue;
            auto it = this->cfg.channels.find(b.channel);
            if (it == this->cfg.channels.end()) continue;
            this->chs.push_back(it->second);

            GroupField gf;
            gf.channel = b.channel;
            gf.pointer = x::json::json::json_pointer(b.pointer);
            if (const auto fill = this->cfg.fill_values.find(b.channel);
                fill != this->cfg.fill_values.end())
                gf.fill = fill->second;
            gf.time_format = time_format(field);
            gf.enum_values = enum_map(field);

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
                    .software_timed_index = this->cfg.software_timed_indexes.contains(
                        idx_key
                    ),
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
            const auto &b = ::synnax::http::base(field);
            if (b.disabled) continue;
            keys.push_back(b.channel);
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

    const auto tick_start = x::telem::TimeStamp::now();
    auto results = this->processor->execute(this->requests);
    const auto tick = x::telem::TimeStamp::now() - tick_start;

    fr.reserve(this->cfg.channels.size() + this->cfg.software_timed_indexes.size());

    std::vector<std::string> warnings;
    std::size_t skipped = 0;
    std::size_t failed = 0;
    std::size_t rejected = 0;
    std::size_t unreachable = 0;

    // Parse all response bodies up front so sampling groups can reference them.
    std::vector<bool> ep_parsed(this->cfg.endpoints.size(), false);
    for (size_t ei = 0; ei < this->cfg.endpoints.size(); ei++) {
        const auto &ep = this->cfg.endpoints[ei];
        auto &[resp, req_err] = results[ei];

        if (req_err.matches(errors::UNREACHABLE_ERROR)) unreachable++;
        if (req_err.matches(errors::SKIPPED_ERROR)) {
            skipped++;
            continue;
        }

        if (req_err) {
            failed++;
            const auto &req = requests[ei];
            warnings.push_back(
                std::string(to_string(req.method)) + " " + req.url +
                " failed: " + req_err.data
            );
            continue;
        }

        if (auto status_err = errors::from_status(resp.status_code); status_err) {
            rejected++;
            const auto &req = requests[ei];
            auto msg = std::string(to_string(req.method)) + " " + req.url +
                       " returned " + std::to_string(resp.status_code);
            if (!resp.body.empty()) msg += ": " + resp.body_excerpt();
            warnings.push_back(msg);
            continue;
        }

        try {
            this->parsed_bodies[ei] = x::json::json::parse(resp.body);
            ep_parsed[ei] = true;
        } catch (const x::json::json::parse_error &e) {
            warnings.push_back(
                "Failed to parse response from " + ep.path + ": " + e.what()
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
            const auto &ch = this->cfg.channels.at(field.channel);
            const auto &dt = ch.data_type;
            if (field.fill.has_value() && is_absent(body, field.pointer, dt)) {
                group_data.emplace_back(
                    field.channel,
                    x::telem::Series(*field.fill, dt)
                );
                continue;
            }
            x::telem::SampleValue sample_val;
            std::string warning;
            if (!body.contains(field.pointer))
                warning = "Field " + field.pointer.to_string() +
                          " not found in response from " + ep.path;
            else {
                const auto *enum_ptr = field.enum_values.empty() ? nullptr
                                                                 : &field.enum_values;
                auto [val, conv_err] = x::json::to_sample_value(
                    body.at(field.pointer),
                    conversion_type(dt),
                    field.time_format,
                    enum_ptr
                );
                if (conv_err)
                    warning = "Failed to convert " + field.pointer.to_string() +
                              " for channel " + ch.name + ": " + conv_err.data;
                else
                    sample_val = dt.cast(val);
            }

            if (!warning.empty()) {
                warnings.push_back(std::move(warning));
                if (!this->cfg.nan_fill_enabled ||
                    (dt != x::telem::FLOAT32_T && dt != x::telem::FLOAT64_T)) {
                    group_ok = false;
                    break;
                }
                sample_val = dt.cast(std::numeric_limits<double>::quiet_NaN());
            }

            group_data.emplace_back(field.channel, x::telem::Series(sample_val, dt));
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
