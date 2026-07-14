// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <ostream>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/status/status.h"
#include "client/cpp/task/json.gen.h"
#include "client/cpp/task/proto.gen.h"
#include "client/cpp/task/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "core/pkg/service/task/pb/task.pb.h"
#include "core/pkg/transport/grpc/task/task.pb.h"

namespace synnax { namespace task {

const std::string SET_CHANNEL = "sy_task_set";
const std::string DELETE_CHANNEL = "sy_task_delete";
const std::string CMD_CHANNEL = "sy_task_cmd";

/// @brief Type alias for the transport used to create a task.
using CreateClient = freighter::
    UnaryClient<grpc::task::CreateRequest, grpc::task::CreateResponse>;

/// @brief Type alias for the transport used to retrieve a task.
using RetrieveClient = freighter::
    UnaryClient<grpc::task::RetrieveRequest, grpc::task::RetrieveResponse>;

/// @brief Type alias for the transport used to delete a task.
using DeleteClient = freighter::
    UnaryClient<grpc::task::DeleteRequest, google::protobuf::Empty>;

/// @brief Returns a unique status key for a task.
/// @param task The task.
/// @returns A unique key for status updates, derived from the task key.
inline std::string status_key(const Task &task) {
    return ontology_id(task.key).string();
}

/// @brief Stream output operator for Task.
/// @param os The output stream.
/// @param task The task to output.
/// @returns The output stream.
inline std::ostream &operator<<(std::ostream &os, const Task &task) {
    return os << task.name << " (key=" << task.key << ",type=" << task.type << ")";
}

/// @brief Stream output operator for Command.
/// @param os The output stream.
/// @param cmd The command to output.
/// @returns The output stream.
inline std::ostream &operator<<(std::ostream &os, const Command &cmd) {
    return os << cmd.type << " (task=" << cmd.task << ",key=" << cmd.key << ")";
}

/// @brief Options for retrieving tasks.
struct RetrieveOptions {
    /// @brief Whether to include status information in the retrieved tasks.
    bool include_status = false;
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

    Client scope_to_rack(const rack::Key &rack_key) const {
        auto c = *this;
        c.rack = rack_key;
        return c;
    }

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
}}
