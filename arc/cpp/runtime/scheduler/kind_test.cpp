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
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/scheduler/testutil/testutil.h"

namespace arc::runtime::scheduler {
using namespace testutil;

/// @brief builds `sequence main { stage first; stage second }` whose transition to
/// second fires on first_node's output with the given kind.
static ir::IR two_step_seq(const ir::EdgeKind kind) {
    auto first = parallel_scope("first", {stratum_of({ir::node_member("first_node")})});
    auto second = parallel_scope(
        "second",
        {stratum_of({ir::node_member("second_node")})}
    );
    ir::Transition t;
    t.on = ir::Handle{"first_node", "output"};
    t.kind = kind;
    t.target_key = step_key_target("second");
    auto main = sequential_scope(
        "main",
        {ir::scope_member(std::move(first)), ir::scope_member(std::move(second))},
        {t}
    );
    main.activations = {
        {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
    };
    return program_of(
        {ir_node("trigger", {"output"}),
         ir_node("first_node", {"output"}),
         ir_node("second_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(main))})
    );
}

/// @brief builds a top-level stage activated by trigger's output with the given kind.
static ir::IR gated_stage(const ir::EdgeKind kind) {
    auto gated = parallel_scope("stage", {stratum_of({ir::node_member("stage_node")})});
    gated.activations = {{.on = ir::Handle{"trigger", "output"}, .kind = kind}};
    return program_of(
        {ir_node("trigger", {"output"}), ir_node("stage_node")},
        {},
        root_scope({ir::node_member("trigger"), ir::scope_member(std::move(gated))})
    );
}

TEST_F(SchedulerTest, ConditionalTransitionIgnoresAFalsyOutput) {
    mock("trigger", {true});
    mock("first_node").on_next = mark_on_next(0);
    const auto &second = mock("second_node");
    const auto s = build(two_step_seq(ir::EdgeKind::Conditional));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(second.next_called, 0);
}

TEST_F(SchedulerTest, ContinuousTransitionFiresOnAFalsyOutput) {
    mock("trigger", {true});
    mock("first_node").on_next = mark_on_next(0);
    const auto &second = mock("second_node");
    const auto s = build(two_step_seq(ir::EdgeKind::Continuous));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(second.next_called, 1);
}

TEST_F(SchedulerTest, ConditionalActivationIgnoresAFalsyOutput) {
    mock("trigger").on_next = mark_on_next(0);
    const auto &stage = mock("stage_node");
    const auto s = build(gated_stage(ir::EdgeKind::Conditional));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(stage.next_called, 0);
}

TEST_F(SchedulerTest, ContinuousActivationFiresOnAFalsyOutput) {
    mock("trigger").on_next = mark_on_next(0);
    const auto &stage = mock("stage_node");
    const auto s = build(gated_stage(ir::EdgeKind::Continuous));
    s->next({.elapsed = x::telem::MICROSECOND, .reason = node::RunReason::TimerTick});
    EXPECT_EQ(stage.next_called, 1);
}

TEST(ValidateTest, AcceptsScopesWithKinds) {
    ASSERT_NIL(validate(two_step_seq(ir::EdgeKind::Continuous).root));
    ASSERT_NIL(validate(gated_stage(ir::EdgeKind::Conditional).root));
}

TEST(ValidateTest, RejectsANestedTransitionWithNoKind) {
    const auto err = validate(two_step_seq(ir::EdgeKind::Unspecified).root);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "scope main has a transition with no kind: on first_node/output ?> second"
    );
}

TEST(ValidateTest, RejectsAnActivationWithNoKind) {
    const auto err = validate(gated_stage(ir::EdgeKind::Unspecified).root);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "scope stage has an activation with no kind");
}
}
