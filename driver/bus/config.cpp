// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <map>
#include <set>
#include <tuple>
#include <unordered_map>
#include <variant>

#include "driver/bus/config.h"

namespace driver::bus {
namespace {
using ChannelMap = std::unordered_map<synnax::channel::Key, synnax::channel::Channel>;

/// @returns the parts every field of the message shares, in field order.
std::vector<const synnax::icd::BaseField *>
field_bases(const synnax::icd::MessageEntry &message) {
    std::vector<const synnax::icd::BaseField *> bases;
    if (const auto *b = std::get_if<synnax::icd::BinaryPayload>(&message.payload))
        for (const auto &f: b->fields)
            bases.push_back(&f);
    else
        for (const auto &f: std::get<synnax::icd::TextPayload>(message.payload).fields)
            bases.push_back(
                std::visit(
                    [](const auto &v) -> const synnax::icd::BaseField * { return &v; },
                    f
                )
            );
    return bases;
}

const synnax::icd::MessageEntry *
find_message(const synnax::icd::ICD &ic, const synnax::icd::EntryKey &key) {
    for (const auto &entry: ic.entries)
        if (const auto *m = std::get_if<synnax::icd::MessageEntry>(&entry);
            m != nullptr && m->key == key)
            return m;
    return nullptr;
}

std::optional<std::size_t> find_field(
    const std::vector<const synnax::icd::BaseField *> &bases,
    const synnax::icd::FieldKey &key
) {
    for (std::size_t i = 0; i < bases.size(); i++)
        if (bases[i]->key == key) return i;
    return std::nullopt;
}

std::string path(const std::size_t message, const std::string &rest) {
    return "messages." + std::to_string(message) + "." + rest;
}

std::string field_path(const std::size_t message, const std::size_t field) {
    return path(message, "fields." + std::to_string(field) + ".field");
}

/// @brief binds an error when the medium cannot carry the message: a CAN bus carries
/// binary payloads with a CAN identifier or none, and a byte stream carries any payload
/// without a CAN identifier.
void check_medium(
    const x::json::Parser &parser,
    const std::size_t index,
    const synnax::icd::MessageEntry &message,
    const Medium medium
) {
    const bool can = medium == Medium::CAN;
    std::string what;
    const auto *binary = std::get_if<synnax::icd::BinaryPayload>(&message.payload);
    if (binary == nullptr) {
        if (can) what = "a text payload";
    } else if (binary->identifier.has_value()) {
        const auto &id = *binary->identifier;
        if (std::holds_alternative<synnax::icd::CanIdentifier>(id) != can)
            what = "a " + std::visit([](const auto &i) { return i.type; }, id) +
                   " identifier";
    }
    if (what.empty()) return;
    parser.field_err(
        path(index, "message"),
        "message " + message.name + " has " + what + ", which " +
            (can ? "a CAN bus" : "a byte stream") + " cannot carry"
    );
}

/// @brief the data types a read task writes decoded values to.
const std::set<x::telem::DataType> DECODED_TYPES = {
    x::telem::FLOAT64_T,
    x::telem::FLOAT32_T,
    x::telem::INT64_T,
    x::telem::INT32_T,
    x::telem::INT16_T,
    x::telem::INT8_T,
    x::telem::UINT64_T,
    x::telem::UINT32_T,
    x::telem::UINT16_T,
    x::telem::UINT8_T,
};

/// @brief binds an error when the channel cannot hold commanded values, or decoded
/// values when decoded is true.
void check_numeric(
    const x::json::Parser &parser,
    const std::string &at,
    const synnax::channel::Channel &ch,
    const bool decoded
) {
    const auto &t = ch.data_type;
    if (decoded ? !DECODED_TYPES.contains(t) : t.is_variable() || t == x::telem::UUID_T)
        parser.field_err(
            at,
            "channel " + ch.name + " has data type " + ch.data_type.name() +
                ", which cannot hold a numeric field"
        );
}

const synnax::channel::Channel *find_channel(
    const x::json::Parser &parser,
    const std::string &at,
    const ChannelMap &channels,
    const synnax::channel::Key key
) {
    if (key == 0) {
        parser.field_err(at, "this field is required");
        return nullptr;
    }
    const auto it = channels.find(key);
    if (it != channels.end()) return &it->second;
    parser.field_err(at, "channel " + std::to_string(key) + " does not exist");
    return nullptr;
}

/// @returns the plan of message, over keys when given. The Core checks every layout it
/// stores, so a layout the Driver cannot compile is a bug in one of the two, and the
/// error is x::errors::INTERNAL.
std::pair<codec::Plan, x::errors::Error> compile_plan(
    const synnax::icd::MessageEntry &message,
    const std::optional<std::vector<synnax::icd::FieldKey>> &keys = std::nullopt
) {
    auto [plan, err] = keys.has_value() ? codec::Plan::compile(message, *keys)
                                        : codec::Plan::compile(message);
    if (!err) return {std::move(plan), x::errors::NIL};
    return {
        codec::Plan{},
        x::errors::Error(
            x::errors::INTERNAL,
            "the Core accepted message " + message.name +
                ", but the Driver cannot compile it (" + err.data +
                "). Please report this as a bug"
        ),
    };
}

std::unique_ptr<codec::framing::Framer> create_framer(
    const x::json::Parser &parser,
    const std::optional<::synnax::bus::Framing> &framing
) {
    if (!framing.has_value()) return nullptr;
    auto [framer, err] = codec::framing::create(*framing);
    if (err) parser.field_err("framing", err);
    return std::move(framer);
}

ChannelMap to_map(const std::vector<synnax::channel::Channel> &channels) {
    ChannelMap map;
    for (const auto &ch: channels)
        map[ch.key] = ch;
    return map;
}

std::pair<synnax::icd::ICD, x::errors::Error> retrieve_icd(
    const synnax::Synnax &client,
    const x::json::Parser &parser,
    const std::optional<synnax::icd::Key> &key
) {
    if (!key.has_value()) {
        parser.field_err("icd", "select an ICD");
        return {{}, x::errors::NIL};
    }
    auto [ic, err] = client.icds.retrieve(*key);
    if (err.matches(x::errors::NOT_FOUND)) {
        parser.field_err("icd", "ICD " + key->to_string() + " does not exist");
        return {{}, x::errors::NIL};
    }
    return {std::move(ic), err};
}

/// @returns the name of the device with the given key. Binds an error to parser when
/// the device does not exist.
std::pair<std::string, x::errors::Error> retrieve_device(
    const synnax::Synnax &client,
    const x::json::Parser &parser,
    const std::string &key
) {
    auto [dev, err] = client.devices.retrieve(key);
    if (err.matches(x::errors::NOT_FOUND)) {
        parser.field_err("device", "device " + key + " does not exist");
        return {"", x::errors::NIL};
    }
    return {std::move(dev.name), err};
}

/// @brief retrieves the channels with the given keys, skipping zeros and keys with no
/// channel.
std::pair<std::vector<synnax::channel::Channel>, x::errors::Error> retrieve_channels(
    const synnax::Synnax &client,
    const std::set<synnax::channel::Key> &keys
) {
    std::vector<synnax::channel::Key> nonzero;
    for (const auto key: keys)
        if (key != 0) nonzero.push_back(key);
    if (nonzero.empty()) return {{}, x::errors::NIL};
    auto [channels, err] = client.channels.retrieve(nonzero);
    if (!err.matches(x::errors::NOT_FOUND)) return {std::move(channels), err};
    // The Core fails the whole batch when any key is missing, so each key is retrieved
    // alone to find the channels that exist.
    channels.clear();
    for (const auto key: nonzero) {
        auto [ch, ch_err] = client.channels.retrieve(key);
        if (ch_err.matches(x::errors::NOT_FOUND)) continue;
        if (ch_err) return {{}, ch_err};
        channels.push_back(std::move(ch));
    }
    return {std::move(channels), x::errors::NIL};
}
}

std::pair<std::vector<std::uint8_t>, x::errors::Error>
unescape(const std::string &escaped) {
    std::vector<std::uint8_t> out;
    out.reserve(escaped.size());
    const auto hex = [](const char c) -> int {
        if (c >= '0' && c <= '9') return c - '0';
        if (c >= 'a' && c <= 'f') return c - 'a' + 10;
        if (c >= 'A' && c <= 'F') return c - 'A' + 10;
        return -1;
    };
    for (std::size_t i = 0; i < escaped.size(); i++) {
        const char c = escaped[i];
        if (c != '\\') {
            out.push_back(static_cast<std::uint8_t>(c));
            continue;
        }
        if (++i == escaped.size())
            return {{}, x::errors::Error(x::errors::VALIDATION, "query ends in \\")};
        switch (escaped[i]) {
            case 'n':
                out.push_back('\n');
                break;
            case 'r':
                out.push_back('\r');
                break;
            case 't':
                out.push_back('\t');
                break;
            case '0':
                out.push_back(0);
                break;
            case '\\':
                out.push_back('\\');
                break;
            case 'x': {
                const int hi = i + 1 < escaped.size() ? hex(escaped[i + 1]) : -1;
                const int lo = i + 2 < escaped.size() ? hex(escaped[i + 2]) : -1;
                if (hi < 0 || lo < 0)
                    return {
                        {},
                        x::errors::Error(
                            x::errors::VALIDATION,
                            "\\x must be followed by two hex digits"
                        )
                    };
                out.push_back(static_cast<std::uint8_t>(hi << 4 | lo));
                i += 2;
                break;
            }
            default:
                return {
                    {},
                    x::errors::Error(
                        x::errors::VALIDATION,
                        std::string("unknown escape \\") + escaped[i]
                    )
                };
        }
    }
    return {out, x::errors::NIL};
}

std::pair<ReadConfig, x::errors::Error> ReadConfig::resolve(
    x::json::Parser &parser,
    const ::synnax::bus::ReadConfig &cfg,
    const ::synnax::bus::PollConfig &poll,
    const std::optional<Stream> &stream,
    const synnax::icd::ICD &ic,
    const std::vector<synnax::channel::Channel> &channels,
    const Medium medium
) {
    ReadConfig out;
    out.data_saving_disabled = cfg.data_saving_disabled;
    out.poll_rate = poll.rate;
    out.poll_timeout = poll.timeout;
    if (stream.has_value()) {
        out.framer = create_framer(parser, stream->framing);
        out.gap = stream->gap;
    }
    const auto chs = to_map(channels);
    if (cfg.device.empty()) parser.field_err("device", "this field is required");

    std::set<synnax::channel::Key> used;
    const auto claim = [&](const std::string &at, const synnax::channel::Key key) {
        if (!used.insert(key).second)
            parser.field_err(
                at,
                "channel " + std::to_string(key) + " is used more than once"
            );
    };

    if (cfg.raw != 0)
        if (const auto *raw = find_channel(parser, "raw", chs, cfg.raw)) {
            if (raw->data_type != x::telem::BYTES_T)
                parser.field_err(
                    "raw",
                    "channel " + raw->name + " must have data type bytes"
                );
            claim("raw", raw->key);
            out.raw = raw->key;
        }

    for (std::size_t i = 0; i < cfg.messages.size(); i++) {
        const auto &m = cfg.messages[i];
        if (m.disabled) continue;
        const auto *entry = find_message(ic, m.message);
        if (entry == nullptr) {
            parser.field_err(
                path(i, "message"),
                "message " + m.message.to_string() + " is not in ICD " + ic.name
            );
            continue;
        }
        check_medium(parser, i, *entry, medium);
        if (medium == Medium::CAN && entry->query.has_value())
            parser.field_err(
                path(i, "message"),
                "message " + entry->name + " has a query, which a CAN bus cannot send"
            );
        ReadMessage msg{.entry = *entry};
        if (entry->query.has_value())
            msg.period = poll.rate.period();
        else if (entry->period.has_value())
            msg.period = *entry->period;
        if (m.fields.empty())
            parser.field_err(path(i, "fields"), "at least one field is required");
        const auto bases = field_bases(*entry);
        std::vector<synnax::icd::FieldKey> keys;
        std::map<synnax::channel::Key, std::size_t> groups;
        for (std::size_t j = 0; j < m.fields.size(); j++) {
            const auto &f = m.fields[j];
            const auto field = find_field(bases, f.field);
            if (!field) {
                parser.field_err(
                    field_path(i, j),
                    "field " + f.field.to_string() + " is not in message " + entry->name
                );
                continue;
            }
            const auto &name = bases[*field]->name;
            if (std::ranges::find(keys, f.field) != keys.end()) {
                parser.field_err(field_path(i, j), "field " + name + " is read twice");
                continue;
            }
            keys.push_back(f.field);
            const auto field_at = path(i, "fields." + std::to_string(j));
            if (f.index == 0)
                msg.groups.push_back({});
            else if (
                const auto [it, added] = groups.emplace(f.index, msg.groups.size());
                added
            ) {
                msg.groups.push_back({.index = f.index});
                const auto at = field_at + ".index";
                if (const auto *idx = find_channel(parser, at, chs, f.index)) {
                    if (!idx->is_index)
                        parser.field_err(
                            at,
                            "channel " + idx->name + " is not an index channel"
                        );
                    claim(at, idx->key);
                }
            }
            auto &group = f.index == 0 ? msg.groups.back()
                                       : msg.groups[groups.at(f.index)];
            group.slots.push_back(keys.size() - 1);
            const auto at = field_at + ".channel";
            const auto *ch = find_channel(parser, at, chs, f.channel);
            if (ch == nullptr) continue;
            check_numeric(parser, at, *ch, true);
            if (ch->index != f.index)
                parser.field_err(
                    at,
                    "channel " + ch->name + " has index " + std::to_string(ch->index) +
                        ", but field " + name + " is stamped on " +
                        std::to_string(f.index)
                );
            claim(at, ch->key);
            msg.channels.push_back(*ch);
        }
        if (!parser.ok()) continue;
        auto [plan, plan_err] = compile_plan(*entry, keys);
        if (plan_err) return {std::move(out), plan_err};
        msg.plan = std::move(plan);
        if (entry->query.has_value()) {
            auto query = std::vector<std::uint8_t>(
                entry->query->begin(),
                entry->query->end()
            );
            if (std::holds_alternative<synnax::icd::BinaryPayload>(entry->payload)) {
                auto [bytes, err] = unescape(*entry->query);
                if (err) parser.field_err(path(i, "message"), err);
                query = std::move(bytes);
            }
            if (out.framer != nullptr) {
                std::vector<std::uint8_t> wire;
                if (const auto err = out.framer->encode(query, wire))
                    parser.field_err(
                        path(i, "message"),
                        "query of " + entry->name +
                            " does not fit the framing: " + err.data
                    );
                query = std::move(wire);
            }
            msg.query = std::move(query);
            auto [reply, reply_err] = codec::Matcher::compile(std::span(&msg.entry, 1));
            if (reply_err) parser.field_err(path(i, "message"), reply_err);
            msg.reply = std::move(reply);
            out.polled.push_back(out.messages.size());
        } else
            out.streamed.push_back(out.messages.size());
        out.messages.push_back(std::move(msg));
    }
    if (cfg.messages.empty() || (out.messages.empty() && parser.ok()))
        parser.field_err("messages", "at least one enabled message is required");
    if (!out.polled.empty()) {
        if (poll.rate <= x::telem::Rate(0))
            parser.field_err("rate", "must be greater than 0");
        if (poll.timeout <= x::telem::TimeSpan(0))
            parser.field_err("timeout", "must be greater than 0");
    }
    if (!parser.ok()) return {std::move(out), x::errors::NIL};

    std::vector<synnax::icd::MessageEntry> streamed;
    for (const auto i: out.streamed)
        streamed.push_back(out.messages[i].entry);
    auto [matcher, err] = codec::Matcher::compile(streamed);
    if (err) parser.field_err("messages", err);
    out.matcher = std::move(matcher);
    return {std::move(out), x::errors::NIL};
}

std::pair<ReadConfig, x::errors::Error> ReadConfig::parse(
    const synnax::Synnax &client,
    x::json::Parser &parser,
    const ::synnax::bus::ReadConfig &cfg,
    const ::synnax::bus::PollConfig &poll,
    const std::optional<Stream> &stream,
    const Medium medium
) {
    auto [ic, ic_err] = retrieve_icd(client, parser, cfg.icd);
    if (ic_err) return {ReadConfig{}, ic_err};
    std::set<synnax::channel::Key> keys{cfg.raw};
    for (const auto &m: cfg.messages) {
        if (m.disabled) continue;
        for (const auto &f: m.fields) {
            keys.insert(f.index);
            keys.insert(f.channel);
        }
    }
    auto [channels, ch_err] = retrieve_channels(client, keys);
    if (ch_err) return {ReadConfig{}, ch_err};
    auto [out, err] = resolve(parser, cfg, poll, stream, ic, channels, medium);
    if (!parser.ok()) return {std::move(out), parser.error()};
    if (err) return {std::move(out), err};
    std::tie(out.device, err) = retrieve_device(client, parser, cfg.device);
    if (!parser.ok()) return {std::move(out), parser.error()};
    return {std::move(out), err};
}

std::pair<WriteConfig, x::errors::Error> WriteConfig::resolve(
    x::json::Parser &parser,
    const ::synnax::bus::WriteConfig &cfg,
    const std::optional<::synnax::bus::Framing> &framing,
    const synnax::icd::ICD &ic,
    const std::vector<synnax::channel::Channel> &channels,
    const Medium medium
) {
    WriteConfig out;
    out.framer = create_framer(parser, framing);
    const auto chs = to_map(channels);
    if (cfg.device.empty()) parser.field_err("device", "this field is required");
    std::set<synnax::channel::Key> commands;
    for (std::size_t i = 0; i < cfg.messages.size(); i++) {
        const auto &m = cfg.messages[i];
        if (m.disabled) continue;
        const auto *entry = find_message(ic, m.message);
        if (entry == nullptr) {
            parser.field_err(
                path(i, "message"),
                "message " + m.message.to_string() + " is not in ICD " + ic.name
            );
            continue;
        }
        check_medium(parser, i, *entry, medium);
        const auto *binary = std::get_if<synnax::icd::BinaryPayload>(&entry->payload);
        if (medium == Medium::CAN && binary != nullptr &&
            !binary->identifier.has_value())
            parser.field_err(
                path(i, "message"),
                "message " + entry->name + " needs a CAN identifier to send with"
            );
        if (entry->period.has_value() && *entry->period <= x::telem::TimeSpan::ZERO())
            parser.field_err(
                path(i, "message"),
                "message " + entry->name + " has a period of " +
                    entry->period->to_string() + ", which must be positive"
            );
        if (m.fields.empty() && !entry->period.has_value())
            parser.field_err(
                path(i, "fields"),
                "message " + entry->name +
                    " has no period, so it needs at least one field"
            );
        auto [plan, plan_err] = compile_plan(*entry);
        if (plan_err) return {std::move(out), plan_err};
        WriteMessage msg{.entry = *entry, .plan = std::move(plan)};
        msg.initial = msg.plan.values();
        for (std::size_t s = 0; s < msg.plan.size(); s++)
            msg.plan.set_raw(msg.initial, s, 0);
        const auto bases = field_bases(*entry);
        std::set<std::size_t> bound;
        for (std::size_t j = 0; j < m.fields.size(); j++) {
            const auto &f = m.fields[j];
            const auto slot = find_field(bases, f.field);
            if (!slot) {
                parser.field_err(
                    field_path(i, j),
                    "field " + f.field.to_string() + " is not in message " + entry->name
                );
                continue;
            }
            if (!bound.insert(*slot).second)
                parser.field_err(
                    field_path(i, j),
                    "field " + bases[*slot]->name + " is mapped twice"
                );
            const auto at = path(i, "fields." + std::to_string(j) + ".channel");
            const auto *ch = find_channel(parser, at, chs, f.channel);
            if (ch == nullptr) continue;
            check_numeric(parser, at, *ch, false);
            msg.bindings.push_back(
                {.channel = ch->key, .slot = *slot, .field = bases[*slot]->name}
            );
            if (commands.insert(ch->key).second) out.commands.push_back(ch->key);
        }
        if (binary == nullptr)
            msg.prefix = std::get<synnax::icd::TextPayload>(entry->payload).prefix;
        else if (binary->identifier.has_value())
            if (const auto *id = std::get_if<synnax::icd::FieldIdentifier>(
                    &*binary->identifier
                ))
                if (const auto slot = find_field(bases, id->field))
                    msg.plan.set_raw(
                        msg.initial,
                        *slot,
                        static_cast<std::int64_t>(id->value)
                    );
        out.messages.push_back(std::move(msg));
    }
    if (cfg.messages.empty() || (out.messages.empty() && parser.ok()))
        parser.field_err("messages", "at least one enabled message is required");
    return {std::move(out), x::errors::NIL};
}

std::pair<WriteConfig, x::errors::Error> WriteConfig::parse(
    const synnax::Synnax &client,
    x::json::Parser &parser,
    const ::synnax::bus::WriteConfig &cfg,
    const std::optional<::synnax::bus::Framing> &framing,
    const Medium medium
) {
    auto [ic, ic_err] = retrieve_icd(client, parser, cfg.icd);
    if (ic_err) return {WriteConfig{}, ic_err};
    std::set<synnax::channel::Key> keys;
    for (const auto &m: cfg.messages) {
        if (m.disabled) continue;
        for (const auto &f: m.fields)
            keys.insert(f.channel);
    }
    auto [channels, ch_err] = retrieve_channels(client, keys);
    if (ch_err) return {WriteConfig{}, ch_err};
    auto [out, err] = resolve(parser, cfg, framing, ic, channels, medium);
    if (!parser.ok()) return {std::move(out), parser.error()};
    if (err) return {std::move(out), err};
    std::tie(out.device, err) = retrieve_device(client, parser, cfg.device);
    if (!parser.ok()) return {std::move(out), parser.error()};
    return {std::move(out), err};
}
}
