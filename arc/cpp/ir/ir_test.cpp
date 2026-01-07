// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "arc/cpp/ir/ir.h"
#include "arc/go/ir/arc/go/ir/ir.pb.h"

/// @brief it should correctly round-trip Handle through protobuf
TEST(IRTest, testHandleProtobufRoundTrip) {
    const arc::ir::Handle original("node1", "param1");
    arc::v1::ir::PBHandle pb;
    original.to_proto(&pb);
    const arc::ir::Handle reconstructed(pb);
    ASSERT_EQ(reconstructed.node, "node1");
    ASSERT_EQ(reconstructed.param, "param1");
}

/// @brief it should correctly round-trip Edge through protobuf
TEST(IRTest, testEdgeProtobufRoundTrip) {
    arc::ir::Handle src("src_node", "output");
    arc::ir::Handle tgt("tgt_node", "input");
    arc::ir::Edge original(src, tgt);
    arc::v1::ir::PBEdge pb;
    original.to_proto(&pb);
    arc::ir::Edge reconstructed(pb);
    ASSERT_EQ(reconstructed.source.node, "src_node");
    ASSERT_EQ(reconstructed.source.param, "output");
    ASSERT_EQ(reconstructed.target.node, "tgt_node");
    ASSERT_EQ(reconstructed.target.param, "input");
}

/// @brief it should correctly round-trip Channels through protobuf
TEST(IRTest, testChannelsProtobufRoundTrip) {
    arc::ir::Channels original;
    original.read[1] = "channel_a";
    original.read[2] = "channel_b";
    original.write[3] = "channel_c";
    arc::v1::symbol::PBChannels pb;
    original.to_proto(&pb);
    arc::ir::Channels reconstructed(pb);
    ASSERT_EQ(reconstructed.read.size(), 2);
    ASSERT_EQ(reconstructed.read[1], "channel_a");
    ASSERT_EQ(reconstructed.read[2], "channel_b");
    ASSERT_EQ(reconstructed.write.size(), 1);
    ASSERT_EQ(reconstructed.write[3], "channel_c");
}

/// @brief it should correctly round-trip Param through protobuf
TEST(IRTest, testParamProtobufRoundTrip) {
    arc::ir::Param original;
    original.name = "test_param";
    original.type = arc::types::Type(arc::types::Kind::F64);
    original.value = 42.5;
    arc::v1::types::PBParam pb;
    original.to_proto(&pb);
    arc::ir::Param reconstructed(pb);
    ASSERT_EQ(reconstructed.name, "test_param");
    ASSERT_EQ(reconstructed.type.kind, arc::types::Kind::F64);
    ASSERT_EQ(reconstructed.get<double>(), 42.5);
}

/// @brief it should correctly round-trip IR through protobuf
TEST(IRTest, testIRProtobufRoundTrip) {
    arc::ir::IR original;

    arc::ir::Function fn;
    fn.key = "test_func";
    fn.channels.read[1] = "chan1";
    original.functions.push_back(fn);

    arc::ir::Node node;
    node.key = "test_node";
    node.type = "add";
    original.nodes.push_back(node);

    arc::ir::Edge edge(arc::ir::Handle("node1", "out"), arc::ir::Handle("node2", "in"));
    original.edges.push_back(edge);

    arc::v1::ir::PBIR pb;
    original.to_proto(&pb);

    arc::ir::IR reconstructed(pb);

    ASSERT_EQ(reconstructed.functions.size(), 1);
    ASSERT_EQ(reconstructed.functions[0].key, "test_func");
    ASSERT_EQ(reconstructed.nodes.size(), 1);
    ASSERT_EQ(reconstructed.nodes[0].key, "test_node");
    ASSERT_EQ(reconstructed.edges.size(), 1);
    ASSERT_EQ(reconstructed.edges[0].source.node, "node1");
}

/// @brief Stage::to_string should format stage with nodes
TEST(StageTest, ToStringFormatsStageWithNodes) {
    arc::ir::Stage stage;
    stage.key = "pressurization";
    stage.nodes = {"timer_1", "controller_1"};
    EXPECT_EQ(stage.to_string(), "pressurization: [timer_1, controller_1]");
}

/// @brief Stage::to_string should format stage with empty nodes
TEST(StageTest, ToStringFormatsEmptyNodes) {
    arc::ir::Stage stage;
    stage.key = "terminal";
    EXPECT_EQ(stage.to_string(), "terminal: []");
}

/// @brief Stage::to_string should include strata when present
TEST(StageTest, ToStringIncludesStrata) {
    arc::ir::Stage stage;
    stage.key = "main";
    stage.nodes = {"a", "b"};
    stage.strata = arc::ir::Strata({{"a"}, {"b"}});
    const auto str = stage.to_string();
    EXPECT_NE(str.find("main: [a, b]"), std::string::npos);
    EXPECT_NE(str.find("[0]: a"), std::string::npos);
    EXPECT_NE(str.find("[1]: b"), std::string::npos);
}

/// @brief Sequence::to_string should format with tree structure
TEST(SequenceTest, ToStringFormatsTreeStructure) {
    arc::ir::Sequence seq;
    seq.key = "main";
    arc::ir::Stage s1, s2;
    s1.key = "precheck";
    s1.nodes = {"check_1"};
    s2.key = "complete";
    seq.stages = {s1, s2};
    const auto str = seq.to_string();
    EXPECT_EQ(str.substr(0, 5), "main\n");
    EXPECT_NE(str.find("precheck:"), std::string::npos);
    EXPECT_NE(str.find("complete:"), std::string::npos);
}

/// @brief Sequence::to_string should use correct tree prefixes
TEST(SequenceTest, ToStringUsesCorrectTreePrefixes) {
    arc::ir::Sequence seq;
    seq.key = "seq";
    arc::ir::Stage s1, s2;
    s1.key = "first";
    s2.key = "last";
    seq.stages = {s1, s2};
    const auto str = seq.to_string();
    // First stage uses ├──, last stage uses └──
    EXPECT_NE(str.find("├── first"), std::string::npos);
    EXPECT_NE(str.find("└── last"), std::string::npos);
}

/// @brief it should access params by name using operator[]
TEST(ParamsTest, testOperatorBracketByName) {
    arc::ir::Params params;
    arc::ir::Param p1;
    p1.name = "alpha";
    p1.value = static_cast<int32_t>(42);
    arc::ir::Param p2;
    p2.name = "beta";
    p2.value = 3.14;
    params.params.push_back(p1);
    params.params.push_back(p2);

    ASSERT_EQ(params["alpha"].get<int32_t>(), 42);
    ASSERT_DOUBLE_EQ(params["beta"].get<double>(), 3.14);
}

/// @brief it should access params by index using operator[]
TEST(ParamsTest, testOperatorBracketByIndex) {
    arc::ir::Params params;
    arc::ir::Param p1;
    p1.name = "first";
    p1.value = static_cast<int32_t>(100);
    arc::ir::Param p2;
    p2.name = "second";
    p2.value = static_cast<int32_t>(200);
    params.params.push_back(p1);
    params.params.push_back(p2);

    ASSERT_EQ(params[0].name, "first");
    ASSERT_EQ(params[0].get<int32_t>(), 100);
    ASSERT_EQ(params[1].name, "second");
    ASSERT_EQ(params[1].get<int32_t>(), 200);
}

/// @brief it should access nodes by key using node()
TEST(IRTest, testNodeAccess) {
    arc::ir::IR ir;
    arc::ir::Node n1;
    n1.key = "node_a";
    n1.type = "add";
    arc::ir::Node n2;
    n2.key = "node_b";
    n2.type = "multiply";
    ir.nodes.push_back(n1);
    ir.nodes.push_back(n2);

    ASSERT_EQ(ir.node("node_a").type, "add");
    ASSERT_EQ(ir.node("node_b").type, "multiply");
}

/// @brief it should access functions by key using function()
TEST(IRTest, testFunctionAccess) {
    arc::ir::IR ir;
    arc::ir::Function f1;
    f1.key = "func_x";
    arc::ir::Function f2;
    f2.key = "func_y";
    ir.functions.push_back(f1);
    ir.functions.push_back(f2);

    ASSERT_EQ(ir.function("func_x").key, "func_x");
    ASSERT_EQ(ir.function("func_y").key, "func_y");
}

/// @brief it should access sequences by key using sequence()
TEST(IRTest, testSequenceAccess) {
    arc::ir::IR ir;
    arc::ir::Sequence s1;
    s1.key = "seq_1";
    arc::ir::Sequence s2;
    s2.key = "seq_2";
    ir.sequences.push_back(s1);
    ir.sequences.push_back(s2);

    ASSERT_EQ(ir.sequence("seq_1").key, "seq_1");
    ASSERT_EQ(ir.sequence("seq_2").key, "seq_2");
}

/// @brief it should find edges by target handle using edge_to()
TEST(IRTest, testEdgeTo) {
    arc::ir::IR ir;
    arc::ir::Handle src1("node_a", "output");
    arc::ir::Handle tgt1("node_b", "input");
    arc::ir::Handle src2("node_c", "out");
    arc::ir::Handle tgt2("node_d", "in");
    ir.edges.emplace_back(src1, tgt1);
    ir.edges.emplace_back(src2, tgt2);

    auto edge1 = ir.edge_to(tgt1);
    ASSERT_TRUE(edge1.has_value());
    ASSERT_EQ(edge1->source.node, "node_a");
    ASSERT_EQ(edge1->source.param, "output");

    auto edge2 = ir.edge_to(tgt2);
    ASSERT_TRUE(edge2.has_value());
    ASSERT_EQ(edge2->source.node, "node_c");

    // Non-existent target should return nullopt
    arc::ir::Handle missing("missing", "input");
    auto no_edge = ir.edge_to(missing);
    ASSERT_FALSE(no_edge.has_value());
}

/// @brief it should return edges grouped by output param using edges_from()
TEST(IRTest, testEdgesFrom) {
    arc::ir::IR ir;
    // Two edges from node_a.output to different targets
    ir.edges.emplace_back(
        arc::ir::Handle("node_a", "output"),
        arc::ir::Handle("node_b", "in1")
    );
    ir.edges.emplace_back(
        arc::ir::Handle("node_a", "output"),
        arc::ir::Handle("node_c", "in2")
    );
    // One edge from node_a.other to another target
    ir.edges.emplace_back(
        arc::ir::Handle("node_a", "other"),
        arc::ir::Handle("node_d", "in3")
    );
    // Edge from different node
    ir.edges.emplace_back(
        arc::ir::Handle("node_x", "out"),
        arc::ir::Handle("node_y", "in")
    );

    auto edges = ir.edges_from("node_a");
    ASSERT_EQ(edges.size(), 2); // Two params: "output" and "other"
    ASSERT_EQ(edges["output"].size(), 2); // Two edges from "output"
    ASSERT_EQ(edges["other"].size(), 1); // One edge from "other"

    // Non-existent node should return empty map
    auto no_edges = ir.edges_from("nonexistent");
    ASSERT_TRUE(no_edges.empty());
}

/// @brief it should return all edges into a node using edges_into()
TEST(IRTest, testEdgesInto) {
    arc::ir::IR ir;
    ir.edges.emplace_back(
        arc::ir::Handle("node_a", "out"),
        arc::ir::Handle("node_target", "in1")
    );
    ir.edges.emplace_back(
        arc::ir::Handle("node_b", "out"),
        arc::ir::Handle("node_target", "in2")
    );
    ir.edges.emplace_back(
        arc::ir::Handle("node_c", "out"),
        arc::ir::Handle("node_other", "in")
    );

    const auto edges = ir.edges_into("node_target");
    ASSERT_EQ(edges.size(), 2);

    const auto no_edges = ir.edges_into("nonexistent");
    ASSERT_TRUE(no_edges.empty());
}

/// @brief it should access stages by key using operator[]
TEST(SequenceTest, testOperatorBracket) {
    arc::ir::Sequence seq;
    seq.key = "my_sequence";
    arc::ir::Stage s1;
    s1.key = "stage_init";
    s1.nodes = {"node1", "node2"};
    arc::ir::Stage s2;
    s2.key = "stage_run";
    s2.nodes = {"node3"};
    seq.stages.push_back(s1);
    seq.stages.push_back(s2);

    ASSERT_EQ(seq["stage_init"].nodes.size(), 2);
    ASSERT_EQ(seq["stage_run"].nodes.size(), 1);
}

/// @brief it should return next stage using next()
TEST(SequenceTest, testNext) {
    arc::ir::Sequence seq;
    arc::ir::Stage s1;
    s1.key = "first";
    arc::ir::Stage s2;
    s2.key = "second";
    arc::ir::Stage s3;
    s3.key = "third";
    seq.stages.push_back(s1);
    seq.stages.push_back(s2);
    seq.stages.push_back(s3);

    auto next1 = seq.next("first");
    ASSERT_TRUE(next1.has_value());
    ASSERT_EQ(next1->key, "second");

    auto next2 = seq.next("second");
    ASSERT_TRUE(next2.has_value());
    ASSERT_EQ(next2->key, "third");

    // Last stage should return nullopt
    auto next3 = seq.next("third");
    ASSERT_FALSE(next3.has_value());

    // Non-existent stage should return nullopt
    auto no_next = seq.next("nonexistent");
    ASSERT_FALSE(no_next.has_value());
}
