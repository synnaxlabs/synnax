// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/ni/chassis/json.gen.h"
#include "client/cpp/ni/device/json.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::ni::device {
/// @brief it should round trip a map of command and state channel pairs through JSON.
TEST(OutputPorts, RoundTripsChannels) {
    const OutputPorts ports{
        .state_index = 1,
        .channels = {{"ao0", CommandStatePair{.command = 2, .state = 3}}},
    };
    const auto j = ports.to_json();
    ASSERT_EQ(j["channels"]["ao0"]["command"], 2);
    ASSERT_EQ(j["channels"]["ao0"]["state"], 3);
    x::json::Parser parser(j);
    const auto parsed = OutputPorts::parse(parser);
    ASSERT_TRUE(parser.ok());
    ASSERT_EQ(parsed.state_index, 1);
    ASSERT_EQ(parsed.channels.at("ao0").command, 2);
    ASSERT_EQ(parsed.channels.at("ao0").state, 3);
}

/// @brief it should apply the schema defaults to an output port group with no fields.
TEST(OutputPorts, ParsesEmptyObject) {
    x::json::Parser parser(x::json::json::object());
    const auto parsed = OutputPorts::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.state_index, 0);
    ASSERT_TRUE(parsed.channels.empty());
}

/// @brief it should default the port groups of a configuration that omits them.
TEST(Configuration, ParsesMissingPortGroups) {
    x::json::Parser parser(x::json::json{{"name", "Module"}});
    const auto parsed = Configuration::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.name, "Module");
    ASSERT_TRUE(parsed.analog_input.channels.empty());
    ASSERT_TRUE(parsed.digital_output.channels.empty());
}

/// @brief it should keep the field group of an update through JSON.
TEST(Update, RoundTripsGroup) {
    IdentityUpdate identity;
    identity.rack = 1;
    identity.serial = "01A";
    const auto j = to_json(Update(identity));
    ASSERT_EQ(j["group"], "identity");
    x::json::Parser parser(j);
    const auto parsed = parse_update(parser);
    ASSERT_TRUE(parser.ok());
    ASSERT_EQ(std::get<IdentityUpdate>(parsed).serial, "01A");
}

/// @brief it should write an empty object for an output port group with no channels.
TEST(OutputPorts, SerializesEmptyChannelsAsObject) {
    ASSERT_TRUE(OutputPorts{}.to_json()["channels"].is_object());
}
}
