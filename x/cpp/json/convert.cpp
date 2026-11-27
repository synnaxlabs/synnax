// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <cstddef>
#include <limits>
#include <stdexcept>
#include <string>
#include <type_traits>
#include <utility>
#include <variant>

#include "x/cpp/base64/base64.h"
#include "x/cpp/date/date.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

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
            BASE_ERROR,
            "unknown time format \"" + str +
                "\": expected \"iso8601\", \"unix_sec\", \"unix_ms\","
                " \"unix_us\", or \"unix_ns\""
        ),
    };
}

namespace {

constexpr int64_t NS = 1000000000;
constexpr int64_t MAX_SECONDS = std::numeric_limits<int64_t>::max() / NS;
constexpr int64_t MIN_SECONDS = std::numeric_limits<int64_t>::min() / NS;
constexpr int64_t MAX_FRAC_NS = std::numeric_limits<int64_t>::max() % NS;
constexpr int64_t MIN_FRAC_NS = std::numeric_limits<int64_t>::min() % NS;

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

    const date::Date civil{
        static_cast<uint16_t>(year),
        static_cast<uint8_t>(month),
        static_cast<uint8_t>(day)
    };
    const int32_t days = date::days_from_civil(civil);
    if (const auto [_, m, d] = date::civil_from_days(days);
        m != civil.month || d != civil.day)
        iso_err("day must be a day of the month");
    const int64_t total_seconds = static_cast<int64_t>(days) * 86400 +
                                  static_cast<int64_t>(hour * 3600) +
                                  static_cast<int64_t>(minute * 60) + second -
                                  tz_offset_seconds;
    // A negative fraction keeps whole * NS + part inside the int64_t range.
    const bool negative = total_seconds < 0;
    const int64_t whole = negative ? total_seconds + 1 : total_seconds;
    const int64_t part = negative ? frac_ns - NS : frac_ns;
    if (whole > MAX_SECONDS || whole < MIN_SECONDS ||
        (whole == MAX_SECONDS && part > MAX_FRAC_NS) ||
        (whole == MIN_SECONDS && part < MIN_FRAC_NS))
        iso_err("outside the range of a timestamp");
    return telem::SampleValue(telem::TimeStamp(whole * NS + part));
}

template<typename T>
telem::SampleValue convert_float(const double v) {
    if constexpr (std::is_integral_v<T>) {
        if (v != std::trunc(v))
            throw std::runtime_error("value has a fractional component");
        // The maximum of a 64-bit type rounds up to 2^digits as a double, so the
        // bound is the exclusive 2^digits.
        if (v < static_cast<double>(std::numeric_limits<T>::min()) ||
            v >= std::ldexp(1.0, std::numeric_limits<T>::digits))
            throw std::runtime_error("value is out of bounds");
    } else if constexpr (std::is_same_v<T, float>) {
        if (std::isfinite(v) && std::abs(v) > std::numeric_limits<float>::max())
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

/// @brief converts the sample of an enum entry to target, throwing when an integer
/// target cannot hold it exactly. A float target takes the nearest value, and throws
/// past its range.
telem::SampleValue
enum_to_sample(const telem::SampleValue &sample, const telem::DataType &target) {
    return std::visit(
        [&target](const auto &v) -> telem::SampleValue {
            using T = std::decay_t<decltype(v)>;
            if constexpr (std::is_same_v<T, std::string>) {
                if (target == telem::STRING_T) return telem::SampleValue(v);
                throw std::runtime_error("a string enum value is not a number");
            } else if constexpr (std::is_same_v<T, telem::TimeStamp>) {
                throw std::runtime_error("a timestamp enum value is not a number");
            } else {
                if (target == telem::STRING_T)
                    throw std::runtime_error("a numeric enum value is not a string");
                return number_to_numeric(v, target);
            }
        },
        sample
    );
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
    double v = NAN;
    v = std::stod(str, &pos);
    if (pos != str.size()) throw std::runtime_error("not a valid number");
    return v;
}

int64_t parse_string_int64(const std::string &str) {
    size_t pos = 0;
    int64_t v = 0;
    v = std::stoll(str, &pos);
    if (pos != str.size()) throw std::runtime_error("not a valid number");
    return v;
}

uint64_t parse_string_uint64(const std::string &str) {
    size_t pos = 0;
    uint64_t v = 0;
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

int64_t nanoseconds_per_unit(const TimeFormat time_format) {
    switch (time_format) {
        case TimeFormat::UnixNanosecond:
            return 1;
        case TimeFormat::UnixMicrosecond:
            return 1000;
        case TimeFormat::UnixMillisecond:
            return 1000000;
        case TimeFormat::UnixSecond:
            return 1000000000;
        case TimeFormat::ISO8601:
            throw std::runtime_error(
                "numeric values cannot be converted with ISO 8601 format"
            );
    }
    throw std::runtime_error("unknown TimeFormat");
}

telem::SampleValue number_to_timestamp(const double v, TimeFormat time_format) {
    const double ns = v * static_cast<double>(nanoseconds_per_unit(time_format));
    if (std::isnan(ns) || ns < -std::ldexp(1.0, 63) || ns >= std::ldexp(1.0, 63))
        throw std::runtime_error("value is out of bounds");
    return telem::SampleValue(telem::TimeStamp(static_cast<int64_t>(ns)));
}

telem::SampleValue integer_to_timestamp(const int64_t v, TimeFormat time_format) {
    const int64_t unit = nanoseconds_per_unit(time_format);
    if (v > std::numeric_limits<int64_t>::max() / unit ||
        v < std::numeric_limits<int64_t>::min() / unit)
        throw std::runtime_error("value is out of bounds");
    return telem::SampleValue(telem::TimeStamp(v * unit));
}

/// @brief performs the actual conversion, throwing on failure.
telem::SampleValue convert(
    const nlohmann::json &value,
    const telem::DataType &target,
    TimeFormat time_format,
    const EnumMap *enum_values
) {
    if (target == telem::TIMESTAMP_T) {
        if (value.is_number_unsigned()) {
            const auto u = value.get<uint64_t>();
            if (u > static_cast<uint64_t>(std::numeric_limits<int64_t>::max()))
                throw std::runtime_error("value is out of bounds");
            return integer_to_timestamp(static_cast<int64_t>(u), time_format);
        }
        if (value.is_number_integer())
            return integer_to_timestamp(value.get<int64_t>(), time_format);
        if (value.is_number())
            return number_to_timestamp(value.get<double>(), time_format);
        if (value.is_string()) {
            if (time_format == TimeFormat::ISO8601)
                return parse_rfc3339(value.get<std::string>());
            const auto &str = value.get_ref<const std::string &>();
            if (is_integer_string(str))
                return integer_to_timestamp(parse_string_int64(str), time_format);
            return number_to_timestamp(parse_string_double(str), time_format);
        }
        throw std::runtime_error("");
    }

    if (target == telem::STRING_T) {
        if (value.is_string()) {
            const auto &str = value.get_ref<const std::string &>();
            if (enum_values != nullptr) {
                auto it = enum_values->find(str);
                if (it != enum_values->end()) return enum_to_sample(it->second, target);
            }
            return telem::SampleValue(str);
        }
        return telem::SampleValue(value.dump());
    }

    if (target == telem::JSON_T) return telem::SampleValue(value.dump());

    if (target == telem::UUID_T || target == telem::BYTES_T) {
        if (!value.is_string()) throw std::runtime_error("value is not a string");
        const auto &str = value.get_ref<const std::string &>();
        if (target == telem::BYTES_T) {
            auto [bytes, err] = base64::decode_strict(str);
            if (err) throw std::runtime_error(err.data);
            return telem::SampleValue(std::move(bytes));
        }
        const auto [uuid, err] = uuid::UUID::parse(str);
        if (err) throw std::runtime_error("not a valid UUID");
        return telem::SampleValue(
            std::string(reinterpret_cast<const char *>(uuid.data()), uuid.size())
        );
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
            if (it != enum_values->end()) return enum_to_sample(it->second, target);
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
        if (!details.empty()) msg += ": " + details;
        return {
            telem::SampleValue(),
            errors::Error(CONVERSION_ERROR, msg),
        };
    }
}

std::pair<int64_t, int64_t>
split_unix(const telem::TimeStamp ts, const TimeFormat format) {
    const int64_t unit = nanoseconds_per_unit(format);
    const int64_t ns = ts.nanoseconds();
    int64_t whole = ns / unit;
    const int64_t frac = ns % unit;
    if (format == TimeFormat::UnixSecond || frac == 0) return {whole, frac};
    // Division truncates toward zero, so a negative remainder takes one unit off.
    if (frac < 0) whole--;
    return {whole, 0};
}

}
