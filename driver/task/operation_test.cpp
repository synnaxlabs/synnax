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
        synnax::task::Key{},
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

/// @brief it should send the current step to a joined start on the next check, and
/// the final status to both starts.
TEST_F(OperationTest, JoinedStartGetsTheStepAndTheFinal) {
    op.step(loading("Starting task"), x::telem::MINUTE);
    ASSERT_TRUE(op.join({.type = "start", .key = "cmd-2"}, "hash"));
    op.check(x::telem::TimeStamp::now());
    ASSERT_EQ(written.size(), 2);
    EXPECT_EQ(written[1].details.cmd, "cmd-2");
    EXPECT_EQ(written[1].message, "Starting task");
    op.answer(final_status("started"));
    ASSERT_EQ(written.size(), 4);
    EXPECT_EQ(written[2].details.cmd, "cmd");
    EXPECT_EQ(written[3].details.cmd, "cmd-2");
    EXPECT_TRUE(op.close().empty());
}

/// @brief it should refuse a start of another config, and any start after a cancel.
TEST_F(OperationTest, JoinRefusesOtherConfigAndCancelledStart) {
    op.step(loading("Starting task"), x::telem::MINUTE);
    EXPECT_FALSE(op.join({.type = "start", .key = "cmd-2"}, "other"));
    ASSERT_TRUE(op.cancel("The Driver stopped"));
    EXPECT_FALSE(op.join({.type = "start", .key = "cmd-3"}, "hash"));
}

/// @brief it should take no join before the first step, and take a join of the same
/// config after it, even for a pinned start.
TEST(OperationJoinTest, JoinWaitsForTheFirstStep) {
    std::vector<synnax::task::Status> written;
    Operation op{
        [&](const synnax::task::Status &s) { written.push_back(s); },
        synnax::task::Key{},
        "task",
        synnax::task::Command{.type = "start", .key = "cmd", .pinned = true},
        "hash",
        x::telem::MINUTE
    };
    EXPECT_FALSE(op.join({.type = "start", .key = "cmd-2"}, "hash"));
    op.step(loading("Starting task"), x::telem::MINUTE);
    EXPECT_TRUE(op.join({.type = "start", .key = "cmd-3", .pinned = true}, "hash"));
}

/// @brief it should return the joined starts that got no final status at close.
TEST_F(OperationTest, CloseReturnsUnansweredJoins) {
    op.step(loading("Starting task"), x::telem::MINUTE);
    ASSERT_TRUE(op.join({.type = "start", .key = "cmd-2"}, "hash"));
    const auto unanswered = op.close();
    ASSERT_EQ(unanswered.size(), 1);
    EXPECT_EQ(unanswered[0].key, "cmd-2");
}

/// @brief it should answer a start that a stop cancelled, and each start that joined
/// it, at close, and refuse a later final.
TEST_F(OperationTest, CloseAnswersAStoppedStartOnce) {
    op.step(loading("Starting task"), x::telem::MINUTE);
    ASSERT_TRUE(op.join({.type = "start", .key = "cmd-2"}, "hash"));
    ASSERT_TRUE(op.cancel({.type = "stop", .key = "stop"}));
    EXPECT_TRUE(op.close().empty());
    ASSERT_EQ(written.size(), 3);
    EXPECT_EQ(written[1].details.cmd, "cmd");
    EXPECT_EQ(written[2].details.cmd, "cmd-2");
    for (size_t i = 1; i < written.size(); i++) {
        EXPECT_EQ(written[i].variant, synnax::status::VARIANT_SUCCESS);
        EXPECT_EQ(written[i].message, "Stopped task");
        EXPECT_EQ(written[i].description, "A stop ended the start before it finished.");
        EXPECT_FALSE(written[i].details.running);
    }
    op.answer(final_status("started late"));
    ASSERT_EQ(written.size(), 4);
    EXPECT_EQ(written[3].details.cmd, "");
}

/// @brief it should answer a stopped start as stopped even when the task manager
/// cancels it again as it stops.
TEST_F(OperationTest, CloseAnswersAStartStoppedBeforeAShutdown) {
    ASSERT_TRUE(op.cancel({.type = "stop", .key = "stop"}));
    op.cancel("The Driver stopped");
    EXPECT_TRUE(op.close().empty());
    ASSERT_EQ(written.size(), 1);
    EXPECT_EQ(written[0].message, "Stopped task");
}

/// @brief it should fail a start that the task manager cancelled as it stopped.
TEST_F(OperationTest, CloseFailsAStartCancelledAtShutdown) {
    ASSERT_TRUE(op.cancel("The Driver stopped"));
    EXPECT_TRUE(op.close().empty());
    ASSERT_EQ(written.size(), 1);
    EXPECT_EQ(written[0].details.cmd, "cmd");
    EXPECT_EQ(written[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(written[0].message, "Failed to start task");
    EXPECT_EQ(
        written[0].description,
        "The Driver stopped while the task was starting. Start the task again."
    );
}

/// @brief it should send nothing at close when the task answered the cancelled start.
TEST_F(OperationTest, CloseKeepsTheAnswerOfACancelledStart) {
    ASSERT_TRUE(op.cancel("The Driver stopped"));
    op.answer(final_status("start cancelled"));
    EXPECT_TRUE(op.close().empty());
    ASSERT_EQ(written.size(), 1);
    EXPECT_EQ(written[0].message, "start cancelled");
}

/// @brief it should say that a stop ended a start only when close answered the start.
TEST_F(OperationTest, EndedByStopOnlyWhenCloseAnswers) {
    ASSERT_TRUE(op.cancel({.type = "stop", .key = "stop"}));
    EXPECT_FALSE(op.ended_by_stop());
    EXPECT_TRUE(op.close().empty());
    EXPECT_TRUE(op.ended_by_stop());
}

/// @brief it should not say that a stop ended a start that sent its own final status.
TEST_F(OperationTest, NotEndedByStopAfterTheTaskAnswers) {
    ASSERT_TRUE(op.cancel({.type = "stop", .key = "stop"}));
    op.answer(final_status("Failed to configure task"));
    EXPECT_TRUE(op.close().empty());
    EXPECT_FALSE(op.ended_by_stop());
}

/// @brief it should stop the token on cancel and tell the stop that it arrived.
TEST_F(OperationTest, CancelStopsTheTokenAndAnswersTheStop) {
    op.step(loading("Starting task"), x::telem::MINUTE);
    ASSERT_TRUE(op.cancel({.type = "stop", .key = "stop"}));
    EXPECT_FALSE(op.token().running());
    op.check(x::telem::TimeStamp::now());
    ASSERT_EQ(written.size(), 2);
    EXPECT_EQ(written[1].details.cmd, "stop");
    EXPECT_EQ(written[1].variant, synnax::status::VARIANT_LOADING);
    EXPECT_EQ(written[1].message, "Stopping task");
}

/// @brief it should name the cause of a cancel by the task manager in the failed
/// start.
TEST_F(OperationTest, CloseNamesTheCauseOfACancel) {
    ASSERT_TRUE(op.cancel("The Driver lost its connection to the Core"));
    EXPECT_TRUE(op.close().empty());
    ASSERT_EQ(written.size(), 1);
    EXPECT_EQ(
        written[0].description,
        "The Driver lost its connection to the Core while the task was starting. "
        "Start the task again."
    );
}

/// @brief it should say that a stop ended an operation with no command, such as a boot
/// deploy, only when a stop cancelled it.
TEST(OperationBootTest, EndedByStopForAnOperationWithNoCommand) {
    std::vector<synnax::task::Status> written;
    const auto new_op = [&] {
        return std::make_unique<Operation>(
            [&](const synnax::task::Status &s) { written.push_back(s); },
            synnax::task::Key{},
            "task",
            synnax::task::Command{},
            "hash",
            x::telem::MINUTE
        );
    };
    const auto stopped = new_op();
    ASSERT_TRUE(stopped->cancel(synnax::task::Command{.type = "stop", .key = "stop"}));
    EXPECT_TRUE(stopped->close().empty());
    EXPECT_TRUE(stopped->ended_by_stop());
    const auto shut_down = new_op();
    ASSERT_TRUE(shut_down->cancel("The Driver stopped"));
    EXPECT_TRUE(shut_down->close().empty());
    EXPECT_FALSE(shut_down->ended_by_stop());
    EXPECT_TRUE(written.empty());
}

/// @brief it should not cancel an operation that sent its final status.
TEST_F(OperationTest, NoCancelAfterTheFinal) {
    op.answer(final_status("started"));
    EXPECT_FALSE(op.cancel("The Driver stopped"));
    EXPECT_TRUE(op.token().running());
}
}
