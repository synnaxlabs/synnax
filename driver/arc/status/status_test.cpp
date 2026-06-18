// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <functional>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/state/state.h"
#include "driver/arc/status/status.h"

namespace driver::arc::status {
namespace {

::arc::runtime::node::Context make_context() {
    return ::arc::runtime::node::Context{
        .elapsed = x::telem::TimeSpan(0),
        .tolerance = x::telem::TimeSpan(0),
        .reason = ::arc::runtime::node::RunReason::TimerTick,
        .mark_changed = [](size_t) {},
        .mark_self_changed = [] {},
        .set_deadline = [](x::telem::TimeSpan) {},
        .report_error = [](const x::errors::Error &) {},
    };
}

// make_set_ir_node builds a `set` IR node with the {key_or_name, message, variant}
// config.
::arc::ir::Node make_set_ir_node(
    const std::string &key_or_name,
    const std::string &message,
    const std::string &variant
) {
    ::arc::ir::Node node;
    node.key = "status";
    node.type = "set";
    ::arc::types::Type str_type;
    str_type.kind = ::arc::types::Kind::String;
    ::arc::types::Param key_param;
    key_param.name = "key_or_name";
    key_param.type = str_type;
    key_param.value = key_or_name;
    ::arc::types::Param message_param;
    message_param.name = "message";
    message_param.type = str_type;
    message_param.value = message;
    ::arc::types::Param variant_param;
    variant_param.name = "variant";
    variant_param.type = str_type;
    variant_param.value = variant;
    node.inputs = ::arc::types::Params{key_param, message_param, variant_param};
    ::arc::types::Type out_type;
    out_type.kind = ::arc::types::Kind::String;
    ::arc::types::Param out;
    out.name = "output";
    out.type = out_type;
    node.outputs = ::arc::types::Params{out};
    return node;
}

// recordingReporter pushes (variant, message) pairs into the provided vector.
Reporter recordingReporter(std::vector<std::pair<std::string, std::string>> *out) {
    return
        [out](const std::string &v, const std::string &m) { out->emplace_back(v, m); };
}

Reporter noopReporter() {
    return [](const std::string &, const std::string &) {};
}

// Unique per test to avoid cross-run contamination on a shared cluster.
std::string unique_name(const std::string &prefix) {
    return prefix + std::to_string(
                        static_cast<unsigned>(x::telem::TimeStamp::now().nanoseconds())
                    );
}

// build_ir wraps a node in a minimal ir::IR so state::State can be built.
::arc::ir::IR build_ir(::arc::ir::Node node) {
    ::arc::ir::IR ir;
    ir.nodes.push_back(std::move(node));
    ::arc::ir::Function fn;
    fn.key = "fn";
    ir.functions.push_back(fn);
    return ir;
}

} // namespace

TEST(StatusModuleTest, HandlesSet) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Module module(client, noopReporter());
    EXPECT_TRUE(module.handles("set"));
    EXPECT_FALSE(module.handles("delete"));
    EXPECT_FALSE(module.handles("set_status"));
    EXPECT_FALSE(module.handles("anything_else"));
}

TEST(StatusModuleTest, ModuleNameIsStatus) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Module module(client, noopReporter());
    EXPECT_EQ(module.module_name(), "status");
}

TEST(StatusModuleTest, CreatesSetNodeFromBareType) {
    auto node = make_set_ir_node("test_key", "msg", "warning");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Module module(client, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    ASSERT_NE(created, nullptr);
    EXPECT_NE(dynamic_cast<SetStatus *>(created.get()), nullptr);
}

TEST(StatusModuleTest, CreatesNodeWithQualifiedTypeViaMultiFactory) {
    auto node = make_set_ir_node("test_key", "msg", "warning");
    node.type = "status.set";
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto module = std::make_shared<Module>(client, noopReporter());
    ::arc::runtime::node::MultiFactory multi({module});
    auto created = ASSERT_NIL_P(
        multi.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    ASSERT_NE(created, nullptr);
    EXPECT_NE(dynamic_cast<SetStatus *>(created.get()), nullptr);
}

TEST(StatusModuleTest, ReturnsNotFoundForUnknownType) {
    auto node = make_set_ir_node("test_key", "msg", "warning");
    node.type = "not_set_or_delete";
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Module module(client, noopReporter());
    auto [created, err] = module.create(
        ::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st))
    );
    EXPECT_EQ(created, nullptr);
    EXPECT_EQ(err, x::errors::NOT_FOUND);
}

TEST(SetStatusTest, NextWritesResolvedKeyToOutput) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto name = unique_name("set_next_");

    // Pre-create so we know the resolved UUID; configure the node with that
    // UUID so we can retrieve the row by key after next().
    const auto preset_key = ASSERT_NIL_P(client->statuses.set_by_key_or_name(
                                             name,
                                             "initial",
                                             "info"
                                         ))
                                .key;

    auto node = make_set_ir_node(preset_key, "the message", "info");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    Module module(client, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));

    auto [retrieved, err] = client->statuses.retrieve(preset_key);
    ASSERT_NIL(err);
    EXPECT_EQ(retrieved.message, "the message");
    EXPECT_EQ(retrieved.variant, "info");
}

TEST(SetStatusTest, NextRepeatedCallsKeepWriting) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto name = unique_name("set_repeat_");

    const auto preset_key = ASSERT_NIL_P(client->statuses.set_by_key_or_name(
                                             name,
                                             "initial",
                                             "info"
                                         ))
                                .key;

    auto node = make_set_ir_node(preset_key, "msg", "info");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    Module module(client, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));
    ASSERT_NIL(created->next(ctx));
    ASSERT_NIL(created->next(ctx));
    auto [retrieved, err] = client->statuses.retrieve(preset_key);
    ASSERT_NIL(err);
    EXPECT_EQ(retrieved.message, "msg");
}

TEST(SetStatusTest, NextWarnsOnInvalidVariant) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto name = unique_name("set_iv_");

    std::vector<std::pair<std::string, std::string>> calls;
    auto node = make_set_ir_node(name, "msg", "bogus");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    Module module(client, recordingReporter(&calls));
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].first, synnax::status::VARIANT_WARNING);
    EXPECT_NE(calls[0].second.find("status.set:"), std::string::npos);
}

TEST(SetStatusTest, IsOutputTruthyDoesNotThrow) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto node = make_set_ir_node("truthy", "msg", "info");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    Module module(client, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    EXPECT_NO_THROW({ (void) created->is_output_truthy(0); });
}

}
