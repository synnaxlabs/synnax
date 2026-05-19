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
#include "arc/cpp/stl/selector/selector.h"

namespace arc::stl::selector {
namespace {
runtime::node::Context make_context() {
    return runtime::node::Context{
        .elapsed = x::telem::TimeSpan(0),
        .tolerance = x::telem::TimeSpan(0),
        .reason = runtime::node::RunReason::ChannelInput,
        .mark_changed = [](size_t) {},
        .report_error = [](const x::errors::Error &) {},
    };
}

/// @brief Builds an IR with a source node connected to a select node.
/// The select node has one input and two outputs ("true" at index 0,
/// "false" at index 1).
struct TestSetup {
    ir::IR ir;
    runtime::state::State state;

    TestSetup():
        ir(build_ir()),
        state(
            runtime::state::Config{.ir = ir, .channels = {}},
            runtime::errors::noop_handler
        ) {}

    runtime::state::Node make_select_node() {
        return ASSERT_NIL_P(this->state.node("select"));
    }

    runtime::state::Node make_source_node() {
        return ASSERT_NIL_P(this->state.node("source"));
    }

private:
    static ir::IR build_ir() {
        types::Param source_output;
        source_output.name = ir::default_output_param;
        source_output.type = types::Type{.kind = types::Kind::U8};

        ir::Node source_node;
        source_node.key = "source";
        source_node.type = "producer";
        source_node.outputs.push_back(source_output);

        types::Param select_input;
        select_input.name = ir::default_input_param;
        select_input.type = types::Type{.kind = types::Kind::U8};

        types::Param true_output;
        true_output.name = "true";
        true_output.type = types::Type{.kind = types::Kind::U8};

        types::Param false_output;
        false_output.name = "false";
        false_output.type = types::Type{.kind = types::Kind::U8};

        ir::Node select_node;
        select_node.key = "select";
        select_node.type = "select";
        select_node.inputs.push_back(select_input);
        select_node.outputs.push_back(true_output);
        select_node.outputs.push_back(false_output);

        ir::Edge edge;
        edge.source = ir::Handle("source", ir::default_output_param);
        edge.target = ir::Handle("select", ir::default_input_param);

        ir::Function fn;
        fn.key = "test";

        ir::IR ir;
        ir.nodes.push_back(source_node);
        ir.nodes.push_back(select_node);
        ir.edges.push_back(edge);
        ir.functions.push_back(fn);
        return ir;
    }
};

/// @brief Helper to write u8 data to the upstream source output.
void write_source(
    runtime::state::Node &source,
    const std::vector<uint8_t> &data,
    const std::vector<int64_t> &timestamps
) {
    source.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}
}

/// @brief Test that module returns NOT_FOUND for non-select node types.
TEST(SelectModuleTest, ReturnsNotFoundForWrongType) {
    TestSetup setup;
    auto ir_node = setup.ir.nodes[1];
    ir_node.type = "not_select";

    Module module;
    ASSERT_OCCURRED_AS_P(
        module.create(
            runtime::node::Config(setup.ir, ir_node, setup.make_select_node())
        ),
        x::errors::NOT_FOUND
    );
}

/// @brief Test that module creates a Select node with valid configuration.
TEST(SelectModuleTest, CreatesSelectNode) {
    TestSetup setup;
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_select_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that no input produces no output.
TEST(SelectTest, HandlesNoInput) {
    TestSetup setup;
    Select node(setup.make_select_node());

    bool changed = false;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_FALSE(changed);
}

/// @brief Test that all-true input routes entirely to true output.
TEST(SelectTest, AllTrueInput) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {1, 1, 1}, {100, 200, 300});

    std::set<size_t> changed_params;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t i) { changed_params.insert(i); };
    ASSERT_NIL(node.next(ctx));

    EXPECT_TRUE(changed_params.contains(TRUE_OUTPUT_IDX));
    EXPECT_FALSE(changed_params.contains(FALSE_OUTPUT_IDX));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(0)->size(), 3);
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 1);
    EXPECT_EQ(checker.output(0)->at<uint8_t>(1), 1);
    EXPECT_EQ(checker.output(0)->at<uint8_t>(2), 1);
    EXPECT_EQ(checker.output(1)->size(), 0);
}

/// @brief Test that all-false input routes entirely to false output.
TEST(SelectTest, AllFalseInput) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {0, 0, 0}, {100, 200, 300});

    std::set<size_t> changed_params;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t i) { changed_params.insert(i); };
    ASSERT_NIL(node.next(ctx));

    EXPECT_FALSE(changed_params.contains(TRUE_OUTPUT_IDX));
    EXPECT_TRUE(changed_params.contains(FALSE_OUTPUT_IDX));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(0)->size(), 0);
    EXPECT_EQ(checker.output(1)->size(), 3);
    EXPECT_EQ(checker.output(1)->at<uint8_t>(0), 1);
    EXPECT_EQ(checker.output(1)->at<uint8_t>(1), 1);
    EXPECT_EQ(checker.output(1)->at<uint8_t>(2), 1);
}

/// @brief Test that mixed input is split correctly.
TEST(SelectTest, MixedInput) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {1, 0, 1, 0}, {100, 200, 300, 400});

    std::set<size_t> changed_params;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t i) { changed_params.insert(i); };
    ASSERT_NIL(node.next(ctx));

    EXPECT_TRUE(changed_params.contains(TRUE_OUTPUT_IDX));
    EXPECT_TRUE(changed_params.contains(FALSE_OUTPUT_IDX));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(0)->size(), 2);
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 1);
    EXPECT_EQ(checker.output(0)->at<uint8_t>(1), 1);
    EXPECT_EQ(checker.output(1)->size(), 2);
    EXPECT_EQ(checker.output(1)->at<uint8_t>(0), 1);
    EXPECT_EQ(checker.output(1)->at<uint8_t>(1), 1);
}

/// @brief Test that true output timestamps match source timestamps.
TEST(SelectTest, TrueTimestampsMatchSource) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {0, 1, 0, 1}, {100, 200, 300, 400});

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_select_node();
    const auto &true_time = checker.output_time(0);
    EXPECT_EQ(true_time->size(), 2);
    EXPECT_EQ(true_time->at<int64_t>(0), 200);
    EXPECT_EQ(true_time->at<int64_t>(1), 400);
}

/// @brief Test that false output timestamps match source timestamps.
TEST(SelectTest, FalseTimestampsMatchSource) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {0, 1, 0, 1}, {100, 200, 300, 400});

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_select_node();
    const auto &false_time = checker.output_time(1);
    EXPECT_EQ(false_time->size(), 2);
    EXPECT_EQ(false_time->at<int64_t>(0), 100);
    EXPECT_EQ(false_time->at<int64_t>(1), 300);
}

/// @brief Test single true value. Mirrors Go test "Should handle single true value".
TEST(SelectTest, SingleTrueValue) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {1}, {100});

    std::set<size_t> changed_params;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t i) { changed_params.insert(i); };
    ASSERT_NIL(node.next(ctx));

    EXPECT_TRUE(changed_params.contains(TRUE_OUTPUT_IDX));
    EXPECT_FALSE(changed_params.contains(FALSE_OUTPUT_IDX));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(0)->size(), 1);
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 1);
}

/// @brief Test single false value. Mirrors Go test "Should handle single false
/// value".
TEST(SelectTest, SingleFalseValue) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {0}, {100});

    std::set<size_t> changed_params;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t i) { changed_params.insert(i); };
    ASSERT_NIL(node.next(ctx));

    EXPECT_FALSE(changed_params.contains(TRUE_OUTPUT_IDX));
    EXPECT_TRUE(changed_params.contains(FALSE_OUTPUT_IDX));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(1)->size(), 1);
    EXPECT_EQ(checker.output(1)->at<uint8_t>(0), 1);
}

/// @brief Test long series (1000 elements). Mirrors Go test "Should handle
/// long series".
TEST(SelectTest, LongSeries) {
    TestSetup setup;
    Select node(setup.make_select_node());

    std::vector<uint8_t> data(1000);
    std::vector<int64_t> timestamps(1000);
    for (size_t i = 0; i < 1000; i++) {
        data[i] = static_cast<uint8_t>(i % 2);
        timestamps[i] = static_cast<int64_t>(i) * 1000;
    }

    auto source = setup.make_source_node();
    write_source(source, data, timestamps);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(0)->size(), 500);
    EXPECT_EQ(checker.output(1)->size(), 500);
}

/// @brief Test consecutive true values with correct timestamps.
/// Mirrors Go test "Should handle consecutive true values".
TEST(SelectTest, ConsecutiveTrueValues) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {0, 0, 1, 1, 1, 0}, {100, 200, 300, 400, 500, 600});

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(0)->size(), 3);
    const auto &true_time = checker.output_time(0);
    EXPECT_EQ(true_time->at<int64_t>(0), 300);
    EXPECT_EQ(true_time->at<int64_t>(1), 400);
    EXPECT_EQ(true_time->at<int64_t>(2), 500);
}

/// @brief Test consecutive false values with correct timestamps.
/// Mirrors Go test "Should handle consecutive false values".
TEST(SelectTest, ConsecutiveFalseValues) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {1, 1, 0, 0, 0, 1}, {100, 200, 300, 400, 500, 600});

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(1)->size(), 3);
    const auto &false_time = checker.output_time(1);
    EXPECT_EQ(false_time->at<int64_t>(0), 300);
    EXPECT_EQ(false_time->at<int64_t>(1), 400);
    EXPECT_EQ(false_time->at<int64_t>(2), 500);
}

/// @brief Test that non-one values (e.g., 2, 255) are treated as false.
TEST(SelectTest, NonOneValuesAreFalse) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    write_source(source, {2, 255, 1, 0}, {100, 200, 300, 400});

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(0)->size(), 1); // Only value 1
    EXPECT_EQ(checker.output(1)->size(), 3); // Values 2, 255, 0
}

/// @brief Test that is_output_truthy delegates to state.
TEST(SelectTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup;
    Select node(setup.make_select_node());

    // Before any output, should be false.
    EXPECT_FALSE(node.is_output_truthy(TRUE_OUTPUT_IDX));
    EXPECT_FALSE(node.is_output_truthy(FALSE_OUTPUT_IDX));
}

/// @brief Test that alignment and time_range are propagated to outputs.
TEST(SelectTest, PropagatesAlignmentAndTimeRange) {
    TestSetup setup;
    Select node(setup.make_select_node());

    auto source = setup.make_source_node();
    auto data = x::mem::make_local_shared<x::telem::Series>(std::vector<uint8_t>{1, 0});
    data->alignment = x::telem::Alignment(3, 10);
    data->time_range = x::telem::TimeRange(1000, 2000);
    source.output(0) = data;
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{100, 200}
    );

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_select_node();
    EXPECT_EQ(checker.output(0)->alignment, x::telem::Alignment(3, 10));
    EXPECT_EQ(checker.output(0)->time_range, x::telem::TimeRange(1000, 2000));
    EXPECT_EQ(checker.output_time(0)->alignment, x::telem::Alignment(3, 10));
    EXPECT_EQ(checker.output_time(0)->time_range, x::telem::TimeRange(1000, 2000));
    EXPECT_EQ(checker.output(1)->alignment, x::telem::Alignment(3, 10));
    EXPECT_EQ(checker.output(1)->time_range, x::telem::TimeRange(1000, 2000));
    EXPECT_EQ(checker.output_time(1)->alignment, x::telem::Alignment(3, 10));
    EXPECT_EQ(checker.output_time(1)->time_range, x::telem::TimeRange(1000, 2000));
}

}
