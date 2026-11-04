// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <limits>
#include <string>
#include <vector>

#include "open62541/types.h"

#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/opcua/errors/errors.h"
#include "driver/opcua/telem/telem.h"

namespace driver::opcua::telem {
x::telem::DataType ua_to_data_type(const UA_DataType *dt) {
    if (dt == &UA_TYPES[UA_TYPES_FLOAT]) return x::telem::FLOAT32_T;
    if (dt == &UA_TYPES[UA_TYPES_DOUBLE]) return x::telem::FLOAT64_T;
    if (dt == &UA_TYPES[UA_TYPES_SBYTE]) return x::telem::INT8_T;
    if (dt == &UA_TYPES[UA_TYPES_INT16]) return x::telem::INT16_T;
    if (dt == &UA_TYPES[UA_TYPES_INT32]) return x::telem::INT32_T;
    if (dt == &UA_TYPES[UA_TYPES_INT64]) return x::telem::INT64_T;
    if (dt == &UA_TYPES[UA_TYPES_BYTE]) return x::telem::UINT8_T;
    if (dt == &UA_TYPES[UA_TYPES_UINT16]) return x::telem::UINT16_T;
    if (dt == &UA_TYPES[UA_TYPES_UINT32]) return x::telem::UINT32_T;
    if (dt == &UA_TYPES[UA_TYPES_UINT64]) return x::telem::UINT64_T;
    if (dt == &UA_TYPES[UA_TYPES_STRING]) return x::telem::STRING_T;
    if (dt == &UA_TYPES[UA_TYPES_DATETIME]) return x::telem::TIMESTAMP_T;
    if (dt == &UA_TYPES[UA_TYPES_GUID]) return x::telem::UUID_T;
    if (dt == &UA_TYPES[UA_TYPES_BOOLEAN]) return x::telem::UINT8_T;
    return x::telem::UNKNOWN_T;
}

UA_DataType *data_type_to_ua(const x::telem::DataType &data_type) {
    if (data_type == x::telem::FLOAT32_T) return &UA_TYPES[UA_TYPES_FLOAT];
    if (data_type == x::telem::FLOAT64_T) return &UA_TYPES[UA_TYPES_DOUBLE];
    if (data_type == x::telem::INT8_T) return &UA_TYPES[UA_TYPES_SBYTE];
    if (data_type == x::telem::INT16_T) return &UA_TYPES[UA_TYPES_INT16];
    if (data_type == x::telem::INT32_T) return &UA_TYPES[UA_TYPES_INT32];
    if (data_type == x::telem::INT64_T) return &UA_TYPES[UA_TYPES_INT64];
    if (data_type == x::telem::UINT16_T) return &UA_TYPES[UA_TYPES_UINT16];
    if (data_type == x::telem::UINT32_T) return &UA_TYPES[UA_TYPES_UINT32];
    if (data_type == x::telem::UINT64_T) return &UA_TYPES[UA_TYPES_UINT64];
    if (data_type == x::telem::STRING_T) return &UA_TYPES[UA_TYPES_STRING];
    if (data_type == x::telem::TIMESTAMP_T) return &UA_TYPES[UA_TYPES_DATETIME];
    if (data_type == x::telem::UUID_T) return &UA_TYPES[UA_TYPES_GUID];
    if (data_type == x::telem::UINT8_T) return &UA_TYPES[UA_TYPES_BOOLEAN];
    return &UA_TYPES[UA_TYPES_VARIANT];
}

x::telem::Series to_series(
    const x::telem::DataType &dt,
    const std::vector<x::telem::SampleValue> &samples
) {
    if (!dt.is_variable()) {
        x::telem::Series series(dt, samples.size());
        for (const auto &sample: samples)
            series.write(sample);
        return series;
    }
    std::vector<std::string> strings;
    strings.reserve(samples.size());
    for (const auto &sample: samples)
        strings.push_back(std::get<std::string>(sample));
    return x::telem::Series(strings, dt);
}

namespace {
// A TimeStamp counts nanoseconds from 1970 in an int64, so it holds only DateTimes from
// 1677 to 2262.
constexpr UA_DateTime MIN_DATETIME = UA_DATETIME_UNIX_EPOCH +
                                     std::numeric_limits<int64_t>::min() / 100;
constexpr UA_DateTime MAX_DATETIME = UA_DATETIME_UNIX_EPOCH +
                                     std::numeric_limits<int64_t>::max() / 100;

std::pair<x::telem::TimeStamp, x::errors::Error> to_timestamp(const UA_DateTime t) {
    if (t < MIN_DATETIME || t > MAX_DATETIME)
        return {
            x::telem::TimeStamp(),
            x::errors::Error(
                x::errors::VALIDATION,
                "the DateTime is outside the years 1677 to 2262 that a timestamp holds"
            )
        };
    return {x::telem::TimeStamp((t - UA_DATETIME_UNIX_EPOCH) * 100), x::errors::NIL};
}

const std::vector NUMERIC_TYPES = {
    x::telem::FLOAT64_T,
    x::telem::FLOAT32_T,
    x::telem::INT64_T,
    x::telem::INT32_T,
    x::telem::INT16_T,
    x::telem::INT8_T,
    x::telem::UINT64_T,
    x::telem::UINT32_T,
    x::telem::UINT16_T,
    x::telem::UINT8_T,
};

/// @brief writes samples to the series. A variable-length series holds no spare room
/// for new samples, so it is built again with them.
size_t
write(x::telem::Series &series, const std::vector<x::telem::SampleValue> &samples) {
    if (!series.data_type().is_variable()) {
        size_t written = 0;
        for (const auto &sample: samples)
            written += series.write(sample);
        return written;
    }
    std::vector<x::telem::SampleValue> all;
    all.reserve(series.size() + samples.size());
    for (size_t i = 0; i < series.size(); ++i)
        all.emplace_back(series.at<std::string>(static_cast<int>(i)));
    all.insert(all.end(), samples.begin(), samples.end());
    series = to_series(series.data_type(), all);
    return samples.size();
}

std::string type_name(const UA_DataType *type) {
#ifdef UA_ENABLE_TYPEDESCRIPTION
    return type->typeName;
#else
    return "value";
#endif
}
}

std::pair<x::telem::SampleValue, x::errors::Error>
to_sample(const UA_Variant &v, const x::telem::DataType &dt) {
    if (v.type == nullptr || v.data == nullptr)
        return {{}, x::errors::Error(x::errors::VALIDATION, "the value is empty")};
    if (!UA_Variant_isScalar(&v))
        return {
            {},
            x::errors::Error(
                x::errors::VALIDATION,
                "the value is an array, but the task reads one value per node"
            )
        };
    x::telem::SampleValue sample;
    if (v.type == &UA_TYPES[UA_TYPES_DATETIME]) {
        auto [ts, err] = to_timestamp(*static_cast<const UA_DateTime *>(v.data));
        if (err) return {{}, err};
        sample = ts;
    } else if (v.type == &UA_TYPES[UA_TYPES_STRING]) {
        const auto *str = static_cast<const UA_String *>(v.data);
        sample = std::string(reinterpret_cast<const char *>(str->data), str->length);
    } else {
        const auto src = ua_to_data_type(v.type);
        if (src == x::telem::UNKNOWN_T || src == x::telem::UUID_T)
            return {
                {},
                x::errors::Error(
                    x::errors::VALIDATION,
                    "cannot convert a " + type_name(v.type) + " to " + dt.name()
                )
            };
        sample = src.cast(v.data, src);
    }
    try {
        return {dt.cast(sample), x::errors::NIL};
    } catch (const std::exception &e) {
        return {
            {},
            x::errors::Error(
                x::errors::VALIDATION,
                "cannot convert a " + type_name(v.type) + " to " + dt.name() + ": " +
                    e.what()
            )
        };
    }
}

std::pair<size_t, x::errors::Error> ua_array_write_to_series(
    x::telem::Series &series,
    const UA_Variant *val,
    const size_t target_size,
    const std::string &name
) {
    const size_t size = val->arrayLength;
    if (size != target_size) {
        const std::string verb = size < target_size ? "small" : "large";
        return {
            0,
            x::errors::Error(
                x::errors::VALIDATION,
                "OPC UA array for " + name + " is too " + verb +
                    " (size: " + std::to_string(size) +
                    ") for configured array size of " + std::to_string(target_size)
            )
        };
    }

    if (UA_Variant_isScalar(val))
        return {
            0,
            x::errors::Error(x::errors::VALIDATION, "cannot convert scalar to series")
        };

    const auto src = ua_to_data_type(val->type);
    if (src.matches(NUMERIC_TYPES) && series.data_type().matches(NUMERIC_TYPES))
        return {series.write_casted(val->data, size, src), x::errors::NIL};

    // Converts every element before it writes one, so a bad element writes nothing.
    std::vector<x::telem::SampleValue> samples;
    samples.reserve(size);
    for (size_t i = 0; i < size; ++i) {
        UA_Variant element;
        UA_Variant_setScalar(
            &element,
            static_cast<char *>(val->data) + i * val->type->memSize,
            val->type
        );
        auto [sample, err] = to_sample(element, series.data_type());
        if (err) return {0, err};
        samples.push_back(std::move(sample));
    }
    return {write(series, samples), x::errors::NIL};
}

std::pair<UA_Variant, x::errors::Error> series_to_variant(const x::telem::Series &s) {
    UA_Variant v;
    UA_Variant_init(&v);
    const auto sample = s.at(-1);
    UA_StatusCode status = UA_STATUSCODE_GOOD;
    if (s.data_type() == x::telem::TIMESTAMP_T) {
        const UA_DateTime t = std::get<x::telem::TimeStamp>(sample).nanoseconds() /
                                  100 +
                              UA_DATETIME_UNIX_EPOCH;
        status = UA_Variant_setScalarCopy(&v, &t, &UA_TYPES[UA_TYPES_DATETIME]);
    } else if (s.data_type().is_variable()) {
        const auto &str = std::get<std::string>(sample);
        const UA_String ua{
            .length = str.size(),
            .data = reinterpret_cast<UA_Byte *>(const_cast<char *>(str.data()))
        };
        status = UA_Variant_setScalarCopy(&v, &ua, &UA_TYPES[UA_TYPES_STRING]);
    } else
        status = UA_Variant_setScalarCopy(
            &v,
            cast_to_void_ptr(sample),
            data_type_to_ua(s.data_type())
        );
    return {v, opcua::errors::parse(status)};
}
}
