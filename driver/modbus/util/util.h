// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// module
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

namespace util {
inline std::pair<telem::SampleValue, xerrors::Error> parse_register(
    const uint16_t *data,
    const size_t offset,
    const telem::DataType &dt,
    const bool swap_bytes,
    const bool swap_words
) {
    auto swap_bytes_if_needed = [swap_bytes](const uint16_t value) -> uint16_t {
        if (swap_bytes)
            return (value & 0xFF) << 8 | (value & 0xFF00) >> 8;
        return value;
    };
    try {
        if (dt == telem::UINT16_T)
            return {swap_bytes_if_needed(data[offset]), xerrors::Error()};
        if (dt == telem::INT16_T) {
            const uint16_t raw = swap_bytes_if_needed(data[offset]);
            return {static_cast<int16_t>(raw), xerrors::Error()};
        }
        if (dt == telem::UINT32_T) {
            uint32_t result = 0;
            const uint16_t word1 = swap_bytes_if_needed(data[offset]);
            const uint16_t word2 = swap_bytes_if_needed(data[offset + 1]);
            if (swap_words)
                result = static_cast<uint32_t>(word1) << 16 | word2;
            else
                result = static_cast<uint32_t>(word2) << 16 | word1;
            return {result, xerrors::Error()};
        }
        if (dt == telem::INT32_T) {
            uint32_t raw = 0;
            const uint16_t word1 = swap_bytes_if_needed(data[offset]);
            const uint16_t word2 = swap_bytes_if_needed(data[offset + 1]);
            if (swap_words)
                raw = static_cast<uint32_t>(word1) << 16 | word2;
            else
                raw = static_cast<uint32_t>(word2) << 16 | word1;
            return {static_cast<int32_t>(raw), xerrors::Error()};
        }
        if (dt == telem::FLOAT32_T) {
            uint32_t raw = 0;
            const uint16_t word1 = swap_bytes_if_needed(data[offset]);
            const uint16_t word2 = swap_bytes_if_needed(data[offset + 1]);
            if (swap_words)
                raw = static_cast<uint32_t>(word1) << 16 | word2;
            else
                raw = static_cast<uint32_t>(word2) << 16 | word1;
            float result;
            std::memcpy(&result, &raw, sizeof(float));
            return {result, xerrors::Error()};
        }
        if (dt == telem::UINT64_T) {
            uint64_t result = 0;
            uint16_t words[4];
            for (int i = 0; i < 4; i++)
                words[i] = swap_bytes_if_needed(data[offset + i]);
            if (swap_words)
                result = static_cast<uint64_t>(words[0]) << 48 |
                         static_cast<uint64_t>(words[1]) << 32 |
                         static_cast<uint64_t>(words[2]) << 16 |
                         words[3];
            else
                result = static_cast<uint64_t>(words[3]) << 48 |
                         static_cast<uint64_t>(words[2]) << 32 |
                         static_cast<uint64_t>(words[1]) << 16 |
                         words[0];
            return {result, xerrors::Error()};
        }
        if (dt == telem::INT64_T) {
            uint64_t raw = 0;
            uint16_t words[4];
            for (int i = 0; i < 4; i++)
                words[i] = swap_bytes_if_needed(data[offset + i]);
            if (swap_words)
                raw = static_cast<uint64_t>(words[0]) << 48 |
                      static_cast<uint64_t>(words[1]) << 32 |
                      static_cast<uint64_t>(words[2]) << 16 |
                      words[3];
            else
                raw = static_cast<uint64_t>(words[3]) << 48 |
                      static_cast<uint64_t>(words[2]) << 32 |
                      static_cast<uint64_t>(words[1]) << 16 |
                      words[0];
            return {static_cast<int64_t>(raw), xerrors::Error()};
        }
        if (dt == telem::FLOAT64_T) {
            uint64_t raw = 0;
            uint16_t words[4];
            for (int i = 0; i < 4; i++)
                words[i] = swap_bytes_if_needed(data[offset + i]);
            if (swap_words)
                raw = static_cast<uint64_t>(words[0]) << 48 |
                      static_cast<uint64_t>(words[1]) << 32 |
                      static_cast<uint64_t>(words[2]) << 16 |
                      words[3];
            else
                raw = static_cast<uint64_t>(words[3]) << 48 |
                      static_cast<uint64_t>(words[2]) << 32 |
                      static_cast<uint64_t>(words[1]) << 16 |
                      words[0];
            double result;
            std::memcpy(&result, &raw, sizeof(double));
            return {result, xerrors::Error()};
        }
        if (dt == telem::UINT8_T) {
            const uint16_t raw = swap_bytes_if_needed(data[offset]);
            return {static_cast<uint8_t>(raw & 0xFF), xerrors::Error()};
        }
        if (dt == telem::INT8_T) {
            const uint16_t raw = swap_bytes_if_needed(data[offset]);
            return {static_cast<int8_t>(raw & 0xFF), xerrors::Error()};
        }
        return {
            telem::SampleValue(),
            xerrors::Error("Unsupported data type: " + dt.name())
        };
    } catch (const std::exception &e) {
        return {
            telem::SampleValue(),
            xerrors::Error("Failed to parse register: " + std::string(e.what()))
        };
    }
}
}