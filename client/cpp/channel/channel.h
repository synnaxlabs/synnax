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

#include "client/cpp/channel/builder.gen.h"
#include "client/cpp/channel/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/api/channel/pb/channel.pb.h"
#include "core/pkg/transport/grpc/channel/channel.pb.h"

namespace synnax::channel {
/// @brief freighter retrieve transport.
using RetrieveClient = freighter::
    UnaryClient<grpc::channel::RetrieveRequest, ::api::channel::pb::RetrieveResponse>;

/// @brief freighter create transport.
using CreateClient = freighter::
    UnaryClient<grpc::channel::CreateRequest, grpc::channel::CreateResponse>;

/// @brief creates a vector of channel keys from a variadic list of channels.
template<typename... Channels>
std::vector<Key> keys_from_channels(const Channels &...channels) {
    std::vector<Key> keys;
    keys.reserve(sizeof...(channels));
    (keys.push_back(channels.key), ...);
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
map_channel_keys(const std::vector<Channel> &channels) {
    std::unordered_map<Key, Channel> map;
    map.reserve(channels.size());
    for (const auto &channel: channels)
        map[channel.key] = channel;
    return map;
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

    /// @brief Retrieves the channel with the given key, with each include option in
    /// includes applied.
    /// @returns the channel, or a NOT_FOUND error when no channel has the key.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<Channel, x::errors::Error>
    retrieve(const Key &key, Includes... includes) const {
        return this->retrieve_one(
            x::filter::request<Where, RetrieveRequest>(
                channel::key.eq(key),
                includes...
            )
        );
    }

    /// @brief Retrieves the channel with the given name, with each include option in
    /// includes applied. A name that is not a valid channel name is a regular
    /// expression, anchored when unanchored.
    /// @returns the channel, or an error when no channel or more than one channel has
    /// the name.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<Channel, x::errors::Error>
    retrieve(const std::string &name, Includes... includes) const {
        return this->retrieve_one(
            x::filter::request<Where, RetrieveRequest>(
                channel::name_patterns.has(name),
                includes...
            )
        );
    }

    /// @brief Retrieves the channels with the given keys, with each include option in
    /// includes applied. The channels come in the order of keys, and a key that no
    /// channel has is a NOT_FOUND error.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error>
    retrieve(const std::vector<Key> &keys, Includes... includes) const {
        return this->retrieve_keys(
            x::filter::request<Where, RetrieveRequest>(
                channel::key.in(keys),
                includes...
            ),
            keys
        );
    }

    /// @brief Retrieves the channels with the given names, with each include option in
    /// includes applied. A valid channel name matches exactly, and any other name is a
    /// regular expression, anchored when unanchored.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error>
    retrieve(const std::vector<std::string> &names, Includes... includes) const {
        return this->retrieve(
            x::filter::request<Where, RetrieveRequest>(
                channel::name_patterns.has_any(names),
                includes...
            )
        );
    }

    /// @brief retrieves the channels that req selects.
    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error>
    retrieve(const RetrieveRequest &req) const;

    /// @brief Retrieves the channels that match every condition in parts, with each
    /// option in parts applied.
    /// @returns the channels, or one channel when parts end with channel::one(). One
    /// channel is an error when no channel or more than one channel matches.
    template<typename... Parts>
        requires x::filter::parts<Where, RetrieveRequest, Parts...>
    [[nodiscard]] auto retrieve(Parts... parts) const {
        const auto req = x::filter::request<Where, RetrieveRequest>(
            std::move(parts)...
        );
        if constexpr (x::filter::returns_one<Parts...>)
            return this->retrieve_one(req);
        else
            return this->retrieve(req);
    }

private:
    [[nodiscard]] std::pair<Channel, x::errors::Error>
    retrieve_one(const RetrieveRequest &req) const;

    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error>
    retrieve_keys(const RetrieveRequest &req, const std::vector<Key> &keys) const;

    /// @brief transport for retrieving channels.
    std::shared_ptr<RetrieveClient> retrieve_client;
    /// @brief transport for creating channels.
    std::shared_ptr<CreateClient> create_client;
};
}
