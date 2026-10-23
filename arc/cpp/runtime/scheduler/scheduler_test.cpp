// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/scheduler/testutil/testutil.h"

namespace arc::runtime::scheduler {
using namespace testutil;

TEST_F(SchedulerTest, EmptyProgramDoesNotCrash) {
    const auto s = build(ir::IR{});
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
}

TEST_F(SchedulerTest, ExecutesAllPhaseZeroMembers) {
    mock("A");
    mock("B");
    mock("C");
    auto ir = program_of(
        {ir_node("A"), ir_node("B"), ir_node("C")},
        {},
        root_scope({ir::node_member("A"), ir::node_member("B"), ir::node_member("C")})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(mocks["A"]->next_called, 1);
    EXPECT_EQ(mocks["B"]->next_called, 1);
    EXPECT_EQ(mocks["C"]->next_called, 1);
}

TEST_F(SchedulerTest, RunsAnEntryPhase0MemberOncePerActivation) {
    auto &a = mock("A");
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    s->next(
        {.elapsed = 3 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    // A has no inputs, so it is an entry node: one run per activation.
    EXPECT_EQ(a.next_called, 1);
}

TEST_F(SchedulerTest, PhaseNSkipsWithoutIncomingChange) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {continuous_edge("A", "output", "B", "input")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 0);
}

TEST_F(SchedulerTest, ContinuousEdgePropagatesToDownstream) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.on_next = mark_on_next(0);
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {continuous_edge("A", "output", "B", "input")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, ConditionalEdgeGatedOnSourceTruthiness) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.on_next = [](node::Context &ctx) {
        ctx.mark_changed(0);
        ctx.mark_self_changed();
    };
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {conditional_edge("A", "output", "B", "input")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(b.next_called, 0);

    a.set_truthy(0);
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, FiresOnlyTheEdgeWhoseSourceParamWasMarked) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    // A declares two outputs ("x", "y"); only "x" (ordinal 0) fires.
    a.on_next = mark_on_next(0);
    auto ir = program_of(
        {ir_node("A", {"x", "y"}), ir_node("B"), ir_node("C")},
        {continuous_edge("A", "x", "B", "in"), continuous_edge("A", "y", "C", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}),
             stratum_of({ir::node_member("B"), ir::node_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 0);
}

TEST_F(SchedulerTest, FansOutToMultipleDownstreamMembers) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    a.on_next = mark_on_next(0);
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B"), ir_node("C")},
        {continuous_edge("A", "output", "B", "in"),
         continuous_edge("A", "output", "C", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}),
             stratum_of({ir::node_member("B"), ir::node_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 1);
}

TEST_F(SchedulerTest, JoinNodeRunsOnceWhenMultipleInputsFire) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    a.on_next = mark_on_next(0);
    b.on_next = mark_on_next(0);
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B", {"output"}), ir_node("C")},
        {continuous_edge("A", "output", "C", "a"),
         continuous_edge("B", "output", "C", "b")},
        root_with_strata(
            {stratum_of({ir::node_member("A"), ir::node_member("B")}),
             stratum_of({ir::node_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(c.next_called, 1);
}

TEST_F(SchedulerTest, NextReturnsTheHighestStampItsNodesReserved) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto first_a = x::telem::TimeStamp(0);
    auto first_b = x::telem::TimeStamp(0);
    a.on_next = [&first_a](node::Context &ctx) { first_a = ctx.reserve_stamps(2); };
    b.on_next = [&first_b](node::Context &ctx) { first_b = ctx.reserve_stamps(3); };
    auto ir = program_of(
        {ir_node("A"), ir_node("B")},
        {},
        root_with_strata({stratum_of({ir::node_member("A"), ir::node_member("B")})})
    );
    const auto s = build(std::move(ir));
    const auto highest = s->next(
        {.now = x::telem::TimeStamp(100), .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(first_a, x::telem::TimeStamp(100));
    EXPECT_EQ(first_b, x::telem::TimeStamp(102));
    EXPECT_EQ(highest, x::telem::TimeStamp(104));
}

TEST_F(SchedulerTest, NextRestartsReservationsAtEachCyclesStamp) {
    auto &a = mock("A");
    auto first = x::telem::TimeStamp(0);
    a.on_next = [&first](node::Context &ctx) {
        first = ctx.reserve_stamps(5);
        ctx.mark_self_changed();
    };
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next({.now = x::telem::TimeStamp(100), .reason = node::RunReason::TimerTick});
    const auto highest = s->next(
        {.now = x::telem::TimeStamp(200), .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(first, x::telem::TimeStamp(200));
    EXPECT_EQ(highest, x::telem::TimeStamp(204));
}

TEST_F(SchedulerTest, NextReturnsZeroWhenNoNodeReservesAStamp) {
    mock("A");
    auto ir = program_of(
        {ir_node("A")},
        {},
        root_with_strata({stratum_of({ir::node_member("A")})})
    );
    const auto s = build(std::move(ir));
    EXPECT_EQ(
        s->next(
            {.now = x::telem::TimeStamp(100), .reason = node::RunReason::TimerTick}
        ),
        x::telem::TimeStamp(0)
    );
}

TEST_F(SchedulerTest, DiamondSinkRunsExactlyOnce) {
    mock("A").on_next = mark_on_next(0);
    mock("B").on_next = mark_on_next(0);
    mock("C").on_next = mark_on_next(0);
    auto &d = mock("D");
    auto ir = program_of(
        {ir_node("A", {"output"}),
         ir_node("B", {"output"}),
         ir_node("C", {"output"}),
         ir_node("D")},
        {continuous_edge("A", "output", "B", "in"),
         continuous_edge("A", "output", "C", "in"),
         continuous_edge("B", "output", "D", "a"),
         continuous_edge("C", "output", "D", "b")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}),
             stratum_of({ir::node_member("B"), ir::node_member("C")}),
             stratum_of({ir::node_member("D")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(d.next_called, 1);
}

TEST_F(SchedulerTest, IgnoresEdgesWithEndpointsOutsideMembership) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto ir = program_of(
        {ir_node("A"), ir_node("B", {"y"})},
        {continuous_edge("ghost", "x", "A", "in"),
         continuous_edge("B", "y", "phantom", "in")},
        root_scope({ir::node_member("A"), ir::node_member("B")})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, ConditionalFiresEveryCycleWhileTruthy) {
    auto &a = mock("A", {true});
    a.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
    auto &b = mock("B");
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {conditional_edge("A", "output", "B", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    s->next(
        {.elapsed = 3 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(b.next_called, 3);
}

TEST_F(SchedulerTest, ConditionalStopsFiringWhenSourceBecomesFalsy) {
    auto &a = mock("A", {true});
    a.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
    auto &b = mock("B");
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {conditional_edge("A", "output", "B", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(b.next_called, 1);

    a.output_truthy[0] = false;
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, ContinuousEdgesIgnoreSourceTruthiness) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.on_next = mark_on_next(0);
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {continuous_edge("A", "output", "B", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, ConditionalEdgesIndependentPerParam) {
    // A declares two outputs ("x", "y"); only "x" (ordinal 0) is truthy.
    auto &a = mock("A", {true, false});
    auto &b = mock("B");
    auto &c = mock("C");
    a.on_next = [](const node::Context &ctx) {
        ctx.mark_changed(0);
        ctx.mark_changed(1);
    };
    auto ir = program_of(
        {ir_node("A", {"x", "y"}), ir_node("B"), ir_node("C")},
        {conditional_edge("A", "x", "B", "in"), conditional_edge("A", "y", "C", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}),
             stratum_of({ir::node_member("B"), ir::node_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 0);
}

TEST_F(SchedulerTest, SelfChangedReplaysUntilNodeStopsMarking) {
    auto &a = mock("A");
    int count = 0;
    a.on_next = [&count](node::Context &ctx) {
        count++;
        if (count <= 2) ctx.mark_self_changed();
    };
    // trigger fires a single change into A on cycle 1, then stays quiet.
    // A's self-marking should drive the next two replays on its own; once
    // it stops marking, the scheduler must not replay again.
    auto &trigger = mock("trigger");
    trigger.suppress_auto_mark = true;
    bool fired = false;
    trigger.on_next = [&fired](const node::Context &ctx) {
        if (!fired) {
            ctx.mark_changed(0);
            fired = true;
        }
    };
    auto ir = program_of(
        {ir_node("trigger", {"kick"}), ir_node("A")},
        {continuous_edge("trigger", "kick", "A", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("trigger")}),
             stratum_of({ir::node_member("A")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    s->next(
        {.elapsed = 3 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    s->next(
        {.elapsed = 4 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(a.next_called, 3);
}

TEST_F(SchedulerTest, ElapsedTimePassedThrough) {
    auto &a = mock("A");
    a.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next(
        {.elapsed = 5 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    s->next(
        {.elapsed = 10 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    ASSERT_EQ(a.elapsed_values.size(), 2);
    EXPECT_EQ(a.elapsed_values[0], 5 * x::telem::MILLISECOND);
    EXPECT_EQ(a.elapsed_values[1], 10 * x::telem::MILLISECOND);
}

TEST_F(SchedulerTest, ReasonChannelInputPassedThrough) {
    auto &a = mock("A");
    node::RunReason received = node::RunReason::TimerTick;
    a.on_next = [&received](const node::Context &ctx) { received = ctx.cycle.reason; };
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next(
        {.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::ChannelInput}
    );
    EXPECT_EQ(received, node::RunReason::ChannelInput);
}

TEST_F(SchedulerTest, NextDeadlineDefaultsToMax) {
    mock("A");
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(s->next_deadline().at, x::telem::TimeSpan::max());
    EXPECT_EQ(s->next_deadline().span, x::telem::TimeSpan::max());
}

TEST_F(SchedulerTest, NextDeadlineReturnsMinimumWithItsSpan) {
    const auto d = deadline_of(
        {.at = 10 * x::telem::MILLISECOND, .span = 5 * x::telem::MILLISECOND},
        {.at = 3 * x::telem::MILLISECOND, .span = 20 * x::telem::MILLISECOND}
    );
    EXPECT_EQ(d.at, 3 * x::telem::MILLISECOND);
    EXPECT_EQ(d.span, 20 * x::telem::MILLISECOND);
}

TEST_F(SchedulerTest, NextDeadlineKeepsShortestSpanOfEqualDeadlines) {
    const auto d = deadline_of(
        {.at = 5 * x::telem::MILLISECOND, .span = 20 * x::telem::MILLISECOND},
        {.at = 5 * x::telem::MILLISECOND, .span = 5 * x::telem::MILLISECOND}
    );
    EXPECT_EQ(d.at, 5 * x::telem::MILLISECOND);
    EXPECT_EQ(d.span, 5 * x::telem::MILLISECOND);
}

TEST_F(SchedulerTest, NextDeadlineKeepsShortestSpanWhenItComesFirst) {
    const auto d = deadline_of(
        {.at = 5 * x::telem::MILLISECOND, .span = 5 * x::telem::MILLISECOND},
        {.at = 5 * x::telem::MILLISECOND, .span = 20 * x::telem::MILLISECOND}
    );
    EXPECT_EQ(d.at, 5 * x::telem::MILLISECOND);
    EXPECT_EQ(d.span, 5 * x::telem::MILLISECOND);
}

TEST_F(SchedulerTest, NextDeadlineResetsBetweenCycles) {
    auto &a = mock("A");
    int call = 0;
    a.on_next = [&call](const node::Context &ctx) {
        call++;
        ctx.mark_self_changed();
        if (call == 1) ctx.set_deadline(x::telem::SECOND, x::telem::SECOND);
    };
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(s->next_deadline().at, x::telem::SECOND);
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(s->next_deadline().at, x::telem::TimeSpan::max());
    EXPECT_EQ(s->next_deadline().span, x::telem::TimeSpan::max());
}

TEST_F(SchedulerTest, ContinuesAfterErrorReport) {
    MockErrorHandler h;
    const auto err = x::errors::Error("boom-A", "test");
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    a.on_next = [&err](const node::Context &ctx) { ctx.report_error(err); };
    auto ir = program_of(
        {ir_node("A"), ir_node("B"), ir_node("C")},
        {},
        root_scope({ir::node_member("A"), ir::node_member("B"), ir::node_member("C")})
    );
    const auto s = build_with_handler(std::move(ir), h.handler);
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 1);
    EXPECT_EQ(h.errors.size(), 1);
}

TEST_F(SchedulerTest, AccumulatesMultipleErrors) {
    MockErrorHandler h;
    const auto err_a = x::errors::Error("boom-A", "test");
    const auto err_b = x::errors::Error("boom-B", "test");
    mock("A").on_next = [&err_a](const node::Context &ctx) { ctx.report_error(err_a); };
    mock("B").on_next = [&err_b](const node::Context &ctx) { ctx.report_error(err_b); };
    auto ir = program_of(
        {ir_node("A"), ir_node("B")},
        {},
        root_scope({ir::node_member("A"), ir::node_member("B")})
    );
    const auto s = build_with_handler(std::move(ir), h.handler);
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(h.errors.size(), 2);
}

TEST_F(SchedulerTest, ZeroElapsedTimeAccepted) {
    auto &a = mock("A");
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::TimeSpan(0), .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(a.elapsed_values[0], x::telem::TimeSpan(0));
}

TEST_F(SchedulerTest, SelfLoopEdgeDoesNotCrash) {
    auto &a = mock("A");
    a.on_next = mark_on_next(0);
    auto ir = program_of(
        {ir_node("A", {"output"})},
        {continuous_edge("A", "output", "A", "in")},
        root_scope({ir::node_member("A")})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
}

TEST_F(SchedulerTest, EmptySequentialScopeTolerated) {
    auto &trigger = mock("trigger", {true});
    ir::Scope main;
    main.key = "main";
    main.mode = ir::ScopeMode::Sequential;
    main.liveness = ir::Liveness::Gated;
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto ir = program_of(
        {ir_node("trigger", {"output"})},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(trigger.next_called, 1);
}
}
