// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/labjack/device/device.h"
#include "driver/labjack/device/mock.h"

namespace driver::labjack::device {
const std::string SERIAL_NUMBER = "470037383";

/// @brief returns an opener that serves dev and counts its calls in opens.
Opener serve(const std::shared_ptr<Device> &dev, size_t &opens) {
    return [dev, &opens](
               const std::string &
           ) -> std::pair<std::shared_ptr<Device>, x::errors::Error> {
        ++opens;
        return {dev, x::errors::NIL};
    };
}

/// @brief starts a stream on the device.
void start_stream(const std::shared_ptr<Device> &dev) {
    double scan_rate = 1000;
    ASSERT_NIL(dev->e_stream_start(1, 0, nullptr, &scan_rate));
}

/// @brief it should stop a stream that another process left running on a device it
/// opens.
TEST(LJMManagerTest, testAcquireStopsLeftoverStream) {
    const auto dev = std::make_shared<Mock>();
    start_stream(dev);
    size_t opens = 0;
    LJMManager devs(nullptr, serve(dev, opens));
    ASSERT_NIL_P(devs.acquire(SERIAL_NUMBER));
    ASSERT_OCCURRED_AS(dev->e_stream_stop(), ljm::STREAM_NOT_RUNNING);
}

/// @brief it should open a device that is not streaming.
TEST(LJMManagerTest, testAcquireOpensIdleDevice) {
    const auto dev = std::make_shared<Mock>();
    size_t opens = 0;
    LJMManager devs(nullptr, serve(dev, opens));
    const auto acquired = ASSERT_NIL_P(devs.acquire(SERIAL_NUMBER));
    EXPECT_EQ(acquired, dev);
    EXPECT_EQ(opens, 1);
}

/// @brief it should return the error of a failed stream stop, keep no handle, and open
/// the device again on the next acquire.
TEST(LJMManagerTest, testAcquireReturnsStreamStopError) {
    const auto dev = std::make_shared<Mock>();
    dev->set_should_fail(true);
    size_t opens = 0;
    LJMManager devs(nullptr, serve(dev, opens));
    ASSERT_OCCURRED_AS_P(devs.acquire(SERIAL_NUMBER), x::errors::Error("mock failure"));
    dev->set_should_fail(false);
    ASSERT_NIL_P(devs.acquire(SERIAL_NUMBER));
    EXPECT_EQ(opens, 2);
}

/// @brief it should leave the stream of a device that another caller holds running.
TEST(LJMManagerTest, testAcquireLeavesHeldStreamRunning) {
    const auto dev = std::make_shared<Mock>();
    size_t opens = 0;
    LJMManager devs(nullptr, serve(dev, opens));
    const auto first = ASSERT_NIL_P(devs.acquire(SERIAL_NUMBER));
    start_stream(first);
    const auto second = ASSERT_NIL_P(devs.acquire(SERIAL_NUMBER));
    EXPECT_EQ(second, first);
    EXPECT_EQ(opens, 1);
    ASSERT_NIL(first->e_stream_stop());
}
}
