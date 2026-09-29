// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>
#include <optional>
#include <span>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/library/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/codec/arinc429.h"
#include "driver/codec/bits.h"
#include "driver/codec/mil1553.h"

namespace driver::codec {
/// @brief Matcher maps a received frame to the message it belongs to. It supports CAN,
/// ARINC 429, MIL-STD-1553, field, and token identifiers, and messages with no
/// identifier, which match every frame that no other message matches.
class Matcher {
public:
    Matcher() = default;

    /// @brief compiles a matcher over messages. Match results index this span.
    /// @returns LAYOUT_ERROR when an identifier is invalid or unsupported, when two
    /// messages have the same exact CAN identifier, ARINC 429 label and SDI,
    /// MIL-STD-1553 address, subaddress, and direction, or token, or when more than
    /// one message has no identifier.
    static std::pair<Matcher, x::errors::Error>
    compile(std::span<const synnax::library::MessageEntry> messages);

    /// @brief matches a CAN frame. Exact identifiers are checked first in constant
    /// time, then masked identifiers in declaration order, then the message with no
    /// identifier.
    /// @param id the arbitration identifier.
    /// @param extended true for a 29-bit identifier.
    /// @returns the index of the message, or nullopt when none matches.
    [[nodiscard]] std::optional<std::size_t>
    match(std::uint32_t id, bool extended) const;

    /// @brief matches a frame from a byte stream or datagram, or a text line. Field
    /// identifiers are checked first in declaration order, then tokens from the longest
    /// prefix to the shortest, then the message with no identifier.
    /// @returns the index of the message, or nullopt when none matches.
    [[nodiscard]] std::optional<std::size_t>
    match(std::span<const std::uint8_t> frame) const;

    /// @brief matches an ARINC 429 word. A message that matches the label and SDI is
    /// checked first, then a message that matches the label alone, then the message
    /// with no identifier.
    /// @returns the index of the message, or nullopt when none matches.
    [[nodiscard]] std::optional<std::size_t> match(arinc429::Word word) const;

    /// @brief matches a MIL-STD-1553 command by address, subaddress, and direction,
    /// then falls back to the message with no identifier. The word count does not
    /// take part, so a transfer with the wrong count still reaches its message.
    /// @returns the index of the message, or nullopt when none matches.
    [[nodiscard]] std::optional<std::size_t>
    match(const mil1553::Command &command) const;

private:
    /// @brief Masked is a CAN identifier with a partial mask.
    struct Masked {
        /// @brief id is the identifier with the mask applied.
        std::uint32_t id = 0;
        /// @brief mask selects the bits that must match.
        std::uint32_t mask = 0;
        /// @brief extended is true for a 29-bit identifier.
        bool extended = false;
        /// @brief message is the index of the message.
        std::size_t message = 0;
    };

    /// @brief Field matches a frame by the raw value of a header field.
    struct Field {
        /// @brief bits is where the field lies.
        BitRange bits;
        /// @brief signed_ is true when the raw value is two's complement.
        bool signed_ = false;
        /// @brief value is the raw value the field must hold.
        std::int64_t value = 0;
        /// @brief message is the index of the message.
        std::size_t message = 0;
    };

    /// @brief Token matches a line by its prefix.
    struct Token {
        /// @brief prefix is the text the line must start with.
        std::string prefix;
        /// @brief message is the index of the message.
        std::size_t message = 0;
    };

    /// @brief exact maps a CAN identifier, with bit 31 set when extended, to its
    /// message.
    std::unordered_map<std::uint32_t, std::size_t> exact;
    /// @brief masked holds CAN identifiers with partial masks in declaration order.
    std::vector<Masked> masked;
    /// @brief fields holds field identifiers in declaration order.
    std::vector<Field> fields;
    /// @brief labels maps an ARINC 429 label to its message. The high byte holds the
    /// SDI plus one when the identifier matches the SDI, and zero otherwise.
    std::unordered_map<std::uint16_t, std::size_t> labels;
    /// @brief transfers maps a MIL-STD-1553 command word with its count cleared to
    /// its message.
    std::unordered_map<std::uint16_t, std::size_t> transfers;
    /// @brief tokens holds token identifiers from the longest prefix to the shortest.
    std::vector<Token> tokens;
    /// @brief fallback is the message with no identifier.
    std::optional<std::size_t> fallback;
};
}
