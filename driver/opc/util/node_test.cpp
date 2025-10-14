// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xjson/xjson.h"

#include "driver/opc/util/util.h"

TEST(NodeTest, ParseNumericNodeId) {
    xjson::Parser parser(std::string(R"({"nodeId": "NS=1;I=42"})"));

    UA_NodeId nodeId = util::parse_node_id("nodeId", parser);

    EXPECT_EQ(nodeId.namespaceIndex, 1);
    EXPECT_EQ(nodeId.identifierType, UA_NODEIDTYPE_NUMERIC);
    EXPECT_EQ(nodeId.identifier.numeric, 42);

    UA_NodeId_clear(&nodeId);
}

TEST(NodeTest, ParseStringNodeId) {
    xjson::Parser parser(std::string(R"({"nodeId": "NS=2;S=TestString"})"));

    UA_NodeId nodeId = util::parse_node_id("nodeId", parser);

    EXPECT_EQ(nodeId.namespaceIndex, 2);
    EXPECT_EQ(nodeId.identifierType, UA_NODEIDTYPE_STRING);
    EXPECT_EQ(
        std::string(
            reinterpret_cast<char *>(nodeId.identifier.string.data),
            nodeId.identifier.string.length
        ),
        "TestString"
    );

    UA_NodeId_clear(&nodeId);
}

TEST(NodeTest, ParseGuidNodeId) {
    xjson::Parser parser(
        std::string(R"({"nodeId": "NS=3;G=12345678-1234-5678-9ABC-123456789ABC"})")
    );

    UA_NodeId nodeId = util::parse_node_id("nodeId", parser);

    EXPECT_EQ(nodeId.namespaceIndex, 3);
    EXPECT_EQ(nodeId.identifierType, UA_NODEIDTYPE_GUID);
    EXPECT_EQ(nodeId.identifier.guid.data1, 0x12345678);
    EXPECT_EQ(nodeId.identifier.guid.data2, 0x1234);
    EXPECT_EQ(nodeId.identifier.guid.data3, 0x5678);
    EXPECT_EQ(nodeId.identifier.guid.data4[0], 0x9A);
    EXPECT_EQ(nodeId.identifier.guid.data4[1], 0xBC);

    UA_NodeId_clear(&nodeId);
}

TEST(NodeTest, ParseInvalidNodeId) {
    xjson::Parser parser(std::string(R"({"nodeId": "Invalid"})"));

    UA_NodeId nodeId = util::parse_node_id("nodeId", parser);

    EXPECT_TRUE(UA_NodeId_isNull(&nodeId));
}

TEST(NodeTest, ParseMissingNodeId) {
    xjson::Parser parser(std::string(R"({"otherField": "value"})"));

    UA_NodeId nodeId = util::parse_node_id("nodeId", parser);

    EXPECT_TRUE(UA_NodeId_isNull(&nodeId));
    EXPECT_FALSE(parser.ok());
}

TEST(NodeTest, NodeIdToStringNumeric) {
    UA_NodeId nodeId = UA_NODEID_NUMERIC(1, 42);
    std::string nodeIdStr = util::node_id_to_string(nodeId);
    EXPECT_EQ(nodeIdStr, "NS=1;I=42");
}

TEST(NodeTest, NodeIdToStringString) {
    UA_String uaStr;
    uaStr.data = (UA_Byte *) "TestString";
    uaStr.length = 10;

    UA_NodeId nodeId;
    nodeId.namespaceIndex = 2;
    nodeId.identifierType = UA_NODEIDTYPE_STRING;
    nodeId.identifier.string = uaStr;

    std::string nodeIdStr = util::node_id_to_string(nodeId);
    EXPECT_EQ(nodeIdStr, "NS=2;S=TestString");
}

TEST(NodeTest, NodeIdToStringGuid) {
    UA_Guid guid;
    guid.data1 = 0x12345678;
    guid.data2 = 0x1234;
    guid.data3 = 0x5678;
    guid.data4[0] = 0x9A;
    guid.data4[1] = 0xBC;
    guid.data4[2] = 0x12;
    guid.data4[3] = 0x34;
    guid.data4[4] = 0x56;
    guid.data4[5] = 0x78;
    guid.data4[6] = 0x9A;
    guid.data4[7] = 0xBC;

    UA_NodeId nodeId;
    nodeId.namespaceIndex = 3;
    nodeId.identifierType = UA_NODEIDTYPE_GUID;
    nodeId.identifier.guid = guid;

    std::string nodeIdStr = util::node_id_to_string(nodeId);
    EXPECT_EQ(nodeIdStr, "NS=3;G=12345678-1234-5678-9abc-123456789abc");
}

TEST(NodeTest, NodeIdToStringByteString) {
    UA_Byte data[] = {0xDE, 0xAD, 0xBE, 0xEF};
    UA_ByteString byteString;
    byteString.data = data;
    byteString.length = 4;

    UA_NodeId nodeId;
    nodeId.namespaceIndex = 4;
    nodeId.identifierType = UA_NODEIDTYPE_BYTESTRING;
    nodeId.identifier.byteString = byteString;

    std::string nodeIdStr = util::node_id_to_string(nodeId);
    EXPECT_EQ(nodeIdStr, "NS=4;B=deadbeef");
}

TEST(NodeTest, RoundTripConversion) {
    // Test numeric node ID round trip
    {
        xjson::Parser parser(std::string(R"({"nodeId": "NS=1;I=42"})"));
        UA_NodeId nodeId = util::parse_node_id("nodeId", parser);
        std::string nodeIdStr = util::node_id_to_string(nodeId);
        EXPECT_EQ(nodeIdStr, "NS=1;I=42");
        UA_NodeId_clear(&nodeId);
    }

    // Test string node ID round trip
    {
        xjson::Parser parser(std::string(R"({"nodeId": "NS=2;S=TestString"})"));
        UA_NodeId nodeId = util::parse_node_id("nodeId", parser);
        std::string nodeIdStr = util::node_id_to_string(nodeId);
        EXPECT_EQ(nodeIdStr, "NS=2;S=TestString");
        UA_NodeId_clear(&nodeId);
    }

    // Test GUID node ID round trip
    {
        xjson::Parser parser(
            std::string(R"({"nodeId": "NS=3;G=12345678-1234-5678-9ABC-123456789ABC"})")
        );
        UA_NodeId nodeId = util::parse_node_id("nodeId", parser);
        std::string nodeIdStr = util::node_id_to_string(nodeId);
        EXPECT_EQ(nodeIdStr, "NS=3;G=12345678-1234-5678-9abc-123456789abc");
        UA_NodeId_clear(&nodeId);
    }
}
