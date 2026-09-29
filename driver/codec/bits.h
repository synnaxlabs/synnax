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
#include <cstddef>
#include <cstdint>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"

namespace driver::codec {
/// @brief BYTE_ORDER_BIG_ENDIAN_WORD_SWAPPED is big-endian with the field's 16-bit
/// words in reverse order (CDAB), a layout common in Modbus devices. The library schema
/// does not store it.
constexpr const char *BYTE_ORDER_BIG_ENDIAN_WORD_SWAPPED = "big_endian_word_swapped";
/// @brief BYTE_ORDER_LITTLE_ENDIAN_WORD_SWAPPED is little-endian with the field's
/// 16-bit words in reverse order (BADC). The library schema does not store it.
constexpr const char
    *BYTE_ORDER_LITTLE_ENDIAN_WORD_SWAPPED = "little_endian_word_swapped";

/// @brief BitRange is the bit positions of a field in a payload, resolved once so that
/// reads and writes are shifts and masks with no branching on byte order.
class BitRange {
public:
    BitRange() = default;

    /// @brief resolves a field's bit positions.
    /// @param start_bit the start bit numbered as in a DBC file: the least significant
    /// bit for little-endian fields and the most significant bit for big-endian ones.
    /// @param bit_length the number of bits, from 1 to 64.
    /// @param byte_order BYTE_ORDER_LITTLE_ENDIAN, BYTE_ORDER_BIG_ENDIAN, or one of
    /// their word-swapped forms, which need a byte-aligned field whose length is a
    /// multiple of 16.
    /// @returns the range, or LAYOUT_ERROR when an argument is out of bounds.
    static std::pair<BitRange, x::errors::Error> compile(
        std::uint16_t start_bit,
        std::uint8_t bit_length,
        const std::string &byte_order
    );

    /// @brief reads the raw bits of the field. The payload must hold at least end()
    /// bytes.
    [[nodiscard]] std::uint64_t read(const std::uint8_t *payload) const {
        std::uint64_t raw = 0;
        for (std::size_t i = 0; i < this->count; i++) {
            const auto &s = this->segments[i];
            const auto bits = static_cast<std::uint64_t>(payload[s.byte] >> s.shift);
            raw |= (bits & s.mask) << s.dest;
        }
        return raw;
    }

    /// @brief reads the raw bits of the field, sign-extended from its length.
    [[nodiscard]] std::int64_t read_signed(const std::uint8_t *payload) const {
        const auto shift = 64 - this->bits;
        return static_cast<std::int64_t>(this->read(payload) << shift) >> shift;
    }

    /// @brief writes the low length() bits of raw into the field, leaving every other
    /// bit of the payload as it was. The payload must hold at least end() bytes.
    void write(std::uint8_t *payload, const std::uint64_t raw) const {
        for (std::size_t i = 0; i < this->count; i++) {
            const auto &s = this->segments[i];
            const auto bits = static_cast<std::uint8_t>((raw >> s.dest) & s.mask);
            const auto cleared = payload[s.byte] & ~(s.mask << s.shift);
            payload[s.byte] = static_cast<std::uint8_t>(cleared | (bits << s.shift));
        }
    }

    /// @returns the number of payload bytes the field needs.
    [[nodiscard]] std::size_t end() const { return this->end_; }

    /// @returns the number of bits in the field.
    [[nodiscard]] std::uint8_t length() const { return this->bits; }

private:
    /// @brief Segment is the part of the field that lies in one payload byte.
    struct Segment {
        /// @brief byte is the index of the payload byte.
        std::size_t byte = 0;
        /// @brief shift is the position of the segment's lowest bit in the byte.
        std::uint8_t shift = 0;
        /// @brief mask selects the segment's bits after the shift.
        std::uint8_t mask = 0;
        /// @brief dest is the position of the segment's lowest bit in the raw value.
        std::uint8_t dest = 0;
    };

    /// @brief segments holds one entry per byte, as a 64-bit field spans 9 bytes.
    std::array<Segment, 9> segments{};
    /// @brief count is the number of used segments.
    std::size_t count = 0;
    /// @brief bits is the field length in bits.
    std::uint8_t bits = 0;
    /// @brief end_ is the number of payload bytes the field needs.
    std::size_t end_ = 0;
};
}
