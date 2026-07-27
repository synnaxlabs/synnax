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

#include "x/cpp/mem/indirect.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/constant/constant.h"

namespace arc::stl::constant {
runtime::node::Context make_context() {
    return runtime::node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [](size_t) {},
        .report_error = [](const x::errors::Error &) {},
    };
}

struct TestSetup {
    ir::IR ir;
    runtime::state::State state;

    TestSetup(const types::Kind kind, const x::json::json &value):
        ir(build_ir(kind, value)),
        state(
            runtime::state::Config{.ir = ir, .channels = {}},
            runtime::errors::noop_handler
        ) {}

    runtime::state::Node make_node() { return ASSERT_NIL_P(state.node("const")); }

private:
    static ir::IR build_ir(const types::Kind kind, const x::json::json &value) {
        types::Param output_param;
        output_param.name = "output";
        output_param.type.kind = kind;

        types::Param value_param;
        value_param.name = "value";
        value_param.type.kind = kind;
        value_param.value = value;

        ir::Node ir_node;
        ir_node.key = "const";
        ir_node.type = "constant";
        ir_node.outputs.push_back(output_param);
        ir_node.inputs.push_back(value_param);

        ir::Function fn;
        fn.key = "test";

        ir::IR ir;
        ir.nodes.push_back(ir_node);
        ir.functions.push_back(fn);
        return ir;
    }
};

/// @brief Test that factory returns NOT_FOUND for non-constant node types.
TEST(ConstantModuleTest, ReturnsNotFoundForWrongType) {
    TestSetup setup(types::Kind::F32, 42.5f);
    auto ir_node = setup.ir.nodes[0];
    ir_node.type = "not_constant";

    Module module;
    ASSERT_OCCURRED_AS_P(
        module.create(runtime::node::Config(setup.ir, ir_node, setup.make_node())),
        x::errors::NOT_FOUND
    );
}

/// @brief Test that module creates a Constant node with valid configuration.
TEST(ConstantModuleTest, CreatesConstantNode) {
    TestSetup setup(types::Kind::F32, 42.5f);
    Module module;
    auto node = ASSERT_NIL_P(module.create(
        runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
    ));
    ASSERT_NE(node, nullptr);
}

/// @brief Test that create returns VALIDATION when the value parameter has no value.
TEST(ConstantModuleTest, ErrorsWhenValueMissing) {
    TestSetup setup(types::Kind::F32, x::json::json(nullptr));
    Module module;
    ASSERT_OCCURRED_AS_P(
        module.create(
            runtime::node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
        ),
        x::errors::VALIDATION
    );
}

/// @brief Test that next() outputs the constant value on first call.
TEST(ConstantTest, NextOutputsValueOnFirstCall) {
    TestSetup setup(types::Kind::F32, 42.5f);
    Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T, true);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_FLOAT_EQ(output->at<float>(0), 42.5f);
}

/// @brief Test that next() is a no-op on subsequent calls.
TEST(ConstantTest, NextNoOpsOnSubsequentCalls) {
    TestSetup setup(types::Kind::F32, 42.5f);
    Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T, true);

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
    TestSetup setup(types::Kind::F32, 42.5f);
    Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T, true);

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
    TestSetup setup(types::Kind::F32, 3.14f);
    Constant node(setup.make_node(), 3.14f, x::telem::FLOAT32_T, true);

    auto ctx = make_context();
    node.next(ctx);

    const auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_FLOAT_EQ(output->at<float>(0), 3.14f);
}

/// @brief Test that int64 values are correctly cast and output.
TEST(ConstantTest, ValueIsCastToCorrectDataType_Int64) {
    TestSetup setup(types::Kind::I64, static_cast<int64_t>(12345));
    Constant
        node(setup.make_node(), static_cast<int64_t>(12345), x::telem::INT64_T, true);

    auto ctx = make_context();
    node.next(ctx);

    const auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<int64_t>(0), 12345);
}

/// @brief Test that uint8 values are correctly cast and output.
TEST(ConstantTest, ValueIsCastToCorrectDataType_U8) {
    TestSetup setup(types::Kind::U8, static_cast<uint8_t>(255));
    Constant
        node(setup.make_node(), static_cast<uint8_t>(255), x::telem::UINT8_T, true);

    auto ctx = make_context();
    node.next(ctx);

    const auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<uint8_t>(0), 255);
}

/// @brief Test that is_output_truthy delegates to state.
TEST(ConstantTest, IsOutputTruthyDelegatesToState) {
    TestSetup setup(types::Kind::F32, 42.5f);
    Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T, true);

    auto ctx = make_context();
    node.next(ctx);

    EXPECT_TRUE(node.is_output_truthy(0));
}

/// @brief Test that mark_changed is called on first next().
TEST(ConstantTest, MarkChangedCalledOnFirstNext) {
    TestSetup setup(types::Kind::F32, 42.5f);
    Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T, true);

    std::vector<size_t> marked;
    auto ctx = make_context();
    ctx.mark_changed = [&](size_t i) { marked.push_back(i); };

    node.next(ctx);

    ASSERT_EQ(marked.size(), 1);
    EXPECT_EQ(marked[0], 0);
}

/// @brief Test that mark_changed is not called on subsequent next() calls.
TEST(ConstantTest, MarkChangedNotCalledOnSubsequentNext) {
    TestSetup setup(types::Kind::F32, 42.5f);
    Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T, true);

    auto ctx = make_context();
    node.next(ctx);

    int call_count = 0;
    ctx.mark_changed = [&](size_t) { call_count++; };

    node.next(ctx);
    node.next(ctx);

    EXPECT_EQ(call_count, 0);
}

/// @brief Test that timestamp is populated on first next().
TEST(ConstantTest, TimestampOutputOnFirstNext) {
    TestSetup setup(types::Kind::F32, 42.5f);
    Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T, true);

    auto ctx = make_context();
    node.next(ctx);

    auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 1);
    EXPECT_GT(output_time->at<int64_t>(0), 0);
}

/// @brief Test that string values are correctly output.
TEST(ConstantTest, StringValueIsOutput) {
    const std::string val = "hello";
    TestSetup setup(types::Kind::String, val);
    Constant node(setup.make_node(), val, x::telem::STRING_T, true);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<std::string>(0), val);
}

/// @brief Test that reset() allows the string value to be output again.
TEST(ConstantTest, StringResetAllowsValueToBeOutputAgain) {
    const std::string val = "hello";
    TestSetup setup(types::Kind::String, val);
    Constant node(setup.make_node(), val, x::telem::STRING_T, true);

    auto ctx = make_context();
    node.next(ctx);
    node.reset();
    ASSERT_NIL(node.next(ctx));

    auto checker = setup.make_node();
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<std::string>(0), val);
}

/// @brief Test that reset produces a new timestamp on subsequent next().
TEST(ConstantTest, ResetProducesNewTimestamp) {
    TestSetup setup(types::Kind::F32, 42.5f);
    Constant node(setup.make_node(), 42.5f, x::telem::FLOAT32_T, true);

    auto ctx = make_context();
    node.next(ctx);

    const auto checker = setup.make_node();
    const auto &output_time = checker.output_time(0);
    const auto ts1 = output_time->at<int64_t>(0);

    node.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(1));
    node.next(ctx);

    const auto ts2 = output_time->at<int64_t>(0);
    EXPECT_GT(ts2, ts1);
}

/// @brief wires a constant whose value input references variable node "v" and gives
/// it an inbound trigger edge so it re-emits on every Next. The IR and state outlive
/// the node, so a VarBound must stay put for the test's duration.
class VarBound {
    ir::IR prog;
    runtime::state::State state;

public:
    std::unique_ptr<runtime::node::Node> node;

    VarBound(const types::Kind kind, x::json::json initial):
        prog(build_ir(kind, std::move(initial))),
        state(
            runtime::state::Config{.ir = prog, .channels = {}},
            runtime::errors::noop_handler
        ) {
        Module module;
        auto state_node = ASSERT_NIL_P(this->state.node("n"));
        this->node = ASSERT_NIL_P(module.create(
            runtime::node::Config(
                this->prog,
                this->prog.nodes[1],
                std::move(state_node)
            )
        ));
    }

    VarBound(const VarBound &) = delete;
    VarBound &operator=(const VarBound &) = delete;

    runtime::state::Node handle(const std::string &key) {
        return ASSERT_NIL_P(this->state.node(key));
    }

private:
    static ir::IR build_ir(const types::Kind kind, x::json::json initial) {
        types::Param var_out;
        var_out.name = ir::default_output_param;
        var_out.type = types::Type{.kind = kind};
        ir::Node v;
        v.key = "v";
        v.type = "variable";
        v.outputs.push_back(var_out);

        types::Param value;
        value.name = "value";
        value.type = types::Type{
            .kind = types::Kind::VarRef,
            .name = "v",
            .elem = x::mem::indirect<types::Type>(types::Type{.kind = kind})
        };
        value.value = std::move(initial);
        types::Param out;
        out.name = ir::default_output_param;
        out.type = types::Type{.kind = kind};
        ir::Node n;
        n.key = "n";
        n.type = "constant";
        n.inputs.push_back(value);
        n.outputs.push_back(out);

        ir::IR ir;
        ir.nodes.push_back(v);
        ir.nodes.push_back(n);
        ir.edges.emplace_back(
            ir::Handle("up", ir::default_output_param),
            ir::Handle("n", ir::default_input_param),
            ir::EdgeKind::Continuous
        );
        return ir;
    }
};

TEST(ConstantVarBoundTest, EmitsTheDeclaredInitialBeforeAnyVariableWrite) {
    VarBound t(types::Kind::I64, static_cast<int64_t>(42));
    auto ctx = make_context();
    ASSERT_NIL(t.node->next(ctx));
    const auto out = t.handle("n").output(0);
    EXPECT_EQ(out->size(), 1);
    EXPECT_EQ(out->at<int64_t>(0), 42);
}

TEST(ConstantVarBoundTest, EmitsTheLiveValueAfterAVariableWrite) {
    VarBound t(types::Kind::I64, static_cast<int64_t>(42));
    *t.handle("v").output(0) = x::telem::Series(std::vector<int64_t>{7});
    auto ctx = make_context();
    ASSERT_NIL(t.node->next(ctx));
    const auto out = t.handle("n").output(0);
    EXPECT_EQ(out->size(), 1);
    EXPECT_EQ(out->at<int64_t>(0), 7);
    EXPECT_EQ(out->data_type(), x::telem::INT64_T);
}

TEST(ConstantVarBoundTest, TracksSuccessiveVariableWrites) {
    VarBound t(types::Kind::I64, static_cast<int64_t>(42));
    auto ctx = make_context();
    *t.handle("v").output(0) = x::telem::Series(std::vector<int64_t>{7});
    ASSERT_NIL(t.node->next(ctx));
    EXPECT_EQ(t.handle("n").output(0)->at<int64_t>(0), 7);
    *t.handle("v").output(0) = x::telem::Series(std::vector<int64_t>{9});
    ASSERT_NIL(t.node->next(ctx));
    EXPECT_EQ(t.handle("n").output(0)->at<int64_t>(0), 9);
}

TEST(ConstantVarBoundTest, EmitsOnlyTheLatestSampleOfTheVariablesSeries) {
    VarBound t(types::Kind::I64, static_cast<int64_t>(42));
    *t.handle("v").output(0) = x::telem::Series(std::vector<int64_t>{1, 2, 3});
    auto ctx = make_context();
    ASSERT_NIL(t.node->next(ctx));
    const auto out = t.handle("n").output(0);
    EXPECT_EQ(out->size(), 1);
    EXPECT_EQ(out->at<int64_t>(0), 3);
}

TEST(ConstantVarBoundTest, EmitsTheDeclaredStringInitialBeforeAnyWrite) {
    VarBound t(types::Kind::String, "hello");
    auto ctx = make_context();
    ASSERT_NIL(t.node->next(ctx));
    const auto out = t.handle("n").output(0);
    EXPECT_EQ(out->size(), 1);
    EXPECT_EQ(out->at<std::string>(-1), "hello");
}

TEST(ConstantVarBoundTest, EmitsTheLiveStringAfterAWrite) {
    VarBound t(types::Kind::String, "hello");
    *t.handle("v").output(0) = x::telem::Series(std::vector<std::string>{"goodbye"});
    auto ctx = make_context();
    ASSERT_NIL(t.node->next(ctx));
    const auto out = t.handle("n").output(0);
    EXPECT_EQ(out->size(), 1);
    EXPECT_EQ(out->at<std::string>(-1), "goodbye");
    EXPECT_EQ(out->data_type(), x::telem::STRING_T);
}

TEST(ConstantVarBoundTest, EmitsOnlyTheLatestStringSample) {
    VarBound t(types::Kind::String, "hello");
    *t.handle("v").output(0) = x::telem::Series(std::vector<std::string>{"a", "b"});
    auto ctx = make_context();
    ASSERT_NIL(t.node->next(ctx));
    const auto out = t.handle("n").output(0);
    EXPECT_EQ(out->size(), 1);
    EXPECT_EQ(out->at<std::string>(-1), "b");
}

TEST(ConstantVarBoundTest, EmitsALiveFloatValueFromRawSampleBytes) {
    VarBound t(types::Kind::F64, 1.5);
    *t.handle("v").output(0) = x::telem::Series(std::vector<double>{2.5});
    auto ctx = make_context();
    ASSERT_NIL(t.node->next(ctx));
    EXPECT_DOUBLE_EQ(t.handle("n").output(0)->at<double>(0), 2.5);
}

TEST(ConstantVarBoundTest, ReEmitsOnEveryTriggerTrackingTheVariable) {
    VarBound t(types::Kind::I64, static_cast<int64_t>(42));
    std::vector<size_t> marked;
    auto ctx = make_context();
    ctx.mark_changed = [&marked](const size_t i) { marked.push_back(i); };
    ASSERT_NIL(t.node->next(ctx));
    EXPECT_EQ(t.handle("n").output(0)->at<int64_t>(0), 42);
    *t.handle("v").output(0) = x::telem::Series(std::vector<int64_t>{7});
    ASSERT_NIL(t.node->next(ctx));
    EXPECT_EQ(t.handle("n").output(0)->at<int64_t>(0), 7);
    EXPECT_EQ(marked.size(), 2);
}
}
