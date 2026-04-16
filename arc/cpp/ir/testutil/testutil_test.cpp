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

/// @brief strata() should layer node members across the Root scope's strata.
TEST(BuilderTest, StrataLayerMembersAcrossRoot) {
    const auto ir = aiut::Builder()
                        .node("A")
                        .node("B")
                        .node("C")
                        .strata({{"A", "B"}, {"C"}})
                        .build();

    EXPECT_EQ(ir.root.mode, arc::ir::ScopeMode::Parallel);
    EXPECT_EQ(ir.root.liveness, arc::ir::Liveness::Always);
    ASSERT_EQ(ir.root.strata.size(), 2);
    ASSERT_EQ(ir.root.strata[0].size(), 2);
    ASSERT_TRUE(ir.root.strata[0][0].node_key.has_value());
    EXPECT_EQ(*ir.root.strata[0][0].node_key, "A");
    EXPECT_EQ(*ir.root.strata[1][0].node_key, "C");
}

/// @brief sequence() with parallel stratum specs should produce a sequential
/// gated scope whose steps are parallel gated child scopes.
TEST(BuilderTest, SequenceAppendsSequentialWithParallelChildren) {
    const auto ir = aiut::Builder()
                        .sequence(
                            "main",
                            {
                                aiut::ScopeSpec{
                                    .key = "stage_a",
                                    .strata = {{"A", "B"}, {"C"}},
                                },
                                aiut::ScopeSpec{
                                    .key = "stage_b",
                                    .strata = {{"D"}},
                                },
                            }
                        )
                        .build();

    ASSERT_EQ(ir.root.strata.size(), 1);
    const auto &root_members = ir.root.strata[0];
    ASSERT_EQ(root_members.size(), 1);
    ASSERT_NE(root_members[0].scope, nullptr);

    const auto &main = *root_members[0].scope;
    EXPECT_EQ(main.key, "main");
    EXPECT_EQ(main.mode, arc::ir::ScopeMode::Sequential);
    EXPECT_EQ(main.liveness, arc::ir::Liveness::Gated);
    ASSERT_EQ(main.steps.size(), 2);

    const auto &stage_a = *main.steps[0].scope;
    EXPECT_EQ(stage_a.mode, arc::ir::ScopeMode::Parallel);
    ASSERT_EQ(stage_a.strata.size(), 2);
    ASSERT_EQ(stage_a.strata[0].size(), 2);
    EXPECT_EQ(*stage_a.strata[0][0].node_key, "A");

    const auto &stage_b = *main.steps[1].scope;
    ASSERT_EQ(stage_b.strata.size(), 1);
    EXPECT_EQ(*stage_b.strata[0][0].node_key, "D");
}

/// @brief sequence() with step specs should produce a sequential gated
/// scope whose steps are sequential gated child scopes.
TEST(BuilderTest, SequenceAppendsSequentialWithSequentialChildren) {
    const auto ir = aiut::Builder()
                        .sequence(
                            "main",
                            {
                                aiut::ScopeSpec{
                                    .key = "flow_a",
                                    .steps = {"N1", "N2"},
                                },
                            }
                        )
                        .build();

    const auto &main = *ir.root.strata[0][0].scope;
    ASSERT_EQ(main.steps.size(), 1);
    const auto &flow_a = *main.steps[0].scope;
    EXPECT_EQ(flow_a.mode, arc::ir::ScopeMode::Sequential);
    ASSERT_EQ(flow_a.steps.size(), 2);
    EXPECT_EQ(*flow_a.steps[0].node_key, "N1");
    EXPECT_EQ(*flow_a.steps[1].node_key, "N2");
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
