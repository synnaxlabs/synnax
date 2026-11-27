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

#include "driver/bus/holders.h"
#include "driver/errors/errors.h"

namespace driver::bus {
namespace {
/// @returns a task with name.
synnax::task::Task task(const std::string &name) {
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.name = name;
    return t;
}
}

/// @brief it should refuse a second read task on a device and name the first, and
/// accept the second once the first is gone.
TEST(Readers, RefusesSecondReadTaskOnDevice) {
    Readers readers("serial");
    auto first = ASSERT_NIL_P(readers.claim("boiler", task("oven")));
    const auto [refused, err] = readers.claim("boiler", task("press"));
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data(),
        "A serial device runs one read task at a time, and the task 'oven' reads from "
        "this device. Add these entries to that task."
    );
    const auto other = ASSERT_NIL_P(readers.claim("mixer", task("press")));
    first.reset();
    const auto second = ASSERT_NIL_P(readers.claim("boiler", task("press")));
}

/// @brief it should let a read task claim its device again, and hold the device until
/// each of its claims is gone.
TEST(Readers, ClaimsDeviceAgainForSameTask) {
    Readers readers("serial");
    const auto oven = task("oven");
    auto first = ASSERT_NIL_P(readers.claim("boiler", oven));
    auto again = ASSERT_NIL_P(readers.claim("boiler", oven));
    first.reset();
    ASSERT_OCCURRED_AS_P(
        readers.claim("boiler", task("press")),
        driver::errors::CONFIGURATION_ERROR
    );
    again.reset();
    const auto press = ASSERT_NIL_P(readers.claim("boiler", task("press")));
}
}
