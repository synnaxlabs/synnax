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
#include <span>
#include <utility>

#include "client/cpp/library/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

/// @brief lays out Modbus register values as codec fields over a block of registers
/// serialized as on the wire: two big-endian bytes per register, in address order.
namespace driver::modbus::registers {
/// @brief returns the field that reads a value of the data type.
/// @param offset the register offset of the value in the block.
/// @param bytes_swapped true when each register holds its bytes little-endian.
/// @param words_swapped true when the first register holds the most significant word.
/// @returns VALIDATION when the data type is not an integer or float of 8 to 64 bits.
std::pair<synnax::library::BinaryField, x::errors::Error> field(
    const x::telem::DataType &data_type,
    bool bytes_swapped,
    bool words_swapped,
    std::size_t offset
);

/// @brief returns the field that writes a value of the data type. Unlike field(), an
/// 8-bit value fills its whole register, sign- or zero-extended to 16 bits.
std::pair<synnax::library::BinaryField, x::errors::Error> whole_field(
    const x::telem::DataType &data_type,
    bool bytes_swapped,
    bool words_swapped,
    std::size_t offset
);

/// @brief serializes registers into bytes, which must hold two per register.
void to_bytes(std::span<const std::uint16_t> registers, std::span<std::uint8_t> bytes);

/// @brief parses bytes into registers, which must hold one per two bytes.
void to_registers(
    std::span<const std::uint8_t> bytes,
    std::span<std::uint16_t> registers
);
}
