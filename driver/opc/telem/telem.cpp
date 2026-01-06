// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "glog/logging.h"
#include "open62541/types.h"

/// module
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

/// internal
#include "driver/opc/errors/errors.h"
#include "driver/opc/telem/telem.h"

namespace opc::telem {
::telem::DataType ua_to_data_type(const UA_DataType *dt) {
    if (dt == &UA_TYPES[UA_TYPES_FLOAT]) return ::telem::FLOAT32_T;
    if (dt == &UA_TYPES[UA_TYPES_DOUBLE]) return ::telem::FLOAT64_T;
    if (dt == &UA_TYPES[UA_TYPES_SBYTE]) return ::telem::INT8_T;
    if (dt == &UA_TYPES[UA_TYPES_INT16]) return ::telem::INT16_T;
    if (dt == &UA_TYPES[UA_TYPES_INT32]) return ::telem::INT32_T;
    if (dt == &UA_TYPES[UA_TYPES_INT64]) return ::telem::INT64_T;
    if (dt == &UA_TYPES[UA_TYPES_BYTE]) return ::telem::UINT8_T;
    if (dt == &UA_TYPES[UA_TYPES_UINT16]) return ::telem::UINT16_T;
    if (dt == &UA_TYPES[UA_TYPES_UINT32]) return ::telem::UINT32_T;
    if (dt == &UA_TYPES[UA_TYPES_UINT64]) return ::telem::UINT64_T;
    if (dt == &UA_TYPES[UA_TYPES_STRING]) return ::telem::STRING_T;
    if (dt == &UA_TYPES[UA_TYPES_DATETIME]) return ::telem::TIMESTAMP_T;
    if (dt == &UA_TYPES[UA_TYPES_GUID]) return ::telem::UUID_T;
    if (dt == &UA_TYPES[UA_TYPES_BOOLEAN]) return ::telem::UINT8_T;
    return ::telem::UNKNOWN_T;
}

UA_DataType *data_type_to_ua(const ::telem::DataType &data_type) {
    if (data_type == ::telem::FLOAT32_T) return &UA_TYPES[UA_TYPES_FLOAT];
    if (data_type == ::telem::FLOAT64_T) return &UA_TYPES[UA_TYPES_DOUBLE];
    if (data_type == ::telem::INT8_T) return &UA_TYPES[UA_TYPES_SBYTE];
    if (data_type == ::telem::INT16_T) return &UA_TYPES[UA_TYPES_INT16];
    if (data_type == ::telem::INT32_T) return &UA_TYPES[UA_TYPES_INT32];
    if (data_type == ::telem::INT64_T) return &UA_TYPES[UA_TYPES_INT64];
    if (data_type == ::telem::UINT16_T) return &UA_TYPES[UA_TYPES_UINT16];
    if (data_type == ::telem::UINT32_T) return &UA_TYPES[UA_TYPES_UINT32];
    if (data_type == ::telem::UINT64_T) return &UA_TYPES[UA_TYPES_UINT64];
    if (data_type == ::telem::STRING_T) return &UA_TYPES[UA_TYPES_STRING];
    if (data_type == ::telem::TIMESTAMP_T) return &UA_TYPES[UA_TYPES_DATETIME];
    if (data_type == ::telem::UUID_T) return &UA_TYPES[UA_TYPES_GUID];
    if (data_type == ::telem::UINT8_T) return &UA_TYPES[UA_TYPES_BOOLEAN];
    return &UA_TYPES[UA_TYPES_VARIANT];
}

///@brief Define constants for the conversion
static constexpr int64_t UNIX_EPOCH_START_1601 = 11644473600LL;
// Seconds from 1601 to 1970
static constexpr int64_t HUNDRED_NANOSECOND_INTERVALS_PER_SECOND = 10000000LL;
// 100-nanosecond intervals per second
constexpr int64_t
    UNIX_EPOCH_START_IN_100_NANO_INTERVALS = UNIX_EPOCH_START_1601 *
                                             HUNDRED_NANOSECOND_INTERVALS_PER_SECOND;

inline int64_t ua_datetime_to_unix_nano(const UA_DateTime dateTime) {
    return (dateTime - UNIX_EPOCH_START_IN_100_NANO_INTERVALS) * 100;
}

std::pair<size_t, xerrors::Error> ua_array_write_to_series(
    ::telem::Series &series,
    const UA_Variant *val,
    const size_t target_size,
    const std::string &name
) {
    const size_t size = val->arrayLength;
    if (size != target_size) {
        const std::string verb = size < target_size ? "small" : "large";
        return {
            0,
            xerrors::Error(
                xerrors::VALIDATION,
                "OPC UA array for " + name + " is too " + verb +
                    " (size: " + std::to_string(size) +
                    ") for configured array size of " + std::to_string(target_size)
            )
        };
    }

    if (UA_Variant_isScalar(val))
        return {
            0,
            xerrors::Error(xerrors::VALIDATION, "cannot not convert scalar to series")
        };

    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_DATETIME])) {
        const UA_DateTime *data = static_cast<UA_DateTime *>(val->data);
        size_t written = 0;
        for (size_t j = 0; j < size; ++j)
            written += series.write(ua_datetime_to_unix_nano(data[j]));
        return {written, xerrors::NIL};
    }

    return {
        series.write_casted(val->data, size, ua_to_data_type(val->type)),
        xerrors::NIL
    };
}

std::pair<UA_Variant, xerrors::Error> series_to_variant(const ::telem::Series &s) {
    UA_Variant v;
    UA_Variant_init(&v);
    const auto dt = data_type_to_ua(s.data_type());
    auto sample = s.at(-1);
    const auto status = UA_Variant_setScalarCopy(&v, cast_to_void_ptr(sample), dt);
    return {v, opc::errors::parse(status)};
}

std::pair<size_t, xerrors::Error>
write_to_series(::telem::Series &s, const UA_Variant &v) {
    if (s.data_type() == ::telem::TIMESTAMP_T &&
        v.type == &UA_TYPES[UA_TYPES_DATETIME]) {
        const auto dt = static_cast<const UA_DateTime *>(v.data);
        return {
            s.write(s.data_type().cast(ua_datetime_to_unix_nano(*dt))),
            xerrors::NIL
        };
    }

    if (v.type == nullptr) {
        return {0, xerrors::Error(xerrors::VALIDATION, "variant has null type")};
    }

    if (v.data == nullptr) {
        return {0, xerrors::Error(xerrors::VALIDATION, "variant has null data")};
    }

    const bool is_scalar = UA_Variant_isScalar(&v);
    if (!is_scalar && v.arrayLength == 0) {
        return {
            0,
            xerrors::Error(xerrors::VALIDATION, "variant is array with zero length")
        };
    }

    try {
        return {
            s.write(s.data_type().cast(v.data, ua_to_data_type(v.type))),
            xerrors::NIL
        };
    } catch (const std::exception &e) {
        return {
            0,
            xerrors::Error(
                xerrors::VALIDATION,
                "exception during cast/write: " + std::string(e.what())
            )
        };
    }
}
}
