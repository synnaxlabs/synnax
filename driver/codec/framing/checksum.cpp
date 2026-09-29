// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>

#include "driver/codec/framing/checksum.h"

namespace driver::codec::framing {
namespace {
constexpr std::array<std::uint16_t, 256> ccitt_table() {
    std::array<std::uint16_t, 256> t{};
    for (std::uint32_t i = 0; i < 256; i++) {
        std::uint16_t crc = static_cast<std::uint16_t>(i << 8);
        for (int b = 0; b < 8; b++)
            crc = static_cast<std::uint16_t>(
                crc & 0x8000 ? (crc << 1) ^ 0x1021 : crc << 1
            );
        t[i] = crc;
    }
    return t;
}

template<typename T>
constexpr std::array<T, 256> reflected_table(const T poly) {
    std::array<T, 256> t{};
    for (std::uint32_t i = 0; i < 256; i++) {
        T crc = static_cast<T>(i);
        for (int b = 0; b < 8; b++)
            crc = static_cast<T>(crc & 1 ? (crc >> 1) ^ poly : crc >> 1);
        t[i] = crc;
    }
    return t;
}

constexpr auto CCITT = ccitt_table();
constexpr auto MODBUS = reflected_table<std::uint16_t>(0xA001);
constexpr auto CRC32_TABLE = reflected_table<std::uint32_t>(0xEDB88320);
}

std::size_t width(const Checksum algorithm) {
    switch (algorithm) {
        case Checksum::NONE:
            return 0;
        case Checksum::CRC16_CCITT_FALSE:
        case Checksum::CRC16_MODBUS:
            return 2;
        case Checksum::CRC32:
            return 4;
    }
    return 0;
}

std::uint32_t
checksum(const Checksum algorithm, const std::span<const std::uint8_t> data) {
    switch (algorithm) {
        case Checksum::NONE:
            return 0;
        case Checksum::CRC16_CCITT_FALSE: {
            std::uint16_t crc = 0xFFFF;
            for (const auto b: data)
                crc = static_cast<std::uint16_t>((crc << 8) ^ CCITT[(crc >> 8) ^ b]);
            return crc;
        }
        case Checksum::CRC16_MODBUS: {
            std::uint16_t crc = 0xFFFF;
            for (const auto b: data)
                crc = static_cast<std::uint16_t>((crc >> 8) ^ MODBUS[(crc ^ b) & 0xFF]);
            return crc;
        }
        case Checksum::CRC32: {
            std::uint32_t crc = 0xFFFFFFFF;
            for (const auto b: data)
                crc = (crc >> 8) ^ CRC32_TABLE[(crc ^ b) & 0xFF];
            return crc ^ 0xFFFFFFFF;
        }
    }
    return 0;
}
}
