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
        {"ports", {{"model", MODEL_T_7}}},
    };
}
}

/// @brief it should parse the ports that the model names and apply the defaults.
TEST(Device, ParsesThePortsOfTheModel) {
    x::json::Parser parser(valid_device());
    const auto d = Device::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(d.model, MODEL_T_7);
    ASSERT_TRUE(std::holds_alternative<T7Ports>(d.ports));
    ASSERT_FALSE(std::get<T7Ports>(d.ports).mux80_attached);
    ASSERT_EQ(d.serial, 470010001);
    ASSERT_EQ(d.connection, CONNECTION_USB);
    ASSERT_EQ(d.identifier, "");
    ASSERT_EQ(d.read_index, 0);
    ASSERT_FALSE(d.status.has_value());
}

/// @brief it should reject a device without a serial number.
TEST(Device, RequiresSerial) {
    auto j = valid_device();
    j.erase("serial");
    x::json::Parser parser(j);
    Device::parse(parser);
    ASSERT_FALSE(parser.ok());
    ASSERT_EQ(parser.errors->size(), 1);
    ASSERT_EQ(parser.errors->at(0)["path"], "serial");
}

/// @brief it should round trip a device through JSON.
TEST(Device, RoundTrips) {
    Device d;
    d.key = x::uuid::create();
    d.rack = 1;
    d.name = "T4-0002";
    d.serial = 440010002;
    d.connection = CONNECTION_ETHERNET;
    d.identifier = "lj";
    d.model = MODEL_T_4;
    d.ports = T4Ports{};
    x::json::Parser parser(d.to_json());
    const auto parsed = Device::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.key, d.key);
    ASSERT_EQ(parsed.model, MODEL_T_4);
    ASSERT_TRUE(std::holds_alternative<T4Ports>(parsed.ports));
    ASSERT_EQ(parsed.connection, CONNECTION_ETHERNET);
    ASSERT_EQ(parsed.identifier, "lj");
}

/// @brief it should round trip a configuration update with a T7's ports.
TEST(Update, RoundTripsConfiguration) {
    ConfigurationUpdate configuration;
    configuration.name = "T7-0001";
    T7Ports ports;
    ports.mux80_attached = true;
    configuration.ports = ports;
    const Update update = configuration;
    x::json::Parser parser(to_json(update));
    const auto parsed = parse_update(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<ConfigurationUpdate>(parsed));
    const auto &got = std::get<ConfigurationUpdate>(parsed);
    ASSERT_EQ(got.name, "T7-0001");
    ASSERT_TRUE(std::holds_alternative<T7Ports>(got.ports));
    ASSERT_TRUE(std::get<T7Ports>(got.ports).mux80_attached);
}
}
