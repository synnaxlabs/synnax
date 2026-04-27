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
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/math/math.h"

namespace arc::stl::math {
namespace {

runtime::node::Context make_context() {
    return runtime::node::Context{
        .elapsed = x::telem::TimeSpan(0),
        .tolerance = x::telem::TimeSpan(0),
        .reason = runtime::node::RunReason::TimerTick,
        .mark_changed = [](size_t) {},
        .report_error = [](const x::errors::Error &) {},
    };
}

struct TestSetup {
    ir::IR ir;
    runtime::state::State state;

    TestSetup(
        types::Kind kind,
        const std::string &node_type,
        const std::vector<types::Param> &config_params = {},
        bool with_reset = false,
        types::Kind output_kind = types::Kind::Invalid
    ):
        ir(build_ir(
            kind,
            node_type,
            config_params,
            with_reset,
            output_kind == types::Kind::Invalid ? kind : output_kind
        )),
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
        bool with_reset,
        types::Kind output_kind
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
        target_output.type = types::Type{.kind = output_kind};

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

TEST(MathFlowModuleTest, HandlesMathTypes) {
    FlowModule module;
    EXPECT_TRUE(module.handles("avg"));
    EXPECT_TRUE(module.handles("min"));
    EXPECT_TRUE(module.handles("max"));
    EXPECT_TRUE(module.handles("derivative"));
    EXPECT_FALSE(module.handles("other"));
}

TEST(MathFlowModuleTest, ReturnsNotFoundForWrongType) {
    TestSetup setup(types::Kind::F64, "avg");
    auto ir_node = setup.ir.nodes[1];
    ir_node.type = "unknown";
    FlowModule module;
    ASSERT_OCCURRED_AS_P(
        module.create(
            runtime::node::Config(setup.ir, ir_node, setup.make_target_node())
        ),
        x::errors::NOT_FOUND
    );
}

TEST(MathAvgTest, ComputesRunningAverage) {
    TestSetup setup(types::Kind::F64, "avg");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {10.0, 20.0, 30.0}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    bool changed = false;
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 1);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 20.0);
    EXPECT_EQ(checker.output_time(0)->size(), 1);
    EXPECT_EQ(checker.output_time(0)->at<int64_t>(0), 3 * sec);
}

TEST(MathAvgTest, AccumulatesAcrossBatches) {
    TestSetup setup(types::Kind::F64, "avg");
    FlowModule module;
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

TEST(MathAvgTest, WeightedAverageWithUnequalBatchSizes) {
    TestSetup setup(types::Kind::F64, "avg");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    // Batch 1: [10, 20] -> avg=15, count=2
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {10.0, 20.0}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    // Batch 2: [40] -> sum=40, count=1
    // Correct weighted: (15*2 + 40) / 3 = 23.333...
    // Naive "average of averages": (15 + 40) / 2 = 27.5
    auto source2 = setup.make_source_node();
    write_source_f64(source2, {40.0}, {3 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_NEAR(checker.output(0)->at<double>(0), 23.333, 0.01);
}

TEST(MathAvgTest, ResetsWithCountConfig) {
    types::Param count_param;
    count_param.name = "count";
    count_param.type = types::Type{.kind = types::Kind::I64};
    count_param.value = static_cast<int64_t>(3);

    TestSetup setup(types::Kind::F64, "avg", {count_param});
    FlowModule module;
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

TEST(MathAvgTest, ResetsWithDurationConfig) {
    types::Param duration_param;
    duration_param.name = "duration";
    duration_param.type = types::Type{.kind = types::Kind::I64};
    duration_param.value = 5 * x::telem::SECOND.nanoseconds();

    TestSetup setup(types::Kind::F64, "avg", {duration_param});
    FlowModule module;
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
    write_source_f64(source2, {100.0, 200.0}, {6 * sec, 7 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker2 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker2.output(0)->at<double>(0), 150.0);
}

TEST(MathAvgTest, ResetsWithSignal) {
    TestSetup setup(types::Kind::F64, "avg", {}, true);
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {10.0, 20.0, 30.0}, {sec, 2 * sec, 3 * sec});
    auto reset1 = setup.make_reset_node();
    write_reset(reset1, {0}, {sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker1 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker1.output(0)->at<double>(0), 20.0);

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {100.0, 200.0}, {4 * sec, 5 * sec});
    auto reset2 = setup.make_reset_node();
    write_reset(reset2, {1}, {4 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker2 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker2.output(0)->at<double>(0), 150.0);
}

TEST(MathMinTest, ComputesRunningMinimum) {
    TestSetup setup(types::Kind::I32, "min");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_i32(source, {50, 10, 70}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    bool changed = false;
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 10);
    EXPECT_EQ(checker.output_time(0)->size(), 1);
    EXPECT_EQ(checker.output_time(0)->at<int64_t>(0), 3 * sec);
}

TEST(MathMinTest, MaintainsMinAcrossBatches) {
    TestSetup setup(types::Kind::I32, "min");
    FlowModule module;
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

TEST(MathMinTest, ResetsWithDurationConfig) {
    types::Param duration_param;
    duration_param.name = "duration";
    duration_param.type = types::Type{.kind = types::Kind::I64};
    duration_param.value = 5 * x::telem::SECOND.nanoseconds();

    TestSetup setup(types::Kind::I32, "min", {duration_param});
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();

    // First batch: timestamps [1s, 2s, 3s], duration 5s.
    // No reset: 3s - 1s = 2s < 5s.
    auto source1 = setup.make_source_node();
    write_source_i32(source1, {50, 10, 70}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker1 = setup.make_target_node();
    EXPECT_EQ(checker1.output(0)->at<int32_t>(0), 10);

    // Second batch: timestamps [6s, 7s, 8s].
    // Reset: 6s - 1s = 5s >= 5s (triggers reset).
    // After reset, min should be 40 (min of second batch only).
    auto source2 = setup.make_source_node();
    write_source_i32(source2, {80, 40, 60}, {6 * sec, 7 * sec, 8 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker2 = setup.make_target_node();
    EXPECT_EQ(checker2.output(0)->at<int32_t>(0), 40);
}

TEST(MathMinTest, ResetsWithCountConfig) {
    types::Param count_param;
    count_param.name = "count";
    count_param.type = types::Type{.kind = types::Kind::I64};
    count_param.value = static_cast<int64_t>(3);

    TestSetup setup(types::Kind::F64, "min", {count_param});
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {5.0, 10.0, 15.0}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker1 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker1.output(0)->at<double>(0), 5.0);

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {50.0, 40.0, 30.0}, {4 * sec, 5 * sec, 6 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker2 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker2.output(0)->at<double>(0), 30.0);
}

TEST(MathMinTest, ResetsWithSignal) {
    TestSetup setup(types::Kind::I32, "min", {}, true);
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_i32(source1, {50, 10, 70}, {sec, 2 * sec, 3 * sec});
    auto reset1 = setup.make_reset_node();
    write_reset(reset1, {0}, {sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker1 = setup.make_target_node();
    EXPECT_EQ(checker1.output(0)->at<int32_t>(0), 10);

    auto source2 = setup.make_source_node();
    write_source_i32(source2, {80, 40, 60}, {4 * sec, 5 * sec, 6 * sec});
    auto reset2 = setup.make_reset_node();
    write_reset(reset2, {1}, {4 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker2 = setup.make_target_node();
    EXPECT_EQ(checker2.output(0)->at<int32_t>(0), 40);
}

TEST(MathMinTest, DoesNotUpdateWhenNewValuesLarger) {
    TestSetup setup(types::Kind::F64, "min");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {5.0}, {sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {10.0, 20.0}, {2 * sec, 3 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 5.0);
}

TEST(MathMaxTest, ComputesRunningMaximum) {
    TestSetup setup(types::Kind::F64, "max");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {10.0, 50.0, 30.0}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    bool changed = false;
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 50.0);
    EXPECT_EQ(checker.output_time(0)->size(), 1);
    EXPECT_EQ(checker.output_time(0)->at<int64_t>(0), 3 * sec);
}

TEST(MathMaxTest, MaintainsMaxAcrossBatches) {
    TestSetup setup(types::Kind::F64, "max");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {10.0, 50.0}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {30.0, 20.0}, {3 * sec, 4 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 50.0);
}

TEST(MathMaxTest, ResetsWithDurationConfig) {
    types::Param duration_param;
    duration_param.name = "duration";
    duration_param.type = types::Type{.kind = types::Kind::I64};
    duration_param.value = 5 * x::telem::SECOND.nanoseconds();

    TestSetup setup(types::Kind::F64, "max", {duration_param});
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_f64(source1, {10.0, 50.0, 30.0}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker1 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker1.output(0)->at<double>(0), 50.0);

    auto source2 = setup.make_source_node();
    write_source_f64(source2, {5.0, 15.0}, {6 * sec, 7 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker2 = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker2.output(0)->at<double>(0), 15.0);
}

TEST(MathMaxTest, ResetsWithCountConfig) {
    types::Param count_param;
    count_param.name = "count";
    count_param.type = types::Type{.kind = types::Kind::I64};
    count_param.value = static_cast<int64_t>(2);

    TestSetup setup(types::Kind::I32, "max", {count_param});
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source_i32(source1, {10, 50}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker1 = setup.make_target_node();
    EXPECT_EQ(checker1.output(0)->at<int32_t>(0), 50);

    auto source2 = setup.make_source_node();
    write_source_i32(source2, {5, 15}, {3 * sec, 4 * sec});
    ASSERT_NIL(node->next(ctx));

    auto checker2 = setup.make_target_node();
    EXPECT_EQ(checker2.output(0)->at<int32_t>(0), 15);
}

TEST(MathMaxTest, ResetsWithSignal) {
    TestSetup setup(types::Kind::F64, "max", {}, true);
    FlowModule module;
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

TEST(MathMaxTest, SumsAlignmentFromResetSignal) {
    TestSetup setup(types::Kind::F64, "max", {}, true);
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    auto data_series = x::telem::Series(std::vector<double>{10.0, 20.0, 30.0});
    data_series.alignment = x::telem::Alignment(100);
    data_series.time_range = x::telem::TimeRange(
        x::telem::TimeStamp(50 * sec),
        x::telem::TimeStamp(150 * sec)
    );
    source.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(data_series)
    );
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{50 * sec, 100 * sec, 150 * sec}
    );

    auto reset = setup.make_reset_node();
    auto reset_series = x::telem::Series(static_cast<uint8_t>(0));
    reset_series.alignment = x::telem::Alignment(75);
    reset_series.time_range = x::telem::TimeRange(
        x::telem::TimeStamp(25 * sec),
        x::telem::TimeStamp(175 * sec)
    );
    reset.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(reset_series)
    );
    reset.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{25 * sec}
    );

    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->alignment, x::telem::Alignment(175));
    EXPECT_EQ(checker.output(0)->time_range.start, x::telem::TimeStamp(25 * sec));
    EXPECT_EQ(checker.output(0)->time_range.end, x::telem::TimeStamp(175 * sec));
    EXPECT_EQ(checker.output_time(0)->alignment, x::telem::Alignment(175));
    EXPECT_EQ(checker.output_time(0)->time_range.start, x::telem::TimeStamp(25 * sec));
    EXPECT_EQ(checker.output_time(0)->time_range.end, x::telem::TimeStamp(175 * sec));
}

TEST(MathMaxTest, ExecutesBeforeResetSignalSendsData) {
    TestSetup setup(types::Kind::F64, "max", {}, true);
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {10.0, 50.0, 30.0}, {sec, 2 * sec, 3 * sec});
    // Note: no data written to the reset signal source.
    bool changed = false;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 50.0);
}

TEST(MathDerivativeTest, ComputesPointwiseDerivative) {
    TestSetup setup(types::Kind::F64, "derivative");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    auto data_series = x::telem::Series(std::vector<double>{10.0, 20.0, 40.0});
    data_series.alignment = x::telem::Alignment(250);
    data_series.time_range = x::telem::TimeRange(
        x::telem::TimeStamp(sec),
        x::telem::TimeStamp(4 * sec)
    );
    source.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(data_series)
    );
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{sec, 2 * sec, 4 * sec}
    );
    bool changed = false;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 3);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 0.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), 10.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(2), 10.0);
    EXPECT_EQ(checker.output(0)->alignment, x::telem::Alignment(250));
    EXPECT_EQ(checker.output(0)->time_range.start, x::telem::TimeStamp(sec));
    EXPECT_EQ(checker.output(0)->time_range.end, x::telem::TimeStamp(4 * sec));
    EXPECT_EQ(checker.output_time(0)->alignment, x::telem::Alignment(250));
}

TEST(MathDerivativeTest, MaintainsStateAcrossBatches) {
    TestSetup setup(types::Kind::F64, "derivative");
    FlowModule module;
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

TEST(MathDerivativeTest, FirstSampleOutputsZero) {
    TestSetup setup(types::Kind::F64, "derivative");
    FlowModule module;
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

TEST(MathDerivativeTest, ResetClearsState) {
    TestSetup setup(types::Kind::F64, "derivative");
    FlowModule module;
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

TEST(MathDerivativeTest, ZeroDtOutputsZero) {
    TestSetup setup(types::Kind::F64, "derivative");
    FlowModule module;
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

TEST(MathDerivativeTest, NegativeDerivative) {
    TestSetup setup(types::Kind::F64, "derivative");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {100.0, 80.0, 50.0}, {sec, 2 * sec, 4 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 0.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), -20.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(2), -15.0);
}

TEST(MathDerivativeTest, I32InputOutputsFloat64) {
    TestSetup setup(types::Kind::I32, "derivative", {}, false, types::Kind::F64);
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_i32(source, {0, 100, 300}, {sec, 2 * sec, 4 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 3);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 0.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), 100.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(2), 100.0);
}

TEST(MathDerivativeTest, U8InputNegativeDerivativeOutputsFloat64) {
    TestSetup setup(types::Kind::U8, "derivative", {}, false, types::Kind::F64);
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    source.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<uint8_t>{100, 80, 50}
    );
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{sec, 2 * sec, 4 * sec}
    );
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 3);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 0.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), -20.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(2), -15.0);
}

TEST(MathAvgTest, HandlesEmptyInput) {
    TestSetup setup(types::Kind::F64, "avg");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    bool changed = false;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_FALSE(changed);
}

TEST(MathMinTest, HandlesEmptyInput) {
    TestSetup setup(types::Kind::I32, "min");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    bool changed = false;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_FALSE(changed);
}

TEST(MathMaxTest, HandlesEmptyInput) {
    TestSetup setup(types::Kind::F64, "max");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    bool changed = false;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_FALSE(changed);
}

TEST(MathAvgTest, WorksWithI32) {
    TestSetup setup(types::Kind::I32, "avg");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_i32(source, {10, 20, 30}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 20);
}

TEST(MathMaxTest, WorksWithI32) {
    TestSetup setup(types::Kind::I32, "max");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));

    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_i32(source, {10, 50, 30}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 50);
}

TEST(MathFlowModuleTest, HandlesArithmeticTypes) {
    FlowModule module;
    EXPECT_TRUE(module.handles("add"));
    EXPECT_TRUE(module.handles("subtract"));
    EXPECT_TRUE(module.handles("multiply"));
    EXPECT_TRUE(module.handles("divide"));
    EXPECT_TRUE(module.handles("mod"));
    EXPECT_TRUE(module.handles("neg"));
}

/// BinaryTestSetup builds an IR with two source nodes (lhs, rhs) wired into
/// a target binary-operation node.
struct BinaryTestSetup {
    ir::IR ir;
    runtime::state::State state;

    BinaryTestSetup(types::Kind kind, const std::string &node_type):
        ir(build_ir(kind, node_type)),
        state(
            runtime::state::Config{.ir = ir, .channels = {}},
            runtime::errors::noop_handler
        ) {}

    runtime::state::Node make_target_node() {
        return ASSERT_NIL_P(this->state.node("target"));
    }

    runtime::state::Node make_lhs_node() {
        return ASSERT_NIL_P(this->state.node("lhs"));
    }

    runtime::state::Node make_rhs_node() {
        return ASSERT_NIL_P(this->state.node("rhs"));
    }

private:
    static ir::IR build_ir(types::Kind kind, const std::string &node_type) {
        types::Param lhs_output;
        lhs_output.name = ir::default_output_param;
        lhs_output.type = types::Type{.kind = kind};

        ir::Node lhs_node;
        lhs_node.key = "lhs";
        lhs_node.type = "producer";
        lhs_node.outputs.push_back(lhs_output);

        types::Param rhs_output;
        rhs_output.name = ir::default_output_param;
        rhs_output.type = types::Type{.kind = kind};

        ir::Node rhs_node;
        rhs_node.key = "rhs";
        rhs_node.type = "producer";
        rhs_node.outputs.push_back(rhs_output);

        types::Param target_lhs;
        target_lhs.name = ir::lhs_input_param;
        target_lhs.type = types::Type{.kind = kind};

        types::Param target_rhs;
        target_rhs.name = ir::rhs_input_param;
        target_rhs.type = types::Type{.kind = kind};

        types::Param target_output;
        target_output.name = ir::default_output_param;
        target_output.type = types::Type{.kind = kind};

        ir::Node target_node;
        target_node.key = "target";
        target_node.type = node_type;
        target_node.inputs.push_back(target_lhs);
        target_node.inputs.push_back(target_rhs);
        target_node.outputs.push_back(target_output);

        ir::Edge lhs_edge;
        lhs_edge.source = ir::Handle("lhs", ir::default_output_param);
        lhs_edge.target = ir::Handle("target", ir::lhs_input_param);

        ir::Edge rhs_edge;
        rhs_edge.source = ir::Handle("rhs", ir::default_output_param);
        rhs_edge.target = ir::Handle("target", ir::rhs_input_param);

        ir::Function fn;
        fn.key = "test";

        ir::IR ir;
        ir.nodes.push_back(lhs_node);
        ir.nodes.push_back(rhs_node);
        ir.nodes.push_back(target_node);
        ir.edges.push_back(lhs_edge);
        ir.edges.push_back(rhs_edge);
        ir.functions.push_back(fn);
        return ir;
    }
};

void write_lhs_f64(
    runtime::state::Node &lhs,
    const std::vector<double> &data,
    const std::vector<int64_t> &timestamps
) {
    lhs.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    lhs.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}

void write_rhs_f64(
    runtime::state::Node &rhs,
    const std::vector<double> &data,
    const std::vector<int64_t> &timestamps
) {
    rhs.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    rhs.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}

void write_lhs_i32(
    runtime::state::Node &lhs,
    const std::vector<int32_t> &data,
    const std::vector<int64_t> &timestamps
) {
    lhs.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    lhs.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}

void write_rhs_i32(
    runtime::state::Node &rhs,
    const std::vector<int32_t> &data,
    const std::vector<int64_t> &timestamps
) {
    rhs.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    rhs.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}

// ───────────────── Add ─────────────────

TEST(MathAddTest, AddsF64ElementWise) {
    BinaryTestSetup setup(types::Kind::F64, "add");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_f64(lhs, {1.5, 2.5, 3.5}, {sec, 2 * sec, 3 * sec});
    write_rhs_f64(rhs, {0.5, 1.5, 2.5}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    bool changed = false;
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 3);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 2.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), 4.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(2), 6.0);
}

TEST(MathAddTest, AddsI32ElementWise) {
    BinaryTestSetup setup(types::Kind::I32, "add");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_i32(lhs, {10, 20, 30}, {sec, 2 * sec, 3 * sec});
    write_rhs_i32(rhs, {1, 2, 3}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 11);
    EXPECT_EQ(checker.output(0)->at<int32_t>(1), 22);
    EXPECT_EQ(checker.output(0)->at<int32_t>(2), 33);
}

// ───────────────── Subtract ─────────────────

TEST(MathSubtractTest, SubtractsF64ElementWise) {
    BinaryTestSetup setup(types::Kind::F64, "subtract");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_f64(lhs, {10.0, 20.0, 30.0}, {sec, 2 * sec, 3 * sec});
    write_rhs_f64(rhs, {3.0, 5.0, 7.0}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 7.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), 15.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(2), 23.0);
}

TEST(MathSubtractTest, SubtractsI32ElementWise) {
    BinaryTestSetup setup(types::Kind::I32, "subtract");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_i32(lhs, {100, 200, 300}, {sec, 2 * sec, 3 * sec});
    write_rhs_i32(rhs, {25, 50, 75}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 75);
    EXPECT_EQ(checker.output(0)->at<int32_t>(1), 150);
    EXPECT_EQ(checker.output(0)->at<int32_t>(2), 225);
}

// ───────────────── Multiply ─────────────────

TEST(MathMultiplyTest, MultipliesF64ElementWise) {
    BinaryTestSetup setup(types::Kind::F64, "multiply");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_f64(lhs, {2.5, 3.0}, {sec, 2 * sec});
    write_rhs_f64(rhs, {4.0, 5.0}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 10.0);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), 15.0);
}

TEST(MathMultiplyTest, MultipliesI32ElementWise) {
    BinaryTestSetup setup(types::Kind::I32, "multiply");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_i32(lhs, {4, 5}, {sec, 2 * sec});
    write_rhs_i32(rhs, {5, 6}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 20);
    EXPECT_EQ(checker.output(0)->at<int32_t>(1), 30);
}

// ───────────────── Divide ─────────────────

TEST(MathDivideTest, DividesF64ElementWise) {
    BinaryTestSetup setup(types::Kind::F64, "divide");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_f64(lhs, {10.0, 21.0}, {sec, 2 * sec});
    write_rhs_f64(rhs, {4.0, 7.0}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), 2.5);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), 3.0);
}

TEST(MathDivideTest, DividesI32ElementWise) {
    BinaryTestSetup setup(types::Kind::I32, "divide");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_i32(lhs, {20, 100}, {sec, 2 * sec});
    write_rhs_i32(rhs, {4, 10}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 5);
    EXPECT_EQ(checker.output(0)->at<int32_t>(1), 10);
}

// ───────────────── Mod ─────────────────

TEST(MathModTest, ModsF64ElementWise) {
    BinaryTestSetup setup(types::Kind::F64, "mod");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_f64(lhs, {10.5, 20.5}, {sec, 2 * sec});
    write_rhs_f64(rhs, {3.0, 6.0}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_NEAR(checker.output(0)->at<double>(0), 1.5, 1e-9);
    EXPECT_NEAR(checker.output(0)->at<double>(1), 2.5, 1e-9);
}

TEST(MathModTest, ModsI32ElementWise) {
    BinaryTestSetup setup(types::Kind::I32, "mod");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_i32(lhs, {10, 15, 23}, {sec, 2 * sec, 3 * sec});
    write_rhs_i32(rhs, {3, 4, 5}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), 1);
    EXPECT_EQ(checker.output(0)->at<int32_t>(1), 3);
    EXPECT_EQ(checker.output(0)->at<int32_t>(2), 3);
}

// ───────────────── Neg ─────────────────

TEST(MathNegTest, NegatesF64ElementWise) {
    TestSetup setup(types::Kind::F64, "neg");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_f64(source, {1.5, -2.5, 3.5}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    bool changed = false;
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(changed);
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 3);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(0), -1.5);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(1), 2.5);
    EXPECT_DOUBLE_EQ(checker.output(0)->at<double>(2), -3.5);
}

TEST(MathNegTest, NegatesI32ElementWise) {
    TestSetup setup(types::Kind::I32, "neg");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source_i32(source, {10, -20, 30}, {sec, 2 * sec, 3 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->at<int32_t>(0), -10);
    EXPECT_EQ(checker.output(0)->at<int32_t>(1), 20);
    EXPECT_EQ(checker.output(0)->at<int32_t>(2), -30);
}

// ───────────────── Edge Cases ─────────────────

TEST(MathArithmeticTest, HandlesMismatchedSeriesLengths) {
    BinaryTestSetup setup(types::Kind::F64, "add");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    const auto sec = x::telem::SECOND.nanoseconds();
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    write_lhs_f64(
        lhs,
        {1.0, 2.0, 3.0, 4.0, 5.0},
        {sec, 2 * sec, 3 * sec, 4 * sec, 5 * sec}
    );
    write_rhs_f64(rhs, {10.0, 20.0}, {sec, 2 * sec});
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->size(), 5);
}

TEST(MathArithmeticTest, NoChangeWhenInputsNotRefreshed) {
    BinaryTestSetup setup(types::Kind::F64, "add");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    auto ctx = make_context();
    bool changed = false;
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node->next(ctx));
    EXPECT_FALSE(changed);
}

TEST(MathArithmeticTest, PropagatesAlignmentFromBothInputs) {
    BinaryTestSetup setup(types::Kind::F64, "add");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[2], setup.make_target_node())
    ));
    auto lhs = setup.make_lhs_node();
    auto rhs = setup.make_rhs_node();
    auto lhs_data = x::mem::make_local_shared<x::telem::Series>(
        std::vector<double>{10.0, 20.0}
    );
    lhs_data->alignment = x::telem::Alignment(100);
    lhs_data->time_range = x::telem::TimeRange(10000, 30000);
    lhs.output(0) = lhs_data;
    lhs.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{10000, 20000}
    );
    auto rhs_data = x::mem::make_local_shared<x::telem::Series>(
        std::vector<double>{5.0, 10.0}
    );
    rhs_data->alignment = x::telem::Alignment(50);
    rhs_data->time_range = x::telem::TimeRange(5000, 25000);
    rhs.output(0) = rhs_data;
    rhs.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{5000, 15000}
    );
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->alignment, x::telem::Alignment(150));
    EXPECT_EQ(checker.output(0)->time_range.start, x::telem::TimeStamp(5000));
    EXPECT_EQ(checker.output(0)->time_range.end, x::telem::TimeStamp(30000));
    EXPECT_EQ(checker.output_time(0)->alignment, x::telem::Alignment(150));
}

TEST(MathArithmeticTest, NegPropagatesAlignmentFromInput) {
    TestSetup setup(types::Kind::F64, "neg");
    FlowModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_target_node())
    ));
    auto source = setup.make_source_node();
    auto src_data = x::mem::make_local_shared<x::telem::Series>(
        std::vector<double>{10.0, 20.0}
    );
    src_data->alignment = x::telem::Alignment(200);
    src_data->time_range = x::telem::TimeRange(100000, 300000);
    source.output(0) = src_data;
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{100000, 200000}
    );
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    auto checker = setup.make_target_node();
    EXPECT_EQ(checker.output(0)->alignment, x::telem::Alignment(200));
    EXPECT_EQ(checker.output(0)->time_range.start, x::telem::TimeStamp(100000));
    EXPECT_EQ(checker.output(0)->time_range.end, x::telem::TimeStamp(300000));
}

// ───────────────── MultiFactory Qualified Routing ─────────────────

TEST(MathFlowModuleTest, CreatesAddWithQualifiedTypeViaMultiFactory) {
    BinaryTestSetup setup(types::Kind::F64, "add");
    auto ir_node = setup.ir.nodes[2];
    ir_node.type = "math.add";
    auto module = std::make_shared<FlowModule>();
    runtime::node::MultiFactory multi({module});
    auto node = ASSERT_NIL_P(
        multi.create(runtime::node::Config(setup.ir, ir_node, setup.make_target_node()))
    );
    ASSERT_NE(node, nullptr);
}

TEST(MathFlowModuleTest, CreatesNegWithQualifiedTypeViaMultiFactory) {
    TestSetup setup(types::Kind::F64, "neg");
    auto ir_node = setup.ir.nodes[1];
    ir_node.type = "math.neg";
    auto module = std::make_shared<FlowModule>();
    runtime::node::MultiFactory multi({module});
    auto node = ASSERT_NIL_P(
        multi.create(runtime::node::Config(setup.ir, ir_node, setup.make_target_node()))
    );
    ASSERT_NE(node, nullptr);
}

TEST(MathFlowModuleTest, CreatesNodeWithQualifiedTypeViaMultiFactory) {
    TestSetup setup(types::Kind::F64, "avg");
    auto ir_node = setup.ir.nodes[1];
    ir_node.type = "math.avg";

    auto module = std::make_shared<FlowModule>();
    runtime::node::MultiFactory multi({module});
    auto node = ASSERT_NIL_P(
        multi.create(runtime::node::Config(setup.ir, ir_node, setup.make_target_node()))
    );
    ASSERT_NE(node, nullptr);
}

TEST(MathFlowModuleTest, CreatesDerivativeWithQualifiedTypeViaMultiFactory) {
    TestSetup setup(types::Kind::F64, "derivative", {}, false, types::Kind::F64);
    auto ir_node = setup.ir.nodes[1];
    ir_node.type = "math.derivative";

    auto module = std::make_shared<FlowModule>();
    runtime::node::MultiFactory multi({module});
    auto node = ASSERT_NIL_P(
        multi.create(runtime::node::Config(setup.ir, ir_node, setup.make_target_node()))
    );
    ASSERT_NE(node, nullptr);
}
}
