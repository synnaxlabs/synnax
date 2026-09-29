// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <string>

#include "driver/codec/bits.h"
#include "driver/codec/errors.h"
#include "driver/codec/mil1553.h"

namespace driver::codec::mil1553 {
namespace {
namespace library = synnax::library;

x::errors::Error error(const library::MessageEntry &m, const std::string &msg) {
    return x::errors::Error(LAYOUT_ERROR, "message " + m.name + ": " + msg);
}

bool bit(const std::uint16_t word, const int n) {
    return (word >> n & 1) != 0;
}
}

Command Command::from(const library::Mil1553Identifier &id) {
    return Command{
        .rt = id.rt,
        .transmit = id.direction == library::DIRECTION_TRANSMIT,
        .subaddress = id.subaddress,
        .count = id.word_count,
    };
}

Command Command::decode(const std::uint16_t word) {
    const auto count = static_cast<std::uint8_t>(word & 0x1F);
    return Command{
        .rt = static_cast<std::uint8_t>(word >> 11 & 0x1F),
        .transmit = bit(word, 10),
        .subaddress = static_cast<std::uint8_t>(word >> 5 & 0x1F),
        .count = count == 0 ? static_cast<std::uint8_t>(MAX_WORDS) : count,
    };
}

std::uint16_t Command::encode() const {
    return static_cast<std::uint16_t>(
        (this->rt & 0x1F) << 11 | (this->transmit ? 1 : 0) << 10 |
        (this->subaddress & 0x1F) << 5 | (this->count & 0x1F)
    );
}

Status Status::decode(const std::uint16_t word) {
    return Status{
        .rt = static_cast<std::uint8_t>(word >> 11 & 0x1F),
        .message_error = bit(word, 10),
        .instrumentation = bit(word, 9),
        .service_request = bit(word, 8),
        .broadcast_received = bit(word, 4),
        .busy = bit(word, 3),
        .subsystem_flag = bit(word, 2),
        .dynamic_bus_control = bit(word, 1),
        .terminal_flag = bit(word, 0),
    };
}

std::uint16_t Status::encode() const {
    return static_cast<std::uint16_t>(
        (this->rt & 0x1F) << 11 | this->message_error << 10 |
        this->instrumentation << 9 | this->service_request << 8 |
        this->broadcast_received << 4 | this->busy << 3 | this->subsystem_flag << 2 |
        this->dynamic_bus_control << 1 | static_cast<int>(this->terminal_flag)
    );
}

void to_payload(
    const std::span<const std::uint16_t> words,
    std::vector<std::uint8_t> &out
) {
    out.resize(words.size() * 2);
    for (std::size_t i = 0; i < words.size(); i++) {
        out[2 * i] = static_cast<std::uint8_t>(words[i] >> 8);
        out[2 * i + 1] = static_cast<std::uint8_t>(words[i]);
    }
}

std::size_t from_payload(
    const std::span<const std::uint8_t> payload,
    const std::span<std::uint16_t> words
) {
    const auto n = std::min(payload.size() / 2, words.size());
    for (std::size_t i = 0; i < n; i++)
        words[i] = static_cast<std::uint16_t>(payload[2 * i] << 8 | payload[2 * i + 1]);
    return n;
}

x::errors::Error validate(const library::MessageEntry &message) {
    if (!message.identifier.has_value())
        return error(message, "MIL-STD-1553 messages need an identifier");
    const auto *id = std::get_if<library::Mil1553Identifier>(&*message.identifier);
    if (id == nullptr)
        return error(message, "identifier is not a MIL-STD-1553 command");
    if (id->rt == BROADCAST)
        return error(message, "broadcast commands are not supported");
    if (id->rt > MAX_RT)
        return error(message, "remote terminal address must be from 0 to 30");
    if (id->subaddress == 0 || id->subaddress == 31)
        return error(message, "mode codes are not supported");
    if (id->subaddress > 31) return error(message, "subaddress must be from 1 to 30");
    if (id->direction != library::DIRECTION_RECEIVE &&
        id->direction != library::DIRECTION_TRANSMIT)
        return error(message, "unknown direction: " + id->direction);
    if (id->word_count < 1 || id->word_count > MAX_WORDS)
        return error(message, "word count must be from 1 to 32");
    if (message.format != library::FORMAT_BINARY)
        return error(message, "MIL-STD-1553 messages must be binary");
    const std::size_t size = 2 * id->word_count;
    if (message.length.has_value() && *message.length != size)
        return error(
            message,
            "length must be " + std::to_string(size) + " bytes for " +
                std::to_string(id->word_count) + " data words"
        );
    for (const auto &field: message.fields) {
        const auto *f = std::get_if<library::BinaryField>(&field);
        if (f == nullptr) return error(message, "MIL-STD-1553 fields must be binary");
        auto [bits, err] = BitRange::compile(
            f->start_bit,
            f->bit_length,
            f->byte_order
        );
        if (err) return error(message, "field " + f->name + ": " + err.data);
        if (bits.end() > size)
            return error(
                message,
                "field " + f->name + " lies beyond its " +
                    std::to_string(id->word_count) + " data words"
            );
    }
    return x::errors::NIL;
}
}
