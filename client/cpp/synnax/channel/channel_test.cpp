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

const Synnax::Config cfg = {
        "localhost",
        9090,
        false,
        "synnax",
        "seldon"
};

/// @brief it should create a rate based channel and assign it a non-zero key.
TEST(ChannelTests, testCreate) {
    auto client = Synnax::Client(cfg);
    auto [channel, err] = client.channels.create(
            "test",
            Telem::DataType("float64"),
            Telem::Rate(1)
    );
    ASSERT_FALSE(err);
    ASSERT_EQ(channel.name, "test");
    ASSERT_FALSE(channel.key == 0);
}

/// @brief it should create an index based channel and assign it a non-zero key.
TEST(ChannelTests, testCreateIndex) {
    auto client = Synnax::Client(cfg);
    auto [index, err] = client.channels.create(
            "test",
            Telem::DataType("timestamp"),
            0,
            true
    );
    ASSERT_FALSE(err);
    auto [indexed, err2] = client.channels.create(
            "test",
            Telem::DataType("float64"),
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
    auto client = Synnax::Client(cfg);
    auto channels = std::vector<Synnax::Channel::Channel>{
            {"test1", Telem::DataType("float64"), Telem::Rate(1)},
            {"test2", Telem::DataType("float64"), Telem::Rate(1)},
            {"test3", Telem::DataType("float64"), Telem::Rate(1)},
    };
    ASSERT_TRUE(client.channels.create(channels).ok());
    ASSERT_EQ(channels.size(), 3);
    for (auto &channel: channels) ASSERT_FALSE(channel.key == 0);
}

/// @brief it should retrieve a channel by key.
TEST(ChannelTest, testRetrieve) {
    auto client = Synnax::Client(cfg);
    auto [channel, err] = client.channels.create(
            "test",
            Telem::DataType("float64"),
            Telem::Rate(1)
    );
    ASSERT_FALSE(err);
    auto [retrieved, err2] = client.channels.retrieve(channel.key);
    ASSERT_FALSE(err2);
    ASSERT_EQ(channel.name, retrieved.name);
    ASSERT_EQ(channel.key, retrieved.key);
    ASSERT_EQ(channel.data_type.value, retrieved.data_type.value);
    ASSERT_EQ(channel.rate.value, retrieved.rate.value);
    ASSERT_EQ(channel.is_index, retrieved.is_index);
    ASSERT_EQ(channel.leaseholder, retrieved.leaseholder);
    ASSERT_EQ(channel.index, retrieved.index);
}

/// @brief it should retrieve many channels by their key.
TEST(ChannelTest, testRetrieveMany) {
    auto client = Synnax::Client(cfg);
    auto channels = std::vector<Synnax::Channel::Channel>{
            {"test1", Telem::DataType("float64"), Telem::Rate(1)},
            {"test2", Telem::DataType("float64"), Telem::Rate(1)},
            {"test3", Telem::DataType("float64"), Telem::Rate(1)},
    };
    ASSERT_TRUE(client.channels.create(channels).ok());
    auto [retrieved, exc] = client.channels.retrieve(
            std::vector<Channel::Key>{channels[0].key, channels[1].key, channels[2].key}
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
                ASSERT_EQ(channel.data_type.value, r.data_type.value);
                ASSERT_EQ(channel.rate.value, r.rate.value);
                ASSERT_EQ(channel.is_index, r.is_index);
                ASSERT_EQ(channel.leaseholder, r.leaseholder);
                ASSERT_EQ(channel.index, r.index);
            }
        }
        ASSERT_TRUE(found);
    }
}