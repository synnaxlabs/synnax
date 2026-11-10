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
#include "client/cpp/task/actions.gen.h"
#include "client/cpp/task/json.gen.h"
#include "client/cpp/task/proto.gen.h"
#include "client/cpp/task/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/filter/field.h"
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

/// @brief Type alias for the transport used to dispatch actions to a task.
using DispatchClient = freighter::
    UnaryClient<grpc::task::DispatchRequest, google::protobuf::Empty>;

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
    /// @param task_dispatch_client Client for dispatching actions to tasks.
    Client(
        std::shared_ptr<CreateClient> task_create_client,
        std::shared_ptr<RetrieveClient> task_retrieve_client,
        std::shared_ptr<DeleteClient> task_delete_client,
        std::shared_ptr<DispatchClient> task_dispatch_client
    ):
        task_create_client(std::move(task_create_client)),
        task_retrieve_client(std::move(task_retrieve_client)),
        task_delete_client(std::move(task_delete_client)),
        task_dispatch_client(std::move(task_dispatch_client)) {}

    /// @brief Creates a task on the rack.
    /// @param task The task to create. Will be updated with the assigned key.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    x::errors::Error create(Task &task) const;

    /// @brief retrieves the task with the given key.
    /// @param key the key of the task.
    /// @param options the options of the retrieve.
    /// @returns the task, and an error that matches NOT_FOUND when no task has the key.
    [[nodiscard]]
    std::pair<Task, x::errors::Error>
    retrieve(Key key, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the task with the given name.
    /// @param name the name of the task.
    /// @param options the options of the retrieve.
    /// @returns the task, and an error that matches NOT_FOUND when no task has the
    /// name, or MULTIPLE_RESULTS when more than one does.
    [[nodiscard]]
    std::pair<Task, x::errors::Error>
    retrieve(const std::string &name, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the tasks with the given names.
    /// @param names the names of the tasks.
    /// @param options the options of the retrieve.
    [[nodiscard]]
    std::pair<std::vector<Task>, x::errors::Error> retrieve(
        const std::vector<std::string> &names,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief retrieves the tasks that where selects, on the rack of the client when
    /// it has one.
    /// @param where the filter that the tasks match. An empty filter matches all tasks.
    /// @param options the options of the retrieve.
    [[nodiscard]]
    std::pair<std::vector<Task>, x::errors::Error>
    retrieve(const Where &where, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the one task that where selects, on the rack of the client when
    /// it has one.
    /// @param where the filter that the task matches.
    /// @param options the options of the retrieve.
    /// @returns the task, and an error that matches NOT_FOUND when no task matches,
    /// MULTIPLE_RESULTS when more than one does, or VALIDATION when options set a
    /// limit or a cursor.
    [[nodiscard]]
    std::pair<Task, x::errors::Error> retrieve(
        const Where &where,
        x::filter::One,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief applies the actions to the task with the given key in order, as one
    /// atomic edit.
    /// @param key the key of the task.
    /// @param actions the actions to apply.
    /// @returns an error that matches NOT_FOUND when no task has the key, or VALIDATION
    /// when the task cannot hold an action.
    [[nodiscard]]
    x::errors::Error dispatch(Key key, const std::vector<Action> &actions) const;

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
    /// @brief Key of rack that this client belongs to.
    rack::Key rack{};
    /// @brief Task creation transport.
    std::shared_ptr<CreateClient> task_create_client;
    /// @brief Task retrieval transport.
    std::shared_ptr<RetrieveClient> task_retrieve_client;
    /// @brief Task deletion transport.
    std::shared_ptr<DeleteClient> task_delete_client;
    /// @brief Task dispatch transport.
    std::shared_ptr<DispatchClient> task_dispatch_client;
};
}
