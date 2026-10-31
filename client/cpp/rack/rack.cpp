// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/errors/errors.h"
#include "client/cpp/rack/rack.h"
#include "client/cpp/task/task.h"
#include "x/cpp/errors/errors.h"

namespace synnax::rack {
Client::Client(
    std::unique_ptr<CreateClient> rack_create_client,
    std::unique_ptr<RetrieveClient> rack_retrieve_client,
    std::unique_ptr<DeleteClient> rack_delete_client,
    task::Client tasks
):
    rack_create_client(std::move(rack_create_client)),
    rack_retrieve_client(std::move(rack_retrieve_client)),
    rack_delete_client(std::move(rack_delete_client)),
    tasks(std::move(tasks)) {}

std::pair<std::vector<Rack>, x::errors::Error>
Client::retrieve(const RetrieveRequest &req) const {
    auto [pb_req, pb_err] = req.to_proto();
    if (pb_err) return {std::vector<Rack>(), pb_err};
    grpc::rack::RetrieveRequest grpc_req;
    *grpc_req.mutable_request() = pb_req;
    auto [res, err] = rack_retrieve_client->send("/rack/retrieve", grpc_req);
    if (err) return {std::vector<Rack>(), err};
    auto [decoded, proto_err] = RetrieveResponse::from_proto(res);
    if (proto_err) return {std::vector<Rack>(), proto_err};
    for (auto &rack: decoded.racks)
        rack.tasks = this->tasks.scope_to_rack(rack.key);
    return {std::move(decoded.racks), x::errors::NIL};
}

std::pair<Rack, x::errors::Error>
Client::retrieve_one(const RetrieveRequest &req) const {
    return errors::exactly_one("rack", req.where, this->retrieve(req));
}

x::errors::Error Client::create(Rack &rack) const {
    auto req = grpc::rack::CreateRequest();
    auto [pb, pb_err] = rack.to_proto();
    if (pb_err) return pb_err;
    *req.add_racks() = pb;
    auto [res, err] = rack_create_client->send("/rack/create", req);
    if (err) return err;
    if (res.racks_size() == 0) return errors::unexpected_missing_error("rack");
    rack.key = res.racks().at(0).key();
    rack.tasks = this->tasks.scope_to_rack(rack.key);
    return err;
}

std::pair<Rack, x::errors::Error> Client::create(const std::string &name) const {
    auto rack = Rack{.name = name};
    auto err = create(rack);
    return {rack, err};
}

x::errors::Error Client::del(const Key key) const {
    auto req = grpc::rack::DeleteRequest();
    req.add_keys(key);
    auto [res, err] = rack_delete_client->send("/rack/delete", req);
    return err;
}
}
