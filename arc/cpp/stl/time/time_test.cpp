// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/time/time.h"

namespace arc::stl {
runtime::node::Context make_context(
    const ::telem::TimeSpan elapsed,
    const ::telem::TimeSpan tolerance = ::telem::TimeSpan(0),
    const runtime::node::RunReason reason = runtime::node::RunReason::TimerTick
) {
    return runtime::node::Context{
        .elapsed = elapsed,
        .tolerance = tolerance,
        .reason = reason,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const xerrors::Error &) {},
        .activate_stage = [] {},
    };
}

struct TestSetup {
    ir::IR ir;
    runtime::state::State state;

    TestSetup(const std::string &type, const std::string &param_name, const int64_t ns):
        ir(build_ir(type, param_name, ns)),
        state(
            runtime::state::Config{.ir = ir, .channels = {}},
            runtime::errors::noop_handler
        ) {}

    runtime::state::Node make_node() { return ASSERT_NIL_P(state.node("timer")); }

private:
    static ir::IR
    build_ir(const std::string &type, const std::string &param_name, const int64_t ns) {
        ir::Param output_param;
        output_param.name = "output";
        output_param.type = types::Type(types::Kind::U8);

        ir::Param cfg_param;
        cfg_param.name = param_name;
        cfg_param.type = types::Type(types::Kind::I64);
        cfg_param.value = ns;

        ir::Node ir_node;
        ir_node.key = "timer";
        ir_node.type = type;
        ir_node.outputs.params.push_back(output_param);
        ir_node.config.params.push_back(cfg_param);

        ir::Function fn;
        fn.key = "test";

        ir::IR ir;
        ir.nodes.push_back(ir_node);
        ir.functions.push_back(fn);
        return ir;
    }
};

/// @brief Test that factory returns NOT_FOUND for non-time node types.
TEST(TimeFactoryTest, ReturnsNotFoundForWrongType) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    auto ir_node = setup.ir.nodes[0];
    ir_node.type = "not_a_time_node";

    time::Factory factory;
    ASSERT_OCCURRED_AS_P(
        factory.create(runtime::node::Config(setup.ir, ir_node, setup.make_node())),
        xerrors::NOT_FOUND
    );
}

/// @brief Test that factory creates an Interval node from valid configuration.
TEST(TimeFactoryTest, CreatesIntervalNode) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    time::Factory factory;
    const auto node = ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that factory creates a Wait node from valid configuration.
TEST(TimeFactoryTest, CreatesWaitNode) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    time::Factory factory;
    const auto node = ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that base_interval is set to the first interval when uninitialized.
TEST(TimeFactoryTest, BaseIntervalSetToFirstInterval) {
    TestSetup setup("interval", "period", (500 * ::telem::MILLISECOND).nanoseconds());
    time::Factory factory;
    ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    EXPECT_EQ(factory.base_interval(), 500 * ::telem::MILLISECOND);
}

/// @brief Test that base_interval computes GCD across multiple intervals.
TEST(TimeFactoryTest, BaseIntervalComputesGCDAcrossNodes) {
    TestSetup setup1("interval", "period", (600 * ::telem::MILLISECOND).nanoseconds());
    TestSetup setup2("wait", "duration", (400 * ::telem::MILLISECOND).nanoseconds());

    time::Factory factory;
    ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup1.ir, setup1.ir.nodes[0], setup1.make_node())
    ));
    ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup2.ir, setup2.ir.nodes[0], setup2.make_node())
    ));
    EXPECT_EQ(factory.base_interval(), 200 * ::telem::MILLISECOND);
}

/// @brief Test that Interval does not fire again before next interval elapses.
TEST(IntervalTest, DoesNotFireBeforeNextIntervalElapses) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    auto ctx2 = make_context(::telem::MILLISECOND * 500);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Interval fires when the interval is reached.
TEST(IntervalTest, FiresWhenIntervalReached) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx = make_context(::telem::SECOND);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 1);
}

/// @brief Test that Interval fires repeatedly at each interval.
TEST(IntervalTest, FiresRepeatedly) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::SECOND);
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);

    auto ctx2 = make_context(::telem::SECOND * 2);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);

    auto ctx3 = make_context(::telem::SECOND * 3);
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Interval sets the timestamp to elapsed time when firing.
TEST(IntervalTest, SetsTimestampOnFire) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx = make_context(::telem::SECOND * 5);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output_time->at<int64_t>(0), (::telem::SECOND * 5).nanoseconds());
}

/// @brief Test that Interval calls mark_changed when firing.
TEST(IntervalTest, CallsMarkChangedOnFire) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    bool changed_called = false;
    std::string changed_param;
    auto ctx = make_context(::telem::SECOND);
    ctx.mark_changed = [&](const std::string &param) {
        changed_called = true;
        changed_param = param;
    };

    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed_called);
    EXPECT_EQ(changed_param, "output");
}

/// @brief Test that Interval does not call mark_changed when not firing.
TEST(IntervalTest, DoesNotCallMarkChangedWhenNotFiring) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::SECOND);
    node.next(ctx1);

    int call_count = 0;
    auto ctx2 = make_context(::telem::SECOND + ::telem::MILLISECOND * 100);
    ctx2.mark_changed = [&](const std::string &) { call_count++; };
    node.next(ctx2);

    EXPECT_EQ(call_count, 0);
}

/// @brief Test that Interval is_output_truthy delegates to state.
TEST(IntervalTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx = make_context(::telem::SECOND);
    node.next(ctx);

    EXPECT_TRUE(node.is_output_truthy("output"));
}

/// @brief Test that Interval is_output_truthy returns false before firing.
TEST(IntervalTest, IsOutputTruthyFalseBeforeFiring) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    EXPECT_FALSE(node.is_output_truthy("output"));
}

/// @brief Test that Interval is_output_truthy returns false for unknown param.
TEST(IntervalTest, IsOutputTruthyFalseForUnknownParam) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx = make_context(::telem::SECOND);
    node.next(ctx);

    EXPECT_FALSE(node.is_output_truthy("nonexistent"));
}

/// @brief Test that Interval reset allows it to fire immediately again.
TEST(IntervalTest, ResetAllowsImmediateFiring) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    node.next(ctx1);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    auto ctx2 = make_context(::telem::MILLISECOND * 500);
    node.next(ctx2);
    EXPECT_EQ(output->size(), 0);

    node.reset();

    auto ctx3 = make_context(::telem::MILLISECOND * 600);
    node.next(ctx3);
    EXPECT_EQ(output->size(), 1);
}

TEST(IntervalTest, OnlyFiresOnTimerTick) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    bool changed_called = false;
    runtime::node::Context ctx;
    ctx.elapsed = ::telem::SECOND;
    ctx.tolerance = ::telem::TimeSpan(0);
    ctx.mark_changed = [&changed_called](const std::string &) {
        changed_called = true;
    };
    ctx.report_error = [](const xerrors::Error &) {};
    ctx.activate_stage = []() {};

    ctx.reason = runtime::node::RunReason::TimerTick;
    ASSERT_NIL(node.next(ctx));
    ASSERT_TRUE(changed_called);

    changed_called = false;
    ctx.elapsed = ::telem::SECOND + ::telem::MILLISECOND * 500;
    ctx.reason = runtime::node::RunReason::ChannelInput;
    ASSERT_NIL(node.next(ctx));
    ASSERT_FALSE(changed_called);

    changed_called = false;
    ctx.reason = runtime::node::RunReason::TimerTick;
    ctx.elapsed = ::telem::SECOND * 2;
    ASSERT_NIL(node.next(ctx));
    ASSERT_TRUE(changed_called);
}

/// @brief Test that Wait does not fire before the duration elapses.
TEST(WaitTest, DoesNotFireBeforeDurationElapses) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx = make_context(::telem::MILLISECOND * 500);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Wait fires once after the duration elapses.
TEST(WaitTest, FiresOnceAfterDuration) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(::telem::SECOND);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 1);
}

/// @brief Test that Wait does not fire again after the first fire.
TEST(WaitTest, DoesNotFireAgain) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(::telem::SECOND);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    auto ctx3 = make_context(::telem::SECOND * 5);
    node.next(ctx3);

    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Wait reset allows it to fire again.
TEST(WaitTest, ResetAllowsFiringAgain) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(::telem::SECOND);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    node.reset();

    auto ctx3 = make_context(::telem::SECOND * 2);
    node.next(ctx3);

    auto ctx4 = make_context(::telem::SECOND * 3);
    node.next(ctx4);

    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 1);
}

TEST(WaitTest, OnlyFiresOnTimerTick) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    bool changed_called = false;
    runtime::node::Context ctx;
    ctx.elapsed = ::telem::TimeSpan(0);
    ctx.tolerance = ::telem::TimeSpan(0);
    ctx.mark_changed = [&changed_called](const std::string &) {
        changed_called = true;
    };
    ctx.report_error = [](const xerrors::Error &) {};
    ctx.activate_stage = []() {};

    ctx.reason = runtime::node::RunReason::TimerTick;
    ASSERT_NIL(node.next(ctx));
    ASSERT_FALSE(changed_called);

    ctx.elapsed = ::telem::MILLISECOND * 500;
    ctx.reason = runtime::node::RunReason::ChannelInput;
    ASSERT_NIL(node.next(ctx));
    ASSERT_FALSE(changed_called);

    ctx.elapsed = ::telem::SECOND;
    ctx.reason = runtime::node::RunReason::ChannelInput;
    ASSERT_NIL(node.next(ctx));
    ASSERT_FALSE(changed_called);

    ctx.reason = runtime::node::RunReason::TimerTick;
    ASSERT_NIL(node.next(ctx));
    ASSERT_TRUE(changed_called);
}

/// @brief Test that Wait measures duration from first next() call, not construction.
TEST(WaitTest, MeasuresDurationFromFirstNextCall) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::SECOND * 10);
    node.next(ctx1);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(::telem::SECOND * 11);
    node.next(ctx2);

    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait sets the timestamp to elapsed time when firing.
TEST(WaitTest, SetsTimestampOnFire) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::SECOND * 2);
    node.next(ctx1);

    auto ctx2 = make_context(::telem::SECOND * 3);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output_time->at<int64_t>(0), (::telem::SECOND * 3).nanoseconds());
}

/// @brief Test that Wait calls mark_changed when firing.
TEST(WaitTest, CallsMarkChangedOnFire) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    node.next(ctx1);

    bool changed_called = false;
    std::string changed_param;
    auto ctx2 = make_context(::telem::SECOND);
    ctx2.mark_changed = [&](const std::string &param) {
        changed_called = true;
        changed_param = param;
    };

    node.next(ctx2);
    EXPECT_TRUE(changed_called);
    EXPECT_EQ(changed_param, "output");
}

/// @brief Test that Wait does not call mark_changed when not firing.
TEST(WaitTest, DoesNotCallMarkChangedWhenNotFiring) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    int call_count = 0;
    auto ctx = make_context(::telem::MILLISECOND * 100);
    ctx.mark_changed = [&](const std::string &) { call_count++; };
    node.next(ctx);

    EXPECT_EQ(call_count, 0);
}

/// @brief Test that Wait is_output_truthy delegates to state.
TEST(WaitTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(::telem::SECOND);
    node.next(ctx2);

    EXPECT_TRUE(node.is_output_truthy("output"));
}

/// @brief Test that Wait reset restarts timing from zero.
TEST(WaitTest, ResetRestartsTimingFromZero) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::SECOND * 5);
    node.next(ctx1);

    node.reset();

    auto ctx2 = make_context(::telem::SECOND * 5 + ::telem::MILLISECOND * 500);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx3 = make_context(::telem::SECOND * 6 + ::telem::MILLISECOND * 500);
    node.next(ctx3);

    EXPECT_EQ(output->size(), 1);
}

/// @brief Test calculate_tolerance for RT_EVENT mode.
TEST(CalculateToleranceTest, RTEventMode) {
    const auto tolerance = time::calculate_tolerance(
        runtime::loop::ExecutionMode::RT_EVENT,
        100 * ::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 100 * ::telem::MICROSECOND);
}

/// @brief Test calculate_tolerance for BUSY_WAIT mode.
TEST(CalculateToleranceTest, BusyWaitMode) {
    const auto tolerance = time::calculate_tolerance(
        runtime::loop::ExecutionMode::BUSY_WAIT,
        100 * ::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 100 * ::telem::MICROSECOND);
}

/// @brief Test calculate_tolerance for HIGH_RATE mode.
TEST(CalculateToleranceTest, HighRateMode) {
    const auto tolerance = time::calculate_tolerance(
        runtime::loop::ExecutionMode::HIGH_RATE,
        100 * ::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, ::telem::MILLISECOND);
}

/// @brief Test calculate_tolerance for EVENT_DRIVEN mode.
TEST(CalculateToleranceTest, EventDrivenMode) {
    const auto tolerance = time::calculate_tolerance(
        runtime::loop::ExecutionMode::EVENT_DRIVEN,
        100 * ::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 5 * ::telem::MILLISECOND);
}

/// @brief Test calculate_tolerance for HYBRID mode.
TEST(CalculateToleranceTest, HybridMode) {
    const auto tolerance = time::calculate_tolerance(
        runtime::loop::ExecutionMode::HYBRID,
        100 * ::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 5 * ::telem::MILLISECOND);
}

/// @brief Test calculate_tolerance with max interval returns fixed 5ms.
TEST(CalculateToleranceTest, MaxInterval) {
    const auto tolerance = time::calculate_tolerance(
        runtime::loop::ExecutionMode::EVENT_DRIVEN,
        ::telem::TimeSpan::max()
    );
    EXPECT_EQ(tolerance, 5 * ::telem::MILLISECOND);
}

/// @brief Test calculate_tolerance respects half-interval minimum.
TEST(CalculateToleranceTest, HalfIntervalMinimum) {
    const auto tolerance = time::calculate_tolerance(
        runtime::loop::ExecutionMode::EVENT_DRIVEN,
        4 * ::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 2 * ::telem::MILLISECOND);
}

/// @brief Test that Interval fires within tolerance.
TEST(IntervalToleranceTest, FiresWithinTolerance) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    output->resize(0);

    auto ctx2 = make_context(
        ::telem::SECOND * 2 - ::telem::MILLISECOND * 5,
        50 * ::telem::MILLISECOND
    );
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Interval does not fire too early even with tolerance.
TEST(IntervalToleranceTest, DoesNotFireTooEarly) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    output->resize(0);

    auto ctx2 = make_context(::telem::MILLISECOND * 900, 50 * ::telem::MILLISECOND);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Wait fires within tolerance.
TEST(WaitToleranceTest, FiresWithinTolerance) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(
        ::telem::SECOND - ::telem::MILLISECOND * 5,
        50 * ::telem::MILLISECOND
    );
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait does not fire too early even with tolerance.
TEST(WaitToleranceTest, DoesNotFireTooEarly) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(::telem::MILLISECOND * 900, 50 * ::telem::MILLISECOND);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Interval fires correctly with zero tolerance (original behavior).
TEST(IntervalToleranceTest, ZeroToleranceRequiresExactTime) {
    TestSetup setup("interval", "period", ::telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    output->resize(0);

    auto ctx2 = make_context(
        ::telem::SECOND - ::telem::NANOSECOND,
        ::telem::TimeSpan(0)
    );
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);

    auto ctx3 = make_context(::telem::SECOND, ::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait fires correctly with zero tolerance (original behavior).
TEST(WaitToleranceTest, ZeroToleranceRequiresExactTime) {
    TestSetup setup("wait", "duration", ::telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(::telem::TimeSpan(0), ::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(
        ::telem::SECOND - ::telem::NANOSECOND,
        ::telem::TimeSpan(0)
    );
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);

    auto ctx3 = make_context(::telem::SECOND, ::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that tolerance is capped at half the interval for small intervals.
TEST(CalculateToleranceTest, SmallIntervalCapsAtHalf) {
    const auto tolerance = time::calculate_tolerance(
        runtime::loop::ExecutionMode::RT_EVENT,
        100 * ::telem::MICROSECOND
    );
    EXPECT_EQ(tolerance, 50 * ::telem::MICROSECOND);
}

/// @brief Test calculate_tolerance for AUTO mode (defaults to EVENT_DRIVEN behavior).
TEST(CalculateToleranceTest, AutoMode) {
    const auto tolerance = time::calculate_tolerance(
        runtime::loop::ExecutionMode::AUTO,
        100 * ::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 5 * ::telem::MILLISECOND);
}
}
