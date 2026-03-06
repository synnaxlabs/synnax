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
#include "arc/cpp/runtime/stable/stable.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::stable {
namespace {
node::Context make_context(
    const x::telem::TimeSpan elapsed,
    const node::RunReason reason = node::RunReason::TimerTick
) {
    return node::Context{
        .elapsed = elapsed,
        .tolerance = x::telem::TimeSpan(0),
        .reason = reason,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };
}

/// @brief Builds an IR with an upstream source node connected to a stable_for node.
struct TestSetup {
    ir::IR ir;
    state::State state;

    explicit TestSetup(const int64_t duration_ns):
        ir(build_ir(duration_ns)),
        state(state::Config{.ir = ir, .channels = {}}, runtime::errors::noop_handler) {}

    state::Node make_stable_node() { return ASSERT_NIL_P(state.node("stable")); }
    state::Node make_source_node() { return ASSERT_NIL_P(state.node("source")); }

private:
    static ir::IR build_ir(const int64_t duration_ns) {
        // Upstream source node with u8 output.
        ir::Param source_output;
        source_output.name = ir::default_output_param;
        source_output.type = types::Type(types::Kind::U8);

        ir::Node source_node;
        source_node.key = "source";
        source_node.type = "producer";
        source_node.outputs.params.push_back(source_output);

        // stable_for node with u8 input and u8 output.
        ir::Param stable_input;
        stable_input.name = ir::default_input_param;
        stable_input.type = types::Type(types::Kind::U8);

        ir::Param stable_output;
        stable_output.name = ir::default_output_param;
        stable_output.type = types::Type(types::Kind::U8);

        ir::Param duration_param;
        duration_param.name = "duration";
        duration_param.type = types::Type(types::Kind::I64);
        duration_param.value = duration_ns;

        ir::Node stable_node;
        stable_node.key = "stable";
        stable_node.type = "stable_for";
        stable_node.inputs.params.push_back(stable_input);
        stable_node.outputs.params.push_back(stable_output);
        stable_node.config.params.push_back(duration_param);

        // Edge from source output to stable input.
        ir::Edge edge;
        edge.source = ir::Handle("source", ir::default_output_param);
        edge.target = ir::Handle("stable", ir::default_input_param);

        ir::Function fn;
        fn.key = "test";

        ir::IR ir;
        ir.nodes.push_back(source_node);
        ir.nodes.push_back(stable_node);
        ir.edges.push_back(edge);
        ir.functions.push_back(fn);
        return ir;
    }
};

/// @brief Helper to write u8 data to the upstream source output.
void write_source(
    state::Node &source,
    const std::vector<uint8_t> &data,
    const std::vector<int64_t> &timestamps
) {
    source.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}
}

/// @brief Test that factory returns NOT_FOUND for non-stable_for node types.
TEST(StableForFactoryTest, ReturnsNotFoundForWrongType) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    auto ir_node = setup.ir.nodes[1];
    ir_node.type = "not_stable_for";

    Factory factory;
    ASSERT_OCCURRED_AS_P(
        factory.create(node::Config(setup.ir, ir_node, setup.make_stable_node())),
        x::errors::NOT_FOUND
    );
}

/// @brief Test that factory creates a StableFor node with valid configuration.
TEST(StableForFactoryTest, CreatesStableForNode) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    Factory factory;
    auto node = ASSERT_NIL_P(factory.create(
        node::Config(setup.ir, setup.ir.nodes[1], setup.make_stable_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that stable_for does not emit before the duration elapses.
TEST(StableForTest, DoesNotEmitBeforeDuration) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // Write value 5 at time 0.
    auto source = setup.make_source_node();
    write_source(source, {5}, {100});

    bool changed = false;
    auto ctx = make_context(x::telem::TimeSpan(0));
    ctx.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_FALSE(changed);

    // Advance to 500ms — still not enough.
    auto ctx2 = make_context(500 * x::telem::MILLISECOND);
    ctx2.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_FALSE(changed);
}

/// @brief Test that stable_for emits when the value has been stable for the duration.
TEST(StableForTest, EmitsWhenStableForDuration) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // Write value 5.
    auto source = setup.make_source_node();
    write_source(source, {5}, {100});

    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    // Advance past the duration.
    bool changed = false;
    std::string changed_param;
    auto ctx2 = make_context(x::telem::SECOND);
    ctx2.mark_changed = [&](const std::string &p) {
        changed = true;
        changed_param = p;
    };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_TRUE(changed);
    EXPECT_EQ(changed_param, "output");

    // Verify the output value.
    auto checker = setup.make_stable_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 5);

    // Verify the output timestamp.
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output_time->at<int64_t>(0), x::telem::SECOND.nanoseconds());
}

/// @brief Test that a value change resets the stability timer.
TEST(StableForTest, ResetsTimerOnValueChange) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // Write value 5 at time 0.
    auto source1 = setup.make_source_node();
    write_source(source1, {5}, {100});
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    // Change to value 10 at 500ms — resets timer.
    auto source2 = setup.make_source_node();
    write_source(source2, {10}, {200});
    auto ctx2 = make_context(500 * x::telem::MILLISECOND);
    ASSERT_NIL(node.next(ctx2));

    // At 1s, only 500ms since the change — should NOT emit.
    bool changed = false;
    auto ctx3 = make_context(x::telem::SECOND);
    ctx3.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx3));
    EXPECT_FALSE(changed);

    // At 1.5s, 1s since the change — should emit.
    auto ctx4 = make_context(x::telem::SECOND + 500 * x::telem::MILLISECOND);
    ctx4.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx4));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 10);
}

/// @brief Test that the same stable value is not emitted twice.
TEST(StableForTest, DoesNotEmitSameValueTwice) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // Write and stabilize.
    auto source = setup.make_source_node();
    write_source(source, {5}, {100});
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    auto ctx2 = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx2));

    // Second call at 2s — same value, should NOT emit again.
    int call_count = 0;
    auto ctx3 = make_context(2 * x::telem::SECOND);
    ctx3.mark_changed = [&](const std::string &) { call_count++; };
    ASSERT_NIL(node.next(ctx3));
    EXPECT_EQ(call_count, 0);
}

/// @brief Test that a new different value emits after stabilizing.
TEST(StableForTest, EmitsDifferentValueAfterStablePeriod) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // First value 5 stabilizes.
    auto source1 = setup.make_source_node();
    write_source(source1, {5}, {100});
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));
    auto ctx2 = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx2));

    // Change to 10 at 2s.
    auto source2 = setup.make_source_node();
    write_source(source2, {10}, {200});
    auto ctx3 = make_context(2 * x::telem::SECOND);
    ASSERT_NIL(node.next(ctx3));

    // At 3s, value 10 should emit.
    bool changed = false;
    auto ctx4 = make_context(3 * x::telem::SECOND);
    ctx4.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx4));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 10);
}

/// @brief Test that multiple values in a single input batch track the last change.
TEST(StableForTest, HandlesMultipleValuesInSingleInput) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // Write batch: [5, 10, 10] — value changes from nil->5, then 5->10.
    auto source = setup.make_source_node();
    write_source(source, {5, 10, 10}, {100, 200, 300});
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    // The last change was at ctx.elapsed=0 (when 5->10 was detected).
    // At 1s, should emit value 10.
    bool changed = false;
    auto ctx2 = make_context(x::telem::SECOND);
    ctx2.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 10);
}

/// @brief Test that reset clears all tracking state.
TEST(StableForTest, ResetClearsState) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // Write and stabilize.
    auto source = setup.make_source_node();
    write_source(source, {5}, {100});
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));
    auto ctx2 = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx2));

    // Reset and check that no output happens without new input.
    node.reset();
    bool changed = false;
    auto ctx3 = make_context(5 * x::telem::SECOND);
    ctx3.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx3));
    EXPECT_FALSE(changed);
}

/// @brief Test that empty input doesn't crash or emit.
TEST(StableForTest, HandlesEmptyInput) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    bool changed = false;
    auto ctx = make_context(x::telem::SECOND);
    ctx.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_FALSE(changed);
}

/// @brief Test that is_output_truthy delegates to state.
TEST(StableForTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // Before any output, should be false.
    EXPECT_FALSE(node.is_output_truthy("output"));

    // Write and stabilize.
    auto source = setup.make_source_node();
    write_source(source, {1}, {100});
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));
    auto ctx2 = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx2));

    EXPECT_TRUE(node.is_output_truthy("output"));
}

/// @brief Test that the output timestamp uses ctx.elapsed (current time), not
/// the input timestamp. Mirrors Go test "Should use output timestamp as current
/// time not input time".
TEST(StableForTest, OutputTimestampUsesCurrentTime) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // Write value 5 at elapsed=0.
    auto source = setup.make_source_node();
    write_source(source, {5}, {100});
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    // Advance elapsed far into the future (100s) — should emit with that time.
    bool changed = false;
    auto ctx2 = make_context(100 * x::telem::SECOND);
    ctx2.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output_time->at<int64_t>(0), (100 * x::telem::SECOND).nanoseconds());
}

/// @brief Test that same value repeated in input uses the first occurrence
/// for the stability timer. Mirrors Go test "Should handle same value repeated
/// in input".
TEST(StableForTest, HandlesSameValueRepeatedInInput) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // Send same value [5, 5, 5, 5] — no change detected after the first,
    // so last_changed stays at the elapsed time of the first call (0).
    auto source = setup.make_source_node();
    write_source(source, {5, 5, 5, 5}, {100, 200, 300, 400});
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));

    // At 1s, should emit — value has been stable since elapsed=0.
    bool changed = false;
    auto ctx2 = make_context(x::telem::SECOND);
    ctx2.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx2));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 5);
}

/// @brief Test that after reset, the same value can be emitted again.
TEST(StableForTest, ResetAllowsSameValueToEmitAgain) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    StableFor node(StableForConfig(setup.ir.nodes[1].config), setup.make_stable_node());

    // First emission of value 5.
    auto source1 = setup.make_source_node();
    write_source(source1, {5}, {100});
    auto ctx1 = make_context(x::telem::TimeSpan(0));
    ASSERT_NIL(node.next(ctx1));
    auto ctx2 = make_context(x::telem::SECOND);
    ASSERT_NIL(node.next(ctx2));

    // Reset.
    node.reset();

    // Write same value 5 again.
    auto source2 = setup.make_source_node();
    write_source(source2, {5}, {200});
    auto ctx3 = make_context(2 * x::telem::SECOND);
    ASSERT_NIL(node.next(ctx3));

    // After reset + stabilization, same value should emit again.
    bool changed = false;
    auto ctx4 = make_context(3 * x::telem::SECOND);
    ctx4.mark_changed = [&](const std::string &) { changed = true; };
    ASSERT_NIL(node.next(ctx4));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 5);
}
}
