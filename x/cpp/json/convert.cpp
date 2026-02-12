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
#include <string>

#include "x/cpp/errors/errors.h"
#include "x/cpp/date/date.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/telem/telem.h"

namespace x::json {

namespace {

std::pair<telem::SampleValue, errors::Error>
parse_rfc3339(const std::string &input) {
    if (input.size() < 20) return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};

    if (input[4] != '-' || input[7] != '-') return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    if (input[10] != 'T' && input[10] != 't' && input[10] != ' ')
        return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    if (input[13] != ':' || input[16] != ':') return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};

    auto parse_uint = [&](const size_t start, const size_t len)
                          -> std::pair<int32_t, bool> {
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
        return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    if (month < 1 || month > 12) return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    if (day < 1 || day > 31) return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    if (hour > 23) return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    if (minute > 59) return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    if (second > 60) return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};

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
    if (pos >= input.size()) return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    if (input[pos] == 'Z' || input[pos] == 'z') {
        // UTC
    } else if (input[pos] == '+' || input[pos] == '-') {
        const bool negative = (input[pos] == '-');
        ++pos;
        if (pos + 5 > input.size() || input[pos + 2] != ':')
            return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
        const auto [tz_hour, tzh_ok] = parse_uint(pos, 2);
        const auto [tz_min, tzm_ok] = parse_uint(pos + 3, 2);
        if (!tzh_ok || !tzm_ok) return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
        tz_offset_seconds = (tz_hour * 3600 + tz_min * 60) * (negative ? -1 : 1);
    } else {
        return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    }

    const int32_t days = date::days_from_civil(
        {static_cast<uint16_t>(year),
         static_cast<uint8_t>(month),
         static_cast<uint8_t>(day)}
    );
    const int64_t total_seconds =
        static_cast<int64_t>(days) * 86400 + hour * 3600 + minute * 60 + second
        - tz_offset_seconds;
    return {telem::SampleValue(telem::TimeStamp(total_seconds * 1000000000 + frac_ns)), errors::NIL};
}

template<typename T>
std::pair<telem::SampleValue, errors::Error>
convert_number(const double v, const bool strict) {
    if constexpr (std::is_integral_v<T>) {
        if (strict) {
            if (v != std::trunc(v))
                return {telem::SampleValue(static_cast<T>(0)), TRUNCATION_ERROR};
            if (v < static_cast<double>(std::numeric_limits<T>::min()) ||
                v > static_cast<double>(std::numeric_limits<T>::max()))
                return {telem::SampleValue(static_cast<T>(0)), OVERFLOW_ERROR};
        }
    }
    return {telem::SampleValue(static_cast<T>(v)), errors::NIL};
}

std::pair<telem::SampleValue, errors::Error>
number_to_numeric(
    const double v,
    const telem::DataType &target,
    const bool strict
) {
    if (target == telem::FLOAT64_T) return convert_number<double>(v, strict);
    if (target == telem::FLOAT32_T) return convert_number<float>(v, strict);
    if (target == telem::INT64_T) return convert_number<int64_t>(v, strict);
    if (target == telem::INT32_T) return convert_number<int32_t>(v, strict);
    if (target == telem::INT16_T) return convert_number<int16_t>(v, strict);
    if (target == telem::INT8_T) return convert_number<int8_t>(v, strict);
    if (target == telem::UINT64_T) return convert_number<uint64_t>(v, strict);
    if (target == telem::UINT32_T) return convert_number<uint32_t>(v, strict);
    if (target == telem::UINT16_T) return convert_number<uint16_t>(v, strict);
    if (target == telem::UINT8_T) return convert_number<uint8_t>(v, strict);
    return {telem::SampleValue(int64_t(0)), UNSUPPORTED_ERROR};
}

} // namespace

std::pair<telem::SampleValue, errors::Error> to_sample_value(
    const nlohmann::json &value,
    const telem::DataType &target,
    const ReadOptions &opts
) {
    if (target == telem::TIMESTAMP_T) {
        if (value.is_number()) {
            switch (opts.time_format) {
                case TimeFormat::UnixNanosecond:
                    return {
                        telem::SampleValue(telem::TimeStamp(value.get<int64_t>())),
                        errors::NIL
                    };
                case TimeFormat::UnixMicrosecond:
                    return {
                        telem::SampleValue(telem::TimeStamp(
                            static_cast<int64_t>(value.get<double>() * 1e3)
                        )),
                        errors::NIL
                    };
                case TimeFormat::UnixMillisecond:
                    return {
                        telem::SampleValue(telem::TimeStamp(
                            static_cast<int64_t>(value.get<double>() * 1e6)
                        )),
                        errors::NIL
                    };
                case TimeFormat::UnixSecond:
                    return {
                        telem::SampleValue(telem::TimeStamp(
                            static_cast<int64_t>(value.get<double>() * 1e9)
                        )),
                        errors::NIL
                    };
                case TimeFormat::ISO8601:
                    return {
                        telem::SampleValue(telem::TimeStamp(0)),
                        UNSUPPORTED_ERROR
                    };
            }
        }
        if (value.is_string() && opts.time_format == TimeFormat::ISO8601)
            return parse_rfc3339(value.get<std::string>());
        return {telem::SampleValue(telem::TimeStamp(0)), UNSUPPORTED_ERROR};
    }

    if (target == telem::STRING_T) {
        if (value.is_number())
            return {telem::SampleValue(value.dump()), errors::NIL};
        if (value.is_string())
            return {telem::SampleValue(value.get<std::string>()), errors::NIL};
        if (value.is_boolean())
            return {
                telem::SampleValue(
                    std::string(value.get<bool>() ? "true" : "false")
                ),
                errors::NIL
            };
        return {telem::SampleValue(std::string()), UNSUPPORTED_ERROR};
    }

    if (value.is_boolean())
        return number_to_numeric(value.get<bool>() ? 1 : 0, target, false);

    if (value.is_number())
        return number_to_numeric(value.get<double>(), target, opts.strict);

    return {telem::SampleValue(int64_t(0)), UNSUPPORTED_ERROR};
}

std::pair<nlohmann::json, errors::Error>
from_sample_value(const telem::SampleValue &value, Type target) {
    return std::visit(
        [target](const auto &v) -> std::pair<nlohmann::json, errors::Error> {
            using T = std::decay_t<decltype(v)>;
            if constexpr (std::is_same_v<T, std::string>) {
                if (target == Type::String)
                    return {nlohmann::json(v), errors::NIL};
                return {nlohmann::json(), UNSUPPORTED_ERROR};
            } else if constexpr (std::is_same_v<T, telem::TimeStamp>) {
                return {nlohmann::json(), UNSUPPORTED_ERROR};
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
                return {nlohmann::json(), UNSUPPORTED_ERROR};
            }
        },
        value
    );
}

errors::Error
check_from_sample_value(const telem::DataType &type, Type target) {
    if (type == telem::STRING_T)
        return target == Type::String ? errors::NIL : UNSUPPORTED_ERROR;
    if (type == telem::FLOAT64_T || type == telem::FLOAT32_T ||
        type == telem::INT64_T || type == telem::INT32_T || type == telem::INT16_T ||
        type == telem::INT8_T || type == telem::UINT64_T || type == telem::UINT32_T ||
        type == telem::UINT16_T || type == telem::UINT8_T)
        return errors::NIL;
    return UNSUPPORTED_ERROR;
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
    }
}

}
