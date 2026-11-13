// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/filter.h"

namespace driver::ethercat::soem {
#ifdef _WIN32
/// @brief it should refuse an interface that is not a Windows network adapter.
TEST(DisableDewesoftFilterTest, RefusesAnInterfaceThatIsNotAnAdapter) {
    const auto err = disable_dewesoft_filter({"eth0"});
    ASSERT_OCCURRED_AS(err, errors::FILTER_ERROR);
    EXPECT_EQ(err.data, "eth0 is not a Windows network adapter.");
}
#else
/// @brief it should refuse to turn off the filter outside Windows.
TEST(DisableDewesoftFilterTest, RefusesOutsideWindows) {
    const auto err = disable_dewesoft_filter({"eth0"});
    ASSERT_OCCURRED_AS(err, errors::FILTER_ERROR);
    EXPECT_EQ(err.data, "Only a Driver on Windows can turn off the DewesoftX filter.");
}
#endif
}
