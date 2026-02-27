// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/json/json.h"

#include "driver/ni/scan_task.h"

namespace driver::ni {
/// @brief it should parse scan task configuration with defaults and custom values.
TEST(ScanTaskTest, testConfigParse) {
    // Test default configuration
    x::json::json j = {{"enabled", true}};
    auto p = x::json::Parser(j);
    ScanTaskConfig cfg(p);

    EXPECT_TRUE(cfg.enabled);
    EXPECT_EQ(cfg.scan_rate.hz(), common::DEFAULT_SCAN_RATE.hz());
    EXPECT_EQ(cfg.ignored_models.size(), DEFAULT_IGNORED_MODELS.size());

    // Test custom configuration
    x::json::json j2 = {
        {"enabled", false},
        {"rate", 10.0},
        {"ignored_models", x::json::json::array({"^Test.*", "^Mock.*"})}
    };
    auto p2 = x::json::Parser(j2);
    ScanTaskConfig cfg2(p2);

    EXPECT_FALSE(cfg2.enabled);
    EXPECT_EQ(cfg2.scan_rate.hz(), 10.0);
    EXPECT_EQ(cfg2.ignored_models.size(), 2);
}

/// @brief it should correctly identify models to ignore based on regex patterns.
TEST(ScanTaskTest, testConfigShouldIgnore) {
    x::json::json j = {
        {"ignored_models", x::json::json::array({"^Test.*", "^Mock.*", "PXI-.*"})}
    };
    auto p = x::json::Parser(j);
    ScanTaskConfig cfg(p);

    // Should ignore models matching the patterns
    EXPECT_TRUE(cfg.should_ignore("TestDevice"));
    EXPECT_TRUE(cfg.should_ignore("MockDevice123"));
    EXPECT_TRUE(cfg.should_ignore("PXI-6255"));

    // Should not ignore models not matching the patterns
    EXPECT_FALSE(cfg.should_ignore("NI-DAQ"));
    EXPECT_FALSE(cfg.should_ignore("Regular-Device"));
    EXPECT_FALSE(cfg.should_ignore("cDAQ-9178"));

    // Test with default configuration
    x::json::json j2 = {};
    auto p2 = x::json::Parser(j2);
    ScanTaskConfig cfg2(p2);

    // Should ignore models matching default patterns
    EXPECT_TRUE(cfg2.should_ignore("cRIO-9068"));
    EXPECT_TRUE(cfg2.should_ignore("nownDevice"));

    // Should not ignore models not matching default patterns
    EXPECT_FALSE(cfg2.should_ignore("PXI-6255"));
    EXPECT_FALSE(cfg2.should_ignore("NI-DAQ"));
}

/// @brief to_synnax should include parent_device from the base class.
TEST(NiDeviceTests, testToSynnaxIncludesParentDevice) {
    ni::Device dev;
    dev.key = "module-1";
    dev.name = "NI 9205";
    dev.rack = 1;
    dev.location = "Slot2";
    dev.make = "NI";
    dev.model = "9205";
    dev.resource_name = "cDAQ1Mod1";
    dev.is_simulated = false;
    dev.is_chassis = false;
    dev.parent_device = "chassis-serial-123";

    auto synnax_dev = dev.to_synnax();
    EXPECT_EQ(synnax_dev.parent_device, "chassis-serial-123");
    EXPECT_EQ(synnax_dev.key, "module-1");
}

/// @brief to_synnax should leave parent_device empty when not set.
TEST(NiDeviceTests, testToSynnaxEmptyParentDevice) {
    ni::Device dev;
    dev.key = "standalone-1";
    dev.name = "NI 6255";
    dev.rack = 1;
    dev.location = "PXI1Slot2";
    dev.make = "NI";
    dev.model = "6255";

    auto synnax_dev = dev.to_synnax();
    EXPECT_EQ(synnax_dev.parent_device, "");
}

/// @brief to_synnax should include is_chassis in properties JSON.
TEST(NiDeviceTests, testToSynnaxIncludesIsChassisInProperties) {
    ni::Device chassis;
    chassis.key = "chassis-1";
    chassis.name = "NI cDAQ-9178";
    chassis.rack = 1;
    chassis.location = "cDAQ1";
    chassis.make = "NI";
    chassis.model = "cDAQ-9178";
    chassis.is_chassis = true;
    chassis.is_simulated = false;
    chassis.resource_name = "cDAQ1";

    auto synnax_dev = chassis.to_synnax();
    x::json::json props(synnax_dev.properties);
    EXPECT_TRUE(props.contains("is_chassis"));
    EXPECT_TRUE(props["is_chassis"].get<bool>());
    EXPECT_TRUE(props.contains("is_simulated"));
    EXPECT_FALSE(props["is_simulated"].get<bool>());
    EXPECT_EQ(props["resource_name"].get<std::string>(), "cDAQ1");
}

/// @brief to_synnax should set is_chassis to false for modules.
TEST(NiDeviceTests, testToSynnaxModuleIsChassisIsFalse) {
    ni::Device module;
    module.key = "module-2";
    module.name = "NI 9205";
    module.rack = 1;
    module.location = "Slot2";
    module.make = "NI";
    module.model = "9205";
    module.is_chassis = false;

    auto synnax_dev = module.to_synnax();
    x::json::json props(synnax_dev.properties);
    EXPECT_FALSE(props["is_chassis"].get<bool>());
}

/// @brief to_synnax should preserve status through conversion.
TEST(NiDeviceTests, testToSynnaxPreservesStatus) {
    ni::Device dev;
    dev.key = "status-dev";
    dev.name = "NI 9205";
    dev.rack = 1;
    dev.location = "Slot1";
    dev.make = "NI";
    dev.model = "9205";
    dev.status.variant = x::status::VARIANT_SUCCESS;
    dev.status.message = "Device present";

    auto synnax_dev = dev.to_synnax();
    EXPECT_EQ(synnax_dev.status.variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(synnax_dev.status.message, "Device present");
}
}
