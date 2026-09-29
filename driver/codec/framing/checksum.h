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

namespace driver::codec::framing {
/// @brief Checksum is a CRC algorithm that guards a frame.
enum class Checksum : std::uint8_t {
    /// @brief NONE means the frame carries no checksum.
    NONE,
    /// @brief CRC16_CCITT_FALSE is poly 0x1021, init 0xFFFF, not reflected.
    CRC16_CCITT_FALSE,
    /// @brief CRC16_MODBUS is poly 0x8005, init 0xFFFF, reflected.
    CRC16_MODBUS,
    /// @brief CRC32 is the ISO-HDLC CRC used by Ethernet and zlib.
    CRC32,
};

/// @returns the number of bytes the checksum occupies in a frame, 0 for NONE.
[[nodiscard]] std::size_t width(Checksum algorithm);

/// @returns the checksum of data, 0 for NONE.
[[nodiscard]] std::uint32_t
checksum(Checksum algorithm, std::span<const std::uint8_t> data);
}
