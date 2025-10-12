// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "open62541/types.h"
#include "gtest/gtest.h"

/// module
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/opc/types/types.h"

// Test NodeId RAII wrapper
TEST(TypesTest, NodeIdRAII) {
    // Test default constructor creates null NodeId
    opc::NodeId nodeId1;
    EXPECT_TRUE(nodeId1.is_null());

    // Test numeric NodeId
    UA_NodeId numeric = UA_NODEID_NUMERIC(1, 1000);
    opc::NodeId nodeId2(numeric);
    EXPECT_FALSE(nodeId2.is_null());
    EXPECT_EQ(nodeId2.get().namespaceIndex, 1);
    EXPECT_EQ(nodeId2.get().identifierType, UA_NODEIDTYPE_NUMERIC);
    EXPECT_EQ(nodeId2.get().identifier.numeric, 1000);

    // Test string NodeId with ALLOC (memory management)
    UA_NodeId string_id = UA_NODEID_STRING_ALLOC(2, "TestNode");
    {
        opc::NodeId nodeId3(string_id);
        EXPECT_FALSE(nodeId3.is_null());
        EXPECT_EQ(nodeId3.get().namespaceIndex, 2);
        EXPECT_EQ(nodeId3.get().identifierType, UA_NODEIDTYPE_STRING);
        // NodeId destructor will clean up string_id copy automatically
    }
    // string_id still needs to be cleaned up
    UA_NodeId_clear(&string_id);
}

// Test NodeId copy semantics
TEST(TypesTest, NodeIdCopySemantics) {
    UA_NodeId string_id = UA_NODEID_STRING_ALLOC(2, "TestNode");
    opc::NodeId nodeId1(string_id);
    UA_NodeId_init(&string_id); // Zero out, nodeId1 now owns it

    // NodeIds are move-only, test move constructor
    opc::NodeId nodeId2(std::move(nodeId1));
    EXPECT_FALSE(nodeId2.is_null());
    EXPECT_TRUE(nodeId1.is_null()); // Moved-from object is null

    // Test move assignment
    opc::NodeId nodeId3;
    nodeId3 = std::move(nodeId2);
    EXPECT_FALSE(nodeId3.is_null());
    EXPECT_TRUE(nodeId2.is_null()); // Moved-from object is null

    // Destructor will clean up nodeId3 automatically
}

// Test NodeId move semantics
TEST(TypesTest, NodeIdMoveSemantics) {
    UA_NodeId string_id = UA_NODEID_STRING_ALLOC(2, "TestNode");
    opc::NodeId nodeId1(string_id);
    UA_NodeId_clear(&string_id);

    // Test move constructor
    opc::NodeId nodeId2(std::move(nodeId1));
    EXPECT_FALSE(nodeId2.is_null());
    EXPECT_TRUE(nodeId1.is_null()); // Original should be null after move

    // Test move assignment
    opc::NodeId nodeId3;
    nodeId3 = std::move(nodeId2);
    EXPECT_FALSE(nodeId3.is_null());
    EXPECT_TRUE(nodeId2.is_null()); // Original should be null after move
}

// Test NodeId parsing
TEST(TypesTest, NodeIdParsing) {
    // Test numeric NodeId
    auto [numeric, err1] = opc::NodeId::parse("NS=1;I=1000");
    ASSERT_NIL(err1);
    EXPECT_EQ(numeric.get().namespaceIndex, 1);
    EXPECT_EQ(numeric.get().identifierType, UA_NODEIDTYPE_NUMERIC);
    EXPECT_EQ(numeric.get().identifier.numeric, 1000);
    // No manual cleanup needed - RAII handles it

    // Test string NodeId
    auto [string_node, err2] = opc::NodeId::parse("NS=2;S=TestNode");
    ASSERT_NIL(err2);
    EXPECT_EQ(string_node.get().namespaceIndex, 2);
    EXPECT_EQ(string_node.get().identifierType, UA_NODEIDTYPE_STRING);
    // No manual cleanup needed - RAII handles it

    // Test invalid NodeId
    auto [invalid, err3] = opc::NodeId::parse("InvalidFormat");
    EXPECT_TRUE(err3.matches(xerrors::VALIDATION));
    EXPECT_TRUE(invalid.is_null());
}

// Test NodeId to_string
TEST(TypesTest, NodeIdToString) {
    // Test numeric NodeId
    UA_NodeId numeric = UA_NODEID_NUMERIC(1, 1000);
    std::string str1 = opc::NodeId::to_string(numeric);
    EXPECT_EQ(str1, "NS=1;I=1000");

    // Test string NodeId
    UA_NodeId string_node = UA_NODEID_STRING_ALLOC(2, "TestNode");
    std::string str2 = opc::NodeId::to_string(string_node);
    EXPECT_EQ(str2, "NS=2;S=TestNode");
    UA_NodeId_clear(&string_node);
}

// Test Variant RAII wrapper
TEST(TypesTest, VariantRAII) {
    // Test default constructor
    opc::Variant var1;
    EXPECT_TRUE(UA_Variant_isEmpty(&var1.get()));

    // Test with scalar float
    UA_Variant ua_var;
    UA_Variant_init(&ua_var);
    UA_Float val = 42.0f;
    UA_Variant_setScalarCopy(&ua_var, &val, &UA_TYPES[UA_TYPES_FLOAT]);

    {
        opc::Variant var2(ua_var);
        EXPECT_FALSE(UA_Variant_isEmpty(&var2.get()));
        EXPECT_TRUE(UA_Variant_hasScalarType(&var2.get(), &UA_TYPES[UA_TYPES_FLOAT]));
        // Destructor cleans up var2's copy
    }
    UA_Variant_clear(&ua_var);
}

// Test Variant copy and move semantics
TEST(TypesTest, VariantCopyMoveSemantics) {
    UA_Variant ua_var;
    UA_Variant_init(&ua_var);
    UA_Float val = 42.0f;
    UA_Variant_setScalarCopy(&ua_var, &val, &UA_TYPES[UA_TYPES_FLOAT]);

    opc::Variant var1(ua_var);
    UA_Variant_clear(&ua_var);

    // Test copy
    opc::Variant var2(std::move(var1)); // Move-only
    EXPECT_TRUE(UA_Variant_hasScalarType(&var2.get(), &UA_TYPES[UA_TYPES_FLOAT]));

    // Test move
    opc::Variant var3(std::move(var2));
    EXPECT_TRUE(UA_Variant_hasScalarType(&var3.get(), &UA_TYPES[UA_TYPES_FLOAT]));
    EXPECT_TRUE(UA_Variant_isEmpty(&var2.get()));
}

// Test ReadResponse RAII wrapper
TEST(TypesTest, ReadResponseRAII) {
    // Create an initialized UA_ReadResponse
    UA_ReadResponse ua_response;
    UA_ReadResponse_init(&ua_response);
    EXPECT_EQ(ua_response.resultsSize, 0);

    opc::ReadResponse response(ua_response);
    EXPECT_EQ(response.get().resultsSize, 0);

    // Test move semantics
    opc::ReadResponse response2(std::move(response));
    EXPECT_EQ(response2.get().resultsSize, 0);
}

// Test WriteResponse RAII wrapper
TEST(TypesTest, WriteResponseRAII) {
    // Create an initialized UA_WriteResponse
    UA_WriteResponse ua_response;
    UA_WriteResponse_init(&ua_response);
    EXPECT_EQ(ua_response.resultsSize, 0);

    opc::WriteResponse response(ua_response);
    EXPECT_EQ(response.get().resultsSize, 0);

    // Test move semantics
    opc::WriteResponse response2(std::move(response));
    EXPECT_EQ(response2.get().resultsSize, 0);
}

// Test LocalizedText RAII wrapper
TEST(TypesTest, LocalizedTextRAII) {
    // Default constructor
    opc::LocalizedText text1;
    EXPECT_EQ(text1.get().locale.length, 0);

    // Constructor with values
    opc::LocalizedText text2("en", "Hello");
    EXPECT_GT(text2.get().text.length, 0);

    // Test move - after move, moved-from object should be empty
    opc::LocalizedText text3(std::move(text2)); // Move-only
    EXPECT_GT(text3.get().text.length, 0); // Moved-to object has the data
    EXPECT_EQ(text2.get().text.length, 0); // Moved-from object is empty

    // Test move again
    opc::LocalizedText text4(std::move(text3));
    EXPECT_GT(text4.get().text.length, 0);
    EXPECT_EQ(text3.get().text.length, 0);
}

// Test QualifiedName RAII wrapper
TEST(TypesTest, QualifiedNameRAII) {
    // Default constructor
    opc::QualifiedName name1;
    EXPECT_EQ(name1.get().name.length, 0);

    // Constructor with values
    opc::QualifiedName name2(1, "TestName");
    EXPECT_EQ(name2.get().namespaceIndex, 1);
    EXPECT_GT(name2.get().name.length, 0);

    // Test move - after move, moved-from object should be empty
    opc::QualifiedName name3(std::move(name2)); // Move-only
    EXPECT_EQ(name3.get().namespaceIndex, 1); // Moved-to object has the data
    EXPECT_EQ(name2.get().namespaceIndex, 0); // Moved-from object is empty
    EXPECT_EQ(name2.get().name.length, 0);

    // Test move again
    opc::QualifiedName name4(std::move(name3));
    EXPECT_EQ(name4.get().namespaceIndex, 1);
    EXPECT_EQ(name3.get().name.length, 0);
}

// Test String RAII wrapper
TEST(TypesTest, StringRAII) {
    // Default constructor
    opc::String str1;
    EXPECT_EQ(str1.get().length, 0);

    // Constructor with C-string
    opc::String str2("Hello");
    EXPECT_GT(str2.get().length, 0);

    // Test move - after move, moved-from object should be empty
    opc::String str3(std::move(str2)); // Move-only
    EXPECT_GT(str3.get().length, 0); // Moved-to object has the data
    EXPECT_EQ(str2.get().length, 0); // Moved-from object is empty

    // Test move again
    opc::String str4(std::move(str3));
    EXPECT_GT(str4.get().length, 0);
    EXPECT_EQ(str3.get().length, 0);
}

// Test ByteString RAII wrapper
TEST(TypesTest, ByteStringRAII) {
    // Default constructor
    opc::ByteString bytes1;
    EXPECT_EQ(bytes1.get().length, 0);

    // Test copy semantics
    opc::ByteString bytes2(std::move(bytes1)); // Move-only
    EXPECT_EQ(bytes2.get().length, 0);

    // Test move semantics
    opc::ByteString bytes3(std::move(bytes2));
    EXPECT_EQ(bytes3.get().length, 0);
}

// Test that RAII wrappers prevent double-free
TEST(TypesTest, NoDoubleFree) {
    // Create a string NodeId with allocated memory
    UA_NodeId string_id = UA_NODEID_STRING_ALLOC(2, "TestNode");

    // Create wrappers that will be moved
    {
        opc::NodeId nodeId1(string_id);
        UA_NodeId_init(&string_id); // Zero out, nodeId1 owns it
        opc::NodeId nodeId2(std::move(nodeId1)); // Move
        opc::NodeId nodeId3;
        nodeId3 = std::move(nodeId2); // Move assign

        // nodeId3 owns the data, will clean up automatically
    } // No double-free should occur here
}

// Test parsing numeric NodeId from JSON
TEST(TypesTest, ParseNumericNodeIdFromJSON) {
    xjson::Parser parser(std::string(R"({"nodeId": "NS=1;I=42"})"));
    opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
    EXPECT_EQ(nodeId.get().namespaceIndex, 1);
    EXPECT_EQ(nodeId.get().identifierType, UA_NODEIDTYPE_NUMERIC);
    EXPECT_EQ(nodeId.get().identifier.numeric, 42);
}

// Test parsing string NodeId from JSON
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

// Test parsing GUID NodeId from JSON
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

// Test parsing invalid NodeId from JSON
TEST(TypesTest, ParseInvalidNodeIdFromJSON) {
    xjson::Parser parser(std::string(R"({"nodeId": "Invalid"})"));
    opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
    EXPECT_TRUE(nodeId.is_null());
}

// Test parsing missing NodeId from JSON
TEST(TypesTest, ParseMissingNodeIdFromJSON) {
    xjson::Parser parser(std::string(R"({"otherField": "value"})"));
    opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
    EXPECT_TRUE(nodeId.is_null());
    EXPECT_FALSE(parser.ok());
}

// Test NodeId to string conversion for numeric type
TEST(TypesTest, NodeIdToStringNumeric) {
    UA_NodeId nodeId = UA_NODEID_NUMERIC(1, 42);
    std::string nodeIdStr = opc::NodeId::to_string(nodeId);
    EXPECT_EQ(nodeIdStr, "NS=1;I=42");
}

// Test NodeId to string conversion for string type
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

// Test NodeId to string conversion for GUID type
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

// Test NodeId to string conversion for ByteString type
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

// Test round-trip conversion for different NodeId types
TEST(TypesTest, NodeIdRoundTripConversion) {
    // Test numeric node ID round trip
    {
        xjson::Parser parser(std::string(R"({"nodeId": "NS=1;I=42"})"));
        opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
        std::string nodeIdStr = opc::NodeId::to_string(nodeId.get());
        EXPECT_EQ(nodeIdStr, "NS=1;I=42");
    }
    // Test string node ID round trip
    {
        xjson::Parser parser(std::string(R"({"nodeId": "NS=2;S=TestString"})"));
        opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
        std::string nodeIdStr = opc::NodeId::to_string(nodeId.get());
        EXPECT_EQ(nodeIdStr, "NS=2;S=TestString");
    }
    // Test GUID node ID round trip
    {
        xjson::Parser parser(
            std::string(R"({"nodeId": "NS=3;G=12345678-1234-5678-9ABC-123456789ABC"})")
        );
        opc::NodeId nodeId = opc::NodeId::parse("nodeId", parser);
        std::string nodeIdStr = opc::NodeId::to_string(nodeId.get());
        EXPECT_EQ(nodeIdStr, "NS=3;G=12345678-1234-5678-9abc-123456789abc");
    }
}
