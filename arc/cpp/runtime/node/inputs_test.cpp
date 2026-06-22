// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/inputs.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::node {

namespace {
arc::types::Param param(const std::string &name, arc::types::Kind kind) {
    arc::types::Param p;
    p.name = name;
    p.type = arc::types::Type{.kind = kind};
    return p;
}

arc::types::Param static_param(
    const std::string &name,
    arc::types::Kind kind,
    const x::json::json &value
) {
    auto p = param(name, kind);
    p.value = value;
    return p;
}

// builds a producer -> consumer state, returning both nodes. The producer's single
// output feeds the consumer input named `edge_target`.
arc::runtime::state::State build_state(
    const arc::ir::Node &consumer,
    const std::string &edge_target,
    arc::types::Kind producer_output_kind
) {
    arc::ir::Node producer;
    producer.key = "src";
    producer.type = "src";
    producer.outputs.push_back(param("output", producer_output_kind));

    arc::ir::IR ir;
    ir.nodes.push_back(producer);
    ir.nodes.push_back(consumer);
    if (!edge_target.empty())
        ir.edges.emplace_back(
            arc::ir::Handle("src", "output"),
            arc::ir::Handle(consumer.key, edge_target)
        );
    arc::runtime::state::Config cfg{.ir = ir, .channels = {}};
    return arc::runtime::state::State(cfg, arc::runtime::errors::noop_handler);
}
}

TEST(ResolvedInputsTest, OverlaysEdgeFedNumericInput) {
    arc::ir::Node consumer;
    consumer.key = "tgt";
    consumer.type = "tgt";
    consumer.inputs.push_back(
        static_param("static_p", arc::types::Kind::I64, static_cast<int64_t>(7))
    );
    consumer.inputs.push_back(param("edge_p", arc::types::Kind::I64));

    auto s = build_state(consumer, "edge_p", arc::types::Kind::I64);
    auto src = ASSERT_NIL_P(s.node("src"));
    auto &o = src.output(0);
    o->resize(1);
    o->set(0, static_cast<int64_t>(42));
    auto &ot = src.output_time(0);
    ot->resize(1);
    ot->set(0, x::telem::TimeStamp(1 * x::telem::MICROSECOND));

    auto tgt = ASSERT_NIL_P(s.node("tgt"));
    ASSERT_TRUE(tgt.refresh_inputs());

    const auto ri = ResolvedInputs::resolve(consumer);
    EXPECT_TRUE(ri.has_edges());
    EXPECT_EQ(std::get<int64_t>(*ri.value_of(tgt, "static_p")), 7);
    EXPECT_EQ(std::get<int64_t>(*ri.value_of(tgt, "edge_p")), 42);
}

TEST(ResolvedInputsTest, ReadsEdgeFedStringInput) {
    arc::ir::Node consumer;
    consumer.key = "tgt";
    consumer.type = "tgt";
    consumer.inputs.push_back(param("name", arc::types::Kind::String));

    auto s = build_state(consumer, "name", arc::types::Kind::String);
    auto src = ASSERT_NIL_P(s.node("src"));
    *src.output(0) = x::telem::Series(std::string("My_range_3"));
    *src.output_time(0) = x::telem::Series(
        x::telem::TimeStamp(1 * x::telem::MICROSECOND)
    );

    auto tgt = ASSERT_NIL_P(s.node("tgt"));
    ASSERT_TRUE(tgt.refresh_inputs());

    const auto ri = ResolvedInputs::resolve(consumer);
    EXPECT_EQ(ri.string_of(tgt, "name"), "My_range_3");
}

TEST(ResolvedInputsTest, NoEdgesWhenEveryInputIsStatic) {
    arc::ir::Node consumer;
    consumer.key = "tgt";
    consumer.type = "tgt";
    consumer.inputs.push_back(
        static_param("p", arc::types::Kind::I64, static_cast<int64_t>(1))
    );

    auto s = build_state(consumer, "", arc::types::Kind::I64);
    auto tgt = ASSERT_NIL_P(s.node("tgt"));

    const auto ri = ResolvedInputs::resolve(consumer);
    EXPECT_FALSE(ri.has_edges());
    EXPECT_EQ(std::get<int64_t>(*ri.value_of(tgt, "p")), 1);
}

TEST(ResolvedInputsTest, ChannelTypedInputIsNotEdgeFed) {
    arc::ir::Node consumer;
    consumer.key = "tgt";
    consumer.type = "tgt";
    consumer.inputs.push_back(param("ch", arc::types::Kind::Chan));
    consumer.inputs.push_back(
        static_param("static_p", arc::types::Kind::I64, static_cast<int64_t>(5))
    );

    const auto ri = ResolvedInputs::resolve(consumer);
    EXPECT_FALSE(ri.has_edges());
}

TEST(ResolvedInputsTest, ValueOfReturnsNulloptForUnknownInput) {
    arc::ir::Node consumer;
    consumer.key = "tgt";
    consumer.type = "tgt";
    consumer.inputs.push_back(
        static_param("p", arc::types::Kind::I64, static_cast<int64_t>(1))
    );

    auto s = build_state(consumer, "", arc::types::Kind::I64);
    auto tgt = ASSERT_NIL_P(s.node("tgt"));

    const auto ri = ResolvedInputs::resolve(consumer);
    EXPECT_FALSE(ri.value_of(tgt, "missing").has_value());
}
}
