// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/stage/stage.h"
#include "arc/cpp/runtime/state/state.h"

using namespace arc::runtime;

namespace {
node::Context make_context() {
    return node::Context{
        .elapsed = telem::SECOND,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const xerrors::Error &) {},
        .activate_stage = [] {},
    };
}

arc::ir::IR build_ir() {
    arc::ir::Node ir_node;
    ir_node.key = "entry";
    ir_node.type = "stage_entry";

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);
    ir.functions.push_back(fn);
    return ir;
}
}

/// @brief Verify factory correctly identifies stage_entry nodes.
TEST(StageFactoryTest, HandlesStageEntryType) {
    const stage::Factory factory;
    EXPECT_TRUE(factory.handles("stage_entry"));
}

/// @brief Verify factory rejects non-stage_entry node types.
TEST(StageFactoryTest, RejectsOtherTypes) {
    const stage::Factory factory;
    EXPECT_FALSE(factory.handles("constant"));
    EXPECT_FALSE(factory.handles("timer"));
    EXPECT_FALSE(factory.handles(""));
}

/// @brief Verify factory creates a valid StageEntry node.
TEST(StageFactoryTest, CreatesStageEntryNode) {
    auto ir = build_ir();
    state::State state(state::Config{.ir = ir, .channels = {}});
    auto state_node = ASSERT_NIL_P(state.node("entry"));

    stage::Factory factory;
    auto node = ASSERT_NIL_P(factory.create(node::Config(ir.nodes[0], std::move(state_node))));
    ASSERT_NE(node, nullptr);
}

/// @brief Verify next() calls activate_stage on the context.
TEST(StageEntryTest, NextActivatesStage) {
    stage::StageEntry entry;

    bool activated = false;
    auto ctx = make_context();
    ctx.activate_stage = [&activated] { activated = true; };

    entry.next(ctx);

    EXPECT_TRUE(activated);
}

/// @brief Verify next() returns nil error.
TEST(StageEntryTest, NextReturnsNil) {
    stage::StageEntry entry;
    auto ctx = make_context();
    ASSERT_NIL(entry.next(ctx));
}

/// @brief Verify is_output_truthy always returns false regardless of parameter.
TEST(StageEntryTest, IsOutputTruthyAlwaysFalse) {
    const stage::StageEntry entry;
    EXPECT_FALSE(entry.is_output_truthy("output"));
    EXPECT_FALSE(entry.is_output_truthy("anything"));
    EXPECT_FALSE(entry.is_output_truthy(""));
}
