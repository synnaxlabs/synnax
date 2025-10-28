// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/nodes/interval/factory.h"

#include "gtest/gtest.h"
#include "x/cpp/xtest/xtest.h"
#include <nlohmann/json.hpp>

namespace arc {

class IntervalNodeFactoryTest : public ::testing::Test {
protected:
    std::unique_ptr<queue::SPSC<ChannelUpdate>> input_queue_;
    std::unique_ptr<queue::SPSC<ChannelOutput>> output_queue_;
    std::unique_ptr<State> state_;
    ir::IR ir_;

    void SetUp() override {
        input_queue_ = std::make_unique<queue::SPSC<ChannelUpdate>>(16);
        output_queue_ = std::make_unique<queue::SPSC<ChannelOutput>>(16);
        state_ = std::make_unique<State>(input_queue_.get(), output_queue_.get());

        // Register a channel for interval output
        state_->register_channel(1, telem::UINT8_T);
    }
};

TEST_F(IntervalNodeFactoryTest, ReturnsNotFoundForNonIntervalType) {
    interval::IntervalNodeFactory factory;

    ir::Node node{"node1"};
    node.type = "not_interval";

    NodeFactoryConfig cfg{node, *state_, ir_};

    auto [created_node, err] = factory.create(cfg);
    EXPECT_EQ(created_node, nullptr);
    ASSERT_OCCURRED_AS(err, "NOT_FOUND");
}

TEST_F(IntervalNodeFactoryTest, ReturnsErrorWhenPeriodMissing) {
    interval::IntervalNodeFactory factory;

    ir::Node node{"interval1"};
    node.type = "interval";
    // Missing period in config_values
    node.channels.write["output"] = 1;

    NodeFactoryConfig cfg{node, *state_, ir_};

    auto [created_node, err] = factory.create(cfg);
    EXPECT_EQ(created_node, nullptr);
    ASSERT_OCCURRED_AS(err, "arc.factory.interval_missing_period");
}

TEST_F(IntervalNodeFactoryTest, ReturnsErrorWhenOutputChannelMissing) {
    interval::IntervalNodeFactory factory;

    ir::Node node{"interval1"};
    node.type = "interval";
    node.config_values["period"] = 1000000000ULL;  // 1 second in nanoseconds
    // Missing output channel

    NodeFactoryConfig cfg{node, *state_, ir_};

    auto [created_node, err] = factory.create(cfg);
    EXPECT_EQ(created_node, nullptr);
    ASSERT_OCCURRED_AS(err, "arc.factory.interval_missing_output");
}

TEST_F(IntervalNodeFactoryTest, ReturnsErrorWhenPeriodInvalidType) {
    interval::IntervalNodeFactory factory;

    ir::Node node{"interval1"};
    node.type = "interval";
    node.config_values["period"] = "not_a_number";  // Invalid type
    node.channels.write["output"] = 1;

    NodeFactoryConfig cfg{node, *state_, ir_};

    auto [created_node, err] = factory.create(cfg);
    EXPECT_EQ(created_node, nullptr);
    ASSERT_OCCURRED_AS(err, "arc.factory.interval_invalid_period");
}

TEST_F(IntervalNodeFactoryTest, CreatesIntervalNodeSuccessfully) {
    interval::IntervalNodeFactory factory;

    ir::Node node{"interval1"};
    node.type = "interval";
    node.config_values["period"] = 100000000ULL;  // 100ms in nanoseconds
    node.channels.write["output"] = 1;

    NodeFactoryConfig cfg{node, *state_, ir_};

    auto [created_node, err] = factory.create(cfg);
    ASSERT_NIL(err);
    ASSERT_NE(created_node, nullptr);
    EXPECT_EQ(created_node->id(), "interval1");

    // Verify it's actually an IntervalNode
    auto* interval_node = dynamic_cast<interval::IntervalNode*>(created_node.get());
    ASSERT_NE(interval_node, nullptr);
}

TEST_F(IntervalNodeFactoryTest, CreatesIntervalNodeWithDifferentPeriods) {
    interval::IntervalNodeFactory factory;

    // Test with 1Hz (1 second)
    ir::Node node1{"interval1"};
    node1.type = "interval";
    node1.config_values["period"] = 1000000000ULL;
    node1.channels.write["output"] = 1;

    NodeFactoryConfig cfg1{node1, *state_, ir_};
    auto [created_node1, err1] = factory.create(cfg1);
    ASSERT_NIL(err1);
    ASSERT_NE(created_node1, nullptr);

    // Test with 10Hz (100ms)
    ir::Node node2{"interval2"};
    node2.type = "interval";
    node2.config_values["period"] = 100000000ULL;
    node2.channels.write["output"] = 1;

    NodeFactoryConfig cfg2{node2, *state_, ir_};
    auto [created_node2, err2] = factory.create(cfg2);
    ASSERT_NIL(err2);
    ASSERT_NE(created_node2, nullptr);

    // Test with 1kHz (1ms)
    ir::Node node3{"interval3"};
    node3.type = "interval";
    node3.config_values["period"] = 1000000ULL;
    node3.channels.write["output"] = 1;

    NodeFactoryConfig cfg3{node3, *state_, ir_};
    auto [created_node3, err3] = factory.create(cfg3);
    ASSERT_NIL(err3);
    ASSERT_NE(created_node3, nullptr);
}

TEST_F(IntervalNodeFactoryTest, DoesNotRequireWASMRuntime) {
    interval::IntervalNodeFactory factory;

    ir::Node node{"interval1"};
    node.type = "interval";
    node.config_values["period"] = 100000000ULL;
    node.channels.write["output"] = 1;

    // Pass nullptr for runtime - interval nodes don't need WASM
    NodeFactoryConfig cfg{node, *state_, ir_};

    auto [created_node, err] = factory.create(cfg);
    ASSERT_NIL(err);
    ASSERT_NE(created_node, nullptr);
}

}  // namespace arc
