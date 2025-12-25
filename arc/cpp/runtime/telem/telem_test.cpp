// Copyright 2025 Synnax Labs, Inc.
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

    state::Config cfg{.ir = ir, .channels = {{10, telem::FLOAT32_T, 11}}};
    state::State s(cfg);

    io::Factory factory;
    auto state_node = ASSERT_NIL_P(s.node("source"));
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir_node, std::move(state_node)))
    );
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
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir_node, std::move(state_node)))
    );
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
    auto [node, create_err] = factory.create(
        node::Config(ir_node, std::move(state_node))
    );
    ASSERT_OCCURRED_AS(create_err, xerrors::NOT_FOUND);
    EXPECT_EQ(node, nullptr);
}
