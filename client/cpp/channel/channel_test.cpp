// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <string>

#include <include/gtest/gtest.h>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::channel {
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
    auto ch = Channel{
        .name = name,
        .data_type = x::telem::FLOAT64_T,
        .is_virtual = true
    };
    const auto client = new_test_client();
    ASSERT_NIL(client.channels.create(ch));
    ASSERT_EQ(ch.name, name);
    ASSERT_TRUE(ch.is_virtual);
    ASSERT_FALSE(ch.key == 0);
}

/// @brief it should create many channels and assign them all non-zero keys.
TEST(TestChannel, testCreateMany) {
    const auto client = new_test_client();
    std::vector<Channel> channels;
    channels.push_back(
        Channel{
            .name = make_unique_channel_name("test1"),
            .data_type = x::telem::FLOAT64_T,
            .is_virtual = true,
        }
    );
    channels.push_back(
        Channel{
            .name = make_unique_channel_name("test2"),
            .data_type = x::telem::FLOAT64_T,
            .is_virtual = true,
        }
    );
    channels.push_back(
        Channel{
            .name = make_unique_channel_name("test3"),
            .data_type = x::telem::FLOAT64_T,
            .is_virtual = true,
        }
    );
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
    const auto [ch, err] = client.channels.retrieve("my_definitely_not_found");
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    ASSERT_EQ(
        err.data,
        R"(Found no channel where name_patterns = ["my_definitely_not_found"])"
    );
}

/// @brief it should reject a name pattern that matches more than one channel.
TEST(TestChannel, testRetrieveByNameMultipleFound) {
    const auto client = new_test_client();
    const auto prefix = make_unique_channel_name("retrieve_by_name_multiple");
    ASSERT_NIL_P(client.channels.create(prefix + "_a", x::telem::FLOAT64_T, true));
    ASSERT_NIL_P(client.channels.create(prefix + "_b", x::telem::FLOAT64_T, true));
    ASSERT_OCCURRED_AS_P(
        client.channels.retrieve(prefix + "_.*"),
        x::errors::MULTIPLE_RESULTS
    );
}

/// @brief it should retrieve many channels by their key.
TEST(TestChannel, testRetrieveMany) {
    auto client = new_test_client();
    std::vector<Channel> channels;
    channels.push_back(
        Channel{
            .name = make_unique_channel_name("retrieve_many_1"),
            .data_type = x::telem::FLOAT64_T,
            .is_virtual = true,
        }
    );
    channels.push_back(
        Channel{
            .name = make_unique_channel_name("retrieve_many_2"),
            .data_type = x::telem::FLOAT64_T,
            .is_virtual = true,
        }
    );
    channels.push_back(
        Channel{
            .name = make_unique_channel_name("retrieve_many_3"),
            .data_type = x::telem::FLOAT64_T,
            .is_virtual = true,
        }
    );
    ASSERT_NIL(client.channels.create(channels));
    auto retrieved = ASSERT_NIL_P(
        client.channels.retrieve(keys_from_channels(channels))
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

/// @brief it should return a not found error that names every missing key.
TEST(TestChannel, testRetrieveManyNotFound) {
    const auto client = new_test_client();
    const auto ch = create_virtual_channel(client);
    const auto [channels, err] = client.channels.retrieve(
        std::vector<Key>{1, ch.key, 3}
    );
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    ASSERT_EQ(err.data, "Found no channel where key in [1, 3]");
    ASSERT_TRUE(channels.empty());
}

/// @brief it should return the channels in the order of the requested keys.
TEST(TestChannel, testRetrieveManyOrder) {
    const auto client = new_test_client();
    const auto a = create_virtual_channel(client);
    const auto b = create_virtual_channel(client);
    const auto c = create_virtual_channel(client);
    const auto channels = ASSERT_NIL_P(
        client.channels.retrieve(std::vector<Key>{c.key, a.key, b.key})
    );
    ASSERT_EQ(keys_from_channels(channels), std::vector<Key>({c.key, a.key, b.key}));
}

/// @brief it should treat a name that is not a valid channel name as a pattern.
TEST(TestChannel, testRetrieveManyByNamePattern) {
    const auto client = new_test_client();
    const auto prefix = make_unique_channel_name("pattern");
    const auto a = ASSERT_NIL_P(
        client.channels.create(prefix + "_a", x::telem::FLOAT32_T, true)
    );
    const auto b = ASSERT_NIL_P(
        client.channels.create(prefix + "_b", x::telem::FLOAT32_T, true)
    );
    auto channels = ASSERT_NIL_P(
        client.channels.retrieve(std::vector<std::string>{prefix + "_.*"})
    );
    auto keys = keys_from_channels(channels);
    std::ranges::sort(keys);
    ASSERT_EQ(keys, std::vector<Key>({a.key, b.key}));
}

/// @brief it should match a valid channel name exactly.
TEST(TestChannel, testRetrieveManyByNameExact) {
    const auto client = new_test_client();
    const auto a = create_virtual_channel(client);
    const auto channels = ASSERT_NIL_P(
        client.channels.retrieve(std::vector<std::string>{a.name, a.name + "_missing"})
    );
    ASSERT_EQ(keys_from_channels(channels), std::vector<Key>({a.key}));
}

/// @brief it should retrieve the channels that a where selects.
TEST(TestChannel, testRetrieveWhere) {
    const auto client = new_test_client();
    const auto a = create_virtual_channel(client, x::telem::FLOAT64_T);
    const auto b = create_virtual_channel(client, x::telem::FLOAT64_T);
    const auto c = create_virtual_channel(client, x::telem::UINT8_T);
    const auto channels = ASSERT_NIL_P(client.channels.retrieve(
        RetrieveRequest{
            .where = Where{
                .or_ =
                    std::vector<Where>{
                        Where{.key = x::filter::Set<Key>{.eq = a.key}},
                        Where{.name = x::filter::String{.eq = c.name}},
                    },
                .data_type = x::filter::Set<x::telem::DataType>{
                    .eq = x::telem::FLOAT64_T,
                },
            },
        }
    ));
    ASSERT_EQ(keys_from_channels(channels), std::vector<Key>({a.key}));
    const auto limited = ASSERT_NIL_P(client.channels.retrieve(
        RetrieveRequest{
            .where =
                Where{
                    .key = x::filter::Set<
                        Key>{.in = std::vector<Key>{a.key, b.key, c.key}},
                },
            .limit = 2,
        }
    ));
    ASSERT_EQ(limited.size(), 2);
}

/// @brief it should retrieve the channels that match every condition.
TEST(TestChannel, testRetrieveConditions) {
    const auto client = new_test_client();
    const auto a = create_virtual_channel(client, x::telem::FLOAT64_T);
    const auto b = create_virtual_channel(client, x::telem::FLOAT64_T);
    const auto c = create_virtual_channel(client, x::telem::UINT8_T);
    const auto channels = ASSERT_NIL_P(client.channels.retrieve(
        channel::key.in({a.key, b.key, c.key}),
        channel::data_type.eq(x::telem::FLOAT64_T),
        channel::virtual_.eq(true),
        !channel::key.eq(b.key)
    ));
    ASSERT_EQ(keys_from_channels(channels), std::vector<Key>({a.key}));
}

/// @brief it should apply the limit and offset options.
TEST(TestChannel, testRetrieveLimitOffset) {
    const auto client = new_test_client();
    const auto a = create_virtual_channel(client);
    const auto b = create_virtual_channel(client);
    const auto c = create_virtual_channel(client);
    const auto keys = std::vector<Key>{a.key, b.key, c.key};
    const auto all = ASSERT_NIL_P(client.channels.retrieve(channel::key.in(keys)));
    const auto page = ASSERT_NIL_P(client.channels.retrieve(
        channel::key.in(keys),
        channel::limit(1),
        channel::offset(1)
    ));
    ASSERT_EQ(keys_from_channels(page), std::vector<Key>({all[1].key}));
}

/// @brief it should reject one when no channel matches.
TEST(TestChannel, testRetrieveOneNotFound) {
    const auto client = new_test_client();
    const auto [ch, err] = client.channels.retrieve(
        channel::name.eq("not_a_channel_name_in_any_cluster"),
        channel::one()
    );
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    ASSERT_EQ(
        err.data,
        R"(Found no channel where name = "not_a_channel_name_in_any_cluster")"
    );
}

/// @brief it should convert a channel key to an ontology ID
TEST(TestChannel, testOntologyId) {
    constexpr Key key = 42;
    const auto id = ontology_id(key);
    ASSERT_EQ(id.type, "channel");
    ASSERT_EQ(id.key, "42");
}

}
