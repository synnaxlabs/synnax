// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "open62541/types.h"

#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xtest/xtest.h"

#include "driver/opc/types/types.h"

/// @brief it should properly manage NodeId lifecycle with RAII.
TEST(TypesTest, NodeIdRAII) {
    opc::NodeId nodeId1;
    EXPECT_TRUE(nodeId1.is_null());

    UA_NodeId numeric = UA_NODEID_NUMERIC(1, 1000);
    opc::NodeId nodeId2(numeric);
    EXPECT_FALSE(nodeId2.is_null());
    EXPECT_EQ(nodeId2.get().namespaceIndex, 1);
    EXPECT_EQ(nodeId2.get().identifierType, UA_NODEIDTYPE_NUMERIC);
    EXPECT_EQ(nodeId2.get().identifier.numeric, 1000);

    UA_NodeId string_id = UA_NODEID_STRING_ALLOC(2, "TestNode");
    {
        opc::NodeId nodeId3(string_id);
        EXPECT_FALSE(nodeId3.is_null());
        EXPECT_EQ(nodeId3.get().namespaceIndex, 2);
        EXPECT_EQ(nodeId3.get().identifierType, UA_NODEIDTYPE_STRING);
    }
    UA_NodeId_clear(&string_id);
}

/// @brief it should properly implement NodeId copy semantics.
TEST(TypesTest, NodeIdCopySemantics) {
    UA_NodeId string_id = UA_NODEID_STRING_ALLOC(2, "TestNode");
    opc::NodeId nodeId1(string_id);
    UA_NodeId_clear(&string_id);

    opc::NodeId nodeId2(std::move(nodeId1));
    EXPECT_FALSE(nodeId2.is_null());
    EXPECT_TRUE(nodeId1.is_null());

    opc::NodeId nodeId3;
    nodeId3 = std::move(nodeId2);
    EXPECT_FALSE(nodeId3.is_null());
    EXPECT_TRUE(nodeId2.is_null());
}

/// @brief it should properly implement NodeId move semantics.
TEST(TypesTest, NodeIdMoveSemantics) {
    UA_NodeId string_id = UA_NODEID_STRING_ALLOC(2, "TestNode");
    opc::NodeId nodeId1(string_id);
    UA_NodeId_clear(&string_id);

    opc::NodeId nodeId2(std::move(nodeId1));
    EXPECT_FALSE(nodeId2.is_null());
    EXPECT_TRUE(nodeId1.is_null());

    opc::NodeId nodeId3;
    nodeId3 = std::move(nodeId2);
    EXPECT_FALSE(nodeId3.is_null());
    EXPECT_TRUE(nodeId2.is_null());
}

/// @brief it should parse NodeId strings in various formats.
TEST(TypesTest, NodeIdParsing) {
    auto [numeric, err1] = opc::NodeId::parse("NS=1;I=1000");
    ASSERT_NIL(err1);
    EXPECT_EQ(numeric.get().namespaceIndex, 1);
    EXPECT_EQ(numeric.get().identifierType, UA_NODEIDTYPE_NUMERIC);
    EXPECT_EQ(numeric.get().identifier.numeric, 1000);

    auto [string_node, err2] = opc::NodeId::parse("NS=2;S=TestNode");
    ASSERT_NIL(err2);
    EXPECT_EQ(string_node.get().namespaceIndex, 2);
    EXPECT_EQ(string_node.get().identifierType, UA_NODEIDTYPE_STRING);

    auto [invalid, err3] = opc::NodeId::parse("InvalidFormat");
    ASSERT_MATCHES(err3, xerrors::VALIDATION);
    EXPECT_TRUE(invalid.is_null());
}

/// @brief it should convert NodeId to string representation.
TEST(TypesTest, NodeIdToString) {
    UA_NodeId numeric = UA_NODEID_NUMERIC(1, 1000);
    std::string str1 = opc::NodeId::to_string(numeric);
    EXPECT_EQ(str1, "NS=1;I=1000");

    UA_NodeId string_node = UA_NODEID_STRING_ALLOC(2, "TestNode");
    std::string str2 = opc::NodeId::to_string(string_node);
    EXPECT_EQ(str2, "NS=2;S=TestNode");
    UA_NodeId_clear(&string_node);
}

/// @brief it should properly manage Variant lifecycle with RAII.
TEST(TypesTest, VariantRAII) {
    opc::Variant var1;
    EXPECT_TRUE(UA_Variant_isEmpty(&var1.get()));

    UA_Variant ua_var;
    UA_Variant_init(&ua_var);
    UA_Float val = 42.0f;
    UA_Variant_setScalarCopy(&ua_var, &val, &UA_TYPES[UA_TYPES_FLOAT]);

    {
        opc::Variant var2(ua_var);
        EXPECT_FALSE(UA_Variant_isEmpty(&var2.get()));
        EXPECT_TRUE(UA_Variant_hasScalarType(&var2.get(), &UA_TYPES[UA_TYPES_FLOAT]));
    }
    UA_Variant_clear(&ua_var);
}

/// @brief it should properly implement Variant copy and move semantics.
TEST(TypesTest, VariantCopyMoveSemantics) {
    UA_Variant ua_var;
    UA_Variant_init(&ua_var);
    UA_Float val = 42.0f;
    UA_Variant_setScalarCopy(&ua_var, &val, &UA_TYPES[UA_TYPES_FLOAT]);

    opc::Variant var1(ua_var);
    UA_Variant_clear(&ua_var);

    opc::Variant var2(std::move(var1));
    EXPECT_TRUE(UA_Variant_hasScalarType(&var2.get(), &UA_TYPES[UA_TYPES_FLOAT]));

    opc::Variant var3(std::move(var2));
    EXPECT_TRUE(UA_Variant_hasScalarType(&var3.get(), &UA_TYPES[UA_TYPES_FLOAT]));
    EXPECT_TRUE(UA_Variant_isEmpty(&var2.get()));
}

/// @brief it should properly manage ReadResponse lifecycle with RAII.
TEST(TypesTest, ReadResponseRAII) {
    UA_ReadResponse ua_response;
    UA_ReadResponse_init(&ua_response);
    EXPECT_EQ(ua_response.resultsSize, 0);

    opc::ReadResponse response(ua_response);
    EXPECT_EQ(response.get().resultsSize, 0);

    opc::ReadResponse response2(std::move(response));
    EXPECT_EQ(response2.get().resultsSize, 0);
}

/// @brief it should properly manage WriteResponse lifecycle with RAII.
TEST(TypesTest, WriteResponseRAII) {
    UA_WriteResponse ua_response;
    UA_WriteResponse_init(&ua_response);
    EXPECT_EQ(ua_response.resultsSize, 0);

    opc::WriteResponse response(ua_response);
    EXPECT_EQ(response.get().resultsSize, 0);

    opc::WriteResponse response2(std::move(response));
    EXPECT_EQ(response2.get().resultsSize, 0);
}

/// @brief it should properly manage LocalizedText lifecycle with RAII.
TEST(TypesTest, LocalizedTextRAII) {
    opc::LocalizedText text1;
    EXPECT_EQ(text1.get().locale.length, 0);

    opc::LocalizedText text2("en", "Hello");
    EXPECT_GT(text2.get().text.length, 0);

    opc::LocalizedText text3(std::move(text2));
    EXPECT_GT(text3.get().text.length, 0);
    EXPECT_EQ(text2.get().text.length, 0);

    opc::LocalizedText text4(std::move(text3));
    EXPECT_GT(text4.get().text.length, 0);
    EXPECT_EQ(text3.get().text.length, 0);
}

/// @brief it should properly manage QualifiedName lifecycle with RAII.
TEST(TypesTest, QualifiedNameRAII) {
    opc::QualifiedName name1;
    EXPECT_EQ(name1.get().name.length, 0);

    opc::QualifiedName name2(1, "TestName");
    EXPECT_EQ(name2.get().namespaceIndex, 1);
    EXPECT_GT(name2.get().name.length, 0);

    opc::QualifiedName name3(std::move(name2));
    EXPECT_EQ(name3.get().namespaceIndex, 1);
    EXPECT_EQ(name2.get().namespaceIndex, 0);
    EXPECT_EQ(name2.get().name.length, 0);

    opc::QualifiedName name4(std::move(name3));
    EXPECT_EQ(name4.get().namespaceIndex, 1);
    EXPECT_EQ(name3.get().name.length, 0);
}

/// @brief it should properly manage String lifecycle with RAII.
TEST(TypesTest, StringRAII) {
    opc::String str1;
    EXPECT_EQ(str1.get().length, 0);

    opc::String str2("Hello");
    EXPECT_GT(str2.get().length, 0);

    opc::String str3(std::move(str2));
    EXPECT_GT(str3.get().length, 0);
    EXPECT_EQ(str2.get().length, 0);

    opc::String str4(std::move(str3));
    EXPECT_GT(str4.get().length, 0);
    EXPECT_EQ(str3.get().length, 0);
}

/// @brief it should properly manage ByteString lifecycle with RAII.
TEST(TypesTest, ByteStringRAII) {
    opc::ByteString bytes1;
    EXPECT_EQ(bytes1.get().length, 0);

    opc::ByteString bytes2(std::move(bytes1));
    EXPECT_EQ(bytes2.get().length, 0);

    opc::ByteString bytes3(std::move(bytes2));
    EXPECT_EQ(bytes3.get().length, 0);
}

/// @brief it should not cause double-free when moving NodeId.
TEST(TypesTest, NoDoubleFree) {
    UA_NodeId string_id = UA_NODEID_STRING_ALLOC(2, "TestNode");

    {
        opc::NodeId nodeId1(string_id);
        UA_NodeId_clear(&string_id);
        opc::NodeId nodeId2(std::move(nodeId1));
        opc::NodeId nodeId3;
        nodeId3 = std::move(nodeId2);
    }
}

/// @brief it should parse numeric NodeId from JSON.
TEST(TypesTest, ParseNumericNodeIdFromJSON) {
    xjson::Parser parser(std::string(R"({"nodeId": "NS=1;I=42"})"));
    opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
    EXPECT_EQ(nodeId.get().namespaceIndex, 1);
    EXPECT_EQ(nodeId.get().identifierType, UA_NODEIDTYPE_NUMERIC);
    EXPECT_EQ(nodeId.get().identifier.numeric, 42);
}

/// @brief it should parse string NodeId from JSON.
TEST(TypesTest, ParseStringNodeIdFromJSON) {
    xjson::Parser parser(std::string(R"({"nodeId": "NS=2;S=TestString"})"));
    opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
    EXPECT_EQ(nodeId.get().namespaceIndex, 2);
    EXPECT_EQ(nodeId.get().identifierType, UA_NODEIDTYPE_STRING);
    EXPECT_EQ(
        std::string(
            reinterpret_cast<char *>(nodeId.get().identifier.string.data),
            nodeId.get().identifier.string.length
        ),
        "TestString"
    );
}

/// @brief it should parse GUID NodeId from JSON.
TEST(TypesTest, ParseGuidNodeIdFromJSON) {
    xjson::Parser parser(
        std::string(R"({"nodeId": "NS=3;G=12345678-1234-5678-9ABC-123456789ABC"})")
    );
    opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
    EXPECT_EQ(nodeId.get().namespaceIndex, 3);
    EXPECT_EQ(nodeId.get().identifierType, UA_NODEIDTYPE_GUID);
    EXPECT_EQ(nodeId.get().identifier.guid.data1, 0x12345678);
    EXPECT_EQ(nodeId.get().identifier.guid.data2, 0x1234);
    EXPECT_EQ(nodeId.get().identifier.guid.data3, 0x5678);
    EXPECT_EQ(nodeId.get().identifier.guid.data4[0], 0x9A);
    EXPECT_EQ(nodeId.get().identifier.guid.data4[1], 0xBC);
}

/// @brief it should return null for invalid NodeId in JSON.
TEST(TypesTest, ParseInvalidNodeIdFromJSON) {
    xjson::Parser parser(std::string(R"({"nodeId": "Invalid"})"));
    opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
    EXPECT_TRUE(nodeId.is_null());
}

/// @brief it should return error for missing NodeId in JSON.
TEST(TypesTest, ParseMissingNodeIdFromJSON) {
    xjson::Parser parser(std::string(R"({"otherField": "value"})"));
    opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
    EXPECT_TRUE(nodeId.is_null());
    EXPECT_FALSE(parser.ok());
}

/// @brief it should convert numeric NodeId to string.
TEST(TypesTest, NodeIdToStringNumeric) {
    UA_NodeId nodeId = UA_NODEID_NUMERIC(1, 42);
    std::string nodeIdStr = opc::NodeId::to_string(nodeId);
    EXPECT_EQ(nodeIdStr, "NS=1;I=42");
}

/// @brief it should convert string NodeId to string.
TEST(TypesTest, NodeIdToStringString) {
    UA_String uaStr;
    uaStr.data = (UA_Byte *) "TestString";
    uaStr.length = 10;
    UA_NodeId nodeId;
    nodeId.namespaceIndex = 2;
    nodeId.identifierType = UA_NODEIDTYPE_STRING;
    nodeId.identifier.string = uaStr;
    std::string nodeIdStr = opc::NodeId::to_string(nodeId);
    EXPECT_EQ(nodeIdStr, "NS=2;S=TestString");
}

/// @brief it should convert GUID NodeId to string.
TEST(TypesTest, NodeIdToStringGuid) {
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
    std::string nodeIdStr = opc::NodeId::to_string(nodeId);
    EXPECT_EQ(nodeIdStr, "NS=3;G=12345678-1234-5678-9abc-123456789abc");
}

/// @brief it should convert ByteString NodeId to string.
TEST(TypesTest, NodeIdToStringByteString) {
    UA_Byte data[] = {0xDE, 0xAD, 0xBE, 0xEF};
    UA_ByteString byteString;
    byteString.data = data;
    byteString.length = 4;
    UA_NodeId nodeId;
    nodeId.namespaceIndex = 4;
    nodeId.identifierType = UA_NODEIDTYPE_BYTESTRING;
    nodeId.identifier.byteString = byteString;
    std::string nodeIdStr = opc::NodeId::to_string(nodeId);
    EXPECT_EQ(nodeIdStr, "NS=4;B=deadbeef");
}

/// @brief it should round-trip NodeId through parse and to_string.
TEST(TypesTest, NodeIdRoundTripConversion) {
    {
        xjson::Parser parser(std::string(R"({"nodeId": "NS=1;I=42"})"));
        opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
        std::string nodeIdStr = opc::NodeId::to_string(nodeId.get());
        EXPECT_EQ(nodeIdStr, "NS=1;I=42");
    }
    {
        xjson::Parser parser(std::string(R"({"nodeId": "NS=2;S=TestString"})"));
        opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
        std::string nodeIdStr = opc::NodeId::to_string(nodeId.get());
        EXPECT_EQ(nodeIdStr, "NS=2;S=TestString");
    }
    {
        xjson::Parser parser(
            std::string(R"({"nodeId": "NS=3;G=12345678-1234-5678-9ABC-123456789ABC"})")
        );
        opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
        std::string nodeIdStr = opc::NodeId::to_string(nodeId.get());
        EXPECT_EQ(nodeIdStr, "NS=3;G=12345678-1234-5678-9abc-123456789abc");
    }
}
