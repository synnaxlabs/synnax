// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <include/gtest/gtest.h>

#include "client/cpp/task/common/json.gen.h"
#include "x/cpp/json/json.h"

namespace synnax::common {
/// @brief it should parse the generated BaseConfig with auto_start present.
TEST(BaseConfig, testParseWithAutoStart) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = BaseConfig::parse(parser);

    EXPECT_TRUE(config.auto_start);
}

/// @brief it should use default values when fields are missing.
TEST(BaseConfig, testParseWithDefaults) {
    const auto json = nlohmann::json{};
    auto parser = x::json::Parser(json);
    const auto config = BaseConfig::parse(parser);

    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse the generated BasePersistConfig with both fields present.
TEST(BasePersistConfig, testParseWithBothFields) {
    const auto json = nlohmann::json{
        {"data_saving_disabled", true},
        {"auto_start", true}
    };
    auto parser = x::json::Parser(json);
    const auto config = BasePersistConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}

/// @brief it should use default values when fields are missing.
TEST(BasePersistConfig, testParseWithDefaults) {
    const auto json = nlohmann::json{};
    auto parser = x::json::Parser(json);
    const auto config = BasePersistConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse with only data_saving_disabled present.
TEST(BasePersistConfig, testParseWithDataSavingDisabledOnly) {
    const auto json = nlohmann::json{{"data_saving_disabled", true}};
    auto parser = x::json::Parser(json);
    const auto config = BasePersistConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should mint a record key for a config that carries none, so a config
/// written by the driver still parses.
TEST(BaseConfig, testParseWithoutKey) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = BaseConfig::parse(parser);

    EXPECT_FALSE(parser.error()) << parser.error().message();
    EXPECT_FALSE(config.key.is_nil());
}

/// @brief it should parse with only auto_start present.
TEST(BasePersistConfig, testParseWithAutoStartOnly) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = BasePersistConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}
}
