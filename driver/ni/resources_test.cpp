// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <stdexcept>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/ni/device/device.h"
#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/ni/resources.h"
#include "driver/ni/store.h"

namespace driver::ni {
namespace {
/// @brief returns hardware that a scan found, keyed as the scanner keys it.
Device scanned(const synnax::rack::Key rack, const bool simulated) {
    Device dev;
    dev.rack = rack;
    dev.is_simulated = simulated;
    dev.resource_name = x::uuid::create().to_string();
    dev.key = simulated ? dev.resource_name : x::uuid::create().to_string();
    dev.model = "9205";
    dev.name = "NI 9205";
    return dev;
}

std::unique_ptr<Resources> create_resources(
    const std::shared_ptr<synnax::Synnax> &client,
    synnax::rack::Key rack
) {
    return std::make_unique<Resources>(
        std::make_unique<ChassisStore>(client),
        std::make_unique<DeviceStore>(client),
        rack
    );
}
}

/// @brief it should split chassis from devices and put each module under its chassis.
TEST(FoundDevices, PutsModulesUnderTheirChassis) {
    auto chassis = scanned(1, false);
    chassis.is_chassis = true;
    chassis.provides_link_name = "link";
    auto module = scanned(1, false);
    module.connects_to_link_name = "link";
    const auto standalone = scanned(1, true);
    const std::vector scan{chassis, module, standalone};
    const auto chassis_key = x::uuid::create();
    ASSERT_EQ(found_chassis(scan).size(), 1);
    ASSERT_EQ(found_chassis(scan).at(0).serial, chassis.key);
    const auto devices = found_devices(scan, [&](const std::string &match) {
        return match == chassis.key ? std::optional(chassis_key) : std::nullopt;
    });
    ASSERT_EQ(devices.size(), 2);
    ASSERT_EQ(devices.at(0).chassis, chassis_key);
    ASSERT_EQ(devices.at(0).serial, module.key);
    ASSERT_FALSE(devices.at(1).chassis.has_value());
    ASSERT_EQ(devices.at(1).serial, "");
    ASSERT_EQ(devices.at(1).resource_name, standalone.resource_name);
}

/// @brief it should store a scanned chassis and its module under it.
TEST(Resources, StoresChassisAndModule) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    synnax::rack::Rack rack{.name = "ni_resources_rack"};
    ASSERT_NIL(client->racks.create(rack));
    auto chassis = scanned(rack.key, false);
    chassis.is_chassis = true;
    chassis.provides_link_name = "link";
    auto module = scanned(rack.key, false);
    module.connects_to_link_name = "link";
    module.hardware = synnax::ni::device::Hardware{.terminals = {"PFI0"}};
    const auto resources = create_resources(client, rack.key);
    ASSERT_NIL(resources->sync({chassis, module}));
    const auto stored_chassis = ASSERT_NIL_P(client->ni.chassis.retrieve(
        synnax::ni::chassis::Where{.serial = chassis.key},
        x::filter::one
    ));
    const auto stored_module = ASSERT_NIL_P(client->ni.devices.retrieve(
        synnax::ni::device::Where{.serial = module.key},
        x::filter::one
    ));
    ASSERT_EQ(stored_module.chassis, stored_chassis.key);
    ASSERT_EQ(resources->stored_hardware(module).value().terminals.at(0), "PFI0");
}

/// @brief it should track a device that a set signal carries.
TEST(Resources, TracksSetSignal) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    synnax::rack::Rack rack{.name = "ni_resources_rack"};
    ASSERT_NIL(client->racks.create(rack));
    const auto dev = scanned(rack.key, true);
    const auto resources = create_resources(client, rack.key);
    ASSERT_NIL(resources->sync({dev}));
    auto stored = ASSERT_NIL_P(client->ni.devices.retrieve(
        synnax::ni::device::Where{.resource_name = dev.resource_name},
        x::filter::one
    ));
    stored.hardware = synnax::ni::device::Hardware{.terminals = {"PFI1"}};
    synnax::channel::Channel set_ch;
    set_ch.name = synnax::ni::device::SET_CHANNEL;
    resources->on_signal(set_ch, x::telem::Series(stored.to_json().dump()));
    ASSERT_EQ(resources->stored_hardware(dev).value().terminals.at(0), "PFI1");
}

/// @brief it should refuse a channel that is not an NI signal.
TEST(Resources, RefusesUnknownSignal) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto resources = create_resources(client, 1);
    synnax::channel::Channel ch;
    ch.name = "sy_device_set";
    ASSERT_THROW(
        resources->on_signal(ch, x::telem::Series(std::string("{}"))),
        std::invalid_argument
    );
}
}
