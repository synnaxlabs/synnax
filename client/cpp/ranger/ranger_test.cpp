// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <random>
#include <string>

#include <include/gtest/gtest.h>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/test/test.h"

namespace synnax::ranger {
std::mt19937 gen_rand = random_generator(std::move("Ranger Tests"));

/// @brief it should create a new range and assign it a non-zero key.
TEST(RangerTests, testCreate) {
    const auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        "test",
        x::telem::TimeRange(x::telem::TimeStamp(10), x::telem::TimeStamp(100))
    ));
    ASSERT_EQ(range.name, "test");
    ASSERT_FALSE(range.key.is_nil());
    ASSERT_EQ(range.time_range.start, x::telem::TimeStamp(10));
    ASSERT_EQ(range.time_range.end, x::telem::TimeStamp(100));
}

/// @brief it should retrieve a range by its key.
TEST(RangerTests, testRetrieveByKey) {
    const auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        "test",
        x::telem::TimeRange(x::telem::TimeStamp(30), x::telem::TimeStamp(100))
    ));
    const auto got = ASSERT_NIL_P(client.ranges.retrieve_by_key(range.key));
    ASSERT_EQ(got.name, "test");
    ASSERT_FALSE(got.key.is_nil());
    ASSERT_EQ(got.time_range.start, x::telem::TimeStamp(30));
    ASSERT_EQ(got.time_range.end, x::telem::TimeStamp(100));
}

/// @brief it should retrieve a range by its name.
TEST(RangerTests, testRetrieveByName) {
    const auto client = new_test_client();
    const auto rand_name = std::to_string(gen_rand());
    const auto range = ASSERT_NIL_P(client.ranges.create(
        rand_name,
        x::telem::TimeRange(x::telem::TimeStamp(10), x::telem::TimeStamp(100))
    ));
    const auto got = ASSERT_NIL_P(client.ranges.retrieve_by_name(rand_name));
    ASSERT_EQ(got.name, rand_name);
    ASSERT_FALSE(got.key.is_nil());
    ASSERT_EQ(got.time_range.start, x::telem::TimeStamp(10));
    ASSERT_EQ(got.time_range.end, x::telem::TimeStamp(100));
}

/// @brief it should return a not found error when retrieving by non-existent name.
TEST(RangerTests, testRetrieveByNameNotFound) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.ranges.retrieve_by_name("not_found"),
        x::errors::NOT_FOUND
    );
}

/// @brief it should retrieve multiple ranges by their names.
TEST(RangerTests, testRetrieveMultipleByName) {
    const auto client = new_test_client();
    const auto rand_name = std::to_string(gen_rand());
    const auto range = ASSERT_NIL_P(client.ranges.create(
        rand_name,
        x::telem::TimeRange(x::telem::TimeStamp(30), x::telem::TimeStamp(100))
    ));
    const auto range2 = ASSERT_NIL_P(client.ranges.create(
        rand_name,
        x::telem::TimeRange(x::telem::TimeStamp(30), x::telem::TimeStamp(100))
    ));
    const auto got = ASSERT_NIL_P(
        client.ranges.retrieve_by_name(std::vector{rand_name})
    );
    ASSERT_EQ(got.size(), 2);
    ASSERT_EQ(got[0].name, rand_name);
    ASSERT_FALSE(got[0].key.is_nil());
    ASSERT_EQ(got[0].time_range.start, x::telem::TimeStamp(30));
    ASSERT_EQ(got[0].time_range.end, x::telem::TimeStamp(100));
    ASSERT_EQ(got[1].name, rand_name);
    ASSERT_FALSE(got[1].key.is_nil());
    ASSERT_EQ(got[1].time_range.start, x::telem::TimeStamp(30));
    ASSERT_EQ(got[1].time_range.end, x::telem::TimeStamp(100));
}

/// @brief it should retrieve multiple ranges by their keys.
TEST(RangerTests, testRetrieveMultipleByKey) {
    auto client = new_test_client();
    auto tr = x::telem::TimeRange(
        x::telem::TimeStamp(10 * x::telem::SECOND),
        x::telem::TimeStamp(100 * x::telem::SECOND)
    );
    const auto range = ASSERT_NIL_P(client.ranges.create("test", tr));
    const auto range2 = ASSERT_NIL_P(client.ranges.create("test2", tr));
    const auto got = ASSERT_NIL_P(
        client.ranges.retrieve_by_key({range.key, range2.key})
    );
    ASSERT_EQ(got.size(), 2);
    ASSERT_EQ(got[0].name, "test");
    ASSERT_FALSE(got[0].key.is_nil());
    ASSERT_EQ(got[0].time_range.start, x::telem::TimeStamp(10 * x::telem::SECOND));
    ASSERT_EQ(got[0].time_range.end, x::telem::TimeStamp(100 * x::telem::SECOND));
    ASSERT_EQ(got[1].name, "test2");
    ASSERT_FALSE(got[1].key.is_nil());
    ASSERT_EQ(got[1].time_range.start, x::telem::TimeStamp(10 * x::telem::SECOND));
    ASSERT_EQ(got[1].time_range.end, x::telem::TimeStamp(100 * x::telem::SECOND));
}

/// @brief it should set a key-value pair on the range.
TEST(RangerTests, testSet) {
    const auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        "test",
        x::telem::TimeRange(x::telem::TimeStamp(30), x::telem::TimeStamp(100))
    ));
    ASSERT_NIL(range.kv.set("test", "test"));
}

/// @brief it should get a key-value pair on the range.
TEST(RangerTests, testGet) {
    const auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        "test",
        x::telem::TimeRange(x::telem::TimeStamp(30), x::telem::TimeStamp(100))
    ));
    ASSERT_NIL(range.kv.set("test", "test"));
    const auto val = ASSERT_NIL_P(range.kv.get("test"));
    ASSERT_EQ(val, "test");
}

/// @brief it should retrieve a key-value pair from a retrieved range.
TEST(RangerTests, testGetFromRetrieved) {
    auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        "test",
        x::telem::TimeRange(x::telem::TimeStamp(30), x::telem::TimeStamp(100))
    ));
    ASSERT_NIL(range.kv.set("test", "test"));
    const auto got = ASSERT_NIL_P(client.ranges.retrieve_by_key(range.key));
    const auto val = ASSERT_NIL_P(got.kv.get("test"));
    ASSERT_EQ(val, "test");
}

/// @brief it should delete a key-value pair on the range.
TEST(RangerTests, testKVDelete) {
    const auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        "test",
        x::telem::TimeRange(
            x::telem::TimeStamp(30),
            x::telem::TimeStamp(10 * x::telem::SECOND)
        )
    ));
    ASSERT_NIL(range.kv.set("test", "test"));
    ASSERT_NIL(range.kv.del("test"));
    ASSERT_OCCURRED_AS_P(range.kv.get("test"), x::errors::NOT_FOUND);
}

/// @brief it should convert a range key to an ontology ID
TEST(RangerTests, testRangeOntologyId) {
    const auto key = x::uuid::UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8").first;
    const auto id = ontology_id(key);
    ASSERT_EQ(id.type, "range");
    ASSERT_EQ(id.key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should convert multiple range keys to ontology IDs
TEST(RangerTests, testRangeOntologyIds) {
    const std::vector<Key> keys = {
        x::uuid::UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8").first,
        x::uuid::UUID::parse("00000000-0000-0000-0000-000000000001").first,
        x::uuid::UUID::parse("00000000-0000-0000-0000-000000000002").first,
    };
    const auto ids = ontology_ids(keys);
    ASSERT_EQ(ids.size(), 3);
    ASSERT_EQ(ids[0].type, "range");
    ASSERT_EQ(ids[0].key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    ASSERT_EQ(ids[1].type, "range");
    ASSERT_EQ(ids[1].key, "00000000-0000-0000-0000-000000000001");
    ASSERT_EQ(ids[2].type, "range");
    ASSERT_EQ(ids[2].key, "00000000-0000-0000-0000-000000000002");
}

/// @brief it should return empty vector for empty input
TEST(RangerTests, testRangeOntologyIdsEmpty) {
    const std::vector<Key> keys;
    const auto ids = ontology_ids(keys);
    ASSERT_TRUE(ids.empty());
}
}
