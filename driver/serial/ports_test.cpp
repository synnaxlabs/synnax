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

#include "client/cpp/task/task.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/errors/errors.h"
#include "driver/serial/ports.h"
#include "driver/transport/errors.h"

namespace driver::serial {
/// @returns a port path that names no device on this platform.
std::string missing_port();

namespace {
/// @returns a task with name.
synnax::task::Task task(const std::string &name) {
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.name = name;
    return t;
}

/// @returns a device with name on the missing port.
Device device(const std::string &name) {
    Device dev{.key = name + "_key", .name = name};
    dev.props.port = missing_port();
    return dev;
}
}

/// @brief it should refuse a second read task on a device and name the first, and
/// accept the second once the first is gone.
TEST(Ports, RefusesSecondReadTaskOnDevice) {
    Ports ports;
    auto first = ASSERT_NIL_P(ports.claim_read("boiler", task("oven")));
    const auto [refused, err] = ports.claim_read("boiler", task("press"));
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "A serial device runs one read task at a time, and the task 'oven' reads from "
        "this device. Add these entries to that task."
    );
    const auto other = ASSERT_NIL_P(ports.claim_read("mixer", task("press")));
    first.reset();
    const auto second = ASSERT_NIL_P(ports.claim_read("boiler", task("press")));
}

/// @brief it should let a read task claim its device again, and hold the device until
/// each of its claims is gone.
TEST(Ports, ClaimsDeviceAgainForSameTask) {
    Ports ports;
    const auto oven = task("oven");
    auto first = ASSERT_NIL_P(ports.claim_read("boiler", oven));
    auto again = ASSERT_NIL_P(ports.claim_read("boiler", oven));
    first.reset();
    ASSERT_OCCURRED_AS_P(
        ports.claim_read("boiler", task("press")),
        driver::errors::CONFIGURATION_ERROR
    );
    again.reset();
    const auto press = ASSERT_NIL_P(ports.claim_read("boiler", task("press")));
}

/// @brief it should refuse a second task on a device that asks for other properties.
TEST(Ports, RefusesOtherPropertiesOnSameDevice) {
    Ports ports;
    const auto first = ASSERT_NIL_P(ports.acquire(device("oven")));
    auto other = device("oven");
    other.props.baud_rate = 115200;
    const auto [refused, err] = ports.acquire(other);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    ASSERT_EQ(err.data, "another task has device oven_key open with other settings");
}
}
