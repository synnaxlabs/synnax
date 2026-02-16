// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <unordered_set>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"

namespace arc::ir {
/// @brief it should correctly round-trip Handle through protobuf
TEST(IRTest, testHandleProtobufRoundTrip) {
    const Handle original("node1", "param1");
    const auto pb = original.to_proto();
    const auto reconstructed = ASSERT_NIL_P(Handle::from_proto(pb));
    ASSERT_EQ(reconstructed.node, "node1");
    ASSERT_EQ(reconstructed.param, "param1");
}

/// @brief it should correctly round-trip Edge through protobuf
TEST(IRTest, testEdgeProtobufRoundTrip) {
    Handle src("src_node", "output");
    Handle tgt("tgt_node", "input");
    Edge original(src, tgt);
    const auto pb = original.to_proto();
    const auto reconstructed = ASSERT_NIL_P(Edge::from_proto(pb));
    ASSERT_EQ(reconstructed.source.node, "src_node");
    ASSERT_EQ(reconstructed.source.param, "output");
    ASSERT_EQ(reconstructed.target.node, "tgt_node");
    ASSERT_EQ(reconstructed.target.param, "input");
}

/// @brief it should correctly round-trip Channels through protobuf
TEST(IRTest, testChannelsProtobufRoundTrip) {
    types::Channels original;
    original.read[1] = "channel_a";
    original.read[2] = "channel_b";
    original.write[3] = "channel_c";
    const auto pb = original.to_proto();
    const auto reconstructed = ASSERT_NIL_P(types::Channels::from_proto(pb));
    ASSERT_EQ(reconstructed.read.size(), 2);
    ASSERT_EQ(reconstructed.read.at(1), "channel_a");
    ASSERT_EQ(reconstructed.read.at(2), "channel_b");
    ASSERT_EQ(reconstructed.write.size(), 1);
    ASSERT_EQ(reconstructed.write.at(3), "channel_c");
}

/// @brief it should correctly round-trip Param through protobuf
TEST(IRTest, testParamProtobufRoundTrip) {
    types::Param original;
    original.name = "test_param";
    original.type.kind = types::Kind::F64;
    original.value = 42.5;
    const auto pb = original.to_proto();
    const auto reconstructed = ASSERT_NIL_P(types::Param::from_proto(pb));
    ASSERT_EQ(reconstructed.name, "test_param");
    ASSERT_EQ(reconstructed.type.kind, types::Kind::F64);
    ASSERT_DOUBLE_EQ(reconstructed.value.get<double>(), 42.5);
}

/// @brief it should correctly round-trip IR through protobuf
TEST(IRTest, testIRProtobufRoundTrip) {
    IR original;

    Function fn;
    fn.key = "test_func";
    fn.channels.read[1] = "chan1";
    original.functions.push_back(fn);

    Node node;
    node.key = "test_node";
    node.type = "add";
    original.nodes.push_back(node);

    Edge edge(Handle("node1", "out"), Handle("node2", "in"));
    original.edges.push_back(edge);

    original.strata.push_back({"a"});
    original.strata.push_back({"b", "c"});

    const auto pb = original.to_proto();
    const auto reconstructed = ASSERT_NIL_P(IR::from_proto(pb));
    ASSERT_EQ(reconstructed.functions.size(), 1);
    ASSERT_EQ(reconstructed.functions[0].key, "test_func");
    ASSERT_EQ(reconstructed.nodes.size(), 1);
    ASSERT_EQ(reconstructed.nodes[0].key, "test_node");
    ASSERT_EQ(reconstructed.edges.size(), 1);
    ASSERT_EQ(reconstructed.edges[0].source.node, "node1");
    ASSERT_EQ(reconstructed.strata.size(), 2);
    ASSERT_EQ(reconstructed.strata[0][0], "a");
    ASSERT_EQ(reconstructed.strata[1][0], "b");
    ASSERT_EQ(reconstructed.strata[1][1], "c");
}

/// @brief it should access nodes by key using node()
TEST(IRTest, testNodeAccess) {
    IR ir;
    Node n1;
    n1.key = "node_a";
    n1.type = "add";
    Node n2;
    n2.key = "node_b";
    n2.type = "multiply";
    ir.nodes.push_back(n1);
    ir.nodes.push_back(n2);

    ASSERT_EQ(ir.node("node_a").type, "add");
    ASSERT_EQ(ir.node("node_b").type, "multiply");
}

/// @brief it should access functions by key using function()
TEST(IRTest, testFunctionAccess) {
    IR ir;
    Function f1;
    f1.key = "func_x";
    Function f2;
    f2.key = "func_y";
    ir.functions.push_back(f1);
    ir.functions.push_back(f2);

    ASSERT_EQ(ir.function("func_x").key, "func_x");
    ASSERT_EQ(ir.function("func_y").key, "func_y");
}

/// @brief it should find edges by target handle using edge_to()
TEST(IRTest, testEdgeTo) {
    IR ir;
    Handle src1("node_a", "output");
    Handle tgt1("node_b", "input");
    Handle src2("node_c", "out");
    Handle tgt2("node_d", "in");
    ir.edges.emplace_back(src1, tgt1);
    ir.edges.emplace_back(src2, tgt2);

    auto edge1 = ir.edge_to(tgt1);
    ASSERT_TRUE(edge1.has_value());
    ASSERT_EQ(edge1->source.node, "node_a");
    ASSERT_EQ(edge1->source.param, "output");

    auto edge2 = ir.edge_to(tgt2);
    ASSERT_TRUE(edge2.has_value());
    ASSERT_EQ(edge2->source.node, "node_c");

    Handle missing("missing", "input");
    auto no_edge = ir.edge_to(missing);
    ASSERT_FALSE(no_edge.has_value());
}

/// @brief it should return edges grouped by output param using edges_from()
TEST(IRTest, testEdgesFrom) {
    IR ir;
    ir.edges.emplace_back(Handle("node_a", "output"), Handle("node_b", "in1"));
    ir.edges.emplace_back(Handle("node_a", "output"), Handle("node_c", "in2"));
    ir.edges.emplace_back(Handle("node_a", "other"), Handle("node_d", "in3"));
    ir.edges.emplace_back(Handle("node_x", "out"), Handle("node_y", "in"));

    auto edges = ir.edges_from("node_a");
    ASSERT_EQ(edges.size(), 2);
    ASSERT_EQ(edges["output"].size(), 2);
    ASSERT_EQ(edges["other"].size(), 1);

    auto no_edges = ir.edges_from("nonexistent");
    ASSERT_TRUE(no_edges.empty());
}

/// @brief it should return all edges into a node using edges_to()
TEST(IRTest, testEdgesTo) {
    IR ir;
    ir.edges.emplace_back(Handle("node_a", "out"), Handle("node_target", "in1"));
    ir.edges.emplace_back(Handle("node_b", "out"), Handle("node_target", "in2"));
    ir.edges.emplace_back(Handle("node_c", "out"), Handle("node_other", "in"));

    const auto edges = ir.edges_to("node_target");
    ASSERT_EQ(edges.size(), 2);

    const auto no_edges = ir.edges_to("nonexistent");
    ASSERT_TRUE(no_edges.empty());
}

/// @brief it should format a Stage as a string
TEST(IRTest, testStageToString) {
    Stage stage;
    stage.key = "stage_1";
    stage.nodes = {"node_a", "node_b"};
    const auto str = stage.to_string();
    ASSERT_NE(str.find("stage_1"), std::string::npos);
    ASSERT_NE(str.find("node_a"), std::string::npos);
    ASSERT_NE(str.find("node_b"), std::string::npos);
}

/// @brief it should access sequence stages by index
TEST(IRTest, testSequenceOperatorBracket) {
    Sequence seq;
    seq.key = "seq_1";
    Stage s0;
    s0.key = "init";
    Stage s1;
    s1.key = "run";
    seq.stages.push_back(s0);
    seq.stages.push_back(s1);
    ASSERT_EQ(seq[0].key, "init");
    ASSERT_EQ(seq[1].key, "run");
}

/// @brief it should get the next stage in a sequence
TEST(IRTest, testSequenceNext) {
    Sequence seq;
    seq.key = "seq_1";
    Stage s0;
    s0.key = "init";
    Stage s1;
    s1.key = "run";
    Stage s2;
    s2.key = "stop";
    seq.stages.push_back(s0);
    seq.stages.push_back(s1);
    seq.stages.push_back(s2);
    ASSERT_EQ(seq.next("init").key, "run");
    ASSERT_EQ(seq.next("run").key, "stop");
    ASSERT_THROW((void) seq.next("stop"), std::runtime_error);
    ASSERT_THROW((void) seq.next("nonexistent"), std::runtime_error);
}

/// @brief it should format a Sequence as a string
TEST(IRTest, testSequenceToString) {
    Sequence seq;
    seq.key = "seq_1";
    Stage s0;
    s0.key = "init";
    Stage s1;
    s1.key = "run";
    seq.stages.push_back(s0);
    seq.stages.push_back(s1);
    const auto str = seq.to_string();
    ASSERT_NE(str.find("seq_1"), std::string::npos);
    ASSERT_NE(str.find("init"), std::string::npos);
    ASSERT_NE(str.find("run"), std::string::npos);
}

/// @brief it should access sequences by key from IR
TEST(IRTest, testIRSequenceAccess) {
    IR ir;
    Sequence s1;
    s1.key = "main";
    Sequence s2;
    s2.key = "cleanup";
    ir.sequences.push_back(s1);
    ir.sequences.push_back(s2);
    ASSERT_EQ(ir.sequence("main").key, "main");
    ASSERT_EQ(ir.sequence("cleanup").key, "cleanup");
    ASSERT_THROW((void) ir.sequence("nonexistent"), std::runtime_error);
}
}
