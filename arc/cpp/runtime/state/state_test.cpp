// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

    auto state = ASSERT_NIL_P(s.node("test"));
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

    auto producer_node = ASSERT_NIL_P(s.node("producer"));

    auto &o = producer_node.output(0);
    o->resize(3);
    o->set(0, 1.0f);
    o->set(1, 2.0f);
    o->set(2, 3.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(3);
    o_time->set(0, telem::TimeStamp(1 * telem::MICROSECOND));
    o_time->set(1, telem::TimeStamp(2 * telem::MICROSECOND));
    o_time->set(2, telem::TimeStamp(3 * telem::MICROSECOND));

    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

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

    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));
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

    auto producer_node = ASSERT_NIL_P(s.node("producer"));
    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    auto &o = producer_node.output(0);
    o->resize(2);
    o->set(0, 1.0f);
    o->set(1, 2.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(2);
    o_time->set(0, telem::TimeStamp(1 * telem::MICROSECOND));
    o_time->set(1, telem::TimeStamp(2 * telem::MICROSECOND));

    bool triggered1 = consumer_node.refresh_inputs();
    ASSERT_TRUE(triggered1);
    EXPECT_EQ(consumer_node.input(0)->size(), 2);

    bool triggered2 = consumer_node.refresh_inputs();
    ASSERT_FALSE(triggered2);

    o->resize(3);
    o->set(2, 3.0f);
    o_time->resize(3);
    o_time->set(2, telem::TimeStamp(3 * telem::MICROSECOND));

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

    auto producer1_node = ASSERT_NIL_P(s.node("producer1"));
    auto producer2_node = ASSERT_NIL_P(s.node("producer2"));
    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    auto &o1 = producer1_node.output(0);
    o1->resize(2);
    o1->set(0, 1.0f);
    o1->set(1, 2.0f);

    auto &o1_time = producer1_node.output_time(0);
    o1_time->resize(2);
    o1_time->set(0, telem::TimeStamp(1 * telem::MICROSECOND));
    o1_time->set(1, telem::TimeStamp(2 * telem::MICROSECOND));

    bool triggered1 = consumer_node.refresh_inputs();
    ASSERT_FALSE(triggered1);

    auto &o2 = producer2_node.output(0);
    o2->resize(2);
    o2->set(0, 10.0f);
    o2->set(1, 20.0f);

    auto &o2_time = producer2_node.output_time(0);
    o2_time->resize(2);
    o2_time->set(0, telem::TimeStamp(1 * telem::MICROSECOND));
    o2_time->set(1, telem::TimeStamp(2 * telem::MICROSECOND));

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

    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    // First refresh triggers because default values are unconsumed
    bool triggered = consumer_node.refresh_inputs();
    ASSERT_TRUE(triggered);
    EXPECT_EQ(consumer_node.input(0)->size(), 1);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 42.0f);

    // Second refresh should NOT trigger because default was consumed
    bool triggered2 = consumer_node.refresh_inputs();
    ASSERT_FALSE(triggered2);
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

    auto producer_node = ASSERT_NIL_P(s.node("producer"));

    auto &o = producer_node.output(0);
    o->resize(2);
    o->set(0, 100.0f);
    o->set(1, 200.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(2);
    o_time->set(0, telem::TimeStamp(1 * telem::MICROSECOND));
    o_time->set(1, telem::TimeStamp(2 * telem::MICROSECOND));

    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    bool triggered = consumer_node.refresh_inputs();
    ASSERT_TRUE(triggered);
    EXPECT_EQ(consumer_node.input(0)->size(), 2);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 100.0f);
    EXPECT_EQ(consumer_node.input(0)->at<float>(1), 200.0f);
}

/// @brief Helper to create a minimal State for channel read/write tests
State create_minimal_state() {
    arc::ir::Node ir_node;
    ir_node.key = "test";
    ir_node.type = "test";

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);
    ir.functions.push_back(fn);

    const Config cfg{.ir = ir, .channels = {}};
    return State(cfg);
}

TEST(StateTest, ClearReads_PreservesLatestSeries) {
    State s = create_minimal_state();

    auto series1 = telem::Series(telem::FLOAT32_T, 3);
    series1.write(1.0f);
    series1.write(2.0f);
    series1.write(3.0f);
    s.ingest(telem::Frame(10, std::move(series1)));

    auto series2 = telem::Series(telem::FLOAT32_T, 2);
    series2.write(4.0f);
    series2.write(5.0f);
    s.ingest(telem::Frame(10, std::move(series2)));

    auto [data_before, ok_before] = s.read_channel(10);
    ASSERT_TRUE(ok_before);
    ASSERT_EQ(data_before.series.size(), 2);

    s.flush();

    auto [data_after, ok_after] = s.read_channel(10);
    ASSERT_TRUE(ok_after);
    ASSERT_EQ(data_after.series.size(), 1);
    EXPECT_EQ(data_after.series[0].size(), 2);
    EXPECT_EQ(data_after.series[0].at<float>(0), 4.0f);
    EXPECT_EQ(data_after.series[0].at<float>(1), 5.0f);
}

TEST(StateTest, ClearReads_PreservesMultipleChannels) {
    State s = create_minimal_state();

    auto series1 = telem::Series(telem::FLOAT32_T, 2);
    series1.write(1.0f);
    series1.write(2.0f);
    s.ingest(telem::Frame(10, std::move(series1)));

    auto series2 = telem::Series(telem::FLOAT64_T, 3);
    series2.write(10.0);
    series2.write(20.0);
    series2.write(30.0);
    s.ingest(telem::Frame(20, std::move(series2)));

    s.flush();

    auto [data10, ok10] = s.read_channel(10);
    ASSERT_TRUE(ok10);
    ASSERT_EQ(data10.series.size(), 1);
    EXPECT_EQ(data10.series[0].at<float>(-1), 2.0f);

    auto [data20, ok20] = s.read_channel(20);
    ASSERT_TRUE(ok20);
    ASSERT_EQ(data20.series.size(), 1);
    EXPECT_EQ(data20.series[0].at<double>(-1), 30.0);
}

TEST(StateTest, ClearReads_PreservedDataAvailableNextCycle) {
    State s = create_minimal_state();

    auto series1 = telem::Series(telem::FLOAT32_T, 2);
    series1.write(1.0f);
    series1.write(2.0f);
    s.ingest(telem::Frame(10, std::move(series1)));
    s.flush();

    auto series2 = telem::Series(telem::FLOAT32_T, 2);
    series2.write(3.0f);
    series2.write(4.0f);
    s.ingest(telem::Frame(20, std::move(series2)));

    auto [data10, ok10] = s.read_channel(10);
    ASSERT_TRUE(ok10);
    EXPECT_EQ(data10.series[0].at<float>(-1), 2.0f);

    auto [data20, ok20] = s.read_channel(20);
    ASSERT_TRUE(ok20);
    EXPECT_EQ(data20.series[0].at<float>(-1), 4.0f);

    s.flush();

    auto [data10_2, ok10_2] = s.read_channel(10);
    ASSERT_TRUE(ok10_2);
    EXPECT_EQ(data10_2.series[0].at<float>(-1), 2.0f);

    auto [data20_2, ok20_2] = s.read_channel(20);
    ASSERT_TRUE(ok20_2);
    EXPECT_EQ(data20_2.series[0].at<float>(-1), 4.0f);
}

TEST(StateTest, ClearReads_NewDataOverwritesPreserved) {
    State s = create_minimal_state();

    auto series1 = telem::Series(telem::FLOAT32_T, 1);
    series1.write(100.0f);
    s.ingest(telem::Frame(10, std::move(series1)));
    s.flush();

    auto [data1, ok1] = s.read_channel(10);
    ASSERT_TRUE(ok1);
    EXPECT_EQ(data1.series[0].at<float>(-1), 100.0f);

    auto series2 = telem::Series(telem::FLOAT32_T, 1);
    series2.write(200.0f);
    s.ingest(telem::Frame(10, std::move(series2)));
    s.flush();

    auto [data2, ok2] = s.read_channel(10);
    ASSERT_TRUE(ok2);
    ASSERT_EQ(data2.series.size(), 1);
    EXPECT_EQ(data2.series[0].at<float>(-1), 200.0f);
}

TEST(StateTest, ClearReads_SingleSeriesNoOp) {
    State s = create_minimal_state();

    auto series = telem::Series(telem::INT32_T, 3);
    series.write(1);
    series.write(2);
    series.write(3);
    s.ingest(telem::Frame(10, std::move(series)));

    s.flush();

    auto [data, ok] = s.read_channel(10);
    ASSERT_TRUE(ok);
    ASSERT_EQ(data.series.size(), 1);
    EXPECT_EQ(data.series[0].size(), 3);
    EXPECT_EQ(data.series[0].at<int32_t>(0), 1);
    EXPECT_EQ(data.series[0].at<int32_t>(1), 2);
    EXPECT_EQ(data.series[0].at<int32_t>(2), 3);
}

TEST(StateTest, ClearReads_EmptyState) {
    State s = create_minimal_state();

    s.flush();

    auto [data, ok] = s.read_channel(10);
    ASSERT_FALSE(ok);
    EXPECT_TRUE(data.series.empty());
}

TEST(StateTest, ReadChannel_UnknownChannel) {
    State s = create_minimal_state();

    auto series = telem::Series(telem::FLOAT32_T, 1);
    series.write(1.0f);
    s.ingest(telem::Frame(10, std::move(series)));

    auto [data, ok] = s.read_channel(99);
    ASSERT_FALSE(ok);
    EXPECT_TRUE(data.series.empty());
}
