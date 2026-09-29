// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <bit>
#include <cstddef>
#include <cstdint>
#include <span>

#include "client/cpp/library/types.gen.h"
#include "x/cpp/errors/errors.h"

namespace driver::codec::arinc429 {
/// @brief the number of bytes in the payload of an ARINC 429 message.
constexpr std::size_t PAYLOAD_SIZE = 4;
/// @brief the largest SDI value.
constexpr std::uint8_t MAX_SDI = 3;

/// @returns the label with its bit order reversed. Bit 1 of an ARINC 429 word holds
/// the most significant bit of the label, so the low byte of a word holds the label
/// reversed.
constexpr std::uint8_t reverse(std::uint8_t label) {
    label = static_cast<std::uint8_t>((label & 0xF0) >> 4 | (label & 0x0F) << 4);
    label = static_cast<std::uint8_t>((label & 0xCC) >> 2 | (label & 0x33) << 2);
    return static_cast<std::uint8_t>((label & 0xAA) >> 1 | (label & 0x55) << 1);
}

/// @brief Word is one 32-bit ARINC 429 word, with bit n of the standard at bit n - 1
/// of raw. This is the order the bits arrive on the wire.
class Word {
public:
    constexpr Word() = default;

    constexpr explicit Word(const std::uint32_t raw): raw_(raw) {}

    /// @returns the word with its label and SDI set from a payload whose bits 10 to
    /// 31 hold the data, SSM, and parity bits, and with odd parity.
    /// @param label the label, in its octal value.
    /// @param sdi the SDI, written only when sdi_matched is true. Otherwise bits 9
    /// and 10 keep the payload's value.
    static Word pack(
        std::uint8_t label,
        std::uint8_t sdi,
        bool sdi_matched,
        std::span<const std::uint8_t, PAYLOAD_SIZE> payload
    );

    /// @returns the raw word.
    [[nodiscard]] constexpr std::uint32_t raw() const { return this->raw_; }

    /// @returns the label, in its octal value.
    [[nodiscard]] constexpr std::uint8_t label() const {
        return reverse(static_cast<std::uint8_t>(this->raw_ & 0xFF));
    }

    /// @returns the SDI from bits 9 and 10.
    [[nodiscard]] constexpr std::uint8_t sdi() const {
        return static_cast<std::uint8_t>(this->raw_ >> 8 & 0x3);
    }

    /// @returns the SSM from bits 30 and 31.
    [[nodiscard]] constexpr std::uint8_t ssm() const {
        return static_cast<std::uint8_t>(this->raw_ >> 29 & 0x3);
    }

    /// @returns true when the word has odd parity.
    [[nodiscard]] constexpr bool parity_valid() const {
        return std::popcount(this->raw_) % 2 == 1;
    }

    /// @returns the word with bit 32 set or cleared so that it has odd parity.
    [[nodiscard]] constexpr Word with_parity() const {
        const auto low = this->raw_ & 0x7FFFFFFF;
        const auto bit = std::popcount(low) % 2 == 0 ? 0x80000000u : 0u;
        return Word(low | bit);
    }

    /// @returns the word as a little-endian payload, so that a binary field numbers
    /// its bits as the standard does, less one.
    [[nodiscard]] constexpr std::array<std::uint8_t, PAYLOAD_SIZE> bytes() const {
        return {
            static_cast<std::uint8_t>(this->raw_),
            static_cast<std::uint8_t>(this->raw_ >> 8),
            static_cast<std::uint8_t>(this->raw_ >> 16),
            static_cast<std::uint8_t>(this->raw_ >> 24),
        };
    }

    constexpr bool operator==(const Word &) const = default;

private:
    /// @brief raw_ is the word with bit n of the standard at bit n - 1.
    std::uint32_t raw_ = 0;
};

/// @brief checks that a message can travel as an ARINC 429 word: it has an arinc429
/// identifier with an SDI from 0 to 3, a binary format, a length of 4 bytes when set,
/// and little-endian fields that lie in bits 11 to 31. Fields may also use bits 9 and
/// 10 when the identifier does not match the SDI.
/// @returns LAYOUT_ERROR naming the message when it cannot.
x::errors::Error validate(const synnax::library::MessageEntry &message);
}
