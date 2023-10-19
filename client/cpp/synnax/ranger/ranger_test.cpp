// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <string>

/// GTest
#include <include/gtest/gtest.h>

/// internal.
#include "synnax/synnax.h"

const synnax::Config cfg = {
        "localhost",
        9090,
        false,
        "synnax",
        "seldon"
};

/// @brief it should create a new range and assign it a non-zero key.
TEST(RangerTests, testCreate) {
    auto client = synnax::Client(cfg);
    auto [range, err] = client.ranges.create(
            "test",
            synnax::TimeRange(
                    synnax::TimeStamp(0),
                    synnax::TimeStamp(100)
            )
    );
    ASSERT_FALSE(err);
    ASSERT_EQ(range.name, "test");
    ASSERT_FALSE(range.key.length() == 0);
    ASSERT_EQ(range.time_range.start, synnax::TimeStamp(0));
    ASSERT_EQ(range.time_range.end, synnax::TimeStamp(100));
}

/// @brief it should retrieve a range by its key.
TEST(RangerTests, testRetrieveByKey) {
    auto client = synnax::Client(cfg);
    auto [range, err] = client.ranges.create(
            "test",
            synnax::TimeRange(
                    synnax::TimeStamp(0),
                    synnax::TimeStamp(100)
            )
    );
    ASSERT_FALSE(err);
    auto [got, err2] = client.ranges.retrieveByKey(range.key);
    ASSERT_FALSE(err2);
    ASSERT_EQ(got.name, "test");
    ASSERT_FALSE(got.key.length() == 0);
    ASSERT_EQ(got.time_range.start, synnax::TimeStamp(0));
    ASSERT_EQ(got.time_range.end, synnax::TimeStamp(100));
}

/// @brief it should retrieve a range by its name.
TEST(RangerTests, testRetrieveByName) {
    auto client = synnax::Client(cfg);
    auto [range, err] = client.ranges.create(
            "test",
            synnax::TimeRange(
                    synnax::TimeStamp(0),
                    synnax::TimeStamp(100)
            )
    );
    ASSERT_FALSE(err);
    auto [got, err2] = client.ranges.retrieveByName("test");
    ASSERT_FALSE(err2);
    ASSERT_EQ(got.name, "test");
    ASSERT_FALSE(got.key.length() == 0);
    ASSERT_EQ(got.time_range.start, synnax::TimeStamp(0));
    ASSERT_EQ(got.time_range.end, synnax::TimeStamp(100));
}

/// @brief it should retrieve multiple ranges by their names.
TEST(RangerTests, testRetrieveMultipleByName) {
    auto client = synnax::Client(cfg);
    auto [range, err] = client.ranges.create(
            "test",
            synnax::TimeRange(
                    synnax::TimeStamp(0),
                    synnax::TimeStamp(100)
            )
    );
    ASSERT_FALSE(err);
    auto [range2, err2] = client.ranges.create(
            "test2",
            synnax::TimeRange(
                    synnax::TimeStamp(0),
                    synnax::TimeStamp(100)
            )
    );
    ASSERT_FALSE(err2);
    auto [got, err3] = client.ranges.retrieveByName(std::vector<std::string>{"test", "test2"});
    ASSERT_FALSE(err3);
    ASSERT_EQ(got.size(), 2);
    ASSERT_EQ(got[0].name, "test");
    ASSERT_FALSE(got[0].key.length() == 0);
    ASSERT_EQ(got[0].time_range.start, synnax::TimeStamp(0));
    ASSERT_EQ(got[0].time_range.end, synnax::TimeStamp(100));
    ASSERT_EQ(got[1].name, "test2");
    ASSERT_FALSE(got[1].key.length() == 0);
    ASSERT_EQ(got[1].time_range.start, synnax::TimeStamp(0));
    ASSERT_EQ(got[1].time_range.end, synnax::TimeStamp(100));
}

/// @brief it should retrieve multiple ranges by their keys.
TEST(RangerTests, testRetrieveMultipleByKey) {
    auto client = synnax::Client(cfg);
    auto tr = synnax::TimeRange(
            synnax::TimeStamp(0 * synnax::SECOND),
            synnax::TimeStamp(100 * synnax::SECOND)
    );
    auto[range, err] = client.ranges.create("test", tr);
    ASSERT_FALSE(err);
    auto [range2, err2] = client.ranges.create("test2",tr);
    ASSERT_FALSE(err2);
    auto [got, err3] = client.ranges.retrieveByKey(std::vector<std::string>{range.key, range2.key});
    ASSERT_FALSE(err3);
    ASSERT_EQ(got.size(), 2);
    ASSERT_EQ(got[0].name, "test");
    ASSERT_FALSE(got[0].key.length() == 0);
    ASSERT_EQ(got[0].time_range.start, synnax::TimeStamp(0));
    ASSERT_EQ(got[0].time_range.end, synnax::TimeStamp(100));
    ASSERT_EQ(got[1].name, "test2");
    ASSERT_FALSE(got[1].key.length() == 0);
    ASSERT_EQ(got[1].time_range.start, synnax::TimeStamp(0));
    ASSERT_EQ(got[1].time_range.end, synnax::TimeStamp(100));
}


/// @brief it should set a key-value pair on the range.
TEST(RangerTests, testSet) {
    auto client = synnax::Client(cfg);
    auto [range, err] = client.ranges.create(
            "test",
            synnax::TimeRange(
                    synnax::TimeStamp(0),
                    synnax::TimeStamp(100)
            )
    );
    ASSERT_FALSE(err);
    err = range.kv.set("test", "test");
    ASSERT_FALSE(err);
}

/// @brief it should get a key-value pair on the range.
TEST(RangerTests, testGet) {
    auto client = synnax::Client(cfg);
    auto [range, err] = client.ranges.create(
            "test",
            synnax::TimeRange(
                    synnax::TimeStamp(0),
                    synnax::TimeStamp(100)
            )
    );
    ASSERT_FALSE(err);
    err = range.kv.set("test", "test");
    ASSERT_FALSE(err);
    auto [val, err2] = range.kv.get("test");
    ASSERT_FALSE(err2);
    ASSERT_EQ(val, "test");
}

/// @brief it should delete a key-value pair on the range.
TEST(RangerTests, testDelete) {
    auto client = synnax::Client(cfg);
    auto [range, err] = client.ranges.create(
            "test",
            synnax::TimeRange(
                    synnax::TimeStamp(0),
                    synnax::TimeStamp(10 * synnax::SECOND)
            )
    );
    ASSERT_FALSE(err);
    err = range.kv.set("test", "test");
    ASSERT_FALSE(err);
    err = range.kv.del("test");
    ASSERT_FALSE(err);
    auto [val, err2] = range.kv.get("test");
    ASSERT_TRUE(err2);
    ASSERT_EQ(val, "");
}