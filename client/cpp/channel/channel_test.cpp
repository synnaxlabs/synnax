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

#include "client/cpp/channel/json.gen.h"
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
    ASSERT_NIL(client.channels.create(channels));
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
    ASSERT_OCCURRED_AS_P(client.channels.retrieve(22), x::errors::QUERY);
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
    ASSERT_OCCURRED_AS_P(
        client.channels.retrieve("my_definitely_not_found"),
        x::errors::NOT_FOUND
    );
}

/// @brief it should read a name pattern as an exact name and find no channel.
TEST(TestChannel, testRetrieveByNamePattern) {
    const auto client = new_test_client();
    const auto prefix = make_unique_channel_name("retrieve_by_name_pattern");
    ASSERT_NIL_P(client.channels.create(prefix + "_a", x::telem::FLOAT64_T, true));
    ASSERT_NIL_P(client.channels.create(prefix + "_b", x::telem::FLOAT64_T, true));
    ASSERT_OCCURRED_AS_P(
        client.channels.retrieve(prefix + "_.*"),
        x::errors::NOT_FOUND
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
    ASSERT_EQ(err.data, "channels matching keys 1, 3 not found.");
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

/// @brief it should never read a name as a pattern.
TEST(TestChannel, testRetrieveManyByNameNotPattern) {
    const auto client = new_test_client();
    const auto prefix = make_unique_channel_name("pattern");
    const auto a = ASSERT_NIL_P(
        client.channels.create(prefix + "_a", x::telem::FLOAT32_T, true)
    );
    ASSERT_NIL_P(client.channels.create(prefix + "_b", x::telem::FLOAT32_T, true));
    const auto channels = ASSERT_NIL_P(
        client.channels.retrieve(std::vector<std::string>{a.name, prefix + "_.*"})
    );
    ASSERT_EQ(keys_from_channels(channels), std::vector<Key>({a.key}));
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
        (Where{.key = a.key} || Where{.name = c.name}) &&
        Where{.data_type = x::telem::FLOAT64_T}
    ));
    ASSERT_EQ(keys_from_channels(channels), std::vector<Key>({a.key}));
    const auto limited = ASSERT_NIL_P(client.channels.retrieve(
        Where{.key = std::vector<Key>{a.key, b.key, c.key}},
        {.limit = 2}
    ));
    ASSERT_EQ(limited.size(), 2);
}

/// @brief it should page a retrieve of channels by name.
TEST(TestChannel, testRetrieveManyByNamePage) {
    const auto client = new_test_client();
    const auto a = create_virtual_channel(client);
    const auto b = create_virtual_channel(client);
    const auto channels = ASSERT_NIL_P(client.channels.retrieve(
        std::vector<std::string>{a.name, b.name},
        {.offset = 1}
    ));
    ASSERT_EQ(channels.size(), 1);
}

/// @brief it should retrieve the one channel that a filter matches.
TEST(TestChannel, testRetrieveOneWhere) {
    const auto client = new_test_client();
    const auto a = create_virtual_channel(client);
    const auto got = ASSERT_NIL_P(
        client.channels.retrieve({.name = a.name}, x::filter::one)
    );
    ASSERT_EQ(got.key, a.key);
}

/// @brief it should name the filter in the error when no channel matches it.
TEST(TestChannel, testRetrieveOneWhereNotFound) {
    const auto client = new_test_client();
    const auto name = make_unique_channel_name("one_none");
    const auto [_, err] = client.channels.retrieve({.name = name}, x::filter::one);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    ASSERT_EQ(err.data, "Found no channel where name = \"" + name + "\"");
}

/// @brief it should return an error when more than one channel matches a filter.
TEST(TestChannel, testRetrieveOneWhereMultipleFound) {
    const auto client = new_test_client();
    const auto a = create_virtual_channel(client);
    const auto b = create_virtual_channel(client);
    ASSERT_OCCURRED_AS_P(
        client.channels.retrieve({.name = {a.name, b.name}}, x::filter::one),
        x::errors::MULTIPLE_RESULTS
    );
}

/// @brief it should reject a limit or an offset on a retrieve of one channel.
TEST(TestChannel, testRetrieveRejectsPage) {
    const auto client = new_test_client();
    for (const auto &options:
         {RetrieveOptions{.limit = 1}, RetrieveOptions{.offset = 1}}) {
        for (const auto &err:
             {client.channels.retrieve(Key(1), options).second,
              client.channels.retrieve("channel", options).second,
              client.channels.retrieve({.name = "channel"}, x::filter::one, options)
                  .second}) {
            ASSERT_MATCHES(err, x::errors::VALIDATION);
            ASSERT_EQ(
                err.data,
                "a query for one record takes no limit or offset, since a page could "
                "hide a second match"
            );
        }
    }
}

/// @brief it should convert a channel key to a resource ID
TEST(TestChannel, testResourceID) {
    constexpr Key key = 42;
    const auto id = resource_id(key);
    ASSERT_EQ(id.type, "channel");
    ASSERT_EQ(id.key, "42");
}

/// @brief it should fall back to exclusive concurrency when the JSON omits it.
TEST(TestChannel, testParseMissingConcurrency) {
    x::json::Parser parser(
        x::json::json{
            {"name", "sensor"},
            {"data_type", "float64"},
        }
    );
    const auto ch = Channel::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(ch.concurrency, x::control::Concurrency::Exclusive);
}
}
