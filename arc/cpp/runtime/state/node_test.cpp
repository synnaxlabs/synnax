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

#include "arc/cpp/runtime/state/node.h"

// Test channel I/O (external Synnax channels)
TEST(NodeStateTest, ReadChannel) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Register external channel
    state.register_channel(1, telem::FLOAT32_T);

    // Create node state (no edges, just for channel access)
    arc::state::Node node_state(&state, "test", {}, {});

    // Add data via input queue
    auto data = std::make_shared<telem::Series>(std::vector<float>{42.0f});
    auto time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{1000000000}}
    );
    input_queue.push({1, data, time});
    state.process_input_queue();

    // Read via NodeState
    auto [value, err] = node_state.read_channel(1);
    ASSERT_NIL(err);
    EXPECT_EQ(std::get<float>(value), 42.0f);
}

// Test channel write (external Synnax channels)
TEST(NodeStateTest, WriteChannel) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    state.register_channel(3, telem::FLOAT64_T);

    arc::state::Node node_state(&state, "test", {}, {});

    // Write via NodeState
    auto err = node_state.write_channel<double>(3, 3.14);
    ASSERT_NIL(err);

    // Verify output queue
    arc::ChannelOutput output;
    EXPECT_TRUE(output_queue.pop(output));
    EXPECT_EQ(output.channel_id, 3);
    EXPECT_EQ(std::get<double>(output.value), 3.14);
}

// Test state variables
TEST(NodeStateTest, StateVariables) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    arc::state::Node node_state(&state, "counter", {}, {});

    // Load state (should initialize)
    int32_t count = node_state.load_state_var<int32_t>(0, 0);
    EXPECT_EQ(count, 0);

    // Store new value
    node_state.store_state_var<int32_t>(0, 10);

    // Load again (should get stored value)
    count = node_state.load_state_var<int32_t>(0, 0);
    EXPECT_EQ(count, 10);
}

// Test state isolation between nodes
TEST(NodeStateTest, StateIsolationBetweenNodes) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    arc::state::Node node1_state(&state, "node1", {}, {});
    arc::state::Node node2_state(&state, "node2", {}, {});

    // Store same var_id in different nodes
    node1_state.store_state_var<int32_t>(0, 100);
    node2_state.store_state_var<int32_t>(0, 200);

    // Should be isolated (different hash of node_id)
    EXPECT_EQ(node1_state.load_state_var<int32_t>(0, 0), 100);
    EXPECT_EQ(node2_state.load_state_var<int32_t>(0, 0), 200);
}

// Test parameter-indexed output
TEST(NodeStateTest, ParameterIndexedOutput) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Register node with output parameter
    arc::NodeMetadata meta{"add"};
    meta.output_params = {"result"};
    state.register_node(meta);

    // Create NodeState with output handle
    std::vector<arc::Handle> outputs = {arc::Handle{"add", "result"}};
    arc::state::Node node_state(&state, "add", {}, outputs);

    // Get output pointer (parameter index 0)
    telem::Series *out = node_state.output(0);
    ASSERT_NE(out, nullptr);

    // Write data to output (reconstruct via shared_ptr since Series isn't copyable)
    arc::Handle handle{"add", "result"};
    auto &vp = state.get_output(handle);
    vp.data = std::make_shared<telem::Series>(std::vector<float>{1.0f, 2.0f, 3.0f});

    // Verify it's stored
    const auto &vp_read = state.get_output(handle);
    EXPECT_EQ(vp_read.data->size(), 3);
    EXPECT_EQ(vp_read.data->at<float>(0), 1.0f);
}

// Test refresh_inputs with single input
TEST(NodeStateTest, RefreshInputsWithSingleInput) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Setup: Node B reads from Node A
    arc::NodeMetadata meta_a{"A"};
    meta_a.output_params = {"out"};
    state.register_node(meta_a);

    arc::NodeMetadata meta_b{"B"};
    meta_b.input_params = {"in"};
    state.register_node(meta_b);

    // Add edge A.out → B.in
    state.add_edge(arc::Edge{arc::Handle{"A", "out"}, arc::Handle{"B", "in"}});

    // Create NodeState for B
    auto edges = state.incoming_edges("B");
    arc::state::Node node_state(&state, "B", edges, {});

    // Initially no data
    EXPECT_FALSE(node_state.refresh_inputs());

    // Node A produces output
    auto &output_a = state.get_output(arc::Handle{"A", "out"});
    output_a.data = std::make_shared<telem::Series>(std::vector<float>{1.0f, 2.0f});
    output_a.time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{100}, telem::TimeStamp{200}}
    );

    // Now refresh should succeed
    EXPECT_TRUE(node_state.refresh_inputs());

    // Check aligned input
    const auto &input = node_state.input(0);
    EXPECT_EQ(input.size(), 2);
    EXPECT_EQ(input.at<float>(0), 1.0f);
    EXPECT_EQ(input.at<float>(1), 2.0f);

    const auto &input_time = node_state.input_time(0);
    EXPECT_EQ(input_time.size(), 2);
}

// Test refresh_inputs with multiple inputs
TEST(NodeStateTest, RefreshInputsWithMultipleInputs) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Setup: Node C reads from A and B
    arc::NodeMetadata meta_a{"A"};
    meta_a.output_params = {"out"};
    state.register_node(meta_a);

    arc::NodeMetadata meta_b{"B"};
    meta_b.output_params = {"out"};
    state.register_node(meta_b);

    arc::NodeMetadata meta_c{"C"};
    meta_c.input_params = {"in1", "in2"};
    state.register_node(meta_c);

    // Add edges
    state.add_edge(arc::Edge{arc::Handle{"A", "out"}, arc::Handle{"C", "in1"}});
    state.add_edge(arc::Edge{arc::Handle{"B", "out"}, arc::Handle{"C", "in2"}});

    // Create NodeState for C
    auto edges = state.incoming_edges("C");
    arc::state::Node node_state(&state, "C", edges, {});

    // A produces data at t=100
    auto &output_a = state.get_output(arc::Handle{"A", "out"});
    output_a.data = std::make_shared<telem::Series>(std::vector<float>{1.0f});
    output_a.time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{100}}
    );

    // B produces data at t=200
    auto &output_b = state.get_output(arc::Handle{"B", "out"});
    output_b.data = std::make_shared<telem::Series>(std::vector<float>{2.0f});
    output_b.time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{200}}
    );

    // Refresh: trigger is A (earliest timestamp t=100)
    EXPECT_TRUE(node_state.refresh_inputs());

    // Note: Edge order matters for input indexing
    // Need to verify which input is which based on edge order
    EXPECT_EQ(node_state.num_inputs(), 2);
}

// Test watermark tracking prevents reprocessing
TEST(NodeStateTest, WatermarkPreventsReprocessing) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Setup: B reads from A
    arc::NodeMetadata meta_a{"A"};
    meta_a.output_params = {"out"};
    state.register_node(meta_a);

    arc::NodeMetadata meta_b{"B"};
    meta_b.input_params = {"in"};
    state.register_node(meta_b);

    state.add_edge(arc::Edge{arc::Handle{"A", "out"}, arc::Handle{"B", "in"}});

    auto edges = state.incoming_edges("B");
    arc::state::Node node_state(&state, "B", edges, {});

    // A produces first output at t=100
    auto &output_a = state.get_output(arc::Handle{"A", "out"});
    output_a.data = std::make_shared<telem::Series>(std::vector<float>{1.0f});
    output_a.time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{100}}
    );

    // First refresh succeeds
    EXPECT_TRUE(node_state.refresh_inputs());

    // Second refresh without new data should fail
    EXPECT_FALSE(node_state.refresh_inputs());

    // A produces new output at t=200
    output_a.data = std::make_shared<telem::Series>(std::vector<float>{2.0f});
    output_a.time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{200}}
    );

    // Third refresh should succeed (new data beyond watermark)
    EXPECT_TRUE(node_state.refresh_inputs());
}
