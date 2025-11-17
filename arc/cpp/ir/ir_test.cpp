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

/// @brief it should correctly round-trip Handle through JSON
TEST(IRTest, testHandleJSONRoundTrip) {
    const arc::ir::Handle original("node1", "param1");
    const nlohmann::json j = original.to_json();
    const arc::ir::Handle reconstructed{xjson::Parser(j)};
    ASSERT_EQ(reconstructed.node, "node1");
    ASSERT_EQ(reconstructed.param, "param1");
}

/// @brief it should correctly round-trip Channels through JSON
TEST(IRTest, testChannelsJSONRoundTrip) {
    arc::ir::Channels original;
    original.read[1] = "channel_a";
    original.read[2] = "channel_b";
    original.write[3] = "channel_c";

    nlohmann::json j = original.to_json();

    arc::ir::Channels reconstructed{xjson::Parser(j)};

    ASSERT_EQ(reconstructed.read.size(), 2);
    ASSERT_EQ(reconstructed.read[1], "channel_a");
    ASSERT_EQ(reconstructed.read[2], "channel_b");
    ASSERT_EQ(reconstructed.write.size(), 1);
    ASSERT_EQ(reconstructed.write[3], "channel_c");
}

/// @brief it should correctly round-trip Param through JSON
TEST(IRTest, testParamJSONRoundTrip) {
    arc::ir::Param original;
    original.name = "test_param";
    original.type = arc::types::Type(arc::types::Kind::F64);
    original.value = nlohmann::json(42.5);

    nlohmann::json j = original.to_json();

    arc::ir::Param reconstructed{xjson::Parser(j)};

    ASSERT_EQ(reconstructed.name, "test_param");
    ASSERT_EQ(reconstructed.type.kind, arc::types::Kind::F64);
    ASSERT_EQ(reconstructed.value.get<double>(), 42.5);
}

/// @brief it should correctly round-trip IR through JSON
TEST(IRTest, testIRJSONRoundTrip) {
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

    nlohmann::json j = original.to_json();

    arc::ir::IR reconstructed{xjson::Parser(j)};

    ASSERT_EQ(reconstructed.functions.size(), 1);
    ASSERT_EQ(reconstructed.functions[0].key, "test_func");
    ASSERT_EQ(reconstructed.nodes.size(), 1);
    ASSERT_EQ(reconstructed.nodes[0].key, "test_node");
    ASSERT_EQ(reconstructed.edges.size(), 1);
    ASSERT_EQ(reconstructed.edges[0].source.node, "node1");
}
