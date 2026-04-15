// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "arc/cpp/ir/testutil/testutil.h"

namespace aiut = arc::ir::testutil;

/// @brief phases() should layer node members across the Root scope's phases.
TEST(BuilderTest, PhasesLayerMembersAcrossRoot) {
    const auto ir = aiut::Builder()
                        .node("A")
                        .node("B")
                        .node("C")
                        .phases({{"A", "B"}, {"C"}})
                        .build();

    EXPECT_EQ(ir.root.mode, arc::ir::ScopeMode::Parallel);
    EXPECT_EQ(ir.root.liveness, arc::ir::Liveness::Always);
    ASSERT_EQ(ir.root.phases.size(), 2);
    ASSERT_EQ(ir.root.phases[0].members.size(), 2);
    ASSERT_TRUE(ir.root.phases[0].members[0].node_ref.has_value());
    EXPECT_EQ(ir.root.phases[0].members[0].node_ref->key, "A");
    EXPECT_EQ(ir.root.phases[1].members[0].node_ref->key, "C");
}

/// @brief sequence() with parallel phase specs should produce a sequential
/// gated scope whose members are parallel gated child scopes.
TEST(BuilderTest, SequenceAppendsSequentialWithParallelChildren) {
    const auto ir = aiut::Builder()
                        .sequence(
                            "main",
                            {
                                aiut::ScopeSpec{
                                    .key = "stage_a",
                                    .phases = {{"A", "B"}, {"C"}},
                                },
                                aiut::ScopeSpec{
                                    .key = "stage_b",
                                    .phases = {{"D"}},
                                },
                            }
                        )
                        .build();

    ASSERT_EQ(ir.root.phases.size(), 1);
    const auto &root_members = ir.root.phases[0].members;
    ASSERT_EQ(root_members.size(), 1);
    ASSERT_NE(root_members[0].scope, nullptr);

    const auto &main = *root_members[0].scope;
    EXPECT_EQ(main.key, "main");
    EXPECT_EQ(main.mode, arc::ir::ScopeMode::Sequential);
    EXPECT_EQ(main.liveness, arc::ir::Liveness::Gated);
    ASSERT_EQ(main.members.size(), 2);

    const auto &stage_a = *main.members[0].scope;
    EXPECT_EQ(stage_a.mode, arc::ir::ScopeMode::Parallel);
    ASSERT_EQ(stage_a.phases.size(), 2);
    ASSERT_EQ(stage_a.phases[0].members.size(), 2);
    EXPECT_EQ(stage_a.phases[0].members[0].node_ref->key, "A");

    const auto &stage_b = *main.members[1].scope;
    ASSERT_EQ(stage_b.phases.size(), 1);
    EXPECT_EQ(stage_b.phases[0].members[0].node_ref->key, "D");
}

/// @brief sequence() with member specs should produce a sequential gated
/// scope whose members are sequential gated child scopes.
TEST(BuilderTest, SequenceAppendsSequentialWithSequentialChildren) {
    const auto ir = aiut::Builder()
                        .sequence(
                            "main",
                            {
                                aiut::ScopeSpec{
                                    .key = "flow_a",
                                    .members = {"N1", "N2"},
                                },
                            }
                        )
                        .build();

    const auto &main = *ir.root.phases[0].members[0].scope;
    ASSERT_EQ(main.members.size(), 1);
    const auto &flow_a = *main.members[0].scope;
    EXPECT_EQ(flow_a.mode, arc::ir::ScopeMode::Sequential);
    ASSERT_EQ(flow_a.members.size(), 2);
    EXPECT_EQ(flow_a.members[0].node_ref->key, "N1");
    EXPECT_EQ(flow_a.members[1].node_ref->key, "N2");
}

/// @brief edge() should create continuous edges.
TEST(BuilderTest, EdgeCreatesContinuousEdges) {
    const auto
        ir = aiut::Builder().node("A").node("B").edge("A", "out", "B", "in").build();

    ASSERT_EQ(ir.edges.size(), 1);
    EXPECT_EQ(ir.edges[0].kind, arc::ir::EdgeKind::Continuous);
    EXPECT_EQ(ir.edges[0].source.node, "A");
    EXPECT_EQ(ir.edges[0].source.param, "out");
    EXPECT_EQ(ir.edges[0].target.node, "B");
    EXPECT_EQ(ir.edges[0].target.param, "in");
}

/// @brief conditional() should create conditional edges.
TEST(BuilderTest, ConditionalCreatesConditionalEdges) {
    const auto ir = aiut::Builder()
                        .node("A")
                        .node("B")
                        .conditional("A", "trigger", "B", "activate")
                        .build();

    ASSERT_EQ(ir.edges.size(), 1);
    EXPECT_EQ(ir.edges[0].kind, arc::ir::EdgeKind::Conditional);
}
