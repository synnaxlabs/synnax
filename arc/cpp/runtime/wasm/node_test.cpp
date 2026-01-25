// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <random>
#include <string>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/wasm/factory.h"
#include "arc/cpp/runtime/wasm/module.h"
#include "arc/cpp/runtime/wasm/node.h"

using namespace arc::runtime;

namespace {
std::mt19937 gen_rand = random_generator("Node Tests");

std::string random_name(const std::string &prefix) {
    std::uniform_int_distribution<> dis(10000, 99999);
    return prefix + std::to_string(dis(gen_rand));
}

/// @brief Compiles an Arc program via the Synnax client.
arc::module::Module
compile_arc(const synnax::Synnax &client, const std::string &source) {
    auto arc = synnax::Arc(random_name("test_arc"));
    arc.text.raw = source;
    if (const auto create_err = client.arcs.create(arc))
        throw std::runtime_error("Failed to create arc: " + create_err.message());

    synnax::RetrieveOptions opts;
    opts.compile = true;
    auto [compiled, err] = client.arcs.retrieve_by_key(arc.key, opts);
    if (err) throw std::runtime_error("Failed to compile arc: " + err.message());
    return compiled.module;
}

/// @brief Finds the IR node with the given type in the module.
const arc::ir::Node *
find_node_by_type(const arc::module::Module &mod, const std::string &type) {
    for (const auto &node: mod.nodes)
        if (node.type == type) return &node;
    return nullptr;
}

node::Context make_context() {
    return node::Context{
        .elapsed = telem::SECOND,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const xerrors::Error &) {},
        .activate_stage = [] {},
    };
}
}

/// @brief Factory::handles returns true for functions in the module.
TEST(FactoryTest, HandlesReturnsTrueForExistingFunction) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    wasm::Factory factory(wasm_mod);

    EXPECT_TRUE(factory.handles("double"));
    EXPECT_FALSE(factory.handles("nonexistent"));
}

/// @brief Factory::create returns NOT_FOUND when function doesn't exist.
TEST(FactoryTest, CreateReturnsErrorWhenFunctionNotFound) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    wasm::Factory factory(wasm_mod);

    arc::ir::Node fake_node;
    fake_node.key = "fake_node";
    fake_node.type = "nonexistent";

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}}
    );
    auto node_state = ASSERT_NIL_P(state.node(mod.nodes[0].key));

    const arc::ir::IR prog = static_cast<arc::ir::IR>(mod);
    node::Config cfg(prog, fake_node, std::move(node_state));
    ASSERT_OCCURRED_AS_P(factory.create(std::move(cfg)), xerrors::NOT_FOUND);
}

/// @brief Factory::create succeeds with valid function.
TEST(FactoryTest, CreateSucceedsWithValidConfig) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    wasm::Factory factory(wasm_mod);

    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}}
    );
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));

    const arc::ir::IR prog = static_cast<arc::ir::IR>(mod);
    node::Config cfg(prog, *func_node, std::move(node_state));
    auto node = ASSERT_NIL_P(factory.create(std::move(cfg)));
    ASSERT_NE(node, nullptr);
}

/// @brief Node::next returns early and doesn't mark outputs when no inputs refreshed.
TEST(NodeTest, NextReturnsEarlyWhenNoInputsRefreshed) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}}
    );
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("double"));

    wasm::Node node(mod, *func_node, std::move(node_state), func);

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](const std::string &name) {
        changed_outputs.push_back(name);
    };

    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed_outputs.empty());
}

/// @brief Node::next executes WASM function and produces correct output.
TEST(NodeTest, NextExecutesFunctionAndProducesOutput) {
    const auto client = new_test_client();

    auto input_idx_name = random_name("input_idx");
    auto input_name = random_name("input_val");
    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output_val");

    auto input_idx = synnax::Channel(input_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(input_idx));
    auto output_idx = synnax::Channel(output_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(output_idx));

    auto input_ch = synnax::Channel(input_name, telem::FLOAT32_T, input_idx.key, false);
    ASSERT_NIL(client.channels.create(input_ch));
    auto output_ch = synnax::Channel(
        output_name,
        telem::FLOAT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + input_name + " -> double{} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {input_idx.key, telem::TIMESTAMP_T, 0},
                {input_ch.key, telem::FLOAT32_T, input_idx.key},
                {output_idx.key, telem::TIMESTAMP_T, 0},
                {output_ch.key, telem::FLOAT32_T, output_idx.key}
            }
        }
    );

    // Find the 'on' node that reads from the input channel
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    // Get the 'on' node's state and set its outputs directly
    // This simulates what on.next() would do after reading from channels
    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));

    auto on_data = telem::Series(std::vector{5.0f, 10.0f, 15.0f});
    on_data.alignment = telem::Alignment(1, 0);
    on_node_state.output(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_data)
    );

    auto on_time = telem::Series(
        std::vector{
            telem::TimeStamp(1 * telem::MICROSECOND),
            telem::TimeStamp(2 * telem::MICROSECOND),
            telem::TimeStamp(3 * telem::MICROSECOND)
        }
    );
    on_time.alignment = telem::Alignment(1, 0);
    on_node_state.output_time(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_time)
    );

    // Now set up the double node
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("double"));

    wasm::Node node(mod, *func_node, std::move(node_state), func);

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](const std::string &name) {
        changed_outputs.push_back(name);
    };

    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(changed_outputs.size(), 1);

    // Verify the double node's output
    auto double_node_state = ASSERT_NIL_P(state.node(func_node->key));
    const auto &output = double_node_state.output(0);
    ASSERT_EQ(output->size(), 3);
    EXPECT_FLOAT_EQ(output->at<float>(0), 10.0f);
    EXPECT_FLOAT_EQ(output->at<float>(1), 20.0f);
    EXPECT_FLOAT_EQ(output->at<float>(2), 30.0f);
}

/// @brief Node::next reports errors via context when WASM execution fails.
TEST(NodeTest, NextReportsErrorOnWasmTrap) {
    const auto client = new_test_client();

    auto idx_name = random_name("time");
    auto input_name = random_name("input");
    auto output_name = random_name("output");

    auto index_ch = synnax::Channel(idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(index_ch));

    auto input_ch = synnax::Channel(input_name, telem::INT32_T, index_ch.key, false);
    ASSERT_NIL(client.channels.create(input_ch));
    auto output_ch = synnax::Channel(output_name, telem::INT32_T, index_ch.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func divide_by_zero(val i32) i32 {
    return val / 0
}
)" + input_name + " -> divide_by_zero{} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "divide_by_zero");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {index_ch.key, telem::TIMESTAMP_T, 0},
                {input_ch.key, telem::INT32_T, index_ch.key},
                {output_ch.key, telem::INT32_T, index_ch.key}
            }
        }
    );

    // Find the 'on' node and set its outputs
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = telem::Series(static_cast<int32_t>(42));
    on_data.alignment = telem::Alignment(1, 0);
    on_node_state.output(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_data)
    );
    auto on_time = telem::Series(telem::TimeStamp(1 * telem::MICROSECOND));
    on_time.alignment = telem::Alignment(1, 0);
    on_node_state.output_time(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("divide_by_zero"));

    wasm::Node node(mod, *func_node, std::move(node_state), func);

    auto ctx = make_context();
    std::vector<xerrors::Error> reported_errors;
    ctx.report_error = [&](const xerrors::Error &err) {
        reported_errors.push_back(err);
    };

    ASSERT_NIL(node.next(ctx));
    EXPECT_GE(reported_errors.size(), 1);
}

/// @brief Node::is_output_truthy returns false for nonexistent outputs.
TEST(NodeTest, IsOutputTruthyReturnsFalseForNonexistent) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}}
    );
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("double"));

    wasm::Node node(mod, *func_node, std::move(node_state), func);
    EXPECT_FALSE(node.is_output_truthy("nonexistent"));
}

/// @brief Node::is_output_truthy correctly evaluates output values.
TEST(NodeTest, IsOutputTruthyEvaluatesOutputValues) {
    const auto client = new_test_client();

    auto idx_name = random_name("time");
    auto input_name = random_name("input");
    auto output_name = random_name("output");

    auto index_ch = synnax::Channel(idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(index_ch));

    auto input_ch = synnax::Channel(input_name, telem::FLOAT32_T, index_ch.key, false);
    ASSERT_NIL(client.channels.create(input_ch));
    auto
        output_ch = synnax::Channel(output_name, telem::FLOAT32_T, index_ch.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func passthrough(val f32) f32 {
    return val
}
)" + input_name + " -> passthrough{} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "passthrough");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {index_ch.key, telem::TIMESTAMP_T, 0},
                {input_ch.key, telem::FLOAT32_T, index_ch.key},
                {output_ch.key, telem::FLOAT32_T, index_ch.key}
            }
        }
    );

    // Find the 'on' node and set its outputs
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = telem::Series(42.0f);
    on_data.alignment = telem::Alignment(1, 0);
    on_node_state.output(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_data)
    );
    auto on_time = telem::Series(telem::TimeStamp(1 * telem::MICROSECOND));
    on_time.alignment = telem::Alignment(1, 0);
    on_node_state.output_time(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("passthrough"));

    wasm::Node node(mod, *func_node, std::move(node_state), func);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    const auto &output_param = func_node->outputs[0];
    EXPECT_TRUE(node.is_output_truthy(output_param.name));
}

TEST(NodeTest, NoInputNodeExecutesOncePerStageEntry) {
    const auto client = new_test_client();

    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto output_idx = synnax::Channel(output_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(output_idx));
    auto
        output_ch = synnax::Channel(output_name, telem::INT64_T, output_idx.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func constant() i64 {
    return 42
}
constant{} -> )" + output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "constant");
    ASSERT_NE(func_node, nullptr);
    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {output_idx.key, telem::TIMESTAMP_T, 0},
                {output_ch.key, telem::INT64_T, output_idx.key}
            }
        }
    );
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("constant"));

    wasm::Node node(mod, *func_node, std::move(node_state), func);

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](const std::string &name) {
        changed_outputs.push_back(name);
    };

    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(changed_outputs.size(), 1);

    auto output_state = ASSERT_NIL_P(state.node(func_node->key));
    const auto &output = output_state.output(0);
    ASSERT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<int64_t>(0), 42);

    changed_outputs.clear();
    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed_outputs.empty());

    node.reset();

    changed_outputs.clear();
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(changed_outputs.size(), 1);
}

TEST(NodeTest, NodeWithInputsExecutesNormally) {
    const auto client = new_test_client();

    auto input_idx_name = random_name("input_idx");
    auto input_name = random_name("input");
    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto input_idx = synnax::Channel(input_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(input_idx));
    auto output_idx = synnax::Channel(output_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(output_idx));

    auto input_ch = synnax::Channel(input_name, telem::INT64_T, input_idx.key, false);
    ASSERT_NIL(client.channels.create(input_ch));
    auto
        output_ch = synnax::Channel(output_name, telem::INT64_T, output_idx.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func double(val i64) i64 {
    return val * 2
}
)" + input_name + " -> double{} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {input_idx.key, telem::TIMESTAMP_T, 0},
                {input_ch.key, telem::INT64_T, input_idx.key},
                {output_idx.key, telem::TIMESTAMP_T, 0},
                {output_ch.key, telem::INT64_T, output_idx.key}
            }
        }
    );

    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = telem::Series(static_cast<int64_t>(5));
    on_data.alignment = telem::Alignment(1, 0);
    on_node_state.output(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_data)
    );
    auto on_time = telem::Series(telem::TimeStamp(1 * telem::MICROSECOND));
    on_time.alignment = telem::Alignment(1, 0);
    on_node_state.output_time(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("double"));

    wasm::Node node(mod, *func_node, std::move(node_state), func);

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](const std::string &name) {
        changed_outputs.push_back(name);
    };

    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(changed_outputs.size(), 1);

    auto on_node_state2 = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data2 = telem::Series(static_cast<int64_t>(10));
    on_data2.alignment = telem::Alignment(2, 0);
    on_node_state2.output(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_data2)
    );
    auto on_time2 = telem::Series(telem::TimeStamp(2 * telem::MICROSECOND));
    on_time2.alignment = telem::Alignment(2, 0);
    on_node_state2.output_time(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_time2)
    );

    changed_outputs.clear();
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(changed_outputs.size(), 1);
}

TEST(NodeTest, FlowExpressionExecutesEveryTime) {
    const auto client = new_test_client();

    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto output_idx = synnax::Channel(output_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(output_idx));
    auto
        output_ch = synnax::Channel(output_name, telem::INT64_T, output_idx.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func counter() i64 {
    return 42
}
counter{} -> )" + output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "counter");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {output_idx.key, telem::TIMESTAMP_T, 0},
                {output_ch.key, telem::INT64_T, output_idx.key}
            }
        }
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("counter"));

    arc::ir::Node expr_node = *func_node;
    expr_node.key = "expression_0";

    wasm::Node node(mod, expr_node, std::move(node_state), func);

    auto ctx = make_context();

    ASSERT_NIL(node.next(ctx));
    auto s1 = ASSERT_NIL_P(state.node(func_node->key));
    EXPECT_EQ(s1.output(0)->at<int64_t>(0), 42);

    ASSERT_NIL(node.next(ctx));
    auto s2 = ASSERT_NIL_P(state.node(func_node->key));
    EXPECT_EQ(s2.output(0)->at<int64_t>(0), 42);

    ASSERT_NIL(node.next(ctx));
    auto s3 = ASSERT_NIL_P(state.node(func_node->key));
    EXPECT_EQ(s3.output(0)->at<int64_t>(0), 42);
}

TEST(NodeTest, FlowExpressionContinuesAfterReset) {
    const auto client = new_test_client();

    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto output_idx = synnax::Channel(output_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(output_idx));
    auto
        output_ch = synnax::Channel(output_name, telem::INT64_T, output_idx.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func counter() i64 {
    return 42
}
counter{} -> )" + output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "counter");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {output_idx.key, telem::TIMESTAMP_T, 0},
                {output_ch.key, telem::INT64_T, output_idx.key}
            }
        }
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("counter"));

    arc::ir::Node expr_node = *func_node;
    expr_node.key = "expression_0";

    wasm::Node node(mod, expr_node, std::move(node_state), func);

    auto ctx = make_context();

    ASSERT_NIL(node.next(ctx));
    auto s1 = ASSERT_NIL_P(state.node(func_node->key));
    EXPECT_EQ(s1.output(0)->at<int64_t>(0), 42);

    node.reset();

    ASSERT_NIL(node.next(ctx));
    auto s2 = ASSERT_NIL_P(state.node(func_node->key));
    EXPECT_EQ(s2.output(0)->at<int64_t>(0), 42);

    ASSERT_NIL(node.next(ctx));
    auto s3 = ASSERT_NIL_P(state.node(func_node->key));
    EXPECT_EQ(s3.output(0)->at<int64_t>(0), 42);
}

TEST(NodeTest, NonExpressionNodeNotTreatedAsExpression) {
    const auto client = new_test_client();

    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto output_idx = synnax::Channel(output_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(output_idx));
    auto
        output_ch = synnax::Channel(output_name, telem::INT64_T, output_idx.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func counter() i64 {
    return 42
}
counter{} -> )" + output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "counter");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {output_idx.key, telem::TIMESTAMP_T, 0},
                {output_ch.key, telem::INT64_T, output_idx.key}
            }
        }
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("counter"));

    arc::ir::Node non_expr_node = *func_node;
    non_expr_node.key = "expr_0";

    wasm::Node node(mod, non_expr_node, std::move(node_state), func);

    auto ctx = make_context();

    ASSERT_NIL(node.next(ctx));
    auto s1 = ASSERT_NIL_P(state.node(func_node->key));
    EXPECT_EQ(s1.output(0)->at<int64_t>(0), 42);

    ASSERT_NIL(node.next(ctx));
    auto s2 = ASSERT_NIL_P(state.node(func_node->key));
    EXPECT_EQ(s2.output(0)->at<int64_t>(0), 42);
}

/// @brief Config parameters are passed to WASM function correctly.
TEST(NodeTest, ConfigParametersPassedToWasm) {
    const auto client = new_test_client();

    auto input_idx_name = random_name("input_idx");
    auto input_name = random_name("input");
    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto input_idx = synnax::Channel(input_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(input_idx));
    auto output_idx = synnax::Channel(output_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(output_idx));

    auto input_ch = synnax::Channel(input_name, telem::INT32_T, input_idx.key, false);
    ASSERT_NIL(client.channels.create(input_ch));
    auto
        output_ch = synnax::Channel(output_name, telem::INT32_T, output_idx.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    // Function with config parameter 'x' and input parameter 'y'
    // Use i32 since integer literals default to i32
    const std::string source = R"(
func add_config{x i32}(y i32) i32 {
    return x + y
}
)" + input_name + " -> add_config{x=10} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "add_config");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {input_idx.key, telem::TIMESTAMP_T, 0},
                {input_ch.key, telem::INT32_T, input_idx.key},
                {output_idx.key, telem::TIMESTAMP_T, 0},
                {output_ch.key, telem::INT32_T, output_idx.key}
            }
        }
    );

    // Find and set up the 'on' node that reads from the input channel
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = telem::Series(std::vector<int32_t>{5});
    on_data.alignment = telem::Alignment(1, 0);
    on_node_state.output(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_data)
    );
    auto on_time = telem::Series(std::vector{telem::TimeStamp(1 * telem::MICROSECOND)});
    on_time.alignment = telem::Alignment(1, 0);
    on_node_state.output_time(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("add_config", func_node->config));

    wasm::Node node(mod, *func_node, std::move(node_state), func);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    // Verify the output: config x=10 + input y=5 = 15
    auto result_state = ASSERT_NIL_P(state.node(func_node->key));
    const auto &output = result_state.output(0);
    ASSERT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<int32_t>(0), 15);
}

/// @brief Multiple config parameters are passed correctly.
TEST(NodeTest, MultipleConfigParametersPassedToWasm) {
    const auto client = new_test_client();

    auto input_idx_name = random_name("input_idx");
    auto input_name = random_name("input");
    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto input_idx = synnax::Channel(input_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(input_idx));
    auto output_idx = synnax::Channel(output_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(output_idx));

    auto input_ch = synnax::Channel(input_name, telem::INT32_T, input_idx.key, false);
    ASSERT_NIL(client.channels.create(input_ch));
    auto
        output_ch = synnax::Channel(output_name, telem::INT32_T, output_idx.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    // Function with two config parameters 'a', 'b' and input parameter 'c'
    const std::string source = R"(
func multi_config{a i32, b i32}(c i32) i32 {
    return a + b + c
}
)" + input_name + " -> multi_config{a=5, b=10} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "multi_config");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {input_idx.key, telem::TIMESTAMP_T, 0},
                {input_ch.key, telem::INT32_T, input_idx.key},
                {output_idx.key, telem::TIMESTAMP_T, 0},
                {output_ch.key, telem::INT32_T, output_idx.key}
            }
        }
    );

    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = telem::Series(std::vector<int32_t>{3});
    on_data.alignment = telem::Alignment(1, 0);
    on_node_state.output(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_data)
    );
    auto on_time = telem::Series(std::vector{telem::TimeStamp(1 * telem::MICROSECOND)});
    on_time.alignment = telem::Alignment(1, 0);
    on_node_state.output_time(0) = xmemory::make_local_shared<telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("multi_config", func_node->config));

    wasm::Node node(mod, *func_node, std::move(node_state), func);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    // Verify the output: a=5 + b=10 + c=3 = 18
    auto result_state = ASSERT_NIL_P(state.node(func_node->key));
    const auto &output = result_state.output(0);
    ASSERT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<int32_t>(0), 18);
}
