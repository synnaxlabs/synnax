// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <iomanip>
#include <sstream>
#include <string>

#include "x/cpp/telem/telem.h"

namespace ethercat {
/// EtherCAT/CoE data types as defined in ETG.1000.6.
/// These correspond to the data type field in the object dictionary.
enum class ECDataType : uint16_t {
    EC_UNKNOWN = 0x0000,
    EC_BOOLEAN = 0x0001,
    EC_INTEGER8 = 0x0002,
    EC_INTEGER16 = 0x0003,
    EC_INTEGER32 = 0x0004,
    EC_UNSIGNED8 = 0x0005,
    EC_UNSIGNED16 = 0x0006,
    EC_UNSIGNED32 = 0x0007,
    EC_REAL32 = 0x0008,
    EC_VISIBLE_STRING = 0x0009,
    EC_OCTET_STRING = 0x000A,
    EC_UNICODE_STRING = 0x000B,
    EC_TIME_OF_DAY = 0x000C,
    EC_TIME_DIFFERENCE = 0x000D,
    EC_DOMAIN = 0x000F,
    EC_INTEGER24 = 0x0010,
    EC_REAL64 = 0x0011,
    EC_INTEGER40 = 0x0012,
    EC_INTEGER48 = 0x0013,
    EC_INTEGER56 = 0x0014,
    EC_INTEGER64 = 0x0015,
    EC_UNSIGNED24 = 0x0016,
    EC_UNSIGNED40 = 0x0018,
    EC_UNSIGNED48 = 0x0019,
    EC_UNSIGNED56 = 0x001A,
    EC_UNSIGNED64 = 0x001B,
    EC_PDO_MAPPING = 0x0021,
    EC_IDENTITY = 0x0023,
    EC_PDO_PARAMETER = 0x0024,
    EC_PDO_COMMUNICATION = 0x0025,
    EC_BIT1 = 0x0030,
    EC_BIT2 = 0x0031,
    EC_BIT3 = 0x0032,
    EC_BIT4 = 0x0033,
    EC_BIT5 = 0x0034,
    EC_BIT6 = 0x0035,
    EC_BIT7 = 0x0036,
    EC_BIT8 = 0x0037,
};

/// Infers a Synnax data type from the bit length when the CoE type is unknown.
/// @param bit_length The size of the data in bits.
/// @param is_signed If true, returns signed types; otherwise unsigned (default).
/// @returns The most appropriate Synnax data type for the given size.
inline telem::DataType
infer_type_from_bit_length(const uint8_t bit_length, const bool is_signed = false) {
    if (bit_length == 0) return is_signed ? telem::INT8_T : telem::UINT8_T;
    if (bit_length == 1) return telem::UINT8_T; // Boolean, always unsigned
    if (bit_length <= 8) return is_signed ? telem::INT8_T : telem::UINT8_T;
    if (bit_length <= 16) return is_signed ? telem::INT16_T : telem::UINT16_T;
    if (bit_length <= 32) return is_signed ? telem::INT32_T : telem::UINT32_T;
    return is_signed ? telem::INT64_T : telem::UINT64_T;
}

/// Maps an EtherCAT CoE data type to a Synnax telem::DataType.
/// @param ec_type The EtherCAT data type from the object dictionary.
/// @param bit_length The bit length of the data (used for validation/fallback).
/// @returns The corresponding Synnax data type.
inline telem::DataType
map_ethercat_to_synnax(const ECDataType ec_type, const uint8_t bit_length) {
    switch (ec_type) {
        case ECDataType::EC_BOOLEAN:
        case ECDataType::EC_BIT1:
        case ECDataType::EC_BIT2:
        case ECDataType::EC_BIT3:
        case ECDataType::EC_BIT4:
        case ECDataType::EC_BIT5:
        case ECDataType::EC_BIT6:
        case ECDataType::EC_BIT7:
        case ECDataType::EC_BIT8:
        case ECDataType::EC_UNSIGNED8:
            return telem::UINT8_T;
        case ECDataType::EC_INTEGER8:
            return telem::INT8_T;
        case ECDataType::EC_UNSIGNED16:
            return telem::UINT16_T;
        case ECDataType::EC_INTEGER16:
            return telem::INT16_T;
        case ECDataType::EC_UNSIGNED24:
        case ECDataType::EC_UNSIGNED32:
            return telem::UINT32_T;
        case ECDataType::EC_INTEGER24:
        case ECDataType::EC_INTEGER32:
            return telem::INT32_T;
        case ECDataType::EC_UNSIGNED40:
        case ECDataType::EC_UNSIGNED48:
        case ECDataType::EC_UNSIGNED56:
        case ECDataType::EC_UNSIGNED64:
            return telem::UINT64_T;
        case ECDataType::EC_INTEGER40:
        case ECDataType::EC_INTEGER48:
        case ECDataType::EC_INTEGER56:
        case ECDataType::EC_INTEGER64:
            return telem::INT64_T;
        case ECDataType::EC_REAL32:
            return telem::FLOAT32_T;
        case ECDataType::EC_REAL64:
            return telem::FLOAT64_T;
        case ECDataType::EC_VISIBLE_STRING:
        case ECDataType::EC_OCTET_STRING:
        case ECDataType::EC_UNICODE_STRING:
            return telem::STRING_T;
        case ECDataType::EC_TIME_OF_DAY:
        case ECDataType::EC_TIME_DIFFERENCE:
            return telem::INT64_T;
        case ECDataType::EC_DOMAIN:
        case ECDataType::EC_PDO_MAPPING:
        case ECDataType::EC_IDENTITY:
        case ECDataType::EC_PDO_PARAMETER:
        case ECDataType::EC_PDO_COMMUNICATION:
        case ECDataType::EC_UNKNOWN:
        default:
            return infer_type_from_bit_length(bit_length);
    }
}

/// Generates a human-readable name for a PDO entry.
/// Uses the CoE name if available, otherwise generates from address.
/// @param coe_name The name from CoE object dictionary (may be empty).
/// @param index The object dictionary index.
/// @param subindex The object dictionary subindex.
/// @param is_input True for TxPDO (input), false for RxPDO (output).
/// @param data_type The Synnax data type.
/// @returns A descriptive name for the PDO entry.
inline std::string generate_pdo_entry_name(
    const std::string &coe_name,
    const uint16_t index,
    const uint8_t subindex,
    const bool is_input,
    const telem::DataType &data_type
) {
    if (!coe_name.empty()) return coe_name;

    std::ostringstream ss;
    ss << (is_input ? "Input" : "Output") << " (" << data_type.name() << ") 0x"
       << std::hex << std::uppercase << std::setfill('0') << std::setw(4) << index
       << ":" << std::setw(2) << static_cast<int>(subindex);
    return ss.str();
}

/// Formats an index:subindex pair as a hex string (e.g., "0x6000:01").
/// @param index The object dictionary index.
/// @param subindex The object dictionary subindex.
/// @returns Formatted hex string.
inline std::string format_index_subindex(const uint16_t index, const uint8_t subindex) {
    std::ostringstream ss;
    ss << "0x" << std::hex << std::uppercase << std::setfill('0') << std::setw(4)
       << index << ":" << std::setw(2) << static_cast<int>(subindex);
    return ss.str();
}
}
