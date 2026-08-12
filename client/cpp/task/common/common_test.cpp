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

namespace synnax::task::common {
/// @brief it should parse the generated BaseStartConfig with both fields present.
TEST(BaseStartConfig, testParseWithBothFields) {
    const auto json = nlohmann::json{
        {"data_saving_disabled", true},
        {"auto_start", true}
    };
    auto parser = x::json::Parser(json);
    const auto config = BaseStartConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}

/// @brief it should use default values when fields are missing.
TEST(BaseStartConfig, testParseWithDefaults) {
    const auto json = nlohmann::json{};
    auto parser = x::json::Parser(json);
    const auto config = BaseStartConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse with only data_saving_disabled present.
TEST(BaseStartConfig, testParseWithDataSavingDisabledOnly) {
    const auto json = nlohmann::json{{"data_saving_disabled", true}};
    auto parser = x::json::Parser(json);
    const auto config = BaseStartConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse with only auto_start present.
TEST(BaseStartConfig, testParseWithAutoStartOnly) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = BaseStartConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}
}
