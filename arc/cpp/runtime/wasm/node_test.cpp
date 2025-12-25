// Copyright 2025 Synnax Labs, Inc.
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

    node::Config cfg(fake_node, std::move(node_state));
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

    node::Config cfg(*func_node, std::move(node_state));
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

    wasm::Node node(*func_node, std::move(node_state), func);

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](const std::string &name) {
        changed_outputs.push_back(name);
    };

    ASSERT_NIL(node.next(ctx));
    EXPECT_TRUE(changed_outputs.empty());
}

/// @brief Node::next executes function with no inputs and produces output.
TEST(NodeTest, NextExecutesFunctionWithNoInputs) {
    const auto client = new_test_client();

    const std::string source = R"(
func constant() f32 {
    return 42.0
}
constant{}
)";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "constant");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}}
    );
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("constant"));

    wasm::Node node(*func_node, std::move(node_state), func);

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](const std::string &name) {
        changed_outputs.push_back(name);
    };

    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(changed_outputs.size(), 1);

    auto &output = node_state.output(0);
    ASSERT_EQ(output->size(), 1);
    EXPECT_FLOAT_EQ(output->at<float>(0), 42.0f);
}

/// @brief Node::next executes function with default input value.
TEST(NodeTest, NextExecutesFunctionWithDefaultValue) {
    const auto client = new_test_client();

    const std::string source = R"(
func double(val f32 = 5.0) f32 {
    return val * 2.0
}
double{}
)";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}}
    );
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("double"));

    wasm::Node node(*func_node, std::move(node_state), func);

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](const std::string &name) {
        changed_outputs.push_back(name);
    };

    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(changed_outputs.size(), 1);

    auto &output = node_state.output(0);
    ASSERT_EQ(output->size(), 1);
    EXPECT_FLOAT_EQ(output->at<float>(0), 10.0f);
}

/// @brief Node::next handles multi-input functions correctly.
TEST(NodeTest, NextHandlesMultipleInputs) {
    const auto client = new_test_client();

    auto idx_name = random_name("idx");
    auto a_name = random_name("cha");
    auto b_name = random_name("chb");
    auto out_name = random_name("chsum");

    auto idx_ch = synnax::Channel(idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client.channels.create(idx_ch));

    auto input_a = synnax::Channel(a_name, telem::FLOAT32_T, idx_ch.key, false);
    ASSERT_NIL(client.channels.create(input_a));
    auto input_b = synnax::Channel(b_name, telem::FLOAT32_T, idx_ch.key, false);
    ASSERT_NIL(client.channels.create(input_b));
    auto output_ch = synnax::Channel(out_name, telem::FLOAT32_T, idx_ch.key, false);
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func add(a f32, b f32) f32 {
    return a + b
}
)" + a_name + ", " + b_name + " -> add{} -> " +
                               out_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.module = mod}));
    const auto *func_node = find_node_by_type(mod, "add");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels = {
                {idx_ch.key, telem::TIMESTAMP_T, 0},
                {input_a.key, telem::FLOAT32_T, idx_ch.key},
                {input_b.key, telem::FLOAT32_T, idx_ch.key},
                {output_ch.key, telem::FLOAT32_T, idx_ch.key}
            }
        }
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("add"));

    wasm::Node node(*func_node, std::move(node_state), func);

    telem::Frame frame(3);
    auto now = telem::TimeStamp::now();
    auto idx_series = telem::Series(now);
    idx_series.alignment = telem::Alignment(1, 0);
    auto a_series = telem::Series(std::vector<float>{1.0f, 2.0f, 3.0f});
    a_series.alignment = telem::Alignment(1, 0);
    auto b_series = telem::Series(std::vector<float>{10.0f, 20.0f, 30.0f});
    b_series.alignment = telem::Alignment(1, 0);
    frame.emplace(idx_ch.key, std::move(idx_series));
    frame.emplace(input_a.key, std::move(a_series));
    frame.emplace(input_b.key, std::move(b_series));
    state.ingest(frame);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    auto writes = state.flush_writes();
    for (const auto &[key, series]: writes) {
        if (key == output_ch.key) {
            ASSERT_EQ(series->size(), 3);
            EXPECT_FLOAT_EQ(series->at<float>(0), 11.0f);
            EXPECT_FLOAT_EQ(series->at<float>(1), 22.0f);
            EXPECT_FLOAT_EQ(series->at<float>(2), 33.0f);
        }
    }
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

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("divide_by_zero"));

    wasm::Node node(*func_node, std::move(node_state), func);

    telem::Frame frame(2);
    auto now = telem::TimeStamp::now();
    auto idx_series = telem::Series(now);
    idx_series.alignment = telem::Alignment(1, 0);
    auto val_series = telem::Series(std::vector<int32_t>{42});
    val_series.alignment = telem::Alignment(1, 0);
    frame.emplace(index_ch.key, std::move(idx_series));
    frame.emplace(input_ch.key, std::move(val_series));
    state.ingest(frame);

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

    wasm::Node node(*func_node, std::move(node_state), func);
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

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("passthrough"));

    wasm::Node node(*func_node, std::move(node_state), func);

    telem::Frame frame(2);
    auto now = telem::TimeStamp::now();
    auto idx_series = telem::Series(now);
    idx_series.alignment = telem::Alignment(1, 0);
    auto val_series = telem::Series(std::vector<float>{42.0f});
    val_series.alignment = telem::Alignment(1, 0);
    frame.emplace(index_ch.key, std::move(idx_series));
    frame.emplace(input_ch.key, std::move(val_series));
    state.ingest(frame);

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    const auto &output_param = func_node->outputs[0];
    EXPECT_TRUE(node.is_output_truthy(output_param.name));
}
