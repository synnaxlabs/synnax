// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>
#include <unordered_set>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

namespace x::uuid {

/// @brief it should create a nil UUID with default constructor.
TEST(UUID, testDefaultConstructor) {
    const auto uuid = UUID();
    ASSERT_TRUE(uuid.is_nil());
    ASSERT_EQ(uuid.to_string(), "00000000-0000-0000-0000-000000000000");
}

/// @brief it should verify the NIL constant is nil.
TEST(UUID, testNilConstant) {
    ASSERT_TRUE(NIL.is_nil());
    ASSERT_EQ(NIL.to_string(), "00000000-0000-0000-0000-000000000000");
}

/// @brief it should parse a valid UUID string.
TEST(UUID, testParseValidUUID) {
    const std::string str = "748d31e2-5732-4cb5-8bc9-64d4ad51efe8";
    const auto uuid = ASSERT_NIL_P(UUID::parse(str));
    ASSERT_FALSE(uuid.is_nil());
    ASSERT_EQ(uuid.to_string(), str);
}

/// @brief it should parse a valid UUID string with uppercase letters.
TEST(UUID, testParseUppercaseUUID) {
    const auto uuid = ASSERT_NIL_P(UUID::parse("748D31E2-5732-4CB5-8BC9-64D4AD51EFE8"));
    ASSERT_FALSE(uuid.is_nil());
    ASSERT_EQ(uuid.to_string(), "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should fail to parse an empty string.
TEST(UUID, testParseEmptyString) {
    ASSERT_OCCURRED_AS_P(UUID::parse(""), INVALID);
}

/// @brief it should fail to parse an invalid UUID string.
TEST(UUID, testParseInvalidString) {
    ASSERT_OCCURRED_AS_P(UUID::parse("not-a-valid-uuid"), INVALID);
}

/// @brief it should fail to parse a UUID with wrong length.
TEST(UUID, testParseWrongLength) {
    ASSERT_OCCURRED_AS_P(UUID::parse("748d31e2-5732-4cb5-8bc9"), INVALID);
}

/// @brief it should generate unique UUIDs.
TEST(UUID, testGenerate) {
    const auto uuid1 = create();
    const auto uuid2 = create();
    ASSERT_FALSE(uuid1.is_nil());
    ASSERT_FALSE(uuid2.is_nil());
    ASSERT_NE(uuid1, uuid2);
}

/// @brief it should compare equal UUIDs.
TEST(UUID, testEqualityOperator) {
    const auto uuid1 = ASSERT_NIL_P(
        UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8")
    );
    const auto uuid2 = ASSERT_NIL_P(
        UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8")
    );
    ASSERT_EQ(uuid1, uuid2);
}

/// @brief it should compare different UUIDs.
TEST(UUID, testInequalityOperator) {
    const auto uuid1 = ASSERT_NIL_P(
        UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8")
    );
    const auto uuid2 = ASSERT_NIL_P(
        UUID::parse("00000000-0000-0000-0000-000000000001")
    );
    ASSERT_NE(uuid1, uuid2);
}

/// @brief it should support less than comparison for ordering.
TEST(UUID, testLessThanOperator) {
    const auto uuid1 = ASSERT_NIL_P(
        UUID::parse("00000000-0000-0000-0000-000000000001")
    );
    const auto uuid2 = ASSERT_NIL_P(
        UUID::parse("00000000-0000-0000-0000-000000000002")
    );
    ASSERT_TRUE(uuid1 < uuid2);
    ASSERT_FALSE(uuid2 < uuid1);
}

/// @brief it should support greater than comparison.
TEST(UUID, testGreaterThanOperator) {
    const auto uuid1 = ASSERT_NIL_P(
        UUID::parse("00000000-0000-0000-0000-000000000002")
    );
    const auto uuid2 = ASSERT_NIL_P(
        UUID::parse("00000000-0000-0000-0000-000000000001")
    );
    ASSERT_TRUE(uuid1 > uuid2);
}

/// @brief it should convert to JSON as a string.
TEST(UUID, testToJson) {
    const auto uuid = ASSERT_NIL_P(UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8"));
    const auto j = uuid.to_json();
    ASSERT_TRUE(j.is_string());
    ASSERT_EQ(j.get<std::string>(), "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should parse from JSON parser.
TEST(UUID, testParseFromJsonParser) {
    json::json j = "748d31e2-5732-4cb5-8bc9-64d4ad51efe8";
    json::Parser parser(j);
    const auto uuid = UUID::parse(parser);
    ASSERT_TRUE(parser.ok());
    ASSERT_FALSE(uuid.is_nil());
    ASSERT_EQ(uuid.to_string(), "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should return nil UUID for empty string in JSON parser.
TEST(UUID, testParseFromJsonParserEmpty) {
    json::json j = "";
    json::Parser parser(j);
    const auto uuid = UUID::parse(parser);
    ASSERT_TRUE(uuid.is_nil());
}

/// @brief it should accumulate error for invalid UUID in JSON parser.
TEST(UUID, testParseFromJsonParserInvalid) {
    json::json j = "not-a-uuid";
    json::Parser parser(j);
    const auto uuid = UUID::parse(parser);
    ASSERT_FALSE(parser.ok());
    ASSERT_TRUE(uuid.is_nil());
}

/// @brief it should stream UUID to output stream.
TEST(UUID, testStreamOperator) {
    const auto uuid = ASSERT_NIL_P(UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8"));
    std::stringstream ss;
    ss << uuid;
    ASSERT_EQ(ss.str(), "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should be usable in unordered containers via std::hash.
TEST(UUID, testHashSupport) {
    const auto uuid1 = ASSERT_NIL_P(
        UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8")
    );
    const auto uuid2 = ASSERT_NIL_P(
        UUID::parse("00000000-0000-0000-0000-000000000001")
    );
    std::unordered_set<UUID> set;
    set.insert(uuid1);
    set.insert(uuid2);
    ASSERT_EQ(set.size(), 2);
    ASSERT_TRUE(set.count(uuid1) == 1);
    ASSERT_TRUE(set.count(uuid2) == 1);
}

/// @brief it should construct from raw bytes.
TEST(UUID, testConstructFromBytes) {
    std::array<std::uint8_t, 16> bytes = {
        0x74,
        0x8d,
        0x31,
        0xe2,
        0x57,
        0x32,
        0x4c,
        0xb5,
        0x8b,
        0xc9,
        0x64,
        0xd4,
        0xad,
        0x51,
        0xef,
        0xe8
    };
    const auto uuid = UUID(bytes);
    ASSERT_EQ(uuid.to_string(), "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should return correct size.
TEST(UUID, testSize) {
    ASSERT_EQ(UUID::size(), 16);
}

/// @brief it should provide access to raw data.
TEST(UUID, testDataAccess) {
    const auto uuid = ASSERT_NIL_P(UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8"));
    const auto *data = uuid.data();
    ASSERT_NE(data, nullptr);
    ASSERT_EQ(data[0], 0x74);
    ASSERT_EQ(data[1], 0x8d);
}

/// @brief it should provide access to underlying boost::uuid.
TEST(UUID, testUnderlyingAccess) {
    const auto uuid = ASSERT_NIL_P(UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8"));
    const auto &underlying = uuid.underlying();
    ASSERT_FALSE(underlying.is_nil());
}

/// @brief it should round-trip through string conversion.
TEST(UUID, testStringRoundTrip) {
    const auto original = create();
    const auto str = original.to_string();
    const auto parsed = ASSERT_NIL_P(UUID::parse(str));
    ASSERT_EQ(original, parsed);
}

/// @brief it should round-trip through JSON conversion.
TEST(UUID, testJsonRoundTrip) {
    const auto original = create();
    const auto j = original.to_json();
    json::Parser parser(j);
    const auto parsed = UUID::parse(parser);
    ASSERT_TRUE(parser.ok());
    ASSERT_EQ(original, parsed);
}

}
