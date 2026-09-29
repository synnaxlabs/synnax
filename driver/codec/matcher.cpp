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
}
