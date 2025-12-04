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

std::pair<Task, xerrors::Error> Task::from_proto(const api::v1::Task &task) {
    Task t;
    t.key = task.key();
    t.name = task.name();
    t.type = task.type();
    t.config = task.config();
    t.internal = task.internal();
    t.snapshot = task.snapshot();
    if (task.has_status()) {
        auto [s, err] = TaskStatus::from_proto(task.status());
        if (err) return {t, err};
        t.status = s;
    }
    return {t, xerrors::NIL};
}

void Task::to_proto(api::v1::Task *task) const {
    task->set_key(key);
    task->set_name(name);
    task->set_type(type);
    task->set_config(config);
    task->set_internal(internal);
    task->set_snapshot(snapshot);
    if (!status.is_zero()) status.to_proto(task->mutable_status());
}

std::pair<Task, xerrors::Error> TaskClient::retrieve(const TaskKey key) const {
    return retrieve(key, TaskRetrieveOptions{});
}

std::pair<Task, xerrors::Error>
TaskClient::retrieve(const TaskKey key, const TaskRetrieveOptions &options) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.add_keys(key);
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0)
        return {Task(), not_found_error("task", "key " + std::to_string(key))};
    return Task::from_proto(res.tasks(0));
}

std::pair<Task, xerrors::Error> TaskClient::retrieve(const std::string &name) const {
    return retrieve(name, TaskRetrieveOptions{});
}

std::pair<Task, xerrors::Error> TaskClient::retrieve(
    const std::string &name,
    const TaskRetrieveOptions &options
) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.add_names(name);
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0) return {Task(), not_found_error("task", "name " + name)};
    return Task::from_proto(res.tasks(0));
}

std::pair<std::vector<Task>, xerrors::Error>
TaskClient::retrieve(const std::vector<std::string> &names) const {
    return retrieve(names, TaskRetrieveOptions{});
}

std::pair<std::vector<Task>, xerrors::Error> TaskClient::retrieve(
    const std::vector<std::string> &names,
    const TaskRetrieveOptions &options
) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.mutable_names()->Add(names.begin(), names.end());
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks;
    tasks.reserve(res.tasks_size());
    for (const auto &t: res.tasks()) {
        auto [task, proto_err] = Task::from_proto(t);
        if (proto_err) return {std::vector<Task>(), proto_err};
        tasks.push_back(std::move(task));
    }
    return {tasks, xerrors::NIL};
}

std::pair<Task, xerrors::Error>
TaskClient::retrieve_by_type(const std::string &type) const {
    return retrieve_by_type(type, TaskRetrieveOptions{});
}

std::pair<Task, xerrors::Error> TaskClient::retrieve_by_type(
    const std::string &type,
    const TaskRetrieveOptions &options
) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.add_types(type);
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0) return {Task(), not_found_error("task", "type " + type)};
    return Task::from_proto(res.tasks(0));
}

std::pair<std::vector<Task>, xerrors::Error>
TaskClient::retrieve_by_type(const std::vector<std::string> &types) const {
    return retrieve_by_type(types, TaskRetrieveOptions{});
}

std::pair<std::vector<Task>, xerrors::Error> TaskClient::retrieve_by_type(
    const std::vector<std::string> &types,
    const TaskRetrieveOptions &options
) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.mutable_types()->Add(types.begin(), types.end());
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks;
    tasks.reserve(res.tasks_size());
    for (const auto &t: res.tasks()) {
        auto [task, proto_err] = Task::from_proto(t);
        if (proto_err) return {std::vector<Task>(), proto_err};
        tasks.push_back(std::move(task));
    }
    return {tasks, xerrors::NIL};
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
    return list(TaskRetrieveOptions{});
}

std::pair<std::vector<Task>, xerrors::Error>
TaskClient::list(const TaskRetrieveOptions &options) const {
    auto req = api::v1::TaskRetrieveRequest();
    req.set_rack(rack);
    req.set_include_status(options.include_status);
    auto [res, err] = task_retrieve_client->send("/task/retrieve", req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks;
    tasks.reserve(res.tasks_size());
    for (const auto &t: res.tasks()) {
        auto [task, proto_err] = Task::from_proto(t);
        if (proto_err) return {std::vector<Task>(), proto_err};
        tasks.push_back(std::move(task));
    }
    return {tasks, xerrors::NIL};
}
}
