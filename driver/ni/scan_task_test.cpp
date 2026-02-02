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

/// @brief it should parse scan task configuration with defaults and custom values.
TEST(ScanTaskTest, testConfigParse) {
    // Test default configuration
    json j = {{"enabled", true}};
    auto p = x::json::Parser(j);
    driver::ni::ScanTaskConfig cfg(p);

    EXPECT_TRUE(cfg.enabled);
    EXPECT_EQ(cfg.scan_rate.hz(), driver::task::common::DEFAULT_SCAN_RATE.hz());
    EXPECT_EQ(cfg.ignored_models.size(), driver::ni::DEFAULT_IGNORED_MODELS.size());

    // Test custom configuration
    json j2 = {
        {"enabled", false},
        {"rate", 10.0},
        {"ignored_models", json::array({"^Test.*", "^Mock.*"})}
    };
    auto p2 = x::json::Parser(j2);
    driver::ni::ScanTaskConfig cfg2(p2);

    EXPECT_FALSE(cfg2.enabled);
    EXPECT_EQ(cfg2.scan_rate.hz(), 10.0);
    EXPECT_EQ(cfg2.ignored_models.size(), 2);
}

/// @brief it should correctly identify models to ignore based on regex patterns.
TEST(ScanTaskTest, testConfigShouldIgnore) {
    json j = {{"ignored_models", json::array({"^Test.*", "^Mock.*", "PXI-.*"})}};
    auto p = x::json::Parser(j);
    driver::ni::ScanTaskConfig cfg(p);

    // Should ignore models matching the patterns
    EXPECT_TRUE(cfg.should_ignore("TestDevice"));
    EXPECT_TRUE(cfg.should_ignore("MockDevice123"));
    EXPECT_TRUE(cfg.should_ignore("PXI-6255"));

    // Should not ignore models not matching the patterns
    EXPECT_FALSE(cfg.should_ignore("NI-DAQ"));
    EXPECT_FALSE(cfg.should_ignore("Regular-Device"));
    EXPECT_FALSE(cfg.should_ignore("cDAQ-9178"));

    // Test with default configuration
    json j2 = {};
    auto p2 = x::json::Parser(j2);
    driver::ni::ScanTaskConfig cfg2(p2);

    // Should ignore models matching default patterns
    EXPECT_TRUE(cfg2.should_ignore("cRIO-9068"));
    EXPECT_TRUE(cfg2.should_ignore("nownDevice"));

    // Should not ignore models not matching default patterns
    EXPECT_FALSE(cfg2.should_ignore("PXI-6255"));
    EXPECT_FALSE(cfg2.should_ignore("NI-DAQ"));
}
