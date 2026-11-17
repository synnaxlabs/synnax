// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/labjack/device/mock.h"
#include "driver/labjack/scan_task.h"

namespace driver::labjack {
class ScannerTest : public ::testing::Test {
protected:
    std::shared_ptr<device::MockManager>
        manager = std::make_shared<device::MockManager>();
    std::unique_ptr<Scanner> scanner;

    void SetUp() override { this->create(10); }

    /// @brief replaces the scanner with one that scans TCP every multiplier scans.
    void create(const int multiplier) {
        auto parser = x::json::Parser(
            x::json::json{{"tcp_scan_multiplier", multiplier}}
        );
        this->scanner = std::make_unique<Scanner>(
            synnax::task::Task{.rack = 1},
            ScanTaskConfig(parser),
            std::make_shared<x::lib::Lazy<device::Manager>>(
                [this]()
                    -> std::pair<std::shared_ptr<device::Manager>, x::errors::Error> {
                    return {this->manager, x::errors::NIL};
                }
            )
        );
    }

    /// @brief runs the scan after count earlier scans.
    std::vector<synnax::device::Device> scan(const std::size_t count = 0) {
        common::Scanner &base = *this->scanner;
        return ASSERT_NIL_P(base.scan(common::ScannerContext{.count = count}));
    }
};

/// @brief it should name each supported device type by its model.
TEST_F(ScannerTest, NamesSupportedModels) {
    this->manager->listed = {
        {.device_type = LJM_dtT4, .serial_number = 440010001},
        {.device_type = LJM_dtT7, .serial_number = 470010002},
        {.device_type = LJM_dtT8, .serial_number = 480010003},
    };
    const auto devices = this->scan();
    ASSERT_EQ(devices.size(), 3);
    EXPECT_EQ(devices[0].model, T4);
    EXPECT_EQ(devices[0].name, "LJM_dtT4-0001");
    EXPECT_EQ(devices[1].model, T7);
    EXPECT_EQ(devices[1].name, "LJM_dtT7-0002");
    EXPECT_EQ(devices[2].model, T8);
    EXPECT_EQ(devices[2].key, "480010003");
    EXPECT_EQ(devices[2].name, "LJM_dtT8-0003");
    EXPECT_EQ(devices[2].location, "LJM_ctUSB");
}

/// @brief it should skip device types that Synnax does not support.
TEST_F(ScannerTest, SkipsUnsupportedDeviceTypes) {
    this->manager->listed = {
        {.device_type = LJM_dtDIGIT, .serial_number = 200010001},
        {.device_type = LJM_dtT8, .serial_number = 480010002},
    };
    const auto devices = this->scan();
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].model, T8);
}

/// @brief it should keep reporting a network device between TCP scans, and drop it
/// after a TCP scan that does not find it.
TEST_F(ScannerTest, KeepsNetworkDevicesBetweenTcpScans) {
    this->create(2);
    this->manager->listed = {
        {.device_type = LJM_dtT7, .connection_type = LJM_ctTCP, .serial_number = 7},
    };
    ASSERT_EQ(this->scan(0).size(), 1);
    this->manager->listed.clear();
    const auto between = this->scan(1);
    ASSERT_EQ(between.size(), 1);
    EXPECT_EQ(between[0].key, "7");
    EXPECT_TRUE(this->scan(2).empty());
}

/// @brief it should report a device on two connections once, by its USB connection.
TEST_F(ScannerTest, ListsADeviceOnTwoConnectionsOnce) {
    this->manager->listed = {
        {.device_type = LJM_dtT7, .connection_type = LJM_ctUSB, .serial_number = 7},
        {.device_type = LJM_dtT7, .connection_type = LJM_ctTCP, .serial_number = 7},
    };
    const auto devices = this->scan();
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].location, "LJM_ctUSB");
}

/// @brief it should reject a TCP scan multiplier that is not positive.
TEST(ScanTaskConfigTest, RejectsNonPositiveTcpScanMultiplier) {
    for (const auto multiplier: {0, -1}) {
        auto parser = x::json::Parser(
            x::json::json{{"tcp_scan_multiplier", multiplier}}
        );
        ScanTaskConfig cfg(parser);
        ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
        EXPECT_EQ(parser.error_json()["errors"][0]["path"], "tcp_scan_multiplier");
    }
}
}
