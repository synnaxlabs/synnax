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
    uint16_t year;  ///< calendar year.
    uint8_t month;  ///< month of year [1, 12].
    uint8_t day;    ///< day of month [1, 31].
};

/// @brief converts a day count (days since 1970-01-01) into civil date components using
/// Howard Hinnant's proleptic Gregorian algorithm. Constant time, integer-only
/// arithmetic with no loops.
/// @param day_count days since Unix epoch (negative for dates before 1970).
/// @returns the civil date components.
[[nodiscard]] inline constexpr Date civil_from_days(const int32_t day_count) {
    // Shift to civil-from-days epoch (0000-03-01)
    int32_t z = day_count + 719468;

    const int32_t era = (z >= 0 ? z : z - 146096) / 146097;

    const uint32_t doe = static_cast<uint32_t>(z - era * 146097); // [0, 146096]
    const uint32_t yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) /
                         365; // [0, 399]

    int32_t year = static_cast<int32_t>(yoe) + era * 400;

    const uint32_t doy = doe - (365 * yoe + yoe / 4 - yoe / 100); // [0, 365]
    const uint32_t mp = (5 * doy + 2) / 153; // [0, 11]

    const uint8_t day = static_cast<uint8_t>(doy - (153 * mp + 2) / 5 + 1); // [1, 31]
    const uint8_t month = static_cast<uint8_t>(mp + (mp < 10 ? 3 : -9)); // [1, 12]

    year += (month <= 2);

    return Date{static_cast<uint16_t>(year), month, day};
}

}
