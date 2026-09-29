// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <stdexcept>
#include <string_view>

#include "driver/codec/errors.h"
#include "driver/codec/matcher.h"

namespace driver::codec {
namespace {
namespace library = synnax::library;

constexpr std::uint32_t STANDARD_ID_MASK = 0x7FF;
constexpr std::uint32_t EXTENDED_ID_MASK = 0x1FFFFFFF;
constexpr std::uint32_t EXTENDED_KEY_BIT = 1u << 31;

std::uint32_t exact_key(const std::uint32_t id, const bool extended) {
    return extended ? id | EXTENDED_KEY_BIT : id;
}

std::uint16_t transfer_key(mil1553::Command command) {
    command.count = 0;
    return command.encode();
}

x::errors::Error message_error(const library::MessageEntry &m, const std::string &msg) {
    return x::errors::Error(CONFIG_ERROR, "message " + m.name + ": " + msg);
}

x::errors::Error duplicate(
    const std::span<const library::MessageEntry> messages,
    const std::size_t first,
    const std::size_t second,
    const std::string &what
) {
    return x::errors::Error(
        CONFIG_ERROR,
        "messages " + messages[first].name + " and " + messages[second].name + " " +
            what
    );
}

const library::BinaryField &
find_field(const library::MessageEntry &m, const library::FieldKey &key) {
    for (const auto &f: m.fields)
        if (const auto &bf = std::get<library::BinaryField>(f); bf.key == key)
            return bf;
    throw std::out_of_range("message " + m.name + " has no field " + key.to_string());
}
}

std::pair<Matcher, x::errors::Error>
Matcher::compile(const std::span<const library::MessageEntry> messages) {
    Matcher m;
    for (std::size_t i = 0; i < messages.size(); i++) {
        const auto &msg = messages[i];
        if (!msg.identifier.has_value()) {
            if (m.fallback.has_value())
                return {
                    {},
                    duplicate(messages, *m.fallback, i, "both have no identifier")
                };
            m.fallback = i;
            continue;
        }
        const auto &identifier = *msg.identifier;
        if (const auto *can = std::get_if<library::CanIdentifier>(&identifier)) {
            const auto valid = can->extended ? EXTENDED_ID_MASK : STANDARD_ID_MASK;
            const auto mask = can->mask.value_or(valid) & valid;
            if (mask != valid) {
                for (const auto &other: m.masked)
                    if (other.id == (can->id & mask) && other.mask == mask &&
                        other.extended == can->extended)
                        return {
                            {},
                            duplicate(
                                messages,
                                other.message,
                                i,
                                "have the same CAN identifier and mask"
                            ),
                        };
                m.masked.push_back({
                    .id = can->id & mask,
                    .mask = mask,
                    .extended = can->extended,
                    .message = i,
                });
                continue;
            }
            const auto [it, ok] = m.exact.emplace(exact_key(can->id, can->extended), i);
            if (!ok)
                return {
                    {},
                    duplicate(messages, it->second, i, "have the same CAN identifier")
                };
            continue;
        }
        if (const auto *field = std::get_if<library::FieldIdentifier>(&identifier)) {
            const auto &bf = find_field(msg, field->field);
            if (bf.float_)
                return {{}, message_error(msg, "identifier field must be an integer")};
            auto [bits, err] = BitRange::compile(
                bf.start_bit,
                bf.bit_length,
                bf.byte_order
            );
            if (err) return {{}, message_error(msg, err.data)};
            auto h = std::find_if(
                m.headers.begin(),
                m.headers.end(),
                [&](const Header &h) {
                    return h.bits == bits && h.signed_ == bf.signed_;
                }
            );
            if (h == m.headers.end())
                h = m.headers.insert(
                    m.headers.end(),
                    Header{.bits = bits, .signed_ = bf.signed_}
                );
            const auto [it, ok] = h->messages.emplace(field->value, i);
            if (!ok)
                return {
                    {},
                    duplicate(
                        messages,
                        it->second,
                        i,
                        "have the same identifier value " + std::to_string(field->value)
                    ),
                };
            continue;
        }
        if (const auto *token = std::get_if<library::TokenIdentifier>(&identifier)) {
            for (const auto &t: m.tokens)
                if (t.prefix == token->prefix)
                    return {
                        {},
                        duplicate(
                            messages,
                            t.message,
                            i,
                            "have the same token " + t.prefix
                        ),
                    };
            m.tokens.push_back({.prefix = token->prefix, .message = i});
            continue;
        }
        if (const auto *c = std::get_if<library::Mil1553Identifier>(&identifier)) {
            if (c->rt > mil1553::MAX_RT)
                return {{}, message_error(msg, "remote terminal address out of range")};
            if (c->subaddress == 0 || c->subaddress > mil1553::MAX_SUBADDRESS)
                return {{}, message_error(msg, "subaddress must be from 1 to 30")};
            if (c->direction != library::DIRECTION_RECEIVE &&
                c->direction != library::DIRECTION_TRANSMIT)
                return {{}, message_error(msg, "unknown direction " + c->direction)};
            const auto [it, ok] = m.transfers.emplace(
                transfer_key(mil1553::Command::from(*c)),
                i
            );
            if (!ok)
                return {
                    {},
                    duplicate(
                        messages,
                        it->second,
                        i,
                        "have the same MIL-STD-1553 address and subaddress"
                    ),
                };
            continue;
        }
        return {{}, message_error(msg, "identifier type is not supported")};
    }
    std::stable_sort(
        m.tokens.begin(),
        m.tokens.end(),
        [](const auto &a, const auto &b) { return a.prefix.size() > b.prefix.size(); }
    );
    return {std::move(m), x::errors::NIL};
}

std::optional<std::size_t>
Matcher::match(const std::uint32_t id, const bool extended) const {
    if (const auto it = this->exact.find(exact_key(id, extended));
        it != this->exact.end())
        return it->second;
    for (const auto &m: this->masked)
        if (m.extended == extended && (id & m.mask) == m.id) return m.message;
    return this->fallback;
}

std::optional<std::size_t>
Matcher::match(const std::span<const std::uint8_t> frame) const {
    for (const auto &h: this->headers) {
        if (frame.size() < h.bits.end()) continue;
        const auto raw = h.signed_
                           ? h.bits.read_signed(frame.data())
                           : static_cast<std::int64_t>(h.bits.read(frame.data()));
        if (const auto it = h.messages.find(raw); it != h.messages.end())
            return it->second;
    }
    const std::string_view line(
        reinterpret_cast<const char *>(frame.data()),
        frame.size()
    );
    for (const auto &t: this->tokens)
        if (line.starts_with(t.prefix)) return t.message;
    return this->fallback;
}

std::optional<std::size_t> Matcher::match(const mil1553::Command &command) const {
    if (const auto it = this->transfers.find(transfer_key(command));
        it != this->transfers.end())
        return it->second;
    return this->fallback;
}
}
