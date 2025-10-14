// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

namespace util {
/// @brief parses data from a modbus register into a compatible telem::SampleValue.
/// @param data buffer of register values to parse from. Must be at least as large as
/// the density of the data type (dt.density()).
/// @param dt the data type to parse from the buffer.
/// @param swap_bytes whether to swap the byte order of the data.
/// @param swap_words whether to swap the word order of the data.
inline std::pair<telem::SampleValue, xerrors::Error> parse_register_value(
    const uint16_t *data,
    const telem::DataType &dt,
    const bool swap_bytes,
    const bool swap_words
) {
    if (data == nullptr) throw std::invalid_argument("modbus register data is null");
    auto swap_bytes_if_needed = [swap_bytes](const uint16_t value) -> uint16_t {
        if (swap_bytes) return (value & 0xFF) << 8 | (value & 0xFF00) >> 8;
        return value;
    };
    try {
        if (dt == telem::UINT16_T) return {swap_bytes_if_needed(data[0]), xerrors::NIL};
        if (dt == telem::INT16_T) {
            const uint16_t raw = swap_bytes_if_needed(data[0]);
            return {static_cast<int16_t>(raw), xerrors::NIL};
        }
        if (dt == telem::UINT32_T) {
            uint32_t result = 0;
            const uint16_t word1 = swap_bytes_if_needed(data[0]);
            const uint16_t word2 = swap_bytes_if_needed(data[1]);
            if (swap_words)
                result = static_cast<uint32_t>(word1) << 16 | word2;
            else
                result = static_cast<uint32_t>(word2) << 16 | word1;
            return {result, xerrors::NIL};
        }
        if (dt == telem::INT32_T) {
            uint32_t raw = 0;
            const uint16_t word1 = swap_bytes_if_needed(data[0]);
            const uint16_t word2 = swap_bytes_if_needed(data[1]);
            if (swap_words)
                raw = static_cast<uint32_t>(word1) << 16 | word2;
            else
                raw = static_cast<uint32_t>(word2) << 16 | word1;
            return {static_cast<int32_t>(raw), xerrors::NIL};
        }
        if (dt == telem::FLOAT32_T) {
            uint32_t raw = 0;
            const uint16_t word1 = swap_bytes_if_needed(data[0]);
            const uint16_t word2 = swap_bytes_if_needed(data[1]);
            if (swap_words)
                raw = static_cast<uint32_t>(word1) << 16 | word2;
            else
                raw = static_cast<uint32_t>(word2) << 16 | word1;
            float result;
            std::memcpy(&result, &raw, sizeof(float));
            return {result, xerrors::NIL};
        }
        if (dt == telem::UINT64_T) {
            uint64_t result = 0;
            uint16_t words[4];
            for (int i = 0; i < 4; i++)
                words[i] = swap_bytes_if_needed(data[i]);
            if (swap_words)
                result = static_cast<uint64_t>(words[0]) << 48 |
                         static_cast<uint64_t>(words[1]) << 32 |
                         static_cast<uint64_t>(words[2]) << 16 | words[3];
            else
                result = static_cast<uint64_t>(words[3]) << 48 |
                         static_cast<uint64_t>(words[2]) << 32 |
                         static_cast<uint64_t>(words[1]) << 16 | words[0];
            return {result, xerrors::NIL};
        }
        if (dt == telem::INT64_T) {
            uint64_t raw = 0;
            uint16_t words[4];
            for (int i = 0; i < 4; i++)
                words[i] = swap_bytes_if_needed(data[i]);
            if (swap_words)
                raw = static_cast<uint64_t>(words[0]) << 48 |
                      static_cast<uint64_t>(words[1]) << 32 |
                      static_cast<uint64_t>(words[2]) << 16 | words[3];
            else
                raw = static_cast<uint64_t>(words[3]) << 48 |
                      static_cast<uint64_t>(words[2]) << 32 |
                      static_cast<uint64_t>(words[1]) << 16 | words[0];
            return {static_cast<int64_t>(raw), xerrors::NIL};
        }
        if (dt == telem::FLOAT64_T) {
            uint64_t raw = 0;
            uint16_t words[4];
            for (int i = 0; i < 4; i++)
                words[i] = swap_bytes_if_needed(data[i]);
            if (swap_words)
                raw = static_cast<uint64_t>(words[0]) << 48 |
                      static_cast<uint64_t>(words[1]) << 32 |
                      static_cast<uint64_t>(words[2]) << 16 | words[3];
            else
                raw = static_cast<uint64_t>(words[3]) << 48 |
                      static_cast<uint64_t>(words[2]) << 32 |
                      static_cast<uint64_t>(words[1]) << 16 | words[0];
            double result;
            std::memcpy(&result, &raw, sizeof(double));
            return {result, xerrors::NIL};
        }
        if (dt == telem::UINT8_T) {
            const uint16_t raw = swap_bytes_if_needed(data[0]);
            return {static_cast<uint8_t>(raw & 0xFF), xerrors::NIL};
        }
        if (dt == telem::INT8_T) {
            const uint16_t raw = swap_bytes_if_needed(data[0]);
            return {static_cast<int8_t>(raw & 0xFF), xerrors::NIL};
        }
        return {
            telem::SampleValue(),
            xerrors::Error(xerrors::VALIDATION, "unsupported data type: " + dt.name())
        };
    } catch (const std::exception &e) {
        return {
            telem::SampleValue(),
            xerrors::Error(
                xerrors::VALIDATION,
                "failed to parse register: " + std::string(e.what())
            )
        };
    }
}

/// @brief formats a telem::SampleValue into a destination buffer of uint16_t values
/// representing modbus registers.
/// @param dt the data type fo the sample value.
/// @param dest the destination buffer to write to.
/// @param swap_bytes whether to swap the byte order of the data.
/// @param swap_words whether to swap the word order of the data.
inline xerrors::Error format_register(
    const telem::SampleValue &value,
    uint16_t *dest,
    const telem::DataType &dt,
    const bool swap_bytes,
    const bool swap_words
) {
    if (dest == nullptr) return xerrors::Error("modbus destination buffer is null");
    auto swap_bytes_if_needed = [swap_bytes](const uint16_t v) -> uint16_t {
        if (swap_bytes) return (v & 0xFF) << 8 | (v & 0xFF00) >> 8;
        return v;
    };

    try {
        if (dt == telem::UINT16_T) {
            dest[0] = swap_bytes_if_needed(telem::cast<uint16_t>(value));
            return xerrors::NIL;
        }
        if (dt == telem::INT16_T) {
            dest[0] = swap_bytes_if_needed(telem::cast<int16_t>(value));
            return xerrors::NIL;
        }
        if (dt == telem::UINT32_T) {
            const auto raw = telem::cast<uint32_t>(value);
            if (swap_words) {
                dest[0] = swap_bytes_if_needed(raw >> 16);
                dest[1] = swap_bytes_if_needed(raw & 0xFFFF);
            } else {
                dest[0] = swap_bytes_if_needed(raw & 0xFFFF);
                dest[1] = swap_bytes_if_needed(raw >> 16);
            }
            return xerrors::NIL;
        }
        if (dt == telem::INT32_T) {
            const auto raw = telem::cast<int32_t>(value);
            if (swap_words) {
                dest[0] = swap_bytes_if_needed(raw >> 16);
                dest[1] = swap_bytes_if_needed(raw & 0xFFFF);
            } else {
                dest[0] = swap_bytes_if_needed(raw & 0xFFFF);
                dest[1] = swap_bytes_if_needed(raw >> 16);
            }
            return xerrors::NIL;
        }
        if (dt == telem::FLOAT32_T) {
            const auto val = telem::cast<float>(value);
            uint32_t raw;
            std::memcpy(&raw, &val, sizeof(float));
            if (swap_words) {
                dest[0] = swap_bytes_if_needed(raw >> 16);
                dest[1] = swap_bytes_if_needed(raw & 0xFFFF);
            } else {
                dest[0] = swap_bytes_if_needed(raw & 0xFFFF);
                dest[1] = swap_bytes_if_needed(raw >> 16);
            }
            return xerrors::NIL;
        }
        if (dt == telem::UINT64_T) {
            const auto raw = telem::cast<uint64_t>(value);
            if (swap_words) {
                dest[0] = swap_bytes_if_needed(raw >> 48);
                dest[1] = swap_bytes_if_needed(raw >> 32 & 0xFFFF);
                dest[2] = swap_bytes_if_needed(raw >> 16 & 0xFFFF);
                dest[3] = swap_bytes_if_needed(raw & 0xFFFF);
            } else {
                dest[0] = swap_bytes_if_needed(raw & 0xFFFF);
                dest[1] = swap_bytes_if_needed(raw >> 16 & 0xFFFF);
                dest[2] = swap_bytes_if_needed(raw >> 32 & 0xFFFF);
                dest[3] = swap_bytes_if_needed(raw >> 48);
            }
            return xerrors::NIL;
        }
        if (dt == telem::INT64_T) {
            const uint64_t raw = telem::cast<int64_t>(value);
            if (swap_words) {
                dest[0] = swap_bytes_if_needed(raw >> 48);
                dest[1] = swap_bytes_if_needed(raw >> 32 & 0xFFFF);
                dest[2] = swap_bytes_if_needed(raw >> 16 & 0xFFFF);
                dest[3] = swap_bytes_if_needed(raw & 0xFFFF);
            } else {
                dest[0] = swap_bytes_if_needed(raw & 0xFFFF);
                dest[1] = swap_bytes_if_needed(raw >> 16 & 0xFFFF);
                dest[2] = swap_bytes_if_needed(raw >> 32 & 0xFFFF);
                dest[3] = swap_bytes_if_needed(raw >> 48);
            }
            return xerrors::NIL;
        }
        if (dt == telem::FLOAT64_T) {
            const auto val = telem::cast<double>(value);
            uint64_t raw;
            std::memcpy(&raw, &val, sizeof(double));
            if (swap_words) {
                dest[0] = swap_bytes_if_needed(raw >> 48);
                dest[1] = swap_bytes_if_needed(raw >> 32 & 0xFFFF);
                dest[2] = swap_bytes_if_needed(raw >> 16 & 0xFFFF);
                dest[3] = swap_bytes_if_needed(raw & 0xFFFF);
            } else {
                dest[0] = swap_bytes_if_needed(raw & 0xFFFF);
                dest[1] = swap_bytes_if_needed(raw >> 16 & 0xFFFF);
                dest[2] = swap_bytes_if_needed(raw >> 32 & 0xFFFF);
                dest[3] = swap_bytes_if_needed(raw >> 48);
            }
            return xerrors::NIL;
        }
        if (dt == telem::UINT8_T) {
            dest[0] = swap_bytes_if_needed(telem::cast<uint8_t>(value));
            return xerrors::NIL;
        }
        if (dt == telem::INT8_T) {
            dest[0] = swap_bytes_if_needed(telem::cast<int8_t>(value));
            return xerrors::NIL;
        }
        return xerrors::Error(
            xerrors::VALIDATION,
            "unsupported data type: " + dt.name()
        );
    } catch (const std::exception &e) {
        return xerrors::Error(
            xerrors::VALIDATION,
            "failed to format register: " + std::string(e.what())
        );
    }
}
}
