// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/variable/variable.h"

namespace arc::stl::variable {

/// @brief builds a param with the given name, type, and configured value.
static types::Param
param(std::string name, const types::Kind kind, x::json::json value = nullptr) {
    types::Param p;
    p.name = std::move(name);
    p.type.kind = kind;
    p.value = std::move(value);
    return p;
}

/// @brief builds an IR node with the given key, type, inputs, and outputs.
static ir::Node make_node(
    std::string key,
    std::string type,
    const std::vector<types::Param> &inputs,
    const std::vector<types::Param> &outputs
) {
    ir::Node n;
    n.key = std::move(key);
    n.type = std::move(type);
    for (const auto &p: inputs)
        n.inputs.push_back(p);
    for (const auto &p: outputs)
        n.outputs.push_back(p);
    return n;
}

/// @brief builds a variable node "v" with a value-carrying param and a feeder
/// node "f" edged into its second param.
static ir::IR register_ir(const types::Kind kind, x::json::json initial) {
    ir::IR prog;
    prog.nodes.push_back(
        make_node("f", "feeder", {}, {param(ir::default_output_param, kind)})
    );
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        {param("value", kind, std::move(initial)), param("f1", kind)},
        {param(ir::default_output_param, kind)}
    ));
    prog.edges.emplace_back(
        ir::Handle{"f", ir::default_output_param},
        ir::Handle{"v", "f1"},
        ir::EdgeKind::Continuous
    );
    return prog;
}

/// @brief builds a variable node "v" fed by a dispatcher stand-in "d" on value,
/// with sel fed from a register stand-in "selsrc".
static ir::IR expr_read_ir() {
    ir::IR prog;
    prog.nodes.push_back(
        make_node("d", "d", {}, {param(ir::default_output_param, types::Kind::I64)})
    );
    // The real lowering feeds sel from the register, a variable node, so sel
    // keeps its fires-on-fresh rearm rule and does not replay on Reset.
    prog.nodes.push_back(make_node(
        "selsrc",
        "stateful_variable",
        {},
        {param(ir::default_output_param, types::Kind::U32)}
    ));
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        {param("value", types::Kind::I64), param("sel", types::Kind::U32)},
        {param(ir::default_output_param, types::Kind::I64)}
    ));
    prog.edges.emplace_back(
        ir::Handle{"d", ir::default_output_param},
        ir::Handle{"v", "value"},
        ir::EdgeKind::Continuous
    );
    prog.edges.emplace_back(
        ir::Handle{"selsrc", ir::default_output_param},
        ir::Handle{"v", "sel"},
        ir::EdgeKind::Continuous
    );
    return prog;
}

/// @brief builds a variable node "v" fed by a dispatcher stand-in "d" and no
/// sel input.
static ir::IR sole_derivation_ir() {
    ir::IR prog;
    prog.nodes.push_back(
        make_node("d", "d", {}, {param(ir::default_output_param, types::Kind::I64)})
    );
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        {param("value", types::Kind::I64)},
        {param(ir::default_output_param, types::Kind::I64)}
    ));
    prog.edges.emplace_back(
        ir::Handle{"d", ir::default_output_param},
        ir::Handle{"v", "value"},
        ir::EdgeKind::Continuous
    );
    return prog;
}

/// @brief owns an IR and the state built from it. Both outlive every node handed
/// out, so a Program must stay put for the test's duration.
class Program {
    ir::IR prog;
    runtime::state::State state;

public:
    explicit Program(ir::IR prog):
        prog(std::move(prog)),
        state(
            runtime::state::Config{.ir = this->prog, .channels = {}},
            runtime::errors::noop_handler
        ) {}

    Program(const Program &) = delete;
    Program &operator=(const Program &) = delete;

    [[nodiscard]] const ir::IR &ir() const { return this->prog; }

    runtime::state::Node node(const std::string &key) {
        return ASSERT_NIL_P(this->state.node(key));
    }
};

/// @brief writes data and a second-precision timestamp to a node's first output.
template<typename T>
static void
emit(const runtime::state::Node &n, T value, const x::telem::TimeStamp seconds) {
    *n.output(0) = x::telem::Series(std::vector<T>{std::move(value)});
    *n.output_time(0) = x::telem::Series(seconds);
}

/// @brief returns a second-precision timestamp.
static x::telem::TimeStamp seconds(const int64_t v) {
    return x::telem::TimeStamp(v * x::telem::SECOND);
}

/// @brief returns a context recording the output ordinals a node marks changed.
static runtime::node::Context mark_context(std::vector<size_t> &marked) {
    return runtime::node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [&marked](const size_t i) { marked.push_back(i); },
        .report_error = [](const x::errors::Error &) {},
    };
}

/// @brief creates a node of the given type over "v" with a declared initial of 42.
static std::unique_ptr<runtime::node::Node>
make_register(Program &p, const std::string &node_type) {
    Module module;
    auto
        ir_node = make_node("v", node_type, {param("value", types::Kind::I64, 42)}, {});
    return ASSERT_NIL_P(
        module.create(runtime::node::Config(p.ir(), ir_node, p.node("v")))
    );
}

// ----- NewSymbols -----

class SymbolShapeTest : public ::testing::TestWithParam<std::string> {};

TEST_P(SymbolShapeTest, HandlesTheInternalFlowSymbol) {
    const Module module;
    EXPECT_TRUE(module.handles(GetParam()));
}

INSTANTIATE_TEST_SUITE_P(
    Variable,
    SymbolShapeTest,
    ::testing::Values(symbol_name, stateful_symbol_name),
    [](const ::testing::TestParamInfo<std::string> &info) { return info.param; }
);

TEST(VariableSymbolsTest, HandlesExactlyTheTwoVariableSymbols) {
    const Module module;
    EXPECT_TRUE(module.handles(symbol_name));
    EXPECT_TRUE(module.handles(stateful_symbol_name));
    EXPECT_FALSE(module.handles("constant"));
}

// ----- Create -----

TEST(VariableModuleTest, ReturnsNotFoundForAnUnknownNodeType) {
    Program p(register_ir(types::Kind::I64, 42));
    Module module;
    auto ir_node = make_node("v", "unknown", {}, {});
    ASSERT_OCCURRED_AS_P(
        module.create(runtime::node::Config(p.ir(), ir_node, p.node("v"))),
        x::errors::NOT_FOUND
    );
}

TEST(VariableModuleTest, CreatesANodeForAValueInitializedVariable) {
    Program p(register_ir(types::Kind::I64, 42));
    Module module;
    auto
        ir_node = make_node("v", "variable", {param("value", types::Kind::I64, 1)}, {});
    ASSERT_NE(
        ASSERT_NIL_P(
            module.create(runtime::node::Config(p.ir(), ir_node, p.node("v")))
        ),
        nullptr
    );
}

TEST(VariableModuleTest, CreatesANodeForAValueInitializedStatefulVariable) {
    Program p(register_ir(types::Kind::I64, 42));
    Module module;
    auto ir_node = make_node(
        "v",
        "stateful_variable",
        {param("value", types::Kind::I64, 1)},
        {}
    );
    ASSERT_NE(
        ASSERT_NIL_P(
            module.create(runtime::node::Config(p.ir(), ir_node, p.node("v")))
        ),
        nullptr
    );
}

TEST(VariableModuleTest, CreatesANodeForAnEdgeFedVariable) {
    Program p(register_ir(types::Kind::I64, 42));
    Module module;
    auto ir_node = make_node("v", "variable", {param("f0", types::Kind::I64)}, {});
    ASSERT_NE(
        ASSERT_NIL_P(
            module.create(runtime::node::Config(p.ir(), ir_node, p.node("v")))
        ),
        nullptr
    );
}

TEST(VariableModuleTest, CreatesANodeForAVariableWithNoInputs) {
    Program p(register_ir(types::Kind::I64, 42));
    Module module;
    auto ir_node = make_node("v", "variable", {}, {});
    ASSERT_NE(
        ASSERT_NIL_P(
            module.create(runtime::node::Config(p.ir(), ir_node, p.node("v")))
        ),
        nullptr
    );
}

TEST(VariableModuleTest, CreatesANodeForAnEdgeFedStatefulVariable) {
    Program p(register_ir(types::Kind::I64, 42));
    Module module;
    auto ir_node = make_node(
        "v",
        "stateful_variable",
        {param("f0", types::Kind::I64)},
        {}
    );
    ASSERT_NE(
        ASSERT_NIL_P(
            module.create(runtime::node::Config(p.ir(), ir_node, p.node("v")))
        ),
        nullptr
    );
}

// ----- Register -----

TEST(RegisterTest, EmitsItsPendingInitialValueOnFirstNext) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "variable");
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    const auto v = p.node("v");
    EXPECT_EQ(v.output(0)->values<int64_t>(), std::vector<int64_t>{42});
    EXPECT_EQ(v.output_time(0)->size(), 1);
}

TEST(RegisterTest, DoesNotReEmitWithoutNewData) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "variable");
    ASSERT_NIL(n->next(ctx));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked.size(), 1);
}

TEST(RegisterTest, EmitsAFeedersValue) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "variable");
    ASSERT_NIL(n->next(ctx));
    emit(p.node("f"), int64_t(7), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked.size(), 2);
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{7});
}

TEST(RegisterTest, TakesTheNewestValueWhenMultipleArePending) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "variable");
    emit(p.node("f"), int64_t(7), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{7});
}

TEST(RegisterTest, DoesNotAliasTheFeedersOutputBuffer) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "variable");
    ASSERT_NIL(n->next(ctx));
    const auto f = p.node("f");
    emit(f, int64_t(7), seconds(10));
    ASSERT_NIL(n->next(ctx));
    f.output(0)->set(0, int64_t(9));
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{7});
}

// ----- Register with a := variable -----

TEST(RegisterScopedTest, RestoresTheInitialValueOnReset) {
    Program p(register_ir(types::Kind::I64, 42));
    const auto n = make_register(p, "variable");
    n->reset();
    const auto v = p.node("v");
    EXPECT_EQ(v.output(0)->values<int64_t>(), std::vector<int64_t>{42});
    EXPECT_EQ(v.output_time(0)->size(), 1);
}

TEST(RegisterScopedTest, DoesNotDoubleEmitTheInitialValueOnNextAfterReset) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "variable");
    n->reset();
    ASSERT_NIL(n->next(ctx));
    EXPECT_TRUE(marked.empty());
}

TEST(RegisterScopedTest, SupersedesAPendingFeederValueOnReset) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "variable");
    emit(p.node("f"), int64_t(99), seconds(10));
    n->reset();
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{42});
    ASSERT_NIL(n->next(ctx));
    EXPECT_TRUE(marked.empty());
}

TEST(RegisterScopedTest, RestoresTheInitialValueOnScopeReEntry) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "variable");
    n->reset();
    emit(p.node("f"), int64_t(7), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{7});
    n->reset();
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{42});
}

TEST(RegisterScopedTest, DoesNotAliasTheInitialValueOnReset) {
    Program p(register_ir(types::Kind::I64, 42));
    const auto n = make_register(p, "variable");
    n->reset();
    p.node("v").output(0)->set(0, int64_t(9));
    n->reset();
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{42});
}

// ----- Register with a $= variable -----

TEST(RegisterStatefulTest, EmitsItsInitialValueOnFirstNext) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "stateful_variable");
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{42});
}

TEST(RegisterStatefulTest, PersistsItsValueAcrossReset) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "stateful_variable");
    emit(p.node("f"), int64_t(7), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{7});
    n->reset();
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{7});
}

TEST(RegisterStatefulTest, LeavesPendingFeederValuesConsumableAfterReset) {
    Program p(register_ir(types::Kind::I64, 42));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_register(p, "stateful_variable");
    emit(p.node("f"), int64_t(7), seconds(10));
    n->reset();
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{7});
}

// ----- Register with a string variable -----

TEST(RegisterStringTest, InitializesAndEmitsStringValues) {
    Program p(register_ir(types::Kind::String, "hello"));
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    Module module;
    auto ir_node = make_node(
        "v",
        "variable",
        {param("value", types::Kind::String, "hello")},
        {}
    );
    const auto n = ASSERT_NIL_P(
        module.create(runtime::node::Config(p.ir(), ir_node, p.node("v")))
    );
    n->reset();
    EXPECT_EQ(p.node("v").output(0)->strings(), std::vector<std::string>{"hello"});
    emit(p.node("f"), std::string("world"), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    EXPECT_EQ(p.node("v").output(0)->strings(), std::vector<std::string>{"world"});
    n->reset();
    EXPECT_EQ(p.node("v").output(0)->strings(), std::vector<std::string>{"hello"});
}

// ----- ExprRead -----

/// @brief creates the deref node over "v" for an edge-fed value param.
static std::unique_ptr<runtime::node::Node> make_expr_read(Program &p) {
    Module module;
    auto ir_node = make_node("v", "variable", {param("value", types::Kind::I64)}, {});
    return ASSERT_NIL_P(
        module.create(runtime::node::Config(p.ir(), ir_node, p.node("v")))
    );
}

TEST(ExprReadTest, EmitsTheDispatchersValue) {
    Program p(expr_read_ir());
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_expr_read(p);
    emit(p.node("d"), int64_t(5), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{5});
}

TEST(ExprReadTest, ReEmitsAnUnchangedRecompute) {
    Program p(expr_read_ir());
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_expr_read(p);
    emit(p.node("d"), int64_t(5), seconds(10));
    ASSERT_NIL(n->next(ctx));
    emit(p.node("d"), int64_t(5), seconds(20));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked.size(), 2)
        << "a fresh recompute fires even when the value is unchanged";
    emit(p.node("d"), int64_t(6), seconds(30));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked.size(), 3);
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{6});
}

TEST(ExprReadTest, AbsorbsTheValueArrivingWithARePoint) {
    Program p(expr_read_ir());
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_expr_read(p);
    emit(p.node("d"), int64_t(5), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    emit(p.node("selsrc"), uint32_t(1), seconds(20));
    emit(p.node("d"), int64_t(9), seconds(20));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked.size(), 1) << "the re-point's own recompute must not fire";
    emit(p.node("d"), int64_t(11), seconds(30));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked.size(), 2);
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{11});
}

TEST(ExprReadTest, FiresTheValueAfterAValueLessRePoint) {
    Program p(expr_read_ir());
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_expr_read(p);
    emit(p.node("selsrc"), uint32_t(1), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_TRUE(marked.empty());
    emit(p.node("d"), int64_t(9), seconds(20));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0})
        << "the dispatcher only emits data-driven values";
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{9});
}

TEST(ExprReadTest, DoesNotAliasTheDispatchersOutputBuffer) {
    Program p(expr_read_ir());
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_expr_read(p);
    const auto d = p.node("d");
    emit(d, int64_t(5), seconds(10));
    ASSERT_NIL(n->next(ctx));
    d.output(0)->set(0, int64_t(9));
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{5});
}

TEST(ExprReadTest, EmitsTheSoleDerivationWhenNoSelInputExists) {
    Program p(sole_derivation_ir());
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_expr_read(p);
    emit(p.node("d"), int64_t(5), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{5});
}

// ----- ExprRead Reset -----

TEST(ExprReadResetTest, FiresTheFirstValueAfterAResetAbsorbedInitialSel) {
    Program p(expr_read_ir());
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_expr_read(p);
    emit(p.node("selsrc"), uint32_t(0), seconds(5));
    n->reset();
    emit(p.node("d"), int64_t(7), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{7});
}

TEST(ExprReadResetTest, CoalescesValuesReplayedByReset) {
    Program p(expr_read_ir());
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_expr_read(p);
    emit(p.node("d"), int64_t(5), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    n->reset();
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked.size(), 1);
    emit(p.node("d"), int64_t(6), seconds(20));
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked.size(), 2);
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{6});
}

TEST(ExprReadResetTest, KeepsSelConsumedAcrossReset) {
    Program p(expr_read_ir());
    std::vector<size_t> marked;
    auto ctx = mark_context(marked);
    const auto n = make_expr_read(p);
    emit(p.node("selsrc"), uint32_t(1), seconds(10));
    emit(p.node("d"), int64_t(9), seconds(10));
    ASSERT_NIL(n->next(ctx));
    EXPECT_TRUE(marked.empty());
    n->reset();
    emit(p.node("d"), int64_t(11), seconds(20));
    // A replayed sel would count as a re-point and swallow the 11.
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(marked, std::vector<size_t>{0});
    EXPECT_EQ(p.node("v").output(0)->values<int64_t>(), std::vector<int64_t>{11});
}

}
