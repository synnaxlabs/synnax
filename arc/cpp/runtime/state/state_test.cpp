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
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/channels/state.h"
#include "arc/cpp/stl/series/state.h"
#include "arc/cpp/stl/stateful/state.h"
#include "arc/cpp/stl/strings/state.h"

namespace arc::runtime::state {

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
    State s(cfg, arc::runtime::errors::noop_handler);

    auto state = ASSERT_NIL_P(s.node("test"));
}

/// @brief Test basic input alignment with two connected nodes
TEST(StateTest, RefreshInputs_BasicAlignment) {
    arc::types::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.push_back(output_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input_param);

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
    State s(cfg, arc::runtime::errors::noop_handler);

    auto producer_node = ASSERT_NIL_P(s.node("producer"));

    auto &o = producer_node.output(0);
    o->resize(3);
    o->set(0, 1.0f);
    o->set(1, 2.0f);
    o->set(2, 3.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(3);
    o_time->set(0, x::telem::TimeStamp(1 * x::telem::MICROSECOND));
    o_time->set(1, x::telem::TimeStamp(2 * x::telem::MICROSECOND));
    o_time->set(2, x::telem::TimeStamp(3 * x::telem::MICROSECOND));

    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    ASSERT_TRUE(consumer_node.refresh_inputs());

    EXPECT_EQ(consumer_node.input(0)->size(), 3);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 1.0f);
    EXPECT_EQ(consumer_node.input(0)->at<float>(1), 2.0f);
    EXPECT_EQ(consumer_node.input(0)->at<float>(2), 3.0f);
}

/// @brief Test that refresh_inputs returns false when upstream output is empty
TEST(StateTest, RefreshInputs_NoTriggerOnEmpty) {
    arc::types::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.push_back(output_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input_param);

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
    State s(cfg, arc::runtime::errors::noop_handler);

    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));
    ASSERT_FALSE(consumer_node.refresh_inputs());
}

/// @brief Test that watermark tracking prevents reprocessing the same data
TEST(StateTest, RefreshInputs_WatermarkTracking) {
    arc::types::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.push_back(output_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input_param);

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
    State s(cfg, arc::runtime::errors::noop_handler);

    auto producer_node = ASSERT_NIL_P(s.node("producer"));
    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    auto &o = producer_node.output(0);
    o->resize(2);
    o->set(0, 1.0f);
    o->set(1, 2.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(2);
    o_time->set(0, x::telem::TimeStamp(1 * x::telem::MICROSECOND));
    o_time->set(1, x::telem::TimeStamp(2 * x::telem::MICROSECOND));

    ASSERT_TRUE(consumer_node.refresh_inputs());
    EXPECT_EQ(consumer_node.input(0)->size(), 2);

    ASSERT_FALSE(consumer_node.refresh_inputs());

    o->resize(3);
    o->set(2, 3.0f);
    o_time->resize(3);
    o_time->set(2, x::telem::TimeStamp(3 * x::telem::MICROSECOND));

    ASSERT_TRUE(consumer_node.refresh_inputs());
    EXPECT_EQ(consumer_node.input(0)->size(), 3);
}

/// @brief Test node with multiple inputs only triggers when all have data
TEST(StateTest, RefreshInputs_MultipleInputs) {
    arc::types::Param output1_param;
    output1_param.name = "output";
    output1_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param output2_param;
    output2_param.name = "output";
    output2_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param input1_param;
    input1_param.name = "input1";
    input1_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param input2_param;
    input2_param.name = "input2";
    input2_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::ir::Node producer1;
    producer1.key = "producer1";
    producer1.type = "producer1";
    producer1.outputs.push_back(output1_param);

    arc::ir::Node producer2;
    producer2.key = "producer2";
    producer2.type = "producer2";
    producer2.outputs.push_back(output2_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input1_param);
    consumer.inputs.push_back(input2_param);

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
    State s(cfg, arc::runtime::errors::noop_handler);

    auto producer1_node = ASSERT_NIL_P(s.node("producer1"));
    auto producer2_node = ASSERT_NIL_P(s.node("producer2"));
    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    auto &o1 = producer1_node.output(0);
    o1->resize(2);
    o1->set(0, 1.0f);
    o1->set(1, 2.0f);

    auto &o1_time = producer1_node.output_time(0);
    o1_time->resize(2);
    o1_time->set(0, x::telem::TimeStamp(1 * x::telem::MICROSECOND));
    o1_time->set(1, x::telem::TimeStamp(2 * x::telem::MICROSECOND));

    ASSERT_FALSE(consumer_node.refresh_inputs());

    auto &o2 = producer2_node.output(0);
    o2->resize(2);
    o2->set(0, 10.0f);
    o2->set(1, 20.0f);

    auto &o2_time = producer2_node.output_time(0);
    o2_time->resize(2);
    o2_time->set(0, x::telem::TimeStamp(1 * x::telem::MICROSECOND));
    o2_time->set(1, x::telem::TimeStamp(2 * x::telem::MICROSECOND));

    ASSERT_TRUE(consumer_node.refresh_inputs());
    EXPECT_EQ(consumer_node.input(0)->size(), 2);
    EXPECT_EQ(consumer_node.input(1)->size(), 2);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 1.0f);
    EXPECT_EQ(consumer_node.input(1)->at<float>(0), 10.0f);
}

/// @brief Test that unconnected optional input uses default value
TEST(StateTest, OptionalInput_UseDefault) {
    arc::types::Param input1_param;
    input1_param.name = "input1";
    input1_param.type = arc::types::Type{.kind = arc::types::Kind::F32};
    input1_param.value = 42.0f;

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input1_param);

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(consumer);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg, arc::runtime::errors::noop_handler);

    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    // First refresh triggers because default values are unconsumed
    ASSERT_TRUE(consumer_node.refresh_inputs());
    EXPECT_EQ(consumer_node.input(0)->size(), 1);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 42.0f);

    // Second refresh should NOT trigger because default was consumed
    ASSERT_FALSE(consumer_node.refresh_inputs());
}

/// @brief reset() re-arms literal-valued inputs so a node whose stage is
/// re-entered runs again instead of staying consumed from its first activation.
TEST(StateTest, Reset_RearmsLiteralInputsOnStageReentry) {
    arc::types::Param input1_param;
    input1_param.name = "input1";
    input1_param.type = arc::types::Type{.kind = arc::types::Kind::F32};
    input1_param.value = 42.0f;

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input1_param);

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(consumer);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg, arc::runtime::errors::noop_handler);

    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    // Runs on first activation, then stays consumed on subsequent cycles.
    ASSERT_TRUE(consumer_node.refresh_inputs());
    ASSERT_FALSE(consumer_node.refresh_inputs());

    // Stage re-entry must re-arm the literal input so the node runs again.
    consumer_node.reset();
    ASSERT_TRUE(consumer_node.refresh_inputs());
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 42.0f);
}

/// @brief Test that connected input overrides default value
TEST(StateTest, OptionalInput_OverrideDefault) {
    arc::types::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type{.kind = arc::types::Kind::F32};
    input_param.value = 42.0f;

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.push_back(output_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input_param);

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
    State s(cfg, arc::runtime::errors::noop_handler);

    auto producer_node = ASSERT_NIL_P(s.node("producer"));

    auto &o = producer_node.output(0);
    o->resize(2);
    o->set(0, 100.0f);
    o->set(1, 200.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(2);
    o_time->set(0, x::telem::TimeStamp(1 * x::telem::MICROSECOND));
    o_time->set(1, x::telem::TimeStamp(2 * x::telem::MICROSECOND));

    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    ASSERT_TRUE(consumer_node.refresh_inputs());
    EXPECT_EQ(consumer_node.input(0)->size(), 2);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 100.0f);
    EXPECT_EQ(consumer_node.input(0)->at<float>(1), 200.0f);
}

/// @brief init_input seeds a connected input so refresh_inputs doesn't block.
TEST(StateTest, InitInput_SeedsConnectedInput) {
    arc::types::Param data_output;
    data_output.name = "output";
    data_output.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param reset_output;
    reset_output.name = "output";
    reset_output.type = arc::types::Type{.kind = arc::types::Kind::U8};

    arc::types::Param input_data;
    input_data.name = "data";
    input_data.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param input_reset;
    input_reset.name = "reset";
    input_reset.type = arc::types::Type{.kind = arc::types::Kind::U8};

    arc::ir::Node data_producer;
    data_producer.key = "data_producer";
    data_producer.type = "producer";
    data_producer.outputs.push_back(data_output);

    arc::ir::Node reset_producer;
    reset_producer.key = "reset_producer";
    reset_producer.type = "producer";
    reset_producer.outputs.push_back(reset_output);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input_data);
    consumer.inputs.push_back(input_reset);

    arc::ir::Edge data_edge(
        arc::ir::Handle("data_producer", "output"),
        arc::ir::Handle("consumer", "data")
    );
    arc::ir::Edge reset_edge(
        arc::ir::Handle("reset_producer", "output"),
        arc::ir::Handle("consumer", "reset")
    );

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(data_producer);
    ir.nodes.push_back(reset_producer);
    ir.nodes.push_back(consumer);
    ir.edges.push_back(data_edge);
    ir.edges.push_back(reset_edge);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg, arc::runtime::errors::noop_handler);

    auto data_node = ASSERT_NIL_P(s.node("data_producer"));
    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    // Without init_input, refresh_inputs blocks because reset has no data.
    auto &o = data_node.output(0);
    *o = x::telem::Series(std::vector<float>{1.0f, 2.0f});
    auto &o_time = data_node.output_time(0);
    *o_time = x::telem::Series(std::vector<int64_t>{1000, 2000});

    ASSERT_FALSE(consumer_node.refresh_inputs());

    // After init_input, the reset input has seed data so refresh_inputs succeeds.
    consumer_node.init_input(
        1,
        x::mem::make_local_shared<x::telem::Series>(static_cast<uint8_t>(0)),
        x::mem::make_local_shared<x::telem::Series>(x::telem::TimeStamp(1))
    );

    ASSERT_TRUE(consumer_node.refresh_inputs());
    EXPECT_EQ(consumer_node.input(0)->size(), 2);
    EXPECT_EQ(consumer_node.input(0)->at<float>(0), 1.0f);
    EXPECT_EQ(consumer_node.input(1)->size(), 1);
    EXPECT_EQ(consumer_node.input(1)->at<uint8_t>(0), 0);
}

/// @brief init_input data gets overwritten when the real source produces data.
TEST(StateTest, InitInput_OverwrittenByRealData) {
    arc::types::Param data_output;
    data_output.name = "output";
    data_output.type = arc::types::Type{.kind = arc::types::Kind::U8};

    arc::types::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type{.kind = arc::types::Kind::U8};

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.push_back(data_output);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input_param);

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
    State s(cfg, arc::runtime::errors::noop_handler);

    auto producer_node = ASSERT_NIL_P(s.node("producer"));
    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    // Seed with init_input.
    consumer_node.init_input(
        0,
        x::mem::make_local_shared<x::telem::Series>(static_cast<uint8_t>(0)),
        x::mem::make_local_shared<x::telem::Series>(x::telem::TimeStamp(1))
    );
    ASSERT_TRUE(consumer_node.refresh_inputs());
    EXPECT_EQ(consumer_node.input(0)->at<uint8_t>(0), 0);

    // Real data overwrites the seed.
    *producer_node.output(0) = x::telem::Series(static_cast<uint8_t>(1));
    *producer_node.output_time(0) = x::telem::Series(
        x::telem::TimeStamp(2000).nanoseconds()
    );
    ASSERT_TRUE(consumer_node.refresh_inputs());
    EXPECT_EQ(consumer_node.input(0)->at<uint8_t>(0), 1);
}

/// @brief Helper to create a minimal State for authority/node tests
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
    return State(cfg, arc::runtime::errors::noop_handler);
}

TEST(ChannelStateTest, FlushPreservesLatestSeries) {
    stl::channels::State s;

    auto series1 = x::telem::Series(x::telem::FLOAT32_T, 3);
    series1.write(1.0f);
    series1.write(2.0f);
    series1.write(3.0f);
    s.ingest(x::telem::Frame(10, std::move(series1)));

    auto series2 = x::telem::Series(x::telem::FLOAT32_T, 2);
    series2.write(4.0f);
    series2.write(5.0f);
    s.ingest(x::telem::Frame(10, std::move(series2)));

    auto [data_before, ok_before] = s.read_value(10);
    ASSERT_TRUE(ok_before);
    ASSERT_EQ(data_before.series.size(), 2);

    x::telem::Frame out;
    s.flush_into(out);

    auto [data_after, ok_after] = s.read_value(10);
    ASSERT_TRUE(ok_after);
    ASSERT_EQ(data_after.series.size(), 1);
    EXPECT_EQ(data_after.series[0].size(), 2);
    EXPECT_EQ(data_after.series[0].at<float>(0), 4.0f);
    EXPECT_EQ(data_after.series[0].at<float>(1), 5.0f);
}

TEST(ChannelStateTest, FlushPreservesMultipleChannels) {
    stl::channels::State s;

    auto series1 = x::telem::Series(x::telem::FLOAT32_T, 2);
    series1.write(1.0f);
    series1.write(2.0f);
    s.ingest(x::telem::Frame(10, std::move(series1)));

    auto series2 = x::telem::Series(x::telem::FLOAT64_T, 3);
    series2.write(10.0);
    series2.write(20.0);
    series2.write(30.0);
    s.ingest(x::telem::Frame(20, std::move(series2)));

    x::telem::Frame out;
    s.flush_into(out);

    auto [data10, ok10] = s.read_value(10);
    ASSERT_TRUE(ok10);
    ASSERT_EQ(data10.series.size(), 1);
    EXPECT_EQ(data10.series[0].at<float>(-1), 2.0f);

    auto [data20, ok20] = s.read_value(20);
    ASSERT_TRUE(ok20);
    ASSERT_EQ(data20.series.size(), 1);
    EXPECT_EQ(data20.series[0].at<double>(-1), 30.0);
}

TEST(ChannelStateTest, PreservedDataAvailableNextCycle) {
    stl::channels::State s;

    auto series1 = x::telem::Series(x::telem::FLOAT32_T, 2);
    series1.write(1.0f);
    series1.write(2.0f);
    s.ingest(x::telem::Frame(10, std::move(series1)));
    x::telem::Frame out1;
    s.flush_into(out1);

    auto series2 = x::telem::Series(x::telem::FLOAT32_T, 2);
    series2.write(3.0f);
    series2.write(4.0f);
    s.ingest(x::telem::Frame(20, std::move(series2)));

    auto [data10, ok10] = s.read_value(10);
    ASSERT_TRUE(ok10);
    EXPECT_EQ(data10.series[0].at<float>(-1), 2.0f);

    auto [data20, ok20] = s.read_value(20);
    ASSERT_TRUE(ok20);
    EXPECT_EQ(data20.series[0].at<float>(-1), 4.0f);

    x::telem::Frame out2;
    s.flush_into(out2);

    auto [data10_2, ok10_2] = s.read_value(10);
    ASSERT_TRUE(ok10_2);
    EXPECT_EQ(data10_2.series[0].at<float>(-1), 2.0f);

    auto [data20_2, ok20_2] = s.read_value(20);
    ASSERT_TRUE(ok20_2);
    EXPECT_EQ(data20_2.series[0].at<float>(-1), 4.0f);
}

TEST(ChannelStateTest, NewDataOverwritesPreserved) {
    stl::channels::State s;

    auto series1 = x::telem::Series(x::telem::FLOAT32_T, 1);
    series1.write(100.0f);
    s.ingest(x::telem::Frame(10, std::move(series1)));
    x::telem::Frame out1;
    s.flush_into(out1);

    auto [data1, ok1] = s.read_value(10);
    ASSERT_TRUE(ok1);
    EXPECT_EQ(data1.series[0].at<float>(-1), 100.0f);

    auto series2 = x::telem::Series(x::telem::FLOAT32_T, 1);
    series2.write(200.0f);
    s.ingest(x::telem::Frame(10, std::move(series2)));
    x::telem::Frame out2;
    s.flush_into(out2);

    auto [data2, ok2] = s.read_value(10);
    ASSERT_TRUE(ok2);
    ASSERT_EQ(data2.series.size(), 1);
    EXPECT_EQ(data2.series[0].at<float>(-1), 200.0f);
}

TEST(ChannelStateTest, SingleSeriesNoOp) {
    stl::channels::State s;

    auto series = x::telem::Series(x::telem::INT32_T, 3);
    series.write(1);
    series.write(2);
    series.write(3);
    s.ingest(x::telem::Frame(10, std::move(series)));

    x::telem::Frame out;
    s.flush_into(out);

    auto [data, ok] = s.read_value(10);
    ASSERT_TRUE(ok);
    ASSERT_EQ(data.series.size(), 1);
    EXPECT_EQ(data.series[0].size(), 3);
    EXPECT_EQ(data.series[0].at<int32_t>(0), 1);
    EXPECT_EQ(data.series[0].at<int32_t>(1), 2);
    EXPECT_EQ(data.series[0].at<int32_t>(2), 3);
}

TEST(ChannelStateTest, EmptyState) {
    stl::channels::State s;

    x::telem::Frame out;
    s.flush_into(out);

    auto [data, ok] = s.read_value(10);
    ASSERT_FALSE(ok);
    EXPECT_TRUE(data.series.empty());
}

TEST(ChannelStateTest, UnknownChannel) {
    stl::channels::State s;

    auto series = x::telem::Series(x::telem::FLOAT32_T, 1);
    series.write(1.0f);
    s.ingest(x::telem::Frame(10, std::move(series)));

    auto [data, ok] = s.read_value(99);
    ASSERT_FALSE(ok);
    EXPECT_TRUE(data.series.empty());
}

TEST(ChannelStateTest, ResetClearsReadsAndWrites) {
    stl::channels::State s;

    auto series = x::telem::Series(x::telem::FLOAT32_T, 2);
    series.write(1.0f);
    series.write(2.0f);
    s.ingest(x::telem::Frame(10, std::move(series)));

    auto [data_before, ok_before] = s.read_value(10);
    ASSERT_TRUE(ok_before);
    ASSERT_EQ(data_before.series.size(), 1);

    s.reset();

    auto [data_after, ok_after] = s.read_value(10);
    ASSERT_FALSE(ok_after);
    EXPECT_TRUE(data_after.series.empty());
}

/// @brief Test that Node::reset clears watermark tracking
TEST(StateTest, NodeReset_ClearsWatermarks) {
    arc::types::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::types::Param input_param;
    input_param.name = "input";
    input_param.type = arc::types::Type{.kind = arc::types::Kind::F32};

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.push_back(output_param);

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(input_param);

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
    State s(cfg, arc::runtime::errors::noop_handler);

    auto producer_node = ASSERT_NIL_P(s.node("producer"));
    auto consumer_node = ASSERT_NIL_P(s.node("consumer"));

    auto &o = producer_node.output(0);
    o->resize(2);
    o->set(0, 1.0f);
    o->set(1, 2.0f);

    auto &o_time = producer_node.output_time(0);
    o_time->resize(2);
    o_time->set(0, x::telem::TimeStamp(1 * x::telem::MICROSECOND));
    o_time->set(1, x::telem::TimeStamp(2 * x::telem::MICROSECOND));

    ASSERT_TRUE(consumer_node.refresh_inputs());

    ASSERT_FALSE(consumer_node.refresh_inputs());

    consumer_node.reset();

    ASSERT_TRUE(consumer_node.refresh_inputs());
}

/// @brief Test that is_series_truthy returns false for empty series
TEST(StateTest, IsSeriesTruthy_EmptySeriesIsFalsy) {
    x::telem::Series empty_series(x::telem::FLOAT32_T, 0);
    EXPECT_FALSE(Node::is_series_truthy(empty_series));
}

/// @brief Test that is_series_truthy returns false for series with zero value
TEST(StateTest, IsSeriesTruthy_ZeroValueIsFalsy) {
    x::telem::Series series(0.0f);
    EXPECT_FALSE(Node::is_series_truthy(series));
}

/// @brief Test that is_series_truthy returns true for series with non-zero value
TEST(StateTest, IsSeriesTruthy_NonZeroValueIsTruthy) {
    x::telem::Series series(42.0f);
    EXPECT_TRUE(Node::is_series_truthy(series));
}

/// @brief Test that is_series_truthy returns false when last element is zero
TEST(StateTest, IsSeriesTruthy_LastElementZeroIsFalsy) {
    x::telem::Series series(x::telem::FLOAT32_T, 3);
    series.write(1.0f);
    series.write(2.0f);
    series.write(0.0f); // Last element is zero
    EXPECT_FALSE(Node::is_series_truthy(series));
}

/// @brief Test that is_series_truthy returns true when last element is non-zero
TEST(StateTest, IsSeriesTruthy_LastElementNonZeroIsTruthy) {
    x::telem::Series series(x::telem::FLOAT32_T, 3);
    series.write(0.0f);
    series.write(0.0f);
    series.write(1.0f); // Last element is non-zero
    EXPECT_TRUE(Node::is_series_truthy(series));
}

/// @brief Test that is_series_truthy works with uint8 series
TEST(StateTest, IsSeriesTruthy_Uint8Series) {
    x::telem::Series zero_series(static_cast<uint8_t>(0));
    EXPECT_FALSE(Node::is_series_truthy(zero_series));

    x::telem::Series one_series(static_cast<uint8_t>(1));
    EXPECT_TRUE(Node::is_series_truthy(one_series));
}

/// @brief Test that is_series_truthy works with int64 series
TEST(StateTest, IsSeriesTruthy_Int64Series) {
    x::telem::Series zero_series(static_cast<int64_t>(0));
    EXPECT_FALSE(Node::is_series_truthy(zero_series));

    x::telem::Series non_zero_series(static_cast<int64_t>(-42));
    EXPECT_TRUE(Node::is_series_truthy(non_zero_series));
}

/// @brief Test that is_series_truthy treats a non-empty string as truthy
TEST(StateTest, IsSeriesTruthy_StringSeries) {
    x::telem::Series empty_string(std::string(""));
    EXPECT_FALSE(Node::is_series_truthy(empty_string));

    x::telem::Series non_empty_string(std::string("ox_alarm"));
    EXPECT_TRUE(Node::is_series_truthy(non_empty_string));
}

/// @brief Test that a real node's string output drives is_output_truthy
TEST(StateTest, IsOutputTruthy_StringOutput) {
    arc::types::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type{.kind = arc::types::Kind::String};

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.push_back(output_param);

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(producer);
    ir.functions.push_back(fn);

    Config cfg{.ir = ir, .channels = {}};
    State s(cfg, arc::runtime::errors::noop_handler);
    auto node = ASSERT_NIL_P(s.node("producer"));

    *node.output(0) = x::telem::Series(std::string("ox_alarm"));
    EXPECT_TRUE(node.is_output_truthy(0));

    *node.output(0) = x::telem::Series(std::string(""));
    EXPECT_FALSE(node.is_output_truthy(0));
}

TEST(StateTest, SetAuthority_BufferAndFlush) {
    State s = create_minimal_state();
    s.set_authority(42, 200);
    auto changes = s.flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    ASSERT_TRUE(changes[0].channel_key.has_value());
    EXPECT_EQ(*changes[0].channel_key, 42);
    EXPECT_EQ(changes[0].authority, 200);
    EXPECT_TRUE(s.flush_authority_changes().empty());
}

TEST(StateTest, SetAuthority_GlobalAuthority) {
    State s = create_minimal_state();
    s.set_authority(std::nullopt, 150);
    auto changes = s.flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    ASSERT_FALSE(changes[0].channel_key.has_value());
    EXPECT_EQ(changes[0].authority, 150);
    EXPECT_TRUE(s.flush_authority_changes().empty());
}

TEST(StateTest, SetAuthority_MultipleChanges) {
    State s = create_minimal_state();
    s.set_authority(1, 100);
    s.set_authority(std::nullopt, 200);
    s.set_authority(2, 50);
    auto changes = s.flush_authority_changes();
    ASSERT_EQ(changes.size(), 3);
    ASSERT_TRUE(changes[0].channel_key.has_value());
    EXPECT_EQ(*changes[0].channel_key, 1);
    EXPECT_EQ(changes[0].authority, 100);
    ASSERT_FALSE(changes[1].channel_key.has_value());
    EXPECT_EQ(changes[1].authority, 200);
    ASSERT_TRUE(changes[2].channel_key.has_value());
    EXPECT_EQ(*changes[2].channel_key, 2);
    EXPECT_EQ(changes[2].authority, 50);
    EXPECT_TRUE(s.flush_authority_changes().empty());
}

TEST(StateTest, ResetClearsBufferedAuthorityChanges) {
    State s = create_minimal_state();
    s.set_authority(42, 200);
    s.set_authority(std::nullopt, 100);
    s.reset();
    EXPECT_TRUE(s.flush_authority_changes().empty());
}

/// @brief resolve_input maps an input name to its declaration-order index and
/// returns NOT_FOUND for an unknown name (so a stdlib input-name typo fails at
/// node construction, not mid-execution).
TEST(StateTest, ResolveInput_ByNameAndMissing) {
    arc::types::Param a;
    a.name = "a";
    a.type = arc::types::Type{.kind = arc::types::Kind::F32};
    a.value = 1.0f;

    arc::types::Param b;
    b.name = "b";
    b.type = arc::types::Type{.kind = arc::types::Kind::F32};
    b.value = 2.0f;

    arc::ir::Node consumer;
    consumer.key = "consumer";
    consumer.type = "consumer";
    consumer.inputs.push_back(a);
    consumer.inputs.push_back(b);

    const auto idx_a = ASSERT_NIL_P(consumer.resolve_input("a"));
    EXPECT_EQ(idx_a, 0u);
    const auto idx_b = ASSERT_NIL_P(consumer.resolve_input("b"));
    EXPECT_EQ(idx_b, 1u);
    ASSERT_OCCURRED_AS_P(consumer.resolve_input("missing"), x::errors::NOT_FOUND);
}

/// @brief builds a value param with the given name, kind, and configured value.
types::Param
value_param(std::string name, const types::Kind kind, x::json::json value = nullptr) {
    types::Param p;
    p.name = std::move(name);
    p.type = types::Type{.kind = kind};
    p.value = std::move(value);
    return p;
}

/// @brief builds a channel-typed reference param over elem.
types::Param
chan_param(std::string name, const types::Kind elem, x::json::json value = nullptr) {
    types::Param p;
    p.name = std::move(name);
    p.type = types::Type{
        .kind = types::Kind::Chan,
        .elem = x::mem::indirect<types::Type>(types::Type{.kind = elem})
    };
    p.value = std::move(value);
    return p;
}

/// @brief builds a param bound to variable node var, carrying its declared initial.
types::Param var_param(
    std::string name,
    const types::Kind elem,
    const std::string &var,
    x::json::json initial
) {
    types::Param p;
    p.name = std::move(name);
    p.type = types::Type{
        .kind = types::Kind::VarRef,
        .name = var,
        .elem = x::mem::indirect<types::Type>(types::Type{.kind = elem})
    };
    p.value = std::move(initial);
    return p;
}

/// @brief builds an IR node with the given key, type, inputs, and outputs.
ir::Node make_node(
    std::string key,
    std::string type,
    const std::vector<types::Param> &inputs = {},
    const std::vector<types::Param> &outputs = {}
) {
    ir::Node n;
    n.key = std::move(key);
    n.type = std::move(type);
    for (const auto &p: inputs)
        n.inputs.push_back(p);
    for (const auto &p: outputs)
        n.outputs.push_back(p);
    return n;
}

/// @brief writes value as the node's sole output sample, timestamped at seconds.
template<typename T>
void emit(const Node &n, const T value, const int64_t seconds) {
    *n.output(0) = x::telem::Series(std::vector<T>{value});
    *n.output_time(0) = x::telem::Series(
        x::telem::TimeStamp(seconds * x::telem::SECOND)
    );
}

/// @brief new_linked_state builds src (i32 output) -> dst (i32 input) and returns the
/// state.
std::shared_ptr<State> new_linked_state() {
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "src",
        "src",
        {},
        {value_param(ir::default_output_param, types::Kind::I32)}
    ));
    prog.nodes.push_back(make_node(
        "dst",
        "dst",
        {value_param(ir::default_input_param, types::Kind::I32)}
    ));
    prog.edges.emplace_back(
        ir::Handle("src", ir::default_output_param),
        ir::Handle("dst", ir::default_input_param)
    );
    return std::make_shared<State>(Config{.ir = prog});
}

/// @brief new_pair_state builds a and b (i32 outputs) -> target (two i32 inputs).
std::shared_ptr<State> new_pair_state() {
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "a",
        "a",
        {},
        {value_param(ir::default_output_param, types::Kind::I32)}
    ));
    prog.nodes.push_back(make_node(
        "b",
        "b",
        {},
        {value_param(ir::default_output_param, types::Kind::I32)}
    ));
    prog.nodes.push_back(make_node(
        "target",
        "target",
        {value_param(ir::lhs_input_param, types::Kind::I32),
         value_param(ir::rhs_input_param, types::Kind::I32)}
    ));
    prog.edges.emplace_back(
        ir::Handle("a", ir::default_output_param),
        ir::Handle("target", ir::lhs_input_param)
    );
    prog.edges.emplace_back(
        ir::Handle("b", ir::default_output_param),
        ir::Handle("target", ir::rhs_input_param)
    );
    return std::make_shared<State>(Config{.ir = prog});
}

/// @brief new_ref_state builds reader with a chan-typed reference input edge-fed from
/// reg's chan-typed output.
std::shared_ptr<State> new_ref_state() {
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "reg",
        "reg",
        {},
        {chan_param(ir::default_output_param, types::Kind::F32)}
    ));
    prog.nodes.push_back(make_node(
        "reader",
        "reader",
        {chan_param("channel", types::Kind::F32),
         value_param("data", types::Kind::F32, 0.0f)}
    ));
    prog.edges.emplace_back(
        ir::Handle("reg", ir::default_output_param),
        ir::Handle("reader", "channel")
    );
    return std::make_shared<State>(Config{.ir = prog});
}

/// @brief reset() should not re-arm a consumed variable register read.
TEST(StateTest, Reset_DoesNotRearmAConsumedVariableRegisterRead) {
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        {},
        {value_param(ir::default_output_param, types::Kind::I32)}
    ));
    prog.nodes.push_back(make_node(
        "reader",
        "dst",
        {value_param(ir::default_input_param, types::Kind::I32)}
    ));
    prog.edges.emplace_back(
        ir::Handle("v", ir::default_output_param),
        ir::Handle("reader", ir::default_input_param)
    );
    State s(Config{.ir = prog});
    const auto v = ASSERT_NIL_P(s.node("v"));
    auto reader = ASSERT_NIL_P(s.node("reader"));
    emit<int32_t>(v, 1, 10);
    ASSERT_TRUE(reader.refresh_inputs());
    reader.reset();
    ASSERT_FALSE(reader.refresh_inputs());
    emit<int32_t>(v, 2, 20);
    ASSERT_TRUE(reader.refresh_inputs());
}

/// @brief reset() should re-arm a self-write feeder only on Reset.
TEST(StateTest, Reset_RearmsASelfWriteFeederOnlyOnReset) {
    // A reader writing back into its source variable is a cycle the analyzer
    // rejects, so build the IR directly.
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        {value_param(ir::default_input_param, types::Kind::I32)},
        {value_param(ir::default_output_param, types::Kind::I32)}
    ));
    prog.nodes.push_back(make_node(
        "reader",
        "f",
        {value_param(ir::default_input_param, types::Kind::I32)},
        {value_param(ir::default_output_param, types::Kind::I32)}
    ));
    prog.edges.emplace_back(
        ir::Handle("v", ir::default_output_param),
        ir::Handle("reader", ir::default_input_param)
    );
    prog.edges.emplace_back(
        ir::Handle("reader", ir::default_output_param),
        ir::Handle("v", ir::default_input_param)
    );
    State s(Config{.ir = prog});
    const auto v = ASSERT_NIL_P(s.node("v"));
    auto reader = ASSERT_NIL_P(s.node("reader"));
    emit<int32_t>(v, 1, 10);
    ASSERT_TRUE(reader.refresh_inputs());
    emit<int32_t>(v, 2, 20);
    ASSERT_FALSE(reader.refresh_inputs());
    reader.reset();
    ASSERT_TRUE(reader.refresh_inputs());
}

/// @brief reset() should absorb a derived variable's pending data so only later
/// values fire.
TEST(StateTest, Reset_AbsorbsADerivedVariablesPendingData) {
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "feeder",
        "feeder",
        {},
        {value_param(ir::default_output_param, types::Kind::I32)}
    ));
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        {value_param(ir::default_input_param, types::Kind::I32)},
        {value_param(ir::default_output_param, types::Kind::I32)}
    ));
    prog.nodes.push_back(make_node(
        "reader",
        "dst",
        {value_param(ir::default_input_param, types::Kind::I32)}
    ));
    prog.edges.emplace_back(
        ir::Handle("feeder", ir::default_output_param),
        ir::Handle("v", ir::default_input_param)
    );
    prog.edges.emplace_back(
        ir::Handle("v", ir::default_output_param),
        ir::Handle("reader", ir::default_input_param)
    );
    State s(Config{.ir = prog});
    const auto v = ASSERT_NIL_P(s.node("v"));
    auto reader = ASSERT_NIL_P(s.node("reader"));
    emit<int32_t>(v, 1, 10);
    reader.reset();
    ASSERT_FALSE(reader.refresh_inputs());
    emit<int32_t>(v, 2, 20);
    ASSERT_TRUE(reader.refresh_inputs());
}

/// @brief State construction should initialize output storage with the declared data
/// types.
TEST(StateTest, New_InitializesOutputStorageWithTheDeclaredDataTypes) {
    const auto s = new_linked_state();
    const auto src = ASSERT_NIL_P(s->node("src"));
    EXPECT_EQ(src.output(0)->data_type(), x::telem::INT32_T);
    EXPECT_EQ(src.output(0)->size(), 0u);
    EXPECT_EQ(src.output_time(0)->data_type(), x::telem::TIMESTAMP_T);
}

/// @brief input and input_time should expose the aligned data and time after a
/// refresh.
TEST(StateTest, Input_ExposesTheAlignedDataAndTimeAfterARefresh) {
    const auto s = new_linked_state();
    const auto src = ASSERT_NIL_P(s->node("src"));
    auto dst = ASSERT_NIL_P(s->node("dst"));
    emit<int32_t>(src, 3, 10);
    ASSERT_TRUE(dst.refresh_inputs());
    EXPECT_EQ(dst.input(0)->at<int32_t>(0), 3);
    EXPECT_EQ(
        dst.input_time(0)->at<x::telem::TimeStamp>(0),
        x::telem::TimeStamp(10 * x::telem::SECOND)
    );
}

/// @brief init_input should ignore an out-of-range index.
TEST(StateTest, InitInput_IgnoresAnOutOfRangeIndex) {
    const auto s = new_linked_state();
    auto dst = ASSERT_NIL_P(s->node("dst"));
    EXPECT_NO_THROW(dst.init_input(
        9,
        x::mem::make_local_shared<x::telem::Series>(static_cast<int32_t>(7)),
        x::mem::make_local_shared<x::telem::Series>(
            x::telem::TimeStamp(5 * x::telem::SECOND)
        )
    ));
}

/// @brief ref_sourced should report an edge-fed reference input as sourced.
TEST(RefInputTest, ReportsAnEdgeFedReferenceInputAsSourced) {
    const auto s = new_ref_state();
    const auto reader = ASSERT_NIL_P(s->node("reader"));
    EXPECT_TRUE(reader.ref_sourced(0));
}

/// @brief ref_input should return the source series for an edge-fed reference input.
TEST(RefInputTest, ReturnsTheSourceSeriesForAnEdgeFedReferenceInput) {
    const auto s = new_ref_state();
    const auto reg = ASSERT_NIL_P(s->node("reg"));
    const auto reader = ASSERT_NIL_P(s->node("reader"));
    *reg.output(0) = x::telem::Series(std::vector<uint32_t>{7});
    EXPECT_EQ(reader.ref_input(0)->at<uint32_t>(0), 7u);
}

/// @brief ref_sourced should report an unedged reference input as unsourced.
TEST(RefInputTest, ReportsAnUnedgedReferenceInputAsUnsourced) {
    ir::IR prog;
    prog.nodes.push_back(
        make_node("reader", "reader", {chan_param("channel", types::Kind::F32)})
    );
    State s(Config{.ir = prog});
    const auto reader = ASSERT_NIL_P(s.node("reader"));
    EXPECT_FALSE(reader.ref_sourced(0));
    EXPECT_TRUE(reader.ref_input(0) == nullptr);
}

/// @brief ref_sourced and ref_input should report false for data inputs and
/// out-of-range indexes.
TEST(RefInputTest, ReportsFalseForDataInputsAndOutOfRangeIndexes) {
    const auto s = new_ref_state();
    const auto reader = ASSERT_NIL_P(s->node("reader"));
    EXPECT_FALSE(reader.ref_sourced(1));
    EXPECT_FALSE(reader.ref_sourced(static_cast<size_t>(-1)));
    EXPECT_FALSE(reader.ref_sourced(9));
    EXPECT_TRUE(reader.ref_input(1) == nullptr);
    EXPECT_TRUE(reader.ref_input(9) == nullptr);
}

/// @brief absorb_inputs should mark inputs consumed at the current source timestamp.
TEST(AbsorbInputsTest, MarksInputsConsumedAtTheCurrentSourceTimestamp) {
    const auto s = new_linked_state();
    const auto src = ASSERT_NIL_P(s->node("src"));
    auto dst = ASSERT_NIL_P(s->node("dst"));
    emit<int32_t>(src, 1, 10);
    dst.absorb_inputs();
    ASSERT_FALSE(dst.refresh_inputs());
    emit<int32_t>(src, 2, 20);
    ASSERT_TRUE(dst.refresh_inputs());
    EXPECT_EQ(dst.input(0)->at<int32_t>(0), 2);
}

/// @brief absorb_inputs should still fire for data arriving after an empty absorb.
TEST(AbsorbInputsTest, StillFiresForDataArrivingAfterAnEmptyAbsorb) {
    const auto s = new_linked_state();
    const auto src = ASSERT_NIL_P(s->node("src"));
    auto dst = ASSERT_NIL_P(s->node("dst"));
    dst.absorb_inputs();
    emit<int32_t>(src, 1, 10);
    ASSERT_TRUE(dst.refresh_inputs());
}

/// @brief absorb_inputs should mark every data input consumed.
TEST(AbsorbInputsTest, MarksEveryDataInputConsumed) {
    const auto s = new_pair_state();
    const auto a = ASSERT_NIL_P(s->node("a"));
    const auto b = ASSERT_NIL_P(s->node("b"));
    auto target = ASSERT_NIL_P(s->node("target"));
    emit<int32_t>(a, 1, 10);
    emit<int32_t>(b, 2, 20);
    target.absorb_inputs();
    ASSERT_FALSE(target.refresh_inputs());
    emit<int32_t>(a, 3, 30);
    ASSERT_TRUE(target.refresh_inputs());
}

/// @brief consume_input should return unconsumed data and mark it consumed.
TEST(ConsumeInputTest, ReturnsUnconsumedDataAndMarksItConsumed) {
    const auto s = new_linked_state();
    const auto src = ASSERT_NIL_P(s->node("src"));
    auto dst = ASSERT_NIL_P(s->node("dst"));
    emit<int32_t>(src, 1, 10);
    const auto [first, first_ok] = dst.consume_input(0);
    ASSERT_TRUE(first_ok);
    EXPECT_EQ(first->at<int32_t>(0), 1);
    EXPECT_FALSE(dst.consume_input(0).second);
    emit<int32_t>(src, 2, 20);
    const auto [second, second_ok] = dst.consume_input(0);
    ASSERT_TRUE(second_ok);
    EXPECT_EQ(second->at<int32_t>(0), 2);
}

/// @brief consume_input should leave nothing for refresh_inputs after consuming.
TEST(ConsumeInputTest, LeavesNothingForRefreshInputsAfterConsuming) {
    const auto s = new_linked_state();
    const auto src = ASSERT_NIL_P(s->node("src"));
    auto dst = ASSERT_NIL_P(s->node("dst"));
    emit<int32_t>(src, 1, 10);
    const auto [data, ok] = dst.consume_input(0);
    ASSERT_TRUE(ok);
    EXPECT_EQ(data->at<int32_t>(0), 1);
    ASSERT_FALSE(dst.refresh_inputs());
}

/// @brief consume_input should return false when the input has no data.
TEST(ConsumeInputTest, ReturnsFalseWhenTheInputHasNoData) {
    const auto s = new_linked_state();
    auto dst = ASSERT_NIL_P(s->node("dst"));
    EXPECT_FALSE(dst.consume_input(0).second);
}

/// @brief consume_input should return false for an out-of-range index.
TEST(ConsumeInputTest, ReturnsFalseForAnOutOfRangeIndex) {
    const auto s = new_linked_state();
    auto dst = ASSERT_NIL_P(s->node("dst"));
    EXPECT_FALSE(dst.consume_input(static_cast<size_t>(-1)).second);
    EXPECT_FALSE(dst.consume_input(9).second);
}

/// @brief consume_input should return false for a reference input.
TEST(ConsumeInputTest, ReturnsFalseForAReferenceInput) {
    const auto s = new_ref_state();
    const auto reg = ASSERT_NIL_P(s->node("reg"));
    auto reader = ASSERT_NIL_P(s->node("reader"));
    *reg.output(0) = x::telem::Series(std::vector<uint32_t>{7});
    EXPECT_FALSE(reader.consume_input(0).second);
}

/// @brief last_changed should return the most recently changed input and consume it.
TEST(LastChangedTest, ReturnsTheMostRecentlyChangedInputAndConsumesIt) {
    const auto s = new_pair_state();
    const auto a = ASSERT_NIL_P(s->node("a"));
    const auto b = ASSERT_NIL_P(s->node("b"));
    auto target = ASSERT_NIL_P(s->node("target"));
    emit<int32_t>(a, 1, 10);
    emit<int32_t>(b, 2, 20);
    const auto [newest, newest_ok] = target.last_changed();
    ASSERT_TRUE(newest_ok);
    EXPECT_EQ(newest->at<int32_t>(0), 2);
    const auto [older, older_ok] = target.last_changed();
    ASSERT_TRUE(older_ok);
    EXPECT_EQ(older->at<int32_t>(0), 1);
    EXPECT_FALSE(target.last_changed().second);
}

/// @brief last_changed should return false when no input has data.
TEST(LastChangedTest, ReturnsFalseWhenNoInputHasData) {
    const auto s = new_pair_state();
    auto target = ASSERT_NIL_P(s->node("target"));
    EXPECT_FALSE(target.last_changed().second);
}

/// @brief last_changed should skip reference inputs.
TEST(LastChangedTest, SkipsReferenceInputs) {
    const auto s = new_ref_state();
    const auto reg = ASSERT_NIL_P(s->node("reg"));
    auto reader = ASSERT_NIL_P(s->node("reader"));
    *reg.output(0) = x::telem::Series(std::vector<uint32_t>{7});
    *reg.output_time(0) = x::telem::Series(x::telem::TimeStamp(10 * x::telem::SECOND));
    // Only the defaulted data input is eligible; the reference never is.
    const auto [defaulted, ok] = reader.last_changed();
    ASSERT_TRUE(ok);
    EXPECT_EQ(defaulted->at<float>(0), 0.0f);
    EXPECT_FALSE(reader.last_changed().second);
}

/// @brief new_string_input_state wires consumer c with a var-bound "tag" (variable
/// node v) and a literal "plain".
std::shared_ptr<State> new_string_input_state() {
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        {},
        {value_param(ir::default_output_param, types::Kind::String)}
    ));
    prog.nodes.push_back(make_node(
        "c",
        "consumer",
        {var_param("tag", types::Kind::String, "v", "init"),
         value_param("plain", types::Kind::String, "cfg")}
    ));
    return std::make_shared<State>(Config{.ir = prog});
}

/// @brief string_input should return the referenced variable's latest value.
TEST(StringInputTest, ReturnsTheReferencedVariablesLatestValue) {
    const auto s = new_string_input_state();
    const auto v = ASSERT_NIL_P(s->node("v"));
    const auto c = ASSERT_NIL_P(s->node("c"));
    *v.output(0) = x::telem::Series(std::vector<std::string>{"first", "live"});
    EXPECT_EQ(c.string_input("tag"), "live");
}

/// @brief string_input should read the declared initial before any write.
TEST(StringInputTest, ReadsTheDeclaredInitialBeforeAnyWrite) {
    const auto s = new_string_input_state();
    const auto c = ASSERT_NIL_P(s->node("c"));
    EXPECT_EQ(c.string_input("tag"), "init");
}

/// @brief string_input should return a literal param's configured value.
TEST(StringInputTest, ReturnsALiteralParamsConfiguredValue) {
    const auto s = new_string_input_state();
    const auto c = ASSERT_NIL_P(s->node("c"));
    EXPECT_EQ(c.string_input("plain"), "cfg");
}

/// @brief string_input should return empty for an unknown input.
TEST(StringInputTest, ReturnsEmptyForAnUnknownInput) {
    const auto s = new_string_input_state();
    const auto c = ASSERT_NIL_P(s->node("c"));
    EXPECT_TRUE(c.string_input("nope").empty());
}

/// @brief new_numeric_input_state wires consumer c with a var-bound "gain" (variable
/// node v) and a literal "offset".
std::shared_ptr<State> new_numeric_input_state() {
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        {},
        {value_param(ir::default_output_param, types::Kind::U8)}
    ));
    prog.nodes.push_back(make_node(
        "c",
        "consumer",
        {var_param("gain", types::Kind::U8, "v", 5),
         value_param("offset", types::Kind::U8, 9)}
    ));
    return std::make_shared<State>(Config{.ir = prog});
}

/// @brief numeric_input should return the referenced variable's latest value.
TEST(NumericInputTest, ReturnsTheReferencedVariablesLatestValue) {
    const auto s = new_numeric_input_state();
    const auto v = ASSERT_NIL_P(s->node("v"));
    const auto c = ASSERT_NIL_P(s->node("c"));
    *v.output(0) = x::telem::Series(std::vector<uint8_t>{3, 7});
    EXPECT_EQ(c.numeric_input<uint8_t>("gain"), 7);
}

/// @brief numeric_input should read the declared initial before any write.
TEST(NumericInputTest, ReadsTheDeclaredInitialBeforeAnyWrite) {
    const auto s = new_numeric_input_state();
    const auto c = ASSERT_NIL_P(s->node("c"));
    EXPECT_EQ(c.numeric_input<uint8_t>("gain"), 5);
}

/// @brief numeric_input should return a literal param's configured value.
TEST(NumericInputTest, ReturnsALiteralParamsConfiguredValue) {
    const auto s = new_numeric_input_state();
    const auto c = ASSERT_NIL_P(s->node("c"));
    EXPECT_EQ(c.numeric_input<uint8_t>("offset"), 9);
}

/// @brief numeric_input should return zero for an unknown input.
TEST(NumericInputTest, ReturnsZeroForAnUnknownInput) {
    const auto s = new_numeric_input_state();
    const auto c = ASSERT_NIL_P(s->node("c"));
    EXPECT_EQ(c.numeric_input<uint8_t>("nope"), 0);
}

/// @brief refresh_inputs should gate a value-fed node on an edge into an undeclared
/// param.
TEST(GatingTest, GatesAValueFedNodeOnAnEdgeIntoAnUndeclaredParam) {
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "trigger",
        "trigger",
        {},
        {value_param(ir::default_output_param, types::Kind::U8)}
    ));
    prog.nodes.push_back(make_node(
        "target",
        "target",
        {value_param("value", types::Kind::I64, 5)},
        {value_param(ir::default_output_param, types::Kind::I64)}
    ));
    prog.edges.emplace_back(
        ir::Handle("trigger", ir::default_output_param),
        ir::Handle("target", ir::default_input_param)
    );
    State s(Config{.ir = prog});
    auto target = ASSERT_NIL_P(s.node("target"));
    ASSERT_FALSE(target.refresh_inputs())
        << "the gate has no data yet, so the node must not fire";
    const auto trigger = ASSERT_NIL_P(s.node("trigger"));
    emit<uint8_t>(trigger, 1, 100);
    ASSERT_TRUE(target.refresh_inputs());
    ASSERT_FALSE(target.refresh_inputs());
}

/// @brief absorb_inputs should skip reference and unsourced inputs when absorbing.
TEST(GatingTest, SkipsReferenceAndUnsourcedInputsWhenAbsorbing) {
    ir::IR prog;
    prog.nodes.push_back(make_node(
        "bind",
        "variable",
        {value_param("f0", types::Kind::U32, 7)},
        {chan_param(ir::default_output_param, types::Kind::F32)}
    ));
    prog.nodes.push_back(make_node(
        "reader",
        "on",
        {chan_param("channel", types::Kind::F32, 7)},
        {value_param(ir::default_output_param, types::Kind::F32)}
    ));
    prog.edges.emplace_back(
        ir::Handle("bind", ir::default_output_param),
        ir::Handle("reader", "channel")
    );
    prog.edges.emplace_back(
        ir::Handle("ghost", ir::default_output_param),
        ir::Handle("reader", "gate")
    );
    State s(Config{.ir = prog});
    const auto bind = ASSERT_NIL_P(s.node("bind"));
    auto reader = ASSERT_NIL_P(s.node("reader"));
    *bind.output(0) = x::telem::Series(std::vector<uint32_t>{9});
    reader.absorb_inputs();
    EXPECT_TRUE(reader.ref_sourced(0));
    EXPECT_EQ(reader.ref_input(0)->at<uint32_t>(-1), 9u);
}

/// @brief Builds a State over the given stateful variable store so tests can
/// observe what State's node-scoped operations do to it.
State create_state_with_vars(const std::shared_ptr<stl::stateful::Variables> &vars) {
    arc::ir::Node ir_node;
    ir_node.key = "test";
    ir_node.type = "test";

    arc::ir::Function fn;
    fn.key = "test";

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);
    ir.functions.push_back(fn);

    const Config cfg{.ir = ir, .channels = {}};
    return State(
        cfg,
        std::make_shared<stl::channels::State>(),
        std::make_shared<stl::strings::State>(),
        std::make_shared<stl::series::State>(),
        vars,
        arc::runtime::errors::noop_handler
    );
}

TEST(StateTest, ClearNodeDiscardsStatefulVariablesForThatNode) {
    auto vars = std::make_shared<stl::stateful::Variables>();
    State s = create_state_with_vars(vars);

    vars->set_current_node_key("node_a");
    vars->store_i32(0, 100);
    vars->set_current_node_key("node_b");
    vars->store_i32(0, 200);

    s.clear_node("node_a");

    vars->set_current_node_key("node_a");
    EXPECT_EQ(vars->load_i32(0, 0), 0);
    vars->set_current_node_key("node_b");
    EXPECT_EQ(vars->load_i32(0, 0), 200);
}

TEST(StateTest, NodeClearNodeDiscardsStatefulVariablesForThatNode) {
    auto vars = std::make_shared<stl::stateful::Variables>();
    State s = create_state_with_vars(vars);
    auto node = ASSERT_NIL_P(s.node("test"));

    vars->set_current_node_key("test");
    vars->store_f64(0, 3.5);
    EXPECT_DOUBLE_EQ(vars->load_f64(0, 0.0), 3.5);

    node.clear_node("test");
    EXPECT_DOUBLE_EQ(vars->load_f64(0, 9.5), 9.5);
}

}
