// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/task/common/status.h"

/// @brief it should correctly communicate the starting state of a task.
TEST(TestTaskStateHandler, testStartCommunication) {
    const auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    const synnax::Task task("task1", "ni_analog_read", "");
    auto handler = driver::task::common::StatusHandler(ctx, task);

    handler.send_start("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.key, task.status_key());
    EXPECT_EQ(first.details.cmd, "cmd_key");
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, status::variant::SUCCESS);
    EXPECT_EQ(first.details.running, true);
    EXPECT_EQ(first.message, "Task started successfully");

    handler.error(x::errors::Error(x::errors::VALIDATION, "task validation error"));
    handler.send_start("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.key, task.status_key());
    EXPECT_EQ(second.details.cmd, "cmd_key");
    EXPECT_EQ(second.name, "task1");
    EXPECT_EQ(second.details.task, task.key);
    EXPECT_EQ(second.variant, status::variant::ERR);
    EXPECT_EQ(second.details.running, false);
    EXPECT_EQ(second.message, "task validation error");
}

/// @brief it should correctly communicate a warning to the context.
TEST(TestTaskStateHandler, testSendWarning) {
    const auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    const synnax::Task task("task1", "ni_analog_read", "");
    auto handler = driver::task::common::StatusHandler(ctx, task);

    handler.send_warning("Test warning message");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, status::variant::WARNING);
    EXPECT_EQ(first.message, "Test warning message");

    handler.error(x::errors::Error(x::errors::VALIDATION, "task validation error"));
    handler.send_warning("This warning should not be sent");
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.details.task, task.key);
    EXPECT_EQ(second.variant, status::variant::ERR);
    EXPECT_EQ(second.message, "task validation error");
}

/// @brief it should correctly move the task back to a nominal running state.
TEST(TestTaskStateHandle, testClearWarning) {
    const auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    const synnax::Task task("task1", "ni_analog_read", "");
    auto handler = driver::task::common::StatusHandler(ctx, task);

    // First send a warning
    handler.send_warning("Test warning message");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, status::variant::WARNING);
    EXPECT_EQ(first.message, "Test warning message");

    // Now clear the warning
    handler.clear_warning();
    ASSERT_GE(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.details.task, task.key);
    EXPECT_EQ(second.variant, status::variant::SUCCESS);
    EXPECT_EQ(second.message, "Task running");

    // Test that clear_warning doesn't do anything if not in warning state
    handler.error(x::errors::Error(x::errors::VALIDATION, "task validation error"));
    handler.send_warning("This is an error");
    ASSERT_GE(ctx->statuses.size(), 3);
    const auto third = ctx->statuses[2];
    EXPECT_EQ(third.variant, status::variant::ERR);

    // Clear warning should have no effect when in error state
    const size_t stateCount = ctx->statuses.size();
    handler.clear_warning();
    EXPECT_EQ(ctx->statuses.size(), stateCount); // No new state should be added
}

/// @brief it should correctly communicate the stopping state of a task.
TEST(TestTaskStateHandler, testStopCommunication) {
    const auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    const synnax::Task task("task1", "ni_analog_read", "");
    auto handler = driver::task::common::StatusHandler(ctx, task);

    handler.send_stop("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.key, task.status_key());
    EXPECT_EQ(first.details.cmd, "cmd_key");
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, status::variant::SUCCESS);
    EXPECT_EQ(first.details.running, false);
    EXPECT_EQ(first.message, "Task stopped successfully");

    handler.error(x::errors::Error(x::errors::VALIDATION, "task validation error"));
    handler.send_stop("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.key, task.status_key());
    EXPECT_EQ(second.details.cmd, "cmd_key");
    EXPECT_EQ(second.details.task, task.key);
    EXPECT_EQ(second.variant, status::variant::ERR);
    EXPECT_EQ(second.details.running, false);
    EXPECT_EQ(second.message, "task validation error");
}
