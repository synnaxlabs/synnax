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

const library::BinaryField &find_field(
    const library::MessageEntry &m,
    const library::BinaryPayload &payload,
    const library::FieldKey &key
) {
    for (const auto &f: payload.fields)
        if (f.key == key) return f;
    throw std::out_of_range("message " + m.name + " has no field " + key.to_string());
}
}

std::pair<Matcher, x::errors::Error>
Matcher::compile(const std::span<const library::MessageEntry> messages) {
    Matcher m;
    for (std::size_t i = 0; i < messages.size(); i++) {
        const auto &msg = messages[i];
        const auto *text = std::get_if<library::TextPayload>(&msg.payload);
        const auto *binary = std::get_if<library::BinaryPayload>(&msg.payload);
        if (text != nullptr ? text->prefix.empty() : !binary->identifier.has_value()) {
            if (m.fallback.has_value())
                return {
                    {},
                    duplicate(messages, *m.fallback, i, "both have no identifier")
                };
            m.fallback = i;
            continue;
        }
        if (text != nullptr) {
            for (const auto &t: m.prefixed)
                if (t.prefix == text->prefix)
                    return {
                        {},
                        duplicate(
                            messages,
                            t.message,
                            i,
                            "have the same prefix " + t.prefix
                        ),
                    };
            m.prefixed.push_back({.prefix = text->prefix, .message = i});
            continue;
        }
        const auto &identifier = *binary->identifier;
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
        const auto &field = std::get<library::FieldIdentifier>(identifier);
        const auto &bf = find_field(msg, *binary, field.field);
        if (bf.float_)
            return {{}, message_error(msg, "identifier field must be an integer")};
        auto [bits, err] = BitRange::compile(
            bf.start_bit,
            bf.bit_length,
            bf.byte_order
        );
        if (err) return {{}, message_error(msg, err.data)};
        auto h = std::find_if(m.headers.begin(), m.headers.end(), [&](const Header &h) {
            return h.bits == bits && h.signed_ == bf.signed_;
        });
        if (h == m.headers.end())
            h = m.headers.insert(
                m.headers.end(),
                Header{.bits = bits, .signed_ = bf.signed_}
            );
        const auto [it, ok] = h->messages.emplace(field.value, i);
        if (!ok)
            return {
                {},
                duplicate(
                    messages,
                    it->second,
                    i,
                    "have the same identifier value " + std::to_string(field.value)
                ),
            };
    }
    std::stable_sort(
        m.prefixed.begin(),
        m.prefixed.end(),
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
    for (const auto &t: this->prefixed)
        if (line.starts_with(t.prefix)) return t.message;
    return this->fallback;
}
}
