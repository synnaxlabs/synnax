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

/// @brief it should write an empty object for an output port group with no channels.
TEST(OutputPorts, SerializesEmptyChannelsAsObject) {
    ASSERT_TRUE(OutputPorts{}.to_json()["channels"].is_object());
}
}
