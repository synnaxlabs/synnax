// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "open62541/types.h"
#include "glog/logging.h"

/// module
#include "x/cpp/telem/series.h"

/// internal
#include "driver/opc/util/util.h"

namespace util {
telem::DataType ua_to_data_type(const UA_DataType *dt) {
    if (dt == &UA_TYPES[UA_TYPES_FLOAT]) return telem::FLOAT32_T;
    if (dt == &UA_TYPES[UA_TYPES_DOUBLE]) return telem::FLOAT64_T;
    if (dt == &UA_TYPES[UA_TYPES_SBYTE]) return telem::INT8_T;
    if (dt == &UA_TYPES[UA_TYPES_INT16]) return telem::INT16_T;
    if (dt == &UA_TYPES[UA_TYPES_INT32]) return telem::INT32_T;
    if (dt == &UA_TYPES[UA_TYPES_INT64]) return telem::INT64_T;
    if (dt == &UA_TYPES[UA_TYPES_BYTE]) return telem::UINT8_T;
    if (dt == &UA_TYPES[UA_TYPES_UINT16]) return telem::UINT16_T;
    if (dt == &UA_TYPES[UA_TYPES_UINT32]) return telem::UINT32_T;
    if (dt == &UA_TYPES[UA_TYPES_UINT64]) return telem::UINT64_T;
    if (dt == &UA_TYPES[UA_TYPES_STRING]) return telem::STRING_T;
    if (dt == &UA_TYPES[UA_TYPES_DATETIME]) return telem::TIMESTAMP_T;
    if (dt == &UA_TYPES[UA_TYPES_GUID]) return telem::UINT128_T;
    if (dt == &UA_TYPES[UA_TYPES_BOOLEAN]) return telem::UINT8_T;
    return telem::UNKNOWN_T;
}

UA_DataType *data_type_to_ua(const telem::DataType &data_type) {
    if (data_type == telem::FLOAT32_T) return &UA_TYPES[UA_TYPES_FLOAT];
    if (data_type == telem::FLOAT64_T) return &UA_TYPES[UA_TYPES_DOUBLE];
    if (data_type == telem::INT8_T) return &UA_TYPES[UA_TYPES_SBYTE];
    if (data_type == telem::INT16_T) return &UA_TYPES[UA_TYPES_INT16];
    if (data_type == telem::INT32_T) return &UA_TYPES[UA_TYPES_INT32];
    if (data_type == telem::INT64_T) return &UA_TYPES[UA_TYPES_INT64];
    if (data_type == telem::UINT8_T) return &UA_TYPES[UA_TYPES_BYTE];
    if (data_type == telem::UINT16_T) return &UA_TYPES[UA_TYPES_UINT16];
    if (data_type == telem::UINT32_T) return &UA_TYPES[UA_TYPES_UINT32];
    if (data_type == telem::UINT64_T) return &UA_TYPES[UA_TYPES_UINT64];
    if (data_type == telem::STRING_T) return &UA_TYPES[UA_TYPES_STRING];
    if (data_type == telem::TIMESTAMP_T) return &UA_TYPES[UA_TYPES_DATETIME];
    if (data_type == telem::UINT128_T) return &UA_TYPES[UA_TYPES_GUID];
    if (data_type == telem::UINT8_T) return &UA_TYPES[UA_TYPES_BOOLEAN];
    return &UA_TYPES[UA_TYPES_VARIANT];
}

///@brief Define constants for the conversion
static constexpr int64_t UNIX_EPOCH_START_1601 = 11644473600LL;
// Seconds from 1601 to 1970
static constexpr int64_t HUNDRED_NANOSECOND_INTERVALS_PER_SECOND = 10000000LL;
// 100-nanosecond intervals per second
constexpr int64_t UNIX_EPOCH_START_IN_100_NANO_INTERVALS =
        UNIX_EPOCH_START_1601 * HUNDRED_NANOSECOND_INTERVALS_PER_SECOND;

inline int64_t ua_datetime_to_unix_nano(const UA_DateTime dateTime) {
    return (dateTime - UNIX_EPOCH_START_IN_100_NANO_INTERVALS) * 100;
}

std::pair<telem::Series, xerrors::Error> ua_array_to_series(
    const telem::DataType &target_type,
    const UA_Variant *val,
    const size_t target_size
) {
    if (val->arrayLength < target_size)
        return {
            telem::Series(0),
            xerrors::Error(xerrors::VALIDATION,
                           "OPC UA array is too small for configured array size of " +
                           std::to_string(target_size))
        };
    size_t size = val->arrayLength;
    if (size > target_size) size = target_size;
    if (UA_Variant_isScalar(val))
        return {
            telem::Series(0),
            xerrors::Error(xerrors::VALIDATION, "cannot not convert scalar to series")
        };
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_DATETIME])) {
        const UA_DateTime *data = static_cast<UA_DateTime *>(val->data);
        auto s = telem::Series(target_type, val->arrayLength);
        size_t acc = 0;
        for (size_t j = 0; j < val->arrayLength; ++j)
            acc += s.write(ua_datetime_to_unix_nano(data[j]));
        return {std::move(s), xerrors::NIL};
    }
    return {
        telem::Series::cast(
            target_type,
            val->data,
            size,
            ua_to_data_type(val->type)
        ),
        xerrors::NIL
    };
}

std::pair<UA_Variant, xerrors::Error> series_to_variant(const telem::Series &s) {
    UA_Variant v;
    UA_Variant_init(&v);
    const auto dt = data_type_to_ua(s.data_type());
    const auto status = UA_Variant_setScalarCopy(
        &v,
        telem::cast_to_void_ptr(s.at(-1)),
        dt
    );
    return {v, parse_error(status)};
}

size_t write_to_series(telem::Series &s, const UA_Variant &v) {
    return s.write(s.data_type().cast(v.data, ua_to_data_type(v.type)));
}
}
