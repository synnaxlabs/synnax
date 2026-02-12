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
std::pair<Rack, x::errors::Error> Rack::from_proto(const api::v1::Rack &rack) {
    Rack r;
    r.key = rack.key();
    r.name = rack.name();
    if (rack.has_status()) {
        auto [s, err] = Status::from_proto(rack.status());
        if (err) return {r, err};
        r.status = s;
    }
    return {r, x::errors::NIL};
}

void Rack::to_proto(api::v1::Rack *rack) const {
    rack->set_key(key);
    rack->set_name(name);
    if (!status.is_zero()) status.to_proto(rack->mutable_status());
}

Client::Client(
    std::unique_ptr<CreateClient> rack_create_client,
    std::unique_ptr<RetrieveClient> rack_retrieve_client,
    std::unique_ptr<DeleteClient> rack_delete_client,
    std::shared_ptr<task::CreateClient> task_create_client,
    std::shared_ptr<task::RetrieveClient> task_retrieve_client,
    std::shared_ptr<task::DeleteClient> task_delete_client
):
    rack_create_client(std::move(rack_create_client)),
    rack_retrieve_client(std::move(rack_retrieve_client)),
    rack_delete_client(std::move(rack_delete_client)),
    task_create_client(std::move(task_create_client)),
    task_retrieve_client(std::move(task_retrieve_client)),
    task_delete_client(std::move(task_delete_client)) {}

std::pair<Rack, x::errors::Error> Client::retrieve(const rack::Key key) const {
    auto req = api::v1::RackRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = rack_retrieve_client->send("/rack/retrieve", req);
    if (err) return {Rack(), err};
    if (res.racks_size() == 0)
        return {Rack(), errors::not_found_error("Rack", "key " + std::to_string(key))};
    auto [rack, proto_err] = Rack::from_proto(res.racks(0));
    if (proto_err) return {Rack(), proto_err};
    rack.tasks = task::Client(
        rack.key,
        task_create_client,
        task_retrieve_client,
        task_delete_client
    );
    return {rack, x::errors::NIL};
}

std::pair<Rack, x::errors::Error> Client::retrieve(const std::string &name) const {
    auto req = api::v1::RackRetrieveRequest();
    req.add_names(name);
    auto [res, err] = rack_retrieve_client->send("/rack/retrieve", req);
    if (err) return {Rack(), err};
    if (res.racks_size() == 0)
        return {Rack(), errors::not_found_error("Rack", "name " + name)};
    if (res.racks_size() > 1)
        return {Rack(), errors::multiple_found_error("racks", "name " + name)};
    auto [rack, proto_err] = Rack::from_proto(res.racks(0));
    if (proto_err) return {Rack(), proto_err};
    rack.tasks = task::Client(
        rack.key,
        task_create_client,
        task_retrieve_client,
        task_delete_client
    );
    return {rack, x::errors::NIL};
}

x::errors::Error Client::create(Rack &rack) const {
    auto req = api::v1::RackCreateRequest();
    rack.to_proto(req.add_racks());
    auto [res, err] = rack_create_client->send("/rack/create", req);
    if (err) return err;
    if (res.racks_size() == 0) return errors::unexpected_missing_error("rack");
    rack.key = res.racks().at(0).key();
    rack.tasks = task::Client(
        rack.key,
        task_create_client,
        task_retrieve_client,
        task_delete_client
    );
    return err;
}

std::pair<Rack, x::errors::Error> Client::create(const std::string &name) const {
    auto rack = Rack{.name = name};
    auto err = create(rack);
    return {rack, err};
}

x::errors::Error Client::del(const rack::Key key) const {
    auto req = api::v1::RackDeleteRequest();
    req.add_keys(key);
    auto [res, err] = rack_delete_client->send("/rack/delete", req);
    return err;
}
}
