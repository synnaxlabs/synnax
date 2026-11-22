// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <limits>
#include <optional>
#include <string>
#include <string_view>
#include <utility>

#include "x/cpp/date/date.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/json/decimal.h"
#include "x/cpp/json/text.h"
#include "x/cpp/json/timestamp.h"
#include "x/cpp/telem/telem.h"

namespace x::json {
namespace {
constexpr std::int64_t NANOS_PER_SECOND = 1000000000;
constexpr std::int64_t MAX_NANOS = std::numeric_limits<std::int64_t>::max();
constexpr std::int64_t MIN_NANOS = std::numeric_limits<std::int64_t>::min();

/// @brief returns the value of the digits of text when it lies in [lo, hi].
std::optional<std::int64_t>
parse_field(const std::string_view text, const std::int64_t lo, const std::int64_t hi) {
    std::int64_t v = 0;
    for (const char c: text) {
        if (c < '0' || c > '9') return std::nullopt;
        v = v * 10 + (c - '0');
    }
    if (v < lo || v > hi) return std::nullopt;
    return v;
}

std::int64_t days_in(const std::int64_t month, const std::int64_t year) {
    if (month == 2) {
        const bool leap = (year % 4 == 0 && year % 100 != 0) || year % 400 == 0;
        return leap ? 29 : 28;
    }
    return month == 4 || month == 6 || month == 9 || month == 11 ? 30 : 31;
}

/// @brief returns the seconds and nanoseconds since the Unix epoch of an RFC 3339
/// timestamp, or nothing when text is not one.
std::optional<std::pair<std::int64_t, std::int64_t>>
parse_rfc3339(std::string_view text) {
    if (text.size() < 19 || text[4] != '-' || text[7] != '-' || text[10] != 'T' ||
        text[13] != ':' || text[16] != ':')
        return std::nullopt;
    const auto year = parse_field(text.substr(0, 4), 0, 9999);
    const auto month = parse_field(text.substr(5, 2), 1, 12);
    if (!year.has_value() || !month.has_value()) return std::nullopt;
    const auto day = parse_field(text.substr(8, 2), 1, days_in(*month, *year));
    const auto hour = parse_field(text.substr(11, 2), 0, 23);
    const auto minute = parse_field(text.substr(14, 2), 0, 59);
    const auto second = parse_field(text.substr(17, 2), 0, 59);
    if (!day.has_value() || !hour.has_value() || !minute.has_value() ||
        !second.has_value())
        return std::nullopt;
    auto rest = text.substr(19);
    std::int64_t nanos = 0;
    if (rest.size() >= 2 && rest[0] == '.' && rest[1] >= '0' && rest[1] <= '9') {
        std::size_t n = 1;
        std::int64_t unit = NANOS_PER_SECOND;
        for (; n < rest.size() && rest[n] >= '0' && rest[n] <= '9'; n++) {
            if (unit == 1) continue;
            unit /= 10;
            nanos += (rest[n] - '0') * unit;
        }
        rest.remove_prefix(n);
    }
    std::int64_t offset = 0;
    if (rest != "Z") {
        if (rest.size() != 6 || (rest[0] != '+' && rest[0] != '-') || rest[3] != ':')
            return std::nullopt;
        const auto offset_hour = parse_field(rest.substr(1, 2), 0, 23);
        const auto offset_minute = parse_field(rest.substr(4, 2), 0, 59);
        if (!offset_hour.has_value() || !offset_minute.has_value()) return std::nullopt;
        offset = (*offset_hour * 60 + *offset_minute) * 60;
        if (rest[0] == '-') offset = -offset;
    }
    const std::int64_t days = date::days_from_civil({
        .year = static_cast<std::uint16_t>(*year),
        .month = static_cast<std::uint8_t>(*month),
        .day = static_cast<std::uint8_t>(*day),
    });
    const std::int64_t seconds = days * 86400 + *hour * 3600 + *minute * 60 + *second -
                                 offset;
    return {{seconds, nanos}};
}

/// @brief returns the nanoseconds since the Unix epoch, or nothing when they do not fit
/// an int64.
std::optional<std::int64_t>
to_nanos(const std::int64_t seconds, const std::int64_t nanos) {
    constexpr std::int64_t max_seconds = MAX_NANOS / NANOS_PER_SECOND;
    constexpr std::int64_t min_seconds = MIN_NANOS / NANOS_PER_SECOND - 1;
    if (seconds > max_seconds || seconds < min_seconds) return std::nullopt;
    if (seconds == max_seconds && nanos > MAX_NANOS % NANOS_PER_SECOND)
        return std::nullopt;
    if (seconds == min_seconds &&
        nanos < NANOS_PER_SECOND + MIN_NANOS % NANOS_PER_SECOND)
        return std::nullopt;
    if (seconds >= 0) return seconds * NANOS_PER_SECOND + nanos;
    return (seconds + 1) * NANOS_PER_SECOND + (nanos - NANOS_PER_SECOND);
}

int nanos_exponent(const TimeFormat format) {
    switch (format) {
        case TimeFormat::UnixSecond:
            return 9;
        case TimeFormat::UnixMillisecond:
            return 6;
        case TimeFormat::UnixMicrosecond:
            return 3;
        case TimeFormat::UnixNanosecond:
        case TimeFormat::ISO8601:
            return 0;
    }
    return 0;
}

std::string format_name(const TimeFormat format) {
    switch (format) {
        case TimeFormat::UnixSecond:
            return "unix_sec";
        case TimeFormat::UnixMillisecond:
            return "unix_ms";
        case TimeFormat::UnixMicrosecond:
            return "unix_us";
        case TimeFormat::UnixNanosecond:
            return "unix_ns";
        case TimeFormat::ISO8601:
            return "iso8601";
    }
    return "";
}
}

std::pair<telem::TimeStamp, errors::Error> parse_iso8601(std::string &text) {
    if (text.size() > 10 && (text[10] == 't' || text[10] == ' ')) text[10] = 'T';
    if (!text.empty() && text.back() == 'z') text.back() = 'Z';
    const auto parsed = parse_rfc3339(text);
    if (!parsed.has_value())
        return {
            telem::TimeStamp(0),
            errors::Error(
                errors::VALIDATION,
                "invalid ISO 8601 timestamp " + quote(text)
            ),
        };
    const auto nanos = to_nanos(parsed->first, parsed->second);
    if (!nanos.has_value())
        return {
            telem::TimeStamp(0),
            errors::Error(
                errors::VALIDATION,
                quote(text) + " is out of range for a timestamp"
            ),
        };
    return {telem::TimeStamp(*nanos), errors::NIL};
}

std::pair<telem::TimeStamp, errors::Error>
parse_unix(const std::string_view text, const TimeFormat format) {
    const auto d = parse_decimal(text, nanos_exponent(format));
    if (!d.has_value()) return {telem::TimeStamp(0), invalid_number(text)};
    std::uint64_t limit = MAX_NANOS;
    if (d->negative) limit++;
    if (d->overflow || d->magnitude > limit)
        return {
            telem::TimeStamp(0),
            errors::Error(
                errors::VALIDATION,
                std::string(text) + " is out of range for a " + format_name(format) +
                    " timestamp"
            ),
        };
    if (d->negative)
        return {
            telem::TimeStamp(static_cast<std::int64_t>(0 - d->magnitude)),
            errors::NIL
        };
    return {telem::TimeStamp(static_cast<std::int64_t>(d->magnitude)), errors::NIL};
}
}
