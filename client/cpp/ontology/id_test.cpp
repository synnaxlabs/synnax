// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/ontology/id.h"
#include "x/cpp/xtest/xtest.h"

/// @brief it should construct an synnax::ontology::ID with type and key.
TEST(OntologyID, testConstruction) {
    const synnax::ontology::ID id("channel", "42");
    EXPECT_EQ(id.type, "channel");
    EXPECT_EQ(id.key, "42");
}

/// @brief it should convert an synnax::ontology::ID to string format "type:key".
TEST(OntologyID, testStringConversion) {
    const synnax::ontology::ID id("channel", "42");
    EXPECT_EQ(id.string(), "channel:42");
}

/// @brief it should parse a valid synnax::ontology::ID string "channel:42".
TEST(OntologyID, testParseValidID) {
    const auto id = ASSERT_NIL_P(synnax::ontology::ID::parse("channel:42"));
    EXPECT_EQ(id.type, "channel");
    EXPECT_EQ(id.key, "42");
}

/// @brief it should parse a valid synnax::ontology::ID with UUID key.
TEST(OntologyID, testParseValidIDWithUUID) {
    const auto id = ASSERT_NIL_P(
        synnax::ontology::ID::parse("group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8")
    );
    EXPECT_EQ(id.type, "group");
    EXPECT_EQ(id.key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should fail to parse an synnax::ontology::ID without a colon separator.
TEST(OntologyID, testParseMalformed) {
    auto [id, err] = synnax::ontology::ID::parse("malformed");
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(xerrors::VALIDATION));
}

/// @brief it should fail to parse an synnax::ontology::ID with only a colon.
TEST(OntologyID, testParseOnlyColon) {
    auto [id, err] = synnax::ontology::ID::parse(":");
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(xerrors::VALIDATION));
}

/// @brief it should fail to parse an synnax::ontology::ID with empty type.
TEST(OntologyID, testParseEmptyType) {
    ASSERT_OCCURRED_AS_P(synnax::ontology::ID::parse(":42"), xerrors::VALIDATION);
}

/// @brief it should support round-trip string conversion: parse(id.string()) == id.
TEST(OntologyID, testStringRoundTrip) {
    const synnax::ontology::ID original("channel", "42");
    const auto parsed = ASSERT_NIL_P(synnax::ontology::ID::parse(original.string()));
    EXPECT_EQ(parsed, original);
}

/// @brief it should compare two IDs for equality.
TEST(OntologyID, testEqualityOperator) {
    const synnax::ontology::ID id1("channel", "42");
    const synnax::ontology::ID id2("channel", "42");
    const synnax::ontology::ID id3("channel", "43");
    const synnax::ontology::ID id4("group", "42");

    EXPECT_TRUE(id1 == id2);
    EXPECT_FALSE(id1 == id3);
    EXPECT_FALSE(id1 == id4);
}

/// @brief it should compare two IDs for inequality.
TEST(OntologyID, testInequalityOperator) {
    const synnax::ontology::ID id1("channel", "42");
    const synnax::ontology::ID id2("channel", "42");
    const synnax::ontology::ID id3("channel", "43");

    EXPECT_FALSE(id1 != id2);
    EXPECT_TRUE(id1 != id3);
}

/// @brief it should parse a vector of synnax::ontology::ID strings.
TEST(OntologyID, testParseIDs) {
    const std::vector<std::string> strs = {
        "channel:42",
        "group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8",
        "user:admin"
    };
    const auto ids = ASSERT_NIL_P(synnax::ontology::parse_ids(strs));
    EXPECT_EQ(ids.size(), 3);
    EXPECT_EQ(ids[0].type, "channel");
    EXPECT_EQ(ids[0].key, "42");
    EXPECT_EQ(ids[1].type, "group");
    EXPECT_EQ(ids[1].key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    EXPECT_EQ(ids[2].type, "user");
    EXPECT_EQ(ids[2].key, "admin");
}

/// @brief it should fail to parse a vector with an invalid synnax::ontology::ID.
TEST(OntologyID, testParseIDsWithInvalid) {
    const std::vector<std::string> strs = {"channel:42", "malformed", "user:admin"};
    ASSERT_OCCURRED_AS_P(synnax::ontology::parse_ids(strs), xerrors::VALIDATION);
}

/// @brief it should convert a vector of IDs to strings.
TEST(OntologyID, testIDsToStrings) {
    const std::vector<synnax::ontology::ID> ids = {
        synnax::ontology::ID("channel", "42"),
        synnax::ontology::ID("group", "748d31e2-5732-4cb5-8bc9-64d4ad51efe8"),
        synnax::ontology::ID("user", "admin")
    };
    const auto strs = synnax::ontology::ids_to_strings(ids);
    EXPECT_EQ(strs.size(), 3);
    EXPECT_EQ(strs[0], "channel:42");
    EXPECT_EQ(strs[1], "group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    EXPECT_EQ(strs[2], "user:admin");
}

/// @brief it should verify ROOT_ID constant.
TEST(OntologyID, testRootIDConstant) {
    EXPECT_EQ(synnax::ontology::ROOT_ID.type, "builtin");
    EXPECT_EQ(synnax::ontology::ROOT_ID.key, "root");
    EXPECT_EQ(synnax::ontology::ROOT_ID.string(), "builtin:root");
}
