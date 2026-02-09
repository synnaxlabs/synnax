// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/runtime.h"
#include "arc/cpp/runtime/testutil/mock_loop.h"

using namespace arc::runtime;

namespace {
/// @brief Creates a minimal Runtime for testing queue behavior.
std::shared_ptr<Runtime> create_test_runtime(
    size_t input_capacity,
    arc::runtime::errors::Handler error_handler
) {
    Config cfg{
        .mod = {},
        .breaker = x::breaker::Config{},
        .retrieve_channels = nullptr,
        .input_queue_capacity = input_capacity,
        .output_queue_capacity = 1,
        .loop = {},
    };

    return std::make_shared<Runtime>(
        cfg,
        nullptr,
        nullptr,
        nullptr,
        nullptr,
        nullptr,
        std::vector<arc::types::ChannelKey>{},
        std::vector<arc::types::ChannelKey>{},
        std::move(error_handler)
    );
}

/// @brief Creates a Runtime for lifecycle testing with MockLoop and minimal components.
/// @param loop Unique pointer to the MockLoop (ownership transferred).
/// @returns Pair of (runtime, raw pointer to loop for inspection).
std::pair<std::shared_ptr<Runtime>, testutil::MockLoop *>
create_lifecycle_runtime(std::unique_ptr<testutil::MockLoop> loop) {
    auto *loop_ptr = loop.get();

    state::Config state_cfg{.ir = arc::ir::IR{}, .channels = {}};
    auto state = std::make_shared<state::State>(state_cfg);

    std::unordered_map<std::string, std::unique_ptr<node::Node>> node_impls;
    auto scheduler = std::make_unique<scheduler::Scheduler>(
        arc::ir::IR{},
        node_impls,
        x::telem::TimeSpan(0)
    );

    Config cfg{
        .mod = {},
        .breaker = x::breaker::Config{},
        .retrieve_channels = nullptr,
        .input_queue_capacity = 256,
        .output_queue_capacity = 256,
        .loop = {},
    };

    auto runtime = std::make_shared<Runtime>(
        cfg,
        nullptr,
        nullptr,
        state,
        std::move(scheduler),
        std::move(loop),
        std::vector<arc::types::ChannelKey>{},
        std::vector<arc::types::ChannelKey>{},
        arc::runtime::errors::noop_handler
    );
    return {runtime, loop_ptr};
}
}

/// @brief Test that write() calls error handler with QUEUE_FULL_INPUT when queue is
/// full.
TEST(RuntimeTest, WriteCallsErrorHandlerOnQueueFull) {
    std::vector<x::errors::Error> reported_errors;
    auto runtime = create_test_runtime(1, [&](const x::errors::Error &err) {
        reported_errors.push_back(err);
    });

    auto series1 = x::telem::Series(x::telem::FLOAT32_T, 1);
    series1.write(1.0f);
    auto series2 = x::telem::Series(x::telem::FLOAT32_T, 1);
    series2.write(2.0f);

    ASSERT_NIL(runtime->write(x::telem::Frame(1, std::move(series1))));
    ASSERT_OCCURRED_AS(
        runtime->write(x::telem::Frame(1, std::move(series2))),
        arc::runtime::errors::QUEUE_FULL_INPUT
    );

    ASSERT_EQ(reported_errors.size(), 1);
    ASSERT_MATCHES(reported_errors[0], arc::runtime::errors::QUEUE_FULL_INPUT);
}

/// @brief Test that write() returns error and calls handler for each failed write.
TEST(RuntimeTest, WriteReportsMultipleQueueFullErrors) {
    std::vector<x::errors::Error> reported_errors;
    const auto runtime = create_test_runtime(1, [&](const x::errors::Error &err) {
        reported_errors.push_back(err);
    });

    for (int i = 0; i < 5; i++) {
        auto series = x::telem::Series(x::telem::FLOAT32_T, 1);
        series.write(static_cast<float>(i));
        runtime->write(x::telem::Frame(1, std::move(series)));
    }
    ASSERT_EQ(reported_errors.size(), 4);
    for (const auto &err: reported_errors)
        ASSERT_MATCHES(err, arc::runtime::errors::QUEUE_FULL_INPUT);
}

/// @brief Test that write() succeeds when queue has capacity.
TEST(RuntimeTest, WriteSucceedsWithCapacity) {
    std::vector<x::errors::Error> reported_errors;
    const auto runtime = create_test_runtime(10, [&](const x::errors::Error &err) {
        reported_errors.push_back(err);
    });
    for (int i = 0; i < 5; i++) {
        auto series = x::telem::Series(x::telem::FLOAT32_T, 1);
        series.write(static_cast<float>(i));
        ASSERT_NIL(runtime->write(x::telem::Frame(1, std::move(series))));
    }
    ASSERT_TRUE(reported_errors.empty());
}

/// @brief Test that start() returns false if already running.
TEST(RuntimeLifecycleTest, StartReturnsFalseIfAlreadyRunning) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );
    ASSERT_TRUE(runtime->start());
    EXPECT_FALSE(runtime->start());
    ASSERT_TRUE(runtime->stop());
}

/// @brief Test that stop() returns false if not running.
TEST(RuntimeLifecycleTest, StopReturnsFalseIfNotRunning) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );
    EXPECT_FALSE(runtime->stop());
}

/// @brief Test that stop() returns true when running and calls wake().
TEST(RuntimeLifecycleTest, StopReturnsTrueWhenRunning) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );
    ASSERT_TRUE(runtime->start());
    ASSERT_TRUE(runtime->stop());
    EXPECT_EQ(loop->wake_count, 1);
}

/// @brief Test that runtime can be restarted after stop.
TEST(RuntimeLifecycleTest, RestartSameInstanceSucceeds) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );

    ASSERT_TRUE(runtime->start());
    ASSERT_TRUE(runtime->stop());
    ASSERT_TRUE(runtime->start());
    ASSERT_TRUE(runtime->stop());

    EXPECT_EQ(loop->start_count, 2);
    EXPECT_EQ(loop->wake_count, 2);
}

/// @brief Test that write() returns closed error after stop.
TEST(RuntimeLifecycleTest, WriteReturnsClosedErrorAfterStop) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );
    ASSERT_TRUE(runtime->start());
    ASSERT_TRUE(runtime->stop());

    auto series = x::telem::Series(x::telem::FLOAT32_T, 1);
    series.write(1.0f);
    auto err = runtime->write(x::telem::Frame(1, std::move(series)));
    EXPECT_TRUE(err.matches("runtime closed"));
}

/// @brief Test that rapid start/stop cycles work without issues.
TEST(RuntimeLifecycleTest, RapidStartStopCycles) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );

    for (int i = 0; i < 10; i++) {
        ASSERT_TRUE(runtime->start()) << "Cycle " << i << " start failed";
        ASSERT_TRUE(runtime->stop()) << "Cycle " << i << " stop failed";
    }

    EXPECT_EQ(loop->start_count, 10);
    EXPECT_EQ(loop->wake_count, 10);
}

/// @brief Test that write succeeds after restart (verifies queue reopened).
TEST(RuntimeLifecycleTest, WriteSucceedsAfterRestart) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );

    ASSERT_TRUE(runtime->start());
    ASSERT_TRUE(runtime->stop());

    auto series = x::telem::Series(x::telem::FLOAT32_T, 1);
    series.write(1.0f);
    auto err = runtime->write(x::telem::Frame(1, std::move(series)));
    EXPECT_TRUE(err.matches("runtime closed")) << "Write should fail when stopped";

    ASSERT_TRUE(runtime->start());

    auto series2 = x::telem::Series(x::telem::FLOAT32_T, 1);
    series2.write(2.0f);
    ASSERT_NIL(runtime->write(x::telem::Frame(1, std::move(series2))));

    ASSERT_TRUE(runtime->stop());
}

/// @brief Test that loop's watch() is called during run.
TEST(RuntimeLifecycleTest, LoopWatchCalled) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );
    ASSERT_TRUE(runtime->start());
    ASSERT_EVENTUALLY_GE(loop->watch_count.load(), 1);
    ASSERT_TRUE(runtime->stop());
}

/// @brief Test that loop's wait() is called during run.
TEST(RuntimeLifecycleTest, LoopWaitCalled) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );
    ASSERT_TRUE(runtime->start());
    ASSERT_EVENTUALLY_GE(loop->wait_count.load(), 1);
    ASSERT_TRUE(runtime->stop());
}

/// @brief Test that write works while runtime is running.
TEST(RuntimeLifecycleTest, WriteSucceedsWhileRunning) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );
    ASSERT_TRUE(runtime->start());

    auto series = x::telem::Series(x::telem::FLOAT32_T, 1);
    series.write(1.0f);
    ASSERT_NIL(runtime->write(x::telem::Frame(1, std::move(series))));

    ASSERT_TRUE(runtime->stop());
}

/// @brief Test that read() returns false after stop (output queue closed).
TEST(RuntimeLifecycleTest, ReadReturnsFalseAfterStop) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );
    ASSERT_TRUE(runtime->start());
    ASSERT_TRUE(runtime->stop());

    Output out;
    EXPECT_FALSE(runtime->read(out)) << "Read should return false when stopped";
}

/// @brief Test that double stop returns false on second call.
TEST(RuntimeLifecycleTest, DoubleStopReturnsFalse) {
    auto [runtime, loop] = create_lifecycle_runtime(
        std::make_unique<testutil::MockLoop>()
    );
    ASSERT_TRUE(runtime->start());
    ASSERT_TRUE(runtime->stop());
    EXPECT_FALSE(runtime->stop()) << "Second stop should return false";
}

TEST(BuildAuthoritiesTest, ReturnsEmptyWhenNoConfig) {
    arc::ir::Authorities auth;
    std::vector<arc::types::ChannelKey> write_keys = {1, 2, 3};
    auto result = build_authorities(auth, write_keys);
    EXPECT_TRUE(result.empty());
}

TEST(BuildAuthoritiesTest, DefaultAuthorityAppliesToAllKeys) {
    arc::ir::Authorities auth;
    auth.default_authority = 100;
    std::vector<arc::types::ChannelKey> write_keys = {1, 2, 3};
    auto result = build_authorities(auth, write_keys);
    ASSERT_EQ(result.size(), 3);
    for (const auto &a: result)
        EXPECT_EQ(a, 100);
}

TEST(BuildAuthoritiesTest, PerChannelOverridesDefault) {
    arc::ir::Authorities auth;
    auth.default_authority = 100;
    auth.channels[2] = 200;

    std::vector<arc::types::ChannelKey> write_keys = {1, 2, 3};
    auto result = build_authorities(auth, write_keys);
    ASSERT_EQ(result.size(), 3);
    EXPECT_EQ(result[0], 100);
    EXPECT_EQ(result[1], 200);
    EXPECT_EQ(result[2], 100);
}

TEST(BuildAuthoritiesTest, NoDefaultUsesAbsolute) {
    arc::ir::Authorities auth;
    auth.channels[1] = 50;

    std::vector<arc::types::ChannelKey> write_keys = {1, 2};
    auto result = build_authorities(auth, write_keys);
    ASSERT_EQ(result.size(), 2);
    EXPECT_EQ(result[0], 50);
    EXPECT_EQ(result[1], x::telem::AUTH_ABSOLUTE);
}

TEST(MockLoopTest, WakeReasonIsConfigurable) {
    testutil::MockLoop loop;
    x::breaker::Breaker breaker(x::breaker::Config{});
    breaker.start();

    loop.wake_reason = loop::WakeReason::Timer;
    ASSERT_EQ(loop.wait(breaker), loop::WakeReason::Timer);

    loop.wake_reason = loop::WakeReason::Input;
    ASSERT_EQ(loop.wait(breaker), loop::WakeReason::Input);

    loop.wake_reason = loop::WakeReason::Timeout;
    ASSERT_EQ(loop.wait(breaker), loop::WakeReason::Timeout);

    loop.wake_reason = loop::WakeReason::Shutdown;
    ASSERT_EQ(loop.wait(breaker), loop::WakeReason::Shutdown);

    breaker.stop();
}
