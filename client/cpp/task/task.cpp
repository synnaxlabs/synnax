// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/errors/errors.h"
#include "client/cpp/task/task.h"
#include "x/cpp/errors/errors.h"

namespace synnax::task {
std::pair<Task, x::errors::Error> Client::retrieve(const Key key) const {
    return retrieve(key, RetrieveOptions{});
}

std::pair<Task, x::errors::Error>
Client::retrieve(const Key key, const RetrieveOptions &options) const {
    auto req = grpc::task::RetrieveRequest();
    req.set_rack(rack);
    req.add_keys(key);
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0)
        return {Task(), not_found_error("task", "key " + std::to_string(key))};
    // Use generated translator, wrap result in Task
    auto [payload, proto_err] = Task::from_proto(res.tasks(0));
    if (proto_err) return {Task(), proto_err};
    return {Task(std::move(payload)), x::errors::NIL};
}

std::pair<Task, x::errors::Error> Client::retrieve(const std::string &name) const {
    return retrieve(name, RetrieveOptions{});
}

std::pair<Task, x::errors::Error>
Client::retrieve(const std::string &name, const RetrieveOptions &options) const {
    auto req = grpc::task::RetrieveRequest();
    req.set_rack(rack);
    req.add_names(name);
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0) return {Task(), not_found_error("task", "name " + name)};
    // Use generated translator, wrap result in Task
    auto [payload, proto_err] = Task::from_proto(res.tasks(0));
    if (proto_err) return {Task(), proto_err};
    return {Task(std::move(payload)), x::errors::NIL};
}

std::pair<std::vector<Task>, x::errors::Error>
Client::retrieve(const std::vector<std::string> &names) const {
    return retrieve(names, RetrieveOptions{});
}

std::pair<std::vector<Task>, x::errors::Error> Client::retrieve(
    const std::vector<std::string> &names,
    const RetrieveOptions &options
) const {
    auto req = grpc::task::RetrieveRequest();
    req.set_rack(rack);
    req.mutable_names()->Add(names.begin(), names.end());
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks;
    tasks.reserve(res.tasks_size());
    for (const auto &t: res.tasks()) {
        auto [payload, proto_err] = Task::from_proto(t);
        if (proto_err) return {std::vector<Task>(), proto_err};
        tasks.push_back(Task(std::move(payload)));
    }
    return {tasks, x::errors::NIL};
}

std::pair<Task, x::errors::Error>
Client::retrieve_by_type(const std::string &type) const {
    return retrieve_by_type(type, RetrieveOptions{});
}

std::pair<Task, x::errors::Error> Client::retrieve_by_type(
    const std::string &type,
    const RetrieveOptions &options
) const {
    auto req = grpc::task::RetrieveRequest();
    req.set_rack(rack);
    req.add_types(type);
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0) return {Task(), not_found_error("task", "type " + type)};
    // Use generated translator, wrap result in Task
    auto [payload, proto_err] = Task::from_proto(res.tasks(0));
    if (proto_err) return {Task(), proto_err};
    return {Task(std::move(payload)), x::errors::NIL};
}

std::pair<std::vector<Task>, x::errors::Error>
Client::retrieve_by_type(const std::vector<std::string> &types) const {
    return retrieve_by_type(types, RetrieveOptions{});
}

std::pair<std::vector<Task>, x::errors::Error> Client::retrieve_by_type(
    const std::vector<std::string> &types,
    const RetrieveOptions &options
) const {
    auto req = grpc::task::RetrieveRequest();
    req.set_rack(rack);
    req.mutable_types()->Add(types.begin(), types.end());
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks;
    tasks.reserve(res.tasks_size());
    for (const auto &t: res.tasks()) {
        auto [payload, proto_err] = Task::from_proto(t);
        if (proto_err) return {std::vector<Task>(), proto_err};
        tasks.push_back(Task(std::move(payload)));
    }
    return {tasks, x::errors::NIL};
}

x::errors::Error Client::create(Task &task) const {
    auto req = grpc::task::CreateRequest();
    // Use generated translator - implicit upcast to Payload works
    *req.add_tasks() = task.to_proto();
    auto [res, err] = task_create_client->send("/task/create", req);
    if (err) return err;
    if (res.tasks_size() == 0) return unexpected_missing_error("task");
    task.key = res.tasks().at(0).key();
    return x::errors::NIL;
}

x::errors::Error Client::del(const Key key) const {
    auto req = grpc::task::DeleteRequest();
    req.add_keys(key);
    auto [res, err] = task_delete_client->send("/task/delete", req);
    return err;
}

std::pair<std::vector<Task>, x::errors::Error> Client::list() const {
    return list(RetrieveOptions{});
}

std::pair<std::vector<Task>, x::errors::Error>
Client::list(const RetrieveOptions &options) const {
    auto req = grpc::task::RetrieveRequest();
    req.set_rack(rack);
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks;
    tasks.reserve(res.tasks_size());
    for (const auto &t: res.tasks()) {
        auto [payload, proto_err] = Task::from_proto(t);
        if (proto_err) return {std::vector<Task>(), proto_err};
        tasks.push_back(Task(std::move(payload)));
    }
    return {tasks, x::errors::NIL};
}
}
