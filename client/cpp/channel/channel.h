// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "freighter/cpp/freighter.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/channel.pb.h"

namespace synnax {
/// @brief an alias for the type of channel's key.
using ChannelKey = std::uint32_t;

/// @brief freighter retrieve transport.
using ChannelRetrieveClient = freighter::
    UnaryClient<api::v1::ChannelRetrieveRequest, api::v1::ChannelRetrieveResponse>;

/// @brief freighter create transport.
using ChannelCreateClient = freighter::
    UnaryClient<api::v1::ChannelCreateRequest, api::v1::ChannelCreateResponse>;

class ChannelClient;

/// @brief A channel is a logical collection of samples emitted by or representing
/// the values of a single source, typically a sensor, actuator, or software
/// generated value. See https://docs.synnaxlabs.com/reference/concepts/channels for
/// an introduction to channels and how they work.
struct Channel {
    /// @brief A human-readable name for the channel.
    std::string name;
    /// @brief the data type of the channel.
    telem::DataType data_type;
    /// @brief the key of the channel. This is auto-assigned by the cluster on calls
    /// to create and retrieve.
    ChannelKey key = 0;
    /// @brief The key of the channel that indexes this channel. This field must be
    /// set if the channel is not an index channel.
    ChannelKey index = 0;
    /// @brief Sets whether the channel itself is an index channel.
    bool is_index = false;
    /// @brief The leaseholder of the channel.
    std::uint32_t leaseholder = 0;
    /// @brief Whether the channel is virtual. Virtual channels are not stored in
    /// the Synnax database, and are purely used for streaming and communication
    /// purposes.
    bool is_virtual = false;
    /// @brief Whether the channel is an internal channel. Internal channels are
    /// created by the DB and generally should not be interacted with unless you
    /// know what you're doing.
    bool internal = false;

    /// @brief constructs an empty, invalid channel.
    Channel() = default;

    /// @brief constructs a new index or indexed channel.
    /// @param name a human-readable name for the channel.
    /// @param data_type the data type of the channel.
    /// @param index the index of the channel.
    /// @param is_index whether the channel is an index channel.
    Channel(
        std::string name,
        telem::DataType data_type,
        ChannelKey index,
        bool is_index = false
    );

    /// @brief constructs a new virtual channel.
    /// @param name a human-readable name for the channel.
    /// @param data_type the data type of the channel.
    /// @param is_virtual whether the channel is virtual.
    Channel(std::string name, telem::DataType data_type, bool is_virtual);

    /// @brief constructs the channel from its protobuf type.
    explicit Channel(const api::v1::Channel &ch);

private:
    /// @brief binds the channel's fields to the protobuf type.
    void to_proto(api::v1::Channel *ch) const;

    friend class ChannelClient;
};

/// @brief creates a vector of channel keys from a variadic list of channels.
template<typename... Channels>
std::vector<ChannelKey> keys_from_channels(const Channels &...channels) {
    std::vector<ChannelKey> keys;
    keys.reserve(sizeof...(channels));
    ((keys.push_back(channels.key)), ...);
    return keys;
}

/// @brief creates a vector of channel keys from a vector of channels.
inline std::vector<ChannelKey>
keys_from_channels(const std::vector<Channel> &channels) {
    std::vector<ChannelKey> keys;
    keys.reserve(channels.size());
    for (const auto &channel: channels)
        keys.push_back(channel.key);
    return keys;
}

inline std::unordered_map<ChannelKey, Channel>
map_channel_Keys(const std::vector<Channel> &channels) {
    std::unordered_map<ChannelKey, Channel> map;
    map.reserve(channels.size());
    for (const auto &channel: channels)
        map[channel.key] = channel;
    return map;
}

/// @brief ChannelClient for creating and retrieving channels from a Synnax cluster.
class ChannelClient {
public:
    ChannelClient() = default;

    ChannelClient(
        std::shared_ptr<ChannelRetrieveClient> retrieve_client,
        std::shared_ptr<ChannelCreateClient> create_client
    ):
        retrieve_client(std::move(retrieve_client)),
        create_client(std::move(create_client)) {}

    /// @brief Creates the given channel in the Synnax cluster.
    /// @param channel The channel to create.
    /// @modifies channel Assigns a unique key to the channel.
    /// @returns an error where ok() is false if the channel could not be created.
    /// Use err.message() to get the error message or err.type to get the error
    /// type.
    [[nodiscard]] xerrors::Error create(Channel &channel) const;

    /// @brief creates the given channels in the Synnax cluster.
    /// @details More efficient than calling create on each channel individually,
    /// and also provides atomicity guarantees.
    /// @modifies channels Assigns a unique key to each channel.
    /// @returns an error where ok() is false if the channels could not be created.
    /// Use err.message() to get the error message or err.type to get the error
    /// type.
    [[nodiscard]] xerrors::Error create(std::vector<Channel> &channels) const;

    /// @brief creates a new index or indexed channel.
    /// @param name a human-readable name for the channel.
    /// @param data_type the data type of the channel.
    /// @param index the index of the channel.
    /// @param is_index whether the channel is an index channel.
    /// @returns a pair containing the created channel and an error where ok() is
    /// false if the channel could not be created. In the case of an error, the
    /// returned channel will be invalid. Use err.message() to get the error message
    /// or err.type to get the error type.
    [[nodiscard]] std::pair<Channel, xerrors::Error> create(
        const std::string &name,
        const telem::DataType &data_type,
        ChannelKey index,
        bool is_index = false
    ) const;

    [[nodiscard]] std::pair<Channel, xerrors::Error> create(
        const std::string &name,
        const telem::DataType &data_type,
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
    [[nodiscard]] std::pair<Channel, xerrors::Error>
    retrieve(const std::string &name) const;

    /// @brief retrieves a channel with the given key.
    /// @param key the key of the channel to retrieve.
    /// @throws QueryError if the channel does not exist.
    /// @returns the retrieved channel.
    /// @returns a pair containing the retrieved channel and an error where ok() is
    /// false if the channel could not be retrieved. In the case of an error, the
    /// returned channel will be invalid. Use err.message() to get the error message
    /// or err.type to get the error type.
    [[nodiscard]] std::pair<Channel, xerrors::Error> retrieve(std::uint32_t key) const;

    /// @brief retrieves channels with the given names.
    /// @param names the names of the channels to retrieve.
    /// @returns all channels matching the given names. If a channel matching a
    /// name, does not exist, it will not be in the returned vector.
    /// @returns a pair containing the retrieved channels and an error where ok() is
    /// false if the channels could not be retrieved. In the case of an error, the
    /// returned channels will be invalid. Use err.message() to get the error
    /// message
    [[nodiscard]] std::pair<std::vector<Channel>, xerrors::Error>
    retrieve(const std::vector<std::string> &names) const;

    /// @brief retrieves channels with the given keys.
    /// @param keys the keys of the channels to retrieve.
    /// @returns all channels matching the given keys. If a channel matching a key
    /// does not exist, it will not be in the returned vector.
    /// @returns a pair containing the retrieved channels and an error where ok() is
    /// false if the channels could not be retrieved. In the case of an error, the
    /// returned channels will be invalid. Use err.message() to get the error
    /// message.
    [[nodiscard]] std::pair<std::vector<Channel>, xerrors::Error>
    retrieve(const std::vector<ChannelKey> &keys) const;

private:
    /// @brief transport for retrieving channels.
    std::shared_ptr<ChannelRetrieveClient> retrieve_client;
    /// @brief transport for creating channels.
    std::shared_ptr<ChannelCreateClient> create_client;
};
}
