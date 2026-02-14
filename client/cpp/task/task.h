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
#include <utility>
#include <vector>

#include "client/cpp/ontology/id.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/status/status.h"

#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/task.pb.h"

namespace synnax {
namespace rack {
using Key = std::uint32_t;
}
namespace task {
const std::string SET_CHANNEL = "sy_task_set";
const std::string DELETE_CHANNEL = "sy_task_delete";
const std::string CMD_CHANNEL = "sy_task_cmd";

/// @brief Type alias for the transport used to create a task.
using CreateClient = freighter::
    UnaryClient<api::v1::TaskCreateRequest, api::v1::TaskCreateResponse>;

/// @brief Type alias for the transport used to retrieve a task.
using RetrieveClient = freighter::
    UnaryClient<api::v1::TaskRetrieveRequest, api::v1::TaskRetrieveResponse>;

/// @brief Type alias for the transport used to delete a task.
using DeleteClient = freighter::
    UnaryClient<api::v1::TaskDeleteRequest, google::protobuf::Empty>;

/// @brief An alias for the type of task's key.
using Key = std::uint64_t;

/// @brief Converts a task key to an ontology ID.
/// @param key The task key.
/// @returns An ontology ID with type "task" and the given key.
inline ontology::ID ontology_id(const Key key) {
    return ontology::ID{.type = "task", .key = std::to_string(key)};
}

/// @brief Converts a vector of task keys to a vector of ontology IDs.
/// @param keys The task keys.
/// @returns A vector of ontology IDs.
inline std::vector<ontology::ID> ontology_ids(const std::vector<Key> &keys) {
    std::vector<ontology::ID> ids;
    ids.reserve(keys.size());
    for (const auto &key: keys)
        ids.push_back(ontology_id(key));
    return ids;
}

/// @brief Creates a task key from a rack key and a local task key.
/// @param rack The rack key.
/// @param task The local task key.
/// @returns A combined task key.
inline Key create_key(const rack::Key rack, const Key task) {
    return static_cast<Key>(rack) << 32 | task;
}

/// @brief Extracts the rack key from a task key.
/// @param key The task key.
/// @returns The rack key portion of the task key.
inline rack::Key rack_key_from_task_key(const Key key) {
    return key >> 32;
}

/// @brief Extracts the local task key from a task key.
/// @param key The task key.
/// @returns The local task key portion of the task key.
inline std::uint32_t local_key(const Key key) {
    return key & 0xFFFFFFFF;
}

/// @brief specific status details for tasks.
struct StatusDetails {
    /// @brief The key of the task that this status is for.
    Key task;
    /// @brief Is a non-empty string if the status is an explicit response to a command.
    std::string cmd;
    /// @brief whether the task is currently running.
    bool running;
    /// @brief additional data associated with the task.
    x::json::json data;

    /// @brief parses the task status details from a JSON parser.
    static StatusDetails parse(x::json::Parser parser) {
        return StatusDetails{
            .task = parser.field<Key>("task"),
            .cmd = parser.field<std::string>("cmd", ""),
            .running = parser.field<bool>("running"),
            .data = parser.field<x::json::json>("data"),
        };
    }

    /// @brief converts the task status details to JSON.
    [[nodiscard]] x::json::json to_json() const {
        x::json::json j;
        j["task"] = this->task;
        j["running"] = this->running;
        j["data"] = this->data;
        j["cmd"] = this->cmd;
        return j;
    }
};

/// @brief status information for a task.
using Status = x::status::Status<StatusDetails>;

/// @brief Options for retrieving tasks.
struct RetrieveOptions {
    /// @brief Whether to include status information in the retrieved tasks.
    bool include_status = false;
};

/// @brief A Task is a data structure used to configure and execute operations on a
/// hardware device. Tasks are associated with a specific rack and can be created,
/// retrieved, and deleted.
struct Task {
    /// @brief The unique identifier for the task.
    Key key = 0;
    /// @brief A human-readable name for the task.
    std::string name;
    /// @brief The type of the task, which determines its behavior.
    std::string type;
    /// @brief Configuration data for the task, typically in JSON format.
    std::string config;
    /// @brief Whether the task is internal to the system.
    bool internal = false;
    /// @brief Whether the task is a snapshot.
    bool snapshot = false;

    /// @brief Status information for the task.
    Status status;

    /// @brief Constructs a task from its protobuf representation.
    /// @param task The protobuf representation of the task.
    /// @returns A pair containing the task and an error if one occurred.
    static std::pair<Task, x::errors::Error> from_proto(const api::v1::Task &task);

    friend std::ostream &operator<<(std::ostream &os, const Task &task) {
        return os << task.name << " (key=" << task.key << ",type=" << task.type << ")";
    }

    /// @brief returns the key used for creating statuses associated with the task.
    [[nodiscard]] std::string status_key() const {
        return ontology_id(this->key).string();
    }

    [[nodiscard]] synnax::rack::Key rack() const {
        return rack_key_from_task_key(this->key);
    }

    /// @brief Converts the task to its protobuf representation.
    /// @param task The protobuf object to populate.
    void to_proto(api::v1::Task *task) const;
};

/// @brief Client for managing tasks on a specific rack.
class Client {
public:
    /// @brief Constructs a new task client for the given rack.
    /// @param rack The rack key that this client operates on.
    /// @param task_create_client Client for creating tasks.
    /// @param task_retrieve_client Client for retrieving tasks.
    /// @param task_delete_client Client for deleting tasks.
    Client(
        const rack::Key rack,
        std::shared_ptr<CreateClient> task_create_client,
        std::shared_ptr<RetrieveClient> task_retrieve_client,
        std::shared_ptr<DeleteClient> task_delete_client
    ):
        rack(rack),
        task_create_client(std::move(task_create_client)),
        task_retrieve_client(std::move(task_retrieve_client)),
        task_delete_client(std::move(task_delete_client)) {}

    /// @brief Creates a task on the rack.
    /// @param task The task to create. Will be updated with the assigned key.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    x::errors::Error create(Task &task) const;

    /// @brief Retrieves a task by its key.
    /// @param key The key of the task to retrieve.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, x::errors::Error> retrieve(Key key) const;

    /// @brief Retrieves a task by its key with options.
    /// @param key The key of the task to retrieve.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, x::errors::Error>
    retrieve(Key key, const RetrieveOptions &options) const;

    /// @brief Retrieves a task by its type.
    /// @param type The type of the task to retrieve.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, x::errors::Error> retrieve_by_type(const std::string &type) const;

    /// @brief Retrieves a task by its type with options.
    /// @param type The type of the task to retrieve.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, x::errors::Error>
    retrieve_by_type(const std::string &type, const RetrieveOptions &options) const;

    /// @brief Retrieves a task by its name.
    /// @param name The name of the task to retrieve.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, x::errors::Error> retrieve(const std::string &name) const;

    /// @brief Retrieves a task by its name with options.
    /// @param name The name of the task to retrieve.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, x::errors::Error>
    retrieve(const std::string &name, const RetrieveOptions &options) const;

    /// @brief Retrieves multiple tasks by their names.
    /// @param names The names of the tasks to retrieve.
    /// @returns A pair containing the retrieved tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, x::errors::Error>
    retrieve(const std::vector<std::string> &names) const;

    /// @brief Retrieves multiple tasks by their names with options.
    /// @param names The names of the tasks to retrieve.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the retrieved tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, x::errors::Error> retrieve(
        const std::vector<std::string> &names,
        const RetrieveOptions &options
    ) const;

    /// @brief Retrieves multiple tasks by their types.
    /// @param types The types of the tasks to retrieve.
    /// @returns A pair containing the retrieved tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, x::errors::Error>
    retrieve_by_type(const std::vector<std::string> &types) const;

    /// @brief Retrieves multiple tasks by their types with options.
    /// @param types The types of the tasks to retrieve.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the retrieved tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, x::errors::Error> retrieve_by_type(
        const std::vector<std::string> &types,
        const RetrieveOptions &options
    ) const;

    /// @brief Deletes a task by its key.
    /// @param key The key of the task to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    x::errors::Error del(Key key) const;

    /// @brief Lists all tasks on the rack.
    /// @returns A pair containing the list of tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, x::errors::Error> list() const;

    /// @brief Lists all tasks on the rack with options.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the list of tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, x::errors::Error>
    list(const RetrieveOptions &options) const;

private:
    /// @brief Key of rack that this client belongs to.
    rack::Key rack;
    /// @brief Task creation transport.
    std::shared_ptr<CreateClient> task_create_client;
    /// @brief Task retrieval transport.
    std::shared_ptr<RetrieveClient> task_retrieve_client;
    /// @brief Task deletion transport.
    std::shared_ptr<DeleteClient> task_delete_client;
};
}
}
