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

std::mt19937 gen_rand = random_generator(std::move("Channel Tests"));

/// @brief it should create a rate based channel and assign it a non-zero key.
TEST(TestChannel, testCreate) {
    const auto client = new_test_client();
    const auto name = make_unique_channel_name("test");
    const auto channel = ASSERT_NIL_P(
        client.channels.create(name, x::telem::FLOAT64_T, true)
    );
    ASSERT_EQ(channel.name, name);
    ASSERT_FALSE(channel.key == 0);
}

/// @brief it should return a validation error when an index channel has the
/// wrong data type.
TEST(TestChannel, testCreateValidation) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.channels.create(
            make_unique_channel_name("validation"),
            x::telem::FLOAT64_T,
            0,
            true
        ),
        x::errors::VALIDATION
    );
}

/// @brief it should create an index based channel and assign it a non-zero key.
TEST(TestChannel, testCreateIndex) {
    auto client = new_test_client();
    const auto index_name = make_unique_channel_name("test_index");
    auto index = ASSERT_NIL_P(
        client.channels.create(index_name, x::telem::TIMESTAMP_T, 0, true)
    );
    const auto indexed_name = make_unique_channel_name("test_indexed");
    auto indexed = ASSERT_NIL_P(
        client.channels.create(indexed_name, x::telem::FLOAT64_T, index.key, false)
    );
    ASSERT_EQ(index.name, index_name);
    ASSERT_FALSE(index.key == 0);
    ASSERT_EQ(indexed.name, indexed_name);
    ASSERT_FALSE(indexed.key == 0);
    ASSERT_EQ(indexed.index, index.key);
}

/// @brief it should create a virtual channel and assign it a non-zero key.
TEST(TestChannel, testCreateVirtual) {
    const auto name = make_unique_channel_name("virtual");
    auto ch = synnax::channel::Channel(name, x::telem::FLOAT64_T, true);
    const auto client = new_test_client();
    ASSERT_NIL(client.channels.create(ch));
    ASSERT_EQ(ch.name, name);
    ASSERT_TRUE(ch.is_virtual);
    ASSERT_FALSE(ch.key == 0);
}

/// @brief it should create many channels and assign them all non-zero keys.
TEST(TestChannel, testCreateMany) {
    const auto client = new_test_client();
    auto channels = std::vector<synnax::channel::Channel>{
        {
            make_unique_channel_name("test1"),
            x::telem::FLOAT64_T,
            true,
        },
        {
            make_unique_channel_name("test2"),
            x::telem::FLOAT64_T,
            true,
        },
        {make_unique_channel_name("test3"), x::telem::FLOAT64_T, true},
    };
    ASSERT_TRUE(client.channels.create(channels).ok());
    ASSERT_EQ(channels.size(), 3);
    for (const auto &ch: channels)
        ASSERT_FALSE(ch.key == 0);
}

/// @brief it should retrieve a channel by key.
TEST(TestChannel, testRetrieve) {
    auto client = new_test_client();
    auto channel = ASSERT_NIL_P(client.channels.create(
        make_unique_channel_name("retrieve"),
        x::telem::FLOAT64_T,
        true
    ));
    auto retrieved = ASSERT_NIL_P(client.channels.retrieve(channel.key));
    ASSERT_EQ(channel.name, retrieved.name);
    ASSERT_EQ(channel.key, retrieved.key);
    ASSERT_EQ(channel.data_type, retrieved.data_type);
    ASSERT_EQ(channel.is_virtual, retrieved.is_virtual);
    ASSERT_EQ(channel.is_index, retrieved.is_index);
    ASSERT_EQ(channel.leaseholder, retrieved.leaseholder);
    ASSERT_EQ(channel.index, retrieved.index);
}

/// @brief it should return a query error when the channel cannot be found.
TEST(TestChannel, testRetrieveNotFound) {
    const auto client = new_test_client();
    auto [retrieved, err] = client.channels.retrieve(22);
    ASSERT_TRUE(err) << err.message();
    ASSERT_TRUE(err.matches(x::errors::QUERY));
}

/// @brief it should correctly retrieve a channel by name.
TEST(TestChannel, testRetrieveByName) {
    auto client = new_test_client();
    const auto name = make_unique_channel_name("retrieve_by_name_test");
    auto channel = ASSERT_NIL_P(
        client.channels.create(name, x::telem::FLOAT64_T, true)
    );
    auto retrieved = ASSERT_NIL_P(client.channels.retrieve(name));
    ASSERT_EQ(channel.name, retrieved.name);
    ASSERT_EQ(channel.key, retrieved.key);
    ASSERT_EQ(channel.data_type, retrieved.data_type);
    ASSERT_EQ(channel.is_index, retrieved.is_index);
    ASSERT_EQ(channel.is_virtual, retrieved.is_virtual);
    ASSERT_EQ(channel.leaseholder, retrieved.leaseholder);
    ASSERT_EQ(channel.index, retrieved.index);
}

/// @brief it should return the correct error when a channel cannot be found by name.
TEST(TestChannel, testRetrieveByNameNotFound) {
    const auto client = new_test_client();
    auto [retrieved, err] = client.channels.retrieve("my_definitely_not_found");
    ASSERT_TRUE(err) << err.message();
    ASSERT_EQ(err, x::errors::NOT_FOUND);
}

/// @brief it should retrieve many channels by their key.
TEST(TestChannel, testRetrieveMany) {
    auto client = new_test_client();
    auto channels = std::vector<synnax::channel::Channel>{
        {
            make_unique_channel_name("retrieve_many_1"),
            x::telem::FLOAT64_T,
            true,
        },
        {
            make_unique_channel_name("retrieve_many_2"),
            x::telem::FLOAT64_T,
            true,
        },
        {make_unique_channel_name("retrieve_many_3"), x::telem::FLOAT64_T, true},
    };
    ASSERT_NIL(client.channels.create(channels));
    auto retrieved = ASSERT_NIL_P(
        client.channels.retrieve(synnax::channel::keys_from_channels(channels))
    );
    ASSERT_EQ(channels.size(), retrieved.size());
    for (auto &channel: channels) {
        auto found = false;
        for (auto &r: retrieved) {
            if (r.key == channel.key) {
                found = true;
                ASSERT_EQ(channel.name, r.name);
                ASSERT_EQ(channel.key, r.key);
                ASSERT_EQ(channel.data_type, r.data_type);
                ASSERT_EQ(channel.is_virtual, r.is_virtual);
                ASSERT_EQ(channel.is_index, r.is_index);
                ASSERT_EQ(channel.leaseholder, r.leaseholder);
                ASSERT_EQ(channel.index, r.index);
            }
        }
        ASSERT_TRUE(found);
    }
}

/// @brief it should return the correct error when a channel cannot be found
/// by key multiple retrieval.
TEST(TestChannel, testRetrieveManyNotFound) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.channels.retrieve(std::vector<synnax::channel::Key>{1, 2, 3}),
        x::errors::NOT_FOUND
    );
}

/// @brief it should convert a channel key to an ontology ID
TEST(TestChannel, testOntologyId) {
    constexpr synnax::channel::Key key = 42;
    const auto id = synnax::channel::ontology_id(key);
    ASSERT_EQ(id.type, "channel");
    ASSERT_EQ(id.key, "42");
}

/// @brief it should convert multiple channel keys to ontology IDs
TEST(TestChannel, testOntologyIds) {
    const std::vector<synnax::channel::Key> keys = {1, 2, 3};
    const auto ids = synnax::channel::ontology_ids(keys);
    ASSERT_EQ(ids.size(), 3);
    ASSERT_EQ(ids[0].type, "channel");
    ASSERT_EQ(ids[0].key, "1");
    ASSERT_EQ(ids[1].type, "channel");
    ASSERT_EQ(ids[1].key, "2");
    ASSERT_EQ(ids[2].type, "channel");
    ASSERT_EQ(ids[2].key, "3");
}

/// @brief it should return empty vector for empty input
TEST(TestChannel, testOntologyIdsEmpty) {
    const std::vector<synnax::channel::Key> keys;
    const auto ids = synnax::channel::ontology_ids(keys);
    ASSERT_TRUE(ids.empty());
}
