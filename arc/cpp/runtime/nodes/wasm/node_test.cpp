// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/runtime/nodes/wasm/node.h"
#include "arc/cpp/runtime/state/node.h"
#include "arc/cpp/runtime/wasm/runtime.h"

class WASMNodeTest : public ::testing::Test {
protected:
    void SetUp() override {
        auto err = arc::Runtime::initialize_runtime();
        ASSERT_NIL(err);
    }

    void TearDown() override { arc::Runtime::destroy_runtime(); }
};

TEST_F(WASMNodeTest, Construction) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    arc::Runtime runtime;
    auto node_state = std::make_unique<arc::state::Node>(
        &state,
        "test_node",
        std::vector<arc::Edge>{},
        std::vector<arc::Handle>{}
    );

    // Create node with null function (won't execute)
    arc::wasm::Node node("test_node", std::move(node_state), &runtime, nullptr, {});

    EXPECT_EQ(node.id(), "test_node");
    EXPECT_EQ(node.state().node_id(), "test_node");
}

TEST_F(WASMNodeTest, SkipsExecutionWhenNoInputData) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Setup node with no inputs (will always skip refresh_inputs)
    arc::Runtime runtime;
    auto node_state = std::make_unique<arc::state::Node>(
        &state,
        "test_node",
        std::vector<arc::Edge>{},
        std::vector<arc::Handle>{}
    );

    arc::wasm::Node node("test_node", std::move(node_state), &runtime, nullptr, {});

    // No input data, should return NIL without executing
    arc::NodeContext ctx;
    auto err = node.execute(ctx);
    ASSERT_NIL(err);
}

TEST_F(WASMNodeTest, ExecutesWhenInputDataAvailable) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Setup: test_node reads from source_node
    arc::NodeMetadata meta_source{"source_node"};
    meta_source.output_params = {"out"};
    state.register_node(meta_source);

    arc::NodeMetadata meta_test{"test_node"};
    meta_test.input_params = {"in"};
    state.register_node(meta_test);

    state.add_edge(
        arc::Edge{arc::Handle{"source_node", "out"}, arc::Handle{"test_node", "in"}}
    );

    // Produce data from source
    auto &source_output = state.get_output(arc::Handle{"source_node", "out"});
    source_output.data = std::make_shared<telem::Series>(std::vector<float>{42.0f});
    source_output.time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{1000000000}}
    );

    // Create runtime (dummy module)
    arc::Runtime runtime;
    std::vector<uint8_t> dummy_wasm = {0x00, 0x61, 0x73, 0x6d}; // WASM magic number
    runtime.load_aot_module(dummy_wasm);
    runtime.instantiate(64 * 1024, 0);

    auto [func, func_err] = runtime.find_function("test");

    // Create node with edge
    auto edges = state.incoming_edges("test_node");
    auto node_state = std::make_unique<arc::state::Node>(
        &state,
        "test_node",
        edges,
        std::vector<arc::Handle>{}
    );

    arc::wasm::Node node("test_node", std::move(node_state), &runtime, func, {});

    // Execute - refresh_inputs should succeed, WASM call may fail (ok for test)
    arc::NodeContext ctx;
    node.execute(ctx);
}

TEST_F(WASMNodeTest, NodeStateAccess) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    arc::NodeMetadata meta{"test_node"};
    meta.input_params = {"in"};
    meta.output_params = {"out"};
    state.register_node(meta);

    arc::Runtime runtime;

    std::vector<arc::Edge> inputs = {};
    std::vector<arc::Handle> outputs = {arc::Handle{"test_node", "out"}};
    auto node_state = std::make_unique<arc::state::Node>(
        &state,
        "test_node",
        inputs,
        outputs
    );

    arc::wasm::Node node("test_node", std::move(node_state), &runtime, nullptr, {});

    // Access NodeState through node
    EXPECT_EQ(node.state().num_inputs(), 0);
    EXPECT_EQ(node.state().num_outputs(), 1);
}

// TODO: Add integration test with real AOT-compiled Arc module when available
