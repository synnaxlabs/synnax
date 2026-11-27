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

/// @brief it should refuse a second read task on a port path and name the first, and
/// accept the second once the first is gone.
TEST(Ports, RefusesSecondReadTaskOnPath) {
    Ports ports;
    const auto path = missing_port();
    auto first = ASSERT_NIL_P(ports.claim_read(path, task("oven")));
    const auto [refused, err] = ports.claim_read(path, task("press"));
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "A serial port runs one read task at a time, and the task 'oven' reads from " +
            path + ". Add these entries to that task."
    );
    first.reset();
    const auto second = ASSERT_NIL_P(ports.claim_read(path, task("press")));
}

/// @brief it should refuse the port of a device to a second device and name the first,
/// and give it to the second once no task on the first holds it.
TEST(Ports, RefusesPortOfDeviceToSecondDevice) {
    Ports ports;
    auto first = ASSERT_NIL_P(ports.acquire(device("oven")));
    auto again = ASSERT_NIL_P(ports.acquire(device("oven")));
    const auto [refused, err] = ports.acquire(device("press"));
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    ASSERT_EQ(
        err.data,
        "The device 'oven' already uses serial port " + missing_port() +
            ". Two devices cannot share one port. Change the port of one of them."
    );
    first = {};
    ASSERT_OCCURRED_AS_P(ports.acquire(device("press")), transport::CONFIG_ERROR);
    again = {};
    const auto second = ASSERT_NIL_P(ports.acquire(device("press")));
}
}
