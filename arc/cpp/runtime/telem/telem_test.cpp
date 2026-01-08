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

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/telem/telem.h"

using namespace arc::runtime;

/// @brief Test factory creates source node for "on" type
TEST(TelemFactoryTest, CreateSourceNode) {
    arc::ir::Param output_param;
    output_param.name = arc::ir::default_output_param;
    output_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    arc::ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = uint32_t(10);
    ir_node.config.params.push_back(channel_config);

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{10, x::telem::FLOAT32_T, 11}}};
    state::State s(cfg);

    io::Factory factory;
    const auto state_node = ASSERT_NIL_P(s.node("source"));

    node::Config node_cfg{.node = ir_node, .state = state_node};
    const auto node = ASSERT_NIL_P(factory.create(node_cfg));
    EXPECT_NE(node, nullptr);
}

/// @brief Test factory creates sink node for "write" type
TEST(TelemFactoryTest, CreateSinkNode) {
    arc::ir::Param input_param;
    input_param.name = arc::ir::default_input_param;
    input_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node ir_node;
    ir_node.key = "sink";
    ir_node.type = "write";
    ir_node.inputs.params.push_back(input_param);

    arc::ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = uint32_t(10);
    ir_node.config.params.push_back(channel_config);

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {}};
    state::State s(cfg);

    io::Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("sink"));

    node::Config node_cfg{.node = ir_node, .state = state_node};
    auto node = ASSERT_NIL_P(factory.create(node_cfg));
    EXPECT_NE(node, nullptr);
}

/// @brief Test factory returns NOT_FOUND for unknown node type
TEST(TelemFactoryTest, UnknownNodeType) {
    arc::ir::Node ir_node;
    ir_node.key = "unknown";
    ir_node.type = "unknown_type";

    arc::ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = uint32_t(10);
    ir_node.config.params.push_back(channel_config);

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {}};
    state::State s(cfg);

    io::Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("unknown"));
    node::Config node_cfg{.node = ir_node, .state = state_node};
    auto [node, create_err] = factory.create(node_cfg);
    ASSERT_OCCURRED_AS(create_err, x::errors::NOT_FOUND);
    EXPECT_EQ(node, nullptr);
}

/// @brief Test factory returns error when channel config is missing
TEST(TelemFactoryTest, MissingChannelConfig) {
    arc::ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {}};
    state::State s(cfg);

    io::Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));

    node::Config node_cfg{.node = ir_node, .state = state_node};
    auto [node, create_err] = factory.create(node_cfg);
    EXPECT_TRUE(create_err);
    EXPECT_EQ(node, nullptr);
}

/// @brief Test On node reads channel data after ingestion
TEST(OnNodeTest, ReadChannelData) {
    arc::ir::Param output_param;
    output_param.name = arc::ir::default_output_param;
    output_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    arc::ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = uint32_t(10);
    ir_node.config.params.push_back(channel_config);

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{10, x::telem::FLOAT32_T, 11}}};
    state::State s(cfg);

    auto state_node = ASSERT_NIL_P(s.node("source"));

    io::Factory factory;
    node::Config node_cfg{.node = ir_node, .state = state_node};
    auto node = ASSERT_NIL_P(factory.create(node_cfg));

    x::telem::Frame frame;
    frame.channels = std::make_unique<std::vector<uint32_t>>();
    frame.series = std::make_unique<std::vector<x::telem::Series>>();
    frame.channels->push_back(10);
    frame.series->push_back(x::telem::Series(std::vector<float>{1.5f, 2.5f, 3.5f}));
    frame.channels->push_back(11);
    frame.series->push_back(
        x::telem::Series(
            std::vector<x::telem::TimeStamp>{
                x::telem::TimeStamp(100),
                x::telem::TimeStamp(101),
                x::telem::TimeStamp(102)
            }
        )
    );
    s.ingest(std::move(frame));

    bool output_changed = false;
    node::Context ctx{
        .elapsed = x::telem::TimeSpan(0),
        .mark_changed = [&](const std::string &) { output_changed = true; },
        .report_error = [](const x::errors::Error &) {}
    };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(output_changed);
}

/// @brief Test On node handles channel without index (generates synthetic timestamps)
TEST(OnNodeTest, ChannelWithoutIndex) {
    arc::ir::Param output_param;
    output_param.name = arc::ir::default_output_param;
    output_param.type = arc::types::Type(arc::types::Kind::I32);

    arc::ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    arc::ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = uint32_t(20);
    ir_node.config.params.push_back(channel_config);

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{20, x::telem::INT32_T, 0}}};
    state::State s(cfg);

    auto state_node = ASSERT_NIL_P(s.node("source"));

    io::Factory factory;
    node::Config node_cfg{.node = ir_node, .state = state_node};
    auto node = ASSERT_NIL_P(factory.create(node_cfg));

    x::telem::Frame frame;
    frame.channels = std::make_unique<std::vector<uint32_t>>();
    frame.series = std::make_unique<std::vector<x::telem::Series>>();
    frame.channels->push_back(20);
    frame.series->push_back(x::telem::Series(std::vector<int32_t>{100, 200}));
    s.ingest(std::move(frame));

    bool output_changed = false;
    node::Context ctx{
        .elapsed = x::telem::TimeSpan(0),
        .mark_changed = [&](const std::string &) { output_changed = true; },
        .report_error = [](const x::errors::Error &) {}
    };
    ASSERT_NIL(node->next(ctx));
    EXPECT_TRUE(output_changed);
}

/// @brief Test On node does not trigger on empty channel
TEST(OnNodeTest, EmptyChannel) {
    arc::ir::Param output_param;
    output_param.name = arc::ir::default_output_param;
    output_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    arc::ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = uint32_t(999);
    ir_node.config.params.push_back(channel_config);

    arc::ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{999, x::telem::FLOAT32_T, 0}}};
    state::State s(cfg);

    auto state_node = ASSERT_NIL_P(s.node("source"));

    io::Factory factory;
    node::Config node_cfg{.node = ir_node, .state = state_node};
    auto node = ASSERT_NIL_P(factory.create(node_cfg));

    bool output_changed = false;
    node::Context ctx{
        .elapsed = x::telem::TimeSpan(0),
        .mark_changed = [&](const std::string &) { output_changed = true; },
        .report_error = [](const x::errors::Error &) {}
    };
    ASSERT_NIL(node->next(ctx));
    EXPECT_FALSE(output_changed);
}

/// @brief Test Write node writes channel data when input available
TEST(WriteNodeTest, WriteChannelData) {
    arc::ir::Param output_param;
    output_param.name = arc::ir::default_output_param;
    output_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Param input_param;
    input_param.name = arc::ir::default_input_param;
    input_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node producer;
    producer.key = "producer";
    producer.type = "producer";
    producer.outputs.params.push_back(output_param);

    arc::ir::Node sink;
    sink.key = "sink";
    sink.type = "write";
    sink.inputs.params.push_back(input_param);

    arc::ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = uint32_t(100);
    sink.config.params.push_back(channel_config);

    arc::ir::Edge edge(
        arc::ir::Handle("producer", arc::ir::default_output_param),
        arc::ir::Handle("sink", arc::ir::default_input_param)
    );

    arc::ir::IR ir;
    ir.nodes.push_back(producer);
    ir.nodes.push_back(sink);
    ir.edges.push_back(edge);

    state::Config cfg{.ir = ir, .channels = {{100, x::telem::FLOAT32_T, 101}}};
    state::State s(cfg);

    auto producer_node = ASSERT_NIL_P(s.node("producer"));
    auto sink_node = ASSERT_NIL_P(s.node("sink"));

    auto &output = producer_node.output(0);
    output->resize(2);
    output->set(0, 7.7f);
    output->set(1, 8.8f);

    auto &output_time = producer_node.output_time(0);
    output_time->resize(2);
    output_time->set(0, x::telem::TimeStamp(500).nanoseconds());
    output_time->set(1, x::telem::TimeStamp(501).nanoseconds());

    io::Factory factory;
    node::Config node_cfg{.node = sink, .state = sink_node};
    auto node = ASSERT_NIL_P(factory.create(node_cfg));

    node::Context ctx{
        .elapsed = x::telem::TimeSpan(0),
        .mark_changed = [](const std::string &) {},
        .report_error = [](const x::errors::Error &) {}
    };
    ASSERT_NIL(node->next(ctx));

    auto writes = s.flush_writes();
    EXPECT_FALSE(writes.empty());
}

/// @brief Test Write node respects refresh_inputs guard
TEST(WriteNodeTest, RefreshInputsGuard) {
    arc::ir::Param input_param;
    input_param.name = arc::ir::default_input_param;
    input_param.type = arc::types::Type(arc::types::Kind::F32);

    arc::ir::Node sink;
    sink.key = "sink";
    sink.type = "write";
    sink.inputs.params.push_back(input_param);

    arc::ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = uint32_t(100);
    sink.config.params.push_back(channel_config);

    arc::ir::IR ir;
    ir.nodes.push_back(sink);

    state::Config cfg{.ir = ir, .channels = {{100, x::telem::FLOAT32_T, 101}}};
    state::State s(cfg);

    auto sink_node = ASSERT_NIL_P(s.node("sink"));

    io::Factory factory;
    node::Config node_cfg{.node = sink, .state = sink_node};
    auto node = ASSERT_NIL_P(factory.create(node_cfg));

    node::Context ctx{
        .elapsed = x::telem::TimeSpan(0),
        .mark_changed = [](const std::string &) {},
        .report_error = [](const x::errors::Error &) {}
    };
    ASSERT_NIL(node->next(ctx));

    auto writes = s.flush_writes();
    EXPECT_TRUE(writes.empty());
}

/// @brief Integration test: source to sink flow
TEST(TelemIntegrationTest, SourceToSinkFlow) {
    arc::ir::Param output_param;
    output_param.name = arc::ir::default_output_param;
    output_param.type = arc::types::Type(arc::types::Kind::I32);

    arc::ir::Param input_param;
    input_param.name = arc::ir::default_input_param;
    input_param.type = arc::types::Type(arc::types::Kind::I32);

    arc::ir::Node source;
    source.key = "read";
    source.type = "on";
    source.outputs.params.push_back(output_param);

    arc::ir::Param source_channel;
    source_channel.name = "channel";
    source_channel.value = uint32_t(1);
    source.config.params.push_back(source_channel);

    arc::ir::Node sink;
    sink.key = "write";
    sink.type = "write";
    sink.inputs.params.push_back(input_param);

    arc::ir::Param sink_channel;
    sink_channel.name = "channel";
    sink_channel.value = uint32_t(3);
    sink.config.params.push_back(sink_channel);

    arc::ir::Edge edge(
        arc::ir::Handle("read", arc::ir::default_output_param),
        arc::ir::Handle("write", arc::ir::default_input_param)
    );

    arc::ir::IR ir;
    ir.nodes.push_back(source);
    ir.nodes.push_back(sink);
    ir.edges.push_back(edge);

    state::Config cfg{
        .ir = ir,
        .channels = {{1, x::telem::INT32_T, 2}, {3, x::telem::INT32_T, 4}}
    };
    state::State s(cfg);

    io::Factory factory;

    auto source_state = ASSERT_NIL_P(s.node("read"));
    node::Config source_cfg{.node = source, .state = source_state};
    auto source_node = ASSERT_NIL_P(factory.create(source_cfg));

    auto sink_state = ASSERT_NIL_P(s.node("write"));
    node::Config sink_cfg{.node = sink, .state = sink_state};
    auto sink_node = ASSERT_NIL_P(factory.create(sink_cfg));

    x::telem::Frame frame;
    frame.channels = std::make_unique<std::vector<uint32_t>>();
    frame.series = std::make_unique<std::vector<x::telem::Series>>();
    frame.channels->push_back(1);
    frame.series->push_back(x::telem::Series(std::vector<int32_t>{42, 99}));
    frame.channels->push_back(2);
    frame.series->push_back(
        x::telem::Series(
            std::vector<x::telem::TimeStamp>{x::telem::TimeStamp(10), x::telem::TimeStamp(20)}
        )
    );
    s.ingest(std::move(frame));

    node::Context ctx{
        .elapsed = x::telem::TimeSpan(0),
        .mark_changed = [](const std::string &) {},
        .report_error = [](const x::errors::Error &) {}
    };

    ASSERT_NIL(source_node->next(ctx));

    EXPECT_TRUE(sink_state.refresh_inputs());
    ASSERT_NIL(sink_node->next(ctx));

    auto writes = s.flush_writes();
    EXPECT_FALSE(writes.empty());
}
