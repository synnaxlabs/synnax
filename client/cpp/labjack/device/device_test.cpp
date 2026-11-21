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
namespace {
x::json::json valid_device() {
    return x::json::json{
        {"key", "22222222-2222-4222-8222-222222222222"},
        {"rack", 1},
        {"name", "T7-0001"},
        {"serial", 470010001},
        {"connection", CONNECTION_USB},
        {"model", MODEL_T_7},
    };
}
}

/// @brief it should parse the variant that the model names and apply the defaults.
TEST(Device, ParsesTheModelVariant) {
    x::json::Parser parser(valid_device());
    const auto parsed = parse_device(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<T7Device>(parsed));
    const auto &b = base(parsed);
    ASSERT_EQ(b.serial, 470010001);
    ASSERT_EQ(b.connection, CONNECTION_USB);
    ASSERT_EQ(b.identifier, "");
    ASSERT_EQ(b.read_index, 0);
    ASSERT_FALSE(b.status.has_value());
}

/// @brief it should reject a device without a serial number.
TEST(Device, RequiresSerial) {
    auto j = valid_device();
    j.erase("serial");
    x::json::Parser parser(j);
    parse_device(parser);
    ASSERT_FALSE(parser.ok());
    ASSERT_EQ(parser.errors->size(), 1);
    ASSERT_EQ(parser.errors->at(0)["path"], "serial");
}

/// @brief it should round trip a device through JSON.
TEST(Device, RoundTrips) {
    T4Device t4;
    t4.key = x::uuid::create();
    t4.rack = 1;
    t4.name = "T4-0002";
    t4.serial = 440010002;
    t4.connection = CONNECTION_TCP;
    t4.identifier = "lj";
    const Device device = t4;
    x::json::Parser parser(to_json(device));
    const auto parsed = parse_device(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<T4Device>(parsed));
    ASSERT_EQ(base(parsed).key, t4.key);
    ASSERT_EQ(base(parsed).connection, CONNECTION_TCP);
    ASSERT_EQ(base(parsed).identifier, "lj");
}
}
