// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <gtest/gtest.h>

#include "client/cpp/ontology/id.h"

using namespace synnax::ontology;

/// @brief it should construct an ID with type and key.
TEST(OntologyID, testConstruction) {
    const ID id("channel", "42");
    EXPECT_EQ(id.type, "channel");
    EXPECT_EQ(id.key, "42");
}

/// @brief it should convert an ID to string format "type:key".
TEST(OntologyID, testStringConversion) {
    const ID id("channel", "42");
    EXPECT_EQ(id.string(), "channel:42");
}

/// @brief it should parse a valid ID string "channel:42".
TEST(OntologyID, testParseValidID) {
    auto [id, err] = ID::parse("channel:42");
    EXPECT_FALSE(err) << err.message();
    EXPECT_EQ(id.type, "channel");
    EXPECT_EQ(id.key, "42");
}

/// @brief it should parse a valid ID with UUID key.
TEST(OntologyID, testParseValidIDWithUUID) {
    auto [id, err] = ID::parse("group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    EXPECT_FALSE(err) << err.message();
    EXPECT_EQ(id.type, "group");
    EXPECT_EQ(id.key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should fail to parse an ID without a colon separator.
TEST(OntologyID, testParseMalformed) {
    auto [id, err] = ID::parse("malformed");
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(xerrors::VALIDATION));
}

/// @brief it should fail to parse an ID with only a colon.
TEST(OntologyID, testParseOnlyColon) {
    auto [id, err] = ID::parse(":");
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(xerrors::VALIDATION));
}

/// @brief it should fail to parse an ID with empty type.
TEST(OntologyID, testParseEmptyType) {
    auto [id, err] = ID::parse(":42");
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(xerrors::VALIDATION));
}

/// @brief it should fail to parse an ID with empty key.
TEST(OntologyID, testParseEmptyKey) {
    auto [id, err] = ID::parse("channel:");
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(xerrors::VALIDATION));
}

/// @brief it should support round-trip string conversion: parse(id.string()) == id.
TEST(OntologyID, testStringRoundTrip) {
    const ID original("channel", "42");
    auto [parsed, err] = ID::parse(original.string());
    EXPECT_FALSE(err) << err.message();
    EXPECT_EQ(parsed, original);
}

/// @brief it should validate that type is required.
TEST(OntologyID, testValidateEmptyType) {
    const ID id("", "42");
    const auto err = id.validate();
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(xerrors::VALIDATION));
}

/// @brief it should validate that key is required.
TEST(OntologyID, testValidateEmptyKey) {
    const ID id("channel", "");
    const auto err = id.validate();
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(xerrors::VALIDATION));
}

/// @brief it should validate a valid ID.
TEST(OntologyID, testValidateValid) {
    const ID id("channel", "42");
    const auto err = id.validate();
    EXPECT_FALSE(err) << err.message();
}

/// @brief it should compare two IDs for equality.
TEST(OntologyID, testEqualityOperator) {
    const ID id1("channel", "42");
    const ID id2("channel", "42");
    const ID id3("channel", "43");
    const ID id4("group", "42");

    EXPECT_TRUE(id1 == id2);
    EXPECT_FALSE(id1 == id3);
    EXPECT_FALSE(id1 == id4);
}

/// @brief it should compare two IDs for inequality.
TEST(OntologyID, testInequalityOperator) {
    const ID id1("channel", "42");
    const ID id2("channel", "42");
    const ID id3("channel", "43");

    EXPECT_FALSE(id1 != id2);
    EXPECT_TRUE(id1 != id3);
}

/// @brief it should serialize an ID to JSON.
TEST(OntologyID, testToJSON) {
    const ID id("channel", "42");
    json j = id;
    EXPECT_EQ(j["type"], "channel");
    EXPECT_EQ(j["key"], "42");
}

/// @brief it should deserialize an ID from JSON.
TEST(OntologyID, testFromJSON) {
    json j = {{"type", "channel"}, {"key", "42"}};
    const ID id = j.get<ID>();
    EXPECT_EQ(id.type, "channel");
    EXPECT_EQ(id.key, "42");
}

/// @brief it should round-trip JSON serialization.
TEST(OntologyID, testJSONRoundTrip) {
    const ID original("group", "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    json j = original;
    const ID parsed = j.get<ID>();
    EXPECT_EQ(parsed, original);
}

/// @brief it should parse a vector of ID strings.
TEST(OntologyID, testParseIDs) {
    const std::vector<std::string> strs = {
        "channel:42",
        "group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8",
        "user:admin"
    };
    auto [ids, err] = parse_ids(strs);
    EXPECT_FALSE(err) << err.message();
    EXPECT_EQ(ids.size(), 3);
    EXPECT_EQ(ids[0].type, "channel");
    EXPECT_EQ(ids[0].key, "42");
    EXPECT_EQ(ids[1].type, "group");
    EXPECT_EQ(ids[1].key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    EXPECT_EQ(ids[2].type, "user");
    EXPECT_EQ(ids[2].key, "admin");
}

/// @brief it should fail to parse a vector with an invalid ID.
TEST(OntologyID, testParseIDsWithInvalid) {
    const std::vector<std::string> strs = {"channel:42", "malformed", "user:admin"};
    auto [ids, err] = parse_ids(strs);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(xerrors::VALIDATION));
}

/// @brief it should convert a vector of IDs to strings.
TEST(OntologyID, testIDsToStrings) {
    const std::vector<ID> ids = {
        ID("channel", "42"),
        ID("group", "748d31e2-5732-4cb5-8bc9-64d4ad51efe8"),
        ID("user", "admin")
    };
    const auto strs = ids_to_strings(ids);
    EXPECT_EQ(strs.size(), 3);
    EXPECT_EQ(strs[0], "channel:42");
    EXPECT_EQ(strs[1], "group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    EXPECT_EQ(strs[2], "user:admin");
}

/// @brief it should verify ROOT_ID constant.
TEST(OntologyID, testRootIDConstant) {
    EXPECT_EQ(ROOT_ID.type, "builtin");
    EXPECT_EQ(ROOT_ID.key, "root");
    EXPECT_EQ(ROOT_ID.string(), "builtin:root");
}
