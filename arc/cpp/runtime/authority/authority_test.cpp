// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/authority/authority.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/state/state.h"

using namespace arc::runtime;

namespace {
struct TestSetup {
    arc::ir::IR ir;
    std::shared_ptr<state::State> state;

    TestSetup(const uint8_t auth_value, const uint32_t channel):
        ir(build_ir(auth_value, channel)),
        state(std::make_shared<state::State>(
            state::Config{.ir = ir, .channels = {}},
            arc::runtime::errors::noop_handler
        )) {}

    state::Node make_node() {
        return ASSERT_NIL_P(this->state->node("set_auth"));
    }

private:
    static arc::ir::IR build_ir(const uint8_t auth_value, const uint32_t channel) {
        arc::ir::Param authority_param;
        authority_param.name = "value";
        authority_param.type = arc::types::Type(arc::types::Kind::U8);
        authority_param.value = auth_value;

        arc::ir::Param channel_param;
        channel_param.name = "channel";
        channel_param.type = arc::types::Type(arc::types::Kind::U32);
        channel_param.value = channel;

        arc::ir::Node ir_node;
        ir_node.key = "set_auth";
        ir_node.type = "set_authority";
        ir_node.config.params.push_back(authority_param);
        ir_node.config.params.push_back(channel_param);

        arc::ir::Function fn;
        fn.key = "test";

        arc::ir::IR ir;
        ir.nodes.push_back(ir_node);
        ir.functions.push_back(fn);
        return ir;
    }
};

node::Context make_context() {
    return node::Context{
        .elapsed = telem::SECOND,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const xerrors::Error &) {},
        .activate_stage = [] {},
    };
}
}

TEST(SetAuthorityFactoryTest, ReturnsNotFoundForWrongType) {
    TestSetup setup(100, 42);
    auto ir_node = setup.ir.nodes[0];
    ir_node.type = "not_set_authority";

    authority::Factory factory(setup.state);
    ASSERT_OCCURRED_AS_P(
        factory.create(node::Config(setup.ir, ir_node, setup.make_node())),
        xerrors::NOT_FOUND
    );
}

TEST(SetAuthorityFactoryTest, CreatesNode) {
    TestSetup setup(100, 42);
    authority::Factory factory(setup.state);
    auto node = ASSERT_NIL_P(
        factory.create(
            node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
        )
    );
    ASSERT_NE(node, nullptr);
}

TEST(SetAuthorityTest, NextBuffersChannelAuthorityChange) {
    TestSetup setup(200, 42);
    authority::Factory factory(setup.state);
    auto node = ASSERT_NIL_P(
        factory.create(
            node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
        )
    );

    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto changes = setup.state->flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    ASSERT_TRUE(changes[0].channel_key.has_value());
    EXPECT_EQ(*changes[0].channel_key, 42);
    EXPECT_EQ(changes[0].authority, 200);
}

TEST(SetAuthorityTest, NextBuffersGlobalAuthorityChange) {
    TestSetup setup(150, 0);
    authority::Factory factory(setup.state);
    auto node = ASSERT_NIL_P(
        factory.create(
            node::Config(setup.ir, setup.ir.nodes[0], setup.make_node())
        )
    );

    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto changes = setup.state->flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    ASSERT_FALSE(changes[0].channel_key.has_value());
    EXPECT_EQ(changes[0].authority, 150);
}
