// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// ReSharper disable CppUseStructuredBinding

/// external
#include "gtest/gtest.h"

/// module
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/task/common/state.h"

/// @brief it should correctly communicate the starting state of a task.
TEST(TestTaskStateHandler, testStartCommunication) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::Task task("rack1", "task1", "ni_analog_read", "");
    auto handler = common::StateHandler(ctx, task);

    handler.send_start("cmd_key");
    ASSERT_GE(ctx->states.size(), 1);
    const auto first = ctx->states[0];
    EXPECT_EQ(first.key, "cmd_key");
    EXPECT_EQ(first.task, task.key);
    EXPECT_EQ(first.variant, "success");
    EXPECT_EQ(first.details["running"], true);
    EXPECT_EQ(first.details["message"], "Task started successfully");

    handler.error(xerrors::Error(xerrors::VALIDATION, "task validation error"));
    handler.send_start("cmd_key");
    ASSERT_GE(ctx->states.size(), 2);
    const auto second = ctx->states[1];
    EXPECT_EQ(second.key, "cmd_key");
    EXPECT_EQ(second.task, task.key);
    EXPECT_EQ(second.variant, "error");
    EXPECT_EQ(second.details["running"], false);
    EXPECT_EQ(second.details["message"], "task validation error");
}

/// @brief it should correctly communicate a warning to the context.
TEST(TestTaskStateHandler, testSendWarning) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::Task task("rack1", "task1", "ni_analog_read", "");
    auto handler = common::StateHandler(ctx, task);

    handler.send_warning("Test warning message");
    ASSERT_GE(ctx->states.size(), 1);
    const auto first = ctx->states[0];
    EXPECT_EQ(first.task, task.key);
    EXPECT_EQ(first.variant, "warning");
    EXPECT_EQ(first.details["message"], "Test warning message");

    handler.error(xerrors::Error(xerrors::VALIDATION, "task validation error"));
    handler.send_warning("This warning should not be sent");
    ASSERT_EQ(ctx->states.size(), 2);
    const auto second = ctx->states[1];
    EXPECT_EQ(second.task, task.key);
    EXPECT_EQ(second.variant, "error");
    EXPECT_EQ(second.details["message"], "task validation error");
}

/// @brief it should correctly move the task back to a nominal running state.
TEST(TestTaskStateHandle, testClearWarning) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::Task task("rack1", "task1", "ni_analog_read", "");
    auto handler = common::StateHandler(ctx, task);

    // First send a warning
    handler.send_warning("Test warning message");
    ASSERT_GE(ctx->states.size(), 1);
    const auto first = ctx->states[0];
    EXPECT_EQ(first.task, task.key);
    EXPECT_EQ(first.variant, "warning");
    EXPECT_EQ(first.details["message"], "Test warning message");

    // Now clear the warning
    handler.clear_warning();
    ASSERT_GE(ctx->states.size(), 2);
    const auto second = ctx->states[1];
    EXPECT_EQ(second.task, task.key);
    EXPECT_EQ(second.variant, "success");
    EXPECT_EQ(second.details["message"], "Task started successfully");

    // Test that clear_warning doesn't do anything if not in warning state
    handler.error(xerrors::Error(xerrors::VALIDATION, "task validation error"));
    handler.send_warning("This is an error");
    ASSERT_GE(ctx->states.size(), 3);
    const auto third = ctx->states[2];
    EXPECT_EQ(third.variant, "error");

    // Clear warning should have no effect when in error state
    const size_t stateCount = ctx->states.size();
    handler.clear_warning();
    EXPECT_EQ(ctx->states.size(), stateCount); // No new state should be added
}

/// @brief it should correctly communicate the stopping state of a task.
TEST(TestTaskStateHandler, testStopCommunication) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::Task task("rack1", "task1", "ni_analog_read", "");
    auto handler = common::StateHandler(ctx, task);

    handler.send_stop("cmd_key");
    ASSERT_GE(ctx->states.size(), 1);
    const auto first = ctx->states[0];
    EXPECT_EQ(first.key, "cmd_key");
    EXPECT_EQ(first.task, task.key);
    EXPECT_EQ(first.variant, "success");
    EXPECT_EQ(first.details["running"], false);
    EXPECT_EQ(first.details["message"], "Task stopped successfully");

    handler.error(xerrors::Error(xerrors::VALIDATION, "task validation error"));
    handler.send_stop("cmd_key");
    ASSERT_GE(ctx->states.size(), 2);
    const auto second = ctx->states[1];
    EXPECT_EQ(second.key, "cmd_key");
    EXPECT_EQ(second.task, task.key);
    EXPECT_EQ(second.variant, "error");
    EXPECT_EQ(second.details["running"], false);
    EXPECT_EQ(second.details["message"], "task validation error");
}
