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
/// @brief it should parse the generated BaseConfig with both fields present.
TEST(BaseConfig, testParseWithBothFields) {
    const auto json = nlohmann::json{
        {"data_saving_disabled", true},
        {"auto_start", true}
    };
    auto parser = x::json::Parser(json);
    const auto config = BaseConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}

/// @brief it should use default values when fields are missing.
TEST(BaseConfig, testParseWithDefaults) {
    const auto json = nlohmann::json{};
    auto parser = x::json::Parser(json);
    const auto config = BaseConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse with only data_saving_disabled present.
TEST(BaseConfig, testParseWithDataSavingDisabledOnly) {
    const auto json = nlohmann::json{{"data_saving_disabled", true}};
    auto parser = x::json::Parser(json);
    const auto config = BaseConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse a config that carries no record key. The Core assigns
/// the key, so a config written without one must still configure.
TEST(BaseConfig, testParseWithoutKey) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = BaseConfig::parse(parser);

    EXPECT_FALSE(parser.error()) << parser.error().message();
    EXPECT_TRUE(config.key.is_nil());
}

/// @brief it should parse with only auto_start present.
TEST(BaseConfig, testParseWithAutoStartOnly) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = BaseConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}
}
