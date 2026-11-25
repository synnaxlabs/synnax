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
#include <string>
#include <tuple>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/json/convert.h"
#include "x/cpp/json/timestamp.h"
#include "x/cpp/telem/telem.h"

namespace x::json {
/// @brief asserts that appending each timestamp of cases in its format after what out
/// holds gives its text.
void assert_appends(
    const std::vector<std::tuple<std::int64_t, TimeFormat, std::string>> &cases
) {
    for (const auto &[ns, format, expected]: cases) {
        std::string out = "x";
        append_timestamp(telem::TimeStamp(ns), format, out);
        EXPECT_EQ(out, "x" + expected) << ns;
    }
}

/// @brief it should append every digit of a timestamp in ISO 8601, unix_sec, and
/// unix_ns after what out holds.
TEST(AppendTimestamp, AppendsEveryDigit) {
    const std::vector<std::tuple<std::int64_t, TimeFormat, std::string>> cases{
        {1500000006, TimeFormat::ISO8601, R"("1970-01-01T00:00:01.500000006Z")"},
        {1767323045000000000, TimeFormat::ISO8601, R"("2026-01-02T03:04:05Z")"},
        {1767323045120000000, TimeFormat::ISO8601, R"("2026-01-02T03:04:05.12Z")"},
        {-1, TimeFormat::ISO8601, R"("1969-12-31T23:59:59.999999999Z")"},
        {2000000000, TimeFormat::UnixSecond, "2"},
        {1767323045123456789, TimeFormat::UnixSecond, "1767323045.123456789"},
        {1000000005, TimeFormat::UnixSecond, "1.000000005"},
        {-1500000000, TimeFormat::UnixSecond, "-1.5"},
        {-500000000, TimeFormat::UnixSecond, "-0.5"},
        {
            std::numeric_limits<std::int64_t>::min(),
            TimeFormat::UnixNanosecond,
            "-9223372036854775808",
        },
        {
            std::numeric_limits<std::int64_t>::min(),
            TimeFormat::UnixSecond,
            "-9223372036.854775808",
        },
    };
    assert_appends(cases);
}

/// @brief it should append a whole number of milliseconds, rounded down.
TEST(AppendTimestamp, RoundsMillisecondsDown) {
    assert_appends({
        {1500000000, TimeFormat::UnixMillisecond, "1500"},
        {1999999, TimeFormat::UnixMillisecond, "1"},
        {1767323045123456789, TimeFormat::UnixMillisecond, "1767323045123"},
    });
}

/// @brief it should append a whole number of microseconds, rounded down.
TEST(AppendTimestamp, RoundsMicrosecondsDown) {
    assert_appends({
        {1500, TimeFormat::UnixMicrosecond, "1"},
        {1767323045123456789, TimeFormat::UnixMicrosecond, "1767323045123456"},
    });
}

/// @brief it should keep every decimal of a time in seconds.
TEST(AppendTimestamp, KeepsDecimalsOfSeconds) {
    assert_appends({
        {1999999, TimeFormat::UnixSecond, "0.001999999"},
        {1500, TimeFormat::UnixSecond, "0.0000015"},
    });
}

/// @brief it should round a time before 1970 toward negative infinity in milliseconds
/// and microseconds, and keep its decimals in seconds.
TEST(AppendTimestamp, RoundsTimesBefore1970Down) {
    assert_appends({
        {-1, TimeFormat::UnixMillisecond, "-1"},
        {-1000000, TimeFormat::UnixMillisecond, "-1"},
        {-1500, TimeFormat::UnixMicrosecond, "-2"},
        {
            std::numeric_limits<std::int64_t>::min(),
            TimeFormat::UnixMillisecond,
            "-9223372036855",
        },
        {-1500, TimeFormat::UnixSecond, "-0.0000015"},
    });
}
}
