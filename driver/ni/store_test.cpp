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

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/common/sync.h"
#include "driver/ni/store.h"

namespace driver::ni {
namespace {
/// @brief a Core client and a fresh rack on it.
struct Env {
    std::shared_ptr<synnax::Synnax> client;
    synnax::rack::Rack rack;
};

Env create_env() {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    synnax::rack::Rack rack{.name = "ni_store_rack"};
    EXPECT_NIL(client->racks.create(rack));
    return {std::move(client), std::move(rack)};
}

/// @brief returns the device that a scan finds on the rack.
synnax::ni::device::Device
found_device(const synnax::rack::Key rack, const std::string &serial) {
    return {
        .rack = rack,
        .name = "NI 9205",
        .serial = serial,
        .resource_name = x::uuid::create().to_string(),
        .model = "NI 9205",
    };
}
}

/// @brief it should create the device that a scan finds, with a present status.
TEST(DeviceStore, CreatesFoundDevice) {
    auto [client, rack] = create_env();
    common::Sync<synnax::ni::device::Device> sync(
        std::make_unique<DeviceStore>(client),
        rack.key,
        ""
    );
    const auto serial = x::uuid::create().to_string();
    ASSERT_NIL(sync.sync({found_device(rack.key, serial)}));
    const auto got = ASSERT_NIL_P(client->ni.devices.retrieve(
        synnax::ni::device::Where{.serial = serial},
        x::filter::one,
        {.include_status = true}
    ));
    ASSERT_EQ(got.rack, rack.key);
    ASSERT_EQ(got.status.value().message, "Device present");
    ASSERT_EQ(got.status.value().details.device, got.key);
}

/// @brief it should write the identity group and keep the configuration that a user
/// wrote.
TEST(DeviceStore, KeepsConfigurationOnRescan) {
    auto [client, rack] = create_env();
    common::Sync<synnax::ni::device::Device> sync(
        std::make_unique<DeviceStore>(client),
        rack.key,
        ""
    );
    const auto serial = x::uuid::create().to_string();
    auto found = found_device(rack.key, serial);
    ASSERT_NIL(sync.sync({found}));
    const auto stored = ASSERT_NIL_P(client->ni.devices.retrieve(
        synnax::ni::device::Where{.serial = serial},
        x::filter::one
    ));
    synnax::ni::device::ConfigurationUpdate config;
    config.name = "Thermocouples";
    config.identifier = "tc";
    ASSERT_NIL_P(client->ni.devices.update(stored.key, config));
    found.model = "NI 9205 (rev B)";
    ASSERT_NIL(sync.sync({found}));
    const auto got = ASSERT_NIL_P(client->ni.devices.retrieve(stored.key));
    ASSERT_EQ(got.model, "NI 9205 (rev B)");
    ASSERT_EQ(got.name, "Thermocouples");
    ASSERT_EQ(got.identifier, "tc");
}

/// @brief it should mark a device that a scan no longer finds as disconnected.
TEST(DeviceStore, MarksMissingDevice) {
    auto [client, rack] = create_env();
    common::Sync<synnax::ni::device::Device> sync(
        std::make_unique<DeviceStore>(client),
        rack.key,
        ""
    );
    const auto serial = x::uuid::create().to_string();
    ASSERT_NIL(sync.sync({found_device(rack.key, serial)}));
    ASSERT_NIL(sync.sync({}));
    const auto got = ASSERT_NIL_P(client->ni.devices.retrieve(
        synnax::ni::device::Where{.serial = serial},
        x::filter::one,
        {.include_status = true}
    ));
    ASSERT_EQ(got.status.value().message, "Device disconnected");
}

/// @brief it should find simulated hardware, which has no serial number, by its
/// resource name.
TEST(DeviceStore, FindsByResourceName) {
    auto [client, rack] = create_env();
    auto dev = found_device(rack.key, "");
    ASSERT_NIL(client->ni.devices.create(dev));
    DeviceStore store(client);
    ASSERT_EQ(store.match_key(dev), dev.resource_name);
    const auto got = ASSERT_NIL_P(store.find(dev));
    ASSERT_EQ(got.value().key, dev.key);
}

/// @brief it should prefer the row on the rack of the scan when two rows hold the same
/// hardware.
TEST(DeviceStore, PrefersRowOnScanRack) {
    auto [client, rack] = create_env();
    synnax::rack::Rack other{.name = "ni_store_other_rack"};
    ASSERT_NIL(client->racks.create(other));
    const auto serial = x::uuid::create().to_string();
    auto elsewhere = found_device(other.key, serial);
    ASSERT_NIL(client->ni.devices.create(elsewhere));
    auto here = found_device(rack.key, serial);
    ASSERT_NIL(client->ni.devices.create(here));
    DeviceStore store(client);
    const auto got = ASSERT_NIL_P(store.find(found_device(rack.key, serial)));
    ASSERT_EQ(got.value().key, here.key);
}

/// @brief it should find no device when none matches.
TEST(DeviceStore, FindsNoDevice) {
    auto [client, rack] = create_env();
    DeviceStore store(client);
    const auto got = ASSERT_NIL_P(
        store.find(found_device(rack.key, x::uuid::create().to_string()))
    );
    ASSERT_FALSE(got.has_value());
}

/// @brief it should keep the stored hardware when the scan did not read it.
TEST(DeviceStore, KeepsHardwareTheScanDidNotRead) {
    synnax::ni::device::Device stored{.identifier = "tc"};
    stored.hardware = synnax::ni::device::Hardware{.terminals = {"PFI0"}};
    const synnax::ni::device::Device found{.serial = "01A", .model = "NI 9205"};
    const auto merged = merge_identity(stored, found);
    ASSERT_EQ(merged.hardware.value().terminals.at(0), "PFI0");
    ASSERT_EQ(merged.serial, "01A");
    ASSERT_EQ(merged.identifier, "tc");
}

/// @brief it should create the chassis that a scan finds, with a present status.
TEST(ChassisStore, CreatesFoundChassis) {
    auto [client, rack] = create_env();
    common::Sync<synnax::ni::chassis::Chassis> sync(
        std::make_unique<ChassisStore>(client),
        rack.key,
        ""
    );
    const auto serial = x::uuid::create().to_string();
    ASSERT_NIL(sync.sync({{
        .rack = rack.key,
        .name = "NI cDAQ-9189",
        .serial = serial,
        .resource_name = x::uuid::create().to_string(),
        .model = "cDAQ-9189",
    }}));
    const auto got = ASSERT_NIL_P(client->ni.chassis.retrieve(
        synnax::ni::chassis::Where{.serial = serial},
        x::filter::one,
        {.include_status = true}
    ));
    ASSERT_EQ(got.name, "NI cDAQ-9189");
    ASSERT_EQ(got.status.value().details.chassis, got.key);
}
}
