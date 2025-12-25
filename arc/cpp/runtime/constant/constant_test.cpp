// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/constant/constant.h"
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

struct TestSetup {
    arc::ir::IR ir;
    state::State state;
    state::Node node_state;

    TestSetup(const arc::types::Kind kind, const telem::SampleValue &value):
        ir(build_ir(kind, value)),
        state(state::Config{.ir = ir, .channels = {}}),
        node_state(ASSERT_NIL_P(state.node("const"))) {}

private:
    static arc::ir::IR
    build_ir(const arc::types::Kind kind, const telem::SampleValue &value) {
        arc::ir::Param output_param;
        output_param.name = "output";
        output_param.type = arc::types::Type(kind);

        arc::ir::Param value_param;
        value_param.name = "value";
        value_param.type = arc::types::Type(kind);
        value_param.value = value;

        arc::ir::Node ir_node;
        ir_node.key = "const";
        ir_node.type = "constant";
        ir_node.outputs.params.push_back(output_param);
        ir_node.config.params.push_back(value_param);

        arc::ir::Function fn;
        fn.key = "test";

        arc::ir::IR ir;
        ir.nodes.push_back(ir_node);
        ir.functions.push_back(fn);
        return ir;
    }
};
}

/// @brief Test that factory returns NOT_FOUND for non-constant node types.
TEST(ConstantFactoryTest, ReturnsNotFoundForWrongType) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    auto ir_node = setup.ir.nodes[0];
    ir_node.type = "not_constant";
    const node::Config cfg{.node = ir_node, .state = setup.node_state};

    constant::Factory factory;
    ASSERT_OCCURRED_AS_P(factory.create(cfg), xerrors::NOT_FOUND);
}

/// @brief Test that factory creates a Constant node with valid configuration.
TEST(ConstantFactoryTest, CreatesConstantNode) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    const node::Config cfg{.node = setup.ir.nodes[0], .state = setup.node_state};

    constant::Factory factory;
    const auto node = ASSERT_NIL_P(factory.create(cfg));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that next() outputs the constant value on first call.
TEST(ConstantTest, NextOutputsValueOnFirstCall) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.node_state, 42.5f, telem::FLOAT32_T);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    const auto &output = setup.node_state.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_FLOAT_EQ(output->at<float>(0), 42.5f);
}

/// @brief Test that next() is a no-op on subsequent calls.
TEST(ConstantTest, NextNoOpsOnSubsequentCalls) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.node_state, 42.5f, telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    const auto &output = setup.node_state.output(0);
    output->set(0, 999.0f);

    ASSERT_NIL(node.next(ctx));
    EXPECT_FLOAT_EQ(output->at<float>(0), 999.0f);
}

/// @brief Test that reset() allows the value to be output again.
TEST(ConstantTest, ResetAllowsValueToBeOutputAgain) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.node_state, 42.5f, telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    const auto &output = setup.node_state.output(0);
    output->set(0, 999.0f);

    node.reset();
    node.next(ctx);

    EXPECT_FLOAT_EQ(output->at<float>(0), 42.5f);
}

/// @brief Test that float32 values are correctly cast and output.
TEST(ConstantTest, ValueIsCastToCorrectDataType_Float32) {
    const TestSetup setup(arc::types::Kind::F32, 3.14f);
    constant::Constant node(setup.node_state, 3.14f, telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    const auto &output = setup.node_state.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_FLOAT_EQ(output->at<float>(0), 3.14f);
}

/// @brief Test that int64 values are correctly cast and output.
TEST(ConstantTest, ValueIsCastToCorrectDataType_Int64) {
    const TestSetup setup(arc::types::Kind::I64, static_cast<int64_t>(12345));
    constant::Constant node(
        setup.node_state,
        static_cast<int64_t>(12345),
        telem::INT64_T
    );

    auto ctx = make_context();
    node.next(ctx);

    const auto &output = setup.node_state.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<int64_t>(0), 12345);
}

/// @brief Test that uint8 values are correctly cast and output.
TEST(ConstantTest, ValueIsCastToCorrectDataType_U8) {
    const TestSetup setup(arc::types::Kind::U8, static_cast<uint8_t>(255));
    constant::Constant node(
        setup.node_state,
        static_cast<uint8_t>(255),
        telem::UINT8_T
    );

    auto ctx = make_context();
    node.next(ctx);

    const auto &output = setup.node_state.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 255);
}

/// @brief Test that is_output_truthy delegates to state.
TEST(ConstantTest, IsOutputTruthyDelegatesToState) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.node_state, 42.5f, telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    EXPECT_TRUE(node.is_output_truthy("output"));
}

/// @brief Test that mark_changed is called on first next().
TEST(ConstantTest, MarkChangedCalledOnFirstNext) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.node_state, 42.5f, telem::FLOAT32_T);

    bool changed_called = false;
    std::string changed_param;
    auto ctx = make_context();
    ctx.mark_changed = [&](const std::string &param) {
        changed_called = true;
        changed_param = param;
    };

    node.next(ctx);

    EXPECT_TRUE(changed_called);
    EXPECT_EQ(changed_param, "output");
}

/// @brief Test that mark_changed is not called on subsequent next() calls.
TEST(ConstantTest, MarkChangedNotCalledOnSubsequentNext) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.node_state, 42.5f, telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    int call_count = 0;
    ctx.mark_changed = [&](const std::string &) { call_count++; };

    node.next(ctx);
    node.next(ctx);

    EXPECT_EQ(call_count, 0);
}

/// @brief Test that timestamp is populated on first next().
TEST(ConstantTest, TimestampOutputOnFirstNext) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.node_state, 42.5f, telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    const auto &output_time = setup.node_state.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_GT(output_time->at<int64_t>(0), 0);
}

/// @brief Test that reset produces a new timestamp on subsequent next().
TEST(ConstantTest, ResetProducesNewTimestamp) {
    const TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.node_state, 42.5f, telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    const auto &output_time = setup.node_state.output_time(0);
    const auto ts1 = output_time->at<int64_t>(0);

    node.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(1));
    node.next(ctx);

    const auto ts2 = output_time->at<int64_t>(0);
    EXPECT_GT(ts2, ts1);
}
