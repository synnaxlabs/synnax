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
#include <optional>
#include <utility>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"

namespace driver::codec {
/// @brief BitRange reads and writes the raw bits of one binary field in a payload.
class BitRange {
    /// @brief Segment is the part of the field that lies in one payload byte.
    struct Segment {
        std::size_t byte = 0;
        /// @brief shift is the position of the lowest bit of the segment in its byte.
        std::uint8_t shift = 0;
        /// @brief mask selects the bits of the segment after the shift.
        std::uint8_t mask = 0;
        /// @brief dest is the position of the lowest bit of the segment in the raw
        /// value.
        std::uint8_t dest = 0;

        bool operator==(const Segment &) const = default;
    };

    /// @brief segments holds one entry for each byte, as a 64-bit field can span 9.
    std::array<Segment, 9> segments{};
    std::size_t count = 0;
    std::uint8_t length = 0;
    std::size_t end_byte = 0;

public:
    /// @brief creates the bit range of a field from its start bit, bit length, and byte
    /// order. The start bit is numbered as in a DBC file.
    /// @returns CONFIG_ERROR when the bit length is not from 1 to 64, or the byte order
    /// is unknown.
    static std::pair<BitRange, x::errors::Error>
    create(const synnax::icd::BinaryBaseField &field);

    /// @brief returns the raw bits of the field. payload must hold at least end()
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

    /// @brief returns the raw bits of the field as a two's complement integer of its
    /// bit length. payload must hold at least end() bytes.
    [[nodiscard]] std::int64_t read_signed(const std::uint8_t *payload) const {
        const auto shift = 64 - this->length;
        return static_cast<std::int64_t>(this->read(payload) << shift) >> shift;
    }

    /// @brief writes the low bits of raw into the field and leaves every other bit of
    /// payload as it was. payload must hold at least end() bytes.
    void write(std::uint8_t *payload, const std::uint64_t raw) const {
        for (std::size_t i = 0; i < this->count; i++) {
            const auto &s = this->segments[i];
            const auto bits = static_cast<std::uint8_t>((raw >> s.dest) & s.mask);
            const auto kept = payload[s.byte] & ~(s.mask << s.shift);
            payload[s.byte] = static_cast<std::uint8_t>(kept | (bits << s.shift));
        }
    }

    /// @brief returns the number of payload bytes that the field needs.
    [[nodiscard]] std::size_t end() const { return this->end_byte; }

    /// @brief returns the number of bits in the field.
    [[nodiscard]] std::uint8_t width() const { return this->length; }

    /// @brief reports whether both ranges read the same bits in the same order.
    bool operator==(const BitRange &) const = default;
};

/// @brief Encoding is how the raw bits of a binary field hold its value.
enum class Encoding : std::uint8_t { UNSIGNED, SIGNED, FLOAT32, FLOAT64 };

/// @brief returns how the raw bits of f hold its value.
/// @returns CONFIG_ERROR when f is a float field that is not 32 or 64 bits.
std::pair<Encoding, x::errors::Error> encoding_of(const synnax::icd::BinaryField &f);

/// @brief returns the raw bits of the default of f, or nothing when f has none.
/// @returns CONFIG_ERROR when f is a float field that is not 32 or 64 bits, or its
/// default overflows float32 or is out of range for its bits.
std::pair<std::optional<std::uint64_t>, x::errors::Error>
default_raw(const synnax::icd::BinaryField &f);
}
