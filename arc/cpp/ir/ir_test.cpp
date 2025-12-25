// Copyright 2025 Synnax Labs, Inc.
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
    original.value = nlohmann::json(42.5);
    arc::v1::types::PBParam pb;
    original.to_proto(&pb);
    arc::ir::Param reconstructed(pb);
    ASSERT_EQ(reconstructed.name, "test_param");
    ASSERT_EQ(reconstructed.type.kind, arc::types::Kind::F64);
    ASSERT_EQ(reconstructed.value.get<double>(), 42.5);
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
    stage.strata.strata = {{"a"}, {"b"}};
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
