// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/ir/ir.h"

#include <nlohmann/json.hpp>

#include "gtest/gtest.h"

using json = nlohmann::json;

TEST(TypeTest, Density) {
    arc::ir::Type t_u8(arc::ir::TypeKind::U8);
    EXPECT_EQ(t_u8.density(), 1);

    arc::ir::Type t_u16(arc::ir::TypeKind::U16);
    EXPECT_EQ(t_u16.density(), 2);

    arc::ir::Type t_i32(arc::ir::TypeKind::I32);
    EXPECT_EQ(t_i32.density(), 4);

    arc::ir::Type t_f64(arc::ir::TypeKind::F64);
    EXPECT_EQ(t_f64.density(), 8);

    arc::ir::Type t_string(arc::ir::TypeKind::String);
    EXPECT_EQ(t_string.density(), 0);  // Variable
}

TEST(TypeTest, ParseFromJSON) {
    json j = "i32";
    arc::ir::Type t = arc::ir::parse_type(j);

    EXPECT_EQ(t.kind, arc::ir::TypeKind::I32);
    EXPECT_TRUE(t.is_valid());
}

TEST(ParamsTest, ParseFromJSON) {
    json j = {{"a", "i32"}, {"b", "f64"}, {"c", "string"}};

    arc::ir::Params params = arc::ir::parse_params(j);

    EXPECT_EQ(params.count(), 3);
    EXPECT_TRUE(params.contains("a"));
    EXPECT_TRUE(params.contains("b"));
    EXPECT_TRUE(params.contains("c"));

    auto *type_a = params.get("a");
    ASSERT_NE(type_a, nullptr);
    EXPECT_EQ(type_a->kind, arc::ir::TypeKind::I32);

    auto *type_b = params.get("b");
    ASSERT_NE(type_b, nullptr);
    EXPECT_EQ(type_b->kind, arc::ir::TypeKind::F64);

    auto *type_c = params.get("c");
    ASSERT_NE(type_c, nullptr);
    EXPECT_EQ(type_c->kind, arc::ir::TypeKind::String);
}

TEST(HandleTest, Construction) {
    arc::ir::Handle h("node1", "output");

    EXPECT_EQ(h.node, "node1");
    EXPECT_EQ(h.param, "output");
}

TEST(EdgeTest, Construction) {
    arc::ir::Handle source("node1", "output");
    arc::ir::Handle target("node2", "input");
    arc::ir::Edge e(source, target);

    EXPECT_EQ(e.source.node, "node1");
    EXPECT_EQ(e.source.param, "output");
    EXPECT_EQ(e.target.node, "node2");
    EXPECT_EQ(e.target.param, "input");
}

TEST(ChannelsTest, ParseFromJSON) {
    json j = {{"read", {{"1", "input_a"}, {"2", "input_b"}}},
              {"write", {{"output", 3}}}};

    arc::ir::Channels channels = arc::ir::parse_channels(j);

    EXPECT_TRUE(channels.has_reads());
    EXPECT_TRUE(channels.has_writes());
    EXPECT_EQ(channels.read[1], "input_a");
    EXPECT_EQ(channels.read[2], "input_b");
    EXPECT_EQ(channels.write["output"], 3);
}

TEST(NodeTest, Construction) {
    arc::ir::Node n("add_node");
    n.type = "add";
    n.inputs.keys = {"a", "b"};
    n.inputs.values["a"] = arc::ir::Type(arc::ir::TypeKind::I32);
    n.inputs.values["b"] = arc::ir::Type(arc::ir::TypeKind::I32);
    n.outputs.keys = {"sum"};
    n.outputs.values["sum"] = arc::ir::Type(arc::ir::TypeKind::I32);
    n.channels.read[1] = "input_a";

    EXPECT_EQ(n.key, "add_node");
    EXPECT_EQ(n.type, "add");
    EXPECT_EQ(n.inputs.count(), 2);
    EXPECT_EQ(n.outputs.count(), 1);
    EXPECT_TRUE(n.channels.has_reads());
}

TEST(FunctionTest, Construction) {
    arc::ir::Function fn("add");
    fn.raw_body = "output <- a + b";
    fn.inputs.keys = {"a", "b"};
    fn.inputs.values["a"] = arc::ir::Type(arc::ir::TypeKind::F64);
    fn.inputs.values["b"] = arc::ir::Type(arc::ir::TypeKind::F64);
    fn.outputs.keys = {"output"};
    fn.outputs.values["output"] = arc::ir::Type(arc::ir::TypeKind::F64);

    EXPECT_EQ(fn.key, "add");
    EXPECT_EQ(fn.raw_body, "output <- a + b");
    EXPECT_EQ(fn.inputs.count(), 2);
    EXPECT_EQ(fn.outputs.count(), 1);
}

TEST(IRTest, BasicStructure) {
    arc::ir::IR ir;

    arc::ir::Function fn("add");
    fn.inputs.keys = {"a"};
    fn.inputs.values["a"] = arc::ir::Type(arc::ir::TypeKind::I32);
    ir.functions.push_back(fn);

    arc::ir::Node node("node1");
    node.type = "add";
    node.inputs.keys = {"a"};
    node.inputs.values["a"] = arc::ir::Type(arc::ir::TypeKind::I32);
    ir.nodes.push_back(node);

    arc::ir::Edge edge(arc::ir::Handle("input", "value"),
                      arc::ir::Handle("node1", "a"));
    ir.edges.push_back(edge);

    ir.strata = {{"input"}, {"node1"}, {"output"}};

    EXPECT_EQ(ir.functions.size(), 1);
    EXPECT_EQ(ir.nodes.size(), 1);
    EXPECT_EQ(ir.edges.size(), 1);
    EXPECT_EQ(ir.strata.size(), 3);

    EXPECT_EQ(ir.functions[0].key, "add");
    EXPECT_EQ(ir.nodes[0].key, "node1");
    EXPECT_EQ(ir.edges[0].source.node, "input");
    EXPECT_EQ(ir.strata[0][0], "input");
}

TEST(IRTest, FindFunction) {
    arc::ir::IR ir;
    ir.functions.push_back(arc::ir::Function("add"));
    ir.functions.push_back(arc::ir::Function("multiply"));

    auto *fn1 = ir.find_function("add");
    ASSERT_NE(fn1, nullptr);
    EXPECT_EQ(fn1->key, "add");

    auto *fn2 = ir.find_function("multiply");
    ASSERT_NE(fn2, nullptr);
    EXPECT_EQ(fn2->key, "multiply");

    auto *fn3 = ir.find_function("nonexistent");
    EXPECT_EQ(fn3, nullptr);
}

TEST(IRTest, FindNode) {
    arc::ir::IR ir;
    ir.nodes.push_back(arc::ir::Node("node1"));
    ir.nodes.push_back(arc::ir::Node("node2"));

    auto *n1 = ir.find_node("node1");
    ASSERT_NE(n1, nullptr);
    EXPECT_EQ(n1->key, "node1");

    auto *n2 = ir.find_node("nonexistent");
    EXPECT_EQ(n2, nullptr);
}

TEST(IRTest, OutgoingEdges) {
    arc::ir::IR ir;
    ir.edges.push_back(
        arc::ir::Edge(arc::ir::Handle("node1", "out"), arc::ir::Handle("node2", "in")));
    ir.edges.push_back(
        arc::ir::Edge(arc::ir::Handle("node1", "out2"), arc::ir::Handle("node3", "in")));
    ir.edges.push_back(
        arc::ir::Edge(arc::ir::Handle("node2", "out"), arc::ir::Handle("node3", "in2")));

    auto outgoing = ir.outgoing_edges("node1");
    EXPECT_EQ(outgoing.size(), 2);
    EXPECT_EQ(outgoing[0].target.node, "node2");
    EXPECT_EQ(outgoing[1].target.node, "node3");
}

TEST(IRTest, IncomingEdges) {
    arc::ir::IR ir;
    ir.edges.push_back(
        arc::ir::Edge(arc::ir::Handle("node1", "out"), arc::ir::Handle("node3", "in1")));
    ir.edges.push_back(
        arc::ir::Edge(arc::ir::Handle("node2", "out"), arc::ir::Handle("node3", "in2")));

    auto incoming = ir.incoming_edges("node3");
    EXPECT_EQ(incoming.size(), 2);
    EXPECT_EQ(incoming[0].source.node, "node1");
    EXPECT_EQ(incoming[1].source.node, "node2");
}
