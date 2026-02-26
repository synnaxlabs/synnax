// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <limits>
#include <stdexcept>
#include <string>
#include <type_traits>

#include "x/cpp/date/date.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/telem/telem.h"

namespace x::json {

std::pair<TimeFormat, errors::Error> parse_time_format(const std::string &str) {
    if (str == "iso8601") return {TimeFormat::ISO8601, errors::NIL};
    if (str == "unix_sec") return {TimeFormat::UnixSecond, errors::NIL};
    if (str == "unix_ms") return {TimeFormat::UnixMillisecond, errors::NIL};
    if (str == "unix_us") return {TimeFormat::UnixMicrosecond, errors::NIL};
    if (str == "unix_ns") return {TimeFormat::UnixNanosecond, errors::NIL};
    return {
        TimeFormat::ISO8601,
        errors::Error(
            ERROR,
            "unknown time format \"" + str +
                "\": expected \"iso8601\", \"unix_sec\", \"unix_ms\","
                " \"unix_us\", or \"unix_ns\""
        ),
    };
}

namespace {

void iso_err(const std::string &reason) {
    throw std::runtime_error("not a valid ISO 8601 timestamp: " + reason);
}

telem::SampleValue parse_rfc3339(const std::string &input) {
    if (input.size() < 20) iso_err("too short (minimum 20 characters)");
    if (input[4] != '-' || input[7] != '-')
        iso_err("expected '-' at positions 4 and 7");
    if (input[10] != 'T' && input[10] != 't' && input[10] != ' ')
        iso_err("expected 'T', 't', or space at position 10");
    if (input[13] != ':' || input[16] != ':')
        iso_err("expected ':' at positions 13 and 16");

    // TODO: this doesn't need to be in line. probably a stdlib we can use.
    auto parse_uint = [&](const size_t start,
                          const size_t len) -> std::pair<int32_t, bool> {
        int32_t result = 0;
        for (size_t i = start; i < start + len; ++i) {
            if (input[i] < '0' || input[i] > '9') return {0, false};
            result = result * 10 + (input[i] - '0');
        }
        return {result, true};
    };

    const auto [year, y_ok] = parse_uint(0, 4);
    const auto [month, mo_ok] = parse_uint(5, 2);
    const auto [day, d_ok] = parse_uint(8, 2);
    const auto [hour, h_ok] = parse_uint(11, 2);
    const auto [minute, mi_ok] = parse_uint(14, 2);
    const auto [second, s_ok] = parse_uint(17, 2);
    if (!y_ok || !mo_ok || !d_ok || !h_ok || !mi_ok || !s_ok)
        iso_err("non-digit character in date or time fields");
    if (month < 1 || month > 12) iso_err("month must be between 1 and 12");
    if (day < 1 || day > 31) iso_err("day must be between 1 and 31");
    if (hour > 23) iso_err("hour must be between 0 and 23");
    if (minute > 59) iso_err("minute must be between 0 and 59");
    if (second > 59) iso_err("second must be between 0 and 59");

    int64_t frac_ns = 0;
    size_t pos = 19;
    if (pos < input.size() && input[pos] == '.') {
        ++pos;
        int64_t multiplier = 100000000;
        while (pos < input.size() && input[pos] >= '0' && input[pos] <= '9') {
            if (multiplier > 0) {
                frac_ns += (input[pos] - '0') * multiplier;
                multiplier /= 10;
            }
            ++pos;
        }
    }

    int32_t tz_offset_seconds = 0;
    if (pos >= input.size()) iso_err("missing timezone designator");
    if (input[pos] == 'Z' || input[pos] == 'z') {
        // UTC
    } else if (input[pos] == '+' || input[pos] == '-') {
        const bool negative = (input[pos] == '-');
        ++pos;
        if (pos + 5 > input.size() || input[pos + 2] != ':')
            iso_err("invalid timezone offset format");
        const auto [tz_hour, tzh_ok] = parse_uint(pos, 2);
        const auto [tz_min, tzm_ok] = parse_uint(pos + 3, 2);
        if (!tzh_ok || !tzm_ok) iso_err("non-digit character in timezone offset");
        tz_offset_seconds = (tz_hour * 3600 + tz_min * 60) * (negative ? -1 : 1);
    } else {
        iso_err(
            std::string("unexpected character \"") + input[pos] +
            "\" where timezone designator expected"
        );
    }

    const int32_t days = date::days_from_civil(
        {static_cast<uint16_t>(year),
         static_cast<uint8_t>(month),
         static_cast<uint8_t>(day)}
    );
    const int64_t total_seconds = static_cast<int64_t>(days) * 86400 + hour * 3600 +
                                  minute * 60 + second - tz_offset_seconds;
    return telem::SampleValue(telem::TimeStamp(total_seconds * 1000000000 + frac_ns));
}

template<typename T>
telem::SampleValue convert_float(const double v) {
    if constexpr (std::is_integral_v<T>) {
        if (v != std::trunc(v))
            throw std::runtime_error("value has a fractional component");
        if (v < static_cast<double>(std::numeric_limits<T>::min()) ||
            v > static_cast<double>(std::numeric_limits<T>::max()))
            throw std::runtime_error("value is out of bounds");
    }
    return telem::SampleValue(static_cast<T>(v));
}

template<typename From, typename To>
telem::SampleValue convert_integer(const From v) {
    if constexpr (std::is_same_v<From, To>) {
        return telem::SampleValue(v);
    } else if constexpr (std::is_floating_point_v<To>) {
        return telem::SampleValue(static_cast<To>(v));
    } else if constexpr (std::is_signed_v<From> && std::is_unsigned_v<To>) {
        if (v < 0) throw std::runtime_error("value is out of bounds");
        if (static_cast<std::make_unsigned_t<From>>(v) > std::numeric_limits<To>::max())
            throw std::runtime_error("value is out of bounds");
        return telem::SampleValue(static_cast<To>(v));
    } else if constexpr (std::is_unsigned_v<From> && std::is_signed_v<To>) {
        if (v > static_cast<std::make_unsigned_t<To>>(std::numeric_limits<To>::max()))
            throw std::runtime_error("value is out of bounds");
        return telem::SampleValue(static_cast<To>(v));
    } else {
        if (v < std::numeric_limits<To>::min() || v > std::numeric_limits<To>::max())
            throw std::runtime_error("value is out of bounds");
        return telem::SampleValue(static_cast<To>(v));
    }
}

template<typename V>
telem::SampleValue number_to_numeric(const V v, const telem::DataType &target) {
    constexpr bool is_float = std::is_floating_point_v<V>;
    if (target == telem::FLOAT64_T) {
        if constexpr (is_float)
            return convert_float<double>(v);
        else
            return convert_integer<V, double>(v);
    }
    if (target == telem::FLOAT32_T) {
        if constexpr (is_float)
            return convert_float<float>(v);
        else
            return convert_integer<V, float>(v);
    }
    if (target == telem::INT64_T) {
        if constexpr (is_float)
            return convert_float<int64_t>(v);
        else
            return convert_integer<V, int64_t>(v);
    }
    if (target == telem::INT32_T) {
        if constexpr (is_float)
            return convert_float<int32_t>(v);
        else
            return convert_integer<V, int32_t>(v);
    }
    if (target == telem::INT16_T) {
        if constexpr (is_float)
            return convert_float<int16_t>(v);
        else
            return convert_integer<V, int16_t>(v);
    }
    if (target == telem::INT8_T) {
        if constexpr (is_float)
            return convert_float<int8_t>(v);
        else
            return convert_integer<V, int8_t>(v);
    }
    if (target == telem::UINT64_T) {
        if constexpr (is_float)
            return convert_float<uint64_t>(v);
        else
            return convert_integer<V, uint64_t>(v);
    }
    if (target == telem::UINT32_T) {
        if constexpr (is_float)
            return convert_float<uint32_t>(v);
        else
            return convert_integer<V, uint32_t>(v);
    }
    if (target == telem::UINT16_T) {
        if constexpr (is_float)
            return convert_float<uint16_t>(v);
        else
            return convert_integer<V, uint16_t>(v);
    }
    if (target == telem::UINT8_T) {
        if constexpr (is_float)
            return convert_float<uint8_t>(v);
        else
            return convert_integer<V, uint8_t>(v);
    }
    throw std::runtime_error("");
}

/// @brief returns true if the string contains only digits with an optional leading
/// sign character ('+' or '-').
bool is_integer_string(const std::string &str) {
    if (str.empty()) return false;
    const size_t start = (str[0] == '-' || str[0] == '+') ? 1 : 0;
    if (start >= str.size()) return false;
    for (size_t i = start; i < str.size(); ++i)
        if (str[i] < '0' || str[i] > '9') return false;
    return true;
}

double parse_string_double(const std::string &str) {
    size_t pos = 0;
    double v;
    v = std::stod(str, &pos);
    if (pos != str.size()) throw std::runtime_error("not a valid number");
    return v;
}

int64_t parse_string_int64(const std::string &str) {
    size_t pos = 0;
    int64_t v;
    v = std::stoll(str, &pos);
    if (pos != str.size()) throw std::runtime_error("not a valid number");
    return v;
}

uint64_t parse_string_uint64(const std::string &str) {
    size_t pos = 0;
    uint64_t v;
    v = std::stoull(str, &pos);
    if (pos != str.size()) throw std::runtime_error("not a valid number");
    return v;
}

/// @brief converts a string to a numeric SampleValue. Uses integer parsing (stoll/
/// stoull) for pure integer strings targeting integer types to preserve full int64/
/// uint64 precision, and falls back to stod for float targets and strings containing
/// decimal points or exponents.
telem::SampleValue
string_to_numeric(const std::string &str, const telem::DataType &target) {
    if (str.empty()) throw std::runtime_error("not a valid number");

    // Float targets always parse as double.
    if (target == telem::FLOAT64_T || target == telem::FLOAT32_T)
        return number_to_numeric(parse_string_double(str), target);

    // For integer targets with pure integer strings, use integer parsing
    // to avoid precision loss through double (matters for int64/uint64).
    if (is_integer_string(str)) {
        const bool is_negative = str[0] == '-';
        const bool is_signed = target == telem::INT64_T || target == telem::INT32_T ||
                               target == telem::INT16_T || target == telem::INT8_T;
        if (is_negative || is_signed)
            return number_to_numeric(parse_string_int64(str), target);
        return number_to_numeric(parse_string_uint64(str), target);
    }

    // Non-integer strings (decimals, exponents) fall back to stod.
    return number_to_numeric(parse_string_double(str), target);
}

telem::SampleValue number_to_timestamp(const double v, TimeFormat time_format) {
    switch (time_format) {
        case TimeFormat::UnixNanosecond:
            return telem::SampleValue(telem::TimeStamp(static_cast<int64_t>(v)));
        case TimeFormat::UnixMicrosecond:
            return telem::SampleValue(telem::TimeStamp(static_cast<int64_t>(v * 1e3)));
        case TimeFormat::UnixMillisecond:
            return telem::SampleValue(telem::TimeStamp(static_cast<int64_t>(v * 1e6)));
        case TimeFormat::UnixSecond:
            return telem::SampleValue(telem::TimeStamp(static_cast<int64_t>(v * 1e9)));
        case TimeFormat::ISO8601:
            throw std::runtime_error(
                "numeric values cannot be converted with ISO 8601 format"
            );
    }
    throw std::runtime_error("unknown TimeFormat");
}

/// @brief performs the actual conversion, throwing on failure.
telem::SampleValue convert(
    const nlohmann::json &value,
    const telem::DataType &target,
    TimeFormat time_format,
    const EnumMap *enum_values
) {
    if (target == telem::TIMESTAMP_T) {
        if (value.is_number())
            return number_to_timestamp(value.get<double>(), time_format);
        if (value.is_string()) {
            if (time_format == TimeFormat::ISO8601)
                return parse_rfc3339(value.get<std::string>());
            return number_to_timestamp(
                parse_string_double(value.get_ref<const std::string &>()),
                time_format
            );
        }
        throw std::runtime_error("");
    }

    if (target == telem::STRING_T) {
        if (value.is_string()) return telem::SampleValue(value.get<std::string>());
        return telem::SampleValue(value.dump());
    }

    if (value.is_boolean()) return number_to_numeric(value.get<bool>() ? 1 : 0, target);
    if (value.is_number_unsigned())
        return number_to_numeric(value.get<uint64_t>(), target);
    if (value.is_number_integer())
        return number_to_numeric(value.get<int64_t>(), target);
    if (value.is_number()) return number_to_numeric(value.get<double>(), target);
    if (value.is_string()) {
        const auto &str = value.get_ref<const std::string &>();
        if (enum_values != nullptr) {
            auto it = enum_values->find(str);
            if (it != enum_values->end())
                return number_to_numeric(it->second, target);
        }
        return string_to_numeric(str, target);
    }

    throw std::runtime_error("");
}

}

std::pair<telem::SampleValue, errors::Error> to_sample_value(
    const nlohmann::json &value,
    const telem::DataType &target,
    TimeFormat time_format,
    const EnumMap *enum_values
) {
    try {
        return {convert(value, target, time_format, enum_values), errors::NIL};
    } catch (const std::exception &e) {
        auto msg = std::string("cannot convert ") + value.dump() + " to " +
                   target.name();
        auto details = std::string(e.what());
        if (details.size() > 0) msg += ": " + details;
        return {
            telem::SampleValue(),
            errors::Error(CONVERSION_ERROR, msg),
        };
    }
}

bool check_to_sample_value(const telem::DataType &target) {
    return target == telem::FLOAT64_T || target == telem::FLOAT32_T ||
           target == telem::INT64_T || target == telem::INT32_T ||
           target == telem::INT16_T || target == telem::INT8_T ||
           target == telem::UINT64_T || target == telem::UINT32_T ||
           target == telem::UINT16_T || target == telem::UINT8_T ||
           target == telem::TIMESTAMP_T || target == telem::STRING_T;
}

std::pair<nlohmann::json, errors::Error>
from_sample_value(const telem::SampleValue &value, Type target) {
    return std::visit(
        [target](const auto &v) -> std::pair<nlohmann::json, errors::Error> {
            using T = std::decay_t<decltype(v)>;
            if constexpr (std::is_same_v<T, std::string>) {
                if (target == Type::String) return {nlohmann::json(v), errors::NIL};
                return {nlohmann::json(), CONVERSION_ERROR};
            } else if constexpr (std::is_same_v<T, telem::TimeStamp>) {
                return {nlohmann::json(), CONVERSION_ERROR};
            } else {
                switch (target) {
                    case Type::Number:
                        return {nlohmann::json(v), errors::NIL};
                    case Type::String: {
                        return {nlohmann::json(std::format("{}", v)), errors::NIL};
                    }
                    case Type::Boolean:
                        return {nlohmann::json(v != 0), errors::NIL};
                }
                return {nlohmann::json(), CONVERSION_ERROR};
            }
        },
        value
    );
}

errors::Error check_from_sample_value(const telem::DataType &type, Type target) {
    if (type == telem::STRING_T)
        return target == Type::String ? errors::NIL : CONVERSION_ERROR;
    if (type == telem::FLOAT64_T || type == telem::FLOAT32_T ||
        type == telem::INT64_T || type == telem::INT32_T || type == telem::INT16_T ||
        type == telem::INT8_T || type == telem::UINT64_T || type == telem::UINT32_T ||
        type == telem::UINT16_T || type == telem::UINT8_T)
        return errors::NIL;
    return CONVERSION_ERROR;
}

nlohmann::json from_timestamp(telem::TimeStamp ts, TimeFormat format) {
    switch (format) {
        case TimeFormat::UnixNanosecond:
            return ts.nanoseconds();
        case TimeFormat::UnixMicrosecond:
            return static_cast<double>(ts.nanoseconds()) / 1e3;
        case TimeFormat::UnixMillisecond:
            return static_cast<double>(ts.nanoseconds()) / 1e6;
        case TimeFormat::UnixSecond:
            return static_cast<double>(ts.nanoseconds()) / 1e9;
        case TimeFormat::ISO8601:
            return ts.iso8601();
        default:
            throw std::runtime_error(
                "unexpected TimeFormat value: " +
                std::to_string(static_cast<int>(format))
            );
    }
}

nlohmann::json zero_value(Type format) {
    switch (format) {
        case Type::Number:
            return 0;
        case Type::String:
            return "";
        case Type::Boolean:
            return false;
        default:
            throw std::runtime_error(
                "unexpected Type value: " + std::to_string(static_cast<int>(format))
            );
    }
}

}
