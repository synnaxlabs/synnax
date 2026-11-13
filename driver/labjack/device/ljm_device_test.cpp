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

/// @brief it should wait for scans that have not arrived.
TEST(LJMDeviceTest, testStreamReadWaitsForScans) {
    const LJMDevice dev(open_fake_api(), LATE_SCANS_HANDLE);
    ASSERT_NIL(start_stream(dev));
    double value = 0;
    int dev_backlog = 0;
    int ljm_backlog = 0;
    ASSERT_NIL(dev.e_stream_read(&value, &dev_backlog, &ljm_backlog));
    EXPECT_EQ(value, READ_VALUE);
}

/// @brief it should report a lost stream, for which LJM never calls back.
TEST(LJMDeviceTest, testStreamReadReportsLostStream) {
    const LJMDevice dev(open_fake_api(), LOST_STREAM_HANDLE);
    ASSERT_NIL(start_stream(dev));
    double value = 0;
    int dev_backlog = 0;
    int ljm_backlog = 0;
    ASSERT_OCCURRED_AS(
        dev.e_stream_read(&value, &dev_backlog, &ljm_backlog),
        ljm::UNREACHABLE
    );
}

/// @brief it should keep the value of a read that warns of a default calibration, and
/// report the calibration as invalid from then on.
TEST(LJMDeviceTest, testReadWarnsOfDefaultCalibration) {
    const LJMDevice dev(open_fake_api(), DEFAULT_CALIBRATION_HANDLE);
    EXPECT_FALSE(dev.calibration_invalid());
    double value = 0;
    ASSERT_NIL(dev.e_read_name("AIN0", &value));
    EXPECT_EQ(value, READ_VALUE);
    EXPECT_TRUE(dev.calibration_invalid());
    ASSERT_NIL(start_stream(dev));
    EXPECT_TRUE(dev.calibration_invalid());
}

/// @brief it should open a device whose open warns of a default calibration, and report
/// the calibration as invalid.
TEST(LJMManagerTest, testOpenWarnsOfDefaultCalibration) {
    LJMManager manager(open_fake_api());
    const auto dev = ASSERT_NIL_P(manager.acquire(DEFAULT_CALIBRATION_SERIAL));
    ASSERT_NE(dev, nullptr);
    EXPECT_TRUE(dev->calibration_invalid());
}

/// @brief it should report a lost connection as unreachable, with one message for
/// every cause.
TEST(ParseErrorTest, testLostConnectionIsUnreachable) {
    const auto api = open_fake_api();
    for (const auto code:
         {LJME_DEVICE_NOT_FOUND,
          LJME_RECONNECT_FAILED,
          LJME_NO_RESPONSE_BYTES_RECEIVED}) {
        const auto err = ljm::parse_error(api, code);
        ASSERT_OCCURRED_AS(err, ljm::UNREACHABLE);
        EXPECT_EQ(err.data, ljm::UNREACHABLE_MESSAGE);
    }
}

/// @brief it should report an error that a retry or a new stream clears as temporary.
TEST(ParseErrorTest, testRetryableErrorIsTemporary) {
    const auto api = open_fake_api();
    for (const auto code: {LJME_LJM_BUFFER_FULL, STREAM_BUFFER_FULL_CODE}) {
        const auto err = ljm::parse_error(api, code);
        ASSERT_OCCURRED_AS(err, ljm::TEMPORARY_ERROR);
        EXPECT_FALSE(err.matches(ljm::UNREACHABLE));
    }
}

/// @brief it should ignore a warning after which the operation still ran.
TEST(ParseErrorTest, testWarningIsNil) {
    const auto api = open_fake_api();
    ASSERT_NIL(ljm::parse_error(api, LJME_USING_DEFAULT_CALIBRATION));
    ASSERT_NIL(ljm::parse_error(api, LJME_DEBUG_LOG_FAILURE));
}

/// @brief it should report a configuration error, and a warning that skipped part of
/// the operation, as critical.
TEST(ParseErrorTest, testOtherErrorsAreCritical) {
    const auto api = open_fake_api();
    for (const auto code:
         {LJME_INVALID_ADDRESS, LJME_FRAMES_OMITTED_DUE_TO_PACKET_SIZE})
        ASSERT_OCCURRED_AS(ljm::parse_error(api, code), ljm::CRITICAL_ERROR);
}
}
