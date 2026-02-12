// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <functional>
#include <vector>

#pragma once

#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

namespace driver::ethercat::pdo {
/// @brief key for PDO offset cache lookup.
struct Key {
    uint16_t slave_position;
    uint16_t index;
    uint8_t sub_index;
    bool is_input;

    bool operator==(const Key &other) const {
        return slave_position == other.slave_position && index == other.index &&
               sub_index == other.sub_index && is_input == other.is_input;
    }
};

/// @brief hash function for pdo::Key.
struct KeyHash {
    size_t operator()(const Key &key) const {
        return std::hash<uint64_t>()(
            (static_cast<uint64_t>(key.slave_position) << 32) |
            (static_cast<uint64_t>(key.index) << 16) |
            (static_cast<uint64_t>(key.sub_index) << 8) |
            static_cast<uint64_t>(key.is_input)
        );
    }
};

struct Offset {
    /// @brief byte offset into the appropriate buffer (input_data or output_data).
    size_t byte = 0;
    /// @brief bit offset within the byte for sub-byte entries (0-7).
    uint8_t bit = 0;
};

using Offsets = std::unordered_map<Key, Offset, KeyHash>;

/// @brief describes a single PDO entry (object) to be exchanged cyclically.
struct Entry {
    /// @brief position of the slave on the EtherCAT bus.
    uint16_t slave_position = 0;
    /// @brief index of the PDO object in the CoE object dictionary.
    uint16_t index = 0;
    /// @brief sub_index of the PDO object.
    uint8_t sub_index = 0;
    /// @brief size of the data in bits.
    uint8_t bit_length = 0;
    /// @brief true for input (TxPDO), false for output (RxPDO).
    bool is_input = true;
    /// @brief actual hardware data type from the PDO.
    x::telem::DataType data_type = x::telem::UNKNOWN_T;

    /// @brief returns the size of this PDO entry in bytes (rounded up from bits).
    [[nodiscard]] size_t byte_length() const { return (bit_length + 7) / 8; }
};

/// @brief information about a single PDO entry discovered during slave enumeration.
struct Properties {
    /// @brief parent PDO index (e.g., 0x1A00 for TxPDO, 0x1600 for RxPDO).
    uint16_t pdo_index;
    /// @brief object dictionary index of this entry.
    uint16_t index;
    /// @brief object dictionary sub_index of this entry.
    uint8_t sub_index;
    /// @brief size of the data in bits.
    uint8_t bit_length;
    /// @brief true for input (TxPDO), false for output (RxPDO).
    bool is_input;
    /// @brief human-readable name from CoE object dictionary.
    std::string name;
    /// @brief Synnax data type for channel creation.
    x::telem::DataType data_type;

    /// @brief returns the size of this PDO entry in bytes (rounded up from bits).
    [[nodiscard]] size_t byte_length() const { return (this->bit_length + 7) / 8; }

    /// @brief parses PDO properties from JSON.
    static Properties parse(x::json::Parser &parser, const bool is_input) {
        return {
            .pdo_index = static_cast<uint16_t>(parser.field<int>("pdo_index", 0)),
            .index = static_cast<uint16_t>(parser.field<int>("index")),
            .sub_index = static_cast<uint8_t>(parser.field<int>("sub_index")),
            .bit_length = static_cast<uint8_t>(parser.field<int>("bit_length")),
            .is_input = is_input,
            .name = parser.field<std::string>("name"),
            .data_type = x::telem::DataType(parser.field<std::string>("data_type")),
        };
    }

    /// @brief serializes this PDO entry to JSON.
    [[nodiscard]] nlohmann::json to_json() const {
        return {
            {"name", this->name},
            {"pdo_index", this->pdo_index},
            {"index", this->index},
            {"sub_index", this->sub_index},
            {"bit_length", this->bit_length},
            {"data_type", this->data_type.name()}
        };
    }
};

/// @brief looks up the offset for a PDO entry in a precomputed offset map.
inline Offset find_offset(const Offsets &offsets, const Entry &entry) {
    Key key{entry.slave_position, entry.index, entry.sub_index, entry.is_input};
    auto it = offsets.find(key);
    if (it != offsets.end()) return it->second;
    return {};
}

/// @brief computes bit-granularity offsets for a slave's PDO properties list,
/// accumulating from a base byte offset.
inline void compute_offsets(
    Offsets &offsets,
    const uint16_t slave_position,
    const std::vector<Properties> &pdos,
    const bool is_input,
    const size_t base_byte_offset
) {
    size_t bit_offset = 0;
    for (const auto &pdo: pdos) {
        Key key{slave_position, pdo.index, pdo.sub_index, is_input};
        offsets[key] = {
            base_byte_offset + bit_offset / 8,
            static_cast<uint8_t>(bit_offset % 8)
        };
        bit_offset += pdo.bit_length;
    }
}

/// @brief computes byte-granularity offsets for registered PDO entries,
/// splitting inputs and outputs from separate base offsets.
inline void compute_offsets(
    Offsets &offsets,
    const std::vector<Entry> &entries,
    const size_t input_base_offset,
    const size_t output_base_offset
) {
    size_t input_byte_offset = input_base_offset;
    size_t output_byte_offset = output_base_offset;
    for (const auto &entry: entries) {
        Key key{entry.slave_position, entry.index, entry.sub_index, entry.is_input};
        if (entry.is_input) {
            offsets[key] = {input_byte_offset, 0};
            input_byte_offset += entry.byte_length();
        } else {
            offsets[key] = {output_byte_offset, 0};
            output_byte_offset += entry.byte_length();
        }
    }
}
}
