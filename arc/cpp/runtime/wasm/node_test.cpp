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
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/wasm/factory.h"
#include "arc/cpp/runtime/wasm/module.h"
#include "arc/cpp/runtime/wasm/node.h"
#include "arc/cpp/stl/channel/channel.h"
#include "arc/cpp/stl/channel/state.h"
#include "arc/cpp/stl/error/error.h"
#include "arc/cpp/stl/math/math.h"
#include "arc/cpp/stl/series/series.h"
#include "arc/cpp/stl/stateful/stateful.h"
#include "arc/cpp/stl/str/str.h"
#include "arc/cpp/stl/time/time.h"

namespace arc::runtime {
std::mt19937 gen_rand = random_generator("Node Tests");

std::string random_name(const std::string &prefix) {
    std::uniform_int_distribution<> dis(10000, 99999);
    return prefix + std::to_string(dis(gen_rand));
}

/// @brief Compiles an Arc program via the Synnax client.
arc::program::Program
compile_arc(const synnax::Synnax &client, const std::string &source) {
    auto arc = synnax::arc::Arc{
        .name = random_name("test_arc"),
        .mode = synnax::arc::MODE_TEXT
    };
    arc.text.raw = source;
    if (const auto create_err = client.arcs.create(arc))
        throw std::runtime_error("Failed to create arc: " + create_err.message());

    synnax::arc::RetrieveOptions opts;
    opts.compile = true;
    auto [compiled, err] = client.arcs.retrieve_by_key(arc.key, opts);
    if (err) throw std::runtime_error("Failed to compile arc: " + err.message());
    return *compiled.program;
}

/// @brief Finds the IR node with the given type in the module.
const arc::ir::Node *
find_node_by_type(const arc::program::Program &mod, const std::string &type) {
    for (const auto &node: mod.nodes)
        if (node.type == type) return &node;
    return nullptr;
}

node::Context make_context() {
    return node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [](size_t) {},
        .report_error = [](const x::errors::Error &) {},
    };
}

/// @brief Builds a set of STL modules from the given state.
std::vector<std::shared_ptr<stl::Module>> build_stl_modules(
    const std::shared_ptr<stl::channel::State> &channel_st,
    const std::shared_ptr<stl::str::State> &str_st,
    const std::shared_ptr<stl::series::State> &series_st,
    const std::shared_ptr<stl::stateful::Variables> &var_st
) {
    return {
        std::make_shared<stl::channel::WasmModule>(channel_st, str_st),
        std::make_shared<stl::stateful::WasmModule>(var_st, series_st, str_st),
        std::make_shared<stl::series::WasmModule>(series_st),
        std::make_shared<stl::str::WasmModule>(str_st),
        std::make_shared<stl::math::WasmModule>(),
        std::make_shared<stl::time::WasmModule>(),
        std::make_shared<stl::error::WasmModule>(arc::runtime::errors::noop_handler),
    };
}

/// @brief compiles a single Arc function definition, calls it, and returns the first
/// output value. Parses the function name and return type from the definition to create
/// the necessary flow and channel automatically.
template<typename T>
T call_func(
    const synnax::Synnax &client,
    const std::string &func_def,
    const std::vector<x::telem::SampleValue> &params = {}
) {
    static const std::unordered_map<std::string, x::telem::DataType> arc_types = {
        {"i8", x::telem::INT8_T},
        {"i16", x::telem::INT16_T},
        {"i32", x::telem::INT32_T},
        {"i64", x::telem::INT64_T},
        {"u8", x::telem::UINT8_T},
        {"u16", x::telem::UINT16_T},
        {"u32", x::telem::UINT32_T},
        {"u64", x::telem::UINT64_T},
        {"f32", x::telem::FLOAT32_T},
        {"f64", x::telem::FLOAT64_T},
    };

    // Parse function name: first word after "func "
    auto func_pos = func_def.find("func ");
    if (func_pos == std::string::npos)
        throw std::runtime_error("call_func: no 'func ' in definition");
    auto name_start = func_pos + 5;
    auto name_end = func_def.find_first_of("( ", name_start);
    auto func_name = func_def.substr(name_start, name_end - name_start);

    // Parse return type: last type token before '{'
    auto brace = func_def.find('{');
    if (brace == std::string::npos)
        throw std::runtime_error("call_func: no '{' in definition");
    auto pre = func_def.substr(0, brace);
    auto pos = pre.find_last_not_of(" \t\n");
    auto start = pre.find_last_of(" \t\n()", pos);
    auto ret_type = pre.substr(start + 1, pos - start);
    auto it = arc_types.find(ret_type);
    auto dt = it != arc_types.end() ? it->second : x::telem::INT64_T;

    // Create virtual output channel and build source
    auto ch = synnax::channel::Channel{
        .name = random_name("out"),
        .data_type = dt,
        .is_virtual = true,
    };
    if (const auto err = client.channels.create(ch))
        throw std::runtime_error("Failed to create channel: " + err.message());
    auto source = func_def + "\n" + func_name + "{} -> " + ch.name;

    auto str_st = std::make_shared<stl::str::State>();
    auto series_st = std::make_shared<stl::series::State>();
    auto var_st = std::make_shared<stl::stateful::Variables>();
    auto channel_st = std::make_shared<stl::channel::State>();
    auto stl_modules = build_stl_modules(channel_st, str_st, series_st, var_st);

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(
        wasm::Module::open({
            .program = mod,
            .modules = stl_modules,
            .strings = str_st,
        })
    );

    auto func = ASSERT_NIL_P(wasm_mod->func(func_name));
    std::vector<wasm::Module::Function::Result> results;
    auto err = func.call(params, results);
    if (err) throw std::runtime_error("WASM call failed: " + err.message());
    if (results.empty() || !results[0].changed)
        throw std::runtime_error("No output produced");
    return std::get<T>(results[0].value);
}

/// @brief Factory::handles returns true for functions in the module.
TEST(FactoryTest, HandlesReturnsTrueForExistingFunction) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    wasm::Factory factory(wasm_mod);

    EXPECT_TRUE(factory.handles("double"));
    EXPECT_FALSE(factory.handles("nonexistent"));
}

/// @brief Factory::create returns NOT_FOUND when function doesn't exist.
TEST(FactoryTest, CreateReturnsErrorWhenFunctionNotFound) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    wasm::Factory factory(wasm_mod);

    arc::ir::Node fake_node;
    fake_node.key = "fake_node";
    fake_node.type = "nonexistent";

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}},
        arc::runtime::errors::noop_handler
    );
    auto node_state = ASSERT_NIL_P(state.node(mod.nodes[0].key));

    const arc::ir::IR prog = static_cast<arc::ir::IR>(mod);
    node::Config cfg(prog, fake_node, std::move(node_state));
    ASSERT_OCCURRED_AS_P(factory.create(std::move(cfg)), x::errors::NOT_FOUND);
}

/// @brief Factory::create succeeds with valid function.
TEST(FactoryTest, CreateSucceedsWithValidConfig) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    wasm::Factory factory(wasm_mod);

    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}},
        arc::runtime::errors::noop_handler
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
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}},
        arc::runtime::errors::noop_handler
    );
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("double"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](size_t i) {
        changed_outputs.push_back(func_node->outputs[i].name);
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

    auto input_idx = synnax::channel::Channel{
        .name = input_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(input_idx));
    auto output_idx = synnax::channel::Channel{
        .name = output_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));

    auto input_ch = synnax::channel::Channel{
        .name = input_name,
        .data_type = x::telem::FLOAT32_T,
        .index = input_idx.key,
    };
    ASSERT_NIL(client.channels.create(input_ch));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::FLOAT32_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + input_name + " -> double{} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{input_idx.key, x::telem::TIMESTAMP_T, 0},
                 {input_ch.key, x::telem::FLOAT32_T, input_idx.key},
                 {output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::FLOAT32_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    // Find the 'on' node that reads from the input channel
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    // Get the 'on' node's state and set its outputs directly
    // This simulates what on.next() would do after reading from channels
    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));

    auto on_data = x::telem::Series(std::vector{5.0f, 10.0f, 15.0f});
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );

    auto on_time = x::telem::Series(
        std::vector{
            x::telem::TimeStamp(1 * x::telem::MICROSECOND),
            x::telem::TimeStamp(2 * x::telem::MICROSECOND),
            x::telem::TimeStamp(3 * x::telem::MICROSECOND)
        }
    );
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    // Now set up the double node
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("double"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](size_t i) {
        changed_outputs.push_back(func_node->outputs[i].name);
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

    auto index_ch = synnax::channel::Channel{
        .name = idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(index_ch));

    auto input_ch = synnax::channel::Channel{
        .name = input_name,
        .data_type = x::telem::INT32_T,
        .index = index_ch.key,
    };
    ASSERT_NIL(client.channels.create(input_ch));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT32_T,
        .index = index_ch.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func divide_by_zero(val i32) i32 {
    return val / 0
}
)" + input_name + " -> divide_by_zero{} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "divide_by_zero");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{index_ch.key, x::telem::TIMESTAMP_T, 0},
                 {input_ch.key, x::telem::INT32_T, index_ch.key},
                 {output_ch.key, x::telem::INT32_T, index_ch.key}}
        },
        arc::runtime::errors::noop_handler
    );

    // Find the 'on' node and set its outputs
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = x::telem::Series(static_cast<int32_t>(42));
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );
    auto on_time = x::telem::Series(x::telem::TimeStamp(1 * x::telem::MICROSECOND));
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("divide_by_zero"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<x::errors::Error> reported_errors;
    ctx.report_error = [&](const x::errors::Error &err) {
        reported_errors.push_back(err);
    };

    ASSERT_NIL(node.next(ctx));
    EXPECT_GE(reported_errors.size(), 1);
}

/// @brief Node::is_output_truthy returns false for nonexistent outputs.
TEST(NodeTest, IsOutputTruthyReturnsFalseForNonexistent) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{.ir = (static_cast<arc::ir::IR>(mod)), .channels = {}},
        arc::runtime::errors::noop_handler
    );
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("double"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());
    EXPECT_FALSE(node.is_output_truthy(7));
}

/// @brief Node::is_output_truthy correctly evaluates output values.
TEST(NodeTest, IsOutputTruthyEvaluatesOutputValues) {
    const auto client = new_test_client();

    auto idx_name = random_name("time");
    auto input_name = random_name("input");
    auto output_name = random_name("output");

    auto index_ch = synnax::channel::Channel{
        .name = idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(index_ch));

    auto input_ch = synnax::channel::Channel{
        .name = input_name,
        .data_type = x::telem::FLOAT32_T,
        .index = index_ch.key,
    };
    ASSERT_NIL(client.channels.create(input_ch));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::FLOAT32_T,
        .index = index_ch.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func passthrough(val f32) f32 {
    return val
}
)" + input_name + " -> passthrough{} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "passthrough");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{index_ch.key, x::telem::TIMESTAMP_T, 0},
                 {input_ch.key, x::telem::FLOAT32_T, index_ch.key},
                 {output_ch.key, x::telem::FLOAT32_T, index_ch.key}}
        },
        arc::runtime::errors::noop_handler
    );

    // Find the 'on' node and set its outputs
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = x::telem::Series(42.0f);
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );
    auto on_time = x::telem::Series(x::telem::TimeStamp(1 * x::telem::MICROSECOND));
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("passthrough"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    EXPECT_TRUE(node.is_output_truthy(0));
}

/// @brief entry nodes with no inputs execute only once per stage entry.
TEST(NodeTest, NoInputNodeExecutesOncePerStageEntry) {
    const auto client = new_test_client();

    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto output_idx = synnax::channel::Channel{
        .name = output_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT64_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func constant() i64 {
    return 42
}
constant{} -> )" + output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "constant");
    ASSERT_NE(func_node, nullptr);
    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT64_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );
    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("constant"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](size_t i) {
        changed_outputs.push_back(func_node->outputs[i].name);
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

/// @brief nodes with inputs execute on every call to next().
TEST(NodeTest, NodeWithInputsExecutesNormally) {
    const auto client = new_test_client();

    auto input_idx_name = random_name("input_idx");
    auto input_name = random_name("input");
    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto input_idx = synnax::channel::Channel{
        .name = input_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(input_idx));
    auto output_idx = synnax::channel::Channel{
        .name = output_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));

    auto input_ch = synnax::channel::Channel{
        .name = input_name,
        .data_type = x::telem::INT64_T,
        .index = input_idx.key,
    };
    ASSERT_NIL(client.channels.create(input_ch));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT64_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func double(val i64) i64 {
    return val * 2
}
)" + input_name + " -> double{} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "double");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{input_idx.key, x::telem::TIMESTAMP_T, 0},
                 {input_ch.key, x::telem::INT64_T, input_idx.key},
                 {output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT64_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = x::telem::Series(static_cast<int64_t>(5));
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );
    auto on_time = x::telem::Series(x::telem::TimeStamp(1 * x::telem::MICROSECOND));
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("double"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](size_t i) {
        changed_outputs.push_back(func_node->outputs[i].name);
    };

    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(changed_outputs.size(), 1);

    auto on_node_state2 = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data2 = x::telem::Series(static_cast<int64_t>(10));
    on_data2.alignment = x::telem::Alignment(2, 0);
    on_node_state2.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data2)
    );
    auto on_time2 = x::telem::Series(x::telem::TimeStamp(2 * x::telem::MICROSECOND));
    on_time2.alignment = x::telem::Alignment(2, 0);
    on_node_state2.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time2)
    );

    changed_outputs.clear();
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(changed_outputs.size(), 1);
}

/// @brief expression_ prefixed nodes bypass the entry-once gate.
TEST(NodeTest, FlowExpressionExecutesEveryTime) {
    const auto client = new_test_client();

    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto output_idx = synnax::channel::Channel{
        .name = output_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT64_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func counter() i64 {
    return 42
}
counter{} -> )" + output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "counter");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT64_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("counter"));

    arc::ir::Node expr_node = *func_node;
    expr_node.key = "expression_0";

    wasm::Node node(mod, expr_node, std::move(node_state), func, wasm_mod->strings());

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

/// @brief expression nodes re-execute after reset.
TEST(NodeTest, FlowExpressionContinuesAfterReset) {
    const auto client = new_test_client();

    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto output_idx = synnax::channel::Channel{
        .name = output_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT64_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func counter() i64 {
    return 42
}
counter{} -> )" + output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "counter");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT64_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("counter"));

    arc::ir::Node expr_node = *func_node;
    expr_node.key = "expression_0";

    wasm::Node node(mod, expr_node, std::move(node_state), func, wasm_mod->strings());

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

/// @brief non-expression_ prefixed entry nodes use the execute-once gate.
TEST(NodeTest, NonExpressionNodeNotTreatedAsExpression) {
    const auto client = new_test_client();

    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto output_idx = synnax::channel::Channel{
        .name = output_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT64_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func counter() i64 {
    return 42
}
counter{} -> )" + output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "counter");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT64_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("counter"));

    arc::ir::Node non_expr_node = *func_node;
    non_expr_node.key = "expr_0";

    wasm::Node
        node(mod, non_expr_node, std::move(node_state), func, wasm_mod->strings());

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

    auto input_idx = synnax::channel::Channel{
        .name = input_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(input_idx));
    auto output_idx = synnax::channel::Channel{
        .name = output_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));

    auto input_ch = synnax::channel::Channel{
        .name = input_name,
        .data_type = x::telem::INT32_T,
        .index = input_idx.key,
    };
    ASSERT_NIL(client.channels.create(input_ch));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT32_T,
        .index = output_idx.key,
    };
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
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "add_config");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{input_idx.key, x::telem::TIMESTAMP_T, 0},
                 {input_ch.key, x::telem::INT32_T, input_idx.key},
                 {output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT32_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    // Find and set up the 'on' node that reads from the input channel
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = x::telem::Series(std::vector<int32_t>{5});
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );
    auto on_time = x::telem::Series(
        std::vector{x::telem::TimeStamp(1 * x::telem::MICROSECOND)}
    );
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("add_config", func_node->config));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

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

    auto input_idx = synnax::channel::Channel{
        .name = input_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(input_idx));
    auto output_idx = synnax::channel::Channel{
        .name = output_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));

    auto input_ch = synnax::channel::Channel{
        .name = input_name,
        .data_type = x::telem::INT32_T,
        .index = input_idx.key,
    };
    ASSERT_NIL(client.channels.create(input_ch));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT32_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    // Function with two config parameters 'a', 'b' and input parameter 'c'
    const std::string source = R"(
func multi_config{a i32, b i32}(c i32) i32 {
    return a + b + c
}
)" + input_name + " -> multi_config{a=5, b=10} -> " +
                               output_name;

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    const auto *func_node = find_node_by_type(mod, "multi_config");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{input_idx.key, x::telem::TIMESTAMP_T, 0},
                 {input_ch.key, x::telem::INT32_T, input_idx.key},
                 {output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT32_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = x::telem::Series(std::vector<int32_t>{3});
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );
    auto on_time = x::telem::Series(
        std::vector{x::telem::TimeStamp(1 * x::telem::MICROSECOND)}
    );
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("multi_config", func_node->config));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));

    // Verify the output: a=5 + b=10 + c=3 = 18
    auto result_state = ASSERT_NIL_P(state.node(func_node->key));
    const auto &output = result_state.output(0);
    ASSERT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<int32_t>(0), 18);
}

/// @brief Regression test: Two node instances of the same function type should
/// have independent stateful variable storage.
TEST(NodeTest, StatefulVariablesAreIsolatedBetweenNodeInstances) {
    const auto client = new_test_client();

    auto idx_name = random_name("time");
    auto trigger_name = random_name("trigger");
    auto output_a_name = random_name("output_a");
    auto output_b_name = random_name("output_b");

    auto index_ch = synnax::channel::Channel{
        .name = idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(index_ch));

    auto trigger_ch = synnax::channel::Channel{
        .name = trigger_name,
        .data_type = x::telem::INT64_T,
        .index = index_ch.key,
    };
    ASSERT_NIL(client.channels.create(trigger_ch));

    auto output_a_ch = synnax::channel::Channel{
        .name = output_a_name,
        .data_type = x::telem::INT64_T,
        .index = index_ch.key,
    };
    ASSERT_NIL(client.channels.create(output_a_ch));

    auto output_b_ch = synnax::channel::Channel{
        .name = output_b_name,
        .data_type = x::telem::INT64_T,
        .index = index_ch.key,
    };
    ASSERT_NIL(client.channels.create(output_b_ch));

    const std::string source = R"(
func counter(trigger i64) i64 {
    count i64 $= 0
    count = count + 1
    return count
}
)" + trigger_name + " -> counter{} -> " +
                               output_a_name + "\n" + trigger_name +
                               " -> counter{} -> " + output_b_name;

    auto mod = compile_arc(client, source);

    auto channel_st = std::make_shared<stl::channel::State>(
        std::vector<state::ChannelDigest>{
            {index_ch.key, x::telem::TIMESTAMP_T, 0},
            {trigger_ch.key, x::telem::INT64_T, index_ch.key},
            {output_a_ch.key, x::telem::INT64_T, index_ch.key},
            {output_b_ch.key, x::telem::INT64_T, index_ch.key}
        }
    );
    auto str_st = std::make_shared<stl::str::State>();
    auto series_st = std::make_shared<stl::series::State>();
    auto var_st = std::make_shared<stl::stateful::Variables>();
    auto state = std::make_shared<state::State>(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{index_ch.key, x::telem::TIMESTAMP_T, 0},
                 {trigger_ch.key, x::telem::INT64_T, index_ch.key},
                 {output_a_ch.key, x::telem::INT64_T, index_ch.key},
                 {output_b_ch.key, x::telem::INT64_T, index_ch.key}}
        },
        channel_st,
        str_st,
        series_st,
        var_st,
        arc::runtime::errors::noop_handler
    );

    auto wasm_mod = ASSERT_NIL_P(
        wasm::Module::open(
            {.program = mod,
             .modules = build_stl_modules(channel_st, str_st, series_st, var_st)}
        )
    );

    // Find the two counter nodes
    std::vector<const arc::ir::Node *> counter_nodes;
    for (const auto &node: mod.nodes)
        if (node.type == "counter") counter_nodes.push_back(&node);
    ASSERT_EQ(counter_nodes.size(), 2);
    const auto *counter_a_node = counter_nodes[0];
    const auto *counter_b_node = counter_nodes[1];

    // Each pipeline has its own on_trigger node. Find and set up all of them.
    for (const auto &node: mod.nodes) {
        if (node.type != "on") continue;
        auto on_node_state = ASSERT_NIL_P(state->node(node.key));
        auto on_data = x::telem::Series(std::vector<int64_t>{1});
        on_data.alignment = x::telem::Alignment(1, 0);
        on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
            std::move(on_data)
        );
        auto on_time = x::telem::Series(
            std::vector{x::telem::TimeStamp(1 * x::telem::MICROSECOND)}
        );
        on_time.alignment = x::telem::Alignment(1, 0);
        on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
            std::move(on_time)
        );
    }

    // Create state and function objects for both counter nodes
    auto node_a_state = ASSERT_NIL_P(state->node(counter_a_node->key));
    auto node_b_state = ASSERT_NIL_P(state->node(counter_b_node->key));
    auto func_a = ASSERT_NIL_P(wasm_mod->func("counter"));
    auto func_b = ASSERT_NIL_P(wasm_mod->func("counter"));

    wasm::Node node_a(
        mod,
        *counter_a_node,
        std::move(node_a_state),
        func_a,
        wasm_mod->strings()
    );
    wasm::Node node_b(
        mod,
        *counter_b_node,
        std::move(node_b_state),
        func_b,
        wasm_mod->strings()
    );

    auto ctx = make_context();

    // Execute counter_a - should return 1
    ASSERT_NIL(node_a.next(ctx));
    auto result_a = ASSERT_NIL_P(state->node(counter_a_node->key));
    ASSERT_EQ(result_a.output(0)->size(), 1);
    EXPECT_EQ(result_a.output(0)->at<int64_t>(0), 1);

    // Execute counter_b - should also return 1 (not 2!), proving isolation
    ASSERT_NIL(node_b.next(ctx));
    auto result_b = ASSERT_NIL_P(state->node(counter_b_node->key));
    ASSERT_EQ(result_b.output(0)->size(), 1);
    EXPECT_EQ(result_b.output(0)->at<int64_t>(0), 1)
        << "counter_b should have its own independent state, returning 1 not 2";
}

/// @brief Regression test: channel-typed config params should correctly read
/// channel data via the WASM channel_read host function.
TEST(NodeTest, ChannelConfigParamReadsChannelData) {
    const auto client = new_test_client();

    auto trigger_idx_name = random_name("trigger_idx");
    auto trigger_name = random_name("trigger");
    auto data_idx_name = random_name("data_idx");
    auto data_name = random_name("data");
    auto output_idx_name = random_name("output_idx");
    auto output_name = random_name("output");

    auto trigger_idx = synnax::channel::Channel{
        .name = trigger_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(trigger_idx));
    auto data_idx = synnax::channel::Channel{
        .name = data_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(data_idx));
    auto output_idx = synnax::channel::Channel{
        .name = output_idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));

    auto trigger_ch = synnax::channel::Channel{
        .name = trigger_name,
        .data_type = x::telem::UINT8_T,
        .index = trigger_idx.key,
    };
    ASSERT_NIL(client.channels.create(trigger_ch));
    auto data_ch = synnax::channel::Channel{
        .name = data_name,
        .data_type = x::telem::FLOAT32_T,
        .index = data_idx.key,
    };
    ASSERT_NIL(client.channels.create(data_ch));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::FLOAT32_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    // Function with a channel-typed config param.
    // 'ch + f32(0.0)' forces a channel read (rather than a channel alias).
    const std::string source = R"(
func read_chan{ch chan f32}(trigger u8) f32 {
    return ch + f32(0.0)
}
)" + trigger_name +
                               " -> read_chan{ch=" + data_name + "} -> " + output_name;

    auto mod = compile_arc(client, source);

    // Verify the node's channels.read includes the config param channel.
    const auto *func_node = find_node_by_type(mod, "read_chan");
    ASSERT_NE(func_node, nullptr);
    EXPECT_TRUE(func_node->channels.read.contains(data_ch.key))
        << "Node channels.read should include the config param channel (key="
        << data_ch.key << "). IR:\n"
        << mod.to_json().dump();

    ASSERT_EQ(func_node->config.size(), 1);
    ASSERT_FALSE(func_node->config[0].value.is_null());
    EXPECT_EQ(
        static_cast<int32_t>(func_node->config[0].value.get<double>()),
        static_cast<int32_t>(data_ch.key)
    ) << "Config param value should be the channel ID";

    auto channel_st = std::make_shared<stl::channel::State>(
        std::vector<state::ChannelDigest>{
            {trigger_idx.key, x::telem::TIMESTAMP_T, 0},
            {trigger_ch.key, x::telem::UINT8_T, trigger_idx.key},
            {data_idx.key, x::telem::TIMESTAMP_T, 0},
            {data_ch.key, x::telem::FLOAT32_T, data_idx.key},
            {output_idx.key, x::telem::TIMESTAMP_T, 0},
            {output_ch.key, x::telem::FLOAT32_T, output_idx.key}
        }
    );
    auto str_st = std::make_shared<stl::str::State>();
    auto series_st = std::make_shared<stl::series::State>();
    auto var_st = std::make_shared<stl::stateful::Variables>();
    auto state = std::make_shared<state::State>(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{trigger_idx.key, x::telem::TIMESTAMP_T, 0},
                 {trigger_ch.key, x::telem::UINT8_T, trigger_idx.key},
                 {data_idx.key, x::telem::TIMESTAMP_T, 0},
                 {data_ch.key, x::telem::FLOAT32_T, data_idx.key},
                 {output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::FLOAT32_T, output_idx.key}}
        },
        channel_st,
        str_st,
        series_st,
        var_st,
        arc::runtime::errors::noop_handler
    );

    auto wasm_mod = ASSERT_NIL_P(
        wasm::Module::open(
            {.program = mod,
             .modules = build_stl_modules(channel_st, str_st, series_st, var_st)}
        )
    );

    // Ingest data for the config param channel so channel_read_f32 can find it.
    auto data_series = x::telem::Series(std::vector{42.5f});
    state->ingest(x::telem::Frame(data_ch.key, std::move(data_series)));

    // Set up the 'on' node that reads the trigger channel.
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state->node(on_node->key));
    auto on_data = x::telem::Series(std::vector<uint8_t>{1});
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );
    auto on_time = x::telem::Series(
        std::vector{x::telem::TimeStamp(1 * x::telem::MICROSECOND)}
    );
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    // Set up the function node with config param.
    auto node_state = ASSERT_NIL_P(state->node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("read_chan", func_node->config));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](size_t i) {
        changed_outputs.push_back(func_node->outputs[i].name);
    };

    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(changed_outputs.size(), 1);

    // Verify the output reads the channel value (42.5 + 0.0 = 42.5).
    auto result_state = ASSERT_NIL_P(state->node(func_node->key));
    const auto &output = result_state.output(0);
    ASSERT_EQ(output->size(), 1);
    EXPECT_FLOAT_EQ(output->at<float>(0), 42.5f)
        << "Channel config param should read the channel value, not the channel ID";
}

/// @brief Node converts string channel input data to i32 handles for WASM.
TEST(NodeTest, StringChannelInputConvertedToHandles) {
    const auto client = new_test_client();

    auto idx_name = random_name("time");
    auto input_name = random_name("str_input");
    auto output_name = random_name("len_output");

    auto index_ch = synnax::channel::Channel{
        .name = idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(index_ch));

    auto input_ch = synnax::channel::Channel{
        .name = input_name,
        .data_type = x::telem::STRING_T,
        .is_virtual = true,
    };
    ASSERT_NIL(client.channels.create(input_ch));

    auto output_idx = synnax::channel::Channel{
        .name = random_name("out_idx"),
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));

    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT64_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func str_len(s str) i64 {
    return len(s)
}
)" + input_name + " -> str_len{} -> " +
                               output_name;

    auto str_st = std::make_shared<stl::str::State>();
    auto series_st = std::make_shared<stl::series::State>();
    auto var_st = std::make_shared<stl::stateful::Variables>();
    auto channel_st = std::make_shared<stl::channel::State>();
    auto stl_modules = build_stl_modules(channel_st, str_st, series_st, var_st);

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(
        wasm::Module::open({
            .program = mod,
            .modules = stl_modules,
            .strings = str_st,
        })
    );

    const auto *func_node = find_node_by_type(mod, "str_len");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{index_ch.key, x::telem::TIMESTAMP_T, 0},
                 {input_ch.key, x::telem::STRING_T, index_ch.key},
                 {output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT64_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    // Set up the 'on' node with string input data
    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = x::telem::Series(std::vector<std::string>{"hello", "world!", ""});
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );
    auto on_time = x::telem::Series(
        std::vector{
            x::telem::TimeStamp(1 * x::telem::SECOND),
            x::telem::TimeStamp(2 * x::telem::SECOND),
            x::telem::TimeStamp(3 * x::telem::SECOND)
        }
    );
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("str_len"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](size_t i) {
        changed_outputs.push_back(func_node->outputs[i].name);
    };

    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(changed_outputs.size(), 1);

    auto result_state = ASSERT_NIL_P(state.node(func_node->key));
    const auto &output = result_state.output(0);
    ASSERT_EQ(output->size(), 3);
    EXPECT_EQ(output->at<int64_t>(0), 5); // "hello"
    EXPECT_EQ(output->at<int64_t>(1), 6); // "world!"
    EXPECT_EQ(output->at<int64_t>(2), 0); // ""
}

/// @brief Node handles string-typed named outputs without panicking on Density().
TEST(NodeTest, DISABLED_NamedStringOutputMemoryLayout) {
    const auto client = new_test_client();

    auto idx_name = random_name("time");
    auto input_name = random_name("input");
    auto label_out_name = random_name("label_out");
    auto value_out_name = random_name("value_out");

    auto index_ch = synnax::channel::Channel{
        .name = idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(index_ch));

    auto input_ch = synnax::channel::Channel{
        .name = input_name,
        .data_type = x::telem::INT64_T,
        .index = index_ch.key,
    };
    ASSERT_NIL(client.channels.create(input_ch));

    auto out_idx = synnax::channel::Channel{
        .name = random_name("out_idx"),
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(out_idx));

    auto label_ch = synnax::channel::Channel{
        .name = label_out_name,
        .data_type = x::telem::STRING_T,
        .is_virtual = true,
    };
    ASSERT_NIL(client.channels.create(label_ch));

    auto value_ch = synnax::channel::Channel{
        .name = value_out_name,
        .data_type = x::telem::INT64_T,
        .index = out_idx.key,
    };
    ASSERT_NIL(client.channels.create(value_ch));

    const std::string source =
        R"arc(
func labeler(x i64) (label str, value i64) {
    label = "ok"
    value = x * 2
}
)arc" + input_name +
        " -> labeler{} -> " + value_out_name;

    auto str_st = std::make_shared<stl::str::State>();
    auto series_st = std::make_shared<stl::series::State>();
    auto var_st = std::make_shared<stl::stateful::Variables>();
    auto channel_st = std::make_shared<stl::channel::State>();
    auto stl_modules = build_stl_modules(channel_st, str_st, series_st, var_st);

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(
        wasm::Module::open({
            .program = mod,
            .modules = stl_modules,
            .strings = str_st,
        })
    );

    const auto *func_node = find_node_by_type(mod, "labeler");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{index_ch.key, x::telem::TIMESTAMP_T, 0},
                 {input_ch.key, x::telem::INT64_T, index_ch.key},
                 {out_idx.key, x::telem::TIMESTAMP_T, 0},
                 {label_ch.key, x::telem::STRING_T, out_idx.key},
                 {value_ch.key, x::telem::INT64_T, out_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = x::telem::Series(static_cast<int64_t>(5));
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );
    auto on_time = x::telem::Series(x::telem::TimeStamp(1 * x::telem::SECOND));
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("labeler"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](size_t i) {
        changed_outputs.push_back(func_node->outputs[i].name);
    };

    // This must not crash — the key assertion is that string-typed named
    // outputs use 4-byte (i32 handle) offsets instead of Density() which
    // returns 0 for strings.
    ASSERT_NIL(node.next(ctx));
    EXPECT_GE(changed_outputs.size(), 1);

    // Verify the numeric output is correct (5 * 2 = 10)
    auto result_state = ASSERT_NIL_P(state.node(func_node->key));
    for (size_t i = 0; i < func_node->outputs.size(); i++) {
        if (func_node->outputs[i].name == "value") {
            const auto &output = result_state.output(i);
            ASSERT_EQ(output->size(), 1);
            EXPECT_EQ(output->at<int64_t>(0), 10);
        }
    }
}

/// @brief Node converts string channel input to handles for qualified string.len().
TEST(NodeTest, StringChannelInputWithQualifiedStringLen) {
    const auto client = new_test_client();

    auto idx_name = random_name("time");
    auto input_name = random_name("str_input");
    auto output_name = random_name("len_output");

    auto index_ch = synnax::channel::Channel{
        .name = idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(index_ch));

    auto input_ch = synnax::channel::Channel{
        .name = input_name,
        .data_type = x::telem::STRING_T,
        .is_virtual = true,
    };
    ASSERT_NIL(client.channels.create(input_ch));

    auto output_idx = synnax::channel::Channel{
        .name = random_name("out_idx"),
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));

    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT64_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    // Uses qualified string.len() instead of builtin len()
    const std::string source = R"(
func qstr_len(s str) i64 {
    return string.len(s)
}
)" + input_name + " -> qstr_len{} -> " +
                               output_name;

    auto str_st = std::make_shared<stl::str::State>();
    auto series_st = std::make_shared<stl::series::State>();
    auto var_st = std::make_shared<stl::stateful::Variables>();
    auto channel_st = std::make_shared<stl::channel::State>();
    auto stl_modules = build_stl_modules(channel_st, str_st, series_st, var_st);

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(
        wasm::Module::open({
            .program = mod,
            .modules = stl_modules,
            .strings = str_st,
        })
    );

    const auto *func_node = find_node_by_type(mod, "qstr_len");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{index_ch.key, x::telem::TIMESTAMP_T, 0},
                 {input_ch.key, x::telem::STRING_T, index_ch.key},
                 {output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT64_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    const auto *on_node = find_node_by_type(mod, "on");
    ASSERT_NE(on_node, nullptr);

    auto on_node_state = ASSERT_NIL_P(state.node(on_node->key));
    auto on_data = x::telem::Series(std::vector<std::string>{"hello", "world!", ""});
    on_data.alignment = x::telem::Alignment(1, 0);
    on_node_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_data)
    );
    auto on_time = x::telem::Series(
        std::vector{
            x::telem::TimeStamp(1 * x::telem::SECOND),
            x::telem::TimeStamp(2 * x::telem::SECOND),
            x::telem::TimeStamp(3 * x::telem::SECOND)
        }
    );
    on_time.alignment = x::telem::Alignment(1, 0);
    on_node_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("qstr_len"));

    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](size_t i) {
        changed_outputs.push_back(func_node->outputs[i].name);
    };

    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(changed_outputs.size(), 1);

    auto result_state = ASSERT_NIL_P(state.node(func_node->key));
    const auto &output = result_state.output(0);
    ASSERT_EQ(output->size(), 3);
    EXPECT_EQ(output->at<int64_t>(0), 5); // "hello"
    EXPECT_EQ(output->at<int64_t>(1), 6); // "world!"
    EXPECT_EQ(output->at<int64_t>(2), 0); // ""
}

/// @brief Node converts string channel inputs to handles for qualified string.concat().
TEST(NodeTest, DISABLED_StringConcatWithChannelData) {
    const auto client = new_test_client();

    auto idx_name = random_name("time");
    auto input_a_name = random_name("str_a");
    auto input_b_name = random_name("str_b");
    auto output_name = random_name("len_output");

    auto index_ch = synnax::channel::Channel{
        .name = idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(index_ch));
    auto input_a_ch = synnax::channel::Channel{
        .name = input_a_name,
        .data_type = x::telem::STRING_T,
        .is_virtual = true,
    };
    ASSERT_NIL(client.channels.create(input_a_ch));
    auto input_b_ch = synnax::channel::Channel{
        .name = input_b_name,
        .data_type = x::telem::STRING_T,
        .is_virtual = true,
    };
    ASSERT_NIL(client.channels.create(input_b_ch));
    auto output_idx = synnax::channel::Channel{
        .name = random_name("out_idx"),
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT64_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func concat_len(a str, b str) i64 {
    return string.len(string.concat(a, b))
}
)" + input_a_name + ", " + input_b_name +
                               " -> concat_len{} -> " + output_name;

    auto str_st = std::make_shared<stl::str::State>();
    auto series_st = std::make_shared<stl::series::State>();
    auto var_st = std::make_shared<stl::stateful::Variables>();
    auto channel_st = std::make_shared<stl::channel::State>();
    auto stl_modules = build_stl_modules(channel_st, str_st, series_st, var_st);

    auto mod = compile_arc(client, source);
    auto wasm_mod = ASSERT_NIL_P(
        wasm::Module::open({
            .program = mod,
            .modules = stl_modules,
            .strings = str_st,
        })
    );

    const auto *func_node = find_node_by_type(mod, "concat_len");
    ASSERT_NE(func_node, nullptr);

    state::State state(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{index_ch.key, x::telem::TIMESTAMP_T, 0},
                 {input_a_ch.key, x::telem::STRING_T, index_ch.key},
                 {input_b_ch.key, x::telem::STRING_T, index_ch.key},
                 {output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT64_T, output_idx.key}}
        },
        arc::runtime::errors::noop_handler
    );

    std::vector<std::string> on_keys;
    for (const auto &n: mod.nodes)
        if (n.type == "on") on_keys.push_back(n.key);
    ASSERT_GE(on_keys.size(), 2);

    auto on_a_state = ASSERT_NIL_P(state.node(on_keys[0]));
    auto on_a_data = x::telem::Series(std::vector<std::string>{"hello"});
    on_a_data.alignment = x::telem::Alignment(1, 0);
    on_a_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_a_data)
    );
    auto on_a_time = x::telem::Series(x::telem::TimeStamp(1 * x::telem::SECOND));
    on_a_time.alignment = x::telem::Alignment(1, 0);
    on_a_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_a_time)
    );

    auto on_b_state = ASSERT_NIL_P(state.node(on_keys[1]));
    auto on_b_data = x::telem::Series(std::vector<std::string>{" world"});
    on_b_data.alignment = x::telem::Alignment(1, 0);
    on_b_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_b_data)
    );
    auto on_b_time = x::telem::Series(x::telem::TimeStamp(1 * x::telem::SECOND));
    on_b_time.alignment = x::telem::Alignment(1, 0);
    on_b_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::move(on_b_time)
    );

    auto node_state = ASSERT_NIL_P(state.node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("concat_len"));
    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();
    std::vector<std::string> changed_outputs;
    ctx.mark_changed = [&](size_t i) {
        changed_outputs.push_back(func_node->outputs[i].name);
    };

    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(changed_outputs.size(), 1);

    auto result_state = ASSERT_NIL_P(state.node(func_node->key));
    const auto &output = result_state.output(0);
    ASSERT_EQ(output->size(), 1);
    EXPECT_EQ(output->at<int64_t>(0), 11);
}

/// @brief string.len() with literal via qualified syntax.
TEST(QualifiedCallTest, StringLenLiteral) {
    const auto client = new_test_client();
    const auto v = call_func<int64_t>(client, R"arc(
func str_len() i64 { return string.len("hello") })arc");
    EXPECT_EQ(v, 5);
}

/// @brief string.concat() with literals via qualified syntax.
TEST(QualifiedCallTest, StringConcatLiteral) {
    const auto client = new_test_client();
    const auto v = call_func<int64_t>(client, R"arc(
func concat_len() i64 { return string.len(string.concat("ab", "cd")) })arc");
    EXPECT_EQ(v, 4);
}

/// @brief string.equal() returns 1 for identical strings.
TEST(QualifiedCallTest, StringEqualTrue) {
    const auto client = new_test_client();
    const auto v = call_func<int32_t>(client, R"arc(
func str_eq() i32 { return string.equal("abc", "abc") })arc");
    EXPECT_EQ(v, 1);
}

/// @brief string.equal() returns 0 for different strings.
TEST(QualifiedCallTest, StringEqualFalse) {
    const auto client = new_test_client();
    const auto v = call_func<int32_t>(client, R"arc(
func str_neq() i32 { return string.equal("abc", "def") })arc");
    EXPECT_EQ(v, 0);
}

/// @brief math.pow(const, const) with i64 literals.
TEST(QualifiedCallTest, MathPowConstConstI64) {
    const auto client = new_test_client();
    const auto v = call_func<int64_t>(client, R"arc(
func pow_ii() i64 { return math.pow(2, 10) })arc");
    EXPECT_EQ(v, 1024);
}

/// @brief math.pow(const, const) with f64 literals.
TEST(QualifiedCallTest, MathPowConstConstF64) {
    const auto client = new_test_client();
    const auto v = call_func<double>(client, R"arc(
func pow_ff() f64 { return math.pow(2.0, 3.0) })arc");
    EXPECT_DOUBLE_EQ(v, 8.0);
}

/// @brief math.pow(chan, const) with f64 channel base.
TEST(QualifiedCallTest, MathPowChanConstF64) {
    const auto client = new_test_client();
    const auto v = call_func<double>(
        client,
        R"arc(func squared(x f64) f64 { return math.pow(x, 2) })arc",
        {3.0}
    );
    EXPECT_DOUBLE_EQ(v, 9.0);
}

/// @brief math.pow(chan, const) with i64 channel base.
TEST(QualifiedCallTest, MathPowChanConstI64) {
    const auto client = new_test_client();
    const auto v = call_func<int64_t>(
        client,
        R"arc(func cubed(x i64) i64 { return math.pow(x, 3) })arc",
        {static_cast<int64_t>(5)}
    );
    EXPECT_EQ(v, 125);
}

/// @brief math.pow(const, chan) with f64 channel exponent.
TEST(QualifiedCallTest, MathPowConstChanF64) {
    const auto client = new_test_client();
    const auto v = call_func<double>(
        client,
        R"arc(func base3(exp f64) f64 { return math.pow(3.0, exp) })arc",
        {2.0}
    );
    EXPECT_DOUBLE_EQ(v, 9.0);
}

/// @brief math.pow(const, chan) with i64 channel exponent.
TEST(QualifiedCallTest, MathPowConstChanI64) {
    const auto client = new_test_client();
    const auto v = call_func<int64_t>(
        client,
        R"arc(func base2(exp i64) i64 { return math.pow(2, exp) })arc",
        {static_cast<int64_t>(4)}
    );
    EXPECT_EQ(v, 16);
}

/// @brief math.pow(chan, chan) with f64 channels.
TEST(QualifiedCallTest, MathPowChanChanF64) {
    const auto client = new_test_client();
    const auto v = call_func<double>(
        client,
        R"arc(func pow_ff(base f64, exp f64) f64 { return math.pow(base, exp) })arc",
        {4.0, 0.5}
    );
    EXPECT_DOUBLE_EQ(v, 2.0);
}

/// @brief math.pow(chan, chan) with i64 channels.
TEST(QualifiedCallTest, MathPowChanChanI64) {
    const auto client = new_test_client();
    const auto v = call_func<int64_t>(
        client,
        R"arc(func pow_ii(base i64, exp i64) i64 { return math.pow(base, exp) })arc",
        {static_cast<int64_t>(2), static_cast<int64_t>(10)}
    );
    EXPECT_EQ(v, 1024);
}

/// @brief for i := range(5) sums 0..4.
TEST(ForLoopTest, Range1Arg) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range1() i64 { sum i64 := 0
 for i := range(5) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 10);
}

/// @brief for i := range(5, 10) sums 5..9.
TEST(ForLoopTest, Range2Arg) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range2() i64 { sum i64 := 0
 for i := range(5, 10) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 35);
}

/// @brief for i := range(0, 10, 2) sums even numbers.
TEST(ForLoopTest, Range3ArgWithStep) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range3() i64 { sum i64 := 0
 for i := range(0, 10, 2) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 20);
}

/// @brief range(0) produces zero iterations.
TEST(ForLoopTest, EmptyRange) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_empty() i64 { sum i64 := 99
 for i := range(0) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 99);
}

/// @brief range(10, 5) with no step produces zero iterations.
TEST(ForLoopTest, ReversedBounds) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_rev() i64 { sum i64 := 99
 for i := range(10, 5) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 99);
}

/// @brief range(0, 10, 3) steps by 3.
TEST(ForLoopTest, StepOf3) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_step3() i64 { sum i64 := 0
 for i := range(0, 10, 3) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 18);
}

/// @brief range(0, 5, 10) step exceeds range, zero iterations.
TEST(ForLoopTest, StepLargerThanRange) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_big_step() i64 { sum i64 := 0
 for i := range(0, 5, 10) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 0);
}

/// @brief range(10, 0, -2) descends by 2.
TEST(ForLoopTest, NegativeStep) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_neg_step() i64 { sum i64 := 0
 for i := range(10, 0, -2) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 30);
}

/// @brief range(10, 0, -1) descends by 1.
TEST(ForLoopTest, NegativeStepOf1) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_neg1() i64 { sum i64 := 0
 for i := range(10, 0, -1) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 55);
}

/// @brief range(0, 10, -1) negative step with ascending bounds, zero iterations.
TEST(ForLoopTest, NegativeStepEmpty) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_neg_empty() i64 { sum i64 := 99
 for i := range(0, 10, -1) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 99);
}

/// @brief range(-5, 5) crosses zero.
TEST(ForLoopTest, NegativeStartPositiveEnd) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_neg_start() i64 { sum i64 := 0
 for i := range(-5, 5) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, -5);
}

/// @brief range(-5, -1) both bounds negative.
TEST(ForLoopTest, BothBoundsNegative) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_neg_both() i64 { sum i64 := 0
 for i := range(-5, -1) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, -14);
}

/// @brief range(3, -3, -1) descends across zero.
TEST(ForLoopTest, PositiveToNegativeDescending) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_pos_to_neg() i64 { sum i64 := 0
 for i := range(3, -3, -1) { sum = sum + i }
 return sum })arc");
    EXPECT_EQ(v, 3);
}

/// @brief range with mixed i32 start and i64 end.
TEST(ForLoopTest, MixedI32I64Range) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_mixed() i64 {
    lo i32 := 1
    hi i64 := 5
    sum i64 := 0
    for i := range(lo, hi) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 10);
}

/// @brief range with mixed i32/i64 bounds and i32 step.
TEST(ForLoopTest, MixedI32I64RangeWithStep) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func range_3arg_mixed() i64 {
    lo i32 := 0
    hi i64 := 10
    s i32 := 3
    sum i64 := 0
    for i := range(lo, hi, s) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 18);
}

/// @brief for x := data sums series elements.
TEST(ForLoopTest, SeriesSumSingleIdent) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func series_sum() i32 {
    data series i32 := [1, 2, 3, 4, 5]
    sum i32 := 0
    for x := data { sum = sum + x }
    return sum
})arc");
    EXPECT_EQ(v, 15);
}

/// @brief for i, x := data computes weighted sum with index.
TEST(ForLoopTest, SeriesWeightedSumTwoIdent) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func series_weighted() i32 {
    data series i32 := [10, 20, 30]
    sum i32 := 0
    for i, x := data { sum = sum + x * (i + 1) }
    return sum
})arc");
    EXPECT_EQ(v, 140);
}

/// @brief iterating an empty series produces zero iterations.
TEST(ForLoopTest, EmptySeriesZeroIterations) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func series_empty() i32 {
    data series i32 := []
    sum i32 := 99
    for x := data { sum = sum + x }
    return sum
})arc");
    EXPECT_EQ(v, 99);
}

/// @brief break exits series iteration when threshold exceeded.
TEST(ForLoopTest, BreakOnThreshold) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func break_thresh() i32 {
    data series i32 := [1, 2, 3, 100, 5]
    sum i32 := 0
    for x := data { if x > 50 { break }
 sum = sum + x }
    return sum
})arc");
    EXPECT_EQ(v, 6);
}

/// @brief continue skips odd indices in range loop.
TEST(ForLoopTest, ContinueSkipOddIndices) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func cont_skip() i32 {
    sum i32 := 0
    for i := range(i32(6)) { if i % 2 != 0 { continue }
 sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 6);
}

/// @brief continue skips negative elements in series iteration.
TEST(ForLoopTest, ContinueSkipSeriesElements) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func cont_series() i32 {
    data series i32 := [10, -1, 20, -1, 30]
    sum i32 := 0
    for x := data { if x < 0 { continue }
 sum = sum + x }
    return sum
})arc");
    EXPECT_EQ(v, 60);
}

/// @brief continue in nested loop only affects inner loop.
TEST(ForLoopTest, ContinueInnerLoopOnly) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func cont_nested() i64 {
    sum i64 := 0
    for i := range(3) { for j := range(4) { if j == 2 { continue }
 sum = sum + 1 } }
    return sum
})arc");
    EXPECT_EQ(v, 9);
}

/// @brief break in nested loop only exits inner loop.
TEST(ForLoopTest, BreakInnerLoopOnly) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func break_inner_only() i64 {
    sum i64 := 0
    for i := range(3) {
        for j := range(10) { if j >= 2 { break }
 sum = sum + 1 }
        sum = sum + 100
    }
    return sum
})arc");
    EXPECT_EQ(v, 306);
}

/// @brief conditional for loop counts down while n > 0.
TEST(ForLoopTest, WhileStyleCountdown) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func while_count() i32 {
    n i32 := 5
    sum i32 := 0
    for n > 0 { sum = sum + n
 n = n - 1 }
    return sum
})arc");
    EXPECT_EQ(v, 15);
}

/// @brief infinite for loop exits via break.
TEST(ForLoopTest, InfiniteLoopWithBreak) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func inf_break() i32 {
    val i32 := 1
    for { val = val * 2
 if val > 100 { break } }
    return val
})arc");
    EXPECT_EQ(v, 128);
}

/// @brief nested range loops compute 3x4 matrix iteration count.
TEST(ForLoopTest, NestedMatrixCount) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func nested_count() i64 {
    count i64 := 0
    for i := range(3) { for j := range(4) { count = count + 1 } }
    return count
})arc");
    EXPECT_EQ(v, 12);
}

/// @brief inner break with outer loop running all iterations.
TEST(ForLoopTest, InnerBreakOuterRunsFully) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func inner_break() i64 {
    count i64 := 0
    for i := range(3) { for j := range(4) { if j > 1 { break }
 count = count + 1 } }
    return count
})arc");
    EXPECT_EQ(v, 6);
}

/// @brief 3-deep nested range loops compute 2x3x4 count.
TEST(ForLoopTest, ThreeDeepNested) {
    const auto c = new_test_client();
    const auto v = call_func<int64_t>(c, R"arc(
func nested_3deep() i64 {
    count i64 := 0
    for i := range(2) { for j := range(3) { for k := range(4) { count = count + 1 } } }
    return count
})arc");
    EXPECT_EQ(v, 24);
}

/// @brief range loop nested inside series iteration.
TEST(ForLoopTest, RangeNestedInsideSeriesIteration) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func series_range_nested() i32 {
    data series i32 := [10, 20, 30]
    sum i32 := 0
    for x := data { for j := range(i32(x / 10)) { sum = sum + 1 } }
    return sum
})arc");
    EXPECT_EQ(v, 6);
}

/// @brief range with mixed i16 start and i32 end.
TEST(ForLoopTest, MixedI16I32Range) {
    const auto c = new_test_client();
    const auto v = call_func<int32_t>(c, R"arc(
func range_i16_i32() i32 {
    lo i16 := 0
    hi i32 := 4
    sum i32 := 0
    for i := range(lo, hi) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 6);
}

/// @brief range with mixed u8 start and u32 end.
TEST(ForLoopTest, MixedU8U32Range) {
    const auto c = new_test_client();
    const auto v = call_func<uint32_t>(c, R"arc(
func range_u8_u32() u32 {
    lo u8 := 1
    hi u32 := 4
    sum u32 := 0
    for i := range(lo, hi) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 6);
}

/// @brief range with u8 bounds.
TEST(ForLoopTest, U8RangeBounds) {
    const auto c = new_test_client();
    const auto v = call_func<uint8_t>(c, R"arc(
func range_u8_only() u8 {
    sum u8 := 0
    for i := range(u8(5)) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 10);
}

/// @brief range with i8 bounds.
TEST(ForLoopTest, I8RangeBounds) {
    const auto c = new_test_client();
    const auto v = call_func<int8_t>(c, R"arc(
func range_i8_only() i8 {
    sum i8 := 0
    for i := range(i8(1), i8(5)) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 10);
}

/// @brief range with u16 bounds.
TEST(ForLoopTest, U16RangeBounds) {
    const auto c = new_test_client();
    const auto v = call_func<uint16_t>(c, R"arc(
func range_u16_only() u16 {
    sum u16 := 0
    for i := range(u16(10)) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 45);
}

/// @brief range with i16 bounds.
TEST(ForLoopTest, I16RangeBounds) {
    const auto c = new_test_client();
    const auto v = call_func<int16_t>(c, R"arc(
func range_i16_only() i16 {
    sum i16 := 0
    for i := range(i16(1), i16(6)) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 15);
}

/// @brief range with mixed i8 start and i16 end.
TEST(ForLoopTest, MixedI8I16Range) {
    const auto c = new_test_client();
    const auto v = call_func<int16_t>(c, R"arc(
func range_i8_i16() i16 {
    lo i8 := 0
    hi i16 := 5
    sum i16 := 0
    for i := range(lo, hi) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 10);
}

/// @brief range with mixed u8/u16 bounds and u8 step.
TEST(ForLoopTest, MixedU8U16RangeWithStep) {
    const auto c = new_test_client();
    const auto v = call_func<uint16_t>(c, R"arc(
func range_u8_u16_step() u16 {
    lo u8 := 0
    hi u16 := 10
    s u8 := 2
    sum u16 := 0
    for i := range(lo, hi, s) { sum = sum + i }
    return sum
})arc");
    EXPECT_EQ(v, 20);
}

/// @brief stateful variable ($=) in a for loop accumulates across reactive executions.
TEST(ForLoopTest, DISABLED_StatefulAccumulationAcrossCalls) {
    const auto client = new_test_client();

    auto idx_name = random_name("time");
    auto trigger_name = random_name("trigger");
    auto output_name = random_name("output");

    auto index_ch = synnax::channel::Channel{
        .name = idx_name,
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(index_ch));
    auto trigger_ch = synnax::channel::Channel{
        .name = trigger_name,
        .data_type = x::telem::INT64_T,
        .index = index_ch.key,
    };
    ASSERT_NIL(client.channels.create(trigger_ch));
    auto output_idx = synnax::channel::Channel{
        .name = random_name("out_idx"),
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
    };
    ASSERT_NIL(client.channels.create(output_idx));
    auto output_ch = synnax::channel::Channel{
        .name = output_name,
        .data_type = x::telem::INT64_T,
        .index = output_idx.key,
    };
    ASSERT_NIL(client.channels.create(output_ch));

    const std::string source = R"(
func loop_state(trigger i64) i64 {
    total i64 $= 0
    for i := range(3) {
        total = total + 1
    }
    return total
}
)" + trigger_name + " -> loop_state{} -> " +
                               output_name;

    auto channel_st = std::make_shared<stl::channel::State>(
        std::vector<state::ChannelDigest>{
            {index_ch.key, x::telem::TIMESTAMP_T, 0},
            {trigger_ch.key, x::telem::INT64_T, index_ch.key},
            {output_idx.key, x::telem::TIMESTAMP_T, 0},
            {output_ch.key, x::telem::INT64_T, output_idx.key}
        }
    );
    auto str_st = std::make_shared<stl::str::State>();
    auto series_st = std::make_shared<stl::series::State>();
    auto var_st = std::make_shared<stl::stateful::Variables>();
    auto stl_modules = build_stl_modules(channel_st, str_st, series_st, var_st);

    auto mod = compile_arc(client, source);
    auto state = std::make_shared<state::State>(
        state::Config{
            .ir = (static_cast<arc::ir::IR>(mod)),
            .channels =
                {{index_ch.key, x::telem::TIMESTAMP_T, 0},
                 {trigger_ch.key, x::telem::INT64_T, index_ch.key},
                 {output_idx.key, x::telem::TIMESTAMP_T, 0},
                 {output_ch.key, x::telem::INT64_T, output_idx.key}}
        },
        channel_st,
        str_st,
        series_st,
        var_st,
        arc::runtime::errors::noop_handler
    );

    auto wasm_mod = ASSERT_NIL_P(
        wasm::Module::open({
            .program = mod,
            .modules = stl_modules,
            .strings = str_st,
        })
    );

    const auto *func_node = find_node_by_type(mod, "loop_state");
    ASSERT_NE(func_node, nullptr);

    for (const auto &n: mod.nodes) {
        if (n.type != "on") continue;
        auto on_state = ASSERT_NIL_P(state->node(n.key));
        auto on_data = x::telem::Series(std::vector<int64_t>{1});
        on_data.alignment = x::telem::Alignment(1, 0);
        on_state.output(0) = x::mem::make_local_shared<x::telem::Series>(
            std::move(on_data)
        );
        auto on_time = x::telem::Series(x::telem::TimeStamp(1 * x::telem::MICROSECOND));
        on_time.alignment = x::telem::Alignment(1, 0);
        on_state.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
            std::move(on_time)
        );
    }

    auto node_state = ASSERT_NIL_P(state->node(func_node->key));
    auto func = ASSERT_NIL_P(wasm_mod->func("loop_state"));
    wasm::Node node(mod, *func_node, std::move(node_state), func, wasm_mod->strings());

    auto ctx = make_context();

    auto set_trigger = [&](uint64_t alignment) {
        for (const auto &n: mod.nodes) {
            if (n.type != "on") continue;
            auto on_st = ASSERT_NIL_P(state->node(n.key));
            auto d = x::telem::Series(std::vector<int64_t>{1});
            d.alignment = x::telem::Alignment(alignment, 0);
            on_st.output(0) = x::mem::make_local_shared<x::telem::Series>(std::move(d));
            auto t = x::telem::Series(x::telem::TimeStamp(1 * x::telem::MICROSECOND));
            t.alignment = x::telem::Alignment(alignment, 0);
            on_st.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
                std::move(t)
            );
        }
    };

    set_trigger(1);
    ASSERT_NIL(node.next(ctx));
    auto r1 = ASSERT_NIL_P(state->node(func_node->key));
    ASSERT_EQ(r1.output(0)->size(), 1);
    EXPECT_EQ(r1.output(0)->at<int64_t>(0), 3);

    node.reset();
    set_trigger(2);
    ASSERT_NIL(node.next(ctx));
    auto r2 = ASSERT_NIL_P(state->node(func_node->key));
    ASSERT_EQ(r2.output(0)->size(), 1);
    EXPECT_EQ(r2.output(0)->at<int64_t>(0), 6);

    node.reset();
    set_trigger(3);
    ASSERT_NIL(node.next(ctx));
    auto r3 = ASSERT_NIL_P(state->node(func_node->key));
    ASSERT_EQ(r3.output(0)->size(), 1);
    EXPECT_EQ(r3.output(0)->at<int64_t>(0), 9);
}

/// @brief Module::func returns VALIDATION error for unresolved output types.
TEST(ModuleFuncTest, ReturnsErrorForUnresolvedOutputType) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2.0
}
)" + ch.name + " -> double{}";

    auto mod = compile_arc(client, source);

    // Corrupt the output type to simulate an unresolved type
    for (auto &fn: mod.functions)
        if (fn.key == "double" && !fn.outputs.empty())
            fn.outputs[0].type = arc::types::Type{};

    auto wasm_mod = ASSERT_NIL_P(wasm::Module::open({.program = mod}));
    auto [func, err] = wasm_mod->func("double");
    ASSERT_TRUE(err.matches(x::errors::VALIDATION));
    EXPECT_NE(err.message().find("unresolved type"), std::string::npos);
}
}
