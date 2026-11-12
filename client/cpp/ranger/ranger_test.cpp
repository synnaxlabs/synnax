// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <random>
#include <string>

#include <include/gtest/gtest.h>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

#include "core/pkg/service/ranger/pb/ranger.pb.h"

namespace synnax::ranger {
// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): one seed per suite,
// printed so that a failing run can be replayed.
std::mt19937 gen_rand = random_generator("Ranger Tests");
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)

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
    const auto got = ASSERT_NIL_P(client.ranges.retrieve(range.key));
    ASSERT_EQ(got.name, "test");
    ASSERT_FALSE(got.key.is_nil());
    ASSERT_EQ(got.time_range.start, x::telem::TimeStamp(30));
    ASSERT_EQ(got.time_range.end, x::telem::TimeStamp(100));
}

/// @brief it should set the end bound of a range while preserving its other fields.
TEST(RangerTests, testSetEnd) {
    const auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        "test",
        x::telem::TimeRange(x::telem::TimeStamp(30), x::telem::TimeStamp(100))
    ));
    ASSERT_NIL(client.ranges.set_end(range.key, x::telem::TimeStamp(500)));
    const auto got = ASSERT_NIL_P(client.ranges.retrieve(range.key));
    ASSERT_EQ(got.name, "test");
    ASSERT_EQ(got.time_range.start, x::telem::TimeStamp(30));
    ASSERT_EQ(got.time_range.end, x::telem::TimeStamp(500));
}

/// @brief it should return an error when setting the end of a nonexistent range.
TEST(RangerTests, testSetEndNotFound) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS(
        client.ranges.set_end(x::uuid::create(), x::telem::TimeStamp(500)),
        x::errors::NOT_FOUND
    );
}

/// @brief it should set an end bound that a float cannot hold exactly.
TEST(RangerTests, testSetEndExact) {
    const auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        "test",
        x::telem::TimeRange(x::telem::TimeStamp(30), x::telem::TimeStamp(100))
    ));
    const auto end = x::telem::TimeStamp(1'700'000'000'000'000'001);
    ASSERT_NIL(client.ranges.set_end(range.key, end));
    const auto got = ASSERT_NIL_P(client.ranges.retrieve(range.key));
    ASSERT_EQ(got.time_range.end, end);
}

/// @brief it should apply each action of a dispatch to a range, and write the
/// timestamps of a time range set exactly.
TEST(RangerTests, testDispatch) {
    const auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        "test",
        x::telem::TimeRange(x::telem::TimeStamp(30), x::telem::TimeStamp(100))
    ));
    const x::telem::TimeRange tr(
        x::telem::TimeStamp(1'700'000'000'000'000'001),
        x::telem::TIME_STAMP_MAX
    );
    ASSERT_NIL(client.ranges.dispatch(
        range.key,
        {set({.path = {"name"}, .value = "renamed"}),
         set({.path = {"time_range"}, .value = tr.to_json()})}
    ));
    const auto got = ASSERT_NIL_P(client.ranges.retrieve(range.key));
    ASSERT_EQ(got.name, "renamed");
    ASSERT_EQ(got.time_range, tr);
}

/// @brief it should return a not found error when it dispatches to a range that does
/// not exist.
TEST(RangerTests, testDispatchNotFound) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS(
        client.ranges
            .dispatch(x::uuid::create(), {set({.path = {"name"}, .value = "renamed"})}),
        x::errors::NOT_FOUND
    );
}

/// @brief it should retrieve a range by its name.
TEST(RangerTests, testRetrieveByName) {
    const auto client = new_test_client();
    const auto rand_name = std::to_string(gen_rand());
    const auto range = ASSERT_NIL_P(client.ranges.create(
        rand_name,
        x::telem::TimeRange(x::telem::TimeStamp(10), x::telem::TimeStamp(100))
    ));
    const auto got = ASSERT_NIL_P(client.ranges.retrieve(rand_name));
    ASSERT_EQ(got.name, rand_name);
    ASSERT_FALSE(got.key.is_nil());
    ASSERT_EQ(got.time_range.start, x::telem::TimeStamp(10));
    ASSERT_EQ(got.time_range.end, x::telem::TimeStamp(100));
}

/// @brief it should return a not found error when retrieving by non-existent name.
TEST(RangerTests, testRetrieveByNameNotFound) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(client.ranges.retrieve("not_found"), x::errors::NOT_FOUND);
}

/// @brief it should reject a name that more than one range has.
TEST(RangerTests, testRetrieveByNameMultipleFound) {
    const auto client = new_test_client();
    const auto rand_name = std::to_string(gen_rand());
    for (int i = 0; i < 2; i++)
        ASSERT_NIL_P(client.ranges.create(
            rand_name,
            x::telem::TimeRange(x::telem::TimeStamp(10), x::telem::TimeStamp(100))
        ));
    ASSERT_OCCURRED_AS_P(
        client.ranges.retrieve(rand_name),
        x::errors::MULTIPLE_RESULTS
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
    const auto got = ASSERT_NIL_P(client.ranges.retrieve(std::vector{rand_name}));
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
    const auto got = ASSERT_NIL_P(client.ranges.retrieve({range.key, range2.key}));
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

/// @brief it should retrieve the ranges that match a tree with an or and a not.
TEST(RangerTests, testRetrieveWhere) {
    const auto client = new_test_client();
    const auto rand_name = std::to_string(gen_rand());
    const auto tr = x::telem::TimeRange(
        x::telem::TimeStamp(10),
        x::telem::TimeStamp(100)
    );
    const auto a = ASSERT_NIL_P(client.ranges.create(rand_name + "_a", tr));
    const auto b = ASSERT_NIL_P(client.ranges.create(rand_name + "_b", tr));
    ASSERT_NIL_P(client.ranges.create(rand_name + "_c", tr));
    const auto got = ASSERT_NIL_P(client.ranges.retrieve(
        (Where{.labels = x::uuid::create()} || Where{.name = {a.name, b.name}}) &&
        !Where{.key = b.key}
    ));
    ASSERT_EQ(got.size(), 1);
    ASSERT_EQ(got[0].key, a.key);
}

/// @brief it should retrieve the one range that a filter matches.
TEST(RangerTests, testRetrieveOneWhere) {
    const auto client = new_test_client();
    const auto range = ASSERT_NIL_P(client.ranges.create(
        std::to_string(gen_rand()),
        x::telem::TimeRange(x::telem::TimeStamp(10), x::telem::TimeStamp(100))
    ));
    const auto got = ASSERT_NIL_P(
        client.ranges.retrieve({.name = range.name}, x::filter::one)
    );
    ASSERT_EQ(got.key, range.key);
}

/// @brief it should name the filter in the error when no range matches it.
TEST(RangerTests, testRetrieveOneWhereNotFound) {
    const auto client = new_test_client();
    const auto name = std::to_string(gen_rand());
    const auto [_, err] = client.ranges.retrieve({.name = name}, x::filter::one);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    ASSERT_EQ(err.data, "Found no range where name = \"" + name + "\"");
}

/// @brief it should return an error when more than one range matches a filter.
TEST(RangerTests, testRetrieveOneWhereMultipleFound) {
    const auto client = new_test_client();
    const auto name = std::to_string(gen_rand());
    for (int i = 0; i < 2; i++)
        ASSERT_NIL_P(client.ranges.create(
            name,
            x::telem::TimeRange(x::telem::TimeStamp(10), x::telem::TimeStamp(100))
        ));
    ASSERT_OCCURRED_AS_P(
        client.ranges.retrieve({.name = name}, x::filter::one),
        x::errors::MULTIPLE_RESULTS
    );
}

/// @brief it should reject a limit or a cursor on a retrieve of one range.
TEST(RangerTests, testRetrieveRejectsPage) {
    const auto client = new_test_client();
    for (const auto &options:
         {RetrieveOptions{.limit = 1}, RetrieveOptions{.after = Cursor{}}})
        for (const auto &err:
             {client.ranges.retrieve("range", options).second,
              client.ranges.retrieve({.name = "range"}, x::filter::one, options)
                  .second}) {
            ASSERT_MATCHES(err, x::errors::VALIDATION);
            ASSERT_EQ(
                err.data,
                "a query for one record takes no limit or cursor, since a page could "
                "hide a second match"
            );
        }
}

/// @brief it should page a retrieve of ranges by keys, by names, or by filter.
TEST(RangerTests, testRetrieveListPage) {
    const auto client = new_test_client();
    const auto name = std::to_string(gen_rand());
    const auto tr = x::telem::TimeRange(
        x::telem::TimeStamp(10),
        x::telem::TimeStamp(20)
    );
    std::vector<Key> keys;
    keys.reserve(3);
    for (int i = 0; i < 3; ++i)
        keys.push_back(ASSERT_NIL_P(client.ranges.create(name, tr)).key);
    const auto by_key = ASSERT_NIL_P(client.ranges.retrieve(keys, {.limit = 2}));
    ASSERT_EQ(by_key.size(), 2);
    const auto where = Where{.name = name};
    const auto first = ASSERT_NIL_P(client.ranges.retrieve(where, {.limit = 2}));
    ASSERT_EQ(first.size(), 2);
    const auto after = RetrieveOptions{.after = Cursor{.key = first[1].key}};
    const auto rest = ASSERT_NIL_P(client.ranges.retrieve(where, after));
    ASSERT_EQ(rest.size(), 1);
    const auto by_name = ASSERT_NIL_P(client.ranges.retrieve(std::vector{name}, after));
    ASSERT_EQ(by_name.size(), 1);
}

/// @brief it should keep only the ranges that match a search term.
TEST(RangerTests, testRetrieveSearchTerm) {
    const auto client = new_test_client();
    const auto name = "searchable-range-" + std::to_string(gen_rand());
    const auto tr = x::telem::TimeRange(
        x::telem::TimeStamp(10),
        x::telem::TimeStamp(20)
    );
    const auto range = ASSERT_NIL_P(client.ranges.create(name, tr));
    const auto other = ASSERT_NIL_P(client.ranges.create("zqxv", tr));
    const auto only_match = [&] {
        const auto [got, err] = client.ranges.retrieve(Where{}, {.search_term = name});
        const auto has = [&](const Key &key) {
            return std::ranges::any_of(got, [&](const Range &r) {
                return r.key == key;
            });
        };
        return !err && has(range.key) && !has(other.key);
    };
    ASSERT_EVENTUALLY_EQ_F(only_match, true);
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
    const auto got = ASSERT_NIL_P(client.ranges.retrieve(range.key));
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

/// @brief it should convert a range key to a resource ID
TEST(RangerTests, testRangeResourceID) {
    const auto key = ASSERT_NIL_P(
        x::uuid::UUID::parse("748d31e2-5732-4cb5-8bc9-64d4ad51efe8")
    );
    const auto id = resource_id(key);
    ASSERT_EQ(id.type, "range");
    ASSERT_EQ(id.key, "748d31e2-5732-4cb5-8bc9-64d4ad51efe8");
}
}
