// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>
#include <variant>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/state/state.h"
#include "driver/arc/ranges/ranges.h"

namespace driver::arc::ranges {
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

::arc::types::Param str_param(const std::string &name, const std::string &value) {
    ::arc::types::Param p;
    p.name = name;
    p.type = ::arc::types::Type{.kind = ::arc::types::Kind::String};
    p.value = value;
    return p;
}

// make_create_ir_node builds a `create` IR node with {name, color, parent}.
::arc::ir::Node make_create_ir_node(
    const std::string &name,
    const std::string &color,
    const std::string &parent
) {
    ::arc::ir::Node node;
    node.key = "ranges";
    node.type = "create";
    node.inputs = ::arc::types::Params{
        str_param("name", name),
        str_param("color", color),
        str_param("parent", parent)
    };
    ::arc::types::Param out;
    out.name = "output";
    out.type = ::arc::types::Type{.kind = ::arc::types::Kind::String};
    node.outputs = ::arc::types::Params{out};
    return node;
}

// make_end_ir_node builds an `end` IR node with {key, time}. time is an i64 value
// (the runtime treats NOW_SENTINEL as "now").
::arc::ir::Node make_end_ir_node(const std::string &key, int64_t time) {
    ::arc::ir::Node node;
    node.key = "ranges";
    node.type = "end";
    ::arc::types::Param key_param = str_param("key", key);
    ::arc::types::Param time_param;
    time_param.name = "time";
    time_param.type = ::arc::types::Type{.kind = ::arc::types::Kind::I64};
    time_param.value = time;
    node.inputs = ::arc::types::Params{key_param, time_param};
    ::arc::types::Param out;
    out.name = "output";
    out.type = ::arc::types::Type{.kind = ::arc::types::Kind::String};
    node.outputs = ::arc::types::Params{out};
    return node;
}

Reporter recordingReporter(std::vector<std::pair<std::string, std::string>> *out) {
    return
        [out](const std::string &v, const std::string &m) { out->emplace_back(v, m); };
}

Reporter noopReporter() {
    return [](const std::string &, const std::string &) {};
}

std::string unique_name(const std::string &prefix) {
    return prefix + std::to_string(
                        static_cast<unsigned>(x::telem::TimeStamp::now().nanoseconds())
                    );
}

::arc::ir::IR build_ir(::arc::ir::Node node) {
    ::arc::ir::IR ir;
    ir.nodes.push_back(std::move(node));
    ::arc::ir::Function fn;
    fn.key = "fn";
    ir.functions.push_back(fn);
    return ir;
}

// reads the string written to a node's first output via a fresh state handle, which
// shares the same output buffers as the executed node.
std::string output_key(::arc::runtime::state::State &s) {
    auto check = ASSERT_NIL_P(s.node("ranges"));
    return std::get<std::string>((*check.output(0)).at(0));
}

} // namespace

TEST(RangesModuleTest, HandlesCreateAndEnd) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Module module(client, noopReporter());
    EXPECT_TRUE(module.handles("create"));
    EXPECT_TRUE(module.handles("end"));
    EXPECT_FALSE(module.handles("start"));
    EXPECT_FALSE(module.handles("anything_else"));
}

TEST(RangesModuleTest, ModuleNameIsRanges) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Module module(client, noopReporter());
    EXPECT_EQ(module.module_name(), "ranges");
}

TEST(RangesModuleTest, CreatesCreateNodeFromBareType) {
    auto node = make_create_ir_node("rng", "", "");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Module module(client, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    ASSERT_NE(created, nullptr);
    EXPECT_NE(dynamic_cast<CreateRange *>(created.get()), nullptr);
}

TEST(RangesModuleTest, CreatesEndNodeFromBareType) {
    auto node = make_end_ir_node(x::uuid::UUID().to_string(), NOW_SENTINEL);
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Module module(client, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    EXPECT_NE(dynamic_cast<EndRange *>(created.get()), nullptr);
}

TEST(RangesModuleTest, ReturnsNotFoundForUnknownType) {
    auto node = make_create_ir_node("rng", "", "");
    node.type = "not_create_or_end";
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    Module module(client, noopReporter());
    auto [created, err] = module.create(
        ::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st))
    );
    EXPECT_EQ(created, nullptr);
    EXPECT_EQ(err, x::errors::NOT_FOUND);
}

TEST(CreateRangeTest, NextCreatesOpenRange) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto name = unique_name("cpp_create_");
    auto node = make_create_ir_node(name, "", "");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    Module module(client, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));

    const auto key = output_key(s);
    const auto uid = ASSERT_NIL_P(x::uuid::UUID::parse(key));
    auto [r, err] = client->ranges.retrieve_by_key(uid);
    ASSERT_NIL(err);
    EXPECT_EQ(r.name, name);
    EXPECT_EQ(r.time_range.end, x::telem::TimeStamp::max());
}

TEST(CreateRangeTest, NextParsesColor) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto name = unique_name("cpp_color_");
    auto node = make_create_ir_node(name, "#df6d38", "");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    Module module(client, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));

    auto [r, err] = client->ranges.retrieve_by_key(
        ASSERT_NIL_P(x::uuid::UUID::parse(output_key(s)))
    );
    ASSERT_NIL(err);
    EXPECT_EQ(r.color.r, 0xdf);
    EXPECT_EQ(r.color.g, 0x6d);
    EXPECT_EQ(r.color.b, 0x38);
}

TEST(CreateRangeTest, NextWarnsOnInvalidColorButStillCreates) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto name = unique_name("cpp_badcolor_");
    std::vector<std::pair<std::string, std::string>> calls;
    auto node = make_create_ir_node(name, "not-a-color", "");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    Module module(client, recordingReporter(&calls));
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));

    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].first, synnax::status::VARIANT_WARNING);
    EXPECT_NE(calls[0].second.find("ranges.create: invalid color"), std::string::npos);

    auto [r, err] = client->ranges.retrieve_by_key(
        ASSERT_NIL_P(x::uuid::UUID::parse(output_key(s)))
    );
    ASSERT_NIL(err);
    EXPECT_EQ(r.name, name);
}

TEST(CreateRangeTest, NextWarnsAndEmitsEmptyKeyOnInvalidParent) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    std::vector<std::pair<std::string, std::string>> calls;
    auto node = make_create_ir_node(unique_name("cpp_badparent_"), "", "not-a-uuid");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    Module module(client, recordingReporter(&calls));
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));

    EXPECT_EQ(output_key(s), "");
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].first, synnax::status::VARIANT_WARNING);
    EXPECT_NE(
        calls[0].second.find("ranges.create: invalid parent key"),
        std::string::npos
    );
}

TEST(EndRangeTest, NextSetsExplicitEnd) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    synnax::ranger::Range r;
    r.name = unique_name("cpp_end_");
    r.time_range = x::telem::TimeRange{
        x::telem::TimeStamp::now(),
        x::telem::TimeStamp::max()
    };
    ASSERT_NIL(client->ranges.create(r));

    const auto end = x::telem::TimeStamp(10 * x::telem::SECOND);
    auto node = make_end_ir_node(r.key.to_string(), end.nanoseconds());
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    Module module(client, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));

    auto [updated, err] = client->ranges.retrieve_by_key(r.key);
    ASSERT_NIL(err);
    EXPECT_EQ(updated.time_range.end, end);
}

TEST(EndRangeTest, NextWarnsWhenRangeDoesNotExist) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    std::vector<std::pair<std::string, std::string>> calls;
    auto node = make_end_ir_node(x::uuid::create().to_string(), NOW_SENTINEL);
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    Module module(client, recordingReporter(&calls));
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));

    EXPECT_EQ(output_key(s), "");
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].first, synnax::status::VARIANT_WARNING);
    EXPECT_NE(calls[0].second.find("ranges.end:"), std::string::npos);
}

TEST(EndRangeTest, NextWarnsOnInvalidKey) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    std::vector<std::pair<std::string, std::string>> calls;
    auto node = make_end_ir_node("not-a-uuid", NOW_SENTINEL);
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("ranges"));
    Module module(client, recordingReporter(&calls));
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].first, synnax::status::VARIANT_WARNING);
    EXPECT_NE(calls[0].second.find("ranges.end: invalid range key"), std::string::npos);
}

}
