// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>

namespace date {

/// @brief civil date components.
struct Date {
    /// @brief calendar year.
    uint16_t year;
    /// @brief month of year [1, 12].
    uint8_t month;
    /// @brief day of month [1, 31].
    uint8_t day;
};

/// @brief converts civil date components to a day count (days since 1970-01-01) using
/// Howard Hinnant's proleptic Gregorian algorithm. Inverse of civil_from_days.
/// @param d the civil date components.
/// @returns days since Unix epoch (negative for dates before 1970).
[[nodiscard]] constexpr int32_t days_from_civil(const Date &d) {
    int32_t y = d.year;
    const uint32_t m = d.month;
    const uint32_t day = d.day;
    y -= (m <= 2);
    const int32_t era = (y >= 0 ? y : y - 399) / 400;
    const auto yoe = static_cast<uint32_t>(y - era * 400);
    const uint32_t doy = (153 * (m > 2 ? m - 3 : m + 9) + 2) / 5 + day - 1;
    const uint32_t doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    return era * 146097 + static_cast<int32_t>(doe) - 719468;
}

/// @brief converts a day count (days since 1970-01-01) into civil date components using
/// Howard Hinnant's proleptic Gregorian algorithm. Constant time, integer-only
/// arithmetic with no loops.
/// @param day_count days since Unix epoch (negative for dates before 1970).
/// @returns the civil date components.
[[nodiscard]] constexpr Date civil_from_days(const int32_t day_count) {
    // Shift to civil-from-days epoch (0000-03-01)
    int32_t z = day_count + 719468;

    const int32_t era = (z >= 0 ? z : z - 146096) / 146097;

    const auto doe = static_cast<uint32_t>(z - era * 146097); // [0, 146096]
    const uint32_t yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) /
                         365; // [0, 399]

    int32_t year = static_cast<int32_t>(yoe) + era * 400;

    const uint32_t doy = doe - (365 * yoe + yoe / 4 - yoe / 100); // [0, 365]
    const uint32_t mp = (5 * doy + 2) / 153; // [0, 11]

    const auto day = static_cast<uint8_t>(doy - (153 * mp + 2) / 5 + 1); // [1, 31]
    const auto month = static_cast<uint8_t>(mp + (mp < 10 ? 3 : -9)); // [1, 12]

    year += (month <= 2);

    return Date{static_cast<uint16_t>(year), month, day};
}

/// @brief the calendar and clock fields of an instant, in UTC.
struct CivilTime {
    /// @brief calendar year.
    int32_t year;
    /// @brief month of year [1, 12].
    uint8_t month;
    /// @brief day of month [1, 31].
    uint8_t day;
    /// @brief hour of day [0, 23].
    uint8_t hour;
    /// @brief minute of hour [0, 59].
    uint8_t minute;
    /// @brief second of minute [0, 59].
    uint8_t second;
    /// @brief nanoseconds within the second [0, 999999999].
    int32_t nanosecond;
    /// @brief day of year, counting from 1 on January 1 [1, 366].
    uint16_t year_day;
    /// @brief day of week, counting from 0 on Sunday [0, 6].
    uint8_t weekday;
    /// @brief ISO 8601 week-based year, which differs from year at the ends of the
    /// year.
    int32_t iso_year;
    /// @brief ISO 8601 week of iso_year [1, 53].
    uint8_t iso_week;
};

/// @brief returns the number of ISO 8601 weeks in a year, 52 or 53.
/// @param year the calendar year.
/// @returns 53 when the year starts on a Thursday, or on a Wednesday in a leap year.
[[nodiscard]] constexpr int iso_weeks_in(const int32_t year) {
    const auto p = [](const int32_t y) { return (y + y / 4 - y / 100 + y / 400) % 7; };
    return p(year) == 4 || p(year - 1) == 3 ? 53 : 52;
}

/// @brief splits nanoseconds since 1970-01-01T00:00:00Z into civil fields in UTC.
/// Every int64_t value splits without overflow, so the earliest and latest
/// timestamps are safe.
/// @param ns nanoseconds since the Unix epoch (negative before 1970).
/// @returns the civil time fields.
[[nodiscard]] constexpr CivilTime civil_from_nanoseconds(const int64_t ns) {
    // The remainder comes first so that no product can leave the int64_t range.
    int64_t sec = ns / 1'000'000'000;
    auto nanosecond = static_cast<int32_t>(ns % 1'000'000'000);
    if (nanosecond < 0) {
        nanosecond += 1'000'000'000;
        sec--;
    }
    const int64_t day = sec / 86400 - (sec % 86400 != 0 && sec < 0);
    const int64_t sod = sec - day * 86400;
    const Date d = civil_from_days(static_cast<int32_t>(day));
    CivilTime c{};
    c.year = d.year;
    c.month = d.month;
    c.day = d.day;
    c.hour = static_cast<uint8_t>(sod / 3600);
    c.minute = static_cast<uint8_t>(sod / 60 % 60);
    c.second = static_cast<uint8_t>(sod % 60);
    c.nanosecond = nanosecond;
    c.year_day = static_cast<uint16_t>(day - days_from_civil({d.year, 1, 1}) + 1);
    // 1970-01-01 was a Thursday.
    c.weekday = static_cast<uint8_t>((day % 7 + 11) % 7);
    const int iso_weekday = (c.weekday + 6) % 7 + 1;
    c.iso_year = c.year;
    int week = (c.year_day - iso_weekday + 10) / 7;
    if (week < 1) {
        c.iso_year--;
        week = iso_weeks_in(c.iso_year);
    } else if (week > iso_weeks_in(c.year)) {
        c.iso_year++;
        week = 1;
    }
    c.iso_week = static_cast<uint8_t>(week);
    return c;
}
}
