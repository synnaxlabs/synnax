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

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/scheduler/testutil/testutil.h"

namespace arc::runtime::scheduler {
using namespace testutil;

namespace {
const node::Cycle JUMP_CYCLE{
    .elapsed = x::telem::MICROSECOND,
    .reason = node::RunReason::TimerTick
};

/// @brief activates a scope on trigger's truthy output.
std::vector<ir::Activation> triggered() {
    return {{.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}};
}

/// @brief returns a conditional transition on key's output that exits to target.
ir::Transition jump(const std::string &key, const std::string &target) {
    ir::Transition t;
    t.on = ir::Handle{key, "output"};
    t.kind = ir::EdgeKind::Conditional;
    t.target = ir::ScopeTarget{.key = target};
    return t;
}

/// @brief builds `sequence main { stage s { ...keys } }`, activated by trigger and
/// holding the given transitions.
ir::Scope main_seq(
    const std::vector<std::string> &keys,
    std::vector<ir::Transition> transitions
) {
    std::vector<ir::Member> members;
    for (const auto &k: keys)
        members.push_back(ir::node_member(k));
    auto s = parallel_scope("s", {stratum_of(std::move(members))});
    auto main = sequential_scope(
        "main",
        {ir::scope_member(std::move(s))},
        std::move(transitions)
    );
    main.activations = triggered();
    return main;
}

/// @brief builds a gated top-level stage holding one node.
ir::Scope target(const std::string &key) {
    return parallel_scope(key, {stratum_of({ir::node_member(key + "_node")})});
}

/// @brief puts trigger first in the root, then scopes in the given order.
ir::IR
jump_program(const std::vector<std::string> &node_keys, std::vector<ir::Scope> scopes) {
    ir::IR prog;
    prog.nodes.push_back(ir_node("trigger", {"output"}));
    for (const auto &k: node_keys)
        prog.nodes.push_back(ir_node(k, {"output"}));
    std::vector<ir::Member> members{ir::node_member("trigger")};
    for (auto &sc: scopes)
        members.push_back(ir::scope_member(std::move(sc)));
    prog.root = root_scope(std::move(members));
    return prog;
}
}

class JumpTest : public SchedulerTest {
protected:
    /// @brief registers a node that runs on every cycle while its scope is active.
    MockNode &alive(const std::string &key) {
        auto &m = this->mock(key);
        m.on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
        return m;
    }
};

TEST_F(JumpTest, ExitsAndStartsATargetDeclaredBeforeInTheSameCycle) {
    mock("trigger", {true});
    const auto &body = alive("body");
    mock("j", {true});
    const auto &abort = mock("abort_node");
    std::vector<ir::Scope> scopes;
    scopes.push_back(target("abort"));
    scopes.push_back(main_seq({"body", "j"}, {jump("j", "abort")}));
    const auto s = build(jump_program({"body", "j", "abort_node"}, std::move(scopes)));
    s->next(JUMP_CYCLE);
    EXPECT_EQ(abort.next_called, 1);
    s->next(JUMP_CYCLE);
    EXPECT_EQ(body.next_called, 1);
}

TEST_F(JumpTest, ActsOnlyOnTheFirstJumpThatFires) {
    mock("trigger", {true});
    mock("j1", {true});
    mock("j2", {true});
    const auto &abort = mock("abort_node");
    const auto &flight = mock("flight_node");
    std::vector<ir::Scope> scopes;
    scopes.push_back(
        main_seq({"j1", "j2"}, {jump("j1", "abort"), jump("j2", "flight")})
    );
    scopes.push_back(target("abort"));
    scopes.push_back(target("flight"));
    const auto s = build(
        jump_program({"j1", "j2", "abort_node", "flight_node"}, std::move(scopes))
    );

    s->next(JUMP_CYCLE);
    EXPECT_EQ(abort.next_called, 1);
    EXPECT_EQ(flight.next_called, 0);
}

TEST_F(JumpTest, ExitsAParallelScope) {
    mock("trigger", {true});
    const auto &body = alive("body");
    mock("j", {true});
    const auto &flight = mock("flight_node");
    auto idle = parallel_scope(
        "idle",
        {stratum_of({ir::node_member("body"), ir::node_member("j")})}
    );
    idle.activations = triggered();
    idle.transitions = {jump("j", "flight")};
    std::vector<ir::Scope> scopes;
    scopes.push_back(std::move(idle));
    scopes.push_back(target("flight"));
    const auto s = build(jump_program({"body", "j", "flight_node"}, std::move(scopes)));
    s->next(JUMP_CYCLE);
    EXPECT_EQ(flight.next_called, 1);
    s->next(JUMP_CYCLE);
    EXPECT_EQ(body.next_called, 1);
}

TEST_F(JumpTest, ActsOnlyOnTheFirstExitOfAParallelScope) {
    mock("trigger", {true});
    mock("j1", {true});
    mock("j2", {true});
    const auto &abort = mock("abort_node");
    const auto &flight = mock("flight_node");
    auto idle = parallel_scope(
        "idle",
        {stratum_of({ir::node_member("j1"), ir::node_member("j2")})}
    );
    idle.activations = triggered();
    idle.transitions = {jump("j1", "abort"), jump("j2", "flight")};
    std::vector<ir::Scope> scopes;
    scopes.push_back(std::move(idle));
    scopes.push_back(target("abort"));
    scopes.push_back(target("flight"));
    const auto s = build(
        jump_program({"j1", "j2", "abort_node", "flight_node"}, std::move(scopes))
    );

    s->next(JUMP_CYCLE);
    EXPECT_EQ(abort.next_called, 1);
    EXPECT_EQ(flight.next_called, 0);
}

TEST_F(JumpTest, RestartsAScopeThatJumpsToItself) {
    mock("trigger", {true});
    const auto &body = mock("body");
    auto &j = mock("j", {true});
    j.suppress_auto_mark = true;
    j.on_next = [&j](node::Context &ctx) {
        if (j.next_called == 1) ctx.mark_changed(0);
    };
    std::vector<ir::Scope> scopes;
    scopes.push_back(main_seq({"body", "j"}, {jump("j", "main")}));
    const auto s = build(jump_program({"body", "j"}, std::move(scopes)));

    s->next(JUMP_CYCLE);
    EXPECT_EQ(body.next_called, 2);
    EXPECT_EQ(body.reset_called, 2);
}

TEST_F(JumpTest, DoesNotActOnALosingJumpAfterItsScopeRestarts) {
    mock("trigger", {true});
    for (const auto &key: {"j1", "j2"}) {
        auto &j = mock(key, {true});
        j.suppress_auto_mark = true;
        j.on_next = [&j](node::Context &ctx) {
            if (j.next_called == 1) ctx.mark_changed(0);
        };
    }
    const auto &flight = mock("flight_node");
    std::vector<ir::Scope> scopes;
    scopes.push_back(
        main_seq({"j1", "j2"}, {jump("j1", "main"), jump("j2", "flight")})
    );
    scopes.push_back(target("flight"));
    const auto s = build(jump_program({"j1", "j2", "flight_node"}, std::move(scopes)));

    s->next(JUMP_CYCLE);
    EXPECT_EQ(flight.next_called, 0);
}

TEST_F(JumpTest, StopsTheScopesNestedInAScopeThatRestarts) {
    mock("trigger", {true});
    auto &start = mock("start", {true});
    start.suppress_auto_mark = true;
    start.on_next = [&start](node::Context &ctx) {
        if (start.next_called == 1) ctx.mark_changed(0);
    };
    auto &j = mock("j", {true});
    j.suppress_auto_mark = true;
    j.on_next = [&j](node::Context &ctx) {
        ctx.mark_self_changed();
        if (j.next_called == 2) ctx.mark_changed(0);
    };
    const auto &body = alive("body");
    auto inner = parallel_scope("inner", {stratum_of({ir::node_member("body")})});
    inner.activations = {
        {.on = ir::Handle{"start", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    auto outer = parallel_scope(
        "outer",
        {stratum_of(
            {ir::node_member("start"),
             ir::node_member("j"),
             ir::scope_member(std::move(inner))}
        )}
    );
    outer.activations = triggered();
    outer.transitions = {jump("j", "outer")};
    std::vector<ir::Scope> scopes;
    scopes.push_back(std::move(outer));
    const auto s = build(jump_program({"start", "j", "body"}, std::move(scopes)));
    for (int i = 0; i < 3; ++i)
        s->next(JUMP_CYCLE);
    EXPECT_EQ(body.next_called, 2);
}

TEST_F(JumpTest, DoesNotRestartATargetThatIsAlreadyActive) {
    mock("trigger", {true});
    mock("j", {true});
    const auto &abort = mock("abort_node");
    auto running = target("abort");
    running.activations = triggered();
    std::vector<ir::Scope> scopes;
    scopes.push_back(std::move(running));
    scopes.push_back(main_seq({"j"}, {jump("j", "abort")}));
    const auto s = build(jump_program({"j", "abort_node"}, std::move(scopes)));

    s->next(JUMP_CYCLE);
    EXPECT_EQ(abort.next_called, 1);
    EXPECT_EQ(abort.reset_called, 1);
}

class ActivationTest : public SchedulerTest,
                       public ::testing::WithParamInterface<std::string> {};

TEST_P(ActivationTest, ActivatesAScopeFromEachOfItsActivations) {
    const auto &firing = GetParam();
    mock("a1", {firing == "a1"});
    mock("a2", {firing == "a2"});
    const auto &stage = mock("stage_node");
    auto gated = target("stage");
    gated.activations = {
        {.on = ir::Handle{"a1", "output"}, .kind = ir::EdgeKind::Conditional},
        {.on = ir::Handle{"a2", "output"}, .kind = ir::EdgeKind::Conditional},
    };
    const auto s = build(program_of(
        {ir_node("a1", {"output"}), ir_node("a2", {"output"}), ir_node("stage_node")},
        {},
        root_scope(
            {ir::node_member("a1"),
             ir::node_member("a2"),
             ir::scope_member(std::move(gated))}
        )
    ));

    s->next(JUMP_CYCLE);
    EXPECT_EQ(stage.next_called, 1);
}

INSTANTIATE_TEST_SUITE_P(Jumps, ActivationTest, ::testing::Values("a1", "a2"));

/// @brief returns the validation error for a top-level scope s that holds t.
static x::errors::Error
validate_transition(ir::Transition t, const ir::ScopeMode mode) {
    auto s = target("s");
    s.mode = mode;
    s.transitions = {std::move(t)};
    std::vector<ir::Scope> scopes;
    scopes.push_back(std::move(s));
    return validate(jump_program({"s_node"}, std::move(scopes)).root);
}

TEST(ValidateTest, RejectsAJumpToAScopeThatIsNotTopLevel) {
    const auto err = validate_transition(
        jump("s_node", "missing"),
        ir::ScopeMode::Parallel
    );
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "scope s has a transition to a scope that is not top-level: "
        "on s_node/output => exit to missing"
    );
}

TEST(ValidateTest, RejectsAStepTargetOnAParallelScope) {
    ir::Transition t;
    t.on = ir::Handle{"s_node", "output"};
    t.kind = ir::EdgeKind::Conditional;
    t.target = ir::StepTarget{.key = "other"};
    const auto err = validate_transition(t, ir::ScopeMode::Parallel);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "scope s has a transition with a step target it cannot take: "
        "on s_node/output => other"
    );
}

TEST(ValidateTest, AcceptsAJumpToATopLevelScope) {
    ASSERT_NIL(validate_transition(jump("s_node", "s"), ir::ScopeMode::Parallel));
}
}
