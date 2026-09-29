// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "driver/codec/arinc429.h"
#include "driver/codec/errors.h"

namespace driver::codec::arinc429 {
namespace {
namespace library = synnax::library;

/// @brief the first zero-based bit a data field may use: bit 11 of the standard.
constexpr std::size_t DATA_START = 10;
/// @brief the first zero-based bit a field may use when bits 9 and 10 carry data.
constexpr std::size_t SDI_START = 8;
/// @brief the last zero-based bit a field may use: bit 31 of the standard.
constexpr std::size_t DATA_END = 30;

x::errors::Error error(const library::MessageEntry &m, const std::string &msg) {
    return x::errors::Error(CONFIG_ERROR, "message " + m.name + ": " + msg);
}
}

Word Word::pack(
    const std::uint8_t label,
    const std::uint8_t sdi,
    const bool sdi_matched,
    const std::span<const std::uint8_t, PAYLOAD_SIZE> payload
) {
    auto raw = static_cast<std::uint32_t>(payload[0]) |
               static_cast<std::uint32_t>(payload[1]) << 8 |
               static_cast<std::uint32_t>(payload[2]) << 16 |
               static_cast<std::uint32_t>(payload[3]) << 24;
    raw = (raw & ~0xFFu) | reverse(label);
    if (sdi_matched) raw = (raw & ~0x300u) | static_cast<std::uint32_t>(sdi & 0x3) << 8;
    return Word(raw).with_parity();
}

x::errors::Error validate(const library::MessageEntry &message) {
    if (!message.identifier.has_value())
        return error(message, "ARINC 429 messages need an identifier");
    const auto *id = std::get_if<library::Arinc429Identifier>(&*message.identifier);
    if (id == nullptr) return error(message, "identifier is not an ARINC 429 label");
    if (id->sdi_matched && id->sdi > MAX_SDI)
        return error(message, "SDI must be from 0 to 3");
    if (message.format != library::FORMAT_BINARY)
        return error(message, "ARINC 429 messages must be binary");
    if (message.length.has_value() && *message.length != PAYLOAD_SIZE)
        return error(message, "ARINC 429 messages are 4 bytes long");
    const auto start = id->sdi_matched ? DATA_START : SDI_START;
    for (const auto &field: message.fields) {
        const auto *f = std::get_if<library::BinaryField>(&field);
        if (f == nullptr) return error(message, "ARINC 429 fields must be binary");
        if (f->byte_order != library::BYTE_ORDER_LITTLE_ENDIAN)
            return error(message, "field " + f->name + " must be little-endian");
        const std::size_t end = f->start_bit + f->bit_length - 1;
        if (f->bit_length == 0 || f->start_bit < start || end > DATA_END)
            return error(
                message,
                "field " + f->name + " must lie in bits " + std::to_string(start + 1) +
                    " to " + std::to_string(DATA_END + 1)
            );
    }
    return x::errors::NIL;
}
}
