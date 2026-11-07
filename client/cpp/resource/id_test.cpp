// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <unordered_set>

#include "client/cpp/resource/id.h"
#include "client/cpp/resource/types.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::resource {
/// @brief it should construct an ID with type and key.
TEST(ResourceID, testConstruction) {
    const ID id{.type = "channel", .key = "42"};
    EXPECT_EQ(id.type, "channel");
    EXPECT_EQ(id.key, "42");
}

/// @brief it should convert an ID to string format "type:key".
TEST(ResourceID, testStringConversion) {
    const ID id{.type = "channel", .key = "42"};
    EXPECT_EQ(id.string(), "channel:42");
}

/// @brief it should parse a valid ID string "channel:42".
TEST(ResourceID, testParseValidID) {
    const auto id = ASSERT_NIL_P(ID::parse("channel:42"));
    EXPECT_EQ(id.type, "channel");
    EXPECT_EQ(id.key, "42");
}

/// @brief it should parse a valid ID with UUID key.
TEST(ResourceID, testParseValidIDWithUUID) {
    const auto id = ASSERT_NIL_P(
        ID::parse("group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8")
    );
    EXPECT_EQ(id.type, "group");
    EXPECT_EQ(id.key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should fail to parse an ID without a colon separator.
TEST(ResourceID, testParseMalformed) {
    const auto [_, err] = ID::parse("malformed");
    ASSERT_MATCHES(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "[resource] - failed to parse id 'malformed': expected format 'type:key'"
    );
}

/// @brief it should fail to parse an ID with only a colon.
TEST(ResourceID, testParseOnlyColon) {
    const auto [_, err] = ID::parse(":");
    ASSERT_MATCHES(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "[resource] - failed to parse id ':': type is empty");
}

/// @brief it should fail to parse an ID with empty type.
TEST(ResourceID, testParseEmptyType) {
    const auto [_, err] = ID::parse(":42");
    ASSERT_MATCHES(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "[resource] - failed to parse id ':42': type is empty");
}

/// @brief it should support round-trip string conversion: parse(id.string()) == id.
TEST(ResourceID, testStringRoundTrip) {
    const ID original{.type = "channel", .key = "42"};
    const auto parsed = ASSERT_NIL_P(ID::parse(original.string()));
    EXPECT_EQ(parsed, original);
}

/// @brief it should compare two IDs for equality.
TEST(ResourceID, testEqualityOperator) {
    const ID id1{.type = "channel", .key = "42"};
    const ID id2{.type = "channel", .key = "42"};
    const ID id3{.type = "channel", .key = "43"};
    const ID id4{.type = "group", .key = "42"};

    EXPECT_TRUE(id1 == id2);
    EXPECT_FALSE(id1 == id3);
    EXPECT_FALSE(id1 == id4);
}

/// @brief it should compare two IDs for inequality.
TEST(ResourceID, testInequalityOperator) {
    const ID id1{.type = "channel", .key = "42"};
    const ID id2{.type = "channel", .key = "42"};
    const ID id3{.type = "channel", .key = "43"};

    EXPECT_FALSE(id1 != id2);
    EXPECT_TRUE(id1 != id3);
}

/// @brief it should parse a vector of ID strings.
TEST(ResourceID, testParseIDs) {
    const std::vector<std::string> strs = {
        "channel:42",
        "group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8",
        "user:admin"
    };
    const auto ids = ASSERT_NIL_P(parse_ids(strs));
    EXPECT_EQ(ids.size(), 3);
    EXPECT_EQ(ids[0].type, "channel");
    EXPECT_EQ(ids[0].key, "42");
    EXPECT_EQ(ids[1].type, "group");
    EXPECT_EQ(ids[1].key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    EXPECT_EQ(ids[2].type, "user");
    EXPECT_EQ(ids[2].key, "admin");
}

/// @brief it should fail to parse a vector with an invalid ID.
TEST(ResourceID, testParseIDsWithInvalid) {
    const std::vector<std::string> strs = {"channel:42", "malformed", "user:admin"};
    const auto [_, err] = parse_ids(strs);
    ASSERT_MATCHES(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "[resource] - failed to parse id 'malformed': expected format 'type:key'"
    );
}

/// @brief it should convert a vector of IDs to strings.
TEST(ResourceID, testIDsToStrings) {
    const std::vector<ID> ids = {
        ID{.type = "channel", .key = "42"},
        ID{.type = "group", .key = "748d31e2-5732-4cb5-8bc9-64d4ad51efe8"},
        ID{.type = "user", .key = "admin"}
    };
    const auto strs = ids_to_strings(ids);
    EXPECT_EQ(strs.size(), 3);
    EXPECT_EQ(strs[0], "channel:42");
    EXPECT_EQ(strs[1], "group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    EXPECT_EQ(strs[2], "user:admin");
}

/// @brief it should return true for a default-constructed (empty) ID.
TEST(ResourceID, testIsZeroDefault) {
    const ID id{};
    EXPECT_TRUE(id.is_zero());
}

/// @brief it should return false for an ID with type and key.
TEST(ResourceID, testIsZeroNonEmpty) {
    const ID id{.type = "channel", .key = "42"};
    EXPECT_FALSE(id.is_zero());
}

/// @brief it should return false when only type is set.
TEST(ResourceID, testIsZeroPartialType) {
    const ID id{.type = "channel", .key = ""};
    EXPECT_FALSE(id.is_zero());
}

/// @brief it should return false when only key is set.
TEST(ResourceID, testIsZeroPartialKey) {
    const ID id{.type = "", .key = "42"};
    EXPECT_FALSE(id.is_zero());
}

/// @brief it should verify ROOT_ID constant.
TEST(ResourceID, testRootIDConstant) {
    EXPECT_EQ(ROOT_ID.type, "builtin");
    EXPECT_EQ(ROOT_ID.key, "root");
    EXPECT_EQ(ROOT_ID.string(), "builtin:root");
}

/// @brief it should parse an ID from JSON.
TEST(ResourceID, testParseFromJSON) {
    x::json::json j = {{"type", "channel"}, {"key", "42"}};
    x::json::Parser parser(j);
    const auto id = ID::parse(parser);
    EXPECT_EQ(id.type, "channel");
    EXPECT_EQ(id.key, "42");
}

/// @brief it should serialize an ID to JSON.
TEST(ResourceID, testToJSON) {
    const ID id{.type = "channel", .key = "42"};
    const auto j = id.to_json();
    EXPECT_EQ(j["type"], "channel");
    EXPECT_EQ(j["key"], "42");
}

/// @brief it should round-trip an ID through JSON.
TEST(ResourceID, testJSONRoundTrip) {
    const ID original{.type = "group", .key = "abc-123"};
    const auto j = original.to_json();
    x::json::Parser parser(j);
    const auto parsed = ID::parse(parser);
    EXPECT_EQ(parsed, original);
}

/// @brief it should round-trip an ID through protobuf.
TEST(ResourceID, testProtoRoundTrip) {
    const ID original{.type = "channel", .key = "42"};
    const auto pb = ASSERT_NIL_P(original.to_proto());
    const auto parsed = ASSERT_NIL_P(ID::from_proto(pb));
    EXPECT_EQ(parsed, original);
}

/// @brief it should round-trip every typed device resource type through protobuf.
TEST(ResourceID, testProtoRoundTripTypedDevices) {
    for (const auto &type:
         {TYPE_ETHERCAT_MASTER,
          TYPE_ETHERCAT_SLAVE,
          TYPE_HTTP_SERVER,
          TYPE_LABJACK_DEVICE,
          TYPE_MODBUS_SERVER,
          TYPE_NI_CHASSIS,
          TYPE_NI_DEVICE,
          TYPE_OPCUA_SERVER}) {
        const ID original{.type = type, .key = "key"};
        const auto pb = ASSERT_NIL_P(original.to_proto());
        const auto parsed = ASSERT_NIL_P(ID::from_proto(pb));
        EXPECT_EQ(parsed, original);
    }
}

/// @brief it should return an error for an unrecognized type in to_proto.
TEST(ResourceID, testToProtoUnrecognizedType) {
    const ID id{.type = "nonexistent", .key = "42"};
    ASSERT_OCCURRED_AS_P(
        id.to_proto(),
        x::errors::Error("unrecognized Type value: nonexistent")
    );
}

/// @brief it should convert a zero ID to protobuf and back.
TEST(ResourceID, testProtoRoundTripBuiltinType) {
    const ID original{.type = "builtin", .key = "root"};
    const auto pb = ASSERT_NIL_P(original.to_proto());
    const auto parsed = ASSERT_NIL_P(ID::from_proto(pb));
    EXPECT_EQ(parsed, original);
}

/// @brief it should find an ID in a hash set by an equal ID.
TEST(ResourceID, testHash) {
    const std::unordered_set<ID> ids{ID{.type = "task", .key = "a"}};
    EXPECT_TRUE(ids.contains(ID{.type = "task", .key = "a"}));
    EXPECT_FALSE(ids.contains(ID{.type = "rack", .key = "a"}));
}
}
