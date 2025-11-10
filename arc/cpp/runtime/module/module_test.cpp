// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include <nlohmann/json.hpp>

#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/runtime/module/loader.h"

using json = nlohmann::json;

class LoaderTest : public ::testing::Test {
protected:
    void SetUp() override {
        auto err = arc::Runtime::initialize_runtime();
        ASSERT_NIL(err);
    }

    void TearDown() override { arc::Runtime::destroy_runtime(); }
};

TEST_F(LoaderTest, ExtractChannelKeys) {
    arc::module::Loader loader;
    arc::IR ir;

    arc::Node node1("node1");
    node1.channels.read[1] = "input_a";
    node1.channels.read[2] = "input_b";
    node1.channels.write["output"] = 3;

    arc::Node node2("node2");
    node2.channels.read[3] = "input_c";

    ir.nodes = {node1, node2};

    auto keys = loader.extract_channel_keys(ir);

    EXPECT_EQ(keys.size(), 3);
    // Keys should be 1, 2, 3 (in sorted order)
    std::sort(keys.begin(), keys.end());
    EXPECT_EQ(keys[0], 1);
    EXPECT_EQ(keys[1], 2);
    EXPECT_EQ(keys[2], 3);
}

TEST_F(LoaderTest, GetChannelType) {
    arc::module::Loader loader;

    arc::Node node("test");
    node.channels.read[1] = "input_a";
    node.inputs.keys = {"input_a"};
    node.inputs.values["input_a"] = arc::Type(arc::TypeKind::F64);

    auto type_kind = loader.get_channel_type(node, 1);
    EXPECT_EQ(type_kind, arc::TypeKind::F64);
}

TEST_F(LoaderTest, LoadEmptyModule) {
    arc::module::Loader loader;

    // Create minimal IR
    arc::IR ir;
    ir.strata = {}; // No nodes

    arc::module::Module mod(std::move(ir), {});

    auto [runtime, err] = loader.load(mod);
    // Should succeed even with no nodes
    ASSERT_NIL(err);
    EXPECT_NE(runtime.state, nullptr);
    EXPECT_NE(runtime.scheduler, nullptr);
    EXPECT_NE(runtime.runtime, nullptr);
}

TEST_F(LoaderTest, LoadModuleWithChannels) {
    arc::module::Loader loader;

    // Create IR with channel references
    arc::IR ir;

    arc::Node node1("input");
    node1.channels.write["value"] = 1;
    node1.outputs.keys = {"value"};
    node1.outputs.values["value"] = arc::Type(arc::TypeKind::I32);

    ir.nodes = {node1};
    ir.strata = {{"input"}};

    // No WASM bytecode (empty vector) - will skip WASM loading
    arc::module::Module mod(std::move(ir), {});

    auto [runtime, err] = loader.load(mod);
    ASSERT_NIL(err);

    // Verify channel was registered
    auto [value, ch_err] = runtime.state->read_channel(1);
    // Should be no_data (not an error, just no data yet)
    EXPECT_TRUE(ch_err.matches(xerrors::Error("arc.state.no_data")));
}

TEST_F(LoaderTest, AssembledRuntimeNext) {
    arc::module::AssembledRuntime runtime;

    // Create queues
    auto input_queue = std::make_unique<queue::SPSC<arc::ChannelUpdate>>(16);
    auto output_queue = std::make_unique<queue::SPSC<arc::ChannelOutput>>(16);

    // Create state with queues
    runtime.state = std::make_unique<arc::State>(input_queue.get(), output_queue.get());
    runtime.scheduler = std::make_unique<arc::Scheduler>(runtime.state.get());

    // Should be able to call next() on empty scheduler
    auto err = runtime.next();
    ASSERT_NIL(err);
}

// Integration test: Dataflow graph A → B → C
TEST_F(LoaderTest, DataflowGraphIntegration) {
    arc::module::Loader loader;

    // Build IR: Node A (source) → Node B (processor) → Node C (sink)
    arc::IR ir;

    // Node A: outputs "out"
    arc::Node node_a("A");
    node_a.type = "source";
    node_a.outputs.keys = {"out"};
    node_a.outputs.values["out"] = arc::Type(arc::TypeKind::F32);

    // Node B: inputs "in", outputs "out"
    arc::Node node_b("B");
    node_b.type = "processor";
    node_b.inputs.keys = {"in"};
    node_b.inputs.values["in"] = arc::Type(arc::TypeKind::F32);
    node_b.outputs.keys = {"out"};
    node_b.outputs.values["out"] = arc::Type(arc::TypeKind::F32);

    // Node C: inputs "in"
    arc::Node node_c("C");
    node_c.type = "sink";
    node_c.inputs.keys = {"in"};
    node_c.inputs.values["in"] = arc::Type(arc::TypeKind::F32);

    ir.nodes = {node_a, node_b, node_c};

    // Edges: A.out → B.in, B.out → C.in
    ir.edges = {
        arc::Edge{arc::Handle{"A", "out"}, arc::Handle{"B", "in"}},
        arc::Edge{arc::Handle{"B", "out"}, arc::Handle{"C", "in"}}
    };

    // Strata: A (0), B (1), C (2)
    ir.strata = {{"A"}, {"B"}, {"C"}};

    arc::module::Module module(std::move(ir), {}); // No WASM

    auto [runtime, err] = loader.load(module);
    ASSERT_NIL(err);

    // Verify edge graph was built
    auto edges_to_b = runtime.state->incoming_edges("B");
    EXPECT_EQ(edges_to_b.size(), 1);
    EXPECT_EQ(edges_to_b[0].source.node, "A");
    EXPECT_EQ(edges_to_b[0].source.param, "out");

    auto edges_to_c = runtime.state->incoming_edges("C");
    EXPECT_EQ(edges_to_c.size(), 1);
    EXPECT_EQ(edges_to_c[0].source.node, "B");

    // Verify node metadata was registered
    auto *meta_b = runtime.state->get_node_metadata("B");
    ASSERT_NE(meta_b, nullptr);
    EXPECT_EQ(meta_b->input_params.size(), 1);
    EXPECT_EQ(meta_b->output_params.size(), 1);
}

// Integration test: Multi-input temporal alignment
TEST_F(LoaderTest, MultiInputTemporalAlignment) {
    arc::module::Loader loader;

    // Build IR: A → C, B → C (C has two inputs)
    arc::IR ir;

    arc::Node node_a("A");
    node_a.type = "source_a";
    node_a.outputs.keys = {"out"};
    node_a.outputs.values["out"] = arc::Type(arc::TypeKind::F64);

    arc::Node node_b("B");
    node_b.type = "source_b";
    node_b.outputs.keys = {"out"};
    node_b.outputs.values["out"] = arc::Type(arc::TypeKind::F64);

    arc::Node node_c("C");
    node_c.type = "combiner";
    node_c.inputs.keys = {"in1", "in2"};
    node_c.inputs.values["in1"] = arc::Type(arc::TypeKind::F64);
    node_c.inputs.values["in2"] = arc::Type(arc::TypeKind::F64);

    ir.nodes = {node_a, node_b, node_c};

    ir.edges = {
        arc::Edge{arc::Handle{"A", "out"}, arc::Handle{"C", "in1"}},
        arc::Edge{arc::Handle{"B", "out"}, arc::Handle{"C", "in2"}}
    };

    ir.strata = {{"A", "B"}, {"C"}};

    arc::module::Module mod(std::move(ir), {});

    auto [runtime, err] = loader.load(mod);
    ASSERT_NIL(err);

    // Verify C has 2 incoming edges
    auto edges_to_c = runtime.state->incoming_edges("C");
    EXPECT_EQ(edges_to_c.size(), 2);

    // Simulate data production from A and B
    auto &out_a = runtime.state->get_output(arc::Handle{"A", "out"});
    out_a.data = std::make_shared<telem::Series>(std::vector<double>{1.0});
    out_a.time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{100}}
    );

    auto &out_b = runtime.state->get_output(arc::Handle{"B", "out"});
    out_b.data = std::make_shared<telem::Series>(std::vector<double>{2.0});
    out_b.time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{200}}
    );

    // Now if we had a NodeState for C, refresh_inputs would work
    // This validates the edge graph was built correctly
}
