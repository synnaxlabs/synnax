// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/state/state.h"
#include "driver/arc/status/status.h"

namespace driver::arc::status {
namespace {
Setter noop_setter = [](x::status::Status<> &) { return x::errors::NIL; };

::arc::runtime::node::Context make_context() {
    return ::arc::runtime::node::Context{
        .elapsed = x::telem::TimeSpan(0),
        .tolerance = x::telem::TimeSpan(0),
        .reason = ::arc::runtime::node::RunReason::TimerTick,
        .mark_changed = [](const std::string &) {},
        .report_error = [](const x::errors::Error &) {},
        .activate_stage = [] {},
    };
}

x::status::Status<> make_status() {
    return x::status::Status<>{
        .key = "test_key",
        .name = "Test Status",
        .variant = "warning",
        .message = "Test message",
    };
}

::arc::ir::Node make_ir_node() {
    ::arc::ir::Node node;
    node.key = "status";
    node.type = "set_status";
    ::arc::types::Type str_type;
    str_type.kind = ::arc::types::Kind::String;
    ::arc::types::Param key_param;
    key_param.name = "status_key";
    key_param.type = str_type;
    key_param.value = std::string("test_key");
    ::arc::types::Param name_param;
    name_param.name = "name";
    name_param.type = str_type;
    name_param.value = std::string("Test Status");
    ::arc::types::Param variant_param;
    variant_param.name = "variant";
    variant_param.type = str_type;
    variant_param.value = std::string("warning");
    ::arc::types::Param message_param;
    message_param.name = "message";
    message_param.type = str_type;
    message_param.value = std::string("Test message");
    node.config = ::arc::types::Params{
        key_param,
        name_param,
        variant_param,
        message_param,
    };
    return node;
}
}

/// @brief Test that factory handles set_status type.
TEST(SetStatusFactoryTest, HandlesSetStatusType) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Factory factory(client);
    EXPECT_TRUE(factory.handles("set_status"));
    EXPECT_FALSE(factory.handles("not_set_status"));
}

/// @brief Test that factory creates a SetStatus node.
TEST(SetStatusFactoryTest, CreatesSetStatusNode) {
    auto status_node = make_ir_node();

    ::arc::ir::Function fn;
    fn.key = "test";

    ::arc::ir::IR ir;
    ir.nodes.push_back(status_node);
    ir.functions.push_back(fn);

    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));

    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Factory factory(client);
    auto node = ASSERT_NIL_P(
        factory.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    ASSERT_NE(node, nullptr);
}

/// @brief Test that next() calls the setter with correct info.
TEST(SetStatusTest, NextCallsSetter) {
    x::status::Status<> received;
    int call_count = 0;
    Setter setter = [&](x::status::Status<> &s) {
        received = s;
        call_count++;
        return x::errors::NIL;
    };
    SetStatus node(make_status(), setter);
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(call_count, 1);
    EXPECT_EQ(received.key, "test_key");
    EXPECT_EQ(received.name, "Test Status");
    EXPECT_EQ(received.variant, "warning");
    EXPECT_EQ(received.message, "Test message");
    EXPECT_NE(received.time.nanoseconds(), 0);
}

/// @brief Test that next() calls setter on every invocation.
TEST(SetStatusTest, NextCallsSetterRepeatedly) {
    int call_count = 0;
    Setter setter = [&](x::status::Status<> &) {
        call_count++;
        return x::errors::NIL;
    };
    SetStatus node(make_status(), setter);
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
    ASSERT_NIL(node.next(ctx));
    ASSERT_NIL(node.next(ctx));
    EXPECT_EQ(call_count, 3);
}

/// @brief Test that next() handles setter errors gracefully.
TEST(SetStatusTest, NextHandlesSetterError) {
    Setter setter = [](x::status::Status<> &) {
        return x::errors::Error("status set failed");
    };
    SetStatus node(make_status(), setter);
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
}

/// @brief Test that is_output_truthy always returns false.
TEST(SetStatusTest, IsOutputTruthyReturnsFalse) {
    SetStatus node(make_status(), noop_setter);
    EXPECT_FALSE(node.is_output_truthy("output"));
    EXPECT_FALSE(node.is_output_truthy("anything"));
}

/// @brief Test that factory creates nodes that set status on the cluster.
TEST(SetStatusFactoryTest, CreatedNodeSetsStatus) {
    auto status_node = make_ir_node();
    ::arc::ir::Function fn;
    fn.key = "test";
    ::arc::ir::IR ir;
    ir.nodes.push_back(status_node);
    ir.functions.push_back(fn);

    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));

    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Factory factory(client);
    auto node = ASSERT_NIL_P(
        factory.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));

    auto [retrieved, err] = client->statuses.retrieve("test_key");
    ASSERT_NIL(err);
    EXPECT_EQ(retrieved.key, "test_key");
    EXPECT_EQ(retrieved.name, "Test Status");
    EXPECT_EQ(retrieved.variant, "warning");
    EXPECT_EQ(retrieved.message, "Test message");
    EXPECT_NE(retrieved.time.nanoseconds(), 0);
}

/// @brief Test that next() updates the timestamp on each invocation.
TEST(SetStatusTest, NextUpdatesTimestamp) {
    std::vector<int64_t> timestamps;
    Setter setter = [&](x::status::Status<> &s) {
        timestamps.push_back(s.time.nanoseconds());
        return x::errors::NIL;
    };
    SetStatus node(make_status(), setter);
    auto ctx = make_context();
    ASSERT_NIL(node.next(ctx));
    ASSERT_NIL(node.next(ctx));
    ASSERT_EQ(timestamps.size(), 2);
    EXPECT_NE(timestamps[0], 0);
    EXPECT_NE(timestamps[1], 0);
    EXPECT_GE(timestamps[1], timestamps[0]);
}
}
