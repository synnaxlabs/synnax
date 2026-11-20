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

#include "x/cpp/testutil/testutil.h"

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
    EXPECT_TRUE(first.key.empty());
    EXPECT_EQ(first.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(first.details.cmd, "cmd_key");
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(first.details.running, true);
    EXPECT_EQ(first.message, "Started task");

    handler.error(
        x::errors::Error(x::errors::VALIDATION, "task validation error"),
        "Failed to start task"
    );
    handler.send_start("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(second.details.cmd, "cmd_key");
    EXPECT_EQ(second.name, "task1");
    EXPECT_EQ(second.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(second.details.running, false);
    EXPECT_EQ(second.message, "Failed to start task");
    EXPECT_EQ(second.description, "task validation error");
}

/// @brief it should send nothing for a start that its cancel ended, and answer the stop
/// that follows as stopped.
TEST(TestTaskStateHandler, testCancelledStartSendsNothing) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);
    handler.error(
        x::errors::Error(x::errors::CANCELLED, "bus start"),
        "Failed to start task"
    );
    handler.send_start("start_cmd");
    EXPECT_TRUE(ctx->statuses.empty());
    handler.send_stop("stop_cmd");
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(ctx->statuses[0].message, "Stopped task");
    EXPECT_EQ(ctx->statuses[0].details.cmd, "stop_cmd");
}

/// @brief it should echo the config hash the instance was built from.
TEST(TestTaskStateHandler, testEchoesConfigHash) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
        .config_hash = "hash1",
    };
    auto handler = StatusHandler(ctx, task);

    handler.send_start("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].details.config_hash, "hash1");

    handler.send_stop("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[1].details.config_hash, "hash1");
}

/// @brief it should correctly communicate a warning to the context.
TEST(TestTaskStateHandler, testSendWarning) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
    };
    auto handler = StatusHandler(ctx, task, x::telem::TimeSpan(0));

    handler.send_warning("Test warning message", "");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(first.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(first.message, "Test warning message");

    handler.error(
        x::errors::Error(x::errors::VALIDATION, "task validation error"),
        "Failed to start task"
    );
    handler.send_warning("This warning should not be sent", "");
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(second.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(second.message, "Failed to start task");
    EXPECT_EQ(second.description, "task validation error");
}

/// @brief it should correctly move the task back to a nominal running state.
TEST(TestTaskStateHandle, testClearWarning) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
    };
    auto handler = StatusHandler(ctx, task, x::telem::TimeSpan(0));

    // First send a warning
    handler.send_warning("Test warning message", "");
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(first.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(first.message, "Test warning message");

    // Now clear the warning
    handler.clear_warning();
    ASSERT_GE(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(second.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(second.message, "Task running");

    // Test that clear_warning doesn't do anything if not in warning state
    handler.error(
        x::errors::Error(x::errors::VALIDATION, "task validation error"),
        "Failed to start task"
    );
    handler.send_warning("This is an error", "");
    ASSERT_GE(ctx->statuses.size(), 3);
    const auto third = ctx->statuses[2];
    EXPECT_EQ(third.variant, synnax::status::VARIANT_ERROR);

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

    handler.send_error(
        x::errors::Error(x::errors::VALIDATION, "fatal runtime error"),
        "Failed to run Arc program"
    );
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(first.details.running, false);
    EXPECT_EQ(first.message, "Failed to run Arc program");
    EXPECT_EQ(first.description, "fatal runtime error");

    // Verify nil errors are ignored
    const size_t count = ctx->statuses.size();
    handler.send_error(x::errors::NIL, "Failed to run Arc program");
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
    EXPECT_EQ(first.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(first.details.cmd, "cmd_key");
    EXPECT_EQ(first.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(first.details.running, false);
    EXPECT_EQ(first.message, "Stopped task");

    handler.error(
        x::errors::Error(x::errors::VALIDATION, "task validation error"),
        "Failed to stop task"
    );
    handler.send_stop("cmd_key");
    ASSERT_GE(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(second.details.cmd, "cmd_key");
    EXPECT_EQ(second.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(second.details.running, false);
    EXPECT_EQ(second.message, "Failed to stop task");
    EXPECT_EQ(second.description, "task validation error");
}

/// @brief ack should re-send the current status unchanged, stamping the command.
TEST(TestTaskStateHandler, testAckReSendsCurrentStatus) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
    };
    auto handler = StatusHandler(ctx, task);

    handler.send_start("cmd_key");
    handler.send_warning("Task degraded", "");
    handler.ack("cmd_key_2", true);
    ASSERT_GE(ctx->statuses.size(), 3);
    const auto third = ctx->statuses[2];
    EXPECT_EQ(third.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(third.details.cmd, "cmd_key_2");
    EXPECT_EQ(third.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(third.message, "Task degraded");
    EXPECT_EQ(third.details.running, true);
}

/// @brief ack before any send should answer with the seeded configured status.
TEST(TestTaskStateHandler, testAckBeforeAnySend) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
    };
    auto handler = StatusHandler(ctx, task);

    handler.ack("cmd_key", false);
    ASSERT_GE(ctx->statuses.size(), 1);
    const auto first = ctx->statuses[0];
    EXPECT_EQ(first.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(first.details.cmd, "cmd_key");
    EXPECT_EQ(first.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(first.message, "Task configured");
    EXPECT_EQ(first.details.running, false);
}

/// @brief it should send repeated identical warnings inside the window only once.
TEST(TestStatusWindow, testSuppressesIdenticalWarnings) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 50 * x::telem::MILLISECOND;
    auto handler = StatusHandler(ctx, task, window);

    handler.send_warning("skew too high", "");
    handler.send_warning("skew too high", "");
    handler.send_warning("skew too high", "");
    std::this_thread::sleep_for((3 * window).chrono());
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 1);
    EXPECT_EQ(statuses[0].message, "skew too high");
    EXPECT_EQ(statuses[0].variant, synnax::status::VARIANT_WARNING);
}

/// @brief it should send an unchanged warning once, however many windows pass.
TEST(TestStatusWindow, testSuppressesIdenticalWarningsAcrossWindows) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "http_read"};
    const auto window = 10 * x::telem::MILLISECOND;
    auto handler = StatusHandler(ctx, task, window);

    for (int i = 0; i < 4; i++) {
        handler.send_warning("Failed to read some data", "Field /a not found");
        std::this_thread::sleep_for((2 * window).chrono());
    }
    ASSERT_EQ(ctx->snapshot().size(), 1);
}

/// @brief it should send a warning again after it clears.
TEST(TestStatusWindow, testSendsWarningAgainAfterClear) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "http_read"};
    const auto window = 10 * x::telem::MILLISECOND;
    auto handler = StatusHandler(ctx, task, window);

    handler.send_warning("Failed to read some data", "Field /a not found");
    std::this_thread::sleep_for((2 * window).chrono());
    handler.clear_warning();
    std::this_thread::sleep_for((2 * window).chrono());
    handler.send_warning("Failed to read some data", "Field /a not found");
    std::this_thread::sleep_for((2 * window).chrono());
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 3);
    EXPECT_EQ(statuses[0].variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(statuses[1].variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(statuses[2].variant, synnax::status::VARIANT_WARNING);
}

/// @brief it should send a change inside the window once the window ends.
TEST(TestStatusWindow, testDelaysChangeUntilWindowEnds) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 100 * x::telem::MILLISECOND;
    auto handler = StatusHandler(ctx, task, window);

    handler.send_warning("skew too high", "");
    handler.send_warning("array mismatch", "");
    EXPECT_EQ(ctx->snapshot().size(), 1);
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    const auto statuses = ctx->snapshot();
    EXPECT_EQ(statuses[0].message, "skew too high");
    EXPECT_EQ(statuses[1].message, "array mismatch");
    EXPECT_GE(statuses[1].time - statuses[0].time, 90 * x::telem::MILLISECOND);
}

/// @brief it should send the recovery from a warning without another call, as a task
/// that recovers and then reads normally makes no further status call.
TEST(TestStatusWindow, testSendsRecoveryWithoutAnotherCall) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task, 100 * x::telem::MILLISECOND);

    handler.send_warning("Failed to read data", "The device is unreachable.");
    handler.clear_warning();
    handler.clear_warning();
    EXPECT_EQ(ctx->snapshot().size(), 1);
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    const auto recovered = ctx->snapshot()[1];
    EXPECT_EQ(recovered.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(recovered.message, "Task running");
    EXPECT_EQ(recovered.description, "");
}

/// @brief it should send only the latest of many changes inside the window.
TEST(TestStatusWindow, testSendsLatestOfManyChanges) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 100 * x::telem::MILLISECOND;
    auto handler = StatusHandler(ctx, task, window);

    for (int i = 0; i < 100; ++i)
        handler.send_warning(i % 2 == 0 ? "warning A" : "warning B", "");
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    std::this_thread::sleep_for((3 * window).chrono());
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 2);
    EXPECT_EQ(statuses[0].message, "warning A");
    EXPECT_EQ(statuses[1].message, "warning B");
}

/// @brief it should send nothing at the end of the window when the status changed and
/// changed back to the one already sent.
TEST(TestStatusWindow, testSkipsChangeThatReturnsToSentStatus) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 50 * x::telem::MILLISECOND;
    auto handler = StatusHandler(ctx, task, window);

    handler.send_warning("warning A", "");
    handler.send_warning("warning B", "");
    handler.send_warning("warning A", "");
    std::this_thread::sleep_for((3 * window).chrono());
    ASSERT_EQ(ctx->snapshot().size(), 1);
}

/// @brief it should send two warnings with the same message and different causes.
TEST(TestStatusWindow, testSendsDifferentWarningDescriptions) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task, 50 * x::telem::MILLISECOND);

    handler.send_warning("Failed to read data", "The device is unreachable.");
    handler.send_warning("Failed to read data", "The channel no longer exists.");
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    const auto statuses = ctx->snapshot();
    EXPECT_EQ(statuses[0].description, "The device is unreachable.");
    EXPECT_EQ(statuses[1].description, "The channel no longer exists.");
}

namespace {
const Warning BACKGROUND{.message = "Falling behind the Core", .description = "Lag."};

StatusHandler &start(StatusHandler &handler) {
    handler.send_start("cmd");
    return handler;
}
}

/// @brief it should show a background warning while the task runs.
TEST(TestBackgroundWarning, ShowsWhileRunning) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    StatusHandler handler(ctx, task, 10 * x::telem::MILLISECOND);
    start(handler).set_background_warning(BACKGROUND);
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    const auto shown = ctx->snapshot()[1];
    EXPECT_EQ(shown.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(shown.message, BACKGROUND.message);
    EXPECT_EQ(shown.description, BACKGROUND.description);
    EXPECT_TRUE(shown.details.running);
    EXPECT_EQ(shown.details.cmd, driver::task::NO_COMMAND);
}

/// @brief it should not show a background warning while the task is stopped.
TEST(TestBackgroundWarning, HiddenWhileStopped) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 10 * x::telem::MILLISECOND;
    StatusHandler handler(ctx, task, window);
    handler.send_stop("cmd");
    handler.set_background_warning(BACKGROUND);
    std::this_thread::sleep_for((3 * window).chrono());
    ASSERT_EQ(ctx->snapshot().size(), 1);
}

/// @brief it should show a warning of the task in place of the background warning,
/// and show the background warning again once the task clears its own.
TEST(TestBackgroundWarning, TaskWarningOverridesIt) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    StatusHandler handler(ctx, task, 10 * x::telem::MILLISECOND);
    start(handler).set_background_warning(BACKGROUND);
    handler.send_warning("Falling behind hardware", "Skew.");
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().back().message, "Falling behind hardware");
    handler.clear_warning();
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().back().message, BACKGROUND.message);
    EXPECT_EQ(ctx->snapshot().back().variant, synnax::status::VARIANT_WARNING);
}

/// @brief it should stop showing the background warning once it is cleared.
TEST(TestBackgroundWarning, Clears) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    StatusHandler handler(ctx, task, 10 * x::telem::MILLISECOND);
    start(handler).set_background_warning(BACKGROUND);
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    handler.set_background_warning({});
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 3);
    const auto cleared = ctx->snapshot()[2];
    EXPECT_EQ(cleared.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(cleared.description, "");
    EXPECT_TRUE(cleared.details.running);
}

/// @brief it should hide a background warning behind an error of the task, and not
/// show it again while the error stands.
TEST(TestBackgroundWarning, ErrorHidesIt) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 10 * x::telem::MILLISECOND;
    StatusHandler handler(ctx, task, window);
    start(handler).set_background_warning(BACKGROUND);
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    handler.send_error(x::errors::VALIDATION, "Failed to read data");
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 3);
    EXPECT_EQ(ctx->snapshot()[2].variant, synnax::status::VARIANT_ERROR);
    handler.set_background_warning({.message = "Dropped frames"});
    handler.clear_warning();
    std::this_thread::sleep_for((3 * window).chrono());
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 3);
    EXPECT_EQ(statuses[2].message, "Failed to read data");
}

/// @brief it should send nothing for a background warning after the handler closes.
TEST(TestBackgroundWarning, SendsNothingAfterClose) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 10 * x::telem::MILLISECOND;
    StatusHandler handler(ctx, task, window);
    start(handler).close();
    handler.set_background_warning(BACKGROUND);
    std::this_thread::sleep_for((3 * window).chrono());
    ASSERT_EQ(ctx->snapshot().size(), 1);
}

/// @brief it should send a repeated identical background warning only once.
TEST(TestBackgroundWarning, SendsRepeatsOnce) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 10 * x::telem::MILLISECOND;
    StatusHandler handler(ctx, task, window);
    start(handler).set_background_warning(BACKGROUND);
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    for (int i = 0; i < 2; i++) {
        handler.set_background_warning(BACKGROUND);
        handler.clear_warning();
    }
    std::this_thread::sleep_for((3 * window).chrono());
    ASSERT_EQ(ctx->snapshot().size(), 2);
}

/// @brief clearing a warning should drop the description that explained it.
TEST(TestTaskStateHandler, ClearWarningDropsDescription) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task, x::telem::TimeSpan(0));

    handler.send_warning("Failed to read data", "The device is unreachable.");
    handler.clear_warning();
    ASSERT_EQ(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[1].message, "Task running");
    EXPECT_EQ(ctx->statuses[1].description, "");
}

/// @brief it should send repeated identical errors inside the window only once.
TEST(TestStatusWindow, testSuppressesIdenticalErrors) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    const auto err = x::errors::Error(x::errors::VALIDATION, "device disconnected");
    handler.send_error(err, "Failed to run Arc program");
    handler.send_error(err, "Failed to run Arc program");
    handler.send_error(err, "Failed to run Arc program");
    EXPECT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses[0].message, "Failed to run Arc program");
    EXPECT_EQ(ctx->statuses[0].description, "device disconnected");
}

/// @brief it should send each start reply at once, as the Console waits for command
/// acknowledgments.
TEST(TestStatusWindow, testStartSkipsWindow) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    handler.send_start("cmd1");
    handler.send_start("cmd2");
    EXPECT_EQ(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[0].details.cmd, "cmd1");
    EXPECT_EQ(ctx->statuses[1].details.cmd, "cmd2");
}

/// @brief it should send each stop reply at once, as the Console waits for command
/// acknowledgments.
TEST(TestStatusWindow, testStopSkipsWindow) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task);

    handler.send_stop("cmd1");
    handler.send_stop("cmd2");
    EXPECT_EQ(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[0].details.cmd, "cmd1");
    EXPECT_EQ(ctx->statuses[1].details.cmd, "cmd2");
}

/// @brief it should send a reply at once inside the window and drop the pending status,
/// as the reply carries the latest state.
TEST(TestStatusWindow, testReplySkipsWindowAndReplacesPending) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 100 * x::telem::MILLISECOND;
    auto handler = StatusHandler(ctx, task, window);

    handler.send_warning("skew too high", "");
    handler.send_warning("array mismatch", "");
    handler.send_stop("stop_cmd");
    auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 2);
    EXPECT_EQ(statuses[1].details.cmd, "stop_cmd");
    EXPECT_EQ(statuses[1].message, "Stopped task");
    std::this_thread::sleep_for((3 * window).chrono());
    EXPECT_EQ(ctx->snapshot().size(), 2);
}

/// @brief it should drop a pending status when the handler is destroyed, as a task that
/// stops without a stop status must write nothing more.
TEST(TestStatusWindow, testDestroyDropsPending) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 50 * x::telem::MILLISECOND;
    {
        auto handler = StatusHandler(ctx, task, window);
        handler.send_warning("skew too high", "");
        handler.send_warning("array mismatch", "");
    }
    std::this_thread::sleep_for((3 * window).chrono());
    EXPECT_EQ(ctx->snapshot().size(), 1);
}

/// @brief it should drop a pending status and later changes once closed, and still send
/// replies.
TEST(TestStatusWindow, testCloseDropsPending) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    const auto window = 50 * x::telem::MILLISECOND;
    auto handler = StatusHandler(ctx, task, window);
    handler.send_warning("skew too high", "");
    handler.send_warning("array mismatch", "");
    handler.close();
    std::this_thread::sleep_for((3 * window).chrono());
    handler.send_warning("device unreachable", "");
    EXPECT_EQ(ctx->snapshot().size(), 1);
    handler.ack("start_cmd", false);
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 2);
    EXPECT_EQ(statuses[1].details.cmd, "start_cmd");
}

/// @brief it should keep a full window between two statuses when another thread starts
/// a window while the flusher waits.
TEST(TestStatusWindow, testKeepsWindowWhenAnotherThreadSends) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    // A wide window absorbs a loaded host pausing a writer between starting the window
    // and stamping the status.
    const auto window = 100 * x::telem::MILLISECOND;
    auto handler = StatusHandler(ctx, task, window);
    // A tight loop of changes often takes the lock at the end of a window before the
    // flusher does, then queues the next change.
    const auto end = std::chrono::steady_clock::now() + std::chrono::seconds(1);
    for (int i = 0; std::chrono::steady_clock::now() < end; i++) {
        handler.send_warning(i % 2 == 0 ? "skew too high" : "array mismatch", "");
        std::this_thread::yield();
    }
    const auto statuses = ctx->snapshot();
    ASSERT_GE(statuses.size(), 2);
    for (size_t i = 1; i < statuses.size(); i++)
        EXPECT_GE(statuses[i].time - statuses[i - 1].time, window / 2) << i;
}

/// @brief it should send a warning and then an error with the same message, as the
/// variant differs.
TEST(TestStatusWindow, testDifferentVariantSameMessage) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task, 50 * x::telem::MILLISECOND);

    handler.send_warning("device issue", "");
    handler.send_error(
        x::errors::Error(x::errors::VALIDATION, "device issue"),
        "Failed to run Arc program"
    );
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    const auto statuses = ctx->snapshot();
    EXPECT_EQ(statuses[0].variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(statuses[1].variant, synnax::status::VARIANT_ERROR);
}

class RecordingTask final : public task::Task {
public:
    std::vector<synnax::task::Command> cmds;
    /// @brief the context whose status count each exec records, or null.
    std::shared_ptr<task::MockContext> ctx;
    std::vector<size_t> statuses_at_exec;
    void exec(synnax::task::Command &cmd, x::breaker::Breaker &) override {
        this->cmds.push_back(cmd);
        if (this->ctx != nullptr)
            this->statuses_at_exec.push_back(this->ctx->statuses.size());
    }
    void stop(bool will_reconfigure) override {}
};

/// @brief a config error during a start-triggered deploy should write an error
/// status.
TEST(TestHandleConfigErr, testErrorWithPendingStart) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
        .config_hash = "hash1"
    };
    cancel.start();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {ConfigureResult{}, x::errors::Error(x::errors::VALIDATION, "bad config")},
        "start_cmd",
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses[0].message, "Failed to configure task");
    EXPECT_EQ(ctx->statuses[0].description, "bad config");
    EXPECT_EQ(ctx->statuses[0].details.cmd, "start_cmd");
    EXPECT_EQ(ctx->statuses[0].details.config_hash, "hash1");
}

/// @brief it should write no status for a configure that its cancel ended, so the
/// start is answered as stopped.
TEST(TestHandleConfigErr, testCancelledConfigureWritesNothing) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {ConfigureResult{}, x::errors::Error(x::errors::CANCELLED, "bus start")},
        "start_cmd",
        cancel
    );
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
}

/// @brief a config error should list each field error on its own line.
TEST(TestHandleConfigErr, ListsFieldErrors) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    x::json::Parser parser(x::json::json::object());
    parser.field_err("device", "this field is required");
    parser.field_err("channels.0.port", "must be positive");
    cancel.start();
    handle_config_err(
        ctx,
        task,
        {ConfigureResult{}, parser.error()},
        "start_cmd",
        cancel
    );
    cancel.stop();
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].message, "Failed to configure task");
    EXPECT_EQ(
        ctx->statuses[0].description,
        "device: this field is required\nchannels.0.port: must be positive"
    );
}

/// @brief a config error at boot with no auto-start should write an error status, so
/// the status of the last run does not stay.
TEST(TestHandleConfigErr, testErrorAtBootWritesError) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task task{.rack = 1, .name = "task1", .type = "ni_analog_read"};
    // A stopped status of another rack, such as one an old Driver left.
    task.status = synnax::task::Status{};
    task.status->details.rack = 2;
    cancel.start();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {ConfigureResult{}, x::errors::Error(x::errors::VALIDATION, "bad config")},
        "",
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses[0].message, "Failed to configure task");
    EXPECT_FALSE(ctx->statuses[0].details.running);
}

/// @brief a config error at boot should write no status for a task moved here, since
/// its old rack runs it until a start arrives.
TEST(TestHandleConfigErr, testErrorAtBootOfAMovedTaskSilent) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task task{.rack = 1, .name = "task1", .type = "ni_analog_read"};
    task.status = synnax::task::Status{};
    task.status->details.rack = 2;
    task.status->details.running = true;
    cancel.start();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {ConfigureResult{}, x::errors::Error(x::errors::VALIDATION, "bad config")},
        "",
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
}

/// @brief a config error at boot with auto-start requested should write an error
/// status.
TEST(TestHandleConfigErr, testErrorAtBootWithAutoStart) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    ConfigureResult res;
    res.auto_start = true;
    cancel.start();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {std::move(res), x::errors::Error(x::errors::VALIDATION, "bad config")},
        "",
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_TRUE(ctx->statuses[0].details.cmd.empty());
}

/// @brief a successful configure during a deploy should write no status. The start
/// that follows it is what answers the command.
TEST(TestHandleConfigErr, testSuccessWithPendingStart) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    ConfigureResult res;
    res.task = std::make_unique<RecordingTask>();
    cancel.start();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {std::move(res), x::errors::NIL},
        "start_cmd",
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_NE(tsk, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
}

/// @brief a successful configure at boot should not write a status.
TEST(TestHandleConfigErr, testSuccessAtBootSilent) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    ConfigureResult res;
    res.task = std::make_unique<RecordingTask>();
    cancel.start();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {std::move(res), x::errors::NIL},
        "",
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_NE(tsk, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
}

/// @brief it should not auto-start a task that a start command deploys, so the start
/// command starts it with its own key.
TEST(TestHandleConfigErr, testAutoStartLeavesThePendingStart) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    ConfigureResult res;
    auto recording = std::make_unique<RecordingTask>();
    auto *raw = recording.get();
    res.task = std::move(recording);
    res.auto_start = true;
    cancel.start();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {std::move(res), x::errors::NIL},
        "start_cmd",
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_NE(tsk, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
    EXPECT_TRUE(raw->cmds.empty());
}

/// @brief auto-start at boot should show the task as starting, and then exec the start
/// command.
TEST(TestHandleConfigErr, testAutoStartAtBootExecsStart) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
        .config_hash = "hash1"
    };
    ConfigureResult res;
    auto recording = std::make_unique<RecordingTask>();
    auto *raw = recording.get();
    raw->ctx = ctx;
    res.task = std::move(recording);
    res.auto_start = true;
    cancel.start();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {std::move(res), x::errors::NIL},
        "",
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_NE(tsk, nullptr);
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_LOADING);
    EXPECT_EQ(ctx->statuses[0].message, "Starting task");
    EXPECT_EQ(ctx->statuses[0].name, "task1");
    EXPECT_EQ(ctx->statuses[0].details.cmd, "");
    EXPECT_EQ(ctx->statuses[0].details.config_hash, "hash1");
    EXPECT_FALSE(ctx->statuses[0].details.running);
    ASSERT_EQ(raw->cmds.size(), 1);
    EXPECT_EQ(raw->cmds[0].type, synnax::task::START_CMD_TYPE);
    EXPECT_EQ(raw->statuses_at_exec, std::vector<size_t>{1});
}
}
