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

#include "core/pkg/api/grpc/v1/ranger.pb.h"

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
    const auto key = ASSERT_NIL_P(
        x::uuid::UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8")
    );
    const auto id = ontology_id(key);
    ASSERT_EQ(id.type, "range");
    ASSERT_EQ(id.key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}

/// @brief it should convert multiple range keys to ontology IDs
TEST(RangerTests, testRangeOntologyIds) {
    const std::vector<Key> keys = {
        ASSERT_NIL_P(x::uuid::UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8")),
        ASSERT_NIL_P(x::uuid::UUID::parse("00000000-0000-0000-0000-000000000001")),
        ASSERT_NIL_P(x::uuid::UUID::parse("00000000-0000-0000-0000-000000000002")),
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

/// @brief it should correctly parse all fields from a valid Range proto.
TEST(RangerTests, testRangeFromProto) {
    api::v1::Range pb;
    pb.set_key("748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    pb.set_name("test range");
    pb.mutable_time_range()->set_start(100);
    pb.mutable_time_range()->set_end(200);
    const auto range = ASSERT_NIL_P(Range::from_proto(pb));
    ASSERT_EQ(range.key.to_string(), "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    ASSERT_EQ(range.name, "test range");
    ASSERT_EQ(range.time_range.start, x::telem::TimeStamp(100));
    ASSERT_EQ(range.time_range.end, x::telem::TimeStamp(200));
}

/// @brief it should return an error when parsing a Range proto with an invalid key.
TEST(RangerTests, testRangeFromProtoInvalidKey) {
    api::v1::Range pb;
    pb.set_key("not-a-valid-uuid");
    pb.set_name("bad range");
    pb.mutable_time_range()->set_start(0);
    pb.mutable_time_range()->set_end(100);
    ASSERT_OCCURRED_AS_P(Range::from_proto(pb), x::uuid::INVALID);
}

/// @brief it should return an error when parsing a Range proto with an empty key.
TEST(RangerTests, testRangeFromProtoEmptyKey) {
    api::v1::Range pb;
    pb.set_name("empty key range");
    pb.mutable_time_range()->set_start(0);
    pb.mutable_time_range()->set_end(100);
    ASSERT_OCCURRED_AS_P(Range::from_proto(pb), x::uuid::INVALID);
}

/// @brief it should correctly handle large nanosecond timestamp values.
TEST(RangerTests, testRangeFromProtoLargeTimestamps) {
    api::v1::Range pb;
    pb.set_key("748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    pb.set_name("large ts range");
    const int64_t start = 1700000000000000000LL;
    const int64_t end = 1700000001000000000LL;
    pb.mutable_time_range()->set_start(start);
    pb.mutable_time_range()->set_end(end);
    const auto range = ASSERT_NIL_P(Range::from_proto(pb));
    ASSERT_EQ(range.time_range.start, x::telem::TimeStamp(start));
    ASSERT_EQ(range.time_range.end, x::telem::TimeStamp(end));
}

/// @brief it should roundtrip Range through proto -> C++ -> proto -> C++.
TEST(RangerTests, testRangeFromProtoRoundtrip) {
    api::v1::Range pb;
    pb.set_key("748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
    pb.set_name("roundtrip range");
    pb.mutable_time_range()->set_start(500);
    pb.mutable_time_range()->set_end(1000);
    const auto first = ASSERT_NIL_P(Range::from_proto(pb));
    api::v1::Range pb2;
    pb2.set_key(first.key.to_string());
    pb2.set_name(first.name);
    pb2.mutable_time_range()->set_start(first.time_range.start.nanoseconds());
    pb2.mutable_time_range()->set_end(first.time_range.end.nanoseconds());
    const auto second = ASSERT_NIL_P(Range::from_proto(pb2));
    ASSERT_EQ(first.key, second.key);
    ASSERT_EQ(first.name, second.name);
    ASSERT_EQ(first.time_range.start, second.time_range.start);
    ASSERT_EQ(first.time_range.end, second.time_range.end);
}
}
