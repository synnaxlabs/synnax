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
#include "client/cpp/task/builder.gen.h"
#include "client/cpp/task/json.gen.h"
#include "client/cpp/task/proto.gen.h"
#include "client/cpp/task/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "core/pkg/api/task/pb/task.pb.h"
#include "core/pkg/transport/grpc/task/task.pb.h"

namespace synnax::task {

const std::string SET_CHANNEL = "sy_task_set";
const std::string DELETE_CHANNEL = "sy_task_delete";
const std::string CMD_CHANNEL = "sy_task_cmd";

/// @brief The command type that deploys a task's stored config and runs it.
const std::string START_CMD_TYPE = "start";

/// @brief Type alias for the transport used to create a task.
using CreateClient = freighter::
    UnaryClient<grpc::task::CreateRequest, grpc::task::CreateResponse>;

/// @brief Type alias for the transport used to retrieve a task.
using RetrieveClient = freighter::
    UnaryClient<grpc::task::RetrieveRequest, ::api::task::pb::RetrieveResponse>;

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

    /// @brief Retrieves the task with the given key, with each include option in
    /// includes applied.
    /// @returns the task, or a NOT_FOUND error when no task has the key.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<Task, x::errors::Error>
    retrieve(const Key &key, Includes... includes) const {
        return this->retrieve_one(
            x::filter::request<Where, RetrieveRequest>(task::key.eq(key), includes...),
            "key " + key.to_string()
        );
    }

    /// @brief Retrieves the task with the given name, with each include option in
    /// includes applied.
    /// @returns the task, or an error when no task or more than one task has the name.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<Task, x::errors::Error>
    retrieve(const std::string &name, Includes... includes) const {
        return this->retrieve_one(
            x::filter::request<Where, RetrieveRequest>(
                task::name.eq(name),
                includes...
            ),
            "name " + name
        );
    }

    /// @brief Retrieves the tasks with the given keys, with each include option in
    /// includes applied. A key that no task has is left out.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<std::vector<Task>, x::errors::Error>
    retrieve(const std::vector<Key> &keys, Includes... includes) const {
        return this->retrieve(
            x::filter::request<Where, RetrieveRequest>(task::key.in(keys), includes...)
        );
    }

    /// @brief Retrieves the tasks with the given names, with each include option in
    /// includes applied. A name that no task has is left out.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<std::vector<Task>, x::errors::Error>
    retrieve(const std::vector<std::string> &names, Includes... includes) const {
        return this->retrieve(
            x::filter::request<Where, RetrieveRequest>(
                task::name.in(names),
                includes...
            )
        );
    }

    /// @brief Retrieves the tasks that req selects, on this client's rack when it has
    /// one.
    [[nodiscard]]
    std::pair<std::vector<Task>, x::errors::Error>
    retrieve(const RetrieveRequest &req) const;

    /// @brief Retrieves the tasks that match every condition in parts, on this client's
    /// rack when it has one, with each option in parts applied.
    /// @returns the tasks, or one task when parts end with task::one(). One task is an
    /// error when no task or more than one task matches.
    template<typename... Parts>
        requires x::filter::parts<Where, RetrieveRequest, Parts...>
    [[nodiscard]] auto retrieve(Parts... parts) const {
        const auto req = x::filter::request<Where, RetrieveRequest>(
            std::move(parts)...
        );
        if constexpr (x::filter::returns_one<Parts...>)
            return this->retrieve_one(req, x::filter::describe(req.where));
        else
            return this->retrieve(req);
    }

    /// @brief Deletes a task by its key.
    /// @param key The key of the task to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    x::errors::Error del(Key key) const;

    [[nodiscard]] Client scope_to_rack(const rack::Key &rack_key) const {
        auto c = *this;
        c.rack = rack_key;
        return c;
    }

private:
    [[nodiscard]]
    std::pair<Task, x::errors::Error>
    retrieve_one(const RetrieveRequest &req, const std::string &query) const;

    /// @brief Key of rack that this client belongs to.
    rack::Key rack{};
    /// @brief Task creation transport.
    std::shared_ptr<CreateClient> task_create_client;
    /// @brief Task retrieval transport.
    std::shared_ptr<RetrieveClient> task_retrieve_client;
    /// @brief Task deletion transport.
    std::shared_ptr<DeleteClient> task_delete_client;
};
}
