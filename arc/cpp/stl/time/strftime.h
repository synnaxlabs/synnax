// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>
#include <string>
#include <string_view>

#include "x/cpp/date/date.h"
#include "x/cpp/telem/telem.h"

namespace arc::stl::time {

/// @brief holds the UTC calendar and clock fields of a timestamp.
struct CivilTime {
    int year;
    int month;
    int day;
    int hour;
    int minute;
    int second;
    int nanosecond;
    /// @brief counts from 1 on January 1.
    int year_day;
    /// @brief counts from 0 on Sunday.
    int weekday;
    int iso_year;
    int iso_week;
};

/// @brief returns the number of ISO weeks in year, 52 or 53.
inline int iso_weeks_in(const int year) {
    const auto p = [](const int y) { return (y + y / 4 - y / 100 + y / 400) % 7; };
    return p(year) == 4 || p(year - 1) == 3 ? 53 : 52;
}

inline CivilTime break_down(const x::telem::TimeStamp ts) {
    const int64_t value = ts.nanoseconds();
    const int64_t sec = value / 1'000'000'000 -
                        (value % 1'000'000'000 != 0 && value < 0);
    const int64_t day = sec / 86400 - (sec % 86400 != 0 && sec < 0);
    const int64_t sod = sec - day * 86400;
    const date::Date d = date::civil_from_days(static_cast<int32_t>(day));
    CivilTime c{};
    c.year = d.year;
    c.month = d.month;
    c.day = d.day;
    c.hour = static_cast<int>(sod / 3600);
    c.minute = static_cast<int>(sod / 60 % 60);
    c.second = static_cast<int>(sod % 60);
    c.nanosecond = static_cast<int>(value - sec * 1'000'000'000);
    c.year_day = static_cast<int>(day - date::days_from_civil({d.year, 1, 1})) + 1;
    // 1970-01-01 was a Thursday.
    c.weekday = static_cast<int>((day % 7 + 11) % 7);
    const int iso_weekday = (c.weekday + 6) % 7 + 1;
    c.iso_year = c.year;
    c.iso_week = (c.year_day - iso_weekday + 10) / 7;
    if (c.iso_week < 1) {
        c.iso_year--;
        c.iso_week = iso_weeks_in(c.iso_year);
    } else if (c.iso_week > iso_weeks_in(c.year)) {
        c.iso_year++;
        c.iso_week = 1;
    }
    return c;
}

/// @brief appends v to b, zero padded to width. A sign counts toward width.
inline void pad(std::string &b, int v, int width) {
    if (v < 0) {
        b += '-';
        v = -v;
        width--;
    }
    const auto digits = std::to_string(v);
    if (digits.size() < static_cast<size_t>(width))
        b.append(static_cast<size_t>(width) - digits.size(), '0');
    b += digits;
}

/// @brief appends the expansion of one format code for c to b. Every strftime code
/// that does not depend on locale is here. The locale codes %a %A %b %B %p %c %x %X
/// print weekday and month names, AM and PM, and the local date style. Those words
/// change with the user's language, and both runtimes must print the same bytes, so
/// the locale codes are left out.
/// @returns false for an unsupported code, with b unchanged.
inline bool expand(std::string &b, const CivilTime &c, const char code) {
    switch (code) {
        case 'Y':
            pad(b, c.year, 4);
            return true;
        case 'y':
            pad(b, c.year % 100, 2);
            return true;
        case 'm':
            pad(b, c.month, 2);
            return true;
        case 'd':
            pad(b, c.day, 2);
            return true;
        case 'e':
            if (c.day < 10) b += ' ';
            b += std::to_string(c.day);
            return true;
        case 'H':
            pad(b, c.hour, 2);
            return true;
        case 'I':
            pad(b, (c.hour + 11) % 12 + 1, 2);
            return true;
        case 'M':
            pad(b, c.minute, 2);
            return true;
        case 'S':
            pad(b, c.second, 2);
            return true;
        case 'j':
            pad(b, c.year_day, 3);
            return true;
        case 'f':
            pad(b, c.nanosecond / 1000, 6);
            return true;
        case 'N':
            pad(b, c.nanosecond, 9);
            return true;
        case 'u':
            pad(b, (c.weekday + 6) % 7 + 1, 1);
            return true;
        case 'w':
            pad(b, c.weekday, 1);
            return true;
        // Days before the first Sunday of the year fall in week 0.
        case 'U':
            pad(b, (c.year_day - 1 - c.weekday + 7) / 7, 2);
            return true;
        // Days before the first Monday of the year fall in week 0.
        case 'W':
            pad(b, (c.year_day - 1 - (c.weekday + 6) % 7 + 7) / 7, 2);
            return true;
        case 'G':
            pad(b, c.iso_year, 4);
            return true;
        case 'V':
            pad(b, c.iso_week, 2);
            return true;
        case 'z':
            b += "+0000";
            return true;
        case 'Z':
            b += "UTC";
            return true;
        case 'F':
            pad(b, c.year, 4);
            b += '-';
            pad(b, c.month, 2);
            b += '-';
            pad(b, c.day, 2);
            return true;
        case 'T':
            pad(b, c.hour, 2);
            b += ':';
            pad(b, c.minute, 2);
            b += ':';
            pad(b, c.second, 2);
            return true;
        case '%':
            b += '%';
            return true;
        default:
            return false;
    }
}

/// @brief renders ts in UTC with format. An unsupported code, and a '%' at the end
/// of format, are copied through unchanged.
inline std::string
strftime(const x::telem::TimeStamp ts, const std::string_view format) {
    const auto c = break_down(ts);
    std::string b;
    b.reserve(format.size() * 2);
    for (size_t i = 0; i < format.size(); i++) {
        if (format[i] != '%' || i + 1 == format.size()) {
            b += format[i];
            continue;
        }
        i++;
        if (!expand(b, c, format[i])) {
            b += '%';
            b += format[i];
        }
    }
    return b;
}

}
