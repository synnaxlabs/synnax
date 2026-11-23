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
/// @brief it should append every digit of a timestamp in each format after what out
/// holds.
TEST(AppendTimestamp, AppendsEveryDigit) {
    const std::vector<std::tuple<std::int64_t, TimeFormat, std::string>> cases{
        {1500000006, TimeFormat::ISO8601, R"("1970-01-01T00:00:01.500000006Z")"},
        {1767323045000000000, TimeFormat::ISO8601, R"("2026-01-02T03:04:05Z")"},
        {1767323045120000000, TimeFormat::ISO8601, R"("2026-01-02T03:04:05.12Z")"},
        {-1, TimeFormat::ISO8601, R"("1969-12-31T23:59:59.999999999Z")"},
        {2000000000, TimeFormat::UnixSecond, "2"},
        {1767323045123456789, TimeFormat::UnixSecond, "1767323045.123456789"},
        {1000000005, TimeFormat::UnixSecond, "1.000000005"},
        {1500000000, TimeFormat::UnixMillisecond, "1500"},
        {1500, TimeFormat::UnixMicrosecond, "1.5"},
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
    for (const auto &[ns, format, expected]: cases) {
        std::string out = "x";
        append_timestamp(telem::TimeStamp(ns), format, out);
        EXPECT_EQ(out, "x" + expected);
    }
}
}
