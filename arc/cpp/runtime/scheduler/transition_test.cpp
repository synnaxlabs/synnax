// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/scheduler/testutil/testutil.h"

namespace arc::runtime::scheduler {
using namespace testutil;

TEST_F(SchedulerTest, AdvancesOnTransitionFire) {
    mock("trigger", {true});
    auto &first = mock("first_node");
    first.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
    auto &second = mock("second_node");

    auto first_scope = parallel_scope(
        "first",
        {stratum_of({ir::node_member("first_node")})}
    );
    auto second_scope = parallel_scope(
        "second",
        {stratum_of({ir::node_member("second_node")})}
    );
    ir::Transition t;
    t.on = ir::Handle{"first_node", "output"};
    t.target_key = step_key_target("second");
    auto main = sequential_scope(
        "main",
        {ir::scope_member(std::move(first_scope)),
         ir::scope_member(std::move(second_scope))},
        {t}
    );
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };

    auto ir = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("first_node", {"output"}),
         ir_node("second_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(first.next_called, 1);
    EXPECT_EQ(second.next_called, 0);

    first.set_truthy(0);
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(second.next_called, 1);
    EXPECT_EQ(second.reset_called, 1);
}

TEST_F(SchedulerTest, ExitTargetDeactivatesSequence) {
    mock("trigger", {true});
    auto &first = mock("first_node");
    first.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };

    auto first_scope = parallel_scope(
        "first",
        {stratum_of({ir::node_member("first_node")})}
    );
    ir::Transition t;
    t.on = ir::Handle{"first_node", "output"};
    t.target_key = exit_target();
    auto main = sequential_scope(
        "main",
        {ir::scope_member(std::move(first_scope))},
        {t}
    );
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto ir = program_of(
        {ir_node("trigger", {"output"}), ir_node("first_node", {"output"})},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    // The exit trips on cycle 2; the one-shot trigger cannot re-activate main.
    first.set_truthy(0);
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    const int count_at_exit = first.next_called;
    s->next(
        {.elapsed = 3 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(first.next_called, count_at_exit);
}

TEST_F(SchedulerTest, FirstMatchWinsWhenMultipleTransitionsTruthy) {
    mock("trigger", {true});
    auto &first = mock("first_node");
    first.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
    auto &a = mock("a_node");
    auto &b = mock("b_node");

    auto first_scope = parallel_scope(
        "first",
        {stratum_of({ir::node_member("first_node")})}
    );
    auto a_scope = parallel_scope("a", {stratum_of({ir::node_member("a_node")})});
    auto b_scope = parallel_scope("b", {stratum_of({ir::node_member("b_node")})});
    ir::Transition t1;
    t1.on = ir::Handle{"first_node", "output"};
    t1.target_key = step_key_target("a");
    ir::Transition t2;
    t2.on = ir::Handle{"first_node", "output"};
    t2.target_key = step_key_target("b");
    auto main = sequential_scope(
        "main",
        {ir::scope_member(std::move(first_scope)),
         ir::scope_member(std::move(a_scope)),
         ir::scope_member(std::move(b_scope))},
        {t1, t2}
    );
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto ir = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("first_node", {"output"}),
         ir_node("a_node"),
         ir_node("b_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    first.set_truthy(0);
    s->next(
        {.elapsed = 2 * x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 0);
}

TEST_F(SchedulerTest, CascadesMultipleTransitionsInOneCycle) {
    mock("trigger", {true});
    auto &s1 = mock("s1", {true});
    auto &s2 = mock("s2", {true});
    auto &s3 = mock("s3");

    auto mk_step = [](const std::string &key, const std::string &node_key) {
        return parallel_scope(key, {stratum_of({ir::node_member(node_key)})});
    };
    auto sc1 = mk_step("s1", "s1");
    auto sc2 = mk_step("s2", "s2");
    auto sc3 = mk_step("s3", "s3");
    ir::Transition t1;
    t1.on = ir::Handle{"s1", "output"};
    t1.target_key = step_key_target("s2");
    ir::Transition t2;
    t2.on = ir::Handle{"s2", "output"};
    t2.target_key = step_key_target("s3");
    auto main = sequential_scope(
        "main",
        {ir::scope_member(std::move(sc1)),
         ir::scope_member(std::move(sc2)),
         ir::scope_member(std::move(sc3))},
        {t1, t2}
    );
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto ir = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("s1", {"output"}),
         ir_node("s2", {"output"}),
         ir_node("s3")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(s1.next_called, 1);
    EXPECT_EQ(s2.next_called, 1);
    EXPECT_EQ(s3.next_called, 1);
}

// Sequential transitions must fire only when the source node called
// MarkChanged with a truthy output this cycle. Nodes whose output cache
// stays truthy across cycles (e.g., wait, interval, latched comparisons)
// must not drive repeated transitions after their one-shot announcement.
// This mirrors the conditional-edge firing semantic of the pre-Scope
// scheduler.

TEST_F(
    SchedulerTest,
    DoesNotFireTransitionWhenSourceIsTruthyButNeverCalledMarkChanged
) {
    mock("trigger", {true});
    auto &latch = mock("latch", {true});
    latch.suppress_auto_mark = true;
    auto &worker = mock("worker");
    worker.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };

    auto body = parallel_scope("body", {stratum_of({ir::node_member("worker")})});
    ir::Transition t_exit;
    t_exit.on = ir::Handle{"latch", "output"};
    t_exit.target_key = exit_target();
    auto main = sequential_scope("main", {ir::scope_member(std::move(body))}, {t_exit});
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };

    auto program = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("latch", {"output"}),
         ir_node("worker")},
        {},
        root_scope(
            {ir::node_member("trigger"),
             ir::node_member("latch"),
             ir::scope_member(std::move(main))}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(mocks["worker"]->next_called, 1);
    EXPECT_EQ(mocks["worker"]->reset_called, 1);
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(mocks["worker"]->next_called, 2);
    EXPECT_EQ(mocks["worker"]->reset_called, 1);
}

TEST_F(
    SchedulerTest,
    DoesNotReFireTransitionOnLaterCycleWhenSourceStaysTruthyButOnlyMarkedOnFirstCycle
) {
    mock("trigger", {true});
    auto &latch = mock("latch", {true});
    latch.suppress_auto_mark = true;
    int marks = 0;
    latch.on_next = [&marks](const node::Context &ctx) {
        marks++;
        ctx.mark_self_changed();
        if (marks == 1) ctx.mark_changed(0);
    };
    mock("worker_a");
    auto &worker_b = mock("worker_b");
    worker_b.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };

    auto a = parallel_scope("a", {stratum_of({ir::node_member("worker_a")})});
    auto b = parallel_scope("b", {stratum_of({ir::node_member("worker_b")})});
    ir::Transition t_ab;
    t_ab.on = ir::Handle{"latch", "output"};
    t_ab.target_key = step_key_target("b");
    auto main = sequential_scope(
        "main",
        {ir::scope_member(std::move(a)), ir::scope_member(std::move(b))},
        {t_ab}
    );
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };

    auto program = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("latch", {"output"}),
         ir_node("worker_a"),
         ir_node("worker_b")},
        {},
        root_scope(
            {ir::node_member("trigger"),
             ir::node_member("latch"),
             ir::scope_member(std::move(main))}
        )
    );
    const auto s = build(std::move(program));

    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(mocks["worker_a"]->next_called, 1);
    EXPECT_EQ(mocks["worker_b"]->next_called, 1);

    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(mocks["worker_a"]->next_called, 1);
    EXPECT_EQ(mocks["worker_b"]->next_called, 2);

    s->next(
        {.elapsed = 3 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(mocks["worker_a"]->next_called, 1);
    EXPECT_EQ(mocks["worker_b"]->next_called, 3);
}

TEST_F(SchedulerTest, FiresTransitionAgainWhenSourceFreshlyMarksChangedOnLaterCycle) {
    mock("trigger", {true});
    auto &latch = mock("latch");
    latch.suppress_auto_mark = true;
    int cycle = 0;
    latch.on_next = [&cycle, &latch](const node::Context &ctx) {
        cycle++;
        ctx.mark_self_changed();
        if (cycle == 2) {
            latch.set_truthy(0);
            ctx.mark_changed(0);
        }
    };
    mock("worker_a");
    mock("worker_b");

    auto a = parallel_scope("a", {stratum_of({ir::node_member("worker_a")})});
    auto b = parallel_scope("b", {stratum_of({ir::node_member("worker_b")})});
    ir::Transition t_ab;
    t_ab.on = ir::Handle{"latch", "output"};
    t_ab.target_key = step_key_target("b");
    auto main = sequential_scope(
        "main",
        {ir::scope_member(std::move(a)), ir::scope_member(std::move(b))},
        {t_ab}
    );
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };

    auto program = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("latch", {"output"}),
         ir_node("worker_a"),
         ir_node("worker_b")},
        {},
        root_scope(
            {ir::node_member("trigger"),
             ir::node_member("latch"),
             ir::scope_member(std::move(main))}
        )
    );
    const auto s = build(std::move(program));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(mocks["worker_b"]->next_called, 0);
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(mocks["worker_b"]->next_called, 1);
}

TEST_F(SchedulerTest, ResetsASequentialScopesStrataMembersOnActivation) {
    mock("trigger", {true});
    auto &v = mock("V");
    auto &stage_node = mock("M");
    auto stage = parallel_scope("stage", {stratum_of({ir::node_member("M")})});
    auto main = sequential_scope("main", {ir::scope_member(std::move(stage))});
    main.strata.push_back(stratum_of({ir::node_member("V")}));
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto program = program_of(
        {ir_node("trigger", {"output"}), ir_node("V"), ir_node("M")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(program));
    const int base = v.reset_called;
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(v.reset_called, base + 1);
    EXPECT_EQ(stage_node.reset_called, 1);
}

TEST_F(SchedulerTest, ClearsAPendingSelfChangeAndReResetsOnScopeReEntry) {
    auto &trigger = mock("trigger", {true});
    trigger.suppress_auto_mark = true;
    // Activate main on cycles 1 and 3; self-marks keep the entry trigger running.
    trigger.on_next = [&trigger](node::Context &ctx) {
        ctx.mark_self_changed();
        if (trigger.next_called == 1 || trigger.next_called == 3) ctx.mark_changed(0);
    };
    auto &src = mock("src");
    src.on_next = [&src](node::Context &ctx) {
        if (src.next_called == 1) ctx.mark_changed(0);
    };
    auto &v = mock("V");
    v.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
    auto &stage_node = mock("A");
    stage_node.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
    auto first = parallel_scope(
        "first",
        {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("V")})}
    );
    ir::Transition t;
    t.on = ir::Handle{"A", "output"};
    t.target_key = exit_target();
    auto main = sequential_scope("main", {ir::scope_member(std::move(first))}, {t});
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto program = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("src", {"output"}),
         ir_node("V"),
         ir_node("A", {"output"})},
        {continuous_edge("src", "output", "V", "in")},
        root_scope(
            {ir::node_member("trigger"),
             ir::node_member("src"),
             ir::scope_member(std::move(main))}
        )
    );
    const auto s = build(std::move(program));
    const int base = v.reset_called;
    // Cycle 1: activation resets V; V runs via the trigger edge and marks
    // itself.
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(v.reset_called, base + 1);
    EXPECT_EQ(v.next_called, 1);
    // Cycle 2: V replays its self-change and re-marks; A exits main.
    stage_node.set_truthy(0);
    s->next(
        {.elapsed = 2 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(v.next_called, 2);
    // Cycle 3: re-activation resets V again and clears the pending
    // self-change, so V does not replay.
    s->next(
        {.elapsed = 3 * x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    );
    EXPECT_EQ(v.reset_called, base + 2);
    EXPECT_EQ(v.next_called, 2);
}

TEST_F(SchedulerTest, IgnoresAStrataVariableMemberWithNoMatchingNode) {
    mock("trigger", {true});
    auto &m = mock("M");
    auto stage = parallel_scope("stage", {stratum_of({ir::node_member("M")})});
    auto main = sequential_scope("main", {ir::scope_member(std::move(stage))});
    main.strata.push_back(stratum_of({ir::node_member("ghost")}));
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto program = program_of(
        {ir_node("trigger", {"output"}), ir_node("M")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(program));
    EXPECT_NO_THROW(s->next(
        {.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick}
    ));
    EXPECT_EQ(m.next_called, 1);
}
}
