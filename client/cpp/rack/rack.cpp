// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xerrors/errors.h"

namespace synnax {
Rack::Rack(const RackKey key, std::string name): key(key), name(std::move(name)) {}

Rack::Rack(std::string name): name(std::move(name)) {}

Rack::Rack(const api::v1::Rack &rack): key(rack.key()), name(rack.name()) {}

void Rack::to_proto(api::v1::Rack *rack) const {
    rack->set_key(key);
    rack->set_name(name);
}

RackClient::RackClient(
    std::unique_ptr<RackCreateClient> rack_create_client,
    std::unique_ptr<RackRetrieveClient> rack_retrieve_client,
    std::unique_ptr<RackDeleteClient> rack_delete_client,
    std::shared_ptr<TaskCreateClient> task_create_client,
    std::shared_ptr<TaskRetrieveClient> task_retrieve_client,
    std::shared_ptr<TaskDeleteClient> task_delete_client
):
    rack_create_client(std::move(rack_create_client)),
    rack_retrieve_client(std::move(rack_retrieve_client)),
    rack_delete_client(std::move(rack_delete_client)),
    task_create_client(std::move(task_create_client)),
    task_retrieve_client(std::move(task_retrieve_client)),
    task_delete_client(std::move(task_delete_client)) {}

const std::string RETRIEVE_RACK_ENDPOINT = "/rack/retrieve";
const std::string CREATE_RACK_ENDPOINT = "/rack/create";
const std::string DELETE_RACK_ENDPOINT = "/rack/delete";

std::pair<Rack, xerrors::Error> RackClient::retrieve(const RackKey key) const {
    auto req = api::v1::RackRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = rack_retrieve_client->send(RETRIEVE_RACK_ENDPOINT, req);
    if (err) return {Rack(), err};
    if (res.racks_size() == 0)
        return {
            Rack(),
            xerrors::Error(
                xerrors::NOT_FOUND,
                "Rack matching" + std::to_string(key) + " not found"
            )
        };
    auto rack = Rack(res.racks(0));
    rack.tasks = TaskClient(
        rack.key,
        task_create_client,
        task_retrieve_client,
        task_delete_client
    );
    return {rack, err};
}

std::pair<Rack, xerrors::Error> RackClient::retrieve(const std::string &name) const {
    auto req = api::v1::RackRetrieveRequest();
    req.add_names(name);
    auto [res, err] = rack_retrieve_client->send(RETRIEVE_RACK_ENDPOINT, req);
    if (err) return {Rack(), err};
    if (res.racks_size() == 0)
        return {
            Rack(),
            xerrors::Error(xerrors::NOT_FOUND, "Rack matching" + name + " not found")
        };
    if (res.racks_size() > 1)
        return {
            Rack(),
            xerrors::Error(
                xerrors::MULTIPLE_RESULTS,
                "Multiple racks matching" + name + " found"
            )
        };
    auto rack = Rack(res.racks(0));
    rack.tasks = TaskClient(
        rack.key,
        task_create_client,
        task_retrieve_client,
        task_delete_client
    );
    return {rack, err};
}

xerrors::Error RackClient::create(Rack &rack) const {
    auto req = api::v1::RackCreateRequest();
    rack.to_proto(req.add_racks());
    auto [res, err] = rack_create_client->send(CREATE_RACK_ENDPOINT, req);
    if (err) return err;
    if (res.racks_size() == 0) return unexpected_missing("rack");
    rack.key = res.racks().at(0).key();
    rack.tasks = TaskClient(
        rack.key,
        task_create_client,
        task_retrieve_client,
        task_delete_client
    );
    return err;
}

std::pair<Rack, xerrors::Error> RackClient::create(const std::string &name) const {
    auto rack = Rack(name);
    auto err = create(rack);
    return {rack, err};
}

xerrors::Error RackClient::del(const RackKey key) const {
    auto req = api::v1::RackDeleteRequest();
    req.add_keys(key);
    auto [res, err] = rack_delete_client->send(DELETE_RACK_ENDPOINT, req);
    return err;
}
}
