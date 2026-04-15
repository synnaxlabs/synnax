// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <functional>
#include <memory>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"

namespace arc::runtime::scheduler {

/// @brief configurable mock node used across scheduler tests. Mirrors the
/// Go scheduler's MockNode: auto-propagates MarkChanged for every
/// currently-truthy param so a single ParamTruthy toggle drives both
/// conditional edges and gated-scope activations.
struct MockNode final : public node::Node {
    int next_called = 0;
    int reset_called = 0;
    std::vector<x::telem::TimeSpan> elapsed_values;

    std::unordered_set<std::string> param_truthy;
    std::function<void(node::Context &)> on_next;

    x::errors::Error next(node::Context &ctx) override {
        next_called++;
        elapsed_values.push_back(ctx.elapsed);
        for (const auto &param: param_truthy)
            ctx.mark_changed(param);
        if (on_next) on_next(ctx);
        return x::errors::NIL;
    }

    void reset() override { reset_called++; }

    [[nodiscard]] bool is_output_truthy(const std::string &param) const override {
        return param_truthy.contains(param);
    }

    void mark_on_next(const std::string &param) {
        on_next = [param](const node::Context &ctx) { ctx.mark_changed(param); };
    }
};

/// @brief collects scheduler-reported errors for assertion.
struct MockErrorHandler {
    std::vector<x::errors::Error> errors;
    errors::Handler handler = [this](const x::errors::Error &e) {
        errors.push_back(e);
    };
};

// ----- IR construction helpers -----

static ir::Member node_ref_member(const std::string &key) {
    ir::Member m;
    m.key = key;
    ir::NodeRef ref;
    ref.key = key;
    m.node_ref = std::move(ref);
    return m;
}

static ir::Member scope_to_member(ir::Scope scope) {
    ir::Member m;
    m.key = scope.key;
    m.scope = x::mem::indirect<ir::Scope>(std::move(scope));
    return m;
}

static ir::Phase phase_of(std::vector<ir::Member> members) {
    ir::Phase p;
    p.members = std::move(members);
    return p;
}

static ir::Scope parallel_scope(std::string key, std::vector<ir::Phase> phases) {
    ir::Scope s;
    s.key = std::move(key);
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Gated;
    s.phases = std::move(phases);
    return s;
}

static ir::Scope sequential_scope(
    std::string key,
    std::vector<ir::Member> members,
    std::vector<ir::Transition> transitions = {}
) {
    ir::Scope s;
    s.key = std::move(key);
    s.mode = ir::ScopeMode::Sequential;
    s.liveness = ir::Liveness::Gated;
    s.members = std::move(members);
    s.transitions = std::move(transitions);
    return s;
}

static ir::Scope root_scope(std::vector<ir::Member> members) {
    ir::Scope s;
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Always;
    if (!members.empty()) s.phases.push_back(phase_of(std::move(members)));
    return s;
}

static ir::Scope root_with_phases(std::vector<ir::Phase> phases) {
    ir::Scope s;
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Always;
    s.phases = std::move(phases);
    return s;
}

static ir::Edge continuous_edge(
    const std::string &src,
    const std::string &src_param,
    const std::string &tgt,
    const std::string &tgt_param
) {
    return ir::Edge{
        ir::Handle{src, src_param},
        ir::Handle{tgt, tgt_param},
        ir::EdgeKind::Continuous
    };
}

static ir::Edge conditional_edge(
    const std::string &src,
    const std::string &src_param,
    const std::string &tgt,
    const std::string &tgt_param
) {
    return ir::Edge{
        ir::Handle{src, src_param},
        ir::Handle{tgt, tgt_param},
        ir::EdgeKind::Conditional
    };
}

static ir::TransitionTarget member_key_target(const std::string &key) {
    ir::TransitionTarget t;
    t.member_key = key;
    return t;
}

static ir::TransitionTarget exit_target() {
    ir::TransitionTarget t;
    t.exit = true;
    return t;
}

static ir::IR program_of(
    std::vector<std::string> node_keys,
    std::vector<ir::Edge> edges,
    ir::Scope root
) {
    ir::IR ir;
    for (auto &k: node_keys) {
        ir::Node n;
        n.key = std::move(k);
        ir.nodes.push_back(std::move(n));
    }
    ir.edges.assign(edges.begin(), edges.end());
    ir.root = std::move(root);
    return ir;
}

class SchedulerTest : public ::testing::Test {
public:
    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes_;
    std::unordered_map<std::string, MockNode *> mocks_;

    MockNode &mock(const std::string &key) {
        auto node = std::make_unique<MockNode>();
        auto *ptr = node.get();
        nodes_[key] = std::move(node);
        mocks_[key] = ptr;
        return *ptr;
    }

    std::unique_ptr<Scheduler> build(ir::IR ir) {
        return std::make_unique<Scheduler>(
            std::move(ir),
            nodes_,
            x::telem::TimeSpan(0)
        );
    }

    std::unique_ptr<Scheduler> build_with_handler(ir::IR ir, errors::Handler handler) {
        return std::make_unique<Scheduler>(
            std::move(ir),
            nodes_,
            x::telem::TimeSpan(0),
            std::move(handler)
        );
    }
};

// ----- Construction -----

TEST_F(SchedulerTest, EmptyProgramDoesNotCrash) {
    const auto s = build(ir::IR{});
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
}

TEST_F(SchedulerTest, ExecutesAllPhaseZeroMembers) {
    mock("A");
    mock("B");
    mock("C");
    auto ir = program_of(
        {"A", "B", "C"},
        {},
        root_scope({node_ref_member("A"), node_ref_member("B"), node_ref_member("C")})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(mocks_["A"]->next_called, 1);
    EXPECT_EQ(mocks_["B"]->next_called, 1);
    EXPECT_EQ(mocks_["C"]->next_called, 1);
}

// ----- Phase-based execution -----

TEST_F(SchedulerTest, Phase0ExecutesUnconditionallyEachCycle) {
    auto &a = mock("A");
    auto ir = program_of({"A"}, {}, root_scope({node_ref_member("A")}));
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(3 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 3);
}

TEST_F(SchedulerTest, PhaseNSkipsWithoutIncomingChange) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto ir = program_of(
        {"A", "B"},
        {continuous_edge("A", "output", "B", "input")},
        root_with_phases(
            {phase_of({node_ref_member("A")}), phase_of({node_ref_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 0);
}

TEST_F(SchedulerTest, ContinuousEdgePropagatesToDownstream) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.mark_on_next("output");
    auto ir = program_of(
        {"A", "B"},
        {continuous_edge("A", "output", "B", "input")},
        root_with_phases(
            {phase_of({node_ref_member("A")}), phase_of({node_ref_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, ConditionalEdgeGatedOnSourceTruthiness) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.mark_on_next("output");
    auto ir = program_of(
        {"A", "B"},
        {conditional_edge("A", "output", "B", "input")},
        root_with_phases(
            {phase_of({node_ref_member("A")}), phase_of({node_ref_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 0);

    a.param_truthy.insert("output");
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, FiresOnlyTheEdgeWhoseSourceParamWasMarked) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    a.on_next = [](const node::Context &ctx) { ctx.mark_changed("x"); };
    auto ir = program_of(
        {"A", "B", "C"},
        {continuous_edge("A", "x", "B", "in"), continuous_edge("A", "y", "C", "in")},
        root_with_phases(
            {phase_of({node_ref_member("A")}),
             phase_of({node_ref_member("B"), node_ref_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 0);
}

TEST_F(SchedulerTest, FansOutToMultipleDownstreamMembers) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    a.mark_on_next("output");
    auto ir = program_of(
        {"A", "B", "C"},
        {continuous_edge("A", "output", "B", "in"),
         continuous_edge("A", "output", "C", "in")},
        root_with_phases(
            {phase_of({node_ref_member("A")}),
             phase_of({node_ref_member("B"), node_ref_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 1);
}

TEST_F(SchedulerTest, JoinNodeRunsOnceWhenMultipleInputsFire) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    a.mark_on_next("output");
    b.mark_on_next("output");
    auto ir = program_of(
        {"A", "B", "C"},
        {continuous_edge("A", "output", "C", "a"),
         continuous_edge("B", "output", "C", "b")},
        root_with_phases(
            {phase_of({node_ref_member("A"), node_ref_member("B")}),
             phase_of({node_ref_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(c.next_called, 1);
}

TEST_F(SchedulerTest, DiamondSinkRunsExactlyOnce) {
    mock("A").mark_on_next("output");
    mock("B").mark_on_next("output");
    mock("C").mark_on_next("output");
    auto &d = mock("D");
    auto ir = program_of(
        {"A", "B", "C", "D"},
        {continuous_edge("A", "output", "B", "in"),
         continuous_edge("A", "output", "C", "in"),
         continuous_edge("B", "output", "D", "a"),
         continuous_edge("C", "output", "D", "b")},
        root_with_phases(
            {phase_of({node_ref_member("A")}),
             phase_of({node_ref_member("B"), node_ref_member("C")}),
             phase_of({node_ref_member("D")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(d.next_called, 1);
}

TEST_F(SchedulerTest, IgnoresEdgesWithEndpointsOutsideMembership) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto ir = program_of(
        {"A", "B"},
        {continuous_edge("ghost", "x", "A", "in"),
         continuous_edge("B", "y", "phantom", "in")},
        root_scope({node_ref_member("A"), node_ref_member("B")})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
}

// ----- Conditional edge lifecycle -----

TEST_F(SchedulerTest, ConditionalFiresEveryCycleWhileTruthy) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.param_truthy.insert("output");
    auto ir = program_of(
        {"A", "B"},
        {conditional_edge("A", "output", "B", "in")},
        root_with_phases(
            {phase_of({node_ref_member("A")}), phase_of({node_ref_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(3 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 3);
}

TEST_F(SchedulerTest, ConditionalStopsFiringWhenSourceBecomesFalsy) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.param_truthy.insert("output");
    auto ir = program_of(
        {"A", "B"},
        {conditional_edge("A", "output", "B", "in")},
        root_with_phases(
            {phase_of({node_ref_member("A")}), phase_of({node_ref_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);

    a.param_truthy.erase("output");
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, ContinuousEdgesIgnoreSourceTruthiness) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.on_next = [](const node::Context &ctx) { ctx.mark_changed("output"); };
    auto ir = program_of(
        {"A", "B"},
        {continuous_edge("A", "output", "B", "in")},
        root_with_phases(
            {phase_of({node_ref_member("A")}), phase_of({node_ref_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);
}

TEST_F(SchedulerTest, ConditionalEdgesIndependentPerParam) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    a.param_truthy.insert("x");
    // "y" is not truthy — its conditional edge must not fire.
    a.on_next = [](const node::Context &ctx) {
        ctx.mark_changed("x");
        ctx.mark_changed("y");
    };
    auto ir = program_of(
        {"A", "B", "C"},
        {conditional_edge("A", "x", "B", "in"), conditional_edge("A", "y", "C", "in")},
        root_with_phases(
            {phase_of({node_ref_member("A")}),
             phase_of({node_ref_member("B"), node_ref_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 0);
}

// ----- Self-changed replay -----

TEST_F(SchedulerTest, SelfChangedReplaysUntilNodeStopsMarking) {
    mock("marker");
    auto &a = mock("A");
    int count = 0;
    a.on_next = [&count](node::Context &ctx) {
        count++;
        if (count <= 2) ctx.mark_self_changed();
    };
    auto ir = program_of(
        {"marker", "A"},
        {},
        root_with_phases(
            {phase_of({node_ref_member("marker")}), phase_of({node_ref_member("A")})}
        )
    );
    const auto s = build(std::move(ir));
    s->mark_node_changed("A");
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(3 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(4 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 3);
}

// ----- Context passthrough -----

TEST_F(SchedulerTest, ElapsedTimePassedThrough) {
    auto &a = mock("A");
    auto ir = program_of({"A"}, {}, root_scope({node_ref_member("A")}));
    const auto s = build(std::move(ir));
    s->next(5 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(10 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(a.elapsed_values.size(), 2);
    EXPECT_EQ(a.elapsed_values[0], 5 * x::telem::MILLISECOND);
    EXPECT_EQ(a.elapsed_values[1], 10 * x::telem::MILLISECOND);
}

TEST_F(SchedulerTest, ReasonChannelInputPassedThrough) {
    auto &a = mock("A");
    node::RunReason received = node::RunReason::TimerTick;
    a.on_next = [&received](const node::Context &ctx) { received = ctx.reason; };
    auto ir = program_of({"A"}, {}, root_scope({node_ref_member("A")}));
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::ChannelInput);
    EXPECT_EQ(received, node::RunReason::ChannelInput);
}

TEST_F(SchedulerTest, NextDeadlineDefaultsToMax) {
    mock("A");
    auto ir = program_of({"A"}, {}, root_scope({node_ref_member("A")}));
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(s->next_deadline(), x::telem::TimeSpan::max());
}

TEST_F(SchedulerTest, NextDeadlineReturnsMinimum) {
    auto &a = mock("A");
    auto &b = mock("B");
    a.on_next = [](const node::Context &ctx) {
        ctx.set_deadline(10 * x::telem::MILLISECOND);
    };
    b.on_next = [](const node::Context &ctx) {
        ctx.set_deadline(3 * x::telem::MILLISECOND);
    };
    auto ir = program_of(
        {"A", "B"},
        {},
        root_scope({node_ref_member("A"), node_ref_member("B")})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(s->next_deadline(), 3 * x::telem::MILLISECOND);
}

TEST_F(SchedulerTest, NextDeadlineResetsBetweenCycles) {
    auto &a = mock("A");
    int call = 0;
    a.on_next = [&call](const node::Context &ctx) {
        call++;
        if (call == 1) ctx.set_deadline(x::telem::SECOND);
    };
    auto ir = program_of({"A"}, {}, root_scope({node_ref_member("A")}));
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(s->next_deadline(), x::telem::SECOND);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(s->next_deadline(), x::telem::TimeSpan::max());
}

// ----- Gated scope activation -----

TEST_F(SchedulerTest, GatedScopeDoesNotExecuteBeforeActivation) {
    auto &trigger = mock("trigger");
    auto &stage_node = mock("stage_node");
    ir::Handle act{"trigger", "output"};
    auto gated = parallel_scope("stage", {phase_of({node_ref_member("stage_node")})});
    gated.activation = act;
    auto ir = program_of(
        {"trigger", "stage_node"},
        {},
        root_scope({node_ref_member("trigger"), scope_to_member(std::move(gated))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(trigger.next_called, 1);
    EXPECT_EQ(stage_node.next_called, 0);
}

TEST_F(SchedulerTest, GatedScopeActivatesOnceHandleFires) {
    auto &trigger = mock("trigger");
    auto &stage_node = mock("stage_node");
    trigger.param_truthy.insert("output");
    ir::Handle act{"trigger", "output"};
    auto gated = parallel_scope("stage", {phase_of({node_ref_member("stage_node")})});
    gated.activation = act;
    auto ir = program_of(
        {"trigger", "stage_node"},
        {},
        root_scope({node_ref_member("trigger"), scope_to_member(std::move(gated))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(stage_node.next_called, 1);
    EXPECT_EQ(stage_node.reset_called, 1);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(stage_node.next_called, 2);
    EXPECT_EQ(stage_node.reset_called, 1); // no re-activation
}

// ----- Sequential scope transitions -----

TEST_F(SchedulerTest, AdvancesOnTransitionFire) {
    auto &trigger = mock("trigger");
    auto &first = mock("first_node");
    auto &second = mock("second_node");
    trigger.param_truthy.insert("output");

    auto first_scope = parallel_scope(
        "first",
        {phase_of({node_ref_member("first_node")})}
    );
    auto second_scope = parallel_scope(
        "second",
        {phase_of({node_ref_member("second_node")})}
    );
    ir::Transition t;
    t.on = ir::Handle{"first_node", "output"};
    t.target = member_key_target("second");
    auto main = sequential_scope(
        "main",
        {scope_to_member(std::move(first_scope)),
         scope_to_member(std::move(second_scope))},
        {t}
    );
    main.activation = ir::Handle{"trigger", "output"};

    auto ir = program_of(
        {"trigger", "first_node", "second_node"},
        {},
        root_scope({node_ref_member("trigger"), scope_to_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(first.next_called, 1);
    EXPECT_EQ(second.next_called, 0);

    first.param_truthy.insert("output");
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(second.next_called, 1);
    EXPECT_EQ(second.reset_called, 1);
}

TEST_F(SchedulerTest, ExitTargetDeactivatesSequence) {
    auto &trigger = mock("trigger");
    auto &first = mock("first_node");
    trigger.param_truthy.insert("output");
    // One-shot: release trigger after cycle 1 so exit is permanent.
    int cycle = 0;
    trigger.on_next = [&cycle, &trigger](const node::Context &ctx) {
        cycle++;
        if (cycle > 1) trigger.param_truthy.erase("output");
    };

    auto first_scope = parallel_scope(
        "first",
        {phase_of({node_ref_member("first_node")})}
    );
    ir::Transition t;
    t.on = ir::Handle{"first_node", "output"};
    t.target = exit_target();
    auto main = sequential_scope(
        "main",
        {scope_to_member(std::move(first_scope))},
        {t}
    );
    main.activation = ir::Handle{"trigger", "output"};
    auto ir = program_of(
        {"trigger", "first_node"},
        {},
        root_scope({node_ref_member("trigger"), scope_to_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    first.param_truthy.insert("output");
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    const int count_at_exit = first.next_called;
    s->next(3 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(first.next_called, count_at_exit);
}

TEST_F(SchedulerTest, FirstMatchWinsWhenMultipleTransitionsTruthy) {
    auto &trigger = mock("trigger");
    auto &first = mock("first_node");
    auto &a = mock("a_node");
    auto &b = mock("b_node");
    trigger.param_truthy.insert("output");

    auto first_scope = parallel_scope(
        "first",
        {phase_of({node_ref_member("first_node")})}
    );
    auto a_scope = parallel_scope("a", {phase_of({node_ref_member("a_node")})});
    auto b_scope = parallel_scope("b", {phase_of({node_ref_member("b_node")})});
    ir::Transition t1;
    t1.on = ir::Handle{"first_node", "output"};
    t1.target = member_key_target("a");
    ir::Transition t2;
    t2.on = ir::Handle{"first_node", "output"};
    t2.target = member_key_target("b");
    auto main = sequential_scope(
        "main",
        {scope_to_member(std::move(first_scope)),
         scope_to_member(std::move(a_scope)),
         scope_to_member(std::move(b_scope))},
        {t1, t2}
    );
    main.activation = ir::Handle{"trigger", "output"};
    auto ir = program_of(
        {"trigger", "first_node", "a_node", "b_node"},
        {},
        root_scope({node_ref_member("trigger"), scope_to_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    first.param_truthy.insert("output");
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 0);
}

TEST_F(SchedulerTest, CascadesMultipleTransitionsInOneCycle) {
    auto &trigger = mock("trigger");
    auto &s1 = mock("s1");
    auto &s2 = mock("s2");
    auto &s3 = mock("s3");
    trigger.param_truthy.insert("output");
    s1.param_truthy.insert("output");
    s2.param_truthy.insert("output");

    auto mk_step = [](const std::string &key, const std::string &node_key) {
        return parallel_scope(key, {phase_of({node_ref_member(node_key)})});
    };
    auto sc1 = mk_step("s1", "s1");
    auto sc2 = mk_step("s2", "s2");
    auto sc3 = mk_step("s3", "s3");
    ir::Transition t1;
    t1.on = ir::Handle{"s1", "output"};
    t1.target = member_key_target("s2");
    ir::Transition t2;
    t2.on = ir::Handle{"s2", "output"};
    t2.target = member_key_target("s3");
    auto main = sequential_scope(
        "main",
        {scope_to_member(std::move(sc1)),
         scope_to_member(std::move(sc2)),
         scope_to_member(std::move(sc3))},
        {t1, t2}
    );
    main.activation = ir::Handle{"trigger", "output"};
    auto ir = program_of(
        {"trigger", "s1", "s2", "s3"},
        {},
        root_scope({node_ref_member("trigger"), scope_to_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(s1.next_called, 1);
    EXPECT_EQ(s2.next_called, 1);
    EXPECT_EQ(s3.next_called, 1);
}

// ----- Error handling -----

TEST_F(SchedulerTest, ContinuesAfterErrorReport) {
    MockErrorHandler h;
    const auto err = x::errors::Error("boom-A", "test");
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    a.on_next = [&err](const node::Context &ctx) { ctx.report_error(err); };
    auto ir = program_of(
        {"A", "B", "C"},
        {},
        root_scope({node_ref_member("A"), node_ref_member("B"), node_ref_member("C")})
    );
    const auto s = build_with_handler(std::move(ir), h.handler);
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
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
        {"A", "B"},
        {},
        root_scope({node_ref_member("A"), node_ref_member("B")})
    );
    const auto s = build_with_handler(std::move(ir), h.handler);
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(h.errors.size(), 2);
}

// ----- Edge cases -----

TEST_F(SchedulerTest, ZeroElapsedTimeAccepted) {
    auto &a = mock("A");
    auto ir = program_of({"A"}, {}, root_scope({node_ref_member("A")}));
    const auto s = build(std::move(ir));
    s->next(x::telem::TimeSpan(0), node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(a.elapsed_values[0], x::telem::TimeSpan(0));
}

TEST_F(SchedulerTest, SelfLoopEdgeDoesNotCrash) {
    auto &a = mock("A");
    a.mark_on_next("output");
    auto ir = program_of(
        {"A"},
        {continuous_edge("A", "output", "A", "in")},
        root_scope({node_ref_member("A")})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
}

TEST_F(SchedulerTest, EmptySequentialScopeTolerated) {
    auto &trigger = mock("trigger");
    trigger.param_truthy.insert("output");
    ir::Scope main;
    main.key = "main";
    main.mode = ir::ScopeMode::Sequential;
    main.liveness = ir::Liveness::Gated;
    main.activation = ir::Handle{"trigger", "output"};
    auto ir = program_of(
        {"trigger"},
        {},
        root_scope({node_ref_member("trigger"), scope_to_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(trigger.next_called, 1);
}

// ----- Complex graph / scope interactions -----

TEST_F(SchedulerTest, IndependentTopLevelGatedScopes) {
    auto &trig_a = mock("trigger_a");
    mock("trigger_b");
    auto &a = mock("A");
    auto &b = mock("B");
    trig_a.param_truthy.insert("output");
    auto stage_a = parallel_scope("stage_a", {phase_of({node_ref_member("A")})});
    stage_a.activation = ir::Handle{"trigger_a", "output"};
    auto stage_b = parallel_scope("stage_b", {phase_of({node_ref_member("B")})});
    stage_b.activation = ir::Handle{"trigger_b", "output"};
    auto ir = program_of(
        {"trigger_a", "trigger_b", "A", "B"},
        {},
        root_scope(
            {node_ref_member("trigger_a"),
             node_ref_member("trigger_b"),
             scope_to_member(std::move(stage_a)),
             scope_to_member(std::move(stage_b))}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 0);
}

TEST_F(SchedulerTest, MixedContinuousAndConditionalInSameGraph) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto &c = mock("C");
    a.on_next = [](const node::Context &ctx) {
        ctx.mark_changed("data");
        ctx.mark_changed("trigger");
    };
    a.param_truthy.insert("trigger");
    auto ir = program_of(
        {"A", "B", "C"},
        {continuous_edge("A", "data", "B", "in"),
         conditional_edge("A", "trigger", "C", "in")},
        root_with_phases(
            {phase_of({node_ref_member("A")}),
             phase_of({node_ref_member("B"), node_ref_member("C")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 1);
}

// ----- External mark injection -----

TEST_F(SchedulerTest, MarkNodeChangedExecutesLaterPhaseMember) {
    auto &a = mock("A");
    auto &b = mock("B");
    auto ir = program_of(
        {"A", "B"},
        {},
        root_with_phases(
            {phase_of({node_ref_member("A")}), phase_of({node_ref_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 0);

    s->mark_node_changed("B");
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 2);
    EXPECT_EQ(b.next_called, 1);
}

}
