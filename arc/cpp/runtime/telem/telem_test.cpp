// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/mem/local_shared.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/telem/telem.h"

namespace arc::runtime::io {
namespace {
node::Context make_context(bool *changed = nullptr) {
    return node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed =
            [changed](const std::string &) {
                if (changed) *changed = true;
            },
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };
}
}

/// @brief Test factory creates source node for "on" type.
TEST(TelemFactoryTest, CreateSourceNode) {
    ir::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type = types::Type(types::Kind::F32);

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{10, x::telem::FLOAT32_T, 11}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir_node, std::move(state_node)))
    );
    EXPECT_NE(node, nullptr);
}

/// @brief Test factory creates sink node for "write" type.
TEST(TelemFactoryTest, CreateSinkNode) {
    ir::Param input_param;
    input_param.name = ir::default_input_param;
    input_param.type = types::Type(types::Kind::F32);

    ir::Node ir_node;
    ir_node.key = "sink";
    ir_node.type = "write";
    ir_node.inputs.params.push_back(input_param);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("sink"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir_node, std::move(state_node)))
    );
    EXPECT_NE(node, nullptr);
}

/// @brief Test factory returns NOT_FOUND for unknown node type.
TEST(TelemFactoryTest, UnknownNodeType) {
    ir::Node ir_node;
    ir_node.key = "unknown";
    ir_node.type = "unknown_type";

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("unknown"));
    auto [node, create_err] = factory.create(
        node::Config(ir, ir_node, std::move(state_node))
    );
    ASSERT_OCCURRED_AS(create_err, x::errors::NOT_FOUND);
    EXPECT_EQ(node, nullptr);
}

/// @brief Test factory handles() returns true for "on" and "write" types.
TEST(TelemFactoryTest, HandlesOnAndWrite) {
    Factory factory;
    EXPECT_TRUE(factory.handles("on"));
    EXPECT_TRUE(factory.handles("write"));
    EXPECT_FALSE(factory.handles("unknown"));
    EXPECT_FALSE(factory.handles("constant"));
}

/// @brief Test source node reads channel data after ingestion.
TEST(OnTest, NextReadsChannelData) {
    ir::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type = types::Type(types::Kind::F32);

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{10, x::telem::FLOAT32_T, 11}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir_node, std::move(state_node)))
    );

    x::telem::Frame frame(2);
    auto data = x::telem::Series(std::vector<float>{1.5f, 2.5f, 3.5f});
    auto time = x::telem::Series(std::vector<int64_t>{100, 101, 102});
    frame.emplace(10, std::move(data));
    frame.emplace(11, std::move(time));
    s.ingest(frame);

    bool changed = false;
    auto ctx = make_context(&changed);
    ASSERT_NIL(node->next(ctx));

    EXPECT_TRUE(changed);
    auto checker = ASSERT_NIL_P(s.node("source"));
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 3);
    EXPECT_FLOAT_EQ(output->at<float>(0), 1.5f);
    EXPECT_FLOAT_EQ(output->at<float>(1), 2.5f);
    EXPECT_FLOAT_EQ(output->at<float>(2), 3.5f);

    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->size(), 3);
    EXPECT_EQ(output_time->at<int64_t>(0), 100);
    EXPECT_EQ(output_time->at<int64_t>(1), 101);
    EXPECT_EQ(output_time->at<int64_t>(2), 102);
}

/// @brief Test source node generates synthetic timestamps when no index channel.
TEST(OnTest, NextHandlesChannelWithoutIndex) {
    ir::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type = types::Type(types::Kind::I32);

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(20);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{20, x::telem::INT32_T, 0}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir_node, std::move(state_node)))
    );

    x::telem::Frame frame(1);
    auto data = x::telem::Series(std::vector<int32_t>{100, 200});
    frame.emplace(20, std::move(data));
    s.ingest(frame);

    bool changed = false;
    auto ctx = make_context(&changed);
    ASSERT_NIL(node->next(ctx));

    EXPECT_TRUE(changed);
    auto checker = ASSERT_NIL_P(s.node("source"));
    const auto &output = checker.output(0);
    EXPECT_EQ(output->size(), 2);
    EXPECT_EQ(output->at<int32_t>(0), 100);
    EXPECT_EQ(output->at<int32_t>(1), 200);

    const auto &output_time = checker.output_time(0);
    EXPECT_EQ(output_time->data_type(), x::telem::TIMESTAMP_T);
    EXPECT_EQ(output_time->size(), 2);
}

/// @brief Test source node returns early when no data available.
TEST(OnTest, NextReturnsEarlyOnEmptyChannel) {
    ir::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type = types::Type(types::Kind::F32);

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(999);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{999, x::telem::FLOAT32_T, 0}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir_node, std::move(state_node)))
    );

    bool changed = false;
    auto ctx = make_context(&changed);
    ASSERT_NIL(node->next(ctx));

    EXPECT_FALSE(changed);
}

/// @brief Test source node handles multiple series with high water mark.
TEST(OnTest, NextHandlesMultipleSeries) {
    ir::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type = types::Type(types::Kind::F32);

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{10, x::telem::FLOAT32_T, 11}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir_node, std::move(state_node)))
    );

    x::telem::Frame frame1(2);
    auto d1 = x::telem::Series(1.0f);
    d1.alignment = x::telem::Alignment(0);
    auto t1 = x::telem::Series(static_cast<int64_t>(10));
    t1.alignment = x::telem::Alignment(0);
    frame1.emplace(10, std::move(d1));
    frame1.emplace(11, std::move(t1));
    s.ingest(frame1);

    x::telem::Frame frame2(2);
    auto d2 = x::telem::Series(2.0f);
    d2.alignment = x::telem::Alignment(1);
    auto t2 = x::telem::Series(static_cast<int64_t>(20));
    t2.alignment = x::telem::Alignment(1);
    frame2.emplace(10, std::move(d2));
    frame2.emplace(11, std::move(t2));
    s.ingest(frame2);

    int call_count = 0;
    auto ctx = node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [&call_count](const std::string &) { call_count++; },
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };

    ASSERT_NIL(node->next(ctx));
    EXPECT_EQ(call_count, 1);
    auto checker1 = ASSERT_NIL_P(s.node("source"));
    EXPECT_FLOAT_EQ(checker1.output(0)->at<float>(0), 1.0f);

    ASSERT_NIL(node->next(ctx));
    EXPECT_EQ(call_count, 2);
    auto checker2 = ASSERT_NIL_P(s.node("source"));
    EXPECT_FLOAT_EQ(checker2.output(0)->at<float>(0), 2.0f);
}

/// @brief Test source node skips data when index series count mismatches.
TEST(OnTest, NextSkipsOnIndexCountMismatch) {
    ir::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type = types::Type(types::Kind::F32);

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{10, x::telem::FLOAT32_T, 11}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir_node, std::move(state_node)))
    );

    x::telem::Frame frame1(2);
    auto d1 = x::telem::Series(1.0f);
    auto t1 = x::telem::Series(static_cast<int64_t>(10));
    frame1.emplace(10, std::move(d1));
    frame1.emplace(11, std::move(t1));
    s.ingest(frame1);

    x::telem::Frame frame2(1);
    auto d2 = x::telem::Series(2.0f);
    frame2.emplace(10, std::move(d2));
    s.ingest(frame2);

    int call_count = 0;
    auto ctx = node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [&call_count](const std::string &) { call_count++; },
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };

    ASSERT_NIL(node->next(ctx));
    EXPECT_EQ(call_count, 1);
}

/// @brief Test source node skips data when alignment mismatches.
TEST(OnTest, NextSkipsOnAlignmentMismatch) {
    ir::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type = types::Type(types::Kind::F64);

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(30);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{30, x::telem::FLOAT64_T, 31}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir_node, std::move(state_node)))
    );

    x::telem::Frame frame(2);
    auto data = x::telem::Series(std::vector<double>{1.0, 2.0});
    data.alignment = x::telem::Alignment(100);
    auto time = x::telem::Series(std::vector<int64_t>{10, 20});
    time.alignment = x::telem::Alignment(200);
    frame.emplace(30, std::move(data));
    frame.emplace(31, std::move(time));
    s.ingest(frame);

    int call_count = 0;
    auto ctx = node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [&call_count](const std::string &) { call_count++; },
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };

    ASSERT_NIL(node->next(ctx));
    EXPECT_EQ(call_count, 0);
}

/// @brief Test source node calls mark_changed callback.
TEST(OnTest, NextCallsMarkChanged) {
    ir::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type = types::Type(types::Kind::F32);

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.params.push_back(output_param);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.params.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    state::Config cfg{.ir = ir, .channels = {{10, x::telem::FLOAT32_T, 11}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir_node, std::move(state_node)))
    );

    x::telem::Frame frame(2);
    auto data = x::telem::Series(1.0f);
    auto time = x::telem::Series(static_cast<int64_t>(10));
    frame.emplace(10, std::move(data));
    frame.emplace(11, std::move(time));
    s.ingest(frame);

    std::string changed_param;
    auto ctx = node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [&changed_param](const std::string &p) { changed_param = p; },
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };

    ASSERT_NIL(node->next(ctx));
    EXPECT_EQ(changed_param, ir::default_output_param);
}

/// @brief Test sink node writes data when input is available.
TEST(WriteTest, NextWritesDataWhenInputAvailable) {
    ir::Param upstream_output;
    upstream_output.name = ir::default_output_param;
    upstream_output.type = types::Type(types::Kind::F32);

    ir::Node upstream_node;
    upstream_node.key = "upstream";
    upstream_node.type = "producer";
    upstream_node.outputs.params.push_back(upstream_output);

    ir::Param sink_input;
    sink_input.name = ir::default_input_param;
    sink_input.type = types::Type(types::Kind::F32);

    ir::Node sink_node;
    sink_node.key = "sink";
    sink_node.type = "write";
    sink_node.inputs.params.push_back(sink_input);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(100);
    sink_node.config.params.push_back(channel_config);

    ir::Edge edge;
    edge.source = ir::Handle("upstream", ir::default_output_param);
    edge.target = ir::Handle("sink", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(upstream_node);
    ir.nodes.push_back(sink_node);
    ir.edges.push_back(edge);

    state::Config cfg{.ir = ir, .channels = {{100, x::telem::FLOAT32_T, 101}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto sink_state = ASSERT_NIL_P(s.node("sink"));
    auto sink = ASSERT_NIL_P(
        factory.create(node::Config(ir, sink_node, std::move(sink_state)))
    );

    auto upstream = ASSERT_NIL_P(s.node("upstream"));
    upstream.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<float>{7.7f, 8.8f}
    );
    upstream.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{500, 501}
    );

    auto sink_checker = ASSERT_NIL_P(s.node("sink"));
    EXPECT_TRUE(sink_checker.refresh_inputs());

    auto ctx = make_context();
    ASSERT_NIL(sink->next(ctx));

    auto writes = s.flush();
    EXPECT_FALSE(writes.empty());
    bool found = false;
    for (const auto &[key, data]: writes) {
        if (key == 100) {
            found = true;
            EXPECT_EQ(data->size(), 2);
            EXPECT_FLOAT_EQ(data->at<float>(0), 7.7f);
            EXPECT_FLOAT_EQ(data->at<float>(1), 8.8f);
        }
    }
    EXPECT_TRUE(found);
}

/// @brief Test sink node respects RefreshInputs guard.
TEST(WriteTest, NextRespectsRefreshInputsGuard) {
    ir::Param upstream_output;
    upstream_output.name = ir::default_output_param;
    upstream_output.type = types::Type(types::Kind::F32);

    ir::Node upstream_node;
    upstream_node.key = "upstream";
    upstream_node.type = "producer";
    upstream_node.outputs.params.push_back(upstream_output);

    ir::Param sink_input;
    sink_input.name = ir::default_input_param;
    sink_input.type = types::Type(types::Kind::F32);

    ir::Node sink_node;
    sink_node.key = "sink";
    sink_node.type = "write";
    sink_node.inputs.params.push_back(sink_input);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(100);
    sink_node.config.params.push_back(channel_config);

    ir::Edge edge;
    edge.source = ir::Handle("upstream", ir::default_output_param);
    edge.target = ir::Handle("sink", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(upstream_node);
    ir.nodes.push_back(sink_node);
    ir.edges.push_back(edge);

    state::Config cfg{.ir = ir, .channels = {{100, x::telem::FLOAT32_T, 101}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto sink_state = ASSERT_NIL_P(s.node("sink"));
    auto sink = ASSERT_NIL_P(
        factory.create(node::Config(ir, sink_node, std::move(sink_state)))
    );

    auto ctx = make_context();
    ASSERT_NIL(sink->next(ctx));

    auto writes = s.flush();
    EXPECT_TRUE(writes.empty());
}

/// @brief Test sink node skips empty input.
TEST(WriteTest, NextSkipsEmptyInput) {
    ir::Param upstream_output;
    upstream_output.name = ir::default_output_param;
    upstream_output.type = types::Type(types::Kind::F32);

    ir::Node upstream_node;
    upstream_node.key = "upstream";
    upstream_node.type = "producer";
    upstream_node.outputs.params.push_back(upstream_output);

    ir::Param sink_input;
    sink_input.name = ir::default_input_param;
    sink_input.type = types::Type(types::Kind::F32);

    ir::Node sink_node;
    sink_node.key = "sink";
    sink_node.type = "write";
    sink_node.inputs.params.push_back(sink_input);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(100);
    sink_node.config.params.push_back(channel_config);

    ir::Edge edge;
    edge.source = ir::Handle("upstream", ir::default_output_param);
    edge.target = ir::Handle("sink", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(upstream_node);
    ir.nodes.push_back(sink_node);
    ir.edges.push_back(edge);

    state::Config cfg{.ir = ir, .channels = {{100, x::telem::FLOAT32_T, 101}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto sink_state = ASSERT_NIL_P(s.node("sink"));
    auto sink = ASSERT_NIL_P(
        factory.create(node::Config(ir, sink_node, std::move(sink_state)))
    );

    auto upstream = ASSERT_NIL_P(s.node("upstream"));
    upstream.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<float>{}
    );
    upstream.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{}
    );

    auto sink_checker = ASSERT_NIL_P(s.node("sink"));
    EXPECT_FALSE(sink_checker.refresh_inputs());

    auto ctx = make_context();
    ASSERT_NIL(sink->next(ctx));

    auto writes = s.flush();
    EXPECT_TRUE(writes.empty());
}

/// @brief Test sink node handles sequential writes.
TEST(WriteTest, NextHandlesSequentialWrites) {
    ir::Param upstream_output;
    upstream_output.name = ir::default_output_param;
    upstream_output.type = types::Type(types::Kind::F32);

    ir::Node upstream_node;
    upstream_node.key = "upstream";
    upstream_node.type = "producer";
    upstream_node.outputs.params.push_back(upstream_output);

    ir::Param sink_input;
    sink_input.name = ir::default_input_param;
    sink_input.type = types::Type(types::Kind::F32);

    ir::Node sink_node;
    sink_node.key = "sink";
    sink_node.type = "write";
    sink_node.inputs.params.push_back(sink_input);

    ir::Param channel_config;
    channel_config.name = "channel";
    channel_config.value = static_cast<uint32_t>(100);
    sink_node.config.params.push_back(channel_config);

    ir::Edge edge;
    edge.source = ir::Handle("upstream", ir::default_output_param);
    edge.target = ir::Handle("sink", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(upstream_node);
    ir.nodes.push_back(sink_node);
    ir.edges.push_back(edge);

    state::Config cfg{.ir = ir, .channels = {{100, x::telem::FLOAT32_T, 101}}};
    state::State s(cfg, errors::noop_handler);

    Factory factory;
    auto sink_state = ASSERT_NIL_P(s.node("sink"));
    auto sink = ASSERT_NIL_P(
        factory.create(node::Config(ir, sink_node, std::move(sink_state)))
    );

    auto ctx = make_context();

    auto upstream1 = ASSERT_NIL_P(s.node("upstream"));
    upstream1.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<float>{1.0f}
    );
    upstream1.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{10}
    );

    auto sink_checker1 = ASSERT_NIL_P(s.node("sink"));
    EXPECT_TRUE(sink_checker1.refresh_inputs());
    ASSERT_NIL(sink->next(ctx));

    auto writes1 = s.flush();
    EXPECT_FALSE(writes1.empty());
    for (const auto &[key, data]: writes1) {
        if (key == 100) { EXPECT_FLOAT_EQ(data->at<float>(0), 1.0f); }
    }

    auto upstream2 = ASSERT_NIL_P(s.node("upstream"));
    upstream2.output(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<float>{2.0f}
    );
    upstream2.output_time(0) = x::mem::make_local_shared<x::telem::Series>(
        std::vector<int64_t>{20}
    );

    auto sink_checker2 = ASSERT_NIL_P(s.node("sink"));
    EXPECT_TRUE(sink_checker2.refresh_inputs());
    ASSERT_NIL(sink->next(ctx));

    auto writes2 = s.flush();
    EXPECT_FALSE(writes2.empty());
    for (const auto &[key, data]: writes2) {
        if (key == 100) { EXPECT_FLOAT_EQ(data->at<float>(0), 2.0f); }
    }
}

/// @brief Test end-to-end flow from source through sink.
TEST(IntegrationTest, SourceToSinkFlow) {
    ir::Param read_output;
    read_output.name = ir::default_output_param;
    read_output.type = types::Type(types::Kind::I32);

    ir::Node read_node;
    read_node.key = "read";
    read_node.type = "on";
    read_node.outputs.params.push_back(read_output);

    ir::Param read_channel;
    read_channel.name = "channel";
    read_channel.value = static_cast<uint32_t>(1);
    read_node.config.params.push_back(read_channel);

    ir::Param write_input;
    write_input.name = ir::default_input_param;
    write_input.type = types::Type(types::Kind::I32);

    ir::Node write_node;
    write_node.key = "write";
    write_node.type = "write";
    write_node.inputs.params.push_back(write_input);

    ir::Param write_channel;
    write_channel.name = "channel";
    write_channel.value = static_cast<uint32_t>(3);
    write_node.config.params.push_back(write_channel);

    ir::Edge edge;
    edge.source = ir::Handle("read", ir::default_output_param);
    edge.target = ir::Handle("write", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(read_node);
    ir.nodes.push_back(write_node);
    ir.edges.push_back(edge);

    state::Config cfg{
        .ir = ir,
        .channels = {{1, x::telem::INT32_T, 2}, {3, x::telem::INT32_T, 4}}
    };
    state::State s(cfg, errors::noop_handler);

    Factory factory;

    auto read_state = ASSERT_NIL_P(s.node("read"));
    auto source = ASSERT_NIL_P(
        factory.create(node::Config(ir, read_node, std::move(read_state)))
    );

    auto write_state = ASSERT_NIL_P(s.node("write"));
    auto sink = ASSERT_NIL_P(
        factory.create(node::Config(ir, write_node, std::move(write_state)))
    );

    x::telem::Frame ingest_frame(2);
    auto data = x::telem::Series(std::vector<int32_t>{42, 99});
    auto time = x::telem::Series(std::vector<int64_t>{10, 20});
    ingest_frame.emplace(1, std::move(data));
    ingest_frame.emplace(2, std::move(time));
    s.ingest(ingest_frame);

    auto ctx = make_context();
    ASSERT_NIL(source->next(ctx));

    auto write_checker = ASSERT_NIL_P(s.node("write"));
    EXPECT_TRUE(write_checker.refresh_inputs());

    ASSERT_NIL(sink->next(ctx));

    auto writes = s.flush();
    EXPECT_FALSE(writes.empty());
    bool found_data = false;
    for (const auto &[key, series]: writes) {
        if (key == 3) {
            found_data = true;
            EXPECT_EQ(series->size(), 2);
            EXPECT_EQ(series->at<int32_t>(0), 42);
            EXPECT_EQ(series->at<int32_t>(1), 99);
        }
    }
    EXPECT_TRUE(found_data);
}
}
