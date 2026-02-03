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
#include <cstring>
#include <iomanip>
#include <sstream>
#include <string>

#include "glog/logging.h"

#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

namespace ethercat {
/// @brief EtherCAT/CoE data types as defined in ETG.1000.6.
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

/// @brief infers a Synnax data type from the bit length when the CoE type is unknown.
inline telem::DataType
infer_type_from_bit_length(const uint8_t bit_length, const bool is_signed = false) {
    if (bit_length == 0) return is_signed ? telem::INT8_T : telem::UINT8_T;
    if (bit_length == 1) return telem::UINT8_T;
    if (bit_length <= 8) return is_signed ? telem::INT8_T : telem::UINT8_T;
    if (bit_length <= 16) return is_signed ? telem::INT16_T : telem::UINT16_T;
    if (bit_length <= 32) return is_signed ? telem::INT32_T : telem::UINT32_T;
    if (bit_length > 64)
        LOG(WARNING) << "bit length " << static_cast<int>(bit_length)
                     << " exceeds 64 bits, truncating to 64-bit type";
    return is_signed ? telem::INT64_T : telem::UINT64_T;
}

/// @brief maps an EtherCAT CoE data type to a Synnax telem::DataType.
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

/// @brief generates a human-readable name for a PDO entry.
inline std::string generate_pdo_entry_name(
    const std::string &coe_name,
    const uint16_t index,
    const uint8_t sub_index,
    const bool is_input,
    const telem::DataType &data_type
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

/// @brief reads a PDO value from a byte buffer and writes it to a series.
/// Handles sub-byte values, 24-bit values, and standard byte-aligned values.
/// @param src pointer to the start of the PDO data in the buffer.
/// @param bit_offset bit offset within the first byte (0-7).
/// @param bit_length total bit length of the value.
/// @param data_type the data type for interpreting the value.
/// @param series the series to write the extracted value to.
inline void read_pdo_to_series(
    const uint8_t *src,
    const uint8_t bit_offset,
    const uint8_t bit_length,
    const telem::DataType data_type,
    telem::Series &series
) {
    if (bit_length < 8) {
        uint16_t two_bytes = src[0];
        if (bit_offset + bit_length > 8)
            two_bytes |= static_cast<uint16_t>(src[1]) << 8;
        const uint8_t mask = static_cast<uint8_t>((1u << bit_length) - 1);
        const uint8_t extracted = static_cast<uint8_t>(
            (two_bytes >> bit_offset) & mask
        );
        series.write_casted(&extracted, 1, telem::UINT8_T);
        return;
    }

    if (bit_length == 24) {
        uint32_t raw = static_cast<uint32_t>(src[0]) |
                       (static_cast<uint32_t>(src[1]) << 8) |
                       (static_cast<uint32_t>(src[2]) << 16);
        if (bit_offset > 0)
            raw = (raw >> bit_offset) |
                  (static_cast<uint32_t>(src[3]) << (24 - bit_offset));
        uint32_t val = raw & 0x00FFFFFF;
        if (data_type == telem::INT32_T || data_type == telem::INT64_T)
            if (val & 0x800000) val |= 0xFF000000;
        series.write_casted(&val, 1, data_type);
        return;
    }

    telem::DataType source_type = data_type;
    if (source_type == telem::UNKNOWN_T) source_type = series.data_type();
    series.write_casted(src, 1, source_type);
}

/// @brief writes a sample value to a byte buffer as a PDO value.
/// Handles sub-byte values, 24-bit values, and standard byte-aligned values.
/// @param dest pointer to the start of the PDO data in the buffer.
/// @param bit_offset bit offset within the first byte (0-7).
/// @param bit_length total bit length of the value.
/// @param data_type the data type for interpreting the value.
/// @param value the sample value to write.
inline void write_pdo_from_value(
    uint8_t *dest,
    const uint8_t bit_offset,
    const uint8_t bit_length,
    const telem::DataType data_type,
    const telem::SampleValue &value
) {
    const auto casted = data_type == telem::UNKNOWN_T ? value : data_type.cast(value);

    if (bit_length < 8) {
        const auto src_val = telem::cast<uint8_t>(casted);
        const uint16_t mask = static_cast<uint16_t>((1u << bit_length) - 1);

        if (bit_offset + bit_length > 8) {
            uint16_t two_bytes = static_cast<uint16_t>(dest[0]) |
                                 (static_cast<uint16_t>(dest[1]) << 8);
            const uint16_t shifted_mask = static_cast<uint16_t>(mask << bit_offset);
            const uint16_t shifted_val = static_cast<uint16_t>(
                (src_val & mask) << bit_offset
            );
            two_bytes = static_cast<uint16_t>(
                (two_bytes & ~shifted_mask) | shifted_val
            );
            dest[0] = static_cast<uint8_t>(two_bytes & 0xFF);
            dest[1] = static_cast<uint8_t>((two_bytes >> 8) & 0xFF);
        } else {
            const uint8_t shifted_mask = static_cast<uint8_t>(mask << bit_offset);
            const uint8_t shifted_val = static_cast<uint8_t>(
                (src_val & mask) << bit_offset
            );
            dest[0] = static_cast<uint8_t>((dest[0] & ~shifted_mask) | shifted_val);
        }
        return;
    }

    if (bit_length == 24) {
        const uint32_t src_val = telem::cast<uint32_t>(casted);
        const uint32_t masked_val = src_val & 0x00FFFFFF;

        if (bit_offset > 0) {
            uint32_t four_bytes = static_cast<uint32_t>(dest[0]) |
                                  (static_cast<uint32_t>(dest[1]) << 8) |
                                  (static_cast<uint32_t>(dest[2]) << 16) |
                                  (static_cast<uint32_t>(dest[3]) << 24);
            const uint32_t write_mask = 0x00FFFFFFu << bit_offset;
            const uint32_t shifted_val = masked_val << bit_offset;
            four_bytes = (four_bytes & ~write_mask) | shifted_val;
            dest[0] = static_cast<uint8_t>(four_bytes & 0xFF);
            dest[1] = static_cast<uint8_t>((four_bytes >> 8) & 0xFF);
            dest[2] = static_cast<uint8_t>((four_bytes >> 16) & 0xFF);
            dest[3] = static_cast<uint8_t>((four_bytes >> 24) & 0xFF);
        } else {
            dest[0] = static_cast<uint8_t>(masked_val & 0xFF);
            dest[1] = static_cast<uint8_t>((masked_val >> 8) & 0xFF);
            dest[2] = static_cast<uint8_t>((masked_val >> 16) & 0xFF);
        }
        return;
    }

    const size_t byte_len = (bit_length + 7) / 8;
    const void *data = telem::cast_to_void_ptr(casted);
    std::memcpy(dest, data, byte_len);
}

/// @brief calculates the number of bytes required to read/write a PDO value.
/// Accounts for bit offsets that may cause values to span additional bytes.
/// @param bit_offset bit offset within the first byte (0-7).
/// @param bit_length total bit length of the value.
/// @return the number of bytes required in the buffer.
inline size_t pdo_required_bytes(const uint8_t bit_offset, const uint8_t bit_length) {
    const size_t byte_len = (bit_length + 7) / 8;
    if (bit_length == 24 && bit_offset > 0) return 4;
    if (bit_length < 8 && bit_offset + bit_length > 8) return 2;
    return byte_len;
}
}
