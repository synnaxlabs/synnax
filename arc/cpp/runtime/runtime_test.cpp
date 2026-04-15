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

#include "arc/cpp/ir/testutil/testutil.h"
#include "arc/cpp/runtime/runtime.h"
#include "arc/cpp/runtime/testutil/mock_loop.h"

namespace arc::runtime {
/// @brief Creates a minimal Runtime for testing queue behavior.
std::shared_ptr<Runtime> create_test_runtime(
    size_t input_capacity,
    arc::runtime::errors::Handler error_handler
) {
    Config cfg{
        .program = {},
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
        .program = {},
        .breaker = x::breaker::Config{},
        .retrieve_channels = nullptr,
        .input_queue_capacity = 256,
        .output_queue_capacity = 256,
        .loop = {},
    };

    auto runtime = std::make_shared<Runtime>(
        cfg,
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

TEST(RuntimeLifecycleTest, LoopStartFailureCallsErrorHandler) {
    auto mock = std::make_unique<testutil::MockLoop>();
    mock->start_error = x::errors::Error("epoll_create failed");
    auto *loop_ptr = mock.get();

    state::Config state_cfg{.ir = arc::ir::IR{}, .channels = {}};
    auto state = std::make_shared<state::State>(state_cfg);
    std::unordered_map<std::string, std::unique_ptr<node::Node>> node_impls;
    auto scheduler = std::make_unique<scheduler::Scheduler>(
        arc::ir::IR{},
        node_impls,
        x::telem::TimeSpan(0)
    );

    std::atomic<bool> error_called{false};
    x::errors::Error captured_error;

    Config cfg{
        .program = {},
        .breaker = x::breaker::Config{},
        .retrieve_channels = nullptr,
        .input_queue_capacity = 256,
        .output_queue_capacity = 256,
        .loop = {},
    };

    auto runtime = std::make_shared<Runtime>(
        cfg,
        nullptr,
        state,
        std::move(scheduler),
        std::move(mock),
        std::vector<arc::types::ChannelKey>{},
        std::vector<arc::types::ChannelKey>{},
        [&](const x::errors::Error &err) {
            captured_error = err;
            error_called = true;
        }
    );

    ASSERT_TRUE(runtime->start());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    EXPECT_TRUE(error_called.load());
    EXPECT_TRUE(captured_error.matches(x::errors::Error("epoll_create failed")));
    EXPECT_EQ(loop_ptr->watch_count.load(), 0);
    runtime->stop();
}

TEST(BuildAuthoritiesTest, ReturnsEmptyWhenNoConfig) {
    arc::ir::Authorities auth;
    std::vector<arc::types::ChannelKey> write_keys = {1, 2, 3};
    auto result = build_authorities(auth, write_keys);
    EXPECT_TRUE(result.empty());
}

TEST(BuildAuthoritiesTest, DefaultAuthorityAppliesToAllKeys) {
    arc::ir::Authorities auth;
    auth.default_ = 100;
    std::vector<arc::types::ChannelKey> write_keys = {1, 2, 3};
    auto result = build_authorities(auth, write_keys);
    ASSERT_EQ(result.size(), 3);
    for (const auto &a: result)
        EXPECT_EQ(a, 100);
}

TEST(BuildAuthoritiesTest, PerChannelOverridesDefault) {
    arc::ir::Authorities auth;
    auth.default_ = 100;
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
    EXPECT_EQ(result[1], x::control::AUTHORITY_ABSOLUTE);
}

/// @brief Mock node that sets a configurable deadline on each execution.
struct DeadlineNode final : public node::Node {
    std::atomic<int64_t> deadline_ns;

    explicit DeadlineNode(const x::telem::TimeSpan deadline):
        deadline_ns(deadline.nanoseconds()) {}

    x::errors::Error next(node::Context &ctx) override {
        const auto d = x::telem::TimeSpan(this->deadline_ns.load());
        if (d != x::telem::TimeSpan::max()) ctx.set_deadline(d);
        return x::errors::NIL;
    }

    void reset() override {}

    [[nodiscard]] bool is_output_truthy(const std::string &) const override {
        return false;
    }
};

/// @brief Creates a runtime with a DeadlineNode for testing timeout conversion.
struct DeadlineRuntimeFixture {
    std::shared_ptr<Runtime> runtime;
    testutil::MockLoop *loop;
    DeadlineNode *node;

    static DeadlineRuntimeFixture create(const x::telem::TimeSpan deadline) {
        auto mock_loop = std::make_unique<testutil::MockLoop>();
        auto *loop_ptr = mock_loop.get();

        auto deadline_node = std::make_unique<DeadlineNode>(deadline);
        auto *node_ptr = deadline_node.get();

        auto prog = arc::ir::testutil::Builder()
                        .node("deadline")
                        .phases({{"deadline"}})
                        .build();

        state::Config state_cfg{.ir = prog, .channels = {}};
        auto state = std::make_shared<state::State>(state_cfg);

        std::unordered_map<std::string, std::unique_ptr<node::Node>> node_impls;
        node_impls["deadline"] = std::move(deadline_node);

        auto scheduler = std::make_unique<scheduler::Scheduler>(
            prog,
            node_impls,
            x::telem::TimeSpan(0)
        );

        Config cfg{
            .program = {},
            .breaker = x::breaker::Config{},
            .retrieve_channels = nullptr,
            .input_queue_capacity = 256,
            .output_queue_capacity = 256,
            .loop = {},
        };

        auto rt = std::make_shared<Runtime>(
            cfg,
            nullptr,
            state,
            std::move(scheduler),
            std::move(mock_loop),
            std::vector<arc::types::ChannelKey>{},
            std::vector<arc::types::ChannelKey>{},
            arc::runtime::errors::noop_handler
        );

        return {rt, loop_ptr, node_ptr};
    }
};

/// @brief When no deadline is set, runtime should pass max_timeout=0 (no constraint)
/// on every cycle.
TEST(RuntimeDeadlineTest, NoDeadlinePassesZeroTimeout) {
    auto [runtime, loop, node] = DeadlineRuntimeFixture::create(
        x::telem::TimeSpan::max()
    );
    ASSERT_TRUE(runtime->start());
    ASSERT_EVENTUALLY_GE(loop->wait_count.load(), 3);
    ASSERT_TRUE(runtime->stop());

    const auto timeouts = loop->get_max_timeouts();
    ASSERT_GE(timeouts.size(), 3);
    for (const auto &t: timeouts)
        EXPECT_EQ(t, x::telem::TimeSpan(0)) << "Expected 0 (no deadline constraint)";
}

/// @brief When a deadline is 10s in the future, runtime should pass a timeout close
/// to 10s that decreases as elapsed time grows.
TEST(RuntimeDeadlineTest, FutureDeadlinePassesDecreasingTimeout) {
    auto [runtime, loop, node] = DeadlineRuntimeFixture::create(x::telem::SECOND * 10);
    ASSERT_TRUE(runtime->start());
    ASSERT_EVENTUALLY_GE(loop->wait_count.load(), 3);
    ASSERT_TRUE(runtime->stop());

    const auto timeouts = loop->get_max_timeouts();
    ASSERT_GE(timeouts.size(), 3);
    // First timeout after the initial cycle is close to 10s (minus small elapsed).
    // Skip index 0 which is the initial 0-timeout seed.
    EXPECT_GT(timeouts[1], x::telem::SECOND * 9);
    EXPECT_LE(timeouts[1], x::telem::SECOND * 10);
    // Subsequent timeouts should be smaller as elapsed grows.
    for (size_t i = 2; i < timeouts.size(); i++)
        EXPECT_LE(timeouts[i], timeouts[i - 1]);
}

/// @brief When the deadline is always in the past (1ns), runtime should pass 1ns
/// on every cycle after the first.
TEST(RuntimeDeadlineTest, PastDeadlinePassesMinimalTimeout) {
    auto [runtime, loop, node] = DeadlineRuntimeFixture::create(x::telem::TimeSpan(1));
    ASSERT_TRUE(runtime->start());
    ASSERT_EVENTUALLY_GE(loop->wait_count.load(), 3);
    ASSERT_TRUE(runtime->stop());

    const auto timeouts = loop->get_max_timeouts();
    ASSERT_GE(timeouts.size(), 3);
    // After the initial seed cycle (index 0), every timeout should be 1ns
    // because 1ns deadline is always < elapsed wall-clock time.
    for (size_t i = 1; i < timeouts.size(); i++)
        EXPECT_EQ(timeouts[i], x::telem::TimeSpan(1));
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
}
