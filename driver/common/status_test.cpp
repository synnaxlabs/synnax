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
    EXPECT_EQ(first.key, synnax::task::status_key(task));
    EXPECT_EQ(first.details.cmd, "cmd_key");
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.details.task, task.key);
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
    EXPECT_EQ(second.key, synnax::task::status_key(task));
    EXPECT_EQ(second.details.cmd, "cmd_key");
    EXPECT_EQ(second.name, "task1");
    EXPECT_EQ(second.details.task, task.key);
    EXPECT_EQ(second.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(second.details.running, false);
    EXPECT_EQ(second.message, "Failed to start task");
    EXPECT_EQ(second.description, "task validation error");
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
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(first.message, "Test warning message");

    handler.error(
        x::errors::Error(x::errors::VALIDATION, "task validation error"),
        "Failed to start task"
    );
    handler.send_warning("This warning should not be sent", "");
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.details.task, task.key);
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
    EXPECT_EQ(first.details.task, task.key);
    EXPECT_EQ(first.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(first.message, "Test warning message");

    // Now clear the warning
    handler.clear_warning();
    ASSERT_GE(ctx->statuses.size(), 2);
    const auto second = ctx->statuses[1];
    EXPECT_EQ(second.details.task, task.key);
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
    EXPECT_EQ(first.key, synnax::task::status_key(task));
    EXPECT_EQ(first.name, "task1");
    EXPECT_EQ(first.details.task, task.key);
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
    EXPECT_EQ(first.key, synnax::task::status_key(task));
    EXPECT_EQ(first.details.cmd, "cmd_key");
    EXPECT_EQ(first.details.task, task.key);
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
    EXPECT_EQ(second.key, synnax::task::status_key(task));
    EXPECT_EQ(second.details.cmd, "cmd_key");
    EXPECT_EQ(second.details.task, task.key);
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
    EXPECT_EQ(third.key, synnax::task::status_key(task));
    EXPECT_EQ(third.details.cmd, "cmd_key_2");
    EXPECT_EQ(third.details.task, task.key);
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
    EXPECT_EQ(first.key, synnax::task::status_key(task));
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
    const auto window = 20 * x::telem::MILLISECOND;
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

/// @brief it should send the same status again once the window ends.
TEST(TestStatusWindow, testWindowExpires) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto handler = StatusHandler(ctx, task, 1 * x::telem::MILLISECOND);

    handler.send_warning("array mismatch", "");
    EXPECT_EQ(ctx->snapshot().size(), 1);

    handler.send_warning("array mismatch", "");
    EXPECT_EQ(ctx->snapshot().size(), 1);

    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    handler.send_warning("array mismatch", "");
    EXPECT_EQ(ctx->snapshot().size(), 2);
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
    void exec(synnax::task::Command &cmd) override { this->cmds.push_back(cmd); }
    void stop(bool will_reconfigure) override {}
};

/// @brief a config error during a start-triggered deploy should write an error
/// status.
TEST(TestHandleConfigErr, testErrorWithPendingStart) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{
        .name = "task1",
        .type = "ni_analog_read",
        .config_hash = "hash1"
    };
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {ConfigureResult{}, x::errors::Error(x::errors::VALIDATION, "bad config")},
        "start_cmd"
    );
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses[0].message, "Failed to configure task");
    EXPECT_EQ(ctx->statuses[0].description, "bad config");
    EXPECT_EQ(ctx->statuses[0].details.cmd, "start_cmd");
    EXPECT_EQ(ctx->statuses[0].details.config_hash, "hash1");
}

/// @brief a config error should list each field error on its own line.
TEST(TestHandleConfigErr, ListsFieldErrors) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    x::json::Parser parser(x::json::json::object());
    parser.field_err("device", "this field is required");
    parser.field_err("channels.0.port", "must be positive");
    handle_config_err(ctx, task, {ConfigureResult{}, parser.error()}, "start_cmd");
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].message, "Failed to configure task");
    EXPECT_EQ(
        ctx->statuses[0].description,
        "device: this field is required\nchannels.0.port: must be positive"
    );
}

/// @brief a config error at boot with no auto-start should not write a status.
TEST(TestHandleConfigErr, testErrorAtBootSilent) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {ConfigureResult{}, x::errors::Error(x::errors::VALIDATION, "bad config")},
        ""
    );
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
}

/// @brief a config error at boot with auto-start requested should write an error
/// status.
TEST(TestHandleConfigErr, testErrorAtBootWithAutoStart) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    ConfigureResult res;
    res.auto_start = true;
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {std::move(res), x::errors::Error(x::errors::VALIDATION, "bad config")},
        ""
    );
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_TRUE(ctx->statuses[0].details.cmd.empty());
}

/// @brief a successful configure during a deploy should write no status. The start
/// that follows it is what answers the command.
TEST(TestHandleConfigErr, testSuccessWithPendingStart) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    ConfigureResult res;
    res.task = std::make_unique<RecordingTask>();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {std::move(res), x::errors::NIL},
        "start_cmd"
    );
    EXPECT_TRUE(handled);
    EXPECT_NE(tsk, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
}

/// @brief a successful configure at boot should not write a status.
TEST(TestHandleConfigErr, testSuccessAtBootSilent) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    ConfigureResult res;
    res.task = std::make_unique<RecordingTask>();
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {std::move(res), x::errors::NIL},
        ""
    );
    EXPECT_TRUE(handled);
    EXPECT_NE(tsk, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
}

/// @brief auto-start at boot should exec the start command instead of writing a
/// configure status.
TEST(TestHandleConfigErr, testAutoStartAtBootExecsStart) {
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const synnax::task::Task task{.name = "task1", .type = "ni_analog_read"};
    ConfigureResult res;
    auto recording = std::make_unique<RecordingTask>();
    auto *raw = recording.get();
    res.task = std::move(recording);
    res.auto_start = true;
    auto [tsk, handled] = handle_config_err(
        ctx,
        task,
        {std::move(res), x::errors::NIL},
        ""
    );
    EXPECT_TRUE(handled);
    EXPECT_NE(tsk, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
    ASSERT_EQ(raw->cmds.size(), 1);
    EXPECT_EQ(raw->cmds[0].type, synnax::task::START_CMD_TYPE);
}
}
