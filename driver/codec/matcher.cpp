// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
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

std::uint16_t
label_key(const std::uint8_t label, const std::optional<std::uint8_t> sdi) {
    return static_cast<std::uint16_t>(label | (sdi.has_value() ? *sdi + 1 : 0) << 8);
}

std::uint16_t transfer_key(mil1553::Command command) {
    command.count = 0;
    return command.encode();
}

x::errors::Error message_error(const library::MessageEntry &m, const std::string &msg) {
    return x::errors::Error(LAYOUT_ERROR, "message " + m.name + ": " + msg);
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
                    message_error(msg, "more than one message has no identifier"),
                };
            m.fallback = i;
            continue;
        }
        const auto &identifier = *msg.identifier;
        if (const auto *can = std::get_if<library::CanIdentifier>(&identifier)) {
            const auto valid = can->extended ? EXTENDED_ID_MASK : STANDARD_ID_MASK;
            if ((can->id & ~valid) != 0)
                return {{}, message_error(msg, "CAN identifier out of range")};
            const auto mask = can->mask.value_or(valid) & valid;
            if (mask != valid) {
                m.masked.push_back({
                    .id = can->id & mask,
                    .mask = mask,
                    .extended = can->extended,
                    .message = i,
                });
                continue;
            }
            if (!m.exact.emplace(exact_key(can->id, can->extended), i).second)
                return {{}, message_error(msg, "duplicate CAN identifier")};
            continue;
        }
        if (const auto *field = std::get_if<library::FieldIdentifier>(&identifier)) {
            const auto it = std::find_if(
                msg.fields.begin(),
                msg.fields.end(),
                [&field](const library::Field &f) {
                    return std::visit(
                        [&field](const auto &v) { return v.key == field->field; },
                        f
                    );
                }
            );
            if (it == msg.fields.end())
                return {{}, message_error(msg, "identifier field is not in message")};
            const auto *bf = std::get_if<library::BinaryField>(&*it);
            if (bf == nullptr || bf->float_)
                return {
                    {},
                    message_error(msg, "identifier field must be a binary integer"),
                };
            auto [bits, err] = BitRange::compile(
                bf->start_bit,
                bf->bit_length,
                bf->byte_order
            );
            if (err) return {{}, message_error(msg, err.data)};
            m.fields.push_back({
                .bits = bits,
                .signed_ = bf->signed_,
                .value = field->value,
                .message = i,
            });
            continue;
        }
        if (const auto *token = std::get_if<library::TokenIdentifier>(&identifier)) {
            for (const auto &t: m.tokens)
                if (t.prefix == token->prefix)
                    return {{}, message_error(msg, "duplicate token " + t.prefix)};
            m.tokens.push_back({.prefix = token->prefix, .message = i});
            continue;
        }
        if (const auto *a = std::get_if<library::Arinc429Identifier>(&identifier)) {
            if (a->sdi_matched && a->sdi > arinc429::MAX_SDI)
                return {{}, message_error(msg, "SDI must be from 0 to 3")};
            const auto key = label_key(
                a->label,
                a->sdi_matched ? std::optional(a->sdi) : std::nullopt
            );
            if (!m.labels.emplace(key, i).second)
                return {{}, message_error(msg, "duplicate ARINC 429 label and SDI")};
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
            if (!m.transfers.emplace(transfer_key(mil1553::Command::from(*c)), i)
                     .second)
                return {
                    {},
                    message_error(msg, "duplicate MIL-STD-1553 address and subaddress"),
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
    for (const auto &f: this->fields) {
        if (frame.size() < f.bits.end()) continue;
        const auto raw = f.signed_
                           ? f.bits.read_signed(frame.data())
                           : static_cast<std::int64_t>(f.bits.read(frame.data()));
        if (raw == f.value) return f.message;
    }
    const std::string_view line(
        reinterpret_cast<const char *>(frame.data()),
        frame.size()
    );
    for (const auto &t: this->tokens)
        if (line.starts_with(t.prefix)) return t.message;
    return this->fallback;
}

std::optional<std::size_t> Matcher::match(const arinc429::Word word) const {
    if (const auto it = this->labels.find(label_key(word.label(), word.sdi()));
        it != this->labels.end())
        return it->second;
    if (const auto it = this->labels.find(label_key(word.label(), std::nullopt));
        it != this->labels.end())
        return it->second;
    return this->fallback;
}

std::optional<std::size_t> Matcher::match(const mil1553::Command &command) const {
    if (const auto it = this->transfers.find(transfer_key(command));
        it != this->transfers.end())
        return it->second;
    return this->fallback;
}
}
