// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xjson/xjson.h"

#include "driver/task/common/common.h"

/// @brief it should return PersistStream when data_saving is true.
TEST(DataSavingWriterMode, testDataSavingTrue) {
    const auto mode = common::data_saving_writer_mode(true);
    EXPECT_EQ(mode, synnax::WriterMode::PersistStream);
}

/// @brief it should return StreamOnly when data_saving is false.
TEST(DataSavingWriterMode, testDataSavingFalse) {
    const auto mode = common::data_saving_writer_mode(false);
    EXPECT_EQ(mode, synnax::WriterMode::StreamOnly);
}

/// @brief it should parse BaseTaskConfig with both fields present.
TEST(BaseTaskConfig, testParseWithBothFields) {
    const auto json = nlohmann::json{{"data_saving", false}, {"auto_start", true}};
    auto parser = xjson::Parser(json);
    const auto config = common::BaseTaskConfig(parser);

    EXPECT_FALSE(config.data_saving);
    EXPECT_TRUE(config.auto_start);
}

/// @brief it should use default values when fields are missing.
TEST(BaseTaskConfig, testParseWithDefaults) {
    const auto json = nlohmann::json{};
    auto parser = xjson::Parser(json);
    const auto config = common::BaseTaskConfig(parser);

    EXPECT_TRUE(config.data_saving); // defaults to true
    EXPECT_FALSE(config.auto_start); // defaults to false
}

/// @brief it should parse with only data_saving present.
TEST(BaseTaskConfig, testParseWithDataSavingOnly) {
    const auto json = nlohmann::json{{"data_saving", false}};
    auto parser = xjson::Parser(json);
    const auto config = common::BaseTaskConfig(parser);

    EXPECT_FALSE(config.data_saving);
    EXPECT_FALSE(config.auto_start); // defaults to false
}

/// @brief it should parse with only auto_start present.
TEST(BaseTaskConfig, testParseWithAutoStartOnly) {
    const auto json = nlohmann::json{{"auto_start", true}};
    auto parser = xjson::Parser(json);
    const auto config = common::BaseTaskConfig(parser);

    EXPECT_TRUE(config.data_saving); // defaults to true
    EXPECT_TRUE(config.auto_start);
}

/// @brief it should support move construction.
TEST(BaseTaskConfig, testMoveConstruction) {
    const auto json = nlohmann::json{{"data_saving", false}, {"auto_start", true}};
    auto parser = xjson::Parser(json);
    auto config1 = common::BaseTaskConfig(parser);

    auto config2 = common::BaseTaskConfig(std::move(config1));

    EXPECT_FALSE(config2.data_saving);
    EXPECT_TRUE(config2.auto_start);
}
