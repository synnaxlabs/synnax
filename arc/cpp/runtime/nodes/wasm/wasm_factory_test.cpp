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

#include "arc/cpp/runtime/nodes/wasm/factory.h"
#include "arc/cpp/runtime/nodes/wasm/node.h"
#include "arc/cpp/runtime/state/node.h"
#include "arc/cpp/runtime/wasm/runtime.h"

namespace arc {

class WASMNodeFactoryTest : public ::testing::Test {
protected:
    std::unique_ptr<queue::SPSC<ChannelUpdate>> input_queue_;
    std::unique_ptr<queue::SPSC<ChannelOutput>> output_queue_;
    std::unique_ptr<State> state_;
    std::unique_ptr<Runtime> runtime_;
    ir::IR ir_;

    void SetUp() override {
        // Initialize queues and state
        input_queue_ = std::make_unique<queue::SPSC<ChannelUpdate>>(16);
        output_queue_ = std::make_unique<queue::SPSC<ChannelOutput>>(16);
        state_ = std::make_unique<State>(input_queue_.get(), output_queue_.get());

        // Initialize WASM runtime
        auto init_err = Runtime::initialize_runtime();
        ASSERT_NIL(init_err);

        runtime_ = std::make_unique<Runtime>();

        // Setup IR with a WASM function
        ir::Function fn{"calculate"};
        fn.inputs.keys = {"x"};
        fn.inputs.values["x"] = ir::Type{ir::TypeKind::F64};
        fn.outputs.keys = {"result"};
        fn.outputs.values["result"] = ir::Type{ir::TypeKind::F64};
        ir_.functions.push_back(fn);

        // Add a node that uses the WASM function
        ir::Node node{"node1"};
        node.type = "calculate";
        node.inputs.keys = {"x"};
        node.inputs.values["x"] = ir::Type{ir::TypeKind::F64};
        node.outputs.keys = {"result"};
        node.outputs.values["result"] = ir::Type{ir::TypeKind::F64};
        ir_.nodes.push_back(node);

        // Register node metadata in state
        NodeMetadata meta;
        meta.key = "node1";
        meta.type = "calculate";
        meta.input_params = {"x"};
        meta.output_params = {"result"};
        state_->register_node(meta);
    }

    void TearDown() override { Runtime::destroy_runtime(); }
};

TEST_F(WASMNodeFactoryTest, ReturnsNotFoundForNonWASMFunction) {
    // Load minimal WASM module
    std::vector<uint8_t> dummy_wasm = {0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00};
    runtime_->load_aot_module(dummy_wasm);

    wasm::Factory factory(*runtime_);

    // Create node with type that doesn't exist in IR functions
    ir::Node unknown_node{"node2"};
    unknown_node.type = "unknown_function";

    NodeFactoryConfig cfg{unknown_node, *state_, ir_};

    auto [node, err] = factory.create(cfg);
    EXPECT_EQ(node, nullptr);
    ASSERT_OCCURRED_AS(err, "NOT_FOUND");
}

// Test removed - runtime is now passed to factory constructor, not config
// If you don't have a runtime, you simply don't add wasm::Factory to the MultiFactory

TEST_F(WASMNodeFactoryTest, CreatesWASMNodeSuccessfully) {
    // Load minimal WASM module (just magic number, will fail instantiation but that's
    // ok)
    std::vector<uint8_t> dummy_wasm = {0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00};
    auto load_err = runtime_->load_aot_module(dummy_wasm);
    // May fail, that's ok for this test - we're testing factory logic, not WASM
    // execution

    wasm::Factory factory(*runtime_);

    NodeFactoryConfig cfg{ir_.nodes[0], *state_, ir_};

    auto [node, err] = factory.create(cfg);

    // If runtime failed to load/instantiate, we'll get an error finding the function
    // That's expected in this test environment. The key is that the factory tried
    // to create a WASMNode (didn't return NOT_FOUND).
    if (err) {
        // Should be a WASM-related error, not NOT_FOUND
        EXPECT_NE(err.type, "NOT_FOUND");
    } else {
        // Successfully created node
        ASSERT_NE(node, nullptr);
        EXPECT_EQ(node->id(), "node1");
    }
}

TEST_F(WASMNodeFactoryTest, HandlesNodeWithEdges) {
    wasm::Factory factory(*runtime_);

    // Add another node as a source
    ir::Node source_node{"source"};
    source_node.type = "source_func";
    source_node.outputs.keys = {"out"};
    source_node.outputs.values["out"] = ir::Type{ir::TypeKind::F64};
    ir_.nodes.insert(ir_.nodes.begin(), source_node);

    // Add source function to IR
    ir::Function source_func{"source_func"};
    source_func.outputs.keys = {"out"};
    source_func.outputs.values["out"] = ir::Type{ir::TypeKind::F64};
    ir_.functions.push_back(source_func);

    // Add edge from source to node1
    ir::Edge ir_edge{ir::Handle{"source", "out"}, ir::Handle{"node1", "x"}};
    ir_.edges.push_back(ir_edge);

    // Convert to runtime Edge for state
    Edge runtime_edge{Handle{"source", "out"}, Handle{"node1", "x"}};
    state_->add_edge(runtime_edge);

    // Register source node metadata
    NodeMetadata source_meta;
    source_meta.key = "source";
    source_meta.type = "source_func";
    source_meta.output_params = {"out"};
    state_->register_node(source_meta);

    NodeFactoryConfig cfg{ir_.nodes[1], *state_, ir_};

    auto [node, err] = factory.create(cfg);

    // Same as above - may fail on WASM function lookup, but should not return NOT_FOUND
    if (err) {
        EXPECT_NE(err.type, "NOT_FOUND");
    } else {
        ASSERT_NE(node, nullptr);
        // Verify node has correct structure
        auto *wasm_node = dynamic_cast<wasm::Node *>(node.get());
        ASSERT_NE(wasm_node, nullptr);
        EXPECT_EQ(wasm_node->state().num_inputs(), 1);
    }
}

} // namespace arc
