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

#include "driver/can/can.h"
#include "driver/can/vendor.h"

namespace driver::can::vendor {
TEST(Load, RefusesAPlatformWithoutTheLibrary) {
    const auto [lib, err] = load({.unsupported = "the library runs on Windows"});
    EXPECT_EQ(lib, nullptr);
    ASSERT_MATCHES(err, UNSUPPORTED_ERROR);
    EXPECT_EQ(err.data, "the library runs on Windows");
}

TEST(Load, PointsToTheDownloadPageWhenNoNameLoads) {
    const auto [lib, err] = load({
        .names = {"synnax_missing_one", "synnax_missing_two"},
        .info = {"Example", "https://example.com"},
    });
    EXPECT_EQ(lib, nullptr);
    ASSERT_MATCHES(err, x::lib::LOAD_ERROR);
    EXPECT_EQ(
        err.data,
        "Example library is not installed. Download here: https://example.com. "
        "Restart Driver after installation."
    );
}
}
