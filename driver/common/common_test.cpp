// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/task/common/json.gen.h"
#include "x/cpp/json/json.h"

#include "driver/common/common.h"

namespace driver::common {
/// @brief it should return PersistStream when data_saving is true.
TEST(DataSavingWriterMode, testDataSavingTrue) {
    const auto mode = data_saving_writer_mode(true);
    EXPECT_EQ(mode, synnax::framer::WriterMode::PersistStream);
}

/// @brief it should return StreamOnly when data_saving is false.
TEST(DataSavingWriterMode, testDataSavingFalse) {
    const auto mode = data_saving_writer_mode(false);
    EXPECT_EQ(mode, synnax::framer::WriterMode::StreamOnly);
}

/// @brief it should parse the generated BaseConfig with both fields present.
TEST(BaseConfig, testParseWithBothFields) {
    const auto json = nlohmann::json{
        {"data_saving_disabled", true},
        {"auto_start", true}
    };
    auto parser = x::json::Parser(json);
    const auto config = ::synnax::common::BaseConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}

/// @brief it should use default values when fields are missing.
TEST(BaseConfig, testParseWithDefaults) {
    const auto json = nlohmann::json{};
    auto parser = x::json::Parser(json);
    const auto config = ::synnax::common::BaseConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse with only data_saving_disabled present.
TEST(BaseConfig, testParseWithDataSavingDisabledOnly) {
    const auto json = nlohmann::json{{"data_saving_disabled", true}};
    auto parser = x::json::Parser(json);
    const auto config = ::synnax::common::BaseConfig::parse(parser);

    EXPECT_TRUE(config.data_saving_disabled);
    EXPECT_FALSE(config.auto_start);
}

/// @brief it should parse with only auto_start present.
TEST(BaseConfig, testParseWithAutoStartOnly) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = x::json::Parser(json);
    const auto config = ::synnax::common::BaseConfig::parse(parser);

    EXPECT_FALSE(config.data_saving_disabled);
    EXPECT_TRUE(config.auto_start);
}

/// @brief it should map a legacy data_saving=false onto data_saving_disabled.
TEST(LegacyDataSavingDisabled, testLegacyFalseDisables) {
    const auto json = nlohmann::json{{"data_saving", false}};
    auto parser = x::json::Parser(json);
    EXPECT_TRUE(legacy_data_saving_disabled(parser, false));
}

/// @brief it should keep saving enabled for a legacy data_saving=true.
TEST(LegacyDataSavingDisabled, testLegacyTrueStaysEnabled) {
    const auto json = nlohmann::json{{"data_saving", true}};
    auto parser = x::json::Parser(json);
    EXPECT_FALSE(legacy_data_saving_disabled(parser, false));
}

/// @brief it should keep saving enabled when neither key is present.
TEST(LegacyDataSavingDisabled, testAbsentKeysStayEnabled) {
    const auto json = nlohmann::json{};
    auto parser = x::json::Parser(json);
    EXPECT_FALSE(legacy_data_saving_disabled(parser, false));
}

/// @brief it should preserve an explicit data_saving_disabled=true.
TEST(LegacyDataSavingDisabled, testExplicitDisabledWins) {
    const auto json = nlohmann::json{{"data_saving", true}};
    auto parser = x::json::Parser(json);
    EXPECT_TRUE(legacy_data_saving_disabled(parser, true));
}
}
