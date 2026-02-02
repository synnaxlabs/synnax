// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/constant/constant.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/state/state.h"

using namespace arc::runtime;

namespace {
node::Context make_context() {
    return node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };
}

struct TestSetup {
    arc::ir::IR ir;
    state::State state;

    TestSetup(const arc::types::Kind kind, const x::json::json &value):
        ir(build_ir(kind, value)),
        state(
            state::Config{.ir = ir, .channels = {}},
            arc::runtime::errors::noop_handler
        ) {}

    state::Node make_node() { return ASSERT_NIL_P(state.node("const")); }

private:
    static arc::ir::IR
    build_ir(const arc::types::Kind kind, const x::json::json &value) {
        arc::types::Param output_param;
        output_param.name = "output";
        output_param.type = arc::types::Type{.kind = kind};

        arc::types::Param value_param;
        value_param.name = "value";
        value_param.type = arc::types::Type{.kind = kind};
        value_param.value = value;

        arc::ir::Node ir_node;
        ir_node.key = "const";
        ir_node.type = "constant";
        ir_node.outputs.push_back(output_param);
        ir_node.config.push_back(value_param);

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
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    auto ir_node = setup.ir.nodes[0];
    ir_node.type = "not_constant";

    constant::Factory factory;
    ASSERT_OCCURRED_AS_P(
        factory.create(node::Config(setup.ir, ir_node, setup.make_node())),
        x::errors::NOT_FOUND
    );
}

/// @brief Test that factory creates a Constant node with valid configuration.
TEST(ConstantFactoryTest, CreatesConstantNode) {
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Factory factory;
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(setup.ir, setup.ir.nodes[0], setup.make_node()))
    );
    ASSERT_NE(node, nullptr);
}

/// @brief Test that next() outputs the constant value on first call.
TEST(ConstantTest, NextOutputsValueOnFirstCall) {
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_FLOAT_EQ(output->at<float>(0), 42.5f);
}

/// @brief Test that next() is a no-op on subsequent calls.
TEST(ConstantTest, NextNoOpsOnSubsequentCalls) {
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    output->set(0, 999.0f);

    ASSERT_NIL(node.next(ctx));
    EXPECT_FLOAT_EQ(output->at<float>(0), 999.0f);
}

/// @brief Test that reset() allows the value to be output again.
TEST(ConstantTest, ResetAllowsValueToBeOutputAgain) {
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    output->set(0, 999.0f);

    node.reset();
    node.next(ctx);

    EXPECT_FLOAT_EQ(output->at<float>(0), 42.5f);
}

/// @brief Test that float32 values are correctly cast and output.
TEST(ConstantTest, ValueIsCastToCorrectDataType_Float32) {
    TestSetup setup(arc::types::Kind::F32, 3.14f);
    constant::Constant node(setup.make_node(), 3.14f, x::telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_FLOAT_EQ(output->at<float>(0), 3.14f);
}

/// @brief Test that int64 values are correctly cast and output.
TEST(ConstantTest, ValueIsCastToCorrectDataType_Int64) {
    TestSetup setup(arc::types::Kind::I64, static_cast<int64_t>(12345));
    constant::Constant node(
        setup.make_node(),
        static_cast<int64_t>(12345),
        x::telem::INT64_T
    );

    auto ctx = make_context();
    node.next(ctx);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<int64_t>(0), 12345);
}

/// @brief Test that uint8 values are correctly cast and output.
TEST(ConstantTest, ValueIsCastToCorrectDataType_U8) {
    TestSetup setup(arc::types::Kind::U8, static_cast<uint8_t>(255));
    constant::Constant node(
        setup.make_node(),
        static_cast<uint8_t>(255),
        x::telem::UINT8_T
    );

    auto ctx = make_context();
    node.next(ctx);

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 255);
}

/// @brief Test that is_output_truthy delegates to state.
TEST(ConstantTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    EXPECT_TRUE(node.is_output_truthy("output"));
}

/// @brief Test that mark_changed is called on first next().
TEST(ConstantTest, MarkChangedCalledOnFirstNext) {
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T);

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
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T);

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
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_GT(output_time->at<int64_t>(0), 0);
}

/// @brief Test that reset produces a new timestamp on subsequent next().
TEST(ConstantTest, ResetProducesNewTimestamp) {
    TestSetup setup(arc::types::Kind::F32, 42.5f);
    constant::Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T);

    auto ctx = make_context();
    node.next(ctx);

    auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    const auto ts1 = output_time->at<int64_t>(0);

    node.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(1));
    node.next(ctx);

    const auto ts2 = output_time->at<int64_t>(0);
    EXPECT_GT(ts2, ts1);
}
