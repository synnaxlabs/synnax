// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "driver/task/operation.h"

namespace driver::task {
namespace {
synnax::task::Status loading(const std::string &message) {
    return {
        .variant = synnax::status::VARIANT_LOADING,
        .message = message,
    };
}

synnax::task::Status final_status(const std::string &message) {
    return {
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = message,
    };
}

class OperationTest : public testing::Test {
protected:
    std::vector<synnax::task::Status> written;
    Operation op{
        [this](const synnax::task::Status &s) { this->written.push_back(s); },
        "task",
        synnax::task::Command{.type = "start", .key = "cmd"},
        "hash",
        x::telem::MINUTE
    };
};
}

/// @brief it should send the first final status with the command key, and a later one
/// with no command key.
TEST_F(OperationTest, SendsOneFinalStatus) {
    op.answer(final_status("started"));
    op.answer(final_status("started again"));
    ASSERT_EQ(written.size(), 2);
    EXPECT_EQ(written[0].details.cmd, "cmd");
    EXPECT_EQ(written[1].details.cmd, "");
    EXPECT_EQ(written[1].message, "started again");
}

/// @brief it should drop a step sent after the final status.
TEST_F(OperationTest, DropsStepAfterFinal) {
    op.answer(final_status("started"));
    op.step(loading("late step"), x::telem::SECOND);
    ASSERT_EQ(written.size(), 1);
    EXPECT_EQ(written[0].message, "started");
}

/// @brief it should warn once, with a loading status, when a step runs past its limit.
TEST_F(OperationTest, WarnsOnceWhenAStepRunsLate) {
    op.step(loading("Starting bus"), x::telem::SECOND);
    const auto now = x::telem::TimeStamp::now();
    op.check(now);
    ASSERT_EQ(written.size(), 1);
    op.check(now + 2 * x::telem::SECOND);
    op.check(now + 3 * x::telem::SECOND);
    ASSERT_EQ(written.size(), 2);
    EXPECT_EQ(written[1].variant, synnax::status::VARIANT_LOADING);
    EXPECT_EQ(written[1].message, "Starting bus");
    EXPECT_EQ(written[1].details.cmd, "cmd");
    EXPECT_EQ(
        written[1].description,
        "Still working after " + x::telem::SECOND.to_string() + "."
    );
}

/// @brief it should give each new step its own warning.
TEST_F(OperationTest, NewStepRestartsTheLimit) {
    op.step(loading("Starting bus"), x::telem::SECOND);
    op.check(x::telem::TimeStamp::now() + 2 * x::telem::SECOND);
    op.answer(loading("Reading modules"));
    const auto now = x::telem::TimeStamp::now();
    op.check(now);
    ASSERT_EQ(written.size(), 3);
    op.check(now + 2 * x::telem::SECOND);
    ASSERT_EQ(written.size(), 4);
    EXPECT_EQ(written[3].message, "Reading modules");
}

/// @brief it should hold a loading status with no limit of its own to the limit of
/// the operation.
TEST_F(OperationTest, LoadingStatusUsesTheOperationLimit) {
    op.answer(loading("Stopping task"));
    const auto now = x::telem::TimeStamp::now();
    op.check(now + 30 * x::telem::SECOND);
    ASSERT_EQ(written.size(), 1);
    op.check(now + 2 * x::telem::MINUTE);
    ASSERT_EQ(written.size(), 2);
    EXPECT_EQ(
        written[1].description,
        "Still working after " + x::telem::MINUTE.to_string() + "."
    );
}

/// @brief it should send no warning status without a step, after the final status,
/// or after close.
TEST_F(OperationTest, NoWarningStatusWithoutAStep) {
    const auto later = x::telem::TimeStamp::now() + x::telem::HOUR;
    op.check(later);
    EXPECT_TRUE(written.empty());
    op.step(loading("Starting bus"), x::telem::SECOND);
    op.answer(final_status("started"));
    op.check(later);
    EXPECT_EQ(written.size(), 2);
}

/// @brief it should send no warning after close, so a status that the task sends
/// after its operation ends stays the latest one.
TEST_F(OperationTest, NoWarningAfterClose) {
    op.step(loading("Starting bus"), x::telem::SECOND);
    op.close();
    op.check(x::telem::TimeStamp::now() + x::telem::HOUR);
    EXPECT_EQ(written.size(), 1);
}
}
