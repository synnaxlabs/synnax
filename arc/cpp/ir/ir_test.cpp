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

#include "arc/cpp/ir/ir.h"

/// @brief it should correctly round-trip Handle through protobuf
TEST(IRTest, testHandleProtobufRoundTrip) {
    arc::ir::Handle original{.node = "node1", .param = "param1"};
    auto pb = original.to_proto();
    auto [reconstructed, err] = arc::ir::Handle::from_proto(pb);
    ASSERT_FALSE(err);
    ASSERT_EQ(reconstructed.node, "node1");
    ASSERT_EQ(reconstructed.param, "param1");
}

/// @brief it should correctly round-trip Edge through protobuf
TEST(IRTest, testEdgeProtobufRoundTrip) {
    arc::ir::Handle src{.node = "src_node", .param = "output"};
    arc::ir::Handle tgt{.node = "tgt_node", .param = "input"};
    arc::ir::Edge original{
        .source = src,
        .target = tgt,
        .kind = arc::ir::EdgeKind::Continuous
    };
    auto pb = original.to_proto();
    auto [reconstructed, err] = arc::ir::Edge::from_proto(pb);
    ASSERT_FALSE(err);
    ASSERT_EQ(reconstructed.source.node, "src_node");
    ASSERT_EQ(reconstructed.source.param, "output");
    ASSERT_EQ(reconstructed.target.node, "tgt_node");
    ASSERT_EQ(reconstructed.target.param, "input");
}

/// @brief it should correctly round-trip Channels through protobuf
TEST(IRTest, testChannelsProtobufRoundTrip) {
    arc::types::Channels original;
    original.read[1] = "channel_a";
    original.read[2] = "channel_b";
    original.write[3] = "channel_c";
    auto pb = original.to_proto();
    auto [reconstructed, err] = arc::types::Channels::from_proto(pb);
    ASSERT_FALSE(err);
    ASSERT_EQ(reconstructed.read.size(), 2);
    ASSERT_EQ(reconstructed.read[1], "channel_a");
    ASSERT_EQ(reconstructed.read[2], "channel_b");
    ASSERT_EQ(reconstructed.write.size(), 1);
    ASSERT_EQ(reconstructed.write[3], "channel_c");
}

/// @brief it should correctly round-trip Param through protobuf
TEST(IRTest, testParamProtobufRoundTrip) {
    arc::types::Param original;
    original.name = "test_param";
    original.type = arc::types::Type{.kind = arc::types::Kind::F64};
    auto pb = original.to_proto();
    auto [reconstructed, err] = arc::types::Param::from_proto(pb);
    ASSERT_FALSE(err);
    ASSERT_EQ(reconstructed.name, "test_param");
    ASSERT_EQ(reconstructed.type.kind, arc::types::Kind::F64);
}

/// @brief it should correctly round-trip Node through protobuf
TEST(IRTest, testNodeProtobufRoundTrip) {
    arc::ir::Node original;
    original.key = "test_node";
    original.type = "add";
    auto pb = original.to_proto();
    auto [reconstructed, err] = arc::ir::Node::from_proto(pb);
    ASSERT_FALSE(err);
    ASSERT_EQ(reconstructed.key, "test_node");
    ASSERT_EQ(reconstructed.type, "add");
}

/// @brief it should correctly round-trip Function through protobuf
TEST(IRTest, testFunctionProtobufRoundTrip) {
    arc::ir::Function original;
    original.key = "test_func";
    original.channels.read[1] = "chan1";
    auto pb = original.to_proto();
    auto [reconstructed, err] = arc::ir::Function::from_proto(pb);
    ASSERT_FALSE(err);
    ASSERT_EQ(reconstructed.key, "test_func");
    ASSERT_EQ(reconstructed.channels.read.size(), 1);
    ASSERT_EQ(reconstructed.channels.read[1], "chan1");
}

/// @brief it should correctly round-trip Stage through protobuf
TEST(IRTest, testStageProtobufRoundTrip) {
    arc::ir::Stage original;
    original.key = "test_stage";
    original.nodes = {"node1", "node2"};
    original.strata = {{"a", "b"}, {"c"}};
    auto pb = original.to_proto();
    auto [reconstructed, err] = arc::ir::Stage::from_proto(pb);
    ASSERT_FALSE(err);
    ASSERT_EQ(reconstructed.key, "test_stage");
    ASSERT_EQ(reconstructed.nodes.size(), 2);
    ASSERT_EQ(reconstructed.nodes[0], "node1");
    ASSERT_EQ(reconstructed.nodes[1], "node2");
    ASSERT_EQ(reconstructed.strata.size(), 2);
    ASSERT_EQ(reconstructed.strata[0].size(), 2);
    ASSERT_EQ(reconstructed.strata[0][0], "a");
    ASSERT_EQ(reconstructed.strata[0][1], "b");
    ASSERT_EQ(reconstructed.strata[1].size(), 1);
    ASSERT_EQ(reconstructed.strata[1][0], "c");
}

/// @brief it should correctly round-trip Sequence through protobuf
TEST(IRTest, testSequenceProtobufRoundTrip) {
    arc::ir::Stage s1;
    s1.key = "stage1";
    s1.nodes = {"n1"};
    arc::ir::Stage s2;
    s2.key = "stage2";
    s2.nodes = {"n2", "n3"};

    arc::ir::Sequence original;
    original.key = "test_seq";
    original.stages = {s1, s2};

    auto pb = original.to_proto();
    auto [reconstructed, err] = arc::ir::Sequence::from_proto(pb);
    ASSERT_FALSE(err);
    ASSERT_EQ(reconstructed.key, "test_seq");
    ASSERT_EQ(reconstructed.stages.size(), 2);
    ASSERT_EQ(reconstructed.stages[0].key, "stage1");
    ASSERT_EQ(reconstructed.stages[1].key, "stage2");
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

    arc::ir::Edge edge{
        .source = {.node = "node1", .param = "out"},
        .target = {.node = "node2", .param = "in"},
        .kind = arc::ir::EdgeKind::Continuous
    };
    original.edges.push_back(edge);

    original.strata = {{"a"}, {"b", "c"}};

    auto pb = original.to_proto();
    auto [reconstructed, err] = arc::ir::IR::from_proto(pb);
    ASSERT_FALSE(err);

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

/// @brief it should correctly hash Handle for use in unordered containers
TEST(IRTest, testHandleHash) {
    arc::ir::Handle h1{.node = "node1", .param = "param1"};
    arc::ir::Handle h2{.node = "node1", .param = "param1"};
    arc::ir::Handle h3{.node = "node2", .param = "param1"};

    std::hash<arc::ir::Handle> hasher;
    ASSERT_EQ(hasher(h1), hasher(h2));
    ASSERT_NE(hasher(h1), hasher(h3));
}

/// @brief it should correctly hash Edge for use in unordered containers
TEST(IRTest, testEdgeHash) {
    arc::ir::Edge e1{
        .source = {.node = "n1", .param = "out"},
        .target = {.node = "n2", .param = "in"},
        .kind = arc::ir::EdgeKind::Continuous
    };
    arc::ir::Edge e2{
        .source = {.node = "n1", .param = "out"},
        .target = {.node = "n2", .param = "in"},
        .kind = arc::ir::EdgeKind::Continuous
    };
    arc::ir::Edge e3{
        .source = {.node = "n1", .param = "out"},
        .target = {.node = "n3", .param = "in"},
        .kind = arc::ir::EdgeKind::Continuous
    };

    std::hash<arc::ir::Edge> hasher;
    ASSERT_EQ(hasher(e1), hasher(e2));
    ASSERT_NE(hasher(e1), hasher(e3));
}

/// @brief it should use Handle in unordered_set
TEST(IRTest, testHandleInUnorderedSet) {
    std::unordered_set<arc::ir::Handle> handles;
    handles.insert({.node = "n1", .param = "p1"});
    handles.insert({.node = "n1", .param = "p1"}); // duplicate
    handles.insert({.node = "n2", .param = "p2"});

    ASSERT_EQ(handles.size(), 2);
}

/// @brief it should use Edge in unordered_set
TEST(IRTest, testEdgeInUnorderedSet) {
    std::unordered_set<arc::ir::Edge> edges;
    edges.insert(
        {.source = {.node = "n1", .param = "out"},
         .target = {.node = "n2", .param = "in"},
         .kind = arc::ir::EdgeKind::Continuous}
    );
    edges.insert(
        {.source = {.node = "n1", .param = "out"},
         .target = {.node = "n2", .param = "in"},
         .kind = arc::ir::EdgeKind::Continuous}
    ); // duplicate
    edges.insert(
        {.source = {.node = "n3", .param = "out"},
         .target = {.node = "n4", .param = "in"},
         .kind = arc::ir::EdgeKind::OneShot}
    );

    ASSERT_EQ(edges.size(), 2);
}
