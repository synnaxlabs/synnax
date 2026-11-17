// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/ni/device/json.gen.h"
#include "client/cpp/ni/device/proto.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::ni::device {
/// @brief it should round trip a port map keyed by port and line number through JSON.
TEST(DigitalOutputChannels, RoundTripsLines) {
    Configuration config{.name = "Module"};
    config.digital_output[2] = DigitalOutputChannels{
        .state_index = 1,
        .lines = {{7, CommandStatePair{.command = 2, .state = 3}}},
    };
    const auto j = config.to_json();
    ASSERT_EQ(j["digital_output"]["2"]["lines"]["7"]["command"], 2);
    x::json::Parser parser(j);
    const auto parsed = Configuration::parse(parser);
    ASSERT_NIL(parser.error());
    const auto &port = parsed.digital_output.at(2);
    ASSERT_EQ(port.state_index, 1);
    ASSERT_EQ(port.lines.at(7).command, 2);
    ASSERT_EQ(port.lines.at(7).state, 3);
}

/// @brief it should round trip the analog input channels keyed by port number.
TEST(AnalogInputPorts, RoundTripsChannels) {
    const AnalogInputPorts ports{.index = 1, .temperature = 2, .channels = {{3, 4}}};
    x::json::Parser parser(ports.to_json());
    const auto parsed = AnalogInputPorts::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.temperature, 2);
    ASSERT_EQ(parsed.channels.at(3), 4);
}

/// @brief it should default the port groups of a configuration that omits them.
TEST(Configuration, ParsesMissingPortGroups) {
    x::json::Parser parser(x::json::json{{"name", "Module"}});
    const auto parsed = Configuration::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.name, "Module");
    ASSERT_TRUE(parsed.analog_input.channels.empty());
    ASSERT_TRUE(parsed.digital_output.empty());
}

/// @brief it should keep the field group of an update through JSON.
TEST(Update, RoundTripsGroup) {
    IdentityUpdate identity;
    identity.rack = 1;
    identity.serial = "01A";
    identity.resource_name = "Mod1";
    const auto j = to_json(Update(identity));
    ASSERT_EQ(j["group"], "identity");
    x::json::Parser parser(j);
    const auto parsed = parse_update(parser);
    ASSERT_TRUE(parser.ok());
    ASSERT_EQ(std::get<IdentityUpdate>(parsed).serial, "01A");
}

/// @brief it should write an empty object for a port map with no ports.
TEST(Configuration, SerializesEmptyPortMapsAsObjects) {
    const auto j = Configuration{}.to_json();
    ASSERT_TRUE(j["analog_input"]["channels"].is_object());
    ASSERT_TRUE(j["digital_output"].is_object());
}

/// @brief it should round trip a device with its port maps through protobuf.
TEST(Device, RoundTripsProto) {
    Device dev{
        .key = x::uuid::create(),
        .rack = 65537,
        .name = "Module",
        .serial = "01A",
        .resource_name = "{GUID}",
        .model = "NI 9205",
        .chassis = x::uuid::create(),
        .identifier = "mod1",
    };
    dev.analog_input.channels[3] = 4;
    dev.counter_input[2] = CounterInputChannels{.index = 8, .channel = 9};
    dev.digital_output[1].lines[255] = CommandStatePair{.command = 13, .state = 14};
    const auto pb = ASSERT_NIL_P(dev.to_proto());
    const auto parsed = ASSERT_NIL_P(Device::from_proto(pb));
    ASSERT_EQ(parsed.to_json(), dev.to_json());
}

/// @brief it should reject a line number that does not fit in a byte.
TEST(DigitalInputChannels, RejectsOutOfRangeLineFromProto) {
    ::service::ni_device::pb::DigitalInputChannels pb;
    (*pb.mutable_lines())[256] = 1;
    ASSERT_OCCURRED_AS_P(DigitalInputChannels::from_proto(pb), x::errors::VALIDATION);
}
}
