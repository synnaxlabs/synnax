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

/// @brief configurable mock node used across scheduler tests. Deals
/// exclusively in ordinals — output names live in ir::Node::outputs and
/// are declared at the IR construction layer (program_of + ir_node).
/// Tests construct mocks via the SchedulerTest::mock helper, which takes
/// a per-ordinal truthy slice that drives both is_output_truthy and the
/// auto-mark loop in next.
struct MockNode final : public node::Node {
    int next_called = 0;
    int reset_called = 0;
    std::vector<x::telem::TimeSpan> elapsed_values;

    /// @brief output_truthy[i] reports whether output ordinal i is
    /// truthy. Drives is_output_truthy and (unless suppress_auto_mark
    /// is set) the auto-mark loop in next. Length need not match the
    /// IR's declared output count — out-of-range ordinals are treated
    /// as non-truthy.
    std::vector<bool> output_truthy;
    /// @brief disables the default behavior of calling MarkChanged for
    /// every currently-truthy output on each next. Tests that want to
    /// model a node whose output stays truthy across cycles but only
    /// announces a change via MarkChanged on specific cycles should set
    /// this and drive MarkChanged manually from on_next.
    bool suppress_auto_mark = false;
    std::function<void(node::Context &)> on_next;

    /// @brief marks the given ordinal as truthy, growing output_truthy
    /// as needed.
    void set_truthy(const size_t ordinal) {
        if (ordinal >= output_truthy.size()) output_truthy.resize(ordinal + 1, false);
        output_truthy[ordinal] = true;
    }

    x::errors::Error next(node::Context &ctx) override {
        next_called++;
        elapsed_values.push_back(ctx.elapsed);
        if (!suppress_auto_mark)
            for (size_t i = 0; i < output_truthy.size(); ++i)
                if (output_truthy[i]) ctx.mark_changed(i);
        if (on_next) on_next(ctx);
        return x::errors::NIL;
    }

    void reset() override { reset_called++; }

    [[nodiscard]] bool is_output_truthy(const size_t output_idx) const override {
        if (output_idx >= output_truthy.size()) return false;
        return output_truthy[output_idx];
    }
};

/// @brief returns an on_next callback that calls mark_changed for the
/// given ordinal each time next runs. Replaces the symbolic
/// mark_on_next("name") form — the ordinal comes from the test's IR
/// declaration.
inline std::function<void(node::Context &)> mark_on_next(const size_t ordinal) {
    return [ordinal](node::Context &ctx) { ctx.mark_changed(ordinal); };
}

/// @brief collects scheduler-reported errors for assertion.
struct MockErrorHandler {
    std::vector<x::errors::Error> errors;
    errors::Handler handler = [this](const x::errors::Error &e) {
        errors.push_back(e);
    };
};

// ----- IR construction helpers -----

static ir::Members stratum_of(std::vector<ir::Member> members) {
    ir::Members s;
    s.reserve(members.size());
    for (auto &m: members)
        s.push_back(std::move(m));
    return s;
}

static ir::Scope parallel_scope(std::string key, std::vector<ir::Members> strata) {
    ir::Scope s;
    s.key = std::move(key);
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Gated;
    s.strata = std::move(strata);
    return s;
}

static ir::Scope always_scope(std::string key, std::vector<ir::Members> strata) {
    ir::Scope s;
    s.key = std::move(key);
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Always;
    s.strata = std::move(strata);
    return s;
}

static ir::Scope sequential_scope(
    std::string key,
    std::vector<ir::Member> steps,
    std::vector<ir::Transition> transitions = {}
) {
    ir::Scope s;
    s.key = std::move(key);
    s.mode = ir::ScopeMode::Sequential;
    s.liveness = ir::Liveness::Gated;
    s.steps = stratum_of(std::move(steps));
    s.transitions = std::move(transitions);
    return s;
}

static ir::Scope root_scope(std::vector<ir::Member> members) {
    ir::Scope s;
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Always;
    if (!members.empty()) s.strata.push_back(stratum_of(std::move(members)));
    return s;
}

static ir::Scope root_with_strata(std::vector<ir::Members> strata) {
    ir::Scope s;
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Always;
    s.strata = std::move(strata);
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

static std::optional<std::string> step_key_target(const std::string &key) {
    return key;
}

static std::optional<std::string> exit_target() {
    return std::nullopt;
}

/// @brief builds an ir::Node with the given key and ordered output
/// names. The IR owns output names; ordinals used by the runtime mock
/// are this list's positions. Pass no names for a node with no outputs.
static ir::Node
ir_node(const std::string &key, std::initializer_list<std::string> outputs = {}) {
    ir::Node n;
    n.key = key;
    for (const auto &name: outputs)
        n.outputs.push_back(arc::types::Param{.name = name});
    return n;
}

/// @brief builds an IR program from the given nodes, edges, and root
/// scope. Output names are declared per node via ir_node — the
/// scheduler reads them exclusively from ir::Node::outputs.
static ir::IR program_of(
    std::initializer_list<ir::Node> nodes,
    std::initializer_list<ir::Edge> edges,
    ir::Scope root
) {
    ir::IR ir;
    for (const auto &n: nodes)
        ir.nodes.push_back(n);
    for (const auto &e: edges)
        ir.edges.push_back(e);
    ir.root = std::move(root);
    return ir;
}

class SchedulerTest : public ::testing::Test {
public:
    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
    std::unordered_map<std::string, MockNode *> mocks;

    /// @brief registers a MockNode under key with per-ordinal initial
    /// truthy values. Pass nothing for a silent mock; pass true/false
    /// per declared output ordinal otherwise. The corresponding ir::Node
    /// and its output names are declared separately at the IR layer
    /// (program_of + ir_node) — the mock is name-agnostic.
    MockNode &mock(const std::string &key, std::initializer_list<bool> truthy = {}) {
        auto node = std::make_unique<MockNode>();
        auto *ptr = node.get();
        if (truthy.size() > 0) ptr->output_truthy.assign(truthy.begin(), truthy.end());
        this->nodes[key] = std::move(node);
        this->mocks[key] = ptr;
        return *ptr;
    }

    std::unique_ptr<Scheduler> build(ir::IR ir) {
        return std::make_unique<Scheduler>(
            std::move(ir),
            this->nodes,
            x::telem::TimeSpan(0)
        );
    }

    std::unique_ptr<Scheduler> build_with_handler(ir::IR ir, errors::Handler handler) {
        return std::make_unique<Scheduler>(
            std::move(ir),
            this->nodes,
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
        {ir_node("A"), ir_node("B"), ir_node("C")},
        {},
        root_scope({ir::node_member("A"), ir::node_member("B"), ir::node_member("C")})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(mocks["A"]->next_called, 1);
    EXPECT_EQ(mocks["B"]->next_called, 1);
    EXPECT_EQ(mocks["C"]->next_called, 1);
}

// ----- Phase-based execution -----

TEST_F(SchedulerTest, Phase0ExecutesUnconditionallyEachCycle) {
    auto &a = mock("A");
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
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
        {ir_node("A", {"output"}), ir_node("B")},
        {continuous_edge("A", "output", "B", "input")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
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
    a.on_next = mark_on_next(0);
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {continuous_edge("A", "output", "B", "input")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
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
    a.on_next = mark_on_next(0);
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {conditional_edge("A", "output", "B", "input")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 0);

    a.set_truthy(0);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(c.next_called, 1);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
}

// ----- Conditional edge lifecycle -----

TEST_F(SchedulerTest, ConditionalFiresEveryCycleWhileTruthy) {
    mock("A", {true});
    auto &b = mock("B");
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {conditional_edge("A", "output", "B", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(3 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 3);
}

TEST_F(SchedulerTest, ConditionalStopsFiringWhenSourceBecomesFalsy) {
    auto &a = mock("A", {true});
    auto &b = mock("B");
    auto ir = program_of(
        {ir_node("A", {"output"}), ir_node("B")},
        {conditional_edge("A", "output", "B", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("B")})}
        )
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);

    a.output_truthy[0] = false;
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 0);
}

// ----- Self-changed replay -----

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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(3 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    s->next(4 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 3);
}

// ----- Context passthrough -----

TEST_F(SchedulerTest, ElapsedTimePassedThrough) {
    auto &a = mock("A");
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
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
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::ChannelInput);
    EXPECT_EQ(received, node::RunReason::ChannelInput);
}

TEST_F(SchedulerTest, NextDeadlineDefaultsToMax) {
    mock("A");
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
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
        {ir_node("A"), ir_node("B")},
        {},
        root_scope({ir::node_member("A"), ir::node_member("B")})
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
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
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
    auto gated = parallel_scope("stage", {stratum_of({ir::node_member("stage_node")})});
    gated.activation = act;
    auto ir = program_of(
        {ir_node("trigger", {"output"}), ir_node("stage_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(gated))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(trigger.next_called, 1);
    EXPECT_EQ(stage_node.next_called, 0);
}

TEST_F(SchedulerTest, GatedScopeActivatesOnceHandleFires) {
    mock("trigger", {true});
    auto &stage_node = mock("stage_node");
    ir::Handle act{"trigger", "output"};
    auto gated = parallel_scope("stage", {stratum_of({ir::node_member("stage_node")})});
    gated.activation = act;
    auto ir = program_of(
        {ir_node("trigger", {"output"}), ir_node("stage_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(gated))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(stage_node.next_called, 1);
    EXPECT_EQ(stage_node.reset_called, 1);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(stage_node.next_called, 2);
    EXPECT_EQ(stage_node.reset_called, 1); // no re-activation
}

// ----- Activation cascading & reset -----

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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    outer.activation = ir::Handle{"trigger", "output"};
    auto ir = program_of(
        {ir_node("trigger", {"output"}), ir_node("inner")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(outer))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(leaf.reset_called, 1);
    EXPECT_EQ(leaf.next_called, 1);
}

// ----- Sequential scope transitions -----

TEST_F(SchedulerTest, AdvancesOnTransitionFire) {
    mock("trigger", {true});
    auto &first = mock("first_node");
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
    main.activation = ir::Handle{"trigger", "output"};

    auto ir = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("first_node", {"output"}),
         ir_node("second_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(first.next_called, 1);
    EXPECT_EQ(second.next_called, 0);

    first.set_truthy(0);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(second.next_called, 1);
    EXPECT_EQ(second.reset_called, 1);
}

TEST_F(SchedulerTest, ExitTargetDeactivatesSequence) {
    auto &trigger = mock("trigger", {true});
    auto &first = mock("first_node");
    // One-shot: release trigger after cycle 1 so exit is permanent.
    int cycle = 0;
    trigger.on_next = [&cycle, &trigger](const node::Context &) {
        cycle++;
        if (cycle > 1) trigger.output_truthy[0] = false;
    };

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
    main.activation = ir::Handle{"trigger", "output"};
    auto ir = program_of(
        {ir_node("trigger", {"output"}), ir_node("first_node", {"output"})},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    first.set_truthy(0);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    const int count_at_exit = first.next_called;
    s->next(3 * x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(first.next_called, count_at_exit);
}

TEST_F(SchedulerTest, FirstMatchWinsWhenMultipleTransitionsTruthy) {
    mock("trigger", {true});
    auto &first = mock("first_node");
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
    main.activation = ir::Handle{"trigger", "output"};
    auto ir = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("first_node", {"output"}),
         ir_node("a_node"),
         ir_node("b_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    first.set_truthy(0);
    s->next(2 * x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    main.activation = ir::Handle{"trigger", "output"};
    auto ir = program_of(
        {ir_node("trigger", {"output"}),
         ir_node("s1", {"output"}),
         ir_node("s2", {"output"}),
         ir_node("s3")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
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
        {ir_node("A"), ir_node("B"), ir_node("C")},
        {},
        root_scope({ir::node_member("A"), ir::node_member("B"), ir::node_member("C")})
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
        {ir_node("A"), ir_node("B")},
        {},
        root_scope({ir::node_member("A"), ir::node_member("B")})
    );
    const auto s = build_with_handler(std::move(ir), h.handler);
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(h.errors.size(), 2);
}

// ----- Edge cases -----

TEST_F(SchedulerTest, ZeroElapsedTimeAccepted) {
    auto &a = mock("A");
    auto ir = program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")}));
    const auto s = build(std::move(ir));
    s->next(x::telem::TimeSpan(0), node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
}

TEST_F(SchedulerTest, EmptySequentialScopeTolerated) {
    auto &trigger = mock("trigger", {true});
    ir::Scope main;
    main.key = "main";
    main.mode = ir::ScopeMode::Sequential;
    main.liveness = ir::Liveness::Gated;
    main.activation = ir::Handle{"trigger", "output"};
    auto ir = program_of(
        {ir_node("trigger", {"output"})},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(ir));
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(trigger.next_called, 1);
}

// ----- Complex graph / scope interactions -----

TEST_F(SchedulerTest, IndependentTopLevelGatedScopes) {
    mock("trigger_a", {true});
    mock("trigger_b");
    auto &a = mock("A");
    auto &b = mock("B");
    auto stage_a = parallel_scope("stage_a", {stratum_of({ir::node_member("A")})});
    stage_a.activation = ir::Handle{"trigger_a", "output"};
    auto stage_b = parallel_scope("stage_b", {stratum_of({ir::node_member("B")})});
    stage_b.activation = ir::Handle{"trigger_b", "output"};
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(b.next_called, 1);
    EXPECT_EQ(c.next_called, 1);
}

// ----- Transitions gated on fresh output marks -----
//
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
    mock("worker");

    auto body = parallel_scope("body", {stratum_of({ir::node_member("worker")})});
    ir::Transition t_exit;
    t_exit.on = ir::Handle{"latch", "output"};
    t_exit.target_key = exit_target();
    auto main = sequential_scope("main", {ir::scope_member(std::move(body))}, {t_exit});
    main.activation = ir::Handle{"trigger", "output"};

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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(mocks["worker"]->next_called, 1);
    EXPECT_EQ(mocks["worker"]->reset_called, 1);
    s->next(2 * x::telem::MICROSECOND, node::RunReason::TimerTick);
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
        if (marks == 1) ctx.mark_changed(0);
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
    main.activation = ir::Handle{"trigger", "output"};

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

    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(mocks["worker_a"]->next_called, 1);
    EXPECT_EQ(mocks["worker_b"]->next_called, 1);

    s->next(2 * x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(mocks["worker_a"]->next_called, 1);
    EXPECT_EQ(mocks["worker_b"]->next_called, 2);

    s->next(3 * x::telem::MICROSECOND, node::RunReason::TimerTick);
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
    main.activation = ir::Handle{"trigger", "output"};

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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(mocks["worker_b"]->next_called, 0);
    s->next(2 * x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(mocks["worker_b"]->next_called, 1);
}

// ----- Settle passes -----

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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 2);
    // B's pass-1 run consumed its flag; no fresh mark, no re-run.
    EXPECT_EQ(b.next_called, 1);
    // The re-pass does not leak into the next cycle.
    s->next(2 * x::telem::MICROSECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    EXPECT_EQ(b.next_called, 1);
    s->next(2 * x::telem::MICROSECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(a.next_called, 1);
    s->next(2 * x::telem::MICROSECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
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
    main.activation = ir::Handle{"trigger", "output"};
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    // The transition fires on pass 1, but second_node waits for the settle
    // pass and runs after V has absorbed first_node's write.
    EXPECT_EQ(
        order,
        (std::vector<std::string>{"trigger", "V", "first", "trigger", "V", "second"})
    );
    EXPECT_EQ(first_node.next_called, 1);
    EXPECT_EQ(second_node.next_called, 1);
}

// ----- Change-flag consumption -----
//
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
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
    main.activation = ir::Handle{"trigger", "output"};
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    // The entry re-runs each pass but marks once; the creator must dispatch
    // exactly once, like a range create in a stage.
    EXPECT_EQ(entry.next_called, 2);
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    // The self-mark lands after consumption, so the re-pass delivers it.
    EXPECT_EQ(looper.next_called, 2);
}

TEST_F(SchedulerTest, DispatchesAgainOnAFreshMarkInTheNextCycle) {
    auto &a = mock("A");
    auto &worker = mock("worker");
    a.on_next = mark_on_next(0);
    auto program = program_of(
        {ir_node("A", {"output"}), ir_node("worker")},
        {continuous_edge("A", "output", "worker", "in")},
        root_with_strata(
            {stratum_of({ir::node_member("A")}),
             stratum_of({ir::node_member("worker")})}
        )
    );
    const auto s = build(std::move(program));
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(worker.next_called, 1);
    s->next(2 * x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(worker.next_called, 2);
}

// ----- Sequential strata variable members -----

TEST_F(SchedulerTest, ResetsASequentialScopesStrataMembersOnActivation) {
    mock("trigger", {true});
    auto &v = mock("V");
    auto &stage_node = mock("M");
    auto stage = parallel_scope("stage", {stratum_of({ir::node_member("M")})});
    auto main = sequential_scope("main", {ir::scope_member(std::move(stage))});
    main.strata.push_back(stratum_of({ir::node_member("V")}));
    main.activation = ir::Handle{"trigger", "output"};
    auto program = program_of(
        {ir_node("trigger", {"output"}), ir_node("V"), ir_node("M")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(program));
    const int base = v.reset_called;
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(v.reset_called, base + 1);
    EXPECT_EQ(stage_node.reset_called, 1);
}

TEST_F(SchedulerTest, ClearsAPendingSelfChangeAndReResetsOnScopeReEntry) {
    auto &trigger = mock("trigger", {true});
    trigger.suppress_auto_mark = true;
    // Activate main on cycles 1 and 3 only.
    trigger.on_next = [&trigger](node::Context &ctx) {
        if (trigger.next_called == 1 || trigger.next_called == 3) ctx.mark_changed(0);
    };
    auto &src = mock("src");
    src.on_next = [&src](node::Context &ctx) {
        if (src.next_called == 1) ctx.mark_changed(0);
    };
    auto &v = mock("V");
    v.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
    auto &stage_node = mock("A");
    auto first = parallel_scope(
        "first",
        {stratum_of({ir::node_member("A")}), stratum_of({ir::node_member("V")})}
    );
    ir::Transition t;
    t.on = ir::Handle{"A", "output"};
    t.target_key = exit_target();
    auto main = sequential_scope("main", {ir::scope_member(std::move(first))}, {t});
    main.activation = ir::Handle{"trigger", "output"};
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
    s->next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(v.reset_called, base + 1);
    EXPECT_EQ(v.next_called, 1);
    // Cycle 2: V replays its self-change and re-marks; A exits main.
    stage_node.set_truthy(0);
    s->next(2 * x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(v.next_called, 2);
    // Cycle 3: re-activation resets V again and clears the pending
    // self-change, so V does not replay.
    s->next(3 * x::telem::MICROSECOND, node::RunReason::TimerTick);
    EXPECT_EQ(v.reset_called, base + 2);
    EXPECT_EQ(v.next_called, 2);
}

TEST_F(SchedulerTest, IgnoresAStrataVariableMemberWithNoMatchingNode) {
    mock("trigger", {true});
    auto &m = mock("M");
    auto stage = parallel_scope("stage", {stratum_of({ir::node_member("M")})});
    auto main = sequential_scope("main", {ir::scope_member(std::move(stage))});
    main.strata.push_back(stratum_of({ir::node_member("ghost")}));
    main.activation = ir::Handle{"trigger", "output"};
    auto program = program_of(
        {ir_node("trigger", {"output"}), ir_node("M")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
    const auto s = build(std::move(program));
    EXPECT_NO_THROW(s->next(x::telem::MICROSECOND, node::RunReason::TimerTick));
    EXPECT_EQ(m.next_called, 1);
}

}
