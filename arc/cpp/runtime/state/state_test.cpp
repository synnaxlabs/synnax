// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/series.h"
#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/runtime/state/state.h"

using namespace arc::runtime::state;

/// @brief Test basic state creation and node retrieval
TEST(StateTest, CreateStateAndGetNode) {
    arc::ir::Node ir_node;
    ir_node.key = "test";
    ir_node.type = "test";

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg);

    auto [node, err] = s.node("test");
    ASSERT_NIL(err);
}

/// @brief Test node retrieval for non-existent node
TEST(StateTest, GetNonExistentNode) {
    arc::ir::IR ir;
    Config cfg{.ir = ir, .channels = {}};
    State s(cfg);

    auto [node, err] = s.node("nonexistent");
    ASSERT_OCCURRED_AS(err, xerrors::NOT_FOUND);
}

/// @brief Test basic input alignment with two connected nodes
TEST(StateTest, RefreshInputs_BasicAlignment) {
    arc::ir::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.params.push_back(output_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.params.push_back(input_param);

    arc::ir::Edge edge(
        arc::ir::Handle("producer", "output"),
        arc::ir::Handle("consumer", "input")
    );

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(producer);
    ir.nodes.push_back(consumer);
    ir.edges.push_back(edge);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg);

    auto [producer_node, err1] = s.node("producer");
    ASSERT_NIL(err1);

    auto &o = producer_node.output(0);
    o->resize(3);
    o->set(0, 1.0f);
    o->set(1, 2.0f);
    o->set(2, 3.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(3);
    o_time->set(0, telem::TimeStamp(1000).nanoseconds());
    o_time->set(1, telem::TimeStamp(2000).nanoseconds());
    o_time->set(2, telem::TimeStamp(3000).nanoseconds());

    auto [consumer_node, err2] = s.node("consumer");
    ASSERT_NIL(err2);

    bool triggered = consumer_node.refresh_inputs();
    ASSERT_TRUE(triggered);

    EXPECT_EQ(consumer_node.input(0)->size(), 3);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 1.0f);
    EXPECT_EQ(consumer_node.input(0)->at<float>(1), 2.0f);
    EXPECT_EQ(consumer_node.input(0)->at<float>(2), 3.0f);
}

/// @brief Test that refresh_inputs returns false when upstream output is empty
TEST(StateTest, RefreshInputs_NoTriggerOnEmpty) {
    arc::ir::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.params.push_back(output_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.params.push_back(input_param);

    arc::ir::Edge edge(
        arc::ir::Handle("producer", "output"),
        arc::ir::Handle("consumer", "input")
    );

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(producer);
    ir.nodes.push_back(consumer);
    ir.edges.push_back(edge);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg);

    auto [consumer_node, err] = s.node("consumer");
    ASSERT_NIL(err);

    bool triggered = consumer_node.refresh_inputs();
    ASSERT_FALSE(triggered);
}

/// @brief Test that watermark tracking prevents reprocessing the same data
TEST(StateTest, RefreshInputs_WatermarkTracking) {
    arc::ir::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.params.push_back(output_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.params.push_back(input_param);

    arc::ir::Edge edge(
        arc::ir::Handle("producer", "output"),
        arc::ir::Handle("consumer", "input")
    );

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(producer);
    ir.nodes.push_back(consumer);
    ir.edges.push_back(edge);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg);

    auto [producer_node, err1] = s.node("producer");
    ASSERT_NIL(err1);
    auto [consumer_node, err2] = s.node("consumer");
    ASSERT_NIL(err2);

    auto &o = producer_node.output(0);
    o->resize(2);
    o->set(0, 1.0f);
    o->set(1, 2.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(2);
    o_time->set(0, telem::TimeStamp(1000).nanoseconds());
    o_time->set(1, telem::TimeStamp(2000).nanoseconds());

    bool triggered1 = consumer_node.refresh_inputs();
    ASSERT_TRUE(triggered1);
    EXPECT_EQ(consumer_node.input(0)->size(), 2);

    bool triggered2 = consumer_node.refresh_inputs();
    ASSERT_FALSE(triggered2);

    o->resize(3);
    o->set(2, 3.0f);
    o_time->resize(3);
    o_time->set(2, telem::TimeStamp(3000).nanoseconds());

    bool triggered3 = consumer_node.refresh_inputs();
    ASSERT_TRUE(triggered3);
    EXPECT_EQ(consumer_node.input(0)->size(), 3);
}

/// @brief Test node with multiple inputs only triggers when all have data
TEST(StateTest, RefreshInputs_MultipleInputs) {
    arc::ir::Param output1_param;
    output1_param.name = "output";
    output1_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Param output2_param;
    output2_param.name = "output";
    output2_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Param input1_param;
    input1_param.name = "input1";
    input1_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Param input2_param;
    input2_param.name = "input2";
    input2_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node producer1;
    producer1.key = "producer1";
    producer1.type = "producer1";
    producer1.outputs.params.push_back(output1_param);

    arc::ir::Node producer2;
    producer2.key = "producer2";
    producer2.type = "producer2";
    producer2.outputs.params.push_back(output2_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.params.push_back(input1_param);
    consumer.inputs.params.push_back(input2_param);

    arc::ir::Edge edge1(
        arc::ir::Handle("producer1", "output"),
        arc::ir::Handle("consumer", "input1")
    );

    arc::ir::Edge edge2(
        arc::ir::Handle("producer2", "output"),
        arc::ir::Handle("consumer", "input2")
    );

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(producer1);
    ir.nodes.push_back(producer2);
    ir.nodes.push_back(consumer);
    ir.edges.push_back(edge1);
    ir.edges.push_back(edge2);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg);

    auto [producer1_node, err1] = s.node("producer1");
    ASSERT_NIL(err1);
    auto [producer2_node, err2] = s.node("producer2");
    ASSERT_NIL(err2);
    auto [consumer_node, err3] = s.node("consumer");
    ASSERT_NIL(err3);

    auto &o1 = producer1_node.output(0);
    o1->resize(2);
    o1->set(0, 1.0f);
    o1->set(1, 2.0f);

    auto &o1_time = producer1_node.output_time(0);
    o1_time->resize(2);
    o1_time->set(0, telem::TimeStamp(1000).nanoseconds());
    o1_time->set(1, telem::TimeStamp(2000).nanoseconds());

    bool triggered1 = consumer_node.refresh_inputs();
    ASSERT_FALSE(triggered1);

    auto &o2 = producer2_node.output(0);
    o2->resize(2);
    o2->set(0, 10.0f);
    o2->set(1, 20.0f);

    auto &o2_time = producer2_node.output_time(0);
    o2_time->resize(2);
    o2_time->set(0, telem::TimeStamp(1000).nanoseconds());
    o2_time->set(1, telem::TimeStamp(2000).nanoseconds());

    bool triggered2 = consumer_node.refresh_inputs();
    ASSERT_TRUE(triggered2);
    EXPECT_EQ(consumer_node.input(0)->size(), 2);
    EXPECT_EQ(consumer_node.input(1)->size(), 2);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 1.0f);
    EXPECT_EQ(consumer_node.input(1)->at<float>(0), 10.0f);
}

/// @brief Test that unconnected optional input uses default value
TEST(StateTest, OptionalInput_UseDefault) {
    arc::ir::Param input1_param;
    input1_param.name = "input1";
    input1_param.type = arc::types::Type(arc::types::Kind::F32);
    input1_param.value = 42.0f;

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.params.push_back(input1_param);

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(consumer);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg);

    auto [consumer_node, err] = s.node("consumer");
    ASSERT_NIL(err);

    bool triggered = consumer_node.refresh_inputs();
    ASSERT_TRUE(triggered);
    EXPECT_EQ(consumer_node.input(0)->size(), 1);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 42.0f);
}

/// @brief Test that connected input overrides default value
TEST(StateTest, OptionalInput_OverrideDefault) {
    arc::ir::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type(arc::types::Kind::F32);
    input_param.value = 42.0f;

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.params.push_back(output_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.params.push_back(input_param);

    arc::ir::Edge edge(
        arc::ir::Handle("producer", "output"),
        arc::ir::Handle("consumer", "input")
    );

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(producer);
    ir.nodes.push_back(consumer);
    ir.edges.push_back(edge);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg);

    auto [producer_node, err1] = s.node("producer");
    ASSERT_NIL(err1);

    auto &o = producer_node.output(0);
    o->resize(2);
    o->set(0, 100.0f);
    o->set(1, 200.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(2);
    o_time->set(0, telem::TimeStamp(1000).nanoseconds());
    o_time->set(1, telem::TimeStamp(2000).nanoseconds());

    auto [consumer_node, err2] = s.node("consumer");
    ASSERT_NIL(err2);

    bool triggered = consumer_node.refresh_inputs();
    ASSERT_TRUE(triggered);
    EXPECT_EQ(consumer_node.input(0)->size(), 2);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 100.0f);
    EXPECT_EQ(consumer_node.input(0)->at<float>(1), 200.0f);
}
