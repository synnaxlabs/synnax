// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "client/cpp/ontology/id.h"
#include "client/cpp/rack/proto.gen.h"
#include "client/cpp/rack/types.gen.h"
#include "client/cpp/task/task.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/json/json.h"

#include "core/pkg/api/grpc/rack/rack.pb.h"
#include "core/pkg/service/rack/pb/rack.pb.h"

namespace synnax::rack {

/// @brief Type alias for the transport used to create a rack.
using CreateClient = freighter::
    UnaryClient<grpc::rack::CreateRequest, grpc::rack::CreateResponse>;

/// @brief Type alias for the transport used to retrieve a rack.
using RetrieveClient = freighter::
    UnaryClient<grpc::rack::RetrieveRequest, grpc::rack::RetrieveResponse>;

/// @brief Type alias for the transport used to delete a rack.
using DeleteClient = freighter::
    UnaryClient<grpc::rack::DeleteRequest, google::protobuf::Empty>;

/// @brief An alias for the type of rack's key.
using Key = std::uint32_t;

/// @brief Converts a rack key to an ontology ID.
/// @param key The rack key.
/// @returns An ontology ID with type "rack" and the given key.
inline ontology::ID rack_ontology_id(const Key key) {
    return ontology::ID("rack", std::to_string(key));
}

/// @brief Converts a vector of rack keys to a vector of ontology IDs.
/// @param keys The rack keys.
/// @returns A vector of ontology IDs.
inline std::vector<ontology::ID> rack_ontology_ids(const std::vector<Key> &keys) {
    std::vector<ontology::ID> ids;
    ids.reserve(keys.size());
    for (const auto &key: keys)
        ids.push_back(rack_ontology_id(key));
    return ids;
}

/// @brief Extracts the node ID from a rack key.
/// @param key The rack key.
/// @returns The node ID portion of the rack key.
inline std::uint16_t rack_key_node(const Key key) {
    return key >> 12;
}

/// @brief Client for managing racks in a Synnax cluster.
class Client {
public:
    /// @brief Constructs a new rack client with the given transport clients.
    /// @param rack_create_client Client for creating racks.
    /// @param rack_retrieve_client Client for retrieving racks.
    /// @param rack_delete_client Client for deleting racks.
    Client(
        std::unique_ptr<CreateClient> rack_create_client,
        std::unique_ptr<RetrieveClient> rack_retrieve_client,
        std::unique_ptr<DeleteClient> rack_delete_client,
        task::Client tasks
    );

    /// @brief Creates a rack in the cluster.
    /// @param rack The rack to create. Will be updated with the assigned key and
    /// task client.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    x::errors::Error create(Rack &rack) const;

    /// @brief Creates a rack with the given name in the cluster.
    /// @param name The name of the rack to create.
    /// @returns A pair containing the created rack and an error if one occurred.
    [[nodiscard]]
    std::pair<Rack, x::errors::Error> create(const std::string &name) const;

    /// @brief Retrieves a rack by its key.
    /// @param key The key of the rack to retrieve.
    /// @returns A pair containing the retrieved rack and an error if one occurred.
    [[nodiscard]]
    std::pair<Rack, x::errors::Error> retrieve(std::uint32_t key) const;

    /// @brief Retrieves a rack by its name.
    /// @param name The name of the rack to retrieve.
    /// @returns A pair containing the retrieved rack and an error if one occurred.
    [[nodiscard]]
    std::pair<Rack, x::errors::Error> retrieve(const std::string &name) const;

    /// @brief Deletes a rack by its key.
    /// @param key The key of the rack to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    x::errors::Error del(std::uint32_t key) const;

private:
    [[nodiscard]]
    std::pair<Rack, x::errors::Error>
    retrieve(grpc::rack::RetrieveRequest &req, const std::string &query) const;

    /// @brief Rack creation transport.
    std::unique_ptr<CreateClient> rack_create_client;
    /// @brief Rack retrieval transport.
    std::unique_ptr<RetrieveClient> rack_retrieve_client;
    /// @brief Rack deletion transport.
    std::unique_ptr<DeleteClient> rack_delete_client;
    task::Client tasks;
};

}
