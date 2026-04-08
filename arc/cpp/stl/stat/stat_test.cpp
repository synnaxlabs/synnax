// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/mem/local_shared.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stat/stat.h"

namespace arc::stl::stat {
namespace {

runtime::node::Context make_context() {
    return runtime::node::Context{
        .elapsed = x::telem::TimeSpan(0),
        .tolerance = x::telem::TimeSpan(0),
        .reason = runtime::node::RunReason::TimerTick,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };
}

struct TestSetup {
    ir::IR ir;
    runtime::state::State state;

    TestSetup(
        types::Kind kind,
        const std::string &node_type,
        const std::vector<types::Param> &config_params = {},
        bool with_reset = false
    ):
        ir(build_ir(kind, node_type, config_params, with_reset)),
        state(
            runtime::state::Config{.ir = ir, .channels = {}},
            runtime::errors::noop_handler
        ) {}

    runtime::state::Node make_target_node() {
        return ASSERT_NIL_P(this->state.node("target"));
    }

    runtime::state::Node make_source_node() {
        return ASSERT_NIL_P(this->state.node("source"));
    }

    runtime::state::Node make_reset_node() {
        return ASSERT_NIL_P(this->state.node("reset_signal"));
    }

private:
    static ir::IR build_ir(
        types::Kind kind,
        const std::string &node_type,
        const std::vector<types::Param> &config_params,
        bool with_reset
    ) {
        types::Param source_output;
        source_output.name = ir::default_output_param;
        source_output.type = types::Type{.kind = kind};

        ir::Node source_node;
        source_node.key = "source";
        source_node.type = "producer";
        source_node.outputs.push_back(source_output);

        types::Param target_input;
        target_input.name = ir::default_input_param;
        target_input.type = types::Type{.kind = kind};

        types::Param target_output;
        target_output.name = ir::default_output_param;
        target_output.type = types::Type{.kind = kind};

        ir::Node target_node;
        target_node.key = "target";
        target_node.type = node_type;
        target_node.inputs.push_back(target_input);
        target_node.outputs.push_back(target_output);
        for (const auto &p: config_params)
            target_node.config.push_back(p);

        ir::Edge edge;
        edge.source = ir::Handle("source", ir::default_output_param);
        edge.target = ir::Handle("target", ir::default_input_param);

        ir::Function fn;
        fn.key = "test";

        ir::IR ir;
        ir.nodes.push_back(source_node);
        ir.nodes.push_back(target_node);
        ir.edges.push_back(edge);

        if (with_reset) {
            types::Param reset_output;
            reset_output.name = ir::default_output_param;
            reset_output.type = types::Type{.kind = types::Kind::U8};

            ir::Node reset_node;
            reset_node.key = "reset_signal";
            reset_node.type = "producer";
            reset_node.outputs.push_back(reset_output);

            types::Param reset_input;
            reset_input.name = "reset";
            reset_input.type = types::Type{.kind = types::Kind::U8};
            target_node.inputs.push_back(reset_input);
            ir.nodes[1] = target_node;

            ir::Edge reset_edge;
            reset_edge.source = ir::Handle("reset_signal", ir::default_output_param);
            reset_edge.target = ir::Handle("target", "reset");

            ir.nodes.push_back(reset_node);
            ir.edges.push_back(reset_edge);
        }

        ir.functions.push_back(fn);
        return ir;
    }
};

void write_source_f64(
    runtime::state::Node &source,
    const std::vector<double> &data,
    const std::vector<int64_t> &timestamps
) {
    source.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}

void write_source_i32(
    runtime::state::Node &source,
    const std::vector<int32_t> &data,
    const std::vector<int64_t> &timestamps
) {
    source.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}

void write_reset(
    runtime::state::Node &reset,
    const std::vector<uint8_t> &data,
    const std::vector<int64_t> &timestamps
) {
    reset.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    reset.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}
}

// ─── Module ──────────────────────────────────────────────────────────────────

TEST(StatModuleTest, HandlesStatTypes) {
    Module module;
    EXPECT_TRUE(module.handles("avg"));
    EXPECT_TRUE(module.handles("min"));
    EXPECT_TRUE(module.handles("max"));
    EXPECT_TRUE(module.handles("derivative"));
    EXPECT_FALSE(module.handles("other"));
}

TEST(StatModuleTest, ReturnsNotFoundForWrongType) {
    TestSetup setup(types::Kind::F64, "avg");
    auto ir_node = setup.ir.nodes[1];
    ir_node.type = "not_stat";
    Module module;
    ASSERT_OCCURRED_AS_P(
        module.create(
            runtime::node::Config(setup.ir, ir_node, setup.make_target_node())
        ),
        x::errors::NOT_FOUND
    );
}

// ─── Avg ─────────────────────────────────────────────────────────────────────

TEST(StatAvgTest, ComputesRunningAverage) {
    TestSetup setup(types::Kind::F64, "avg");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {10.0, 20.0, 30.0}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    bool changed = false;
    ctx.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 1);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 20.0);
}

TEST(StatAvgTest, AccumulatesAcrossBatches) {
    TestSetup setup(types::Kind::F64, "avg");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {10.0, 20.0}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {30.0}, {3 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 20.0);
}

TEST(StatAvgTest, ResetsWithCountConfig) {
    types::Param count_param;
    count_param.name = "count";
    count_param.type = types::Type{.kind = types::Kind::I64};
    count_param.value = static_cast<int64_t>(3);

    TestSetup setup(types::Kind::F64, "avg", {count_param});
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {10.0, 20.0, 30.0}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker1 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker1.output(0)->at<double>(0), 20.0);

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {40.0, 50.0, 60.0}, {4 * sec, 5 * sec, 6 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker2 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker2.output(0)->at<double>(0), 50.0);
}

// ─── Min ─────────────────────────────────────────────────────────────────────

TEST(StatMinTest, ComputesRunningMinimum) {
    TestSetup setup(types::Kind::I32, "min");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_i32(source, {50, 10, 70}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    bool changed = false;
    ctx.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 10);
}

TEST(StatMinTest, MaintainsMinAcrossBatches) {
    TestSetup setup(types::Kind::I32, "min");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_i32(source1, {50, 30}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto source2 = setup.make_source_node();
    write_source_i32(source2, {40, 60}, {3 * sec, 4 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 30);
}

// ─── Max ─────────────────────────────────────────────────────────────────────

TEST(StatMaxTest, ComputesRunningMaximum) {
    TestSetup setup(types::Kind::F64, "max");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {10.0, 50.0, 30.0}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    bool changed = false;
    ctx.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 50.0);
}

TEST(StatMaxTest, ResetsWithSignal) {
    TestSetup setup(types::Kind::F64, "max", {}, true);
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {10.0, 50.0, 30.0}, {sec, 2 * sec, 3 * sec});
    auto reset1 = setup.make_reset_node();
    write_reset(reset1, {0}, {sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker1 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker1.output(0)->at<double>(0), 50.0);

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {5.0, 15.0}, {4 * sec, 5 * sec});
    auto reset2 = setup.make_reset_node();
    write_reset(reset2, {1}, {4 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker2 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker2.output(0)->at<double>(0), 15.0);
}

// ─── Derivative ──────────────────────────────────────────────────────────────

TEST(StatDerivativeTest, ComputesPointwiseDerivative) {
    TestSetup setup(types::Kind::F64, "derivative");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {10.0, 20.0, 40.0}, {sec, 2 * sec, 4 * sec});
    bool changed = false;
    auto ctx = make_context();
    ctx.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 3);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 0.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), 10.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(2), 10.0);
}

TEST(StatDerivativeTest, MaintainsStateAcrossBatches) {
    TestSetup setup(types::Kind::F64, "derivative");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {0.0, 10.0}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {30.0}, {4 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 1);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 10.0);
}

TEST(StatDerivativeTest, FirstSampleOutputsZero) {
    TestSetup setup(types::Kind::F64, "derivative");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {5.0}, {sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 0.0);
}

TEST(StatDerivativeTest, ResetClearsState) {
    TestSetup setup(types::Kind::F64, "derivative");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {10.0, 20.0}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    node->reset();

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {100.0}, {10 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 0.0);
}

TEST(StatDerivativeTest, ZeroDtOutputsZero) {
    TestSetup setup(types::Kind::F64, "derivative");
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {10.0, 20.0}, {sec, sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 0.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), 0.0);
}
}
