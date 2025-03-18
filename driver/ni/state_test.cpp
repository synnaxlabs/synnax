// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// module
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/ni/ni.h"
#include "driver/task/task.h"

// TEST(TestTaskStateHandler, testStartCommunication) {
//     const auto ctx = std::make_shared<task::MockContext>(nullptr);
//     const synnax::Task task("rack1", "task1", "ni_analog_read", "");
//     auto handler = ni::TaskStateHandler(ctx, task);
//
//     handler.send_start("cmd_key");
//     ASSERT_GE(ctx->states.size(), 1);
//     const auto first = ctx->states[0];
//     EXPECT_EQ(first.key, "cmd_key");
//     EXPECT_EQ(first.task, task.key);
//     EXPECT_EQ(first.variant, "success");
//     EXPECT_EQ(first.details["running"], true);
//     EXPECT_EQ(first.details["message"], "Task started successfully");
//
//     handler.error(xerrors::VALIDATION);
//     handler.send_start("cmd_key");
//     ASSERT_GE(ctx->states.size(), 2);
//     const auto second = ctx->states[1];
//     EXPECT_EQ(second.key, "cmd_key");
//     EXPECT_EQ(second.task, task.key);
//     EXPECT_EQ(second.variant, "error");
//     EXPECT_EQ(second.details["running"], false);
//     EXPECT_EQ(second.details["message"], "[sy.validation] ");
// }
//
// TEST(TestTaskStateHandler, testSendWarning) {
//     const auto ctx = std::make_shared<task::MockContext>(nullptr);
//     const synnax::Task task("rack1", "task1", "ni_analog_read", "");
//     auto handler = ni::TaskStateHandler(ctx, task);
//
//     handler.send_warning("Test warning message");
//     ASSERT_GE(ctx->states.size(), 1);
//     const auto first = ctx->states[0];
//     EXPECT_EQ(first.task, task.key);
//     EXPECT_EQ(first.variant, "warning");
//     EXPECT_EQ(first.details["message"], "Test warning message");
//
//     handler.error(xerrors::VALIDATION);
//     handler.send_warning("This warning should not be sent");
//     ASSERT_EQ(ctx->states.size(), 2);
//     const auto second = ctx->states[1];
//     EXPECT_EQ(second.task, task.key);
//     EXPECT_EQ(second.variant, "error");
//     EXPECT_EQ(second.details["message"], "[sy.validation] ");
// }
//
// TEST(TestTaskStateHandler, testStopCommunication) {
//     const auto ctx = std::make_shared<task::MockContext>(nullptr);
//     const synnax::Task task("rack1", "task1", "ni_analog_read", "");
//     auto handler = ni::TaskStateHandler(ctx, task);
//
//     handler.send_stop("cmd_key");
//     ASSERT_GE(ctx->states.size(), 1);
//     const auto first = ctx->states[0];
//     EXPECT_EQ(first.key, "cmd_key");
//     EXPECT_EQ(first.task, task.key);
//     EXPECT_EQ(first.variant, "success");
//     EXPECT_EQ(first.details["running"], false);
//     EXPECT_EQ(first.details["message"], "Task stopped successfully");
//
//     handler.error(xerrors::VALIDATION);
//     handler.send_stop("cmd_key");
//     ASSERT_GE(ctx->states.size(), 2);
//     const auto second = ctx->states[1];
//     EXPECT_EQ(second.key, "cmd_key");
//     EXPECT_EQ(second.task, task.key);
//     EXPECT_EQ(second.variant, "error");
//     EXPECT_EQ(second.details["running"], false);
//     EXPECT_EQ(second.details["message"], "[sy.validation] ");
// }
