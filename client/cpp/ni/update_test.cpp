// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <variant>

#include "gtest/gtest.h"

#include "client/cpp/ni/json.gen.h"

namespace synnax::ni {
/// @brief it should parse an update without a type as base fields only.
TEST(AIChannelUpdate, testParseWithoutTypeAsBase) {
    const auto update = parse_ai_channel_update(
        x::json::Parser(x::json::json{{"key", "c3"}, {"name", "Pressure"}})
    );
    ASSERT_TRUE(std::holds_alternative<AIChannelBaseUpdate>(update));
    const auto &base = std::get<AIChannelBaseUpdate>(update);
    EXPECT_EQ(base.key, "c3");
    EXPECT_EQ(base.name, "Pressure");
}

/// @brief it should parse an update with a type as the named variant.
TEST(AIChannelUpdate, testParseWithTypeAsVariant) {
    const auto update = parse_ai_channel_update(
        x::json::Parser(
            x::json::json{{"key", "c3"}, {"type", "ai_voltage"}, {"max_val", 10}}
        )
    );
    ASSERT_TRUE(std::holds_alternative<AIVoltageChannelUpdate>(update));
    EXPECT_EQ(std::get<AIVoltageChannelUpdate>(update).max_val, 10);
}

/// @brief it should write an update without a type as base fields only.
TEST(AIChannelUpdate, testToJSONWithoutType) {
    const AIChannelUpdate update = AIChannelBaseUpdate{.key = "c3", .name = "Pressure"};
    EXPECT_EQ(to_json(update), (x::json::json{{"key", "c3"}, {"name", "Pressure"}}));
}

/// @brief it should parse and write the channel updates of a config by their variant.
TEST(AnalogReadConfigUpdate, testChannelUpdatesByVariant) {
    const auto j = x::json::json::parse(R"({
        "channels": {
            "update": [
                {"key": "c3", "name": "Pressure"},
                {"key": "c4", "type": "ai_voltage", "max_val": 10}
            ]
        }
    })");
    const auto update = AnalogReadConfigUpdate::parse(x::json::Parser(j));
    ASSERT_TRUE(update.channels.has_value());
    ASSERT_TRUE(update.channels->update.has_value());
    const auto &channels = *update.channels->update;
    ASSERT_EQ(channels.size(), 2);
    EXPECT_TRUE(std::holds_alternative<AIChannelBaseUpdate>(channels[0]));
    EXPECT_TRUE(std::holds_alternative<AIVoltageChannelUpdate>(channels[1]));
    EXPECT_EQ(update.to_json(), j);
}
}
