// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>
#include <unordered_map>
#include <utility>
#include <variant>

#include "driver/bus/config.h"
#include "driver/codec/arinc429.h"

namespace driver::bus {
namespace {
using ChannelMap = std::unordered_map<synnax::channel::Key, synnax::channel::Channel>;

const synnax::library::BaseField &base(const synnax::library::Field &field) {
    return std::visit(
        [](const auto &f) -> const synnax::library::BaseField & { return f; },
        field
    );
}

const synnax::library::MessageEntry *find_message(
    const synnax::library::Library &lib,
    const synnax::library::EntryKey &key
) {
    for (const auto &entry: lib.entries)
        if (const auto *m = std::get_if<synnax::library::MessageEntry>(&entry);
            m != nullptr && m->key == key)
            return m;
    return nullptr;
}

std::optional<std::size_t> find_field(
    const synnax::library::MessageEntry &message,
    const synnax::library::FieldKey &key
) {
    for (std::size_t i = 0; i < message.fields.size(); i++)
        if (base(message.fields[i]).key == key) return i;
    return std::nullopt;
}

std::string path(const std::size_t message, const std::string &rest) {
    return "messages." + std::to_string(message) + "." + rest;
}

std::string field_path(const std::size_t message, const std::size_t field) {
    return path(message, "fields." + std::to_string(field) + ".field");
}

/// @returns what the medium is, for errors.
std::string describe(const Medium medium) {
    switch (medium) {
        case Medium::BYTES:
            return "a byte stream";
        case Medium::CAN:
            return "a CAN bus";
        case Medium::ARINC429:
            return "an ARINC 429 channel";
    }
    std::unreachable();
}

/// @returns true when the medium can carry messages with the identifier.
bool carries(const Medium medium, const synnax::library::Identifier &id) {
    switch (medium) {
        case Medium::BYTES:
            return std::holds_alternative<synnax::library::FieldIdentifier>(id) ||
                   std::holds_alternative<synnax::library::TokenIdentifier>(id);
        case Medium::CAN:
            return std::holds_alternative<synnax::library::CanIdentifier>(id);
        case Medium::ARINC429:
            return std::holds_alternative<synnax::library::Arinc429Identifier>(id);
    }
    std::unreachable();
}

/// @brief binds an error when the medium cannot carry the message: its identifier,
/// or, on ARINC 429, its layout.
void check_identifier(
    const x::json::Parser &parser,
    const std::size_t index,
    const synnax::library::MessageEntry &message,
    const Medium medium
) {
    if (medium == Medium::ARINC429) {
        if (const auto err = codec::arinc429::validate(message))
            parser.field_err(path(index, "message"), err);
        return;
    }
    if (!message.identifier.has_value()) return;
    const auto &id = *message.identifier;
    if (carries(medium, id)) return;
    const auto type = std::visit([](const auto &i) { return i.type; }, id);
    parser.field_err(
        path(index, "message"),
        "message " + message.name + " has a " + type + " identifier, which " +
            describe(medium) + " cannot carry"
    );
}

/// @brief binds an error when the channel cannot hold decoded or commanded values.
void check_numeric(
    const x::json::Parser &parser,
    const std::string &at,
    const synnax::channel::Channel &ch
) {
    if (ch.data_type.is_variable() || ch.data_type == x::telem::UUID_T)
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

std::pair<synnax::library::Library, x::errors::Error> retrieve_library(
    const synnax::Synnax &client,
    const x::json::Parser &parser,
    const synnax::library::Key &key
) {
    if (key.is_nil()) {
        parser.field_err("library", "this field is required");
        return {{}, x::errors::NIL};
    }
    auto [lib, err] = client.libraries.retrieve(key);
    if (err.matches(x::errors::NOT_FOUND)) {
        parser.field_err("library", "library " + key.to_string() + " does not exist");
        return {{}, x::errors::NIL};
    }
    return {std::move(lib), err};
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

ReadConfig ReadConfig::resolve(
    x::json::Parser &parser,
    const ::synnax::bus::ReadConfig &cfg,
    const ::synnax::bus::PollConfig &poll,
    const std::optional<::synnax::bus::Framing> &framing,
    const synnax::library::Library &library,
    const std::vector<synnax::channel::Channel> &channels,
    const Medium medium
) {
    ReadConfig out;
    out.data_saving_disabled = cfg.data_saving_disabled;
    out.poll_rate = poll.rate;
    out.poll_timeout = poll.timeout;
    out.framer = create_framer(parser, framing);
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
        const auto *entry = find_message(library, m.message);
        if (entry == nullptr) {
            parser.field_err(
                path(i, "message"),
                "message " + m.message.to_string() + " is not in library " +
                    library.name
            );
            continue;
        }
        check_identifier(parser, i, *entry, medium);
        if (medium != Medium::BYTES && entry->query.has_value())
            parser.field_err(
                path(i, "message"),
                "message " + entry->name + " has a query, which " + describe(medium) +
                    " cannot send"
            );
        ReadMessage msg{.entry = *entry, .index = m.index};
        if (m.index != 0)
            if (const auto
                    *idx = find_channel(parser, path(i, "index"), chs, m.index)) {
                if (!idx->is_index)
                    parser.field_err(
                        path(i, "index"),
                        "channel " + idx->name + " is not an index channel"
                    );
                claim(path(i, "index"), idx->key);
            }
        if (m.fields.empty())
            parser.field_err(path(i, "fields"), "at least one field is required");
        std::vector<synnax::library::FieldKey> keys;
        for (std::size_t j = 0; j < m.fields.size(); j++) {
            const auto &f = m.fields[j];
            if (!find_field(*entry, f.field)) {
                parser.field_err(
                    field_path(i, j),
                    "field " + f.field.to_string() + " is not in message " + entry->name
                );
                continue;
            }
            keys.push_back(f.field);
            const auto at = path(i, "fields." + std::to_string(j) + ".channel");
            const auto *ch = find_channel(parser, at, chs, f.channel);
            if (ch == nullptr) continue;
            check_numeric(parser, at, *ch);
            if (ch->index != m.index)
                parser.field_err(
                    at,
                    "channel " + ch->name + " has index " + std::to_string(ch->index) +
                        ", but message " + entry->name + " is stamped on " +
                        std::to_string(m.index)
                );
            claim(at, ch->key);
            msg.channels.push_back(*ch);
        }
        if (!parser.ok()) continue;
        auto [plan, plan_err] = codec::Plan::compile(*entry, keys);
        if (plan_err) {
            parser.field_err(path(i, "message"), plan_err);
            continue;
        }
        msg.plan = std::move(plan);
        if (entry->query.has_value()) {
            auto query = std::vector<std::uint8_t>(
                entry->query->begin(),
                entry->query->end()
            );
            if (entry->format != synnax::library::FORMAT_TEXT) {
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
    if (!parser.ok()) return out;

    std::vector<synnax::library::MessageEntry> streamed;
    for (const auto i: out.streamed)
        streamed.push_back(out.messages[i].entry);
    auto [matcher, err] = codec::Matcher::compile(streamed);
    if (err) parser.field_err("messages", err);
    out.matcher = std::move(matcher);
    return out;
}

std::pair<ReadConfig, x::errors::Error> ReadConfig::parse(
    const synnax::Synnax &client,
    x::json::Parser &parser,
    const ::synnax::bus::ReadConfig &cfg,
    const ::synnax::bus::PollConfig &poll,
    const std::optional<::synnax::bus::Framing> &framing,
    const Medium medium
) {
    auto [library, lib_err] = retrieve_library(client, parser, cfg.library);
    if (lib_err) return {ReadConfig{}, lib_err};
    std::set<synnax::channel::Key> keys{cfg.raw};
    for (const auto &m: cfg.messages) {
        if (m.disabled) continue;
        keys.insert(m.index);
        for (const auto &f: m.fields)
            keys.insert(f.channel);
    }
    auto [channels, ch_err] = retrieve_channels(client, keys);
    if (ch_err) return {ReadConfig{}, ch_err};
    auto out = resolve(parser, cfg, poll, framing, library, channels, medium);
    if (!parser.ok()) return {std::move(out), parser.error()};
    return {std::move(out), x::errors::NIL};
}

WriteConfig WriteConfig::resolve(
    x::json::Parser &parser,
    const ::synnax::bus::WriteConfig &cfg,
    const std::optional<::synnax::bus::Framing> &framing,
    const synnax::library::Library &library,
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
        const auto *entry = find_message(library, m.message);
        if (entry == nullptr) {
            parser.field_err(
                path(i, "message"),
                "message " + m.message.to_string() + " is not in library " +
                    library.name
            );
            continue;
        }
        check_identifier(parser, i, *entry, medium);
        if (medium == Medium::CAN && !entry->identifier.has_value())
            parser.field_err(
                path(i, "message"),
                "message " + entry->name + " needs a CAN identifier to send with"
            );
        if (m.fields.empty() && !entry->period.has_value())
            parser.field_err(
                path(i, "fields"),
                "message " + entry->name +
                    " has no period, so it needs at least one field"
            );
        auto [plan, plan_err] = codec::Plan::compile(*entry);
        if (plan_err) {
            parser.field_err(path(i, "message"), plan_err);
            continue;
        }
        WriteMessage msg{.entry = *entry, .plan = std::move(plan)};
        msg.initial = msg.plan.values();
        for (std::size_t s = 0; s < msg.plan.size(); s++)
            msg.initial.set(s, std::int64_t{0});
        std::set<std::size_t> bound;
        for (std::size_t j = 0; j < m.fields.size(); j++) {
            const auto &f = m.fields[j];
            const auto slot = find_field(*entry, f.field);
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
                    "field " + base(entry->fields[*slot]).name + " is mapped twice"
                );
            const auto at = path(i, "fields." + std::to_string(j) + ".channel");
            const auto *ch = find_channel(parser, at, chs, f.channel);
            if (ch == nullptr) continue;
            check_numeric(parser, at, *ch);
            msg.bindings.push_back({.channel = ch->key, .slot = *slot});
            if (commands.insert(ch->key).second) out.commands.push_back(ch->key);
        }
        if (entry->identifier.has_value()) {
            if (const auto *id = std::get_if<synnax::library::FieldIdentifier>(
                    &*entry->identifier
                ))
                if (const auto slot = find_field(*entry, id->field))
                    msg.initial.set(*slot, static_cast<std::int64_t>(id->value));
            if (const auto *id = std::get_if<synnax::library::TokenIdentifier>(
                    &*entry->identifier
                ))
                msg.prefix = id->prefix;
        }
        out.messages.push_back(std::move(msg));
    }
    if (cfg.messages.empty() || (out.messages.empty() && parser.ok()))
        parser.field_err("messages", "at least one enabled message is required");
    return out;
}

std::pair<WriteConfig, x::errors::Error> WriteConfig::parse(
    const synnax::Synnax &client,
    x::json::Parser &parser,
    const ::synnax::bus::WriteConfig &cfg,
    const std::optional<::synnax::bus::Framing> &framing,
    const Medium medium
) {
    auto [library, lib_err] = retrieve_library(client, parser, cfg.library);
    if (lib_err) return {WriteConfig{}, lib_err};
    std::set<synnax::channel::Key> keys;
    for (const auto &m: cfg.messages) {
        if (m.disabled) continue;
        for (const auto &f: m.fields)
            keys.insert(f.channel);
    }
    auto [channels, ch_err] = retrieve_channels(client, keys);
    if (ch_err) return {WriteConfig{}, ch_err};
    auto out = resolve(parser, cfg, framing, library, channels, medium);
    if (!parser.ok()) return {std::move(out), parser.error()};
    return {std::move(out), x::errors::NIL};
}
}
