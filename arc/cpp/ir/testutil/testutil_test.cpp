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

/// @brief sequence() should carry strata through to the stage step
TEST(BuilderTest, SequenceCarriesStrataThrough) {
    auto ir = arc::ir::testutil::Builder()
                  .sequence("main", {{"stage_a", {{"A", "B"}, {"C"}}}})
                  .build();

    ASSERT_EQ(ir.root.sequences.size(), 1);
    ASSERT_EQ(ir.root.sequences[0].key, "main");
    ASSERT_EQ(ir.root.sequences[0].steps.size(), 1);

    const auto &step = ir.root.sequences[0].steps[0];
    ASSERT_NE(step.stage, nullptr);
    ASSERT_EQ(step.key, "stage_a");
    ASSERT_EQ(step.stage->key, "stage_a");
    ASSERT_EQ(step.stage->strata.size(), 2);
    ASSERT_EQ(step.stage->strata[0].size(), 2);
    EXPECT_EQ(step.stage->strata[0][0], "A");
    EXPECT_EQ(step.stage->strata[0][1], "B");
    ASSERT_EQ(step.stage->strata[1].size(), 1);
    EXPECT_EQ(step.stage->strata[1][0], "C");
}

/// @brief sequence() should handle multiple stages
TEST(BuilderTest, SequenceHandlesMultipleStages) {
    auto ir = arc::ir::testutil::Builder()
                  .sequence("seq", {{"first", {{"X"}}}, {"second", {{"Y"}, {"Z"}}}})
                  .build();

    ASSERT_EQ(ir.root.sequences[0].steps.size(), 2);
    ASSERT_EQ(ir.root.sequences[0].steps[0].stage->strata.size(), 1);
    EXPECT_EQ(ir.root.sequences[0].steps[0].stage->strata[0][0], "X");
    ASSERT_EQ(ir.root.sequences[0].steps[1].stage->strata.size(), 2);
}

/// @brief sequence() should handle empty strata
TEST(BuilderTest, SequenceHandlesEmptyStrata) {
    auto ir = arc::ir::testutil::Builder().sequence("empty", {{"stage", {}}}).build();

    ASSERT_EQ(ir.root.sequences[0].steps[0].stage->strata.size(), 0);
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

/// @brief conditional() should create conditional edges
TEST(BuilderTest, ConditionalCreatesConditionalEdges) {
    auto ir = arc::ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .conditional("A", "trigger", "B", "activate")
                  .build();

    ASSERT_EQ(ir.edges.size(), 1);
    EXPECT_EQ(ir.edges[0].kind, arc::ir::EdgeKind::Conditional);
}
