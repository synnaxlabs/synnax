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
#include "arc/cpp/stl/channel/channel.h"

namespace arc::stl::channel {
runtime::node::Context make_context(bool *changed = nullptr) {
    return runtime::node::Context{
        .elapsed = ::x::telem::SECOND,
        .mark_changed =
            [changed](size_t) {
                if (changed) *changed = true;
            },
        .report_error = [](const x::errors::Error &) {},
    };
}

TEST(ChannelModuleTest, CreateSourceNode) {
    types::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type.kind = types::Kind::F32;

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.push_back(output_param);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {{10, ::x::telem::FLOAT32_T, 11}}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node)))
    );
    EXPECT_NE(node, nullptr);
}

TEST(ChannelModuleTest, CreateSinkNode) {
    types::Param input_param;
    input_param.name = ir::default_input_param;
    input_param.type.kind = types::Kind::F32;

    ir::Node ir_node;
    ir_node.key = "sink";
    ir_node.type = "write";
    ir_node.inputs.push_back(input_param);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("sink"));
    auto node = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node)))
    );
    EXPECT_NE(node, nullptr);
}

TEST(ChannelModuleTest, ReturnsErrorForNullChannelParam) {
    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = nullptr;
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("source"));
    ASSERT_OCCURRED_AS_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node))),
        x::errors::VALIDATION
    );
}

TEST(ChannelModuleTest, UnknownNodeType) {
    ir::Node ir_node;
    ir_node.key = "unknown";
    ir_node.type = "unknown_type";

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("unknown"));
    auto [node, create_err] = module.create(
        runtime::node::Config(ir, ir_node, std::move(state_node))
    );
    ASSERT_OCCURRED_AS(create_err, x::errors::NOT_FOUND);
    EXPECT_EQ(node, nullptr);
}

TEST(ChannelModuleTest, HandlesOnAndWrite) {
    channel::WasmModule module(nullptr, nullptr);
    EXPECT_TRUE(module.handles("on"));
    EXPECT_TRUE(module.handles("write"));
    EXPECT_FALSE(module.handles("unknown"));
    EXPECT_FALSE(module.handles("constant"));
}

TEST(OnTest, NextReadsChannelData) {
    types::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type.kind = types::Kind::F32;

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.push_back(output_param);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {{10, ::x::telem::FLOAT32_T, 11}}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node)))
    );

    ::x::telem::Frame frame(2);
    auto data = ::x::telem::Series(std::vector<float>{1.5f, 2.5f, 3.5f});
    auto time = ::x::telem::Series(std::vector<int64_t>{100, 101, 102});
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

TEST(OnTest, NextHandlesChannelWithoutIndex) {
    types::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type.kind = types::Kind::I32;

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.push_back(output_param);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(20);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {{20, ::x::telem::INT32_T, 0}}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node)))
    );

    ::x::telem::Frame frame(1);
    auto data = ::x::telem::Series(std::vector<int32_t>{100, 200});
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
    EXPECT_EQ(output_time->data_type(), ::x::telem::TIMESTAMP_T);
    EXPECT_EQ(output_time->size(), 2);
}

TEST(OnTest, NextReturnsEarlyOnEmptyChannel) {
    types::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type.kind = types::Kind::F32;

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.push_back(output_param);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(999);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {{999, ::x::telem::FLOAT32_T, 0}}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node)))
    );

    bool changed = false;
    auto ctx = make_context(&changed);
    ASSERT_NIL(node->next(ctx));

    EXPECT_FALSE(changed);
}

TEST(OnTest, NextHandlesMultipleSeries) {
    types::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type.kind = types::Kind::F32;

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.push_back(output_param);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {{10, ::x::telem::FLOAT32_T, 11}}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node)))
    );

    ::x::telem::Frame frame1(2);
    auto d1 = ::x::telem::Series(1.0f);
    d1.alignment = ::x::telem::Alignment(0);
    auto t1 = ::x::telem::Series(static_cast<int64_t>(10));
    t1.alignment = ::x::telem::Alignment(0);
    frame1.emplace(10, std::move(d1));
    frame1.emplace(11, std::move(t1));
    s.ingest(frame1);

    ::x::telem::Frame frame2(2);
    auto d2 = ::x::telem::Series(2.0f);
    d2.alignment = ::x::telem::Alignment(1);
    auto t2 = ::x::telem::Series(static_cast<int64_t>(20));
    t2.alignment = ::x::telem::Alignment(1);
    frame2.emplace(10, std::move(d2));
    frame2.emplace(11, std::move(t2));
    s.ingest(frame2);

    int call_count = 0;
    auto ctx = runtime::node::Context{
        .elapsed = ::x::telem::SECOND,
        .mark_changed = [&call_count](size_t) { call_count++; },
        .report_error = [](const x::errors::Error &) {},
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

TEST(OnTest, NextSkipsOnIndexCountMismatch) {
    types::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type.kind = types::Kind::F32;

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.push_back(output_param);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {{10, ::x::telem::FLOAT32_T, 11}}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node)))
    );

    ::x::telem::Frame frame1(2);
    auto d1 = ::x::telem::Series(1.0f);
    auto t1 = ::x::telem::Series(static_cast<int64_t>(10));
    frame1.emplace(10, std::move(d1));
    frame1.emplace(11, std::move(t1));
    s.ingest(frame1);

    ::x::telem::Frame frame2(1);
    auto d2 = ::x::telem::Series(2.0f);
    frame2.emplace(10, std::move(d2));
    s.ingest(frame2);

    int call_count = 0;
    auto ctx = runtime::node::Context{
        .elapsed = ::x::telem::SECOND,
        .mark_changed = [&call_count](size_t) { call_count++; },
        .report_error = [](const x::errors::Error &) {},
    };

    ASSERT_NIL(node->next(ctx));
    EXPECT_EQ(call_count, 1);
}

TEST(OnTest, NextSkipsOnAlignmentMismatch) {
    types::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type.kind = types::Kind::F64;

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.push_back(output_param);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(30);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {{30, ::x::telem::FLOAT64_T, 31}}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node)))
    );

    ::x::telem::Frame frame(2);
    auto data = ::x::telem::Series(std::vector<double>{1.0, 2.0});
    data.alignment = ::x::telem::Alignment(100);
    auto time = ::x::telem::Series(std::vector<int64_t>{10, 20});
    time.alignment = ::x::telem::Alignment(200);
    frame.emplace(30, std::move(data));
    frame.emplace(31, std::move(time));
    s.ingest(frame);

    int call_count = 0;
    auto ctx = runtime::node::Context{
        .elapsed = ::x::telem::SECOND,
        .mark_changed = [&call_count](size_t) { call_count++; },
        .report_error = [](const x::errors::Error &) {},
    };

    ASSERT_NIL(node->next(ctx));
    EXPECT_EQ(call_count, 0);
}

TEST(OnTest, NextCallsMarkChanged) {
    types::Param output_param;
    output_param.name = ir::default_output_param;
    output_param.type.kind = types::Kind::F32;

    ir::Node ir_node;
    ir_node.key = "source";
    ir_node.type = "on";
    ir_node.outputs.push_back(output_param);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(10);
    ir_node.config.push_back(channel_config);

    ir::IR ir;
    ir.nodes.push_back(ir_node);

    runtime::state::Config cfg{.ir = ir, .channels = {{10, ::x::telem::FLOAT32_T, 11}}};
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, ir_node, std::move(state_node)))
    );

    ::x::telem::Frame frame(2);
    auto data = ::x::telem::Series(1.0f);
    auto time = ::x::telem::Series(static_cast<int64_t>(10));
    frame.emplace(10, std::move(data));
    frame.emplace(11, std::move(time));
    s.ingest(frame);

    std::vector<size_t> marked;
    auto ctx = runtime::node::Context{
        .elapsed = ::x::telem::SECOND,
        .mark_changed = [&](size_t i) { marked.push_back(i); },
        .report_error = [](const x::errors::Error &) {},
    };

    ASSERT_NIL(node->next(ctx));
    ASSERT_EQ(marked.size(), 1);
    EXPECT_EQ(marked[0], 0);
}

TEST(WriteTest, NextWritesDataWhenInputAvailable) {
    types::Param upstream_output;
    upstream_output.name = ir::default_output_param;
    upstream_output.type.kind = types::Kind::F32;

    ir::Node upstream_node;
    upstream_node.key = "upstream";
    upstream_node.type = "producer";
    upstream_node.outputs.push_back(upstream_output);

    types::Param sink_input;
    sink_input.name = ir::default_input_param;
    sink_input.type.kind = types::Kind::F32;

    types::Param sink_output;
    sink_output.name = ir::default_output_param;
    sink_output.type.kind = types::Kind::U8;

    ir::Node sink_node;
    sink_node.key = "sink";
    sink_node.type = "write";
    sink_node.inputs.push_back(sink_input);
    sink_node.outputs.push_back(sink_output);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(100);
    sink_node.config.push_back(channel_config);

    ir::Edge edge;
    edge.source = ir::Handle("upstream", ir::default_output_param);
    edge.target = ir::Handle("sink", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(upstream_node);
    ir.nodes.push_back(sink_node);
    ir.edges.push_back(edge);

    runtime::state::Config cfg{
        .ir = ir,
        .channels = {{100, ::x::telem::FLOAT32_T, 101}}
    };
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto sink_state = ASSERT_NIL_P(s.node("sink"));
    auto sink = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, sink_node, std::move(sink_state)))
    );

    auto upstream = ASSERT_NIL_P(s.node("upstream"));
    auto input_data = x::mem::make_local_shared<::x::telem::Series>(
        std::vector<float>{7.7f, 8.8f}
    );
    input_data->alignment = ::x::telem::Alignment(42);
    input_data->time_range = ::x::telem::TimeRange(500, 501);
    upstream.output(0) = input_data;
    upstream.output_time(0) = x::mem::make_local_shared<::x::telem::Series>(
        std::vector<int64_t>{500, 501}
    );

    auto sink_checker = ASSERT_NIL_P(s.node("sink"));
    EXPECT_TRUE(sink_checker.refresh_inputs());

    bool changed = false;
    runtime::node::Context ctx{.mark_changed = [&](size_t) { changed = true; }};
    ASSERT_NIL(sink->next(ctx));
    EXPECT_TRUE(changed);

    auto out_checker = ASSERT_NIL_P(s.node("sink"));
    const auto &out_data = out_checker.output(0);
    EXPECT_EQ(out_data->size(), 1);
    EXPECT_EQ(out_data->at<uint8_t>(0), 1);
    EXPECT_EQ(out_data->alignment, ::x::telem::Alignment(42));
    EXPECT_EQ(out_data->time_range.start, 500);

    const auto &out_time = out_checker.output_time(0);
    EXPECT_EQ(out_time->size(), 1);
    EXPECT_GT(out_time->at<int64_t>(0), 0);
    EXPECT_EQ(out_time->alignment, ::x::telem::Alignment(42));

    x::telem::Frame out;
    s.flush_into(out);
    EXPECT_FALSE(out.empty());
    ASSERT_TRUE(out.contains(100));
    EXPECT_FLOAT_EQ(out.at<float>(100, 0), 7.7f);
    EXPECT_FLOAT_EQ(out.at<float>(100, 1), 8.8f);
}

TEST(WriteTest, NextRespectsRefreshInputsGuard) {
    types::Param upstream_output;
    upstream_output.name = ir::default_output_param;
    upstream_output.type.kind = types::Kind::F32;

    ir::Node upstream_node;
    upstream_node.key = "upstream";
    upstream_node.type = "producer";
    upstream_node.outputs.push_back(upstream_output);

    types::Param sink_input;
    sink_input.name = ir::default_input_param;
    sink_input.type.kind = types::Kind::F32;

    types::Param sink_output;
    sink_output.name = ir::default_output_param;
    sink_output.type.kind = types::Kind::U8;

    ir::Node sink_node;
    sink_node.key = "sink";
    sink_node.type = "write";
    sink_node.inputs.push_back(sink_input);
    sink_node.outputs.push_back(sink_output);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(100);
    sink_node.config.push_back(channel_config);

    ir::Edge edge;
    edge.source = ir::Handle("upstream", ir::default_output_param);
    edge.target = ir::Handle("sink", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(upstream_node);
    ir.nodes.push_back(sink_node);
    ir.edges.push_back(edge);

    runtime::state::Config cfg{
        .ir = ir,
        .channels = {{100, ::x::telem::FLOAT32_T, 101}}
    };
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto sink_state = ASSERT_NIL_P(s.node("sink"));
    auto sink = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, sink_node, std::move(sink_state)))
    );

    auto ctx = make_context();
    ASSERT_NIL(sink->next(ctx));

    x::telem::Frame out;
    s.flush_into(out);
    EXPECT_TRUE(out.empty());
}

TEST(WriteTest, NextSkipsEmptyInput) {
    types::Param upstream_output;
    upstream_output.name = ir::default_output_param;
    upstream_output.type.kind = types::Kind::F32;

    ir::Node upstream_node;
    upstream_node.key = "upstream";
    upstream_node.type = "producer";
    upstream_node.outputs.push_back(upstream_output);

    types::Param sink_input;
    sink_input.name = ir::default_input_param;
    sink_input.type.kind = types::Kind::F32;

    types::Param sink_output;
    sink_output.name = ir::default_output_param;
    sink_output.type.kind = types::Kind::U8;

    ir::Node sink_node;
    sink_node.key = "sink";
    sink_node.type = "write";
    sink_node.inputs.push_back(sink_input);
    sink_node.outputs.push_back(sink_output);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(100);
    sink_node.config.push_back(channel_config);

    ir::Edge edge;
    edge.source = ir::Handle("upstream", ir::default_output_param);
    edge.target = ir::Handle("sink", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(upstream_node);
    ir.nodes.push_back(sink_node);
    ir.edges.push_back(edge);

    runtime::state::Config cfg{
        .ir = ir,
        .channels = {{100, ::x::telem::FLOAT32_T, 101}}
    };
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto sink_state = ASSERT_NIL_P(s.node("sink"));
    auto sink = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, sink_node, std::move(sink_state)))
    );

    auto upstream = ASSERT_NIL_P(s.node("upstream"));
    upstream.output(0) = x::mem::make_local_shared<::x::telem::Series>(
        std::vector<float>{}
    );
    upstream.output_time(0) = x::mem::make_local_shared<::x::telem::Series>(
        std::vector<int64_t>{}
    );

    auto sink_checker = ASSERT_NIL_P(s.node("sink"));
    EXPECT_FALSE(sink_checker.refresh_inputs());

    auto ctx = make_context();
    ASSERT_NIL(sink->next(ctx));

    x::telem::Frame out;
    s.flush_into(out);
    EXPECT_TRUE(out.empty());
}

TEST(WriteTest, NextHandlesSequentialWrites) {
    types::Param upstream_output;
    upstream_output.name = ir::default_output_param;
    upstream_output.type.kind = types::Kind::F32;

    ir::Node upstream_node;
    upstream_node.key = "upstream";
    upstream_node.type = "producer";
    upstream_node.outputs.push_back(upstream_output);

    types::Param sink_input;
    sink_input.name = ir::default_input_param;
    sink_input.type.kind = types::Kind::F32;

    types::Param sink_output;
    sink_output.name = ir::default_output_param;
    sink_output.type.kind = types::Kind::U8;

    ir::Node sink_node;
    sink_node.key = "sink";
    sink_node.type = "write";
    sink_node.inputs.push_back(sink_input);
    sink_node.outputs.push_back(sink_output);

    types::Param channel_config;
    channel_config.name = "channel";
    channel_config.type.kind = types::Kind::U32;
    channel_config.value = static_cast<uint32_t>(100);
    sink_node.config.push_back(channel_config);

    ir::Edge edge;
    edge.source = ir::Handle("upstream", ir::default_output_param);
    edge.target = ir::Handle("sink", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(upstream_node);
    ir.nodes.push_back(sink_node);
    ir.edges.push_back(edge);

    runtime::state::Config cfg{
        .ir = ir,
        .channels = {{100, ::x::telem::FLOAT32_T, 101}}
    };
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);
    auto sink_state = ASSERT_NIL_P(s.node("sink"));
    auto sink = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, sink_node, std::move(sink_state)))
    );

    auto ctx = make_context();

    auto upstream1 = ASSERT_NIL_P(s.node("upstream"));
    upstream1.output(0) = x::mem::make_local_shared<::x::telem::Series>(
        std::vector<float>{1.0f}
    );
    upstream1.output_time(0) = x::mem::make_local_shared<::x::telem::Series>(
        std::vector<int64_t>{10}
    );

    auto sink_checker1 = ASSERT_NIL_P(s.node("sink"));
    EXPECT_TRUE(sink_checker1.refresh_inputs());
    ASSERT_NIL(sink->next(ctx));

    x::telem::Frame out1;
    s.flush_into(out1);
    EXPECT_FALSE(out1.empty());
    ASSERT_TRUE(out1.contains(100));
    EXPECT_FLOAT_EQ(out1.at<float>(100, 0), 1.0f);

    auto upstream2 = ASSERT_NIL_P(s.node("upstream"));
    upstream2.output(0) = x::mem::make_local_shared<::x::telem::Series>(
        std::vector<float>{2.0f}
    );
    upstream2.output_time(0) = x::mem::make_local_shared<::x::telem::Series>(
        std::vector<int64_t>{20}
    );

    auto sink_checker2 = ASSERT_NIL_P(s.node("sink"));
    EXPECT_TRUE(sink_checker2.refresh_inputs());
    ASSERT_NIL(sink->next(ctx));

    x::telem::Frame out2;
    s.flush_into(out2);
    EXPECT_FALSE(out2.empty());
    ASSERT_TRUE(out2.contains(100));
    EXPECT_FLOAT_EQ(out2.at<float>(100, 0), 2.0f);
}

TEST(IntegrationTest, SourceToSinkFlow) {
    types::Param read_output;
    read_output.name = ir::default_output_param;
    read_output.type.kind = types::Kind::I32;

    ir::Node read_node;
    read_node.key = "read";
    read_node.type = "on";
    read_node.outputs.push_back(read_output);

    types::Param read_channel;
    read_channel.name = "channel";
    read_channel.type.kind = types::Kind::U32;
    read_channel.value = static_cast<uint32_t>(1);
    read_node.config.push_back(read_channel);

    types::Param write_input;
    write_input.name = ir::default_input_param;
    write_input.type.kind = types::Kind::I32;

    types::Param write_output;
    write_output.name = ir::default_output_param;
    write_output.type.kind = types::Kind::U8;

    ir::Node write_node;
    write_node.key = "write";
    write_node.type = "write";
    write_node.inputs.push_back(write_input);
    write_node.outputs.push_back(write_output);

    types::Param write_channel;
    write_channel.name = "channel";
    write_channel.type.kind = types::Kind::U32;
    write_channel.value = static_cast<uint32_t>(3);
    write_node.config.push_back(write_channel);

    ir::Edge edge;
    edge.source = ir::Handle("read", ir::default_output_param);
    edge.target = ir::Handle("write", ir::default_input_param);

    ir::IR ir;
    ir.nodes.push_back(read_node);
    ir.nodes.push_back(write_node);
    ir.edges.push_back(edge);

    runtime::state::Config cfg{
        .ir = ir,
        .channels = {{1, ::x::telem::INT32_T, 2}, {3, ::x::telem::INT32_T, 4}}
    };
    runtime::state::State s(cfg, runtime::errors::noop_handler);

    channel::WasmModule module(nullptr, nullptr);

    auto read_state = ASSERT_NIL_P(s.node("read"));
    auto source = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, read_node, std::move(read_state)))
    );

    auto write_state = ASSERT_NIL_P(s.node("write"));
    auto sink = ASSERT_NIL_P(
        module.create(runtime::node::Config(ir, write_node, std::move(write_state)))
    );

    ::x::telem::Frame ingest_frame(2);
    auto data = ::x::telem::Series(std::vector<int32_t>{42, 99});
    auto time = ::x::telem::Series(std::vector<int64_t>{10, 20});
    ingest_frame.emplace(1, std::move(data));
    ingest_frame.emplace(2, std::move(time));
    s.ingest(ingest_frame);

    auto ctx = make_context();
    ASSERT_NIL(source->next(ctx));

    auto write_checker = ASSERT_NIL_P(s.node("write"));
    EXPECT_TRUE(write_checker.refresh_inputs());

    ASSERT_NIL(sink->next(ctx));

    x::telem::Frame out;
    s.flush_into(out);
    EXPECT_FALSE(out.empty());
    ASSERT_TRUE(out.contains(3));
    EXPECT_EQ(out.at<int32_t>(3, 0), 42);
    EXPECT_EQ(out.at<int32_t>(3, 1), 99);
}

TEST(ChannelStateTest, WriteValue_AccumulatesSameChannelIntoSingleSeries) {
    State channel_state(
        std::vector<Digest>{{.key = 1, .data_type = ::x::telem::FLOAT32_T, .index = 2}}
    );

    auto data1 = ::x::mem::make_local_shared<::x::telem::Series>(1.0f);
    auto time1 = ::x::mem::make_local_shared<::x::telem::Series>(
        ::x::telem::TimeStamp(1 * ::x::telem::SECOND)
    );
    channel_state.write_value(1, data1, time1);

    auto data2 = ::x::mem::make_local_shared<::x::telem::Series>(2.0f);
    auto time2 = ::x::mem::make_local_shared<::x::telem::Series>(
        ::x::telem::TimeStamp(2 * ::x::telem::SECOND)
    );
    channel_state.write_value(1, data2, time2);

    x::telem::Frame out;
    channel_state.flush_into(out);

    ASSERT_TRUE(out.contains(1));
    EXPECT_EQ(out.at<float>(1, 0), 1.0f);
    EXPECT_EQ(out.at<float>(1, 1), 2.0f);

    ASSERT_TRUE(out.contains(2));
}

TEST(ChannelStateTest, WriteChannelTyped_IndexedWritesTimestamp) {
    State channel_state(
        std::vector<Digest>{{.key = 5, .data_type = ::x::telem::INT32_T, .index = 6}}
    );

    channel_state.write_channel_i32(5, 10);
    channel_state.write_channel_i32(5, 20);

    x::telem::Frame out;
    channel_state.flush_into(out);

    ASSERT_TRUE(out.contains(5));
    EXPECT_EQ(out.at<int32_t>(5, 0), 10);
    EXPECT_EQ(out.at<int32_t>(5, 1), 20);

    ASSERT_TRUE(out.contains(6));
}

TEST(ChannelStateTest, WriteChannelTyped_NoIndexWritesOnlyData) {
    State channel_state(
        std::vector<Digest>{{.key = 7, .data_type = ::x::telem::FLOAT64_T, .index = 0}}
    );

    channel_state.write_channel_f64(7, 1.5);
    channel_state.write_channel_f64(7, 2.5);

    x::telem::Frame out;
    channel_state.flush_into(out);
    ASSERT_EQ(out.size(), 1);
    ASSERT_TRUE(out.contains(7));
    EXPECT_DOUBLE_EQ(out.at<double>(7, 0), 1.5);
    EXPECT_DOUBLE_EQ(out.at<double>(7, 1), 2.5);
}

TEST(ChannelStateTest, WriteValue_MultipleWritesSameKeyPreservedInFlush) {
    State channel_state;

    auto data1 = ::x::mem::make_local_shared<::x::telem::Series>(1.0f);
    auto time1 = ::x::mem::make_local_shared<::x::telem::Series>(
        ::x::telem::TimeStamp(::x::telem::MICROSECOND)
    );
    auto data2 = ::x::mem::make_local_shared<::x::telem::Series>(2.0f);
    auto time2 = ::x::mem::make_local_shared<::x::telem::Series>(
        ::x::telem::TimeStamp(::x::telem::MICROSECOND * 2)
    );
    channel_state.write_value(10, data1, time1);
    channel_state.write_value(10, data2, time2);

    x::telem::Frame out;
    channel_state.flush_into(out);
    ASSERT_EQ(out.size(), 1);
    ASSERT_TRUE(out.contains(10));
    EXPECT_EQ(out.at<float>(10, 0), 1.0f);
    EXPECT_EQ(out.at<float>(10, 1), 2.0f);
}

TEST(ChannelStateTest, ReadSeries_ReturnsDataAndTimeForIndexedChannel) {
    State channel_state(
        std::vector<Digest>{{.key = 1, .data_type = ::x::telem::FLOAT32_T, .index = 2}}
    );

    auto data = ::x::telem::Series(std::vector<float>{1.0f, 2.0f});
    auto time = ::x::telem::Series(std::vector<int64_t>{100, 200});
    channel_state.ingest(::x::telem::Frame(1, std::move(data)));
    channel_state.ingest(::x::telem::Frame(2, std::move(time)));

    auto [result_data, result_time, ok] = channel_state.read_series(1);
    ASSERT_TRUE(ok);
    ASSERT_EQ(result_data.series.size(), 1);
    EXPECT_EQ(result_data.series[0].at<float>(0), 1.0f);
    EXPECT_EQ(result_data.series[0].at<float>(1), 2.0f);
    ASSERT_EQ(result_time.series.size(), 1);
    EXPECT_EQ(result_time.series[0].at<int64_t>(0), 100);
    EXPECT_EQ(result_time.series[0].at<int64_t>(1), 200);
}

TEST(ChannelStateTest, ReadSeries_ReturnsDataOnlyForNonIndexedChannel) {
    State channel_state(
        std::vector<Digest>{{.key = 1, .data_type = ::x::telem::FLOAT32_T, .index = 0}}
    );

    auto data = ::x::telem::Series(std::vector<float>{3.0f, 4.0f});
    channel_state.ingest(::x::telem::Frame(1, std::move(data)));

    auto [result_data, result_time, ok] = channel_state.read_series(1);
    ASSERT_TRUE(ok);
    ASSERT_EQ(result_data.series.size(), 1);
    EXPECT_EQ(result_data.series[0].at<float>(0), 3.0f);
    EXPECT_EQ(result_data.series[0].at<float>(1), 4.0f);
    EXPECT_TRUE(result_time.series.empty());
}

TEST(ChannelStateTest, ReadSeries_ReturnsFalseForUnknownChannel) {
    State channel_state;
    auto [result_data, result_time, ok] = channel_state.read_series(99);
    ASSERT_FALSE(ok);
    EXPECT_TRUE(result_data.series.empty());
    EXPECT_TRUE(result_time.series.empty());
}

TEST(ChannelStateTest, ReadSeries_ReturnsFalseWhenTimeMissing) {
    State channel_state(
        std::vector<Digest>{{.key = 1, .data_type = ::x::telem::FLOAT32_T, .index = 2}}
    );

    auto data = ::x::telem::Series(std::vector<float>{1.0f});
    channel_state.ingest(::x::telem::Frame(1, std::move(data)));

    auto [result_data, result_time, ok] = channel_state.read_series(1);
    ASSERT_FALSE(ok);
}

TEST(ChannelStateTest, WriteSeries_RoundTripsViaReadSeries) {
    State channel_state(
        std::vector<Digest>{{.key = 1, .data_type = ::x::telem::FLOAT32_T, .index = 2}}
    );

    auto data = ::x::mem::make_local_shared<::x::telem::Series>(
        std::vector<float>{5.0f, 6.0f}
    );
    auto time = ::x::mem::make_local_shared<::x::telem::Series>(
        std::vector<int64_t>{300, 400}
    );
    channel_state.write_series(1, data, time);

    x::telem::Frame out;
    channel_state.flush_into(out);
    ASSERT_TRUE(out.contains(1));
    EXPECT_EQ(out.at<float>(1, 0), 5.0f);
    EXPECT_EQ(out.at<float>(1, 1), 6.0f);
    ASSERT_TRUE(out.contains(2));
    EXPECT_EQ(out.at<int64_t>(2, 0), 300);
    EXPECT_EQ(out.at<int64_t>(2, 1), 400);
}

}
