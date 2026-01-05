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

/// @brief sequence() should collect nodes from strata into stage nodes list
TEST(BuilderTest, SequenceCollectsNodesFromStrata) {
    auto ir = arc::ir::testutil::Builder()
                  .sequence("main", {{"stage_a", {{"A", "B"}, {"C"}}}})
                  .build();

    ASSERT_EQ(ir.sequences.size(), 1);
    ASSERT_EQ(ir.sequences[0].key, "main");
    ASSERT_EQ(ir.sequences[0].stages.size(), 1);

    const auto &stage = ir.sequences[0].stages[0];
    ASSERT_EQ(stage.key, "stage_a");
    ASSERT_EQ(stage.nodes.size(), 3);
    EXPECT_EQ(stage.nodes[0], "A");
    EXPECT_EQ(stage.nodes[1], "B");
    EXPECT_EQ(stage.nodes[2], "C");
    ASSERT_EQ(stage.strata.size(), 2);
}

/// @brief sequence() should handle multiple stages
TEST(BuilderTest, SequenceHandlesMultipleStages) {
    auto ir = arc::ir::testutil::Builder()
                  .sequence("seq", {{"first", {{"X"}}}, {"second", {{"Y"}, {"Z"}}}})
                  .build();

    ASSERT_EQ(ir.sequences[0].stages.size(), 2);
    ASSERT_EQ(ir.sequences[0].stages[0].nodes.size(), 1);
    EXPECT_EQ(ir.sequences[0].stages[0].nodes[0], "X");
    ASSERT_EQ(ir.sequences[0].stages[1].nodes.size(), 2);
}

/// @brief sequence() should handle empty strata
TEST(BuilderTest, SequenceHandlesEmptyStrata) {
    auto ir = arc::ir::testutil::Builder().sequence("empty", {{"stage", {}}}).build();

    ASSERT_EQ(ir.sequences[0].stages[0].nodes.size(), 0);
}

/// @brief edge() should create continuous edges
TEST(BuilderTest, EdgeCreatesContinuousEdges) {
    auto ir = arc::ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .edge("A", "out", "B", "in")
                  .build();

    ASSERT_EQ(ir.edges.size(), 1);
    EXPECT_EQ(ir.edges[0].kind, arc::ir::EdgeKind::Continuous);
    EXPECT_EQ(ir.edges[0].source.node, "A");
    EXPECT_EQ(ir.edges[0].source.param, "out");
    EXPECT_EQ(ir.edges[0].target.node, "B");
    EXPECT_EQ(ir.edges[0].target.param, "in");
}

/// @brief oneshot() should create one-shot edges
TEST(BuilderTest, OneshotCreatesOneShotEdges) {
    auto ir = arc::ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .oneshot("A", "trigger", "B", "activate")
                  .build();

    ASSERT_EQ(ir.edges.size(), 1);
    EXPECT_EQ(ir.edges[0].kind, arc::ir::EdgeKind::OneShot);
}
