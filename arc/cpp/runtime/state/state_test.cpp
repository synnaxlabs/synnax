// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/state/state.h"

#include "gtest/gtest.h"
#include "x/cpp/xtest/xtest.h"

TEST(ChannelBufferTest, UpdateAndRead) {
    arc::ChannelBuffer buffer(telem::FLOAT32_T);

    // Initially empty
    EXPECT_FALSE(buffer.has_data());

    // Create and update with shared_ptr
    auto data = std::make_shared<telem::Series>(
        std::vector<float>{1.0f, 2.0f, 3.0f}
    );
    auto time = std::make_shared<telem::Series>(
        telem::Series::linspace(
            telem::TimeStamp(0),
            telem::TimeStamp(3000000000),
            3
        )
    );

    buffer.update(data, time);

    // Should have data now
    EXPECT_TRUE(buffer.has_data());

    // Latest value should be last element
    auto value = buffer.latest_value();
    EXPECT_EQ(std::get<float>(value), 3.0f);
}

TEST(ChannelBufferTest, SharedPtrOwnership) {
    arc::ChannelBuffer buffer(telem::INT32_T);

    auto data = std::make_shared<telem::Series>(
        std::vector<int32_t>{10, 20, 30}
    );
    auto time = std::make_shared<telem::Series>(
        telem::Series::linspace(
            telem::TimeStamp(0),
            telem::TimeStamp(3000000000),
            3
        )
    );

    // Refcount should be 1
    EXPECT_EQ(data.use_count(), 1);
    EXPECT_EQ(time.use_count(), 1);

    buffer.update(data, time);

    // Refcount should be 2 (original + buffer)
    EXPECT_EQ(data.use_count(), 2);
    EXPECT_EQ(time.use_count(), 2);

    // Get series back
    auto data_ref = buffer.data();
    EXPECT_EQ(data_ref.use_count(), 3);  // original + buffer + data_ref
}

TEST(StateTest, RegisterChannel) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    state.register_channel(1, telem::FLOAT32_T);
    state.register_channel(2, telem::INT64_T);

    // Reading uninitialized channel should return no_data error
    auto [value, err] = state.read_channel(1);
    EXPECT_TRUE(err.matches(xerrors::Error("arc.state.no_data")));
}

TEST(StateTest, ProcessInputQueue) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    state.register_channel(1, telem::FLOAT32_T);

    // Simulate I/O thread pushing data
    auto data = std::make_shared<telem::Series>(
        std::vector<float>{1.0f, 2.0f, 3.0f}
    );
    auto time = std::make_shared<telem::Series>(
        telem::Series::linspace(
            telem::TimeStamp(0),
            telem::TimeStamp(3000000000),
            3
        )
    );

    arc::ChannelUpdate update{1, data, time};
    EXPECT_TRUE(input_queue.push(std::move(update)));

    // Process queue (RT thread)
    state.process_input_queue();

    // Read should now succeed
    auto [value, err] = state.read_channel(1);
    ASSERT_NIL(err);
    EXPECT_EQ(std::get<float>(value), 3.0f);  // Latest value
}

TEST(StateTest, MultipleChannelUpdates) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    state.register_channel(1, telem::FLOAT32_T);
    state.register_channel(2, telem::INT32_T);

    // Update channel 1
    {
        auto data = std::make_shared<telem::Series>(
            std::vector<float>{1.5f}
        );
        auto time = std::make_shared<telem::Series>(
            telem::Series::linspace(
                telem::TimeStamp(0),
                telem::TimeStamp(1000000000),
                1
            )
        );
        input_queue.push({1, data, time});
    }

    // Update channel 2
    {
        auto data = std::make_shared<telem::Series>(
            std::vector<int32_t>{42}
        );
        auto time = std::make_shared<telem::Series>(
            telem::Series::linspace(
                telem::TimeStamp(0),
                telem::TimeStamp(1000000000),
                1
            )
        );
        input_queue.push({2, data, time});
    }

    state.process_input_queue();

    // Verify both channels
    {
        auto [value, err] = state.read_channel(1);
        ASSERT_NIL(err);
        EXPECT_EQ(std::get<float>(value), 1.5f);
    }
    {
        auto [value, err] = state.read_channel(2);
        ASSERT_NIL(err);
        EXPECT_EQ(std::get<int32_t>(value), 42);
    }
}

TEST(StateTest, WriteChannel) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    state.register_channel(1, telem::FLOAT64_T);

    // Write from RT thread
    auto err = state.write_channel<double>(1, 3.14);
    ASSERT_NIL(err);

    // Check output queue
    arc::ChannelOutput output;
    EXPECT_TRUE(output_queue.pop(output));
    EXPECT_EQ(output.channel_id, 1);
    EXPECT_EQ(std::get<double>(output.value), 3.14);
}

TEST(StateTest, StatePersistence) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Load state (should initialize)
    arc::StateKey key = arc::make_state_key(1, 5);
    int32_t value1 = state.load_state(key, 42);
    EXPECT_EQ(value1, 42);

    // Store new value
    state.store_state(key, 100);

    // Load again (should get stored value, not init)
    int32_t value2 = state.load_state(key, 42);
    EXPECT_EQ(value2, 100);
}

TEST(StateTest, MultipleStateVariables) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Store multiple variables
    for (uint32_t i = 0; i < 100; i++) {
        arc::StateKey key = arc::make_state_key(1, i);
        state.store_state(key, static_cast<int32_t>(i * 10));
    }

    // Load and verify
    for (uint32_t i = 0; i < 100; i++) {
        arc::StateKey key = arc::make_state_key(1, i);
        int32_t value = state.load_state(key, 0);
        EXPECT_EQ(value, static_cast<int32_t>(i * 10));
    }
}

TEST(StateTest, RegisterNode) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    arc::NodeMetadata meta{"add"};
    meta.type = "binary_op";
    meta.output_params = {"out"};
    meta.read_channels = {1, 2};
    meta.write_channels = {3};

    state.register_node(meta);

    auto *retrieved_meta = state.get_node_metadata("add");
    ASSERT_NE(retrieved_meta, nullptr);
    EXPECT_EQ(retrieved_meta->key, "add");
    EXPECT_EQ(retrieved_meta->type, "binary_op");
    EXPECT_EQ(retrieved_meta->output_params.size(), 1);
    EXPECT_EQ(retrieved_meta->read_channels.size(), 2);

    // Check output was pre-allocated
    arc::Handle out_handle{"add", "out"};
    const auto& vp = state.get_output(out_handle);
    // Initially null
    EXPECT_EQ(vp.data, nullptr);
    EXPECT_EQ(vp.time, nullptr);
}

TEST(StateTest, EdgeGraphOperations) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    // Build graph: A.out → B.in1, A.out → C.in2, B.out → C.in1
    state.add_edge(arc::Edge{arc::Handle{"A", "out"}, arc::Handle{"B", "in1"}});
    state.add_edge(arc::Edge{arc::Handle{"A", "out"}, arc::Handle{"C", "in2"}});
    state.add_edge(arc::Edge{arc::Handle{"B", "out"}, arc::Handle{"C", "in1"}});

    // Test incoming edges
    auto incoming_b = state.incoming_edges("B");
    EXPECT_EQ(incoming_b.size(), 1);
    EXPECT_EQ(incoming_b[0].source.node, "A");
    EXPECT_EQ(incoming_b[0].source.param, "out");

    auto incoming_c = state.incoming_edges("C");
    EXPECT_EQ(incoming_c.size(), 2);

    // Test outgoing edges
    auto outgoing_a = state.outgoing_edges("A");
    EXPECT_EQ(outgoing_a.size(), 2);

    auto outgoing_b = state.outgoing_edges("B");
    EXPECT_EQ(outgoing_b.size(), 1);
    EXPECT_EQ(outgoing_b[0].target.node, "C");
}

TEST(StateTest, NodeOutputStorage) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);

    arc::NodeMetadata meta{"test_node"};
    meta.output_params = {"result"};
    state.register_node(meta);

    // Get output handle
    arc::Handle handle{"test_node", "result"};
    auto& vp = state.get_output(handle);

    // Write data
    vp.data = std::make_shared<telem::Series>(std::vector<float>{1.0f, 2.0f});
    vp.time = std::make_shared<telem::Series>(
        std::vector<telem::TimeStamp>{telem::TimeStamp{100}, telem::TimeStamp{200}}
    );

    // Read back
    const auto& vp_read = state.get_output(handle);
    EXPECT_EQ(vp_read.data->size(), 2);
    EXPECT_EQ(vp_read.data->at<float>(0), 1.0f);
    EXPECT_EQ(vp_read.time->size(), 2);
}

TEST(StateTest, StateKeyHelpers) {
    uint32_t func_id = 0x12345678;
    uint32_t var_id = 0x9ABCDEF0;

    arc::StateKey key = arc::make_state_key(func_id, var_id);

    EXPECT_EQ(arc::state_key_func_id(key), func_id);
    EXPECT_EQ(arc::state_key_var_id(key), var_id);
}
