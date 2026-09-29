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

#include "driver/codec/bits.h"

namespace driver::codec {
/// @brief Matcher maps a received frame to the message it belongs to. It matches binary
/// messages by CAN or field identifier and text messages by prefix. A binary message
/// with no identifier or a text message with an empty prefix matches every frame that
/// no other message matches.
class Matcher {
public:
    Matcher() = default;

    /// @brief compiles a matcher over messages that the Core has validated. Match
    /// results index this span.
    /// @returns CONFIG_ERROR naming both messages when two messages have the same CAN
    /// identifier and mask, header field value, or prefix, or when neither has an
    /// identifier or prefix. CONFIG_ERROR when an identifier field is a float.
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

    /// @brief matches a frame from a byte stream or datagram, or a text line. Header
    /// fields are read first, once each, in the order they first appear, then prefixes
    /// are checked from the longest to the shortest, then the message with no
    /// identifier or prefix.
    /// @returns the index of the message, or nullopt when none matches.
    [[nodiscard]] std::optional<std::size_t>
    match(std::span<const std::uint8_t> frame) const;

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

    /// @brief Header matches frames by the raw value of a header field that several
    /// messages can share.
    struct Header {
        /// @brief bits is where the field lies.
        BitRange bits;
        /// @brief signed_ is true when the raw value is two's complement.
        bool signed_ = false;
        /// @brief messages maps each raw value to the index of its message.
        std::unordered_map<std::int64_t, std::size_t> messages;
    };

    /// @brief Prefixed matches a line by its prefix.
    struct Prefixed {
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
    /// @brief headers holds each distinct header field in the order it first appears.
    std::vector<Header> headers;
    /// @brief prefixed holds text messages from the longest prefix to the shortest.
    std::vector<Prefixed> prefixed;
    /// @brief fallback is the message with no identifier or prefix.
    std::optional<std::size_t> fallback;
};
}
