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
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/time/time.h"

using namespace arc::runtime;

namespace {
node::Context make_context(const telem::TimeSpan elapsed) {
    return node::Context{
        .elapsed = elapsed,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const xerrors::Error &) {},
        .activate_stage = [] {},
    };
}

struct TestSetup {
    arc::ir::IR ir;
    state::State state;

    TestSetup(const std::string &type, const std::string &param_name, const int64_t ns):
        ir(build_ir(type, param_name, ns)),
        state(
            state::Config{.ir = ir, .channels = {}},
            arc::runtime::errors::noop_handler
        ) {}

    state::Node make_node() { return ASSERT_NIL_P(state.node("timer")); }

private:
    static arc::ir::IR
    build_ir(const std::string &type, const std::string &param_name, const int64_t ns) {
        arc::ir::Param output_param;
        output_param.name = "output";
        output_param.type = arc::types::Type(arc::types::Kind::U8);

        arc::ir::Param cfg_param;
        cfg_param.name = param_name;
        cfg_param.type = arc::types::Type(arc::types::Kind::I64);
        cfg_param.value = ns;

        arc::ir::Node ir_node;
        ir_node.key = "timer";
        ir_node.type = type;
        ir_node.outputs.params.push_back(output_param);
        ir_node.config.params.push_back(cfg_param);

        arc::ir::Function fn;
        fn.key = "test";

        arc::ir::IR ir;
        ir.nodes.push_back(ir_node);
        ir.functions.push_back(fn);
        return ir;
    }
};
}

/// @brief Test that factory returns NOT_FOUND for non-time node types.
TEST(TimeFactoryTest, ReturnsNotFoundForWrongType) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    auto ir_node = setup.ir.nodes[0];
    ir_node.type = "not_a_time_node";

    time::Factory factory;
    ASSERT_OCCURRED_AS_P(
        factory.create(node::Config(setup.ir, ir_node, setup.make_node())),
        xerrors::NOT_FOUND
    );
}

/// @brief Test that factory creates an Interval node from valid configuration.
TEST(TimeFactoryTest, CreatesIntervalNode) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    time::Factory factory;
    const auto node = ASSERT_NIL_P(
        factory.create(node::Config(setup.ir, setup.ir.nodes[0], setup.make_node()))
    );
    ASSERT_NE(node, nullptr);
}

/// @brief Test that factory creates a Wait node from valid configuration.
TEST(TimeFactoryTest, CreatesWaitNode) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    time::Factory factory;
    const auto node = ASSERT_NIL_P(
        factory.create(node::Config(setup.ir, setup.ir.nodes[0], setup.make_node()))
    );
    ASSERT_NE(node, nullptr);
}

/// @brief Test that timing_base is set to the first interval when uninitialized.
TEST(TimeFactoryTest, TimingBaseSetToFirstInterval) {
    TestSetup setup("interval", "period", (500 * telem::MILLISECOND).nanoseconds());
    time::Factory factory;
    ASSERT_NIL_P(
        factory.create(node::Config(setup.ir, setup.ir.nodes[0], setup.make_node()))
    );
    EXPECT_EQ(factory.timing_base, 500 * telem::MILLISECOND);
}

/// @brief Test that timing_base computes GCD across multiple intervals.
TEST(TimeFactoryTest, TimingBaseComputesGCDAcrossNodes) {
    TestSetup setup1("interval", "period", (600 * telem::MILLISECOND).nanoseconds());
    TestSetup setup2("wait", "duration", (400 * telem::MILLISECOND).nanoseconds());

    time::Factory factory;
    ASSERT_NIL_P(
        factory.create(node::Config(setup1.ir, setup1.ir.nodes[0], setup1.make_node()))
    );
    ASSERT_NIL_P(
        factory.create(node::Config(setup2.ir, setup2.ir.nodes[0], setup2.make_node()))
    );
    EXPECT_EQ(factory.timing_base, 200 * telem::MILLISECOND);
}

/// @brief Test that Interval does not fire again before next interval elapses.
TEST(IntervalTest, DoesNotFireBeforeNextIntervalElapses) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    auto ctx2 = make_context(telem::MILLISECOND * 500);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Interval fires when the interval is reached.
TEST(IntervalTest, FiresWhenIntervalReached) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx = make_context(telem::SECOND);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 1);
}

/// @brief Test that Interval fires repeatedly at each interval.
TEST(IntervalTest, FiresRepeatedly) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::SECOND);
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);

    auto ctx2 = make_context(telem::SECOND * 2);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);

    auto ctx3 = make_context(telem::SECOND * 3);
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Interval sets the timestamp to elapsed time when firing.
TEST(IntervalTest, SetsTimestampOnFire) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx = make_context(telem::SECOND * 5);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output_time->at<int64_t>(0), (telem::SECOND * 5).nanoseconds());
}

/// @brief Test that Interval calls mark_changed when firing.
TEST(IntervalTest, CallsMarkChangedOnFire) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    bool changed_called = false;
    std::string changed_param;
    auto ctx = make_context(telem::SECOND);
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
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::SECOND);
    node.next(ctx1);

    int call_count = 0;
    auto ctx2 = make_context(telem::SECOND + telem::MILLISECOND * 100);
    ctx2.mark_changed = [&](const std::string &) { call_count++; };
    node.next(ctx2);

    EXPECT_EQ(call_count, 0);
}

/// @brief Test that Interval is_output_truthy delegates to state.
TEST(IntervalTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx = make_context(telem::SECOND);
    node.next(ctx);

    EXPECT_TRUE(node.is_output_truthy("output"));
}

/// @brief Test that Interval is_output_truthy returns false before firing.
TEST(IntervalTest, IsOutputTruthyFalseBeforeFiring) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    // Before any next() call, output should be falsy (empty series)
    EXPECT_FALSE(node.is_output_truthy("output"));
}

/// @brief Test that Interval is_output_truthy returns false for unknown param.
TEST(IntervalTest, IsOutputTruthyFalseForUnknownParam) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx = make_context(telem::SECOND);
    node.next(ctx);

    // Unknown parameter should return false
    EXPECT_FALSE(node.is_output_truthy("nonexistent"));
}

/// @brief Test that Interval reset allows it to fire immediately again.
TEST(IntervalTest, ResetAllowsImmediateFiring) {
    TestSetup setup("interval", "period", telem::SECOND.nanoseconds());
    const time::IntervalConfig cfg(setup.ir.nodes[0].config);
    time::Interval node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::TimeSpan(0));
    node.next(ctx1);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    auto ctx2 = make_context(telem::MILLISECOND * 500);
    node.next(ctx2);
    EXPECT_EQ(output->size(), 0);

    node.reset();

    auto ctx3 = make_context(telem::MILLISECOND * 600);
    node.next(ctx3);
    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait does not fire before the duration elapses.
TEST(WaitTest, DoesNotFireBeforeDurationElapses) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx = make_context(telem::MILLISECOND * 500);
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Wait fires once after the duration elapses.
TEST(WaitTest, FiresOnceAfterDuration) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(telem::SECOND);
    ASSERT_NIL(node.next(ctx2));
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 1);
}

/// @brief Test that Wait does not fire again after the first fire.
TEST(WaitTest, DoesNotFireAgain) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(telem::SECOND);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    auto ctx3 = make_context(telem::SECOND * 5);
    node.next(ctx3);

    EXPECT_EQ(output->size(), 0);
}

/// @brief Test that Wait reset allows it to fire again.
TEST(WaitTest, ResetAllowsFiringAgain) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(telem::SECOND);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    output->resize(0);

    node.reset();

    auto ctx3 = make_context(telem::SECOND * 2);
    node.next(ctx3);

    auto ctx4 = make_context(telem::SECOND * 3);
    node.next(ctx4);

    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 1);
}

/// @brief Test that Wait measures duration from first next() call, not construction.
TEST(WaitTest, MeasuresDurationFromFirstNextCall) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::SECOND * 10);
    node.next(ctx1);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx2 = make_context(telem::SECOND * 11);
    node.next(ctx2);

    EXPECT_EQ(output->size(), 1);
}

/// @brief Test that Wait sets the timestamp to elapsed time when firing.
TEST(WaitTest, SetsTimestampOnFire) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::SECOND * 2);
    node.next(ctx1);

    auto ctx2 = make_context(telem::SECOND * 3);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output_time->at<int64_t>(0), (telem::SECOND * 3).nanoseconds());
}

/// @brief Test that Wait calls mark_changed when firing.
TEST(WaitTest, CallsMarkChangedOnFire) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::TimeSpan(0));
    node.next(ctx1);

    bool changed_called = false;
    std::string changed_param;
    auto ctx2 = make_context(telem::SECOND);
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
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    int call_count = 0;
    auto ctx = make_context(telem::MILLISECOND * 100);
    ctx.mark_changed = [&](const std::string &) { call_count++; };
    node.next(ctx);

    EXPECT_EQ(call_count, 0);
}

/// @brief Test that Wait is_output_truthy delegates to state.
TEST(WaitTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::TimeSpan(0));
    node.next(ctx1);

    auto ctx2 = make_context(telem::SECOND);
    node.next(ctx2);

    EXPECT_TRUE(node.is_output_truthy("output"));
}

/// @brief Test that Wait reset restarts timing from zero.
TEST(WaitTest, ResetRestartsTimingFromZero) {
    TestSetup setup("wait", "duration", telem::SECOND.nanoseconds());
    const time::WaitConfig cfg(setup.ir.nodes[0].config);
    time::Wait node(cfg, setup.make_node());

    auto ctx1 = make_context(telem::SECOND * 5);
    node.next(ctx1);

    node.reset();

    auto ctx2 = make_context(telem::SECOND * 5 + telem::MILLISECOND * 500);
    node.next(ctx2);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 0);

    auto ctx3 = make_context(telem::SECOND * 6 + telem::MILLISECOND * 500);
    node.next(ctx3);

    EXPECT_EQ(output->size(), 1);
}
