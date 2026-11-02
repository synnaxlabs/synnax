// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cctype>
#include <charconv>
#include <cmath>
#include <cstring>
#include <format>
#include <limits>
#include <locale>
#include <sstream>
#include <string>
#include <type_traits>
#include <utility>
#include <vector>

#include "open62541/types.h"

#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/opcua/errors/errors.h"
#include "driver/opcua/telem/telem.h"

namespace driver::opcua::telem {
x::telem::DataType to_data_type(const UA_DataType *dt) {
    if (dt == nullptr) return x::telem::UNKNOWN_T;
    switch (dt->typeKind) {
        case UA_DATATYPEKIND_BOOLEAN:
            return x::telem::BOOLEAN_T;
        case UA_DATATYPEKIND_SBYTE:
            return x::telem::INT8_T;
        case UA_DATATYPEKIND_BYTE:
            return x::telem::UINT8_T;
        case UA_DATATYPEKIND_INT16:
            return x::telem::INT16_T;
        case UA_DATATYPEKIND_UINT16:
            return x::telem::UINT16_T;
        case UA_DATATYPEKIND_INT32:
        case UA_DATATYPEKIND_ENUM:
            return x::telem::INT32_T;
        case UA_DATATYPEKIND_UINT32:
            return x::telem::UINT32_T;
        case UA_DATATYPEKIND_INT64:
            return x::telem::INT64_T;
        case UA_DATATYPEKIND_UINT64:
            return x::telem::UINT64_T;
        case UA_DATATYPEKIND_FLOAT:
            return x::telem::FLOAT32_T;
        case UA_DATATYPEKIND_DOUBLE:
            return x::telem::FLOAT64_T;
        case UA_DATATYPEKIND_STRING:
            return x::telem::STRING_T;
        case UA_DATATYPEKIND_DATETIME:
            return x::telem::TIMESTAMP_T;
        case UA_DATATYPEKIND_GUID:
            return x::telem::UUID_T;
        case UA_DATATYPEKIND_BYTESTRING:
            return x::telem::BYTES_T;
        default:
            return x::telem::UNKNOWN_T;
    }
}

x::telem::DataType to_data_type(const UA_NodeId &data_type) {
    return to_data_type(UA_findDataType(&data_type));
}

UA_DataType *data_type_to_ua(const x::telem::DataType &data_type) {
    if (data_type == x::telem::FLOAT32_T) return &UA_TYPES[UA_TYPES_FLOAT];
    if (data_type == x::telem::FLOAT64_T) return &UA_TYPES[UA_TYPES_DOUBLE];
    if (data_type == x::telem::INT8_T) return &UA_TYPES[UA_TYPES_SBYTE];
    if (data_type == x::telem::INT16_T) return &UA_TYPES[UA_TYPES_INT16];
    if (data_type == x::telem::INT32_T) return &UA_TYPES[UA_TYPES_INT32];
    if (data_type == x::telem::INT64_T) return &UA_TYPES[UA_TYPES_INT64];
    if (data_type == x::telem::UINT8_T) return &UA_TYPES[UA_TYPES_BYTE];
    if (data_type == x::telem::UINT16_T) return &UA_TYPES[UA_TYPES_UINT16];
    if (data_type == x::telem::UINT32_T) return &UA_TYPES[UA_TYPES_UINT32];
    if (data_type == x::telem::UINT64_T) return &UA_TYPES[UA_TYPES_UINT64];
    if (data_type == x::telem::STRING_T) return &UA_TYPES[UA_TYPES_STRING];
    if (data_type == x::telem::TIMESTAMP_T) return &UA_TYPES[UA_TYPES_DATETIME];
    if (data_type == x::telem::UUID_T) return &UA_TYPES[UA_TYPES_GUID];
    if (data_type == x::telem::BYTES_T) return &UA_TYPES[UA_TYPES_BYTESTRING];
    if (data_type == x::telem::BOOLEAN_T) return &UA_TYPES[UA_TYPES_BOOLEAN];
    return &UA_TYPES[UA_TYPES_VARIANT];
}

namespace {
constexpr size_t UUID_SIZE = 16;

/// @returns the samples of a series of a variable-length data type.
std::vector<std::string> variable_samples(const x::telem::Series &series) {
    std::vector<std::string> samples;
    samples.reserve(series.size());
    const auto *ptr = series.data();
    const auto *end = ptr + series.byte_size();
    while (ptr + VARIABLE_LENGTH_PREFIX_SIZE <= end) {
        const uint32_t len = read_u32_le(ptr);
        ptr += VARIABLE_LENGTH_PREFIX_SIZE;
        samples.emplace_back(reinterpret_cast<const char *>(ptr), len);
        ptr += len;
    }
    return samples;
}

/// @returns the 16 bytes of the UUID that guid holds, in RFC 4122 order.
std::string uuid_bytes(const UA_Guid &guid) {
    std::string bytes(UUID_SIZE, '\0');
    for (int i = 0; i < 4; ++i)
        bytes[i] = static_cast<char>(guid.data1 >> (24 - 8 * i));
    for (int i = 0; i < 2; ++i) {
        bytes[4 + i] = static_cast<char>(guid.data2 >> (8 - 8 * i));
        bytes[6 + i] = static_cast<char>(guid.data3 >> (8 - 8 * i));
    }
    std::memcpy(bytes.data() + 8, guid.data4, 8);
    return bytes;
}

/// @returns the Guid that the 16 bytes of a UUID in RFC 4122 order hold.
UA_Guid to_guid(const std::byte *bytes) {
    const auto at = [bytes](const int i) {
        return static_cast<UA_UInt32>(static_cast<uint8_t>(bytes[i]));
    };
    UA_Guid guid;
    guid.data1 = at(0) << 24 | at(1) << 16 | at(2) << 8 | at(3);
    guid.data2 = static_cast<UA_UInt16>(at(4) << 8 | at(5));
    guid.data3 = static_cast<UA_UInt16>(at(6) << 8 | at(7));
    std::memcpy(guid.data4, bytes + 8, 8);
    return guid;
}

std::string guid_text(const UA_Guid &guid) {
    UA_String out = UA_STRING_NULL;
    if (UA_Guid_print(&guid, &out) != UA_STATUSCODE_GOOD) throw std::bad_alloc();
    std::string text(reinterpret_cast<const char *>(out.data), out.length);
    UA_String_clear(&out);
    return text;
}

/// @brief writes one uuid sample, which holds the 16 bytes of the UUID.
size_t write_uuid(x::telem::Series &series, const x::telem::SampleValue &sample) {
    const auto &bytes = std::get<std::string>(sample);
    return series.write(reinterpret_cast<const uint8_t *>(bytes.data()), 1);
}
}

x::telem::Series to_series(
    const x::telem::DataType &dt,
    const std::vector<x::telem::SampleValue> &samples
) {
    if (!dt.is_variable()) {
        x::telem::Series series(dt, samples.size());
        for (const auto &sample: samples)
            if (dt == x::telem::UUID_T)
                write_uuid(series, sample);
            else
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
}

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

UA_DateTime to_date_time(const x::telem::TimeStamp ts) {
    return ts.nanoseconds() / 100 + UA_DATETIME_UNIX_EPOCH;
}

namespace {
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
    const auto &dt = series.data_type();
    if (!dt.is_variable()) {
        size_t written = 0;
        for (const auto &sample: samples)
            written += dt == x::telem::UUID_T ? write_uuid(series, sample)
                                              : series.write(sample);
        return written;
    }
    std::vector<x::telem::SampleValue> all;
    all.reserve(series.size() + samples.size());
    for (auto &sample: variable_samples(series))
        all.emplace_back(std::move(sample));
    all.insert(all.end(), samples.begin(), samples.end());
    series = to_series(dt, all);
    return samples.size();
}

std::string type_name(const UA_DataType *type) {
    return type->typeName;
}

x::errors::Error cannot_convert(const UA_DataType *type, const x::telem::DataType &dt) {
    return x::errors::Error(
        x::errors::VALIDATION,
        "cannot convert a " + type_name(type) + " to " + dt.name()
    );
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
    switch (v.type->typeKind) {
        case UA_DATATYPEKIND_DATETIME: {
            auto [ts, err] = to_timestamp(*static_cast<const UA_DateTime *>(v.data));
            if (err) return {{}, err};
            sample = ts;
            break;
        }
        case UA_DATATYPEKIND_STRING: {
            const auto *str = static_cast<const UA_String *>(v.data);
            sample = std::string(
                reinterpret_cast<const char *>(str->data),
                str->length
            );
            break;
        }
        case UA_DATATYPEKIND_BYTESTRING: {
            if (dt != x::telem::BYTES_T) return {{}, cannot_convert(v.type, dt)};
            const auto *bytes = static_cast<const UA_ByteString *>(v.data);
            return {
                std::string(reinterpret_cast<const char *>(bytes->data), bytes->length),
                x::errors::NIL
            };
        }
        case UA_DATATYPEKIND_GUID: {
            const auto &guid = *static_cast<const UA_Guid *>(v.data);
            if (dt == x::telem::UUID_T) return {uuid_bytes(guid), x::errors::NIL};
            if (dt == x::telem::STRING_T) return {guid_text(guid), x::errors::NIL};
            return {{}, cannot_convert(v.type, dt)};
        }
        default: {
            const auto src = to_data_type(v.type);
            if (src == x::telem::UNKNOWN_T) return {{}, cannot_convert(v.type, dt)};
            sample = src.cast(v.data, src);
        }
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

    const auto src = to_data_type(val->type);
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

namespace {
/// @brief converts the last sample of a non-empty series to a variant of the matching
/// OPC UA type.
std::pair<UA_Variant, x::errors::Error> series_to_variant(const x::telem::Series &s) {
    UA_Variant v;
    UA_Variant_init(&v);
    const auto &dt = s.data_type();
    UA_StatusCode status = UA_STATUSCODE_GOOD;
    if (dt == x::telem::UUID_T) {
        const auto guid = to_guid(s.data() + (s.size() - 1) * UUID_SIZE);
        status = UA_Variant_setScalarCopy(&v, &guid, &UA_TYPES[UA_TYPES_GUID]);
    } else if (dt == x::telem::BYTES_T) {
        const auto bytes = variable_samples(s).back();
        const UA_ByteString ua{
            .length = bytes.size(),
            .data = reinterpret_cast<UA_Byte *>(const_cast<char *>(bytes.data()))
        };
        status = UA_Variant_setScalarCopy(&v, &ua, &UA_TYPES[UA_TYPES_BYTESTRING]);
    } else if (dt == x::telem::TIMESTAMP_T) {
        const auto sample = s.at(-1);
        const UA_DateTime t = to_date_time(std::get<x::telem::TimeStamp>(sample));
        status = UA_Variant_setScalarCopy(&v, &t, &UA_TYPES[UA_TYPES_DATETIME]);
    } else if (dt.is_variable()) {
        const auto str = std::get<std::string>(s.at(-1));
        const UA_String ua{
            .length = str.size(),
            .data = reinterpret_cast<UA_Byte *>(const_cast<char *>(str.data()))
        };
        status = UA_Variant_setScalarCopy(&v, &ua, &UA_TYPES[UA_TYPES_STRING]);
    } else if (dt == x::telem::BOOLEAN_T) {
        const UA_Boolean b = std::get<uint8_t>(s.at(-1)) != 0;
        status = UA_Variant_setScalarCopy(&v, &b, &UA_TYPES[UA_TYPES_BOOLEAN]);
    } else {
        const auto sample = s.at(-1);
        status = UA_Variant_setScalarCopy(
            &v,
            cast_to_void_ptr(sample),
            data_type_to_ua(dt)
        );
    }
    return {v, opcua::errors::parse(status)};
}

std::string describe(const x::telem::SampleValue &sample) {
    return std::visit(
        []<typename T>(const T &v) -> std::string {
            if constexpr (std::is_same_v<T, std::string>)
                return "\"" + v + "\"";
            else if constexpr (std::is_same_v<T, x::telem::TimeStamp>)
                return "the timestamp " + std::to_string(v.nanoseconds());
            else if constexpr (std::is_floating_point_v<T>)
                return std::format("{}", v);
            else
                return std::to_string(+v);
        },
        sample
    );
}

x::errors::Error
conversion_error(const x::telem::SampleValue &sample, const UA_DataType *target) {
    const auto value = describe(sample);
    const auto type = type_name(target);
    std::string reason;
    if (std::holds_alternative<std::string>(sample))
        reason = value + " is not a valid " + type;
    else if (std::holds_alternative<x::telem::TimeStamp>(sample))
        reason = "cannot write " + value + " to a " + type + " node";
    else
        reason = value + " is out of range for a " + type + " node";
    return x::errors::Error(x::errors::VALIDATION, reason);
}

/// @returns whether the sample is a finite number with a fraction.
bool fractional(const x::telem::SampleValue &sample) {
    return std::visit(
        []<typename V>(const V &v) {
            if constexpr (std::is_floating_point_v<V>)
                return std::isfinite(v) && std::trunc(v) != v;
            else
                return false;
        },
        sample
    );
}

/// @returns the text with its letters in lower case.
std::string lower(std::string text) {
    std::ranges::transform(text, text.begin(), [](const unsigned char c) {
        return static_cast<char>(std::tolower(c));
    });
    return text;
}

std::optional<double> parse_double(const std::string &text) {
    std::istringstream in(text);
    in.imbue(std::locale::classic());
    double value = 0;
    in >> value;
    if (in.fail() || !in.eof()) return std::nullopt;
    return value;
}

/// @returns the sample as an integer of type T, or nothing when T cannot hold it
/// exactly.
template<typename T>
std::optional<T> to_integer(const x::telem::SampleValue &sample) {
    return std::visit(
        []<typename V>(const V &v) -> std::optional<T> {
            if constexpr (std::is_integral_v<V>) {
                if (!std::in_range<T>(v)) return std::nullopt;
                return static_cast<T>(v);
            } else if constexpr (std::is_floating_point_v<V>) {
                // 2^digits is the first value past the range of T, and is exact.
                const double end = std::ldexp(1.0, std::numeric_limits<T>::digits);
                const double start = std::is_signed_v<T> ? -end : 0;
                const auto d = static_cast<double>(v);
                if (!std::isfinite(d) || std::trunc(d) != d || d < start || d >= end)
                    return std::nullopt;
                return static_cast<T>(d);
            } else if constexpr (std::is_same_v<V, std::string>) {
                T value{};
                const auto *end = v.data() + v.size();
                const auto [ptr, ec] = std::from_chars(v.data(), end, value);
                if (ec != std::errc() || ptr != end) return std::nullopt;
                return value;
            } else
                return std::nullopt;
        },
        sample
    );
}

std::optional<double> to_double(const x::telem::SampleValue &sample) {
    return std::visit(
        []<typename V>(const V &v) -> std::optional<double> {
            if constexpr (std::is_arithmetic_v<V>)
                return static_cast<double>(v);
            else if constexpr (std::is_same_v<V, std::string>)
                return parse_double(v);
            else
                return std::nullopt;
        },
        sample
    );
}

std::optional<UA_Boolean> to_boolean(const x::telem::SampleValue &sample) {
    if (const auto *text = std::get_if<std::string>(&sample)) {
        const auto value = lower(*text);
        if (value == "true" || value == "1") return true;
        if (value == "false" || value == "0") return false;
        return std::nullopt;
    }
    const auto value = to_integer<uint8_t>(sample);
    if (!value.has_value() || *value > 1) return std::nullopt;
    return *value == 1;
}

/// @brief the sample that a command converts, and the type of its node.
struct Conversion {
    const x::telem::SampleValue &sample;
    const UA_DataType *target;
    const UA_DataType *builtin;

    /// @returns a variant that holds value, or the conversion error when value is
    /// empty.
    template<typename T>
    [[nodiscard]] std::pair<UA_Variant, x::errors::Error>
    scalar(const std::optional<T> &value) const {
        UA_Variant v;
        UA_Variant_init(&v);
        if (!value.has_value()) return {v, conversion_error(sample, target)};
        const auto status = UA_Variant_setScalarCopy(&v, &*value, builtin);
        return {v, opcua::errors::parse(status)};
    }

    template<typename T>
    [[nodiscard]] std::pair<UA_Variant, x::errors::Error> integer() const {
        if (!fractional(sample)) return this->scalar(to_integer<T>(sample));
        UA_Variant v;
        UA_Variant_init(&v);
        return {
            v,
            x::errors::Error(
                x::errors::VALIDATION,
                describe(sample) + " is not a whole number, so a " + type_name(target) +
                    " node cannot hold it"
            )
        };
    }
};

/// @brief converts the sample to the builtin type of a node.
std::pair<UA_Variant, x::errors::Error> sample_to_variant(const Conversion &c) {
    const auto &sample = c.sample;
    switch (c.builtin->typeKind) {
        case UA_DATATYPEKIND_BOOLEAN:
            return c.scalar(to_boolean(sample));
        case UA_DATATYPEKIND_SBYTE:
            return c.integer<UA_SByte>();
        case UA_DATATYPEKIND_BYTE:
            return c.integer<UA_Byte>();
        case UA_DATATYPEKIND_INT16:
            return c.integer<UA_Int16>();
        case UA_DATATYPEKIND_UINT16:
            return c.integer<UA_UInt16>();
        case UA_DATATYPEKIND_INT32:
            return c.integer<UA_Int32>();
        case UA_DATATYPEKIND_UINT32:
            return c.integer<UA_UInt32>();
        case UA_DATATYPEKIND_INT64:
            return c.integer<UA_Int64>();
        case UA_DATATYPEKIND_UINT64:
            return c.integer<UA_UInt64>();
        case UA_DATATYPEKIND_DOUBLE:
            return c.scalar(to_double(sample));
        case UA_DATATYPEKIND_FLOAT: {
            const auto d = to_double(sample);
            constexpr auto max = std::numeric_limits<float>::max();
            if (d.has_value() && std::isfinite(*d) && std::abs(*d) > max)
                return c.scalar(std::optional<UA_Float>());
            return c.scalar(
                d.has_value() ? std::optional(static_cast<UA_Float>(*d)) : std::nullopt
            );
        }
        case UA_DATATYPEKIND_STRING: {
            if (std::holds_alternative<x::telem::TimeStamp>(sample))
                return c.scalar(std::optional<UA_String>());
            const auto str = std::holds_alternative<std::string>(sample)
                               ? std::get<std::string>(sample)
                               : describe(sample);
            const UA_String ua{
                .length = str.size(),
                .data = reinterpret_cast<UA_Byte *>(const_cast<char *>(str.data()))
            };
            return c.scalar(std::optional(ua));
        }
        case UA_DATATYPEKIND_BYTESTRING: {
            const auto *str = std::get_if<std::string>(&sample);
            if (str == nullptr) return c.scalar(std::optional<UA_ByteString>());
            const UA_ByteString ua{
                .length = str->size(),
                .data = reinterpret_cast<UA_Byte *>(const_cast<char *>(str->data()))
            };
            return c.scalar(std::optional(ua));
        }
        case UA_DATATYPEKIND_GUID: {
            const auto *str = std::get_if<std::string>(&sample);
            UA_Guid guid;
            if (str == nullptr ||
                UA_Guid_parse(&guid, UA_STRING(const_cast<char *>(str->c_str()))) !=
                    UA_STATUSCODE_GOOD)
                return c.scalar(std::optional<UA_Guid>());
            return c.scalar(std::optional(guid));
        }
        case UA_DATATYPEKIND_DATETIME: {
            const auto *ts = std::get_if<x::telem::TimeStamp>(&sample);
            if (ts == nullptr) return c.scalar(std::optional<UA_DateTime>());
            return c.scalar(
                std::optional<UA_DateTime>(
                    ts->nanoseconds() / 100 + UA_DATETIME_UNIX_EPOCH
                )
            );
        }
        default:
            return {
                UA_Variant{},
                x::errors::Error(
                    x::errors::VALIDATION,
                    "cannot write to a " + type_name(c.target) + " node"
                )
            };
    }
}
}

std::pair<UA_Variant, x::errors::Error>
to_variant(const x::telem::Series &s, const UA_DataType *target) {
    if (s.empty())
        return {
            UA_Variant{},
            x::errors::Error(x::errors::VALIDATION, "the command has no value")
        };
    // A node of BaseDataType has the Variant type and takes a value of any type.
    if (target == nullptr || target->typeKind == UA_DATATYPEKIND_VARIANT)
        return series_to_variant(s);
    const auto kind = target->typeKind;
    if (kind > UA_DATATYPEKIND_DIAGNOSTICINFO && kind != UA_DATATYPEKIND_ENUM)
        return {
            UA_Variant{},
            x::errors::Error(
                x::errors::VALIDATION,
                "cannot write to a " + type_name(target) + " node"
            )
        };
    // The builtin types come first in UA_TYPES, in the order of their kinds.
    const auto *builtin = kind == UA_DATATYPEKIND_ENUM ? &UA_TYPES[UA_TYPES_INT32]
                                                       : &UA_TYPES[kind];
    const auto &dt = s.data_type();
    if (dt == x::telem::UUID_T || dt == x::telem::BYTES_T) {
        if (builtin->typeKind != (dt == x::telem::UUID_T ? UA_DATATYPEKIND_GUID
                                                         : UA_DATATYPEKIND_BYTESTRING))
            return {
                UA_Variant{},
                x::errors::Error(
                    x::errors::VALIDATION,
                    "cannot write a " + dt.name() + " command to a " +
                        type_name(target) + " node"
                )
            };
        return series_to_variant(s);
    }
    const auto sample = s.at(-1);
    return sample_to_variant({.sample = sample, .target = target, .builtin = builtin});
}

namespace {
/// @returns one value of a type as JSON text.
std::pair<std::string, x::errors::Error>
json(const void *data, const UA_DataType *type) {
    UA_EncodeJsonOptions options{};
    options.stringNodeIds = true;
    UA_ByteString out = UA_BYTESTRING_NULL;
    if (const auto status = UA_encodeJson(data, type, &out, &options);
        status != UA_STATUSCODE_GOOD)
        return {
            "",
            x::errors::Error(
                x::errors::VALIDATION,
                std::string("the value does not encode: ") + UA_StatusCode_name(status)
            )
        };
    std::string text(reinterpret_cast<const char *>(out.data), out.length);
    UA_ByteString_clear(&out);
    return {std::move(text), x::errors::NIL};
}
}

std::pair<std::string, x::errors::Error> to_json(const UA_Variant &value) {
    if (value.type == nullptr) return {"null", x::errors::NIL};
    if (UA_Variant_isScalar(&value)) return json(value.data, value.type);
    std::string text = "[";
    const auto *data = static_cast<const char *>(value.data);
    for (std::size_t i = 0; i < value.arrayLength; ++i) {
        auto [element, err] = json(data + i * value.type->memSize, value.type);
        if (err) return {"", err};
        if (i > 0) text += ", ";
        text += element;
    }
    return {text + "]", x::errors::NIL};
}
}
