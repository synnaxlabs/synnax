// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/errors/errors.h"
#include "client/cpp/task/task.h"
#include "x/cpp/xerrors/errors.h"

namespace synnax {
Task::Task(
    TaskKey key,
    std::string name,
    std::string type,
    std::string config,
    bool internal,
    bool snapshot
):
    key(key),
    name(std::move(name)),
    type(std::move(type)),
    config(std::move(config)),
    internal(internal),
    snapshot(snapshot) {}

Task::Task(
    std::string name,
    std::string type,
    std::string config,
    bool internal,
    bool snapshot
):
    key(create_task_key(0, 0)),
    name(std::move(name)),
    type(std::move(type)),
    config(std::move(config)),
    internal(internal),
    snapshot(snapshot) {}

Task::Task(
    RackKey rack,
    std::string name,
    std::string type,
    std::string config,
    bool internal,
    bool snapshot
):
    key(create_task_key(rack, 0)),
    name(std::move(name)),
    type(std::move(type)),
    config(std::move(config)),
    internal(internal),
    snapshot(snapshot) {}

Task::Task(const api::v1::Task &task):
    key(task.key()),
    name(task.name()),
    type(task.type()),
    config(task.config()),
    internal(task.internal()),
    snapshot(task.snapshot()) {}

void Task::to_proto(api::v1::Task *task) const {
    task->set_key(key);
    task->set_name(name);
    task->set_type(type);
    task->set_config(config);
    task->set_internal(internal);
    task->set_snapshot(snapshot);
}

std::pair<Task, xerrors::Error> TaskClient::retrieve(const TaskKey key) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.add_keys(key);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0)
        return {Task(), not_found_error("task", "key " + std::to_string(key))};
    return {Task(res.tasks(0)), err};
}

std::pair<Task, xerrors::Error> TaskClient::retrieve(const std::string &name) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.add_names(name);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0) return {Task(), not_found_error("task", "name " + name)};
    return {Task(res.tasks(0)), err};
}

std::pair<std::vector<Task>, xerrors::Error>
TaskClient::retrieve(const std::vector<std::string> &names) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.mutable_names()->Add(names.begin(), names.end());
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks = {res.tasks().begin(), res.tasks().end()};
    return {tasks, err};
}

std::pair<Task, xerrors::Error>
TaskClient::retrieve_by_type(const std::string &type) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.add_types(type);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0) return {Task(), not_found_error("task", "type " + type)};
    return {Task(res.tasks(0)), err};
}

std::pair<std::vector<Task>, xerrors::Error>
TaskClient::retrieve_by_type(const std::vector<std::string> &types) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.mutable_types()->Add(types.begin(), types.end());
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks = {res.tasks().begin(), res.tasks().end()};
    return {tasks, err};
}

xerrors::Error TaskClient::create(Task &task) const {
    auto req = api::v1::TaskCreateRequest();
    task.to_proto(req.add_tasks());
    auto [res, err] = task_create_client->send("/task/create", req);
    if (err) return err;
    if (res.tasks_size() == 0) return unexpected_missing_error("task");
    task.key = res.tasks().at(0).key();
    return err;
}

xerrors::Error TaskClient::del(const TaskKey key) const {
    auto req = api::v1::TaskDeleteRequest();
    req.add_keys(key);
    auto [res, err] = task_delete_client->send("/task/delete", req);
    return err;
}

std::pair<std::vector<Task>, xerrors::Error> TaskClient::list() const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks = {res.tasks().begin(), res.tasks().end()};
    return {tasks, err};
}
}
