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
#include "x/cpp/json/struct.h"

namespace synnax::task {
std::pair<Task, x::errors::Error>
Client::retrieve(const Key key, const RetrieveOptions &options) const {
    return this->retrieve(Where{.key = key}, x::filter::one, options);
}

std::pair<Task, x::errors::Error>
Client::retrieve(const std::string &name, const RetrieveOptions &options) const {
    return this->retrieve(Where{.name = name}, x::filter::one, options);
}

std::pair<std::vector<Task>, x::errors::Error> Client::retrieve(
    const std::vector<std::string> &names,
    const RetrieveOptions &options
) const {
    return this->retrieve(Where{.name = names}, options);
}

std::pair<std::vector<Task>, x::errors::Error>
Client::retrieve(const Where &where, const RetrieveOptions &options) const {
    auto scoped = where;
    if (this->rack != 0) scoped = std::move(scoped) && Where{.rack = this->rack};
    auto [pb_req, pb_err] = RetrieveRequest::from(std::move(scoped), options)
                                .to_proto();
    if (pb_err) return {std::vector<Task>(), pb_err};
    grpc::task::RetrieveRequest grpc_req;
    *grpc_req.mutable_request() = pb_req;
    auto [res, err] = task_retrieve_client->send("/task/retrieve", grpc_req);
    if (err) return {std::vector<Task>(), err};
    auto [decoded, proto_err] = RetrieveResponse::from_proto(res);
    if (proto_err) return {std::vector<Task>(), proto_err};
    return {std::move(decoded.tasks), x::errors::NIL};
}

std::pair<Task, x::errors::Error> Client::retrieve(
    const Where &where,
    x::filter::One,
    const RetrieveOptions &options
) const {
    if (options.limit.has_value() || options.after.has_value())
        return {Task{}, errors::paged_one_error()};
    auto [tasks, err] = this->retrieve(where, options);
    if (err) return {Task{}, err};
    if (const auto one_err = errors::check_one(
            "task",
            where.to_json(),
            options.search_term,
            tasks.size()
        ))
        return {Task{}, one_err};
    return {std::move(tasks[0]), x::errors::NIL};
}

x::errors::Error Client::create(Task &task) const {
    if (task.rack == 0) task.rack = this->rack;
    auto req = grpc::task::CreateRequest();
    auto [pb, pb_err] = task.to_proto();
    if (pb_err) return pb_err;
    *req.add_tasks() = pb;
    auto [res, err] = task_create_client->send("/task/create", req);
    if (err) return err;
    if (res.tasks_size() == 0) return errors::unexpected_missing_error("task");
    auto [key, key_err] = x::uuid::UUID::parse(res.tasks().at(0).key());
    if (key_err) return key_err;
    task.key = key;
    task.config_hash = res.tasks().at(0).config_hash();
    return x::errors::NIL;
}

x::errors::Error
Client::dispatch(const Key key, const std::vector<Action> &actions) const {
    grpc::task::DispatchRequest req;
    req.set_key(key.to_string());
    for (const auto &action: actions)
        req.add_actions(to_json(action).dump());
    return this->task_dispatch_client->send("/task/dispatch", req).second;
}

x::errors::Error Client::del(const Key key) const {
    auto req = grpc::task::DeleteRequest();
    req.add_keys(key.to_string());
    auto [res, err] = task_delete_client->send("/task/delete", req);
    return err;
}
}
