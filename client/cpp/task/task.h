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
#include "client/cpp/task/proto.gen.h"
#include "client/cpp/task/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/status/status.h"

#include "core/pkg/api/grpc/task/task.pb.h"
#include "core/pkg/service/task/pb/task.pb.h"

namespace synnax::task {
// Forward declaration for RackKey (needed for task key utilities)
using RackKey = std::uint32_t;

/// @brief Type alias for the transport used to create a task.
using CreateClient = freighter::
    UnaryClient<grpc::task::CreateRequest, grpc::task::CreateResponse>;

/// @brief Type alias for the transport used to retrieve a task.
using RetrieveClient = freighter::
    UnaryClient<grpc::task::RetrieveRequest, grpc::task::RetrieveResponse>;

/// @brief Type alias for the transport used to delete a task.
using DeleteClient = freighter::
    UnaryClient<grpc::task::DeleteRequest, google::protobuf::Empty>;

/// @brief Converts a task key to an ontology ID.
/// @param key The task key.
/// @returns An ontology ID with type "task" and the given key.
inline ontology::ID ontology_id(const Key key) {
    return ontology::ID("task", std::to_string(key));
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
inline Key create_task_key(const RackKey rack, const Key task) {
    return static_cast<task::Key>(rack) << 32 | task;
}

/// @brief Extracts the rack key from a task key.
/// @param key The task key.
/// @returns The rack key portion of the task key.
inline RackKey rack_key_from_task_key(const Key key) {
    return key >> 32;
}

/// @brief Extracts the local task key from a task key.
/// @param key The task key.
/// @returns The local task key portion of the task key.
inline std::uint32_t local_task_key(const Key key) {
    return key & 0xFFFFFFFF;
}

/// @brief Options for retrieving tasks.
struct RetrieveOptions {
    /// @brief Whether to include status information in the retrieved tasks.
    bool include_status = false;
};

/// @brief A Task is a data structure used to configure and execute operations on a
/// hardware device. Tasks are associated with a specific rack and can be created,
/// retrieved, and deleted.
///
/// Task extends the generated task::Payload struct, adding convenience constructors
/// and utility methods while leveraging generated code for data fields and
/// protobuf translation.
class Task : public Payload {
public:
    /// @brief Default constructor for an empty task.
    Task() = default;

    /// @brief Converting constructor from generated Payload type.
    /// @param payload The generated payload to convert from.
    Task(Payload &&payload): Payload(std::move(payload)) {}

    /// @brief Converting constructor from generated Payload type (const ref).
    /// @param payload The generated payload to convert from.
    Task(const Payload &payload): Payload(payload) {}

    /// @brief Constructs a new task with the given properties.
    /// @param name A human-readable name for the task.
    /// @param type The type of the task.
    /// @param config Configuration data for the task.
    /// @param internal Whether the task is internal to the system.
    /// @param snapshot Whether the task is a snapshot and cannot be modified.
    Task(
        std::string name,
        std::string type,
        std::string config,
        bool internal = false,
        bool snapshot = false
    );

    /// @brief Constructs a new task with the given properties and key.
    /// @param key The unique identifier for the task.
    /// @param name A human-readable name for the task.
    /// @param type The type of the task.
    /// @param config Configuration data for the task.
    /// @param internal Whether the task is internal to the system.
    /// @param snapshot Whether the task is a snapshot and cannot be modified.
    Task(
        Key key,
        std::string name,
        std::string type,
        std::string config,
        bool internal = false,
        bool snapshot = false
    );

    /// @brief Constructs a new task with the given properties and rack.
    /// @param rack The rack key that this task belongs to.
    /// @param name A human-readable name for the task.
    /// @param type The type of the task.
    /// @param config Configuration data for the task.
    /// @param internal Whether the task is internal to the system.
    /// @param snapshot Whether the task is a snapshot and cannot be modified.
    Task(
        RackKey rack,
        std::string name,
        std::string type,
        std::string config,
        bool internal = false,
        bool snapshot = false
    );

    friend std::ostream &operator<<(std::ostream &os, const Task &task) {
        return os << task.name << " (" << task.key << ")";
    }

    /// @brief returns the key used for creating statuses associated with the task.
    [[nodiscard]] std::string status_key() const {
        return ontology_id(this->key).string();
    }

    /// @brief Returns the rack key that this task belongs to.
    [[nodiscard]] RackKey rack() const { return rack_key_from_task_key(this->key); }

    friend class Client;
};

/// @brief Client for managing tasks on a specific rack.
class Client {
public:
    Client() = default;
    /// @brief Constructs a new task client for the given rack.
    /// @param rack The rack key that this client operates on.
    /// @param task_create_client Client for creating tasks.
    /// @param task_retrieve_client Client for retrieving tasks.
    /// @param task_delete_client Client for deleting tasks.
    Client(
        std::shared_ptr<CreateClient> task_create_client,
        std::shared_ptr<RetrieveClient> task_retrieve_client,
        std::shared_ptr<DeleteClient> task_delete_client
    ):
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
    x::errors::Error del(task::Key key) const;

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
    RackKey rack;
    /// @brief Task creation transport.
    std::shared_ptr<CreateClient> task_create_client;
    /// @brief Task retrieval transport.
    std::shared_ptr<RetrieveClient> task_retrieve_client;
    /// @brief Task deletion transport.
    std::shared_ptr<DeleteClient> task_delete_client;
};

}
