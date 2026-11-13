// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/labjack/device/json.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::labjack::device {
/// @brief it should parse a device and apply the schema defaults.
TEST(Device, ParsesWithDefaults) {
    x::json::Parser parser(
        x::json::json{
            {"key", "22222222-2222-4222-8222-222222222222"},
            {"rack", 1},
            {"name", "T7-0001"},
            {"serial", "470010001"},
            {"model", MODEL_T_7},
        }
    );
    const auto parsed = Device::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.serial, "470010001");
    ASSERT_EQ(parsed.model, MODEL_T_7);
    ASSERT_EQ(parsed.identifier, "");
    ASSERT_FALSE(parsed.status.has_value());
}

/// @brief it should reject a device without a serial number.
TEST(Device, RequiresSerial) {
    x::json::Parser parser(
        x::json::json{
            {"key", "22222222-2222-4222-8222-222222222222"},
            {"rack", 1},
            {"name", "T7-0001"},
            {"model", MODEL_T_7},
        }
    );
    Device::parse(parser);
    ASSERT_FALSE(parser.ok());
}

/// @brief it should round trip a device through JSON.
TEST(Device, RoundTrips) {
    const Device device{
        .key = x::uuid::create(),
        .rack = 1,
        .name = "T4-0002",
        .serial = "440010002",
        .model = MODEL_T_4,
        .identifier = "lj",
    };
    x::json::Parser parser(device.to_json());
    const auto parsed = Device::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.key, device.key);
    ASSERT_EQ(parsed.model, MODEL_T_4);
    ASSERT_EQ(parsed.identifier, "lj");
}
}
