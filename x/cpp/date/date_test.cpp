// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <ctime>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/date/date.h"

TEST(CivilFromDays, Epoch) {
    const date::Date dt = date::civil_from_days(0);
    ASSERT_EQ(dt.year, 1970);
    ASSERT_EQ(dt.month, 1);
    ASSERT_EQ(dt.day, 1);
}

TEST(CivilFromDays, LeapYearFeb29) {
    // 2000-02-29 = day 11016
    const date::Date dt = date::civil_from_days(11016);
    ASSERT_EQ(dt.year, 2000);
    ASSERT_EQ(dt.month, 2);
    ASSERT_EQ(dt.day, 29);
}

TEST(CivilFromDays, LeapYearMar1) {
    // 2000-03-01 = day 11017
    const date::Date dt = date::civil_from_days(11017);
    ASSERT_EQ(dt.year, 2000);
    ASSERT_EQ(dt.month, 3);
    ASSERT_EQ(dt.day, 1);
}

TEST(CivilFromDays, NegativeDay) {
    // 1969-12-31 = day -1
    const date::Date dt = date::civil_from_days(-1);
    ASSERT_EQ(dt.year, 1969);
    ASSERT_EQ(dt.month, 12);
    ASSERT_EQ(dt.day, 31);
}

TEST(CivilFromDays, Year2100NotLeap) {
    // 2100-03-01 = day 47541
    const date::Date dt = date::civil_from_days(47541);
    ASSERT_EQ(dt.year, 2100);
    ASSERT_EQ(dt.month, 3);
    ASSERT_EQ(dt.day, 1);
}

TEST(DaysFromCivil, Epoch) {
    ASSERT_EQ(date::days_from_civil({1970, 1, 1}), 0);
}

TEST(DaysFromCivil, LeapYearFeb29) {
    ASSERT_EQ(date::days_from_civil({2000, 2, 29}), 11016);
}

TEST(DaysFromCivil, LeapYearMar1) {
    ASSERT_EQ(date::days_from_civil({2000, 3, 1}), 11017);
}

TEST(DaysFromCivil, PreEpoch) {
    ASSERT_EQ(date::days_from_civil({1969, 12, 31}), -1);
}

TEST(DaysFromCivil, Year2100NotLeap) {
    ASSERT_EQ(date::days_from_civil({2100, 3, 1}), 47541);
}

TEST(DaysFromCivil, RoundTrip) {
    for (int32_t d = -50000; d <= 50000; d += 137) {
        const auto dt = date::civil_from_days(d);
        ASSERT_EQ(date::days_from_civil(dt), d);
    }
}

TEST(IsoWeeksIn, LongAndShortYears) {
    // 2015 starts on a Thursday and 2020 on a Wednesday in a leap year.
    ASSERT_EQ(date::iso_weeks_in(2015), 53);
    ASSERT_EQ(date::iso_weeks_in(2020), 53);
    ASSERT_EQ(date::iso_weeks_in(2021), 52);
    ASSERT_EQ(date::iso_weeks_in(2024), 52);
    ASSERT_EQ(date::iso_weeks_in(2026), 53);
}

TEST(CivilFromNanoseconds, Epoch) {
    const date::CivilTime c = date::civil_from_nanoseconds(0);
    ASSERT_EQ(c.year, 1970);
    ASSERT_EQ(c.month, 1);
    ASSERT_EQ(c.day, 1);
    ASSERT_EQ(c.hour, 0);
    ASSERT_EQ(c.minute, 0);
    ASSERT_EQ(c.second, 0);
    ASSERT_EQ(c.nanosecond, 0);
    ASSERT_EQ(c.year_day, 1);
    ASSERT_EQ(c.weekday, 4);
    ASSERT_EQ(c.iso_year, 1970);
    ASSERT_EQ(c.iso_week, 1);
}

TEST(CivilFromNanoseconds, Earliest) {
    const date::CivilTime c = date::civil_from_nanoseconds(INT64_MIN);
    ASSERT_EQ(c.year, 1677);
    ASSERT_EQ(c.month, 9);
    ASSERT_EQ(c.day, 21);
    ASSERT_EQ(c.hour, 0);
    ASSERT_EQ(c.minute, 12);
    ASSERT_EQ(c.second, 43);
    ASSERT_EQ(c.nanosecond, 145'224'192);
    ASSERT_EQ(c.year_day, 264);
    ASSERT_EQ(c.weekday, 2);
    ASSERT_EQ(c.iso_year, 1677);
    ASSERT_EQ(c.iso_week, 38);
}

TEST(CivilFromNanoseconds, Latest) {
    const date::CivilTime c = date::civil_from_nanoseconds(INT64_MAX);
    ASSERT_EQ(c.year, 2262);
    ASSERT_EQ(c.month, 4);
    ASSERT_EQ(c.day, 11);
    ASSERT_EQ(c.hour, 23);
    ASSERT_EQ(c.minute, 47);
    ASSERT_EQ(c.second, 16);
    ASSERT_EQ(c.nanosecond, 854'775'807);
    ASSERT_EQ(c.year_day, 101);
    ASSERT_EQ(c.weekday, 5);
    ASSERT_EQ(c.iso_year, 2262);
    ASSERT_EQ(c.iso_week, 15);
}

TEST(CivilFromNanoseconds, AroundTheSecond) {
    const int64_t secs[] = {
        INT64_MIN / 1'000'000'000 + 1,
        -1'000'000,
        -1,
        0,
        1,
        1'000'000,
        INT64_MAX / 1'000'000'000 - 1,
    };
    const int64_t nanos[] = {-999'999'999, -1, 0, 1, 999'999'999};
    for (const int64_t sec: secs) {
        for (const int64_t ns: nanos) {
            const int64_t value = sec * 1'000'000'000 + ns;
            const date::CivilTime c = date::civil_from_nanoseconds(value);
            const int64_t expected = ((value % 1'000'000'000) + 1'000'000'000) %
                                     1'000'000'000;
            ASSERT_EQ(c.nanosecond, expected) << value;
        }
    }
}

TEST(CivilFromNanoseconds, LastNanosecondBeforeTheEpoch) {
    const date::CivilTime c = date::civil_from_nanoseconds(-1);
    ASSERT_EQ(c.year, 1969);
    ASSERT_EQ(c.month, 12);
    ASSERT_EQ(c.day, 31);
    ASSERT_EQ(c.hour, 23);
    ASSERT_EQ(c.minute, 59);
    ASSERT_EQ(c.second, 59);
    ASSERT_EQ(c.nanosecond, 999'999'999);
    ASSERT_EQ(c.year_day, 365);
    ASSERT_EQ(c.weekday, 3);
    ASSERT_EQ(c.iso_year, 1970);
    ASSERT_EQ(c.iso_week, 1);
}

TEST(CivilFromNanoseconds, IsoWeekRollover) {
    // January 1, 2021 falls in ISO week 53 of 2020.
    const date::CivilTime a = date::civil_from_nanoseconds(1'609'459'200'000'000'000);
    ASSERT_EQ(a.year, 2021);
    ASSERT_EQ(a.iso_year, 2020);
    ASSERT_EQ(a.iso_week, 53);
    // December 30, 2024 falls in ISO week 1 of 2025.
    const date::CivilTime b = date::civil_from_nanoseconds(1'735'516'800'000'000'000);
    ASSERT_EQ(b.year, 2024);
    ASSERT_EQ(b.iso_year, 2025);
    ASSERT_EQ(b.iso_week, 1);
    // Sunday, January 1, 2012 falls in ISO week 52 of 2011.
    const date::CivilTime c = date::civil_from_nanoseconds(1'325'376'000'000'000'000);
    ASSERT_EQ(c.weekday, 0);
    ASSERT_EQ(c.iso_year, 2011);
    ASSERT_EQ(c.iso_week, 52);
}

TEST(CivilFromNanoseconds, MatchesTheCLibraryOnEveryDay) {
    // 4'133'980'800 is 2101-01-01T00:00:00Z.
    for (int64_t sec = 13 * 3600 + 4 * 60 + 5; sec < 4'133'980'800; sec += 86400) {
        const auto t = static_cast<std::time_t>(sec);
        std::tm tm{};
#ifdef _WIN32
        ASSERT_EQ(gmtime_s(&tm, &t), 0) << sec;
#else
        ASSERT_NE(gmtime_r(&t, &tm), nullptr) << sec;
#endif
        const date::CivilTime c = date::civil_from_nanoseconds(sec * 1'000'000'000);
        ASSERT_EQ(c.year, tm.tm_year + 1900) << sec;
        ASSERT_EQ(c.month, tm.tm_mon + 1) << sec;
        ASSERT_EQ(c.day, tm.tm_mday) << sec;
        ASSERT_EQ(c.hour, tm.tm_hour) << sec;
        ASSERT_EQ(c.minute, tm.tm_min) << sec;
        ASSERT_EQ(c.second, tm.tm_sec) << sec;
        ASSERT_EQ(c.nanosecond, 0) << sec;
        ASSERT_EQ(c.year_day, tm.tm_yday + 1) << sec;
        ASSERT_EQ(c.weekday, tm.tm_wday) << sec;
        char buf[16];
        ASSERT_GT(std::strftime(buf, sizeof(buf), "%G %V", &tm), 0u) << sec;
        ASSERT_EQ(
            std::to_string(c.iso_year) + " " + (c.iso_week < 10 ? "0" : "") +
                std::to_string(c.iso_week),
            std::string(buf)
        ) << sec;
    }
}
