// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/scheduler/testutil/testutil.h"

namespace arc::runtime::scheduler {
using namespace testutil;

TEST_F(SchedulerTest, ReRunsAnAlreadyVisitedNodeWhenALaterNodeWritesBackToIt) {
    auto &a = mock("A");
    auto &b = mock("B");
    // Each node marks only on its first run so the re-pass converges.
    a.on_next = [&a](node::Context &ctx) {
        if (a.next_called == 1) ctx.mark_changed(0);
    };
    b.on_next = [&b](node::Context &ctx) {
        if (b.next_called == 1) ctx.mark_changed(0);
    };
    auto program = program_of(
        {ir_node("A", {"output"}), ir_node("B", {"output"})},
        {continuous_edge("A", "output", "B", "in"),
         continuous_edge("B", "output", "A", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(program));
    // B's backward write lands on already-visited A, forcing a second pass
    // within the same cycle.
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 2);
    // B's pass-1 run consumed its flag; no fresh mark, no re-run.
    EXPECT_EQ(b.next_called, 1);
    // The re-pass does not leak into the next cycle.
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(a.next_called, 3);
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, DoesNotRePassWhenAConditionalBackwardEdgeStaysFalsy) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.on_next = mark_on_next(0);
    b.on_next = mark_on_next(0);
    auto program = program_of(
        {ir_node("A", {"output"}), ir_node("B", {"output"})},
        {continuous_edge("A", "output", "B", "in"),
         conditional_edge("B", "output", "A", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(program));
    // B marks its falsy output each run; the gated backward edge never lands
    // the change, so each cycle stays a single pass.
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(a.next_called, 2);
    EXPECT_EQ(b.next_called, 2);
}

TEST_F(SchedulerTest, DoesNotRePassWhenANodeMarksItselfThroughASelfLoop) {
    auto &a = mock("A");
    a.on_next = mark_on_next(0);
    auto program = program_of(
        {ir_node("A", {"output"})},
        {continuous_edge("A", "output", "A", "in")},
        root_scope({ir::node_member("A")})
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 1);
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(a.next_called, 2);
}

TEST_F(SchedulerTest, BoundsSettlePassesForAMutuallyMarkingCycle) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.on_next = mark_on_next(0);
    b.on_next = mark_on_next(0);
    auto program = program_of(
        {ir_node("A", {"output"}), ir_node("B", {"output"})},
        {continuous_edge("A", "output", "B", "in"),
         continuous_edge("B", "output", "A", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    // Every pass unsettles, so the cycle runs to the bound of node-count + 1
    // passes and stops.
    EXPECT_EQ(a.next_called, 3);
    EXPECT_EQ(b.next_called, 3);
}

TEST_F(SchedulerTest, RunsTheNextSequentialStepOnTheSettlePassSoItObservesPriorWrites) {
    std::vector<std::string> order;
    auto &trigger = mock("trigger", {true});
    auto &v = mock("V");
    auto &first_node = mock("first_node", {true});
    auto &second_node = mock("second_node");
    trigger.on_next = [&order](node::Context &) { order.emplace_back("trigger"); };
    v.on_next = [&order](node::Context &) { order.emplace_back("V"); };
    first_node.on_next = [&order](node::Context &) { order.emplace_back("first"); };
    second_node.on_next = [&order](node::Context &) { order.emplace_back("second"); };
    auto first = parallel_scope("first", {stratum_of({ir::node_member("first_node")})});
    auto second = parallel_scope(
        "second",
        {stratum_of({ir::node_member("second_node")})}
    );
    ir::Transition t;
    t.on = ir::Handle{"first_node", "output"};
    t.target_key = step_key_target("second");
    auto main = sequential_scope(
        "main",
        {ir::scope_member(std::move(first)), ir::scope_member(std::move(second))},
        {t}
    );
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto program = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("V"),
         ir_node("first_node", {"output"}),
         ir_node("second_node")},
        // first_node writes back to V, which ran earlier in the pass.
        {continuous_edge("first_node", "output", "V", "in")},
        root_scope(
            {ir::node_member("trigger"),
             ir::node_member("V"),
             ir::scope_member(std::move(main))}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    // The transition fires on pass 1, but second_node waits for the settle pass and
    // runs after V has absorbed first_node's write. The entry trigger does not re-run
    // on the settle pass.
    EXPECT_EQ(
        order,
        (std::vector<std::string>{"trigger", "V", "first", "V", "second"})
    );
    EXPECT_EQ(first_node.next_called, 1);
    EXPECT_EQ(second_node.next_called, 1);
}

TEST_F(SchedulerTest, RunsAnEntryNodeOncePerActivationAcrossCycles) {
    auto &a = mock("A");
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    s->next(
        {.elapsed = 3 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(a.next_called, 1);
}

TEST_F(SchedulerTest, KeepsDispatchingANonEntryStratumZeroNodeEveryCycle) {
    auto &reader = mock("reader");
    auto n = ir_node("reader", {"output"});
    // A channel read makes the node non-entry.
    n.channels.read[1] = "ch";
    auto ir = program_of({n}, {}, root_scope({ir::node_member("reader")}));
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    s->next(
        {.elapsed = 3 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(reader.next_called, 3);
}

TEST_F(SchedulerTest, ReplaysAnEntryNodeThatMarksItself) {
    auto &a = mock("A");
    a.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    s->next(
        {.elapsed = 3 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(a.next_called, 3);
}

TEST_F(SchedulerTest, ReFiresAnEntryNodeWhenItsScopeReActivates) {
    auto &trigger = mock("trigger", {true});
    trigger.suppress_auto_mark = true;
    // Marks on cycles 1 and 3; self-marks keep the trigger running.
    trigger.on_next = [&trigger](node::Context &ctx) {
        ctx.mark_self_changed();
        if (trigger.next_called == 1 || trigger.next_called == 3) ctx.mark_changed(0);
    };
    auto &entry_node = mock("A", {true});
    auto first = parallel_scope("first", {stratum_of({ir::node_member("A")})});
    ir::Transition t;
    t.on = ir::Handle{"A", "output"};
    t.target_key = exit_target();
    auto main = sequential_scope("main", {ir::scope_member(std::move(first))}, {t});
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto program = program_of(
        {ir_node("trigger", {"output"}), ir_node("A", {"output"})},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(program));
    // Cycle 1: main activates; A runs once and exits the sequence.
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(entry_node.next_called, 1);
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(entry_node.next_called, 1);
    // Cycle 3: main re-activates; the reset lets A fire again.
    s->next(
        {.elapsed = 3 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(entry_node.next_called, 2);
}

TEST_F(SchedulerTest, RunsAnEntrySequentialFlowStepOnceWhileTheStepStaysActive) {
    mock("trigger", {true});
    auto &step = mock("step");
    auto main = sequential_scope("main", {ir::node_member("step")});
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto program = program_of(
        {ir_node("trigger", {"output"}), ir_node("step")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    s->next(
        {.elapsed = 3 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(step.next_called, 1);
}

// A node's pending-change flag is consumed when it runs; settle passes must
// not re-dispatch side-effecting nodes without a fresh mark.

/// @brief configures m to announce output 0 only on its first run.
static void mark_once(MockNode &m) {
    m.on_next = [&m](node::Context &ctx) {
        if (m.next_called == 1) ctx.mark_changed(0);
    };
}

TEST_F(SchedulerTest, DispatchesAMarkedNodeOnceDespiteAnUnrelatedRePass) {
    auto &trigger = mock("trigger");
    auto &worker = mock("worker");
    auto &a = mock("A");
    auto &b = mock("B");
    mark_once(trigger);
    mark_once(a);
    mark_once(b);
    auto program = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("worker"),
         ir_node("A", {"output"}),
         ir_node("B", {"output"})},
        {continuous_edge("trigger", "output", "worker", "in"),
         continuous_edge("A", "output", "B", "in"),
         continuous_edge("B", "output", "A", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("trigger"), ir::node_member("A")}),
             stratum_of({ir::node_member("worker"), ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    // A's re-run proves a second pass happened.
    EXPECT_EQ(a.next_called, 2);
    EXPECT_EQ(worker.next_called, 1);
}

TEST_F(SchedulerTest, DispatchesChainNodesOncePerMarkAcrossSettlePasses) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    mark_once(a);
    mark_once(b);
    mark_once(c);
    auto program = program_of(
        {ir_node("A", {"output"}), ir_node("B", {"output"}), ir_node("C", {"output"})},
        {continuous_edge("A", "output", "B", "in"),
         continuous_edge("B", "output", "C", "in"),
         continuous_edge("C", "output", "A", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}),
             stratum_of({ir::node_member("B")}),
             stratum_of({ir::node_member("C")})}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 2);
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 1);
}

TEST_F(SchedulerTest, FiresAStagesOneShotTriggeredNodeOncePerActivation) {
    mock("trigger", {true});
    auto &entry = mock("entry");
    auto &creator = mock("creator");
    auto &a = mock("A");
    auto &b = mock("B");
    mark_once(entry);
    mark_once(a);
    mark_once(b);
    auto stage = parallel_scope(
        "stage",
        {stratum_of({ir::node_member("entry")}),
         stratum_of({ir::node_member("creator")})}
    );
    auto main = sequential_scope("main", {ir::scope_member(std::move(stage))});
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto program = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("entry", {"output"}),
         ir_node("creator"),
         ir_node("A", {"output"}),
         ir_node("B", {"output"})},
        {continuous_edge("entry", "output", "creator", "in"),
         continuous_edge("A", "output", "B", "in"),
         continuous_edge("B", "output", "A", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("trigger"), ir::node_member("A")}),
             stratum_of({ir::node_member("B"), ir::scope_member(std::move(main))})}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    // The entry node fires once per activation; the creator must dispatch exactly
    // once, like a range create in a stage.
    EXPECT_EQ(entry.next_called, 1);
    EXPECT_EQ(creator.next_called, 1);
}

TEST_F(SchedulerTest, ReDispatchesANodeMarkedAgainAfterItAlreadyRan) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    mark_once(a);
    mark_once(c);
    auto program = program_of(
        {ir_node("A", {"output"}), ir_node("B"), ir_node("C", {"output"})},
        {continuous_edge("A", "output", "B", "in"),
         continuous_edge("A", "output", "C", "in"),
         continuous_edge("C", "output", "B", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}),
             stratum_of({ir::node_member("B")}),
             stratum_of({ir::node_member("C")})}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    // C's write marked B after B ran; the fresh mark re-dispatches it.
    EXPECT_EQ(b.next_called, 2);
    EXPECT_EQ(c.next_called, 1);
}

TEST_F(SchedulerTest, NeverDispatchesAnUnmarkedNodeAcrossSettlePasses) {
    mock("quiet");
    auto &silent = mock("silent");
    auto &a = mock("A");
    auto &b = mock("B");
    mark_once(a);
    mark_once(b);
    auto program = program_of(
        {ir_node("quiet", {"output"}),
         ir_node("silent"),
         ir_node("A", {"output"}),
         ir_node("B", {"output"})},
        {continuous_edge("quiet", "output", "silent", "in"),
         continuous_edge("A", "output", "B", "in"),
         continuous_edge("B", "output", "A", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("quiet"), ir::node_member("A")}),
             stratum_of({ir::node_member("silent"), ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(a.next_called, 2);
    EXPECT_EQ(silent.next_called, 0);
}

TEST_F(SchedulerTest, PreservesAMarkANodeSetsOnItselfWhileItRuns) {
    auto &starter = mock("starter");
    auto &looper = mock("looper");
    auto &a = mock("A");
    auto &b = mock("B");
    mark_once(starter);
    mark_once(looper);
    mark_once(a);
    mark_once(b);
    auto program = program_of(
        {ir_node("starter", {"output"}),
         ir_node("looper", {"output"}),
         ir_node("A", {"output"}),
         ir_node("B", {"output"})},
        {continuous_edge("starter", "output", "looper", "in"),
         continuous_edge("looper", "output", "looper", "in"),
         continuous_edge("A", "output", "B", "in"),
         continuous_edge("B", "output", "A", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("starter"), ir::node_member("A")}),
             stratum_of({ir::node_member("looper"), ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    // The self-mark lands after consumption, so the re-pass delivers it.
    EXPECT_EQ(looper.next_called, 2);
}

TEST_F(SchedulerTest, DispatchesAgainOnAFreshMarkInTheNextCycle) {
    auto &a = mock("A");
    auto &worker = mock("worker");
    a.on_next = [](node::Context &ctx) {
        ctx.mark_changed(0);
        ctx.mark_self_changed();
    };
    auto program = program_of(
        {ir_node("A", {"output"}), ir_node("worker")},
        {continuous_edge("A", "output", "worker", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}),
             stratum_of({ir::node_member("worker")})}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(worker.next_called, 1);
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(worker.next_called, 2);
}
}
