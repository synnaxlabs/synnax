// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"

#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"

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

/// @brief it should show a license warning at once.
TEST(CoreWarning, ShowsLicenseAtOnce) {
    const auto w = core_warning(
        synnax::errors::LICENSE_MISSING,
        x::telem::TimeSpan::ZERO(),
        x::telem::SECOND
    );
    ASSERT_EQ(w.message, "Waiting for a license");
}

/// @brief it should show a Core warning only once the wait reaches the delay.
TEST(CoreWarning, ShowsOtherErrorsAfterTheDelay) {
    ASSERT_TRUE(
        core_warning(freighter::TRANSPORT, x::telem::MILLISECOND, x::telem::SECOND)
            .empty()
    );
    const auto w = core_warning(
        freighter::TRANSPORT,
        x::telem::SECOND,
        x::telem::SECOND
    );
    ASSERT_EQ(w.message, "Failed to reach the Core");
    ASSERT_NE(w.description.find(freighter::TRANSPORT.type), std::string::npos);
}

/// @brief it should show no warning before the delay, and one once the wait reaches it.
TEST(CoreWait, WarnsOnceTheWaitReachesTheDelay) {
    CoreWait wait(x::telem::SECOND);
    const auto t0 = x::telem::TimeStamp(x::telem::SECOND);
    ASSERT_TRUE(wait.failed(freighter::TRANSPORT, t0).empty());
    ASSERT_EQ(
        wait.failed(freighter::TRANSPORT, t0 + x::telem::SECOND).message,
        "Failed to reach the Core"
    );
}

/// @brief it should keep waiting through an open that fails again before the delay.
TEST(CoreWait, KeepsWaitingThroughAShortOpen) {
    CoreWait wait(x::telem::SECOND);
    const auto t0 = x::telem::TimeStamp(x::telem::SECOND);
    ASSERT_TRUE(wait.failed(freighter::TRANSPORT, t0).empty());
    wait.opened(t0 + x::telem::MILLISECOND * 500);
    ASSERT_FALSE(wait.recovered(t0 + x::telem::MILLISECOND * 600));
    ASSERT_FALSE(
        wait.failed(freighter::TRANSPORT, t0 + x::telem::MILLISECOND * 1100).empty()
    );
}

/// @brief it should end the wait once an open lasts the delay.
TEST(CoreWait, EndsTheWaitAfterALongOpen) {
    CoreWait wait(x::telem::SECOND);
    const auto t0 = x::telem::TimeStamp(x::telem::SECOND);
    ASSERT_TRUE(wait.failed(freighter::TRANSPORT, t0).empty());
    wait.opened(t0 + x::telem::SECOND);
    ASSERT_TRUE(wait.recovered(t0 + x::telem::SECOND * 2));
    ASSERT_TRUE(wait.failed(freighter::TRANSPORT, t0 + x::telem::SECOND * 3).empty());
}

/// @brief it should forget the wait on reset, as a restart does.
TEST(CoreWait, ForgetsTheWaitOnReset) {
    CoreWait wait(x::telem::SECOND);
    const auto t0 = x::telem::TimeStamp(x::telem::SECOND);
    ASSERT_TRUE(wait.failed(freighter::TRANSPORT, t0).empty());
    wait.reset();
    ASSERT_TRUE(wait.failed(freighter::TRANSPORT, t0 + x::telem::SECOND * 2).empty());
}

/// @brief it should describe both warnings with the error text as one sentence.
TEST(CoreWarning, DescribesTheErrorTextAsASentence) {
    const auto license = core_warning(
        x::errors::Error(synnax::errors::LICENSE_MISSING, "no license."),
        x::telem::TimeSpan::ZERO(),
        x::telem::SECOND
    );
    ASSERT_EQ(license.description.find('['), std::string::npos);
    ASSERT_EQ(license.description.find(".."), std::string::npos);
    ASSERT_NE(license.description.find("no license."), std::string::npos);
    const auto core = core_warning(
        x::errors::Error(freighter::TRANSPORT, "connection reset"),
        x::telem::SECOND,
        x::telem::SECOND
    );
    ASSERT_EQ(core.description.find('['), std::string::npos);
    ASSERT_EQ(core.description.rfind("connection reset. ", 0), 0);
}

/// @brief it should end a license wait as soon as a request opens.
TEST(CoreWait, EndsALicenseWaitOnOpen) {
    CoreWait wait(x::telem::SECOND * 30);
    const auto t0 = x::telem::TimeStamp(x::telem::SECOND);
    ASSERT_FALSE(wait.failed(synnax::errors::LICENSE_MISSING, t0).empty());
    const auto opened = t0 + x::telem::MILLISECOND;
    wait.opened(opened);
    ASSERT_TRUE(wait.recovered(opened));
}
}
