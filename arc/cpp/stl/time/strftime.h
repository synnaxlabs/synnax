// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstddef>
#include <cstdint>
#include <string>
#include <string_view>

#include "x/cpp/date/date.h"
#include "x/cpp/telem/telem.h"

namespace arc::stl::time {

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

const std::array<std::string_view, 7> WEEKDAYS = {
    "Sunday",
    "Monday",
    "Tuesday",
    "Wednesday",
    "Thursday",
    "Friday",
    "Saturday",
};

const std::array<std::string_view, 12> MONTHS = {
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December",
};

/// @brief appends the expansion of one format code for c to b.
/// @returns false for an unsupported code, with b unchanged.
inline bool expand(std::string &b, const date::CivilTime &c, const char code) {
    switch (code) {
        case 'Y':
            pad(b, c.year, 4);
            return true;
        case 'y':
            pad(b, c.year % 100, 2);
            return true;
        case 'C':
            pad(b, c.year / 100, 2);
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
        case 'g':
            pad(b, c.iso_year % 100, 2);
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
        case 'D':
            pad(b, c.month, 2);
            b += '/';
            pad(b, c.day, 2);
            b += '/';
            pad(b, c.year % 100, 2);
            return true;
        case 'R':
            pad(b, c.hour, 2);
            b += ':';
            pad(b, c.minute, 2);
            return true;
        case 'n':
            b += '\n';
            return true;
        case 't':
            b += '\t';
            return true;
        case 'a':
            b += WEEKDAYS[c.weekday].substr(0, 3);
            return true;
        case 'A':
            b += WEEKDAYS[c.weekday];
            return true;
        case 'b':
        case 'h':
            b += MONTHS[c.month - 1].substr(0, 3);
            return true;
        case 'B':
            b += MONTHS[c.month - 1];
            return true;
        case 'p':
            b += c.hour < 12 ? "AM" : "PM";
            return true;
        case 'c':
            expand(b, c, 'a');
            b += ' ';
            expand(b, c, 'b');
            b += ' ';
            expand(b, c, 'e');
            b += ' ';
            expand(b, c, 'T');
            b += ' ';
            return expand(b, c, 'Y');
        case 'x':
            return expand(b, c, 'D');
        case 'X':
            return expand(b, c, 'T');
        case 'r':
            expand(b, c, 'I');
            b += ':';
            expand(b, c, 'M');
            b += ':';
            expand(b, c, 'S');
            b += ' ';
            return expand(b, c, 'p');
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
    const date::CivilTime c = date::civil_from_nanoseconds(ts.nanoseconds());
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
