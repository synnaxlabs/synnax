// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/status/status.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xjson/xjson.h"

#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/task.pb.h"

namespace synnax {
// Forward declaration for RackKey (needed for task key utilities)
using RackKey = std::uint32_t;

/// @brief Type alias for the transport used to create a task.
using TaskCreateClient = freighter::
    UnaryClient<api::v1::TaskCreateRequest, api::v1::TaskCreateResponse>;

/// @brief Type alias for the transport used to retrieve a task.
using TaskRetrieveClient = freighter::
    UnaryClient<api::v1::TaskRetrieveRequest, api::v1::TaskRetrieveResponse>;

/// @brief Type alias for the transport used to delete a task.
using TaskDeleteClient = freighter::
    UnaryClient<api::v1::TaskDeleteRequest, google::protobuf::Empty>;

/// @brief An alias for the type of task's key.
using TaskKey = std::uint64_t;

/// @brief Converts a task key to an ontology ID.
/// @param key The task key.
/// @returns An ontology ID with type "task" and the given key.
inline ontology::ID task_ontology_id(TaskKey key) {
    return ontology::ID("task", std::to_string(key));
}

/// @brief Converts a vector of task keys to a vector of ontology IDs.
/// @param keys The task keys.
/// @returns A vector of ontology IDs.
inline std::vector<ontology::ID> task_ontology_ids(const std::vector<TaskKey> &keys) {
    std::vector<ontology::ID> ids;
    ids.reserve(keys.size());
    for (const auto &key: keys)
        ids.push_back(task_ontology_id(key));
    return ids;
}

/// @brief Creates a task key from a rack key and a local task key.
/// @param rack The rack key.
/// @param task The local task key.
/// @returns A combined task key.
inline TaskKey create_task_key(const RackKey rack, const TaskKey task) {
    return static_cast<TaskKey>(rack) << 32 | task;
}

/// @brief Extracts the rack key from a task key.
/// @param key The task key.
/// @returns The rack key portion of the task key.
inline RackKey rack_key_from_task_key(const TaskKey key) {
    return key >> 32;
}

/// @brief Extracts the local task key from a task key.
/// @param key The task key.
/// @returns The local task key portion of the task key.
inline std::uint32_t local_task_key(const TaskKey key) {
    return key & 0xFFFFFFFF;
}

/// @brief specific status details for tasks.
struct TaskStatusDetails {
    /// @brief The key of the task that this status is for.
    TaskKey task;
    /// @brief Is a non-empty string if the status is an explicit response to a command.
    std::string cmd;
    /// @brief whether the task is currently running.
    bool running;
    /// @brief additional data associated with the task.
    json data;

    /// @brief parses the task status details from a JSON parser.
    static TaskStatusDetails parse(xjson::Parser parser) {
        return TaskStatusDetails{
            .task = parser.field<TaskKey>("task"),
            .cmd = parser.field<std::string>("cmd"),
            .running = parser.field<bool>("running"),
            .data = parser.field<json>("data"),
        };
    }

    /// @brief converts the task status details to JSON.
    [[nodiscard]] json to_json() const {
        json j;
        j["task"] = this->task;
        j["running"] = this->running;
        j["data"] = this->data;
        j["cmd"] = this->cmd;
        return j;
    }
};

/// @brief status information for a task.
using TaskStatus = status::Status<TaskStatusDetails>;

/// @brief A Task is a data structure used to configure and execute operations on a
/// hardware device. Tasks are associated with a specific rack and can be created,
/// retrieved, and deleted.
class Task {
public:
    /// @brief The unique identifier for the task.
    TaskKey key = 0;
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
    TaskStatus status;

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
        TaskKey key,
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

    /// @brief Constructs a task from its protobuf representation.
    /// @param task The protobuf representation of the task.
    explicit Task(const api::v1::Task &task);

    /// @brief Default constructor for an empty task.
    Task() = default;

    friend std::ostream &operator<<(std::ostream &os, const Task &task) {
        return os << task.name << " (" << task.key << ")";
    }

    std::string status_key() const { return task_ontology_id(this->key).string(); }

private:
    /// @brief Converts the task to its protobuf representation.
    /// @param task The protobuf object to populate.
    void to_proto(api::v1::Task *task) const;

    friend class TaskClient;
};

/// @brief Client for managing tasks on a specific rack.
class TaskClient {
public:
    /// @brief Constructs a new task client for the given rack.
    /// @param rack The rack key that this client operates on.
    /// @param task_create_client Client for creating tasks.
    /// @param task_retrieve_client Client for retrieving tasks.
    /// @param task_delete_client Client for deleting tasks.
    TaskClient(
        const RackKey rack,
        std::shared_ptr<TaskCreateClient> task_create_client,
        std::shared_ptr<TaskRetrieveClient> task_retrieve_client,
        std::shared_ptr<TaskDeleteClient> task_delete_client
    ):
        rack(rack),
        task_create_client(std::move(task_create_client)),
        task_retrieve_client(std::move(task_retrieve_client)),
        task_delete_client(std::move(task_delete_client)) {}

    /// @brief Creates a task on the rack.
    /// @param task The task to create. Will be updated with the assigned key.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    xerrors::Error create(Task &task) const;

    /// @brief Retrieves a task by its key.
    /// @param key The key of the task to retrieve.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, xerrors::Error> retrieve(std::uint64_t key) const;

    /// @brief Retrieves a task by its type.
    /// @param type The type of the task to retrieve.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, xerrors::Error> retrieve_by_type(const std::string &type) const;

    /// @brief Retrieves a task by its name.
    /// @param name The name of the task to retrieve.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, xerrors::Error> retrieve(const std::string &name) const;

    /// @brief Retrieves multiple tasks by their names.
    /// @param names The names of the tasks to retrieve.
    /// @returns A pair containing the retrieved tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, xerrors::Error>
    retrieve(const std::vector<std::string> &names) const;

    /// @brief Retrieves multiple tasks by their types.
    /// @param types The types of the tasks to retrieve.
    /// @returns A pair containing the retrieved tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, xerrors::Error>
    retrieve_by_type(const std::vector<std::string> &types) const;

    /// @brief Deletes a task by its key.
    /// @param key The key of the task to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    xerrors::Error del(std::uint64_t key) const;

    /// @brief Lists all tasks on the rack.
    /// @returns A pair containing the list of tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, xerrors::Error> list() const;

private:
    /// @brief Key of rack that this client belongs to.
    RackKey rack;
    /// @brief Task creation transport.
    std::shared_ptr<TaskCreateClient> task_create_client;
    /// @brief Task retrieval transport.
    std::shared_ptr<TaskRetrieveClient> task_retrieve_client;
    /// @brief Task deletion transport.
    std::shared_ptr<TaskDeleteClient> task_delete_client;
};

}
