// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/ontology/id.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/api/channel/pb/channel.pb.h"
#include "core/pkg/api/grpc/channel/channel.pb.h"

namespace synnax::channel {
/// @brief freighter retrieve transport.
using RetrieveClient = freighter::
    UnaryClient<grpc::channel::RetrieveRequest, grpc::channel::RetrieveResponse>;

/// @brief freighter create transport.
using CreateClient = freighter::
    UnaryClient<grpc::channel::CreateRequest, grpc::channel::CreateResponse>;

/// @brief creates a vector of channel keys from a variadic list of channels.
template<typename... Channels>
std::vector<Key> keys_from_channels(const Channels &...channels) {
    std::vector<Key> keys;
    keys.reserve(sizeof...(channels));
    ((keys.push_back(channels.key)), ...);
    return keys;
}

/// @brief creates a vector of channel keys from a vector of channels.
inline std::vector<Key> keys_from_channels(const std::vector<Channel> &channels) {
    std::vector<Key> keys;
    keys.reserve(channels.size());
    for (const auto &channel: channels)
        keys.push_back(channel.key);
    return keys;
}

inline std::unordered_map<Key, Channel>
map_channel_Keys(const std::vector<Channel> &channels) {
    std::unordered_map<Key, Channel> map;
    map.reserve(channels.size());
    for (const auto &channel: channels)
        map[channel.key] = channel;
    return map;
}

/// @brief Converts a channel key to an ontology ID.
/// @param key The channel key.
/// @returns An ontology ID with type "channel" and the given key.
inline ontology::ID ontology_id(Key key) {
    return ontology::ID("channel", std::to_string(key));
}

/// @brief Converts a vector of channel keys to a vector of ontology IDs.
/// @param keys The channel keys.
/// @returns A vector of ontology IDs.
inline std::vector<ontology::ID> ontology_ids(const std::vector<Key> &keys) {
    std::vector<ontology::ID> ids;
    ids.reserve(keys.size());
    for (const auto &key: keys)
        ids.push_back(ontology_id(key));
    return ids;
}

/// @brief ChannelClient for creating and retrieving channels from a Synnax cluster.
class Client {
public:
    Client() = default;

    Client(
        std::shared_ptr<RetrieveClient> retrieve_client,
        std::shared_ptr<CreateClient> create_client
    ):
        retrieve_client(std::move(retrieve_client)),
        create_client(std::move(create_client)) {}

    /// @brief Creates the given channel in the Synnax cluster.
    /// @param channel The channel to create.
    /// @modifies channel Assigns a unique key to the channel.
    /// @returns an error where ok() is false if the channel could not be created.
    /// Use err.message() to get the error message or err.type to get the error
    /// type.
    [[nodiscard]] x::errors::Error create(Channel &channel) const;

    /// @brief creates the given channels in the Synnax cluster.
    /// @details More efficient than calling create on each channel individually,
    /// and also provides atomicity guarantees.
    /// @modifies channels Assigns a unique key to each channel.
    /// @returns an error where ok() is false if the channels could not be created.
    /// Use err.message() to get the error message or err.type to get the error
    /// type.
    [[nodiscard]] x::errors::Error create(std::vector<Channel> &channels) const;

    /// @brief creates a new index or indexed channel.
    /// @param name a human-readable name for the channel.
    /// @param data_type the data type of the channel.
    /// @param index the index of the channel.
    /// @param is_index whether the channel is an index channel.
    /// @returns a pair containing the created channel and an error where ok() is
    /// false if the channel could not be created. In the case of an error, the
    /// returned channel will be invalid. Use err.message() to get the error message
    /// or err.type to get the error type.
    [[nodiscard]] std::pair<Channel, x::errors::Error> create(
        const std::string &name,
        const x::telem::DataType &data_type,
        Key index,
        bool is_index = false
    ) const;

    [[nodiscard]] std::pair<Channel, x::errors::Error> create(
        const std::string &name,
        const x::telem::DataType &data_type,
        bool is_virtual = true
    ) const;

    /// @brief retrieves a channel with the given name.
    /// @param name the name of the channel to retrieve.
    /// @throws QueryError if the channel does not exist or multiple channels with
    /// the same name exist.
    /// @returns the retrieved channel.
    /// @returns a pair containing the retrieved channel and an error where ok() is
    /// false if the channel could not be retrieved. In the case of an error, the
    /// returned channel will be invalid. Use err.message() to get the error message
    /// or err.type to get the error type.
    [[nodiscard]] std::pair<Channel, x::errors::Error>
    retrieve(const std::string &name) const;

    /// @brief retrieves a channel with the given key.
    /// @param key the key of the channel to retrieve.
    /// @throws QueryError if the channel does not exist.
    /// @returns the retrieved channel.
    /// @returns a pair containing the retrieved channel and an error where ok() is
    /// false if the channel could not be retrieved. In the case of an error, the
    /// returned channel will be invalid. Use err.message() to get the error message
    /// or err.type to get the error type.
    [[nodiscard]] std::pair<Channel, x::errors::Error>
    retrieve(std::uint32_t key) const;

    /// @brief retrieves channels with the given names.
    /// @param names the names of the channels to retrieve.
    /// @returns all channels matching the given names. If a channel matching a
    /// name, does not exist, it will not be in the returned vector.
    /// @returns a pair containing the retrieved channels and an error where ok() is
    /// false if the channels could not be retrieved. In the case of an error, the
    /// returned channels will be invalid. Use err.message() to get the error
    /// message
    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error>
    retrieve(const std::vector<std::string> &names) const;

    /// @brief retrieves channels with the given keys.
    /// @param keys the keys of the channels to retrieve.
    /// @returns all channels matching the given keys. If a channel matching a key
    /// does not exist, it will not be in the returned vector.
    /// @returns a pair containing the retrieved channels and an error where ok() is
    /// false if the channels could not be retrieved. In the case of an error, the
    /// returned channels will be invalid. Use err.message() to get the error
    /// message.
    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error>
    retrieve(const std::vector<Key> &keys) const;

private:
    /// @brief transport for retrieving channels.
    std::shared_ptr<RetrieveClient> retrieve_client;
    /// @brief transport for creating channels.
    std::shared_ptr<CreateClient> create_client;
};
}
