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

std::pair<Rack, x::errors::Error> Client::retrieve(const Key key) const {
    auto req = grpc::rack::RetrieveRequest();
    req.add_keys(key);
    auto [res, err] = rack_retrieve_client->send("/rack/retrieve", req);
    if (err) return {Rack{}, err};
    if (res.racks_size() == 0)
        return {Rack{}, not_found_error("Rack", "key " + std::to_string(key))};
    auto [pld, proto_err] = Rack::from_proto(res.racks(0));
    if (proto_err) return {Rack{}, proto_err};
    Rack rack(std::move(pld));
    return {rack, x::errors::NIL};
}

std::pair<Rack, x::errors::Error> Client::retrieve(const std::string &name) const {
    auto req = grpc::rack::RetrieveRequest();
    req.add_names(name);
    auto [res, err] = rack_retrieve_client->send("/rack/retrieve", req);
    if (err) return {Rack{}, err};
    if (res.racks_size() == 0) return {Rack{}, not_found_error("Rack", "name " + name)};
    if (res.racks_size() > 1)
        return {Rack{}, multiple_found_error("racks", "name " + name)};
    auto [pld, proto_err] = Rack::from_proto(res.racks(0));
    if (proto_err) return {Rack{}, proto_err};
    return {pld, x::errors::NIL};
}

x::errors::Error Client::create(Rack &rack) const {
    auto req = grpc::rack::CreateRequest();
    *req.add_racks() = rack.to_proto();
    auto [res, err] = rack_create_client->send("/rack/create", req);
    if (err) return err;
    if (res.racks_size() == 0) return unexpected_missing_error("rack");
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
