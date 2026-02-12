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
#include <ostream>
#include <string>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "client/cpp/ontology/id.h"
#include "client/cpp/rack/proto.gen.h"
#include "client/cpp/rack/types.gen.h"
#include "client/cpp/task/task.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/json/json.h"
#include "x/cpp/status/status.h"

#include "core/pkg/api/grpc/rack/rack.pb.h"
#include "core/pkg/service/rack/pb/rack.pb.h"

namespace synnax::rack {

/// @brief Type alias for the transport used to create a rack.
using CreateClient = freighter::
    UnaryClient<api::v1::RackCreateRequest, api::v1::RackCreateResponse>;

/// @brief Type alias for the transport used to retrieve a rack.
using RetrieveClient = freighter::
    UnaryClient<api::v1::RackRetrieveRequest, api::v1::RackRetrieveResponse>;

/// @brief Type alias for the transport used to delete a rack.
using DeleteClient = freighter::
    UnaryClient<api::v1::RackDeleteRequest, google::protobuf::Empty>;

/// @brief An alias for the type of rack's key.
using Key = std::uint32_t;

/// @brief Converts a rack key to an ontology ID.
/// @param key The rack key.
/// @returns An ontology ID with type "rack" and the given key.
inline ontology::ID ontology_id(const rack::Key key) {
    return ontology::ID{.type = "rack", .key = std::to_string(key)};
}

/// @brief Converts a vector of rack keys to a vector of ontology IDs.
/// @param keys The rack keys.
/// @returns A vector of ontology IDs.
inline std::vector<ontology::ID> ontology_ids(const std::vector<rack::Key> &keys) {
    std::vector<ontology::ID> ids;
    ids.reserve(keys.size());
    for (const auto &key: keys)
        ids.push_back(ontology_id(key));
    return ids;
}

/// @brief Extracts the node ID from a rack key.
/// @param key The rack key.
/// @returns The node ID portion of the rack key.
inline std::uint16_t rack_key_node(const rack::Key key) {
    return key >> 12;
}

/// @brief Specific status details for racks.
struct StatusDetails {
    /// @brief The rack that this status is for.
    rack::Key rack = 0;

    /// @brief Parses the rack status details from a JSON parser.
    static StatusDetails parse(x::json::Parser parser) {
        return StatusDetails{
            .rack = parser.field<rack::Key>("rack"),
        };
    }

    /// @brief Converts the rack status details to JSON.
    [[nodiscard]] x::json::json to_json() const {
        x::json::json j;
        j["rack"] = this->rack;
        return j;
    }
};

/// @brief Status information for a rack.
using Status = x::status::Status<StatusDetails>;

/// @brief A Rack represents a physical or logical grouping of hardware devices.
/// Racks contain tasks that can be used to interact with hardware.
struct Rack {
    /// @brief The unique identifier for the rack.
    rack::Key key{};

    /// @brief A human-readable name for the rack.
    std::string name;

    /// @brief Status information for the rack.
    Status status;

    /// @brief Client for managing tasks on this rack.
    /// Note: This will be initialized after construction by RackClient.
    task::Client tasks = task::Client(0, nullptr, nullptr, nullptr);

    /// @brief Constructs a rack from its protobuf representation.
    /// @param rack The protobuf representation of the rack.
    /// @returns A pair containing the rack and an error if one occurred.
    static std::pair<Rack, x::errors::Error> from_proto(const api::v1::Rack &rack);

    /// @brief Equality operator for racks.
    /// @param rack The rack to compare with.
    /// @returns True if the racks have the same key.
    bool operator==(const Rack &rack) const { return rack.key == key; }

    /// @brief Stream output operator for racks.
    /// @param os The output stream.
    /// @param rack The rack to output.
    /// @returns The output stream.
    friend std::ostream &operator<<(std::ostream &os, const Rack &rack) {
        os << rack.name << " (" << rack.key << ")";
        return os;
    }

    /// @brief Converts the rack to its protobuf representation.
    /// @param rack The protobuf object to populate.
    void to_proto(api::v1::Rack *rack) const;
};

/// @brief Client for managing racks in a Synnax cluster.
class Client {
public:
    /// @brief Constructs a new rack client with the given transport clients.
    /// @param rack_create_client Client for creating racks.
    /// @param rack_retrieve_client Client for retrieving racks.
    /// @param rack_delete_client Client for deleting racks.
    /// @param task_create_client Client for creating tasks (shared for TaskClient).
    /// @param task_retrieve_client Client for retrieving tasks (shared for TaskClient).
    /// @param task_delete_client Client for deleting tasks (shared for TaskClient).
    Client(
        std::unique_ptr<CreateClient> rack_create_client,
        std::unique_ptr<RetrieveClient> rack_retrieve_client,
        std::unique_ptr<DeleteClient> rack_delete_client,
        std::shared_ptr<task::CreateClient> task_create_client,
        std::shared_ptr<task::RetrieveClient> task_retrieve_client,
        std::shared_ptr<task::DeleteClient> task_delete_client
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
    /// @brief Task creation transport (shared for creating TaskClient).
    std::shared_ptr<task::CreateClient> task_create_client;
    /// @brief Task retrieval transport (shared for creating TaskClient).
    std::shared_ptr<task::RetrieveClient> task_retrieve_client;
    /// @brief Task deletion transport (shared for creating TaskClient).
    std::shared_ptr<task::DeleteClient> task_delete_client;
};

}
