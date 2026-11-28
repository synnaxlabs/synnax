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

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/filter/field.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/labjack/device/mock.h"
#include "driver/labjack/scan_task.h"
#include "driver/labjack/store.h"

namespace driver::labjack {
namespace lj = synnax::labjack::device;

class ScannerTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client = std::make_shared<synnax::Synnax>(
        new_test_client()
    );
    synnax::rack::Rack rack{.name = "labjack_scan_rack"};
    std::shared_ptr<device::MockManager>
        manager = std::make_shared<device::MockManager>();
    std::unique_ptr<Scanner> scanner;
    /// @brief a serial number that no other test device has. A test adds up to 9 to
    /// it for more devices.
    int serial = 0;

    void SetUp() override {
        ASSERT_NIL(this->client->racks.create(this->rack));
        // Rack keys are unique on a Core, so the serial numbers of each test are too.
        this->serial = static_cast<int>(this->rack.key % 100'000'000) * 10 + 1'000'000;
        this->create(10);
    }

    /// @brief replaces the scanner with one that scans TCP every multiplier scans.
    void create(const int multiplier) {
        auto parser = x::json::Parser(
            x::json::json{{"tcp_scan_multiplier", multiplier}}
        );
        this->scanner = std::make_unique<Scanner>(
            synnax::task::Task{.rack = this->rack.key},
            ScanTaskConfig(parser),
            std::make_shared<x::lib::Lazy<device::Manager>>(
                [this]()
                    -> std::pair<std::shared_ptr<device::Manager>, x::errors::Error> {
                    return {this->manager, x::errors::NIL};
                }
            ),
            std::make_unique<Store>(this->client)
        );
    }

    /// @brief runs the scan after count earlier scans.
    void scan(const std::size_t count = 0) {
        common::Scanner &base = *this->scanner;
        ASSERT_TRUE(
            ASSERT_NIL_P(base.scan(common::ScannerContext{.count = count})).empty()
        );
    }

    /// @brief returns the devices on the rack of the scanner.
    std::vector<Device> stored() {
        return ASSERT_NIL_P(this->client->labjack.devices.retrieve(
            lj::Where{.rack = this->rack.key},
            {.include_status = true}
        ));
    }

    /// @brief returns the device with the serial.
    Device stored(const int serial) {
        return ASSERT_NIL_P(this->client->labjack.devices.retrieve(
            lj::Where{.serial = static_cast<std::uint32_t>(serial)},
            x::filter::one,
            {.include_status = true}
        ));
    }
};

/// @brief it should create each supported device type with its model and name.
TEST_F(ScannerTest, CreatesSupportedModels) {
    this->manager->listed = {
        {.device_type = LJM_dtT4, .serial_number = this->serial},
        {.device_type = LJM_dtT7, .serial_number = this->serial + 1},
        {.device_type = LJM_dtT8, .serial_number = this->serial + 2},
    };
    this->scan();
    const auto t4 = this->stored(this->serial);
    EXPECT_EQ(t4.model, lj::MODEL_T_4);
    EXPECT_TRUE(std::holds_alternative<lj::T4Ports>(t4.ports));
    EXPECT_EQ(t4.rack, this->rack.key);
    EXPECT_EQ(t4.connection, lj::CONNECTION_USB);
    EXPECT_EQ(t4.status.value().message, "Device present");
    const auto last_four = std::to_string(this->serial + 2)
                               .substr(std::to_string(this->serial + 2).size() - 4);
    const auto t8 = this->stored(this->serial + 2);
    EXPECT_EQ(t8.model, lj::MODEL_T_8);
    EXPECT_EQ(t8.name, "T8-" + last_four);
    EXPECT_TRUE(t8.identifier.empty());
    EXPECT_EQ(this->stored(this->serial + 1).model, lj::MODEL_T_7);
}

/// @brief it should skip device types that Synnax does not support.
TEST_F(ScannerTest, SkipsUnsupportedDeviceTypes) {
    this->manager->listed = {
        {.device_type = LJM_dtDIGIT, .serial_number = this->serial},
        {.device_type = LJM_dtT8, .serial_number = this->serial + 1},
    };
    this->scan();
    const auto devices = this->stored();
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].model, lj::MODEL_T_8);
}

/// @brief it should keep reporting a network device between TCP scans, and mark it
/// disconnected after a TCP scan that does not find it.
TEST_F(ScannerTest, KeepsNetworkDevicesBetweenTcpScans) {
    this->create(2);
    this->manager->listed = {
        {.device_type = LJM_dtT7,
         .connection_type = LJM_ctETHERNET,
         .serial_number = this->serial},
    };
    this->scan(0);
    EXPECT_EQ(this->stored(this->serial).connection, lj::CONNECTION_ETHERNET);
    this->manager->listed.clear();
    this->scan(1);
    EXPECT_EQ(this->stored(this->serial).status.value().message, "Device present");
    this->scan(2);
    EXPECT_EQ(this->stored(this->serial).status.value().message, "Device disconnected");
}

/// @brief it should store a device on two connections once, by its USB connection.
TEST_F(ScannerTest, StoresADeviceOnTwoConnectionsOnce) {
    this->manager->listed = {
        {.device_type = LJM_dtT7,
         .connection_type = LJM_ctUSB,
         .serial_number = this->serial},
        {.device_type = LJM_dtT7,
         .connection_type = LJM_ctWIFI,
         .serial_number = this->serial},
    };
    this->scan();
    const auto devices = this->stored();
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].connection, lj::CONNECTION_USB);
}

/// @brief it should keep the name that a user sets when it scans the device again.
TEST_F(ScannerTest, KeepsTheNameAUserSets) {
    this->manager->listed = {
        {.device_type = LJM_dtT7, .serial_number = this->serial},
    };
    this->scan();
    const auto dev = this->stored(this->serial);
    ASSERT_NIL(this->client->labjack.devices.dispatch(
        dev.key,
        {lj::set({.path = {"name"}, .value = "Bench T7"})}
    ));
    this->scan(1);
    EXPECT_EQ(this->stored(this->serial).name, "Bench T7");
}

/// @brief it should parse set and delete signals, skip a set sample that does not
/// parse, and keep the stored device on the next scan.
TEST_F(ScannerTest, HandlesDeviceSignals) {
    this->manager->listed = {
        {.device_type = LJM_dtT7, .serial_number = this->serial},
    };
    this->scan();
    const auto dev = this->stored(this->serial);
    common::Scanner &base = *this->scanner;
    synnax::channel::Channel set_ch{.name = lj::SET_CHANNEL};
    ASSERT_NO_THROW(base.on_signal(set_ch, x::telem::Series(dev.to_json().dump())));
    ASSERT_NO_THROW(base.on_signal(set_ch, x::telem::Series(std::string("{}"))));
    synnax::channel::Channel delete_ch{.name = lj::DELETE_CHANNEL};
    x::telem::Series keys(x::telem::UUID_T, 1);
    keys.write(dev.key.data(), 1);
    ASSERT_NO_THROW(base.on_signal(delete_ch, keys));
    this->scan(1);
    EXPECT_EQ(this->stored(this->serial).key, dev.key);
}

/// @brief it should refuse a channel that is not a LabJack signal.
TEST_F(ScannerTest, RefusesUnknownSignal) {
    common::Scanner &base = *this->scanner;
    synnax::channel::Channel ch{.name = "sy_ni_device_set"};
    ASSERT_THROW(
        base.on_signal(ch, x::telem::Series(std::string("{}"))),
        std::invalid_argument
    );
}

/// @brief it should report a failed write as a partial scan, so the scan count moves.
TEST_F(ScannerTest, ReportsAFailedWriteAsPartial) {
    this->manager->listed = {
        {.device_type = LJM_dtT7, .serial_number = this->serial},
    };
    // The Core refuses a device on a rack that does not exist.
    auto parser = x::json::Parser(x::json::json{{"tcp_scan_multiplier", 10}});
    this->scanner = std::make_unique<Scanner>(
        synnax::task::Task{.rack = 999'999'999},
        ScanTaskConfig(parser),
        std::make_shared<x::lib::Lazy<device::Manager>>(
            [this]() -> std::pair<std::shared_ptr<device::Manager>, x::errors::Error> {
                return {this->manager, x::errors::NIL};
            }
        ),
        std::make_unique<Store>(this->client)
    );
    common::Scanner &base = *this->scanner;
    ASSERT_OCCURRED_AS_P(
        base.scan(common::ScannerContext{}),
        common::PARTIAL_SCAN_ERROR
    );
}

/// @brief it should map only the LJM connection types that a scan reports.
TEST(TypedConnection, MapsScanConnectionTypes) {
    EXPECT_EQ(typed_connection(LJM_ctUSB), lj::CONNECTION_USB);
    EXPECT_EQ(typed_connection(LJM_ctETHERNET), lj::CONNECTION_ETHERNET);
    EXPECT_EQ(typed_connection(LJM_ctWIFI), lj::CONNECTION_WIFI);
    EXPECT_EQ(typed_connection(LJM_ctNETWORK_UDP), std::nullopt);
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
