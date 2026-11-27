// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/port.h"
#include "driver/transport/errors.h"

namespace driver::serial {
/// @returns a port path that names no device on this platform.
std::string missing_port();

/// @brief it should fail to open a port that does not exist.
TEST(Port, FailsToOpenAMissingPort) {
    synnax::serial::Properties p;
    p.port = missing_port();
    const auto [port, err] = Port::open(p);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(err.data().starts_with("failed to open serial port " + p.port + ": "))
        << err.data();
}

/// @brief it should refuse invalid settings before it opens the port.
TEST(Port, RefusesInvalidSettingsBeforeItOpensThePort) {
    synnax::serial::Properties p;
    p.port = missing_port();
    p.data_bits = 9;
    const auto [port, err] = Port::open(p);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_EQ(err.data(), "data bits must be 5, 6, 7, or 8, got 9");
}

/// @brief it should refuse an empty port path.
TEST(Port, RejectsAnEmptyPath) {
    const auto [port, err] = Port::open(synnax::serial::Properties{});
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_EQ(err.data(), "serial port path is required");
}
}
