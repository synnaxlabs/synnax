// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/common/status.h"

namespace driver::common {
/// @brief it should correctly communicate the starting state of a task.
TEST(TestTaskStateHandler, testStartCommunication) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
    };
    auto handler = StatusHandler(ctx, task);

    handler.send_start("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.key, task.status_key());
    EXPECT_EQ(first.details.cmd, "cmd_key");
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, x::status::VARIANT_SUCCESS);
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
    EXPECT_EQ(second.variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(second.details.running, false);
    EXPECT_EQ(second.message, "task validation error");
}

/// @brief it should correctly communicate a warning to the context.
TEST(TestTaskStateHandler, testSendWarning) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
    };
    auto handler = StatusHandler(ctx, task);

    handler.send_warning("Test warning message");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(first.message, "Test warning message");

    handler.error(x::errors::Error(x::errors::VALIDATION, "task validation error"));
    handler.send_warning("This warning should not be sent");
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.details.task, task.key);
    EXPECT_EQ(second.variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(second.message, "task validation error");
}

/// @brief it should correctly move the task back to a nominal running state.
TEST(TestTaskStateHandle, testClearWarning) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
    };
    auto handler = StatusHandler(ctx, task);

    // First send a warning
    handler.send_warning("Test warning message");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, x::status::VARIANT_WARNING);
    EXPECT_EQ(first.message, "Test warning message");

    // Now clear the warning
    handler.clear_warning();
    ASSERT_GE(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.details.task, task.key);
    EXPECT_EQ(second.variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(second.message, "Task running");

    // Test that clear_warning doesn't do anything if not in warning state
    handler.error(x::errors::Error(x::errors::VALIDATION, "task validation error"));
    handler.send_warning("This is an error");
    ASSERT_GE(ctx->statuses.size(), 3);
    const auto third = ctx->statuses[2];
    EXPECT_EQ(third.variant, x::status::VARIANT_ERROR);

    // Clear warning should have no effect when in error state
    const size_t stateCount = ctx->statuses.size();
    handler.clear_warning();
    EXPECT_EQ(ctx->statuses.size(), stateCount); // No new state should be added
}

/// @brief it should immediately send an error status to the context.
TEST(TestTaskStateHandler, testSendError) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
    };
    auto handler = StatusHandler(ctx, task);

    handler.send_error(x::errors::Error(x::errors::VALIDATION, "fatal runtime error"));
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.key, task.status_key());
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(first.details.running, false);
    EXPECT_EQ(first.message, "fatal runtime error");

    // Verify nil errors are ignored
    const size_t count = ctx->statuses.size();
    handler.send_error(x::errors::NIL);
    EXPECT_EQ(ctx->statuses.size(), count);
}

/// @brief it should correctly communicate the stopping state of a task.
TEST(TestTaskStateHandler, testStopCommunication) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
    };
    auto handler = StatusHandler(ctx, task);

    handler.send_stop("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.key, task.status_key());
    EXPECT_EQ(first.details.cmd, "cmd_key");
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(first.details.running, false);
    EXPECT_EQ(first.message, "Task stopped successfully");

    handler.error(x::errors::Error(x::errors::VALIDATION, "task validation error"));
    handler.send_stop("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.key, task.status_key());
    EXPECT_EQ(second.details.cmd, "cmd_key");
    EXPECT_EQ(second.details.task, task.key);
    EXPECT_EQ(second.variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(second.details.running, false);
    EXPECT_EQ(second.message, "task validation error");
}

/// @brief identical repeated warnings should be suppressed within the rate
/// limit window.
TEST(TestStatusRateLimit, testSuppressesIdenticalWarnings) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    handler.send_warning("skew too high");
    handler.send_warning("skew too high");
    handler.send_warning("skew too high");
    EXPECT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].message, "skew too high");
    EXPECT_EQ(ctx->statuses[0].variant, x::status::VARIANT_WARNING);
}

/// @brief a different warning message should go through immediately even if
/// another warning was recently sent.
TEST(TestStatusRateLimit, testAllowsDifferentWarningMessages) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    handler.send_warning("skew too high");
    handler.send_warning("array mismatch");
    EXPECT_EQ(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[0].message, "skew too high");
    EXPECT_EQ(ctx->statuses[1].message, "array mismatch");
}

/// @brief alternating A/B/A/B spam should only produce two statuses (one per
/// distinct message) within the rate limit window.
TEST(TestStatusRateLimit, testSuppressesAlternatingSpam) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    for (int i = 0; i < 100; ++i) {
        handler.send_warning(i % 2 == 0 ? "warning A" : "warning B");
    }
    EXPECT_EQ(ctx->statuses.size(), 2);
}

/// @brief identical repeated errors should be suppressed within the rate limit
/// window.
TEST(TestStatusRateLimit, testSuppressesIdenticalErrors) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    const auto err = x::errors::Error(x::errors::VALIDATION, "device disconnected");
    handler.send_error(err);
    handler.send_error(err);
    handler.send_error(err);
    EXPECT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, x::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses[0].message, "device disconnected");
}

/// @brief start messages should always go through (bypasses rate limiter)
/// because the Console waits for command acknowledgments.
TEST(TestStatusRateLimit, testStartBypassesRateLimit) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    handler.send_start("cmd1");
    handler.send_start("cmd2");
    EXPECT_EQ(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[0].details.cmd, "cmd1");
    EXPECT_EQ(ctx->statuses[1].details.cmd, "cmd2");
}

/// @brief stop messages should always go through (bypasses rate limiter)
/// because the Console waits for command acknowledgments.
TEST(TestStatusRateLimit, testStopBypassesRateLimit) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    handler.send_stop("cmd1");
    handler.send_stop("cmd2");
    EXPECT_EQ(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[0].details.cmd, "cmd1");
    EXPECT_EQ(ctx->statuses[1].details.cmd, "cmd2");
}

/// @brief clear_warning after a warning should go through (different variant),
/// but repeated clears should be suppressed.
TEST(TestStatusRateLimit, testClearWarningThenSuppressRepeatedClears) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    handler.send_warning("skew too high");
    EXPECT_EQ(ctx->statuses.size(), 1);

    handler.clear_warning();
    EXPECT_EQ(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[1].variant, x::status::VARIANT_SUCCESS);

    // Sending the same warning again should be suppressed (it was sent <5s ago).
    handler.send_warning("skew too high");
    EXPECT_EQ(ctx->statuses.size(), 2);
}

/// @brief reset() should clear the rate-limit state, allowing previously
/// suppressed messages to go through again.
TEST(TestStatusRateLimit, testResetClearsRateLimitState) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    handler.send_warning("skew too high");
    EXPECT_EQ(ctx->statuses.size(), 1);

    handler.send_warning("skew too high");
    EXPECT_EQ(ctx->statuses.size(), 1);

    handler.reset();
    handler.send_warning("skew too high");
    EXPECT_EQ(ctx->statuses.size(), 2);
}

/// @brief when the dedup map reaches MAX_RECENT_STATUSES, the single oldest
/// entry should be evicted to make room for the new one.
TEST(TestStatusRateLimit, testEvictsOldestOnOverflow) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    // Fill the map to capacity with unique warnings (0 is the oldest).
    for (size_t i = 0; i < StatusHandler::MAX_RECENT_STATUSES; ++i)
        handler.send_warning("warning " + std::to_string(i));
    EXPECT_EQ(ctx->statuses.size(), StatusHandler::MAX_RECENT_STATUSES);

    // One overflow: evicts "warning 0" (oldest), inserts the new message.
    handler.send_warning("overflow warning");
    EXPECT_EQ(ctx->statuses.size(), StatusHandler::MAX_RECENT_STATUSES + 1);

    // "warning 0" was the evicted entry — it should go through again.
    handler.send_warning("warning 0");
    EXPECT_EQ(ctx->statuses.size(), StatusHandler::MAX_RECENT_STATUSES + 2);

    // "warning 49" is the newest original entry and was never evicted — still
    // suppressed.
    handler.send_warning("warning 49");
    EXPECT_EQ(ctx->statuses.size(), StatusHandler::MAX_RECENT_STATUSES + 2);
}

/// @brief N sequential overflows should evict exactly the N oldest entries and
/// leave the rest of the map intact.
TEST(TestStatusRateLimit, testEvictsNOldestOnNOverflows) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    // Fill to capacity.
    for (size_t i = 0; i < StatusHandler::MAX_RECENT_STATUSES; ++i)
        handler.send_warning("warning " + std::to_string(i));

    // 3 overflows evict the 3 oldest entries: "warning 0", "warning 1", "warning 2".
    handler.send_warning("overflow 0");
    handler.send_warning("overflow 1");
    handler.send_warning("overflow 2");

    // The 3 evicted entries should now go through.
    handler.send_warning("warning 0");
    handler.send_warning("warning 1");
    handler.send_warning("warning 2");
    EXPECT_EQ(ctx->statuses.size(), StatusHandler::MAX_RECENT_STATUSES + 6);

    // "warning 49" is the newest original entry and was never evicted — still
    // suppressed.
    handler.send_warning("warning 49");
    EXPECT_EQ(ctx->statuses.size(), StatusHandler::MAX_RECENT_STATUSES + 6);
}

/// @brief the same message should be re-sent once the rate-limit window expires.
/// This guards against the bug where contains(key) suppresses entries forever
/// rather than for only STATUS_RATE_LIMIT.
TEST(TestStatusRateLimit, testWindowExpires) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    // Use a 1 ms window so the test completes quickly.
    auto handler = StatusHandler(ctx, task, 1 * x::telem::MILLISECOND);

    handler.send_warning("array mismatch");
    EXPECT_EQ(ctx->statuses.size(), 1);

    // Still within the window — should be suppressed.
    handler.send_warning("array mismatch");
    EXPECT_EQ(ctx->statuses.size(), 1);

    // Wait for the window to expire, then re-send.
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    handler.send_warning("array mismatch");
    EXPECT_EQ(ctx->statuses.size(), 2);
}

/// @brief a warning followed by an error with the same message text should go
/// through because the variant differs.
TEST(TestStatusRateLimit, testDifferentVariantSameMessage) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    handler.send_warning("device issue");
    EXPECT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, x::status::VARIANT_WARNING);

    handler.send_error(x::errors::Error(x::errors::VALIDATION, "device issue"));
    EXPECT_EQ(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[1].variant, x::status::VARIANT_ERROR);
}
}
