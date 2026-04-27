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
#include "arc/cpp/stl/stable/stable.h"

namespace arc::stl::stable {
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

/// @brief Builds an IR with an upstream source node connected to a stable_for node.
struct TestSetup {
    ir::IR ir;
    runtime::state::State state;

    explicit TestSetup(const int64_t duration_ns):
        ir(build_ir(duration_ns)),
        state(
            runtime::state::Config{.ir = ir, .channels = {}},
            runtime::errors::noop_handler
        ) {}

    runtime::state::Node make_stable_node() {
        return ASSERT_NIL_P(this->state.node("stable"));
    }

    runtime::state::Node make_source_node() {
        return ASSERT_NIL_P(this->state.node("source"));
    }

private:
    static ir::IR build_ir(const int64_t duration_ns) {
        types::Param source_output;
        source_output.name = ir::default_output_param;
        source_output.type = types::Type{.kind = types::Kind::U8};

        ir::Node source_node;
        source_node.key = "source";
        source_node.type = "producer";
        source_node.outputs.push_back(source_output);

        types::Param stable_input;
        stable_input.name = ir::default_input_param;
        stable_input.type = types::Type{.kind = types::Kind::U8};

        types::Param stable_output;
        stable_output.name = ir::default_output_param;
        stable_output.type = types::Type{.kind = types::Kind::U8};

        types::Param duration_param;
        duration_param.name = "duration";
        duration_param.type = types::Type{.kind = types::Kind::I64};
        duration_param.value = duration_ns;

        ir::Node stable_node;
        stable_node.key = "stable";
        stable_node.type = "stable_for";
        stable_node.inputs.push_back(stable_input);
        stable_node.outputs.push_back(stable_output);
        stable_node.config.push_back(duration_param);

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
    runtime::state::Node &source,
    const std::vector<uint8_t> &data,
    const std::vector<int64_t> &timestamps
) {
    source.output(0) = x::mem::make_local_shared<x::telem::Series>(data);
    source.output_time(0) = x::mem::make_local_shared<x::telem::Series>(timestamps);
}

/// @brief Helper that returns a NowFunc capturing a mutable time reference.
x::telem::NowFunc make_now(x::telem::TimeStamp &current_time) {
    return [&current_time]() { return current_time; };
}
}

TEST(StableForConfigTest, CreatesConfigFromValidParams) {
    types::Param duration_param;
    duration_param.name = "duration";
    duration_param.type = types::Type{.kind = types::Kind::I64};
    duration_param.value = x::telem::SECOND.nanoseconds();
    types::Params params;
    params.push_back(duration_param);
    const auto cfg = ASSERT_NIL_P(StableForConfig::create(params));
    EXPECT_EQ(cfg.duration, x::telem::SECOND);
}

TEST(StableForConfigTest, ReturnsErrorForNullDuration) {
    types::Param duration_param;
    duration_param.name = "duration";
    duration_param.type = types::Type{.kind = types::Kind::I64};
    duration_param.value = nullptr;
    types::Params params;
    params.push_back(duration_param);
    ASSERT_OCCURRED_AS_P(StableForConfig::create(params), x::errors::VALIDATION);
}

TEST(StableForModuleTest, ReturnsNotFoundForWrongType) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    auto ir_node = setup.ir.nodes[1];
    ir_node.type = "not_stable_for";

    WasmModule module;
    ASSERT_OCCURRED_AS_P(
        module.create(
            runtime::node::Config(setup.ir, ir_node, setup.make_stable_node())
        ),
        x::errors::NOT_FOUND
    );
}

TEST(StableForModuleTest, CreatesStableForNode) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    WasmModule module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[1], setup.make_stable_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Does not emit before the duration elapses.
TEST(StableForTest, DoesNotEmitBeforeDuration) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    const auto start_ns = x::telem::MILLISECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source(source, {5}, {start_ns});

    bool changed = false;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_FALSE(changed);

    // Advance to 500ms — still not enough.
    current_time = x::telem::TimeStamp(500 * x::telem::MILLISECOND.nanoseconds());
    ASSERT_NIL(node.next(ctx));
    EXPECT_FALSE(changed);
}

/// @brief Emits when the value has been stable for the configured duration.
TEST(StableForTest, EmitsWhenStableForDuration) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    const auto start_ns = x::telem::MILLISECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source(source, {5}, {start_ns});

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    // Advance past the duration (start + 1s).
    const auto emit_ns = start_ns + x::telem::SECOND.nanoseconds();
    current_time = x::telem::TimeStamp(emit_ns);
    std::vector<size_t> marked;
    ctx.mark_changed = [&](size_t i) { marked.push_back(i); };
    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(marked.size(), 1);
    EXPECT_EQ(marked[0], 0);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->size(), 1);
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 5);

    // Output timestamp should be current_time (now), not input timestamp.
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_EQ(output_time->at<int64_t>(0), emit_ns);
}

/// @brief A value change resets the stability timer using the new sample's timestamp.
TEST(StableForTest, ResetsTimerOnValueChange) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    const auto start_ns = x::telem::MILLISECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source(source1, {5}, {start_ns});
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    // Change to value 10 with timestamp 500ms — resets timer.
    auto source2 = setup.make_source_node();
    const auto ts_500ms = 500 * x::telem::MILLISECOND.nanoseconds();
    write_source(source2, {10}, {ts_500ms});
    current_time = x::telem::TimeStamp(ts_500ms);
    ASSERT_NIL(node.next(ctx));

    // At 1s, only 500ms since the change at 500ms — should NOT emit.
    bool changed = false;
    current_time = x::telem::TimeStamp(x::telem::SECOND.nanoseconds());
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_FALSE(changed);

    // At 1.5s, 1s since the change — should emit.
    const auto ts_1500ms = x::telem::SECOND.nanoseconds() +
                           500 * x::telem::MILLISECOND.nanoseconds();
    current_time = x::telem::TimeStamp(ts_1500ms);
    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 10);
}

/// @brief The same stable value is not emitted twice.
TEST(StableForTest, DoesNotEmitSameValueTwice) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    const auto start_ns = x::telem::MILLISECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source(source, {5}, {start_ns});
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    // First emission after stable duration.
    current_time = x::telem::TimeStamp(start_ns + x::telem::SECOND.nanoseconds());
    ASSERT_NIL(node.next(ctx));

    // Second call later — same value, should NOT emit again.
    int call_count = 0;
    current_time = x::telem::TimeStamp(start_ns + 2 * x::telem::SECOND.nanoseconds());
    ctx.mark_changed = [&](size_t) { call_count++; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(call_count, 0);
}

/// @brief A new different value emits after stabilizing.
TEST(StableForTest, EmitsDifferentValueAfterStablePeriod) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    // First value 5 stabilizes.
    const auto start_ns = x::telem::MILLISECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source(source1, {5}, {start_ns});
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
    current_time = x::telem::TimeStamp(start_ns + x::telem::SECOND.nanoseconds());
    ASSERT_NIL(node.next(ctx));

    // Change to 10 with timestamp 2s.
    auto source2 = setup.make_source_node();
    const auto ts_2s = 2 * x::telem::SECOND.nanoseconds();
    write_source(source2, {10}, {ts_2s});
    current_time = x::telem::TimeStamp(ts_2s);
    ASSERT_NIL(node.next(ctx));

    // At 3s, value 10 should emit.
    bool changed = false;
    current_time = x::telem::TimeStamp(3 * x::telem::SECOND.nanoseconds());
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 10);
}

/// @brief Multiple values in a single input batch track the last change timestamp.
TEST(StableForTest, HandlesMultipleValuesInSingleInput) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    // Write batch: [5, 10, 10] with timestamps [100ms, 200ms, 300ms].
    // Value changes from nil->5 at 100ms, then 5->10 at 200ms.
    const auto ms = x::telem::MILLISECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source(source, {5, 10, 10}, {100 * ms, 200 * ms, 300 * ms});
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    // last_changed is at 200ms (when 5->10 occurred). At 1.2s, should emit.
    current_time = x::telem::TimeStamp(1200 * ms);
    bool changed = false;
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 10);
}

/// @brief Reset clears all tracking state.
TEST(StableForTest, ResetClearsState) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    const auto start_ns = x::telem::MILLISECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source(source, {5}, {start_ns});
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
    current_time = x::telem::TimeStamp(start_ns + x::telem::SECOND.nanoseconds());
    ASSERT_NIL(node.next(ctx));

    node.reset();
    bool changed = false;
    current_time = x::telem::TimeStamp(5 * x::telem::SECOND.nanoseconds());
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_FALSE(changed);
}

/// @brief Empty input doesn't crash or emit.
TEST(StableForTest, HandlesEmptyInput) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(x::telem::SECOND.nanoseconds());
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    bool changed = false;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_FALSE(changed);
}

/// @brief is_output_truthy delegates to state.
TEST(StableForTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    EXPECT_FALSE(node.is_output_truthy(0));

    const auto start_ns = x::telem::MILLISECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source(source, {1}, {start_ns});
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
    current_time = x::telem::TimeStamp(start_ns + x::telem::SECOND.nanoseconds());
    ASSERT_NIL(node.next(ctx));

    EXPECT_TRUE(node.is_output_truthy(0));
}

/// @brief Same value repeated in input uses the first occurrence for stability.
TEST(StableForTest, HandlesSameValueRepeatedInInput) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    // Send same value [5, 5, 5, 5] — last_changed stays at first sample's time.
    const auto ms = x::telem::MILLISECOND.nanoseconds();
    auto source = setup.make_source_node();
    write_source(source, {5, 5, 5, 5}, {100 * ms, 200 * ms, 300 * ms, 400 * ms});
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    // last_changed at 100ms. At 1.1s, should emit.
    current_time = x::telem::TimeStamp(1100 * ms);
    bool changed = false;
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 5);
}

/// @brief After reset, the same value can be emitted again.
TEST(StableForTest, ResetAllowsSameValueToEmitAgain) {
    TestSetup setup(x::telem::SECOND.nanoseconds());
    x::telem::TimeStamp current_time(0);
    StableFor node(
        ASSERT_NIL_P(StableForConfig::create(setup.ir.nodes[1].config)),
        setup.make_stable_node(),
        make_now(current_time)
    );

    const auto start_ns = x::telem::MILLISECOND.nanoseconds();
    auto source1 = setup.make_source_node();
    write_source(source1, {5}, {start_ns});
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
    current_time = x::telem::TimeStamp(start_ns + x::telem::SECOND.nanoseconds());
    ASSERT_NIL(node.next(ctx));

    node.reset();

    const auto ts_2s = 2 * x::telem::SECOND.nanoseconds();
    auto source2 = setup.make_source_node();
    write_source(source2, {5}, {ts_2s});
    current_time = x::telem::TimeStamp(ts_2s);
    ASSERT_NIL(node.next(ctx));

    bool changed = false;
    current_time = x::telem::TimeStamp(3 * x::telem::SECOND.nanoseconds());
    ctx.mark_changed = [&](size_t) { changed = true; };
    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed);

    auto checker = setup.make_stable_node();
    EXPECT_EQ(checker.output(0)->at<uint8_t>(0), 5);
}

TEST(StableModuleTest, CreatesNodeWithQualifiedTypeViaMultiFactory) {
    const auto dur = x::telem::SECOND.nanoseconds();
    TestSetup setup(dur);
    auto ir_node = setup.ir.nodes[1];
    ir_node.type = "stable.stable_for";

    auto module = std::make_shared<WasmModule>();
    runtime::node::MultiFactory multi({module});
    auto node = ASSERT_NIL_P(
        multi.create(runtime::node::Config(setup.ir, ir_node, setup.make_stable_node()))
    );
    ASSERT_NE(node, nullptr);
}
}
