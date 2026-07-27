// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/mem/indirect.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/testutil/compile.h"
#include "arc/cpp/stl/time/time.h"

namespace arc::stl::time {
runtime::node::Context make_context(
    const x::telem::TimeSpan elapsed,
    const x::telem::TimeSpan tolerance = x::telem::TimeSpan(0),
    const runtime::node::RunReason reason = runtime::node::RunReason::TimerTick
) {
    return runtime::node::Context{
        .elapsed = elapsed,
        .tolerance = tolerance,
        .reason = reason,
        .mark_changed = [](size_t) {},
        .mark_self_changed = [] {},
        .set_deadline = [](x::telem::TimeSpan) {},
        .report_error = [](const x::errors::Error &) {},
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
        arc::types::Param output_param;
        output_param.name = "output";
        output_param.type = arc::types::Type{.kind = arc::types::Kind::U8};

        arc::types::Param cfg_param;
        cfg_param.name = param_name;
        cfg_param.type = arc::types::Type{.kind = arc::types::Kind::I64};
        cfg_param.value = ns;

        ir::Node ir_node;
        ir_node.key = "timer";
        ir_node.type = type;
        ir_node.outputs.push_back(output_param);
        ir_node.inputs.push_back(cfg_param);

        ir::Function fn;
        fn.key = "test";

        ir::IR ir;
        ir.nodes.push_back(ir_node);
        ir.functions.push_back(fn);
        return ir;
    }
};

TEST(IntervalInputsTest, CreatesInputsFromValidParams) {
    types::Param period_param;
    period_param.name = "period";
    period_param.type = types::Type{.kind = types::Kind::I64};
    period_param.value = x::telem::SECOND.nanoseconds();
    types::Params params;
    params.push_back(period_param);
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(params));
    EXPECT_EQ(inputs.interval, x::telem::SECOND);
}

TEST(IntervalInputsTest, ReturnsErrorForNullPeriod) {
    types::Param period_param;
    period_param.name = "period";
    period_param.type = types::Type{.kind = types::Kind::I64};
    period_param.value = nullptr;
    types::Params params;
    params.push_back(period_param);
    ASSERT_OCCURRED_AS_P(IntervalInputs::create(params), x::errors::VALIDATION);
}

TEST(WaitInputsTest, CreatesInputsFromValidParams) {
    types::Param duration_param;
    duration_param.name = "duration";
    duration_param.type = types::Type{.kind = types::Kind::I64};
    duration_param.value = x::telem::SECOND.nanoseconds();
    types::Params params;
    params.push_back(duration_param);
    const auto inputs = ASSERT_NIL_P(WaitInputs::create(params));
    EXPECT_EQ(inputs.duration, x::telem::SECOND);
}

TEST(WaitInputsTest, ReturnsErrorForNullDuration) {
    types::Param duration_param;
    duration_param.name = "duration";
    duration_param.type = types::Type{.kind = types::Kind::I64};
    duration_param.value = nullptr;
    types::Params params;
    params.push_back(duration_param);
    ASSERT_OCCURRED_AS_P(WaitInputs::create(params), x::errors::VALIDATION);
}

/// @brief Test that module returns NOT_FOUND for non-time node types.
TEST(TimeModuleTest, ReturnsNotFoundForWrongType) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    auto ir_node = setup.ir.nodes[0];
    ir_node.type = "not_a_time_node";

    Module factory;
    ASSERT_OCCURRED_AS_P(
        factory.create(runtime::node::Config(setup.ir, ir_node, setup.make_node())),
        x::errors::NOT_FOUND
    );
}

/// @brief Test that module creates an Interval node from valid configuration.
TEST(TimeModuleTest, CreatesIntervalNode) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    Module factory;
    const auto node = ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that MultiFactory strips prefix for qualified time.interval type.
TEST(TimeModuleTest, CreatesIntervalNodeQualifiedViaMultiFactory) {
    TestSetup setup("time.interval", "period", x::telem::SECOND.nanoseconds());
    auto module = std::make_shared<Module>();
    runtime::node::MultiFactory multi({module});
    const auto node = ASSERT_NIL_P(multi.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that module creates a Wait node from valid configuration.
TEST(TimeModuleTest, CreatesWaitNode) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Module factory;
    const auto node = ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that MultiFactory strips prefix for qualified time.wait type.
TEST(TimeModuleTest, CreatesWaitNodeQualifiedViaMultiFactory) {
    TestSetup setup("time.wait", "duration", x::telem::SECOND.nanoseconds());
    auto module = std::make_shared<Module>();
    runtime::node::MultiFactory multi({module});
    const auto node = ASSERT_NIL_P(multi.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that base_interval is set to the first interval when uninitialized.
TEST(TimeModuleTest, BaseIntervalSetToFirstInterval) {
    TestSetup setup("interval", "period", (500 * x::telem::MILLISECOND).nanoseconds());
    Module factory;
    ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    EXPECT_EQ(factory.base_interval(), 500 * x::telem::MILLISECOND);
}

/// @brief Test that base_interval computes GCD across multiple intervals.
TEST(TimeModuleTest, BaseIntervalComputesGCDAcrossNodes) {
    TestSetup setup1("interval", "period", (600 * x::telem::MILLISECOND).nanoseconds());
    TestSetup setup2("wait", "duration", (400 * x::telem::MILLISECOND).nanoseconds());

    Module factory;
    ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup1.ir, setup1.ir.nodes[0], setup1.make_node())
    ));
    ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup2.ir, setup2.ir.nodes[0], setup2.make_node())
    ));
    EXPECT_EQ(factory.base_interval(), 200 * x::telem::MILLISECOND);
}

/// @brief Test that Interval does not fire again before next interval elapses.
TEST(IntervalTest, DoesNotFireBeforeNextIntervalElapses) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    auto ctx2 = make_context(x::telem::MILLISECOND * 500);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Interval fires when the interval is reached.
TEST(IntervalTest, FiresWhenIntervalReached) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 1);
}

/// @brief Test that Interval fires repeatedly at each interval.
TEST(IntervalTest, FiresRepeatedly) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx1 = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);

    auto ctx2 = make_context(x::telem::SECOND * 2);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);

    auto ctx3 = make_context(x::telem::SECOND * 3);
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Interval sets the timestamp to elapsed time when firing.
TEST(IntervalTest, SetsTimestampOnFire) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx = make_context(x::telem::SECOND * 5);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output_time->at<int64_t>(0), (x::telem::SECOND * 5).nanoseconds());
}

/// @brief Test that Interval calls mark_changed when firing.
TEST(IntervalTest, CallsMarkChangedOnFire) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    std::vector<size_t> marked;
    auto ctx = make_context(x::telem::SECOND);
    ctx.mark_changed = [&](size_t i) { marked.push_back(i); };

    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(marked.size(), 1);
    EXPECT_EQ(marked[0], 0);
}

/// @brief Test that Interval does not call mark_changed when not firing.
TEST(IntervalTest, DoesNotCallMarkChangedWhenNotFiring) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx1 = make_context(x::telem::SECOND);
    node.next(ctx1);

    int call_count = 0;
    auto ctx2 = make_context(x::telem::SECOND + x::telem::MILLISECOND * 100);
    ctx2.mark_changed = [&](size_t) { call_count++; };
    node.next(ctx2);

    EXPECT_EQ(call_count, 0);
}

/// @brief Test that Interval is_output_truthy delegates to state.
TEST(IntervalTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx = make_context(x::telem::SECOND);
    node.next(ctx);

    EXPECT_TRUE(node.is_output_truthy(0));
}

/// @brief Test that Interval is_output_truthy returns false before firing.
TEST(IntervalTest, IsOutputTruthyFalseBeforeFiring) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    EXPECT_FALSE(node.is_output_truthy(0));
}

/// @brief Test that Interval is_output_truthy returns false for unknown param.
TEST(IntervalTest, IsOutputTruthyFalseForUnknownParam) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx = make_context(x::telem::SECOND);
    node.next(ctx);

    EXPECT_FALSE(node.is_output_truthy(7));
}

/// @brief Test that Interval reset allows it to fire immediately again.
TEST(IntervalTest, ResetAllowsImmediateFiring) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    node.next(ctx1);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    auto ctx2 = make_context(x::telem::MILLISECOND * 500);
    node.next(ctx2);
    EXPECT_EQ(output->size(), 0);

    node.reset();

    auto ctx3 = make_context(x::telem::MILLISECOND * 600);
    node.next(ctx3);
    EXPECT_EQ(output->size(), 1);
}

TEST(IntervalTest, OnlyFiresOnTimerTick) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    bool changed_called = false;
    runtime::node::Context ctx;
    ctx.elapsed = x::telem::SECOND;
    ctx.tolerance = x::telem::TimeSpan(0);
    ctx.mark_changed = [&changed_called](size_t) { changed_called = true; };
    ctx.mark_self_changed = [] {};
    ctx.set_deadline = [](x::telem::TimeSpan) {};
    ctx.report_error = [](const x::errors::Error &) {};

    ctx.reason = runtime::node::RunReason::TimerTick;
    ASSERT_NIL(node.next(ctx));
    ASSERT_TRUE(changed_called);

    changed_called = false;
    ctx.elapsed = x::telem::SECOND + x::telem::MILLISECOND * 500;
    ctx.reason = runtime::node::RunReason::ChannelInput;
    ASSERT_NIL(node.next(ctx));
    ASSERT_FALSE(changed_called);

    changed_called = false;
    ctx.reason = runtime::node::RunReason::TimerTick;
    ctx.elapsed = x::telem::SECOND * 2;
    ASSERT_NIL(node.next(ctx));
    ASSERT_TRUE(changed_called);
}

/// @brief Test that Wait does not fire before the duration elapses.
TEST(WaitTest, DoesNotFireBeforeDurationElapses) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx = make_context(x::telem::MILLISECOND * 500);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Wait fires once after the duration elapses.
TEST(WaitTest, FiresOnceAfterDuration) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 1);
}

/// @brief Test that Wait does not fire again after the first fire.
TEST(WaitTest, DoesNotFireAgain) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(x::telem::SECOND);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    auto ctx3 = make_context(x::telem::SECOND * 5);
    node.next(ctx3);

    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Wait reset allows it to fire again.
TEST(WaitTest, ResetAllowsFiringAgain) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(x::telem::SECOND);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    node.reset();

    auto ctx3 = make_context(x::telem::SECOND * 2);
    node.next(ctx3);

    auto ctx4 = make_context(x::telem::SECOND * 3);
    node.next(ctx4);

    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 1);
}

TEST(WaitTest, OnlyFiresOnTimerTick) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    bool changed_called = false;
    runtime::node::Context ctx;
    ctx.elapsed = x::telem::TimeSpan(0);
    ctx.tolerance = x::telem::TimeSpan(0);
    ctx.mark_changed = [&changed_called](size_t) { changed_called = true; };
    ctx.mark_self_changed = [] {};
    ctx.set_deadline = [](x::telem::TimeSpan) {};
    ctx.report_error = [](const x::errors::Error &) {};

    ctx.reason = runtime::node::RunReason::TimerTick;
    ASSERT_NIL(node.next(ctx));
    ASSERT_FALSE(changed_called);

    ctx.elapsed = x::telem::MILLISECOND * 500;
    ctx.reason = runtime::node::RunReason::ChannelInput;
    ASSERT_NIL(node.next(ctx));
    ASSERT_FALSE(changed_called);

    ctx.elapsed = x::telem::SECOND;
    ctx.reason = runtime::node::RunReason::ChannelInput;
    ASSERT_NIL(node.next(ctx));
    ASSERT_FALSE(changed_called);

    ctx.reason = runtime::node::RunReason::TimerTick;
    ASSERT_NIL(node.next(ctx));
    ASSERT_TRUE(changed_called);
}

/// @brief Test that Wait measures duration from first next() call, not construction.
TEST(WaitTest, MeasuresDurationFromFirstNextCall) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::SECOND * 10);
    node.next(ctx1);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(x::telem::SECOND * 11);
    node.next(ctx2);

    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait starts timing from a channel input that activates the stage.
/// Regression: wait{duration=3s} took ~6s because startTime was only set on the first
/// TimerTick, not when the stage was activated via channel input.
TEST(WaitTest, StartsTimingFromChannelInputThatActivatesStage) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(
        x::telem::SECOND * 5,
        x::telem::TimeSpan(0),
        runtime::node::RunReason::ChannelInput
    );
    node.next(ctx1);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(x::telem::SECOND * 6);
    node.next(ctx2);

    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait starts timing from channel input after a reset.
/// Regression: after stage re-entry via channel input, startTime was deferred to the
/// next TimerTick, effectively doubling the wait duration.
TEST(WaitTest, StartsTimingFromChannelInputAfterReset) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(x::telem::SECOND);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    node.reset();

    auto ctx3 = make_context(
        x::telem::SECOND * 2,
        x::telem::TimeSpan(0),
        runtime::node::RunReason::ChannelInput
    );
    node.next(ctx3);
    EXPECT_EQ(output->size(), 0);

    auto ctx4 = make_context(x::telem::SECOND * 3);
    node.next(ctx4);

    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait calls mark_self_changed when active but not yet fired.
TEST(WaitTest, CallsMarkSelfChangedWhenActiveButNotFired) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    int self_changed_calls = 0;
    bool changed_called = false;

    // Tick at t=0: starts timer, should call mark_self_changed
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ctx1.mark_self_changed = [&]() { self_changed_calls++; };
    ctx1.mark_changed = [&](size_t) { changed_called = true; };
    ASSERT_NIL(node.next(ctx1));
    EXPECT_EQ(self_changed_calls, 1);
    EXPECT_FALSE(changed_called);

    // Tick at 500ms: still timing, should call mark_self_changed again
    self_changed_calls = 0;
    auto ctx2 = make_context(x::telem::MILLISECOND * 500);
    ctx2.mark_self_changed = [&]() { self_changed_calls++; };
    ctx2.mark_changed = [&](size_t) { changed_called = true; };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(self_changed_calls, 1);
    EXPECT_FALSE(changed_called);

    // Tick at 1s: fires, should NOT call mark_self_changed
    self_changed_calls = 0;
    auto ctx3 = make_context(x::telem::SECOND);
    ctx3.mark_self_changed = [&]() { self_changed_calls++; };
    ctx3.mark_changed = [&](size_t) { changed_called = true; };
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(self_changed_calls, 0);
    EXPECT_TRUE(changed_called);
}

/// @brief Test that Wait calls mark_self_changed on channel input to survive
/// non-tick cycles without being starved.
TEST(WaitTest, CallsMarkSelfChangedOnChannelInputToSurvive) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    int self_changed_calls = 0;
    bool changed_called = false;

    // Tick at t=0: starts timer
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ctx1.mark_self_changed = [&]() { self_changed_calls++; };
    ctx1.mark_changed = [&](size_t) { changed_called = true; };
    ASSERT_NIL(node.next(ctx1));
    EXPECT_EQ(self_changed_calls, 1);
    EXPECT_FALSE(changed_called);

    // Channel input at 200ms: duration not elapsed, should call mark_self_changed
    self_changed_calls = 0;
    auto ctx2 = make_context(
        x::telem::MILLISECOND * 200,
        x::telem::TimeSpan(0),
        runtime::node::RunReason::ChannelInput
    );
    ctx2.mark_self_changed = [&]() { self_changed_calls++; };
    ctx2.mark_changed = [&](size_t) { changed_called = true; };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(self_changed_calls, 1);
    EXPECT_FALSE(changed_called);

    // Timer tick at 1s: should fire normally
    self_changed_calls = 0;
    auto ctx3 = make_context(x::telem::SECOND);
    ctx3.mark_self_changed = [&]() { self_changed_calls++; };
    ctx3.mark_changed = [&](size_t) { changed_called = true; };
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(self_changed_calls, 0);
    EXPECT_TRUE(changed_called);
}

/// @brief Test that Wait sets the timestamp to elapsed time when firing.
TEST(WaitTest, SetsTimestampOnFire) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::SECOND * 2);
    node.next(ctx1);

    auto ctx2 = make_context(x::telem::SECOND * 3);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output_time->at<int64_t>(0), (x::telem::SECOND * 3).nanoseconds());
}

/// @brief Test that Wait calls mark_changed when firing.
TEST(WaitTest, CallsMarkChangedOnFire) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    node.next(ctx1);

    std::vector<size_t> marked;
    auto ctx2 = make_context(x::telem::SECOND);
    ctx2.mark_changed = [&](size_t i) { marked.push_back(i); };

    node.next(ctx2);
    ASSERT_EQ(marked.size(), 1);
    EXPECT_EQ(marked[0], 0);
}

/// @brief Test that Wait does not call mark_changed when not firing.
TEST(WaitTest, DoesNotCallMarkChangedWhenNotFiring) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    int call_count = 0;
    auto ctx = make_context(x::telem::MILLISECOND * 100);
    ctx.mark_changed = [&](size_t) { call_count++; };
    node.next(ctx);

    EXPECT_EQ(call_count, 0);
}

/// @brief Test that Wait is_output_truthy delegates to state.
TEST(WaitTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(x::telem::SECOND);
    node.next(ctx2);

    EXPECT_TRUE(node.is_output_truthy(0));
}

/// @brief Test that Wait reset restarts timing from zero.
TEST(WaitTest, ResetRestartsTimingFromZero) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::SECOND * 5);
    node.next(ctx1);

    node.reset();

    auto ctx2 = make_context(x::telem::SECOND * 5 + x::telem::MILLISECOND * 500);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx3 = make_context(x::telem::SECOND * 6 + x::telem::MILLISECOND * 500);
    node.next(ctx3);

    EXPECT_EQ(output->size(), 1);
}

/// @brief Test calculate_tolerance for RT_EVENT mode.
TEST(CalculateToleranceTest, RTEventMode) {
    const auto tolerance = calculate_tolerance(
        runtime::loop::ExecutionMode::RT_EVENT,
        100 * x::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 100 * x::telem::MICROSECOND);
}

/// @brief Test calculate_tolerance for BUSY_WAIT mode.
TEST(CalculateToleranceTest, BusyWaitMode) {
    const auto tolerance = calculate_tolerance(
        runtime::loop::ExecutionMode::BUSY_WAIT,
        100 * x::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 100 * x::telem::MICROSECOND);
}

/// @brief Test calculate_tolerance for HIGH_RATE mode.
TEST(CalculateToleranceTest, HighRateMode) {
    const auto tolerance = calculate_tolerance(
        runtime::loop::ExecutionMode::HIGH_RATE,
        100 * x::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, x::telem::MILLISECOND);
}

/// @brief Test calculate_tolerance for EVENT_DRIVEN mode.
TEST(CalculateToleranceTest, EventDrivenMode) {
    const auto tolerance = calculate_tolerance(
        runtime::loop::ExecutionMode::EVENT_DRIVEN,
        100 * x::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 5 * x::telem::MILLISECOND);
}

/// @brief Test calculate_tolerance for HYBRID mode.
TEST(CalculateToleranceTest, HybridMode) {
    const auto tolerance = calculate_tolerance(
        runtime::loop::ExecutionMode::HYBRID,
        100 * x::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 5 * x::telem::MILLISECOND);
}

/// @brief Test calculate_tolerance with max interval returns fixed 5ms.
TEST(CalculateToleranceTest, MaxInterval) {
    const auto tolerance = calculate_tolerance(
        runtime::loop::ExecutionMode::EVENT_DRIVEN,
        x::telem::TimeSpan::max()
    );
    EXPECT_EQ(tolerance, 5 * x::telem::MILLISECOND);
}

/// @brief Test calculate_tolerance respects half-interval minimum.
TEST(CalculateToleranceTest, HalfIntervalMinimum) {
    const auto tolerance = calculate_tolerance(
        runtime::loop::ExecutionMode::EVENT_DRIVEN,
        4 * x::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 2 * x::telem::MILLISECOND);
}

/// @brief Test that Interval fires within tolerance.
TEST(IntervalToleranceTest, FiresWithinTolerance) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    output->resize(0);

    auto ctx2 = make_context(
        x::telem::SECOND * 2 - x::telem::MILLISECOND * 5,
        50 * x::telem::MILLISECOND
    );
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Interval does not fire too early even with tolerance.
TEST(IntervalToleranceTest, DoesNotFireTooEarly) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    output->resize(0);

    auto ctx2 = make_context(x::telem::MILLISECOND * 900, 50 * x::telem::MILLISECOND);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Wait fires within tolerance.
TEST(WaitToleranceTest, FiresWithinTolerance) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(
        x::telem::SECOND - x::telem::MILLISECOND * 5,
        50 * x::telem::MILLISECOND
    );
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait does not fire too early even with tolerance.
TEST(WaitToleranceTest, DoesNotFireTooEarly) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(x::telem::MILLISECOND * 900, 50 * x::telem::MILLISECOND);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Interval fires correctly with zero tolerance (original behavior).
TEST(IntervalToleranceTest, ZeroToleranceRequiresExactTime) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(IntervalInputs::create(setup.ir.nodes[0].inputs));
    Interval node(setup.make_node(), inputs.interval);

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    output->resize(0);

    auto ctx2 = make_context(
        x::telem::SECOND - x::telem::NANOSECOND,
        x::telem::TimeSpan(0)
    );
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);

    auto ctx3 = make_context(x::telem::SECOND, x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait fires correctly with zero tolerance (original behavior).
TEST(WaitToleranceTest, ZeroToleranceRequiresExactTime) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0), x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(
        x::telem::SECOND - x::telem::NANOSECOND,
        x::telem::TimeSpan(0)
    );
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);

    auto ctx3 = make_context(x::telem::SECOND, x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that tolerance is capped at half the interval for small intervals.
TEST(CalculateToleranceTest, SmallIntervalCapsAtHalf) {
    const auto tolerance = calculate_tolerance(
        runtime::loop::ExecutionMode::RT_EVENT,
        100 * x::telem::MICROSECOND
    );
    EXPECT_EQ(tolerance, 50 * x::telem::MICROSECOND);
}

/// @brief Test calculate_tolerance for AUTO mode (defaults to EVENT_DRIVEN behavior).
TEST(CalculateToleranceTest, AutoMode) {
    const auto tolerance = calculate_tolerance(
        runtime::loop::ExecutionMode::AUTO,
        100 * x::telem::MILLISECOND
    );
    EXPECT_EQ(tolerance, 5 * x::telem::MILLISECOND);
}

TEST(IntervalDeadlineTest, SetsDeadlineToLastFiredPlusPeriod) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(
        time::IntervalInputs::create(setup.ir.nodes[0].inputs)
    );
    time::Interval node(setup.make_node(), inputs.interval);

    x::telem::TimeSpan reported_deadline(-1);
    auto ctx = make_context(x::telem::TimeSpan(0));
    ctx.set_deadline = [&](x::telem::TimeSpan d) { reported_deadline = d; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(reported_deadline, x::telem::SECOND);
}

TEST(IntervalDeadlineTest, SetsDeadlineOnNonTimerTick) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(
        time::IntervalInputs::create(setup.ir.nodes[0].inputs)
    );
    time::Interval node(setup.make_node(), inputs.interval);

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    x::telem::TimeSpan reported_deadline(-1);
    auto ctx2 = make_context(
        x::telem::MILLISECOND * 500,
        x::telem::TimeSpan(0),
        runtime::node::RunReason::ChannelInput
    );
    ctx2.set_deadline = [&](x::telem::TimeSpan d) { reported_deadline = d; };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(reported_deadline, x::telem::SECOND);
}

TEST(IntervalDeadlineTest, SetsDeadlineAfterFiring) {
    TestSetup setup("interval", "period", x::telem::SECOND.nanoseconds());
    const auto inputs = ASSERT_NIL_P(
        time::IntervalInputs::create(setup.ir.nodes[0].inputs)
    );
    time::Interval node(setup.make_node(), inputs.interval);

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    x::telem::TimeSpan reported_deadline(-1);
    auto ctx2 = make_context(x::telem::SECOND);
    ctx2.set_deadline = [&](x::telem::TimeSpan d) { reported_deadline = d; };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(reported_deadline, x::telem::SECOND * 2);
}

TEST(WaitDeadlineTest, SetsDeadlineToStartTimePlusDuration) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    time::Wait node(setup.make_node());

    x::telem::TimeSpan reported_deadline(-1);
    auto ctx = make_context(x::telem::SECOND * 5);
    ctx.set_deadline = [&](x::telem::TimeSpan d) { reported_deadline = d; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(reported_deadline, x::telem::SECOND * 6);
}

TEST(WaitDeadlineTest, SetsDeadlineOnChannelInput) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    time::Wait node(setup.make_node());

    x::telem::TimeSpan reported_deadline(-1);
    auto ctx = make_context(
        x::telem::SECOND * 2,
        x::telem::TimeSpan(0),
        runtime::node::RunReason::ChannelInput
    );
    ctx.set_deadline = [&](x::telem::TimeSpan d) { reported_deadline = d; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(reported_deadline, x::telem::SECOND * 3);
}

TEST(WaitDeadlineTest, DoesNotSetDeadlineAfterFiring) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    time::Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto ctx2 = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx2));

    x::telem::TimeSpan reported_deadline(-1);
    auto ctx3 = make_context(x::telem::SECOND * 5);
    ctx3.set_deadline = [&](x::telem::TimeSpan d) { reported_deadline = d; };
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(reported_deadline, x::telem::TimeSpan(-1));
}

TEST(WaitDeadlineTest, SetsCorrectDeadlineAfterReset) {
    TestSetup setup("wait", "duration", x::telem::SECOND.nanoseconds());
    time::Wait node(setup.make_node());

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));
    auto ctx2 = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx2));

    node.reset();

    x::telem::TimeSpan reported_deadline(-1);
    auto ctx3 = make_context(x::telem::SECOND * 10);
    ctx3.set_deadline = [&](x::telem::TimeSpan d) { reported_deadline = d; };
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(reported_deadline, x::telem::SECOND * 11);
}

/// @brief Helper to build IR for Now.
struct NowTestSetup {
    ir::IR ir;
    runtime::state::State state;
    x::telem::MonoClock clock;

    NowTestSetup():
        ir(build_ir()),
        state(
            runtime::state::Config{.ir = ir, .channels = {}},
            runtime::errors::noop_handler
        ) {}

    runtime::state::Node make_node() { return ASSERT_NIL_P(state.node("now_node")); }

private:
    static ir::IR build_ir() {
        arc::types::Param output_param;
        output_param.name = "output";
        output_param.type = arc::types::Type{.kind = arc::types::Kind::I64};

        ir::Node ir_node;
        ir_node.key = "now_node";
        ir_node.type = "now";
        ir_node.outputs.push_back(output_param);

        ir::Function fn;
        fn.key = "test";

        ir::IR ir;
        ir.nodes.push_back(ir_node);
        ir.functions.push_back(fn);
        return ir;
    }
};

/// @brief Now node outputs a valid wall-clock timestamp.
TEST(NowTest, OutputsWallClockTimestamp) {
    NowTestSetup setup;
    const auto inputs = ASSERT_NIL_P(time::NowInputs::create(setup.ir.nodes[0].inputs));
    time::Now node(inputs, setup.make_node(), &setup.clock);

    const auto before = x::telem::TimeStamp::now().nanoseconds();
    auto ctx = make_context(x::telem::SECOND * 5);
    bool changed = false;
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    const auto after = x::telem::TimeStamp::now().nanoseconds();

    EXPECT_TRUE(changed);
    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    const auto ts = output->at<int64_t>(0);
    EXPECT_GE(ts, before);
    EXPECT_LE(ts, after);
}

/// @brief Now node fires on any RunReason (not just TimerTick).
TEST(NowTest, FiresOnChannelInput) {
    NowTestSetup setup;
    const auto inputs = ASSERT_NIL_P(time::NowInputs::create(setup.ir.nodes[0].inputs));
    time::Now node(inputs, setup.make_node(), &setup.clock);

    bool changed = false;
    auto ctx = make_context(
        x::telem::TimeSpan(0),
        x::telem::TimeSpan(0),
        runtime::node::RunReason::ChannelInput
    );
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
}

/// @brief Now node output and output_time contain the same timestamp.
TEST(NowTest, OutputAndOutputTimeMatch) {
    NowTestSetup setup;
    const auto inputs = ASSERT_NIL_P(time::NowInputs::create(setup.ir.nodes[0].inputs));
    time::Now node(inputs, setup.make_node(), &setup.clock);

    auto ctx = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output->at<int64_t>(0), output_time->at<int64_t>(0));
}

/// @brief Now node works correctly after reset.
TEST(NowTest, WorksAfterReset) {
    NowTestSetup setup;
    const auto inputs = ASSERT_NIL_P(time::NowInputs::create(setup.ir.nodes[0].inputs));
    time::Now node(inputs, setup.make_node(), &setup.clock);

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    node.reset();

    bool changed = false;
    auto ctx2 = make_context(x::telem::SECOND);
    ctx2.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_TRUE(changed);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
}

/// @brief Now node is_output_truthy returns false for unknown param.
TEST(NowTest, IsOutputTruthyFalseForUnknownParam) {
    NowTestSetup setup;
    const auto inputs = ASSERT_NIL_P(time::NowInputs::create(setup.ir.nodes[0].inputs));
    time::Now node(inputs, setup.make_node(), &setup.clock);
    EXPECT_FALSE(node.is_output_truthy(999));
}

/// @brief Now node does not affect base_interval.
TEST(TimeModuleTest, NowDoesNotAffectBaseInterval) {
    NowTestSetup setup;
    Module factory;
    ASSERT_NIL_P(factory.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    EXPECT_EQ(factory.base_interval(), UNSET_BASE_INTERVAL);
}

struct TickResult {
    bool fired = false;
    x::telem::TimeSpan deadline{0};
};

/// @brief builds a config whose span input is var-bound: value holds the declared
/// initial and set writes the variable's live slot. The IR and state outlive the node,
/// so a VarConfig must stay put for the test's duration.
class VarConfig {
    ir::IR prog;
    runtime::state::State state;

public:
    Module factory;
    std::unique_ptr<runtime::node::Node> node;

    VarConfig(
        const std::string &node_type,
        const std::string &param,
        const x::telem::TimeSpan initial
    ):
        prog(build_ir(node_type, param, initial)),
        state(
            runtime::state::Config{.ir = prog, .channels = {}},
            runtime::errors::noop_handler
        ) {
        this->node = ASSERT_NIL_P(this->factory.create(
            runtime::node::Config(
                this->prog,
                this->prog.nodes[1],
                ASSERT_NIL_P(this->state.node("n"))
            )
        ));
    }

    VarConfig(const VarConfig &) = delete;
    VarConfig &operator=(const VarConfig &) = delete;

    void set(const x::telem::TimeSpan span) {
        auto v = ASSERT_NIL_P(this->state.node("v"));
        *v.output(0) = x::telem::Series(span.nanoseconds());
    }

    TickResult tick(
        const x::telem::TimeSpan elapsed,
        const runtime::node::RunReason reason
    ) const {
        TickResult r;
        auto ctx = make_context(elapsed, x::telem::TimeSpan(0), reason);
        ctx.mark_changed = [&r](size_t) { r.fired = true; };
        ctx.set_deadline = [&r](const x::telem::TimeSpan d) { r.deadline = d; };
        EXPECT_FALSE(this->node->next(ctx));
        return r;
    }

private:
    static ir::IR build_ir(
        const std::string &node_type,
        const std::string &param,
        const x::telem::TimeSpan initial
    ) {
        types::Param var_out;
        var_out.name = ir::default_output_param;
        var_out.type = types::Type{.kind = types::Kind::I64};
        ir::Node v;
        v.key = "v";
        v.type = "variable";
        v.outputs.push_back(var_out);

        types::Param span;
        span.name = param;
        span.type = types::Type{
            .kind = types::Kind::VarRef,
            .name = "v",
            .elem = x::mem::indirect<types::Type>(types::Type{.kind = types::Kind::I64})
        };
        span.value = initial.nanoseconds();
        types::Param out;
        out.name = ir::default_output_param;
        out.type = types::Type{.kind = types::Kind::U8};
        ir::Node n;
        n.key = "n";
        n.type = node_type;
        n.inputs.push_back(span);
        n.outputs.push_back(out);

        ir::IR ir;
        ir.nodes.push_back(v);
        ir.nodes.push_back(n);
        return ir;
    }
};

TEST(IntervalVarTest, HonorsTheDeclaredInitialBeforeAnyWrite) {
    const VarConfig t("interval", "period", x::telem::SECOND);
    EXPECT_TRUE(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).fired
    );
    EXPECT_FALSE(
        t.tick(500 * x::telem::MILLISECOND, runtime::node::RunReason::TimerTick).fired
    );
    EXPECT_TRUE(t.tick(x::telem::SECOND, runtime::node::RunReason::TimerTick).fired);
}

TEST(IntervalVarTest, AdoptsAShortenedPeriodAtTheNextEvaluation) {
    VarConfig t("interval", "period", x::telem::SECOND);
    EXPECT_TRUE(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).fired
    );
    t.set(100 * x::telem::MILLISECOND);
    EXPECT_TRUE(
        t.tick(100 * x::telem::MILLISECOND, runtime::node::RunReason::TimerTick).fired
    );
}

TEST(IntervalVarTest, AdoptsALengthenedPeriodWithoutFiringEarly) {
    VarConfig t("interval", "period", 100 * x::telem::MILLISECOND);
    EXPECT_TRUE(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).fired
    );
    t.set(x::telem::SECOND);
    EXPECT_FALSE(
        t.tick(100 * x::telem::MILLISECOND, runtime::node::RunReason::TimerTick).fired
    );
    EXPECT_TRUE(t.tick(x::telem::SECOND, runtime::node::RunReason::TimerTick).fired);
}

TEST(IntervalVarTest, ReportsTheDeadlineFromTheLivePeriod) {
    VarConfig t("interval", "period", x::telem::SECOND);
    EXPECT_EQ(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).deadline,
        x::telem::SECOND
    );
    t.set(2 * x::telem::SECOND);
    const auto r = t.tick(
        500 * x::telem::MILLISECOND,
        runtime::node::RunReason::ChannelInput
    );
    EXPECT_FALSE(r.fired);
    EXPECT_EQ(r.deadline, 2 * x::telem::SECOND);
}

TEST(IntervalVarTest, FiresImmediatelyAfterResetUsingTheLivePeriod) {
    VarConfig t("interval", "period", x::telem::SECOND);
    EXPECT_TRUE(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).fired
    );
    EXPECT_TRUE(t.tick(x::telem::SECOND, runtime::node::RunReason::TimerTick).fired);
    t.set(5 * x::telem::SECOND);
    t.node->reset();
    EXPECT_TRUE(
        t.tick(1500 * x::telem::MILLISECOND, runtime::node::RunReason::TimerTick).fired
    );
}

TEST(IntervalVarTest, SeedsTheTimingBaseFromTheDeclaredValueOnly) {
    VarConfig t("interval", "period", 100 * x::telem::MILLISECOND);
    EXPECT_EQ(t.factory.base_interval(), 100 * x::telem::MILLISECOND);
    t.set(x::telem::MILLISECOND);
    EXPECT_TRUE(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).fired
    );
    EXPECT_EQ(t.factory.base_interval(), 100 * x::telem::MILLISECOND);
}

TEST(WaitVarTest, HonorsTheDeclaredInitialBeforeAnyWrite) {
    const VarConfig t("wait", "duration", x::telem::SECOND);
    EXPECT_FALSE(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).fired
    );
    EXPECT_FALSE(
        t.tick(500 * x::telem::MILLISECOND, runtime::node::RunReason::TimerTick).fired
    );
    EXPECT_TRUE(t.tick(x::telem::SECOND, runtime::node::RunReason::TimerTick).fired);
}

TEST(WaitVarTest, FiresEarlierWhenTheDurationIsShortenedMidWait) {
    VarConfig t("wait", "duration", 10 * x::telem::SECOND);
    EXPECT_FALSE(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).fired
    );
    t.set(x::telem::SECOND);
    EXPECT_TRUE(t.tick(x::telem::SECOND, runtime::node::RunReason::TimerTick).fired);
}

TEST(WaitVarTest, FiresLaterWhenTheDurationIsLengthenedMidWait) {
    VarConfig t("wait", "duration", x::telem::SECOND);
    EXPECT_FALSE(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).fired
    );
    t.set(5 * x::telem::SECOND);
    EXPECT_FALSE(t.tick(x::telem::SECOND, runtime::node::RunReason::TimerTick).fired);
    EXPECT_TRUE(
        t.tick(5 * x::telem::SECOND, runtime::node::RunReason::TimerTick).fired
    );
}

TEST(WaitVarTest, ReportsTheDeadlineFromTheLiveDuration) {
    VarConfig t("wait", "duration", x::telem::SECOND);
    EXPECT_EQ(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).deadline,
        x::telem::SECOND
    );
    t.set(3 * x::telem::SECOND);
    const auto r = t.tick(
        500 * x::telem::MILLISECOND,
        runtime::node::RunReason::ChannelInput
    );
    EXPECT_FALSE(r.fired);
    EXPECT_EQ(r.deadline, 3 * x::telem::SECOND);
}

TEST(WaitVarTest, StaysOneShotAfterAShorteningWrite) {
    VarConfig t("wait", "duration", x::telem::SECOND);
    EXPECT_FALSE(
        t.tick(x::telem::TimeSpan(0), runtime::node::RunReason::TimerTick).fired
    );
    EXPECT_TRUE(t.tick(x::telem::SECOND, runtime::node::RunReason::TimerTick).fired);
    t.set(100 * x::telem::MILLISECOND);
    EXPECT_FALSE(
        t.tick(2 * x::telem::SECOND, runtime::node::RunReason::TimerTick).fired
    );
}

/// @brief replaces every occurrence of from in s with to.
std::string replace_all(std::string s, const std::string &from, const std::string &to) {
    for (size_t at = s.find(from); at != std::string::npos; at = s.find(from, at))
        s.replace(at, from.size(), to);
    return s;
}

/// @brief compiles source and creates every timer node through a fresh time Host,
/// returning the resulting base_interval. The %a% and %b% placeholders stand in for
/// the channels the sources write to.
x::telem::TimeSpan
compile_base(const synnax::Synnax &client, const std::string &source) {
    const auto a = ASSERT_NIL_P(
        client.channels.create(make_unique_channel_name("a"), x::telem::UINT8_T, true)
    );
    const auto b = ASSERT_NIL_P(
        client.channels.create(make_unique_channel_name("b"), x::telem::UINT8_T, true)
    );
    const auto prog = runtime::testutil::compile_text(
        client,
        replace_all(replace_all("import time\n" + source, "%a%", a.name), "%b%", b.name)
    );
    auto factory = std::make_shared<Module>();
    runtime::state::State s(
        runtime::state::Config{.ir = static_cast<const ir::IR &>(prog), .channels = {}},
        runtime::errors::noop_handler
    );
    runtime::node::MultiFactory multi({factory});
    for (const auto &n: prog.nodes) {
        auto [node, err] = multi.create(
            runtime::node::Config(prog, n, ASSERT_NIL_P(s.node(n.key)))
        );
        if (err && !err.matches(x::errors::NOT_FOUND))
            ADD_FAILURE() << "create " << n.key << ": " << err.message();
    }
    return factory->base_interval();
}

struct GcdCase {
    std::string name;
    std::string source;
    int64_t expected_ms;
};

class TimingBaseGcdTest : public testing::TestWithParam<GcdCase> {};

TEST_P(TimingBaseGcdTest, ComputesTheGcdOverDeclaredAndLiteralReassignedSpans) {
    const auto client = new_test_client();
    EXPECT_EQ(
        compile_base(client, GetParam().source),
        GetParam().expected_ms * x::telem::MILLISECOND
    );
}

INSTANTIATE_TEST_SUITE_P(
    Sources,
    TimingBaseGcdTest,
    testing::Values(
        GcdCase{
            "two_literal_intervals",
            R"(
time.interval{period=100ms} -> %a%
time.interval{period=60ms} -> %b%
)",
            20
        },
        GcdCase{
            "two_intervals_fed_by_vars_never_reassigned",
            R"(
sequence main {
    p := i64 ns(100ms)
    q := i64 ns(60ms)
    stage run {
        time.interval{period=p} -> %a%
        time.interval{period=q} -> %b%
    }
}
)",
            20
        },
        GcdCase{
            "two_intervals_fed_by_vars_each_reassigned_with_a_literal",
            R"(
sequence main {
    p := i64 ns(100ms)
    q := i64 ns(60ms)
    stage run {
        time.interval{period=p} -> %a%
        time.interval{period=q} -> %b%
        1 => faster
    }
    stage faster {
        p = i64 ns(10ms)
        q = i64 ns(45ms)
    }
}
)",
            5
        },
        GcdCase{
            "two_intervals_fed_by_vars_expression_reassignments_excluded",
            R"(
sequence main {
    p := i64 ns(100ms)
    q := i64 ns(60ms)
    stage run {
        time.interval{period=p} -> %a%
        time.interval{period=q} -> %b%
        1 => faster
    }
    stage faster {
        p = i64 ns(2 * 25ms)
        q = i64 ns(3 * 20ms)
    }
}
)",
            20
        },
        GcdCase{
            "two_literal_waits",
            R"(
time.wait{duration=75ms} -> %a%
time.wait{duration=50ms} -> %b%
)",
            25
        },
        GcdCase{
            "two_waits_fed_by_vars_each_reassigned_with_a_literal",
            R"(
sequence main {
    d := i64 ns(80ms)
    e := i64 ns(50ms)
    stage run {
        time.wait{duration=d} -> %a%
        time.wait{duration=e} -> %b%
        1 => faster
    }
    stage faster {
        d = i64 ns(30ms)
        e = i64 ns(35ms)
    }
}
)",
            5
        },
        GcdCase{
            "interval_and_wait_fed_by_vars_never_reassigned",
            R"(
sequence main {
    p := i64 ns(100ms)
    d := i64 ns(75ms)
    stage run {
        time.interval{period=p} -> %a%
        time.wait{duration=d} -> %b%
    }
}
)",
            25
        },
        GcdCase{
            "interval_and_wait_fed_by_vars_each_reassigned_with_a_literal",
            R"(
sequence main {
    p := i64 ns(100ms)
    d := i64 ns(80ms)
    stage run {
        time.interval{period=p} -> %a%
        time.wait{duration=d} -> %b%
        1 => faster
    }
    stage faster {
        p = i64 ns(60ms)
        d = i64 ns(30ms)
    }
}
)",
            10
        },
        GcdCase{
            "interval_and_wait_fed_by_vars_expression_reassignments_excluded",
            R"(
sequence main {
    p := i64 ns(100ms)
    d := i64 ns(75ms)
    stage run {
        time.interval{period=p} -> %a%
        time.wait{duration=d} -> %b%
        1 => faster
    }
    stage faster {
        p = i64 ns(2 * 25ms)
        d = i64 ns(3 * 15ms)
    }
}
)",
            25
        },
        GcdCase{
            "literal_interval_and_reassigned_var_wait",
            R"(
sequence main {
    d := i64 ns(60ms)
    stage run {
        time.interval{period=100ms} -> %a%
        time.wait{duration=d} -> %b%
        1 => faster
    }
    stage faster {
        d = i64 ns(45ms)
    }
}
)",
            5
        },
        GcdCase{
            "var_interval_two_reassignment_sites",
            R"(
sequence main {
    p := i64 ns(100ms)
    stage run {
        time.interval{period=p} -> %a%
        1 => mid
    }
    stage mid {
        p = i64 ns(50ms)
        1 => fast
    }
    stage fast {
        p = i64 ns(30ms)
    }
}
)",
            10
        },
        GcdCase{
            "same_var_feeding_both_timer_kinds",
            R"(
sequence main {
    p := i64 ns(40ms)
    stage run {
        time.interval{period=p} -> %a%
        time.wait{duration=p} -> %b%
        1 => faster
    }
    stage faster {
        p = i64 ns(30ms)
    }
}
)",
            10
        },
        GcdCase{
            "reassignment_in_an_unreached_stage_still_counts",
            R"(
sequence main {
    p := i64 ns(100ms)
    stage run {
        time.interval{period=p} -> %a%
    }
    stage never {
        p = i64 ns(30ms)
    }
}
)",
            10
        }
    ),
    [](const testing::TestParamInfo<GcdCase> &info) { return info.param.name; }
);
}
