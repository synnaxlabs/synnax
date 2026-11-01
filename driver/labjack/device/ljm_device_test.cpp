// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <functional>
#include <memory>
#include <utility>

#include "gmock/gmock.h"
#include "gtest/gtest.h"

#include "x/cpp/lib/lib.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/labjack/device/device.h"
#include "driver/labjack/device/fake_ljm.h"

namespace driver::labjack::device {
const auto FAKE_FAILURE = ljm::CRITICAL_ERROR.sub("FAKE_FAILURE");
const int WORKING_HANDLE = 0;

/// @brief returns the LJM API over the fake library.
std::shared_ptr<ljm::API> open_fake_api() {
    auto lib = std::make_unique<x::lib::Shared>(
        "./driver/labjack/device/libfake_ljm.so"
    );
    EXPECT_TRUE(lib->load());
    return std::make_shared<ljm::API>(std::move(lib));
}

/// @brief starts a stream on the device.
x::errors::Error start_stream(
    const LJMDevice &dev,
    const std::function<x::errors::Error()> &configure = [] { return x::errors::NIL; }
) {
    double scan_rate = 1000;
    return dev.e_stream_start(1, 0, nullptr, &scan_rate, configure);
}

/// @brief it should refuse a second stream, tell the user what to do, and allow a new
/// stream after a stop.
TEST(LJMDeviceTest, testSecondStreamStartIsRefused) {
    const LJMDevice dev(open_fake_api(), WORKING_HANDLE);
    ASSERT_NIL(start_stream(dev));
    const auto err = start_stream(dev);
    ASSERT_OCCURRED_AS(err, ljm::STREAM_IS_ACTIVE);
    EXPECT_THAT(
        err.data,
        ::testing::HasSubstr("Add these channels to the running task.")
    );
    ASSERT_NIL(dev.e_stream_stop());
    ASSERT_NIL(start_stream(dev));
}

/// @brief it should leave the device untouched when it refuses a stream.
TEST(LJMDeviceTest, testRefusedStartDoesNotConfigure) {
    const LJMDevice dev(open_fake_api(), WORKING_HANDLE);
    ASSERT_NIL(start_stream(dev));
    bool configured = false;
    ASSERT_OCCURRED_AS(
        start_stream(
            dev,
            [&configured] {
                configured = true;
                return x::errors::NIL;
            }
        ),
        ljm::STREAM_IS_ACTIVE
    );
    EXPECT_FALSE(configured);
}

/// @brief it should allow a new stream after a configure that failed.
TEST(LJMDeviceTest, testFailedConfigureAllowsNextStart) {
    const LJMDevice dev(open_fake_api(), WORKING_HANDLE);
    ASSERT_OCCURRED_AS(start_stream(dev, [] { return FAKE_FAILURE; }), FAKE_FAILURE);
    ASSERT_NIL(start_stream(dev));
}

/// @brief it should attempt a new stream after a start that failed.
TEST(LJMDeviceTest, testFailedStartAllowsNextStart) {
    const LJMDevice dev(open_fake_api(), FAILING_START_HANDLE);
    ASSERT_OCCURRED_AS(start_stream(dev), FAKE_FAILURE);
    ASSERT_OCCURRED_AS(start_stream(dev), FAKE_FAILURE);
}

/// @brief it should allow a new stream after a stop that failed.
TEST(LJMDeviceTest, testFailedStopAllowsNextStart) {
    const LJMDevice dev(open_fake_api(), FAILING_STOP_HANDLE);
    ASSERT_NIL(start_stream(dev));
    ASSERT_OCCURRED_AS(dev.e_stream_stop(), FAKE_FAILURE);
    ASSERT_NIL(start_stream(dev));
}
}
