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

#include "client/cpp/rack/actions.gen.h"
#include "client/cpp/rack/proto.gen.h"
#include "client/cpp/rack/types.gen.h"
#include "client/cpp/resource/id.h"
#include "client/cpp/task/task.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/filter/field.h"
#include "x/cpp/json/json.h"

#include "core/pkg/api/rack/pb/rack.pb.h"
#include "core/pkg/transport/grpc/rack/rack.pb.h"

namespace synnax::rack {

const std::string SET_CHANNEL = "sy_rack_set";

/// @brief Type alias for the transport used to create a rack.
using CreateClient = freighter::
    UnaryClient<grpc::rack::CreateRequest, grpc::rack::CreateResponse>;

/// @brief Type alias for the transport used to retrieve a rack.
using RetrieveClient = freighter::
    UnaryClient<grpc::rack::RetrieveRequest, ::api::rack::pb::RetrieveResponse>;

/// @brief Type alias for the transport used to delete a rack.
using DeleteClient = freighter::
    UnaryClient<grpc::rack::DeleteRequest, google::protobuf::Empty>;

using DispatchClient = freighter::
    UnaryClient<grpc::rack::DispatchRequest, google::protobuf::Empty>;

/// @brief An alias for the type of rack's key.
using Key = std::uint32_t;

/// @brief Extracts the node ID from a rack key.
/// @param key The rack key.
/// @returns The node ID portion of the rack key.
inline std::uint16_t rack_key_node(const Key key) {
    return key >> 16;
}

/// @brief Client for managing racks in a Synnax cluster.
class Client {
public:
    /// @brief Constructs a new rack client with the given transport clients.
    /// @param rack_create_client Client for creating racks.
    /// @param rack_retrieve_client Client for retrieving racks.
    /// @param rack_delete_client Client for deleting racks.
    /// @param rack_dispatch_client Client for dispatching actions to racks.
    Client(
        std::unique_ptr<CreateClient> rack_create_client,
        std::unique_ptr<RetrieveClient> rack_retrieve_client,
        std::unique_ptr<DeleteClient> rack_delete_client,
        std::unique_ptr<DispatchClient> rack_dispatch_client,
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

    /// @brief retrieves the rack with the given key.
    /// @param key the key of the rack.
    /// @param options the options of the retrieve.
    /// @returns the rack, and an error that matches NOT_FOUND when no rack has the key.
    [[nodiscard]]
    std::pair<Rack, x::errors::Error>
    retrieve(Key key, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the rack with the given name.
    /// @param name the name of the rack.
    /// @param options the options of the retrieve.
    /// @returns the rack, and an error that matches NOT_FOUND when no rack has the
    /// name, or MULTIPLE_RESULTS when more than one does.
    [[nodiscard]]
    std::pair<Rack, x::errors::Error>
    retrieve(const std::string &name, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the racks that where selects.
    /// @param where the filter that the racks match. An empty filter matches all racks.
    /// @param options the options of the retrieve.
    [[nodiscard]]
    std::pair<std::vector<Rack>, x::errors::Error>
    retrieve(const Where &where, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the one rack that where selects.
    /// @param where the filter that the rack matches.
    /// @param options the options of the retrieve.
    /// @returns the rack, and an error that matches NOT_FOUND when no rack matches,
    /// MULTIPLE_RESULTS when more than one does, or VALIDATION when options set a
    /// limit or a cursor.
    [[nodiscard]]
    std::pair<Rack, x::errors::Error> retrieve(
        const Where &where,
        x::filter::One,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief applies the actions to the rack with the given key in order, as one
    /// atomic edit.
    /// @param key the key of the rack.
    /// @param actions the actions to apply.
    /// @returns an error that matches NOT_FOUND when no rack has the key, or VALIDATION
    /// when the rack cannot hold an action.
    [[nodiscard]]
    x::errors::Error dispatch(Key key, const std::vector<Action> &actions) const;

    /// @brief Deletes a rack by its key.
    /// @param key The key of the rack to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    x::errors::Error del(std::uint32_t key) const;

private:
    /// @brief Rack creation transport.
    std::unique_ptr<CreateClient> rack_create_client;
    /// @brief Rack retrieval transport.
    std::unique_ptr<RetrieveClient> rack_retrieve_client;
    /// @brief Rack deletion transport.
    std::unique_ptr<DeleteClient> rack_delete_client;
    /// @brief Rack dispatch transport.
    std::unique_ptr<DispatchClient> rack_dispatch_client;
    task::Client tasks;
};

inline std::ostream &operator<<(std::ostream &os, const Rack &rack) {
    os << rack.name << " (" << rack.key << ")";
    return os;
}

}
