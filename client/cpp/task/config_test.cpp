// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <include/gtest/gtest.h>

#include "client/cpp/task/json.gen.h"
#include "x/cpp/json/json.h"

namespace synnax::task {
/// @brief it should parse the generated StartConfig with auto_start present.
TEST(StartConfig, testParseWithAutoStart) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = StartConfig::parse(parser);

    EXPECT_TRUE(config.auto_start);
}

/// @brief it should use default values when fields are missing.
TEST(StartConfig, testParseWithDefaults) {
    const auto json = nlohmann::json{};
    auto parser = x::json::Parser(json);
    const auto config = StartConfig::parse(parser);

    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse the generated PersistConfig with both fields present.
TEST(PersistConfig, testParseWithBothFields) {
    const auto json = nlohmann::json{
        {"data_saving_disabled", true},
        {"auto_start", true}
    };
    auto parser = x::json::Parser(json);
    const auto config = PersistConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}

/// @brief it should use default values when fields are missing.
TEST(PersistConfig, testParseWithDefaults) {
    const auto json = nlohmann::json{};
    auto parser = x::json::Parser(json);
    const auto config = PersistConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse with only data_saving_disabled present.
TEST(PersistConfig, testParseWithDataSavingDisabledOnly) {
    const auto json = nlohmann::json{{"data_saving_disabled", true}};
    auto parser = x::json::Parser(json);
    const auto config = PersistConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should mint a record key for a config that carries none, so a config
/// written by the driver still parses.
TEST(StartConfig, testParseWithoutKey) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = StartConfig::parse(parser);

    EXPECT_FALSE(parser.error()) << parser.error().message();
    EXPECT_FALSE(config.key.is_nil());
}

/// @brief it should parse with only auto_start present.
TEST(PersistConfig, testParseWithAutoStartOnly) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = PersistConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}
}
