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
    return prefix + "_" + std::to_string(dis(gen_rand));
}

node::Context make_context() {
    return node::Context{
        .elapsed = telem::SECOND,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const xerrors::Error &) {},
        .activate_stage = [] {},
    };
}

/// @brief Compiles an Arc program via the Synnax client.
arc::module::Module
compile_arc(const synnax::Synnax &client, const std::string &source) {
    auto arc = synnax::Arc(random_name("test_arc"));
    arc.text.raw = source;
    const auto create_err = client.arcs.create(arc);
    if (create_err)
        throw std::runtime_error("Failed to create arc: " + create_err.message());

    synnax::RetrieveOptions opts;
    opts.compile = true;
    auto [compiled, err] = client.arcs.retrieve_by_key(arc.key, opts);
    if (err) throw std::runtime_error("Failed to compile arc: " + err.message());
    return compiled.module;
}

/// @brief Test fixture that sets up a compiled Arc module with channels.
struct NodeTestSetup {
    synnax::Synnax client;
    synnax::Channel input_ch;
    synnax::Channel output_ch;
    arc::module::Module mod;
    std::shared_ptr<wasm::Module> wasm_mod;
    state::State state;
    state::Node node_state;
    wasm::Module::Function func;

    NodeTestSetup():
        client(new_test_client()),
        input_ch(ASSERT_NIL_P(
            client.channels.create(random_name("input"), telem::FLOAT32_T, true)
        )),
        output_ch(ASSERT_NIL_P(
            client.channels.create(random_name("output"), telem::FLOAT32_T, true)
        )),
        mod(compile_arc(client, make_source())),
        wasm_mod(ASSERT_NIL_P(wasm::Module::open({.module = mod}))),
        state(state::Config{.ir = mod, .channels = {}}),
        node_state(ASSERT_NIL_P(state.node(find_func_node_key()))),
        func(ASSERT_NIL_P(wasm_mod->func("double"))) {}

private:
    std::string make_source() const {
        return R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + input_ch.name +
               " -> double{} -> " + output_ch.name;
    }

    std::string find_func_node_key() const {
        for (const auto &node: mod.nodes)
            if (node.type == "double") return node.key;
        throw std::runtime_error("Could not find 'double' node in IR");
    }
};
}

/// @brief Factory returns NOT_FOUND when function doesn't exist.
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

    state::State state(state::Config{.ir = mod, .channels = {}});
    auto [node_state, state_err] = state.node(mod.nodes[0].key);
    ASSERT_NIL(state_err);

    node::Config cfg(fake_node, std::move(node_state));
    auto [node, err] = factory.create(std::move(cfg));
    ASSERT_TRUE(err.matches(xerrors::NOT_FOUND));
    ASSERT_EQ(node, nullptr);
}

/// @brief Factory creates node with valid function.
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

    state::State state(state::Config{.ir = mod, .channels = {}});

    const arc::ir::Node *func_node = nullptr;
    for (const auto &node: mod.nodes) {
        if (node.type == "double") {
            func_node = &node;
            break;
        }
    }
    ASSERT_NE(func_node, nullptr);

    auto [node_state, state_err] = state.node(func_node->key);
    ASSERT_NIL(state_err);

    node::Config cfg(*func_node, std::move(node_state));
    auto [node, err] = factory.create(std::move(cfg));
    ASSERT_NIL(err);
    ASSERT_NE(node, nullptr);
}

/// @brief Node::next returns early when no inputs refreshed.
TEST(NodeTest, NextReturnsEarlyWhenNoInputsRefreshed) {
    NodeTestSetup setup;

    auto ctx = make_context();
    int mark_called = 0;
    ctx.mark_changed = [&](const std::string &) { mark_called++; };

    const arc::ir::Node *func_node = nullptr;
    for (const auto &node: setup.mod.nodes) {
        if (node.type == "double") {
            func_node = &node;
            break;
        }
    }
    ASSERT_NE(func_node, nullptr);

    wasm::Node node(*func_node, std::move(setup.node_state), setup.func);
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(mark_called, 0);
}

/// @brief Node::is_output_truthy delegates to state.
TEST(NodeTest, IsOutputTruthyDelegatesToState) {
    NodeTestSetup setup;

    const arc::ir::Node *func_node = nullptr;
    for (const auto &node: setup.mod.nodes) {
        if (node.type == "double") {
            func_node = &node;
            break;
        }
    }
    ASSERT_NE(func_node, nullptr);

    wasm::Node node(*func_node, std::move(setup.node_state), setup.func);
    EXPECT_FALSE(node.is_output_truthy("nonexistent"));
}
