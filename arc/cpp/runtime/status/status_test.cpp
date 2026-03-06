// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/status/status.h"

namespace arc::runtime::status {
namespace {
node::Context make_context() {
    return node::Context{
        .elapsed = x::telem::TimeSpan(0),
        .tolerance = x::telem::TimeSpan(0),
        .reason = node::RunReason::TimerTick,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };
}

Info make_info() {
    return Info{
        .key = "test_key",
        .name = "Test Status",
        .variant = "warning",
        .message = "Test message",
    };
}

ir::Node make_ir_node() {
    ir::Node node;
    node.key = "status";
    node.type = "set_status";
    ir::Param key_param;
    key_param.name = "status_key";
    key_param.value = std::string("test_key");
    ir::Param name_param;
    name_param.name = "name";
    name_param.value = std::string("Test Status");
    ir::Param variant_param;
    variant_param.name = "variant";
    variant_param.value = std::string("warning");
    ir::Param message_param;
    message_param.name = "message";
    message_param.value = std::string("Test message");
    node.config = ir::Params(
        std::vector<ir::Param>{key_param, name_param, variant_param, message_param}
    );
    return node;
}
}

/// @brief Test that factory handles set_status type.
TEST(SetStatusFactoryTest, HandlesSetStatusType) {
    Factory factory;
    EXPECT_TRUE(factory.handles("set_status"));
    EXPECT_FALSE(factory.handles("not_set_status"));
}

/// @brief Test that factory creates a SetStatus node.
TEST(SetStatusFactoryTest, CreatesSetStatusNode) {
    auto status_node = make_ir_node();

    ir::Function fn;
    fn.key = "test";

    ir::IR ir;
    ir.nodes.push_back(status_node);
    ir.functions.push_back(fn);

    state::State s(
        state::Config{.ir = ir, .channels = {}},
        runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));

    Factory factory;
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir.nodes[0], std::move(st)))
    );
    ASSERT_NE(node, nullptr);
}

/// @brief Test that next() calls the setter with correct info.
TEST(SetStatusTest, NextCallsSetter) {
    Info received;
    int call_count = 0;
    Setter setter = [&](const Info &info) {
        received = info;
        call_count++;
        return x::errors::NIL;
    };
    SetStatus node(make_info(), setter);
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(call_count, 1);
    EXPECT_EQ(received.key, "test_key");
    EXPECT_EQ(received.name, "Test Status");
    EXPECT_EQ(received.variant, "warning");
    EXPECT_EQ(received.message, "Test message");
}

/// @brief Test that next() calls setter on every invocation.
TEST(SetStatusTest, NextCallsSetterRepeatedly) {
    int call_count = 0;
    Setter setter = [&](const Info &) {
        call_count++;
        return x::errors::NIL;
    };
    SetStatus node(make_info(), setter);
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
    ASSERT_NIL(node.next(ctx));
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(call_count, 3);
}

/// @brief Test that next() handles setter errors gracefully.
TEST(SetStatusTest, NextHandlesSetterError) {
    Setter setter = [](const Info &) { return x::errors::Error("status set failed"); };
    SetStatus node(make_info(), setter);
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
}

/// @brief Test that is_output_truthy always returns false.
TEST(SetStatusTest, IsOutputTruthyReturnsFalse) {
    SetStatus node(make_info(), noop_setter);
    EXPECT_FALSE(node.is_output_truthy("output"));
    EXPECT_FALSE(node.is_output_truthy("anything"));
}

/// @brief Test that factory passes setter to created nodes.
TEST(SetStatusFactoryTest, PassesSetterToNode) {
    int call_count = 0;
    Setter setter = [&](const Info &) {
        call_count++;
        return x::errors::NIL;
    };

    auto status_node = make_ir_node();
    ir::Function fn;
    fn.key = "test";
    ir::IR ir;
    ir.nodes.push_back(status_node);
    ir.functions.push_back(fn);

    state::State s(
        state::Config{.ir = ir, .channels = {}},
        runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));

    Factory factory(setter);
    auto node = ASSERT_NIL_P(
        factory.create(node::Config(ir, ir.nodes[0], std::move(st)))
    );
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    EXPECT_EQ(call_count, 1);
}
}
