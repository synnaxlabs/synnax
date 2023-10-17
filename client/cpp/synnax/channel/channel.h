// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>
#include <utility>

/// freighter
#include "freighter/freighter.h"

/// api protos
#include "v1/channel.pb.h"

/// grpc
#include <grpcpp/grpcpp.h>

/// internal
#include "synnax/telem/telem.h"


using namespace Synnax;


namespace Synnax::Channel {

/// @brief an alias for the type of a channel's key.
typedef std::uint32_t Key;

/// @brief freighter retrieve transport.
typedef Freighter::UnaryClient<
        api::v1::ChannelRetrieveResponse,
        api::v1::ChannelRetrieveRequest
> RetrieveClient;

/// @brief freighter create transport.
typedef Freighter::UnaryClient<
        api::v1::ChannelCreateResponse,
        api::v1::ChannelCreateRequest
> CreateClient;

class Client;

/// @brief A channel is a logical collection of samples emitted by or representing the
/// values of a single source, typically a sensor, actuator, or software generated value.
/// See https:://docs.synnaxlabs.com/concepts/channels for an introduction to channels
/// and how they work.
class Channel {

public:
    /// @brief A human-readable name for the channel.
    std::string name;
    /// @brief the data type of the channel.
    Telem::DataType data_type;
    /// @brief the key of the channel. This is auto-assigned by the cluster on calls to
    /// create and retrieve.
    Key key = 0;
    /// @brief The key of the channel that indexes this channel. If this field is zero,
    /// the channel must have a non-zero rate, and is considered rate based. This this
    /// field is non-zero, the channel is considered index based, and the rate field must
    /// be zero.
    Key index = 0;
    /// @brief The sampling rate of the channel. If this parameter is non-zero, is_index
    /// must be false and index must be zero.
    Telem::Rate rate = Telem::Rate(0);
    /// @brief Sets whether the channel itself is an index channel. Index channels must
    /// cannot have a rate, and must have a data type of TIMESTAMP.
    bool is_index = false;
    /// @brief The leaseholder of the channel.
    std::uint32_t leaseholder = 0;

    /// @brief constructs an empty, invalid channel.
    Channel() = default;

    /// @brief constructs a new index or indexed channel.
    /// @param name a human-readable name for the channel.
    /// @param data_type the data type of the channel.
    /// @param index the index of the channel.
    /// @param is_index whether the channel is an index channel.
    Channel(
            const std::string &name,
            Telem::DataType data_type,
            Key index,
            bool is_index = false
    );

    /// @brief constructs a new rate based channel.
    /// @param name a human-readable name for the channel.
    /// @param data_type the data type of the channel.
    /// @param rate the rate of the channel.
    Channel(
            const std::string &name,
            Telem::DataType data_type,
            Telem::Rate rate
    );

    /// @brief constructs the channel from its protobuf type.
    explicit Channel(const api::v1::Channel &ch);

private:
    /// @brief binds the channel's fields to the protobuf type.
    void to_proto(api::v1::Channel *ch) const;

    friend class Client;
};

/// @brief Client for creating and retrieving channels from a Synnax cluster.
class Client {
public:
    Client(RetrieveClient *retrieve_client, CreateClient *create_client) :
            retrieve_client(retrieve_client),
            create_client(create_client) {}


    /// @brief creates the given channels in the Synnax cluster.
    /// @details More efficient than calling create on each channel individually, and
    /// also provides atomicity guarantees.
    /// @modifies channels Assigns a unique key to each channel.
    [[nodiscard]] Freighter::Error create(std::vector<Channel> &channels) const;

    /// @brief Creates the given channel in the Synnax cluster.
    /// @param channel The channel to create.
    /// @modifies channel Assigns a unique key to the channel.
    [[nodiscard]] Freighter::Error create(Channel &channel) const;

    /// @brief creates a new index or indexed channel.
    /// @param name a human-readable name for the channel.
    /// @param data_type the data type of the channel.
    /// @param index the index of the channel.
    /// @param is_index whether the channel is an index channel.
    /// @returns the created channel with a unique key assigned.
    [[nodiscard]] std::pair<Channel, Freighter::Error> create(
            std::string name,
            Telem::DataType data_type,
            Key index,
            bool is_index = false
    ) const;

    /// @brief creates a new rate based channel.
    /// @param name a human-readable name for the channel.
    /// @param data_type the data type of the channel.
    /// @param rate the rate of the channel.
    /// @returns the created channel with a unique key assigned.
    [[nodiscard]] std::pair<Channel, Freighter::Error> create(
            std::string name,
            Telem::DataType data_type,
            Telem::Rate rate
    ) const;

    /// @brief retrieves a channel with the given name.
    /// @param name the name of the channel to retrieve.
    /// @throws QueryError if the channel does not exist or multiple channels with the
    /// same name exist.
    /// @returns the retrieved channel.
    [[nodiscard]] std::pair<Channel, Freighter::Error> retrieve(
            const std::string &name) const;

    /// @brief retrieves a channel with the given key.
    /// @param key the key of the channel to retrieve.
    /// @throws QueryError if the channel does not exist.
    /// @returns the retrieved channel.
    [[nodiscard]] std::pair<Channel, Freighter::Error> retrieve(std::uint32_t key) const;

    /// @brief retrieves channels with the given names.
    /// @param names the names of the channels to retrieve.
    /// @returns all channels matching the given names. If a channel matching a name,
    /// does not exist, it will not be in the returned vector.
    [[nodiscard]] std::pair<std::vector<Channel>, Freighter::Error>
    retrieve(const std::vector<std::string> &names) const;

    /// @brief retrieves channels with the given keys.
    /// @param keys the keys of the channels to retrieve.
    /// @returns all channels matching the given keys. If a channel matching a key
    /// does not exist, it will not be in the returned vector.
    [[nodiscard]] std::pair<std::vector<Channel>, Freighter::Error>
    retrieve(const std::vector<Key> &keys) const;

private:
    RetrieveClient *retrieve_client;
    CreateClient *create_client;
};

}
