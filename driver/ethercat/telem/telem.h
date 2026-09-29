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
#include <utility>

#include "absl/log/log.h"

#include "client/cpp/library/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/plan.h"
#include "driver/ethercat/pdo/pdo.h"

namespace driver::ethercat::telem {
/// @brief EtherCAT/CoE data types as defined in ETG.1000.6.
enum class DataType : uint16_t {
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

/// @brief infers a Synnax data type from the bit length when the CoE type is unknown.
inline x::telem::DataType
infer_type_from_bit_length(const uint8_t bit_length, const bool is_signed = false) {
    if (bit_length == 0) return is_signed ? x::telem::INT8_T : x::telem::UINT8_T;
    if (bit_length == 1) return x::telem::UINT8_T;
    if (bit_length <= 8) return is_signed ? x::telem::INT8_T : x::telem::UINT8_T;
    if (bit_length <= 16) return is_signed ? x::telem::INT16_T : x::telem::UINT16_T;
    if (bit_length <= 32) return is_signed ? x::telem::INT32_T : x::telem::UINT32_T;
    if (bit_length > 64)
        LOG(WARNING) << "bit length " << static_cast<int>(bit_length)
                     << " exceeds 64 bits, truncating to 64-bit type";
    return is_signed ? x::telem::INT64_T : x::telem::UINT64_T;
}

/// @brief maps an EtherCAT CoE data type to a Synnax x::telem::DataType.
inline x::telem::DataType
map_ethercat_to_synnax(const DataType ec_type, const uint8_t bit_length) {
    switch (ec_type) {
        case DataType::EC_BOOLEAN:
        case DataType::EC_BIT1:
        case DataType::EC_BIT2:
        case DataType::EC_BIT3:
        case DataType::EC_BIT4:
        case DataType::EC_BIT5:
        case DataType::EC_BIT6:
        case DataType::EC_BIT7:
        case DataType::EC_BIT8:
        case DataType::EC_UNSIGNED8:
            return x::telem::UINT8_T;
        case DataType::EC_INTEGER8:
            return x::telem::INT8_T;
        case DataType::EC_UNSIGNED16:
            return x::telem::UINT16_T;
        case DataType::EC_INTEGER16:
            return x::telem::INT16_T;
        case DataType::EC_UNSIGNED24:
        case DataType::EC_UNSIGNED32:
            return x::telem::UINT32_T;
        case DataType::EC_INTEGER24:
        case DataType::EC_INTEGER32:
            return x::telem::INT32_T;
        case DataType::EC_UNSIGNED40:
        case DataType::EC_UNSIGNED48:
        case DataType::EC_UNSIGNED56:
        case DataType::EC_UNSIGNED64:
            return x::telem::UINT64_T;
        case DataType::EC_INTEGER40:
        case DataType::EC_INTEGER48:
        case DataType::EC_INTEGER56:
        case DataType::EC_INTEGER64:
            return x::telem::INT64_T;
        case DataType::EC_REAL32:
            return x::telem::FLOAT32_T;
        case DataType::EC_REAL64:
            return x::telem::FLOAT64_T;
        case DataType::EC_VISIBLE_STRING:
        case DataType::EC_OCTET_STRING:
        case DataType::EC_UNICODE_STRING:
            return x::telem::STRING_T;
        case DataType::EC_TIME_OF_DAY:
        case DataType::EC_TIME_DIFFERENCE:
            return x::telem::INT64_T;
        case DataType::EC_DOMAIN:
        case DataType::EC_PDO_MAPPING:
        case DataType::EC_IDENTITY:
        case DataType::EC_PDO_PARAMETER:
        case DataType::EC_PDO_COMMUNICATION:
        case DataType::EC_UNKNOWN:
        default:
            return infer_type_from_bit_length(bit_length);
    }
}

/// @brief generates a human-readable name for a PDO entry.
inline std::string generate_pdo_entry_name(
    const std::string &coe_name,
    const uint16_t index,
    const uint8_t sub_index,
    const bool is_input,
    const x::telem::DataType &data_type
) {
    if (!coe_name.empty()) return coe_name;

    std::ostringstream ss;
    ss << (is_input ? "Input" : "Output") << " (" << data_type.name() << ") 0x"
       << std::hex << std::uppercase << std::setfill('0') << std::setw(4) << index
       << ":" << std::setw(2) << static_cast<int>(sub_index);
    return ss.str();
}

/// @brief formats an index:sub_index pair as a hex string (e.g., "0x6000:01").
inline std::string
format_index_sub_index(const uint16_t index, const uint8_t sub_index) {
    std::ostringstream ss;
    ss << "0x" << std::hex << std::uppercase << std::setfill('0') << std::setw(4)
       << index << ":" << std::setw(2) << static_cast<int>(sub_index);
    return ss.str();
}

/// @brief compiles a codec plan whose one slot is the entry's value, in a buffer that
/// starts at the entry's first byte. Signed and float data types decode as two's
/// complement and IEEE 754; every other type decodes unsigned.
/// @param bit the bit offset of the entry within its first byte, from 0 to 7.
/// @returns CONFIG_ERROR when the bit length is not from 1 to 64, or when a float
/// entry is not 32 or 64 bits.
inline std::pair<codec::Plan, x::errors::Error>
plan(const pdo::Entry &entry, const uint8_t bit) {
    synnax::library::BinaryField f;
    f.name = format_index_sub_index(entry.index, entry.sub_index);
    f.start_bit = bit;
    f.bit_length = entry.bit_length;
    f.signed_ = entry.data_type.matches(
        {x::telem::INT8_T, x::telem::INT16_T, x::telem::INT32_T, x::telem::INT64_T}
    );
    f.float_ = entry.data_type.matches({x::telem::FLOAT32_T, x::telem::FLOAT64_T});
    synnax::library::MessageEntry message;
    message.fields = {f};
    return codec::Plan::compile(message);
}
}
