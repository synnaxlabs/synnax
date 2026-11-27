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

#include "driver/serial/ports.h"
#include "driver/transport/errors.h"

namespace driver::serial {
/// @returns a port path that names no device on this platform.
std::string missing_port();

namespace {
/// @returns a device with name on the missing port.
Device device(const std::string &name) {
    Device dev{.key = name + "_key", .name = name};
    dev.props.port = missing_port();
    return dev;
}
}

/// @brief it should refuse a second task on a device that asks for other properties.
TEST(Ports, RefusesOtherPropertiesOnSameDevice) {
    Ports ports;
    const auto first = ASSERT_NIL_P(ports.acquire(device("oven")));
    auto other = device("oven");
    other.props.baud_rate = 115200;
    const auto [refused, err] = ports.acquire(other);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    ASSERT_EQ(err.data(), "another task has device oven_key open with other settings");
}
}
