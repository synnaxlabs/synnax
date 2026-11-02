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
std::pair<std::vector<Task>, x::errors::Error>
Client::retrieve(const RetrieveRequest &req) const {
    auto scoped = req;
    if (this->rack != 0) {
        const auto on_rack = Where{
            .rack = x::filter::Set<rack::Key>{.eq = this->rack},
        };
        scoped.where = scoped.where.has_value()
                         ? Where{.and_ = std::vector<Where>{*scoped.where, on_rack}}
                         : on_rack;
    }
    auto [pb_req, pb_err] = scoped.to_proto();
    if (pb_err) return {std::vector<Task>(), pb_err};
    grpc::task::RetrieveRequest grpc_req;
    *grpc_req.mutable_request() = pb_req;
    auto [res, err] = task_retrieve_client->send("/task/retrieve", grpc_req);
    if (err) return {std::vector<Task>(), err};
    auto [decoded, proto_err] = RetrieveResponse::from_proto(res);
    if (proto_err) return {std::vector<Task>(), proto_err};
    return {std::move(decoded.tasks), x::errors::NIL};
}

std::pair<Task, x::errors::Error>
Client::retrieve_one(const RetrieveRequest &req, const std::string &query) const {
    auto [tasks, err] = this->retrieve(req);
    if (err) return {Task{}, err};
    if (tasks.empty()) return {Task{}, errors::not_found_error("task", query)};
    if (tasks.size() > 1) return {Task{}, errors::multiple_found_error("tasks", query)};
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

x::errors::Error Client::del(const Key key) const {
    auto req = grpc::task::DeleteRequest();
    req.add_keys(key.to_string());
    auto [res, err] = task_delete_client->send("/task/delete", req);
    return err;
}
}
