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

/// internal
#include "synnax/synnax.h"

const synnax::Config cfg = {
        .host =  "localhost",
        .port =  9090,
        .secure =  false,
        .username =  "synnax",
        .password =  "seldon"
};

/// @brief it should create a rate based channel and assign it a non-zero key.
TEST(ChannelTests, testCreate) {
    auto client = synnax::Client(cfg);
    auto [channel, err] = client.channels.create(
            "test",
            synnax::FLOAT64,
            1 * synnax::HZ
    );
    ASSERT_FALSE(err);
    ASSERT_EQ(channel.name, "test");
    ASSERT_FALSE(channel.key == 0);
}

/// @brief it should return a validation error when an index channel has the
/// wrong data type.
TEST(ChannelTests, testCreateValidation) {
    auto client = synnax::Client(cfg);
    auto [channel, err] = client.channels.create(
            "validation",
            synnax::FLOAT64,
            0,
            true
    );
    ASSERT_TRUE(err);
    ASSERT_EQ(err.type, synnax::VALIDATION_ERROR);
}

/// @brief it should create an index based channel and assign it a non-zero key.
TEST(ChannelTests, testCreateIndex) {
    auto client = synnax::Client(cfg);
    auto [index, err] = client.channels.create(
            "test",
            synnax::FLOAT64,
            0,
            true
    );
    ASSERT_FALSE(err);
    auto [indexed, err2] = client.channels.create(
            "test",
            synnax::FLOAT64,
            index.key,
            false
    );
    ASSERT_FALSE(err2);
    ASSERT_EQ(index.name, "test");
    ASSERT_FALSE(index.key == 0);
    ASSERT_EQ(indexed.name, "test");
    ASSERT_FALSE(indexed.key == 0);
    ASSERT_EQ(indexed.index, index.key);
}

/// @brief it should create many channels and assign them all non-zero keys.
TEST(ChannelTests, testCreateMany) {
    auto client = synnax::Client(cfg);
    auto channels = std::vector<synnax::Channel>{
            {"test1", synnax::FLOAT64, 2 * synnax::HZ},
            {"test2", synnax::FLOAT64, 4 * synnax::HZ},
            {"test3", synnax::FLOAT64, 8 * synnax::HZ}
    };
    ASSERT_TRUE(client.channels.create(channels).ok());
    ASSERT_EQ(channels.size(), 3);
    for (auto &channel: channels) ASSERT_FALSE(channel.key == 0);
}

/// @brief it should retrieve a channel by key.
TEST(ChannelTest, testRetrieve) {
    auto client = synnax::Client(cfg);
    auto [channel, err] = client.channels.create(
            "test",
            synnax::FLOAT64,
            synnax::Rate(1)
    );
    ASSERT_FALSE(err);
    auto [retrieved, err2] = client.channels.retrieve(channel.key);
    ASSERT_FALSE(err2);
    ASSERT_EQ(channel.name, retrieved.name);
    ASSERT_EQ(channel.key, retrieved.key);
    ASSERT_EQ(channel.data_type, retrieved.data_type);
    ASSERT_EQ(channel.rate, retrieved.rate);
    ASSERT_EQ(channel.is_index, retrieved.is_index);
    ASSERT_EQ(channel.leaseholder, retrieved.leaseholder);
    ASSERT_EQ(channel.index, retrieved.index);
}

/// @brief it should return a query error when the channel cannot be found.
TEST(ChannelTest, testRetrieveNotFound) {
    auto client = synnax::Client(cfg);
    auto [retrieved, err] = client.channels.retrieve(0);
    ASSERT_TRUE(err);
    ASSERT_EQ(err.type, synnax::QUERY_ERROR);
}

/// @brief it should retrieve many channels by their key.
TEST(ChannelTest, testRetrieveMany) {
    auto client = synnax::Client(cfg);
    auto channels = std::vector<synnax::Channel>{
            {"test1", synnax::FLOAT64, 5 * synnax::HZ},
            {"test2", synnax::FLOAT64, 10 * synnax::HZ},
            {"test3", synnax::FLOAT64, 20 * synnax::HZ}
    };
    ASSERT_TRUE(client.channels.create(channels).ok());
    auto [retrieved, exc] = client.channels.retrieve(
            std::vector<ChannelKey>{channels[0].key, channels[1].key, channels[2].key}
    );
    ASSERT_FALSE(exc);
    ASSERT_EQ(channels.size(), retrieved.size());
    for (auto &channel: channels) {
        auto found = false;
        for (auto &r: retrieved) {
            if (r.key == channel.key) {
                found = true;
                ASSERT_EQ(channel.name, r.name);
                ASSERT_EQ(channel.key, r.key);
                ASSERT_EQ(channel.data_type, r.data_type);
                ASSERT_EQ(channel.rate, r.rate);
                ASSERT_EQ(channel.is_index, r.is_index);
                ASSERT_EQ(channel.leaseholder, r.leaseholder);
                ASSERT_EQ(channel.index, r.index);
            }
        }
        ASSERT_TRUE(found);
    }
}