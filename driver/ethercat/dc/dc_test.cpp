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

#include "driver/ethercat/dc/dc.h"

namespace driver::ethercat::dc {

/// @brief it should give back the same mode after a JSON round trip, negative factors
/// and shifts included.
TEST(OpMode, RoundTripsThroughJSON) {
    const OpMode mode{
        .name = "DC-Synchron",
        .assign_activate = 0x0700,
        .sync0_cycle = 1000,
        .sync0_factor = -2,
        .sync0_shift = -100000,
        .sync1_factor = -1000,
        .sync1_shift = 25000,
    };
    auto parser = x::json::Parser(mode.to_json());
    EXPECT_EQ(OpMode::parse(parser), mode);
    EXPECT_TRUE(parser.ok());
}

/// @brief it should report a field that the JSON leaves out.
TEST(OpMode, ParseReportsAMissingField) {
    auto j = OpMode{.name = "DC"}.to_json();
    j.erase("sync1_shift");
    auto parser = x::json::Parser(j);
    OpMode::parse(parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "sync1_shift");
}

}
