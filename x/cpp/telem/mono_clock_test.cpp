// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/mono_clock.h"

namespace x::telem {

/// @brief it should return strictly increasing timestamps from a strictly
/// increasing source.
TEST(MonoClockTests, testStrictlyIncreasingSource) {
    std::int64_t counter = 100;
    MonoClock clock([&counter] { return TimeStamp(counter += 10); });
    const auto t1 = clock.now();
    const auto t2 = clock.now();
    const auto t3 = clock.now();
    ASSERT_LT(t1, t2);
    ASSERT_LT(t2, t3);
    ASSERT_EQ(t1.nanoseconds(), 110);
    ASSERT_EQ(t2.nanoseconds(), 120);
    ASSERT_EQ(t3.nanoseconds(), 130);
}

/// @brief it should bump by one nanosecond when the source returns a duplicate
/// timestamp.
TEST(MonoClockTests, testBumpsOnDuplicate) {
    const auto fixed = TimeStamp(500);
    MonoClock clock([fixed] { return fixed; });
    const auto t1 = clock.now();
    const auto t2 = clock.now();
    const auto t3 = clock.now();
    ASSERT_EQ(t1.nanoseconds(), 500);
    ASSERT_EQ(t2.nanoseconds(), 501);
    ASSERT_EQ(t3.nanoseconds(), 502);
}

/// @brief it should bump when the source goes backwards.
TEST(MonoClockTests, testBumpsOnRegression) {
    std::vector<TimeStamp> stamps{TimeStamp(1000), TimeStamp(500), TimeStamp(750)};
    std::size_t i = 0;
    MonoClock clock([&] { return stamps[i++]; });
    const auto t1 = clock.now();
    const auto t2 = clock.now();
    const auto t3 = clock.now();
    ASSERT_EQ(t1.nanoseconds(), 1000);
    ASSERT_EQ(t2.nanoseconds(), 1001);
    ASSERT_EQ(t3.nanoseconds(), 1002);
}

/// @brief it should default to TimeStamp::now when no source is provided.
TEST(MonoClockTests, testDefaultSourceMonotonic) {
    MonoClock clock;
    const auto t1 = clock.now();
    const auto t2 = clock.now();
    const auto t3 = clock.now();
    ASSERT_LT(t1, t2);
    ASSERT_LT(t2, t3);
}

}
