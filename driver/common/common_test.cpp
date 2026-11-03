// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/common/common.h"

namespace driver::common {
/// @brief it should return StreamOnly when data saving is disabled.
TEST(DataSavingWriterMode, testDataSavingDisabled) {
    const auto mode = data_saving_writer_mode(true);
    EXPECT_EQ(mode, synnax::framer::WriterMode::StreamOnly);
}

/// @brief it should return PersistStream when data saving is enabled.
TEST(DataSavingWriterMode, testDataSavingEnabled) {
    const auto mode = data_saving_writer_mode(false);
    EXPECT_EQ(mode, synnax::framer::WriterMode::PersistStream);
}

/// @brief it should retry without a limit, at most 5 s apart.
TEST(UnlimitedRetryConfig, testRetriesWithoutLimit) {
    const auto cfg = unlimited_retry_config("task");
    EXPECT_EQ(cfg.name, "task");
    EXPECT_EQ(cfg.max_retries, x::breaker::RETRY_INFINITELY);
    EXPECT_EQ(cfg.max_interval, 5 * x::telem::SECOND);
}
}
