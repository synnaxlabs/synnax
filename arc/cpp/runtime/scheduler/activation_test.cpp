// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/scheduler/testutil/testutil.h"

namespace arc::runtime::scheduler {
using namespace testutil;

TEST_F(SchedulerTest, GatedScopeDoesNotExecuteBeforeActivation) {
    auto &trigger = mock("trigger");
    auto &stage_node = mock("stage_node");
    ir::Handle act{"trigger", "output"};
    auto gated = parallel_scope("stage", {stratum_of({ir::node_member("stage_node")})});
    gated.activations = {{.on = act, .kind = ir::EdgeKind::Conditional}};
    auto ir = program_of(
        {ir_node("trigger", {"output"}), ir_node("stage_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(gated))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(trigger.next_called, 1);
    EXPECT_EQ(stage_node.next_called, 0);
}

TEST_F(SchedulerTest, GatedScopeActivatesOnceHandleFires) {
    mock("trigger", {true});
    auto &stage_node = mock("stage_node");
    ir::Handle act{"trigger", "output"};
    auto gated = parallel_scope("stage", {stratum_of({ir::node_member("stage_node")})});
    gated.activations = {{.on = act, .kind = ir::EdgeKind::Conditional}};
    auto ir = program_of(
        {ir_node("trigger", {"output"}), ir_node("stage_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(gated))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(stage_node.next_called, 1);
    EXPECT_EQ(stage_node.reset_called, 1);
    // Stays active without re-activating; the entry member fired once.
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(stage_node.next_called, 1);
    EXPECT_EQ(stage_node.reset_called, 1);
}

// Stopping a program calls Scheduler::reset outside any cycle, so the context it
// hands nodes still holds the last cycle's stamp. The stage re-activation on the
// next run resets the node again from inside a cycle, so nothing carries the old
// stamp into a value a downstream node can read.
TEST_F(SchedulerTest, ReactivationAfterResetStampsFromTheNewCycle) {
    mock("trigger", {true});
    auto &stage_node = mock("stage_node");
    ir::Handle act{"trigger", "output"};
    auto gated = parallel_scope("stage", {stratum_of({ir::node_member("stage_node")})});
    gated.activations = {{.on = act, .kind = ir::EdgeKind::Conditional}};
    auto ir = program_of(
        {ir_node("trigger", {"output"}), ir_node("stage_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(gated))})
    );
    const auto s = build(std::move(ir));

    const auto first = x::telem::TimeStamp(5 * x::telem::SECOND);
    s->next(
        {.now = first,
         .elapsed = x::telem::MILLISECOND,
         .reason = node::RunReason::TimerTick}
    );
    ASSERT_EQ(stage_node.reset_now.size(), 1);
    EXPECT_EQ(stage_node.reset_now[0], first);

    s->reset();
    EXPECT_EQ(stage_node.reset_now.back(), first);

    const auto second = x::telem::TimeStamp(9 * x::telem::SECOND);
    s->next(
        {.now = second,
         .elapsed = x::telem::MILLISECOND,
         .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(stage_node.reset_now.back(), second);
}

// Anonymous top-level scopes cannot be referenced by `=>` from source, so the
// analyzer emits them as LivenessAlways to mark them as program entrypoints.
// The parallel cascade activates every always-live child of an active parent.
TEST_F(SchedulerTest, AnonymousTopLevelAlwaysScopeAutoActivates) {
    auto &n = mock("n");
    auto anon = always_scope("anon", {stratum_of({ir::node_member("n")})});
    auto ir = program_of(
        {ir_node("n")},
        {},
        root_scope({ir::scope_member(std::move(anon))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(n.reset_called, 1);
    EXPECT_EQ(n.next_called, 1);
}

// A named top-level scope with no activation is emitted by the analyzer when
// the user declared `sequence main { ... }` but no `=> main` trigger exists
// anywhere in source. It must stay inert — the only way to activate it is an
// external trigger.
TEST_F(SchedulerTest, NamedTopLevelGatedScopeWithoutHandleStaysInert) {
    auto &n = mock("n");
    auto gated = parallel_scope("main", {stratum_of({ir::node_member("n")})});
    auto ir = program_of(
        {ir_node("n")},
        {},
        root_scope({ir::scope_member(std::move(gated))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(n.reset_called, 0);
    EXPECT_EQ(n.next_called, 0);
}

// When an outer gated scope activates via its handle, it cascade-activates a
// nested always-live child.
TEST_F(SchedulerTest, CascadeResetsNestedAlwaysScopeOnActivation) {
    mock("trigger", {true});
    auto &inner = mock("inner");
    auto nested = always_scope("nested", {stratum_of({ir::node_member("inner")})});
    auto outer = parallel_scope(
        "outer",
        {stratum_of({ir::scope_member(std::move(nested))})}
    );
    outer.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto ir = program_of(
        {ir_node("trigger", {"output"}), ir_node("inner")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(outer))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(inner.reset_called, 1);
    EXPECT_EQ(inner.next_called, 1);
}

// Exercises the uniform cascade rule at a non-root depth: root (Always) →
// outer (Always) → middle (Always) → leaf node. Any break in the rule would
// leave the leaf unactivated.
TEST_F(SchedulerTest, CascadeThroughNestedAlwaysScopesAtDepth) {
    auto &leaf = mock("leaf");
    auto inner = always_scope("inner", {stratum_of({ir::node_member("leaf")})});
    auto middle = always_scope(
        "middle",
        {stratum_of({ir::scope_member(std::move(inner))})}
    );
    auto outer = always_scope(
        "outer",
        {stratum_of({ir::scope_member(std::move(middle))})}
    );
    auto ir = program_of(
        {ir_node("leaf")},
        {},
        root_scope({ir::scope_member(std::move(outer))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(leaf.reset_called, 1);
    EXPECT_EQ(leaf.next_called, 1);
}

TEST_F(SchedulerTest, IndependentTopLevelGatedScopes) {
    mock("trigger_a", {true});
    mock("trigger_b");
    auto &a = mock("A");
    auto &b = mock("B");
    auto stage_a = parallel_scope("stage_a", {stratum_of({ir::node_member("A")})});
    stage_a.activations = {
        {.on = ir::Handle{"trigger_a", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto stage_b = parallel_scope("stage_b", {stratum_of({ir::node_member("B")})});
    stage_b.activations = {
        {.on = ir::Handle{"trigger_b", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto ir = program_of(
        {ir_node("trigger_a", {"output"}),
         ir_node("trigger_b", {"output"}),
         ir_node("A"),
         ir_node("B")},
        {},
        root_scope(
            {ir::node_member("trigger_a"),
             ir::node_member("trigger_b"),
             ir::scope_member(std::move(stage_a)),
             ir::scope_member(std::move(stage_b))}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 0);
}

TEST_F(SchedulerTest, MixedContinuousAndConditionalInSameGraph) {
    auto &a = mock("A", {true, true});
    auto &b = mock("B");
    auto &c = mock("C");
    a.on_next = [](const node::Context &ctx) {
        ctx.mark_changed(0);
        ctx.mark_changed(1);
    };
    auto ir = program_of(
        {ir_node("A", {"data", "trigger"}), ir_node("B"), ir_node("C")},
        {continuous_edge("A", "data", "B", "in"),
         conditional_edge("A", "trigger", "C", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}),
             stratum_of({ir::node_member("B"), ir::node_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 1);
}
}
