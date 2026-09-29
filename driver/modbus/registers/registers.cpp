// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/codec/bits.h"
#include "driver/modbus/registers/registers.h"

namespace driver::modbus::registers {
namespace library = synnax::library;

std::pair<library::BinaryField, x::errors::Error> field(
    const x::telem::DataType &data_type,
    const bool bytes_swapped,
    const bool words_swapped,
    const std::size_t offset
) {
    library::BinaryField f;
    f.signed_ = data_type.matches(
        {x::telem::INT8_T, x::telem::INT16_T, x::telem::INT32_T, x::telem::INT64_T}
    );
    f.float_ = data_type.matches({x::telem::FLOAT32_T, x::telem::FLOAT64_T});
    if (!f.signed_ && !f.float_ &&
        !data_type.matches(
            {x::telem::UINT8_T,
             x::telem::UINT16_T,
             x::telem::UINT32_T,
             x::telem::UINT64_T}
        ))
        return {
            {},
            x::errors::Error(
                x::errors::VALIDATION,
                "unsupported data type: " + data_type.name()
            ),
        };
    f.bit_length = static_cast<std::uint8_t>(data_type.density() * 8);
    const auto byte = offset * 2;
    if (f.bit_length == 8) {
        f.start_bit = static_cast<std::uint16_t>((byte + (bytes_swapped ? 0 : 1)) * 8);
        return {f, x::errors::NIL};
    }
    // Big-endian orders the words high first and little-endian low first, while the
    // registers hold the low word first unless words_swapped.
    const bool reversed = bytes_swapped == words_swapped;
    if (bytes_swapped) {
        f.byte_order = reversed ? codec::BYTE_ORDER_LITTLE_ENDIAN_WORD_SWAPPED
                                : library::BYTE_ORDER_LITTLE_ENDIAN;
        f.start_bit = static_cast<std::uint16_t>(byte * 8);
    } else {
        f.byte_order = reversed ? codec::BYTE_ORDER_BIG_ENDIAN_WORD_SWAPPED
                                : library::BYTE_ORDER_BIG_ENDIAN;
        f.start_bit = static_cast<std::uint16_t>(byte * 8 + 7);
    }
    return {f, x::errors::NIL};
}

std::pair<library::BinaryField, x::errors::Error> whole_field(
    const x::telem::DataType &data_type,
    const bool bytes_swapped,
    const bool words_swapped,
    const std::size_t offset
) {
    auto widened = data_type;
    if (data_type == x::telem::INT8_T) widened = x::telem::INT16_T;
    if (data_type == x::telem::UINT8_T) widened = x::telem::UINT16_T;
    return field(widened, bytes_swapped, words_swapped, offset);
}

void to_bytes(
    const std::span<const std::uint16_t> registers,
    const std::span<std::uint8_t> bytes
) {
    for (std::size_t i = 0; i < registers.size(); i++) {
        bytes[2 * i] = static_cast<std::uint8_t>(registers[i] >> 8);
        bytes[2 * i + 1] = static_cast<std::uint8_t>(registers[i]);
    }
}

void to_registers(
    const std::span<const std::uint8_t> bytes,
    const std::span<std::uint16_t> registers
) {
    for (std::size_t i = 0; i < registers.size(); i++)
        registers[i] = static_cast<std::uint16_t>(bytes[2 * i] << 8 | bytes[2 * i + 1]);
}
}
