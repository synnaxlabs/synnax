// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>
#include <vector>

#include "client/cpp/arc/arc.h"
#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

namespace synnax::arc {
const std::string CREATE_ENDPOINT = "/api/v1/arc/create";
const std::string RETRIEVE_ENDPOINT = "/api/v1/arc/retrieve";
const std::string DELETE_ENDPOINT = "/api/v1/arc/delete";

Client::Client(
    std::shared_ptr<RetrieveClient> retrieve_client,
    std::shared_ptr<CreateClient> create_client,
    std::shared_ptr<DeleteClient> delete_client
):
    retrieve_client(std::move(retrieve_client)),
    create_client(std::move(create_client)),
    delete_client(std::move(delete_client)) {}

x::errors::Error Client::create(Arc &arc) const {
    auto req = grpc::arc::CreateRequest();
    auto [pb, pb_err] = arc.to_proto();
    if (pb_err) return pb_err;
    *req.add_arcs() = pb;
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (err) return err;
    if (res.arcs_size() == 0) return errors::unexpected_missing_error("arc");

    auto [updated_arc, from_err] = Arc::from_proto(res.arcs(0));
    if (from_err) return from_err;
    arc = updated_arc;

    return x::errors::NIL;
}

x::errors::Error Client::create(std::vector<Arc> &arcs) const {
    auto req = grpc::arc::CreateRequest();
    req.mutable_arcs()->Reserve(static_cast<int>(arcs.size()));
    for (const auto &arc: arcs) {
        auto [pb, pb_err] = arc.to_proto();
        if (pb_err) return pb_err;
        *req.add_arcs() = pb;
    }

    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (err) return err;

    for (int i = 0; i < res.arcs_size(); i++) {
        auto [updated_arc, from_err] = Arc::from_proto(res.arcs(i));
        if (from_err) return from_err;
        arcs[i] = updated_arc;
    }

    return x::errors::NIL;
}

std::pair<Arc, x::errors::Error> Client::create(const std::string &name) const {
    Arc arc{.name = name, .mode = MODE_TEXT};
    auto err = create(arc);
    return {arc, err};
}

std::pair<Arc, x::errors::Error> Client::retrieve_by_name(
    const std::string &name,
    const RetrieveOptions &options
) const {
    return this->retrieve_one(
        {
            .where = Where{.name = x::filter::String{.eq = name}},
            .include_program = options.include_program,
        },
        "name " + name
    );
}

std::pair<Arc, x::errors::Error> Client::retrieve_by_key(
    const x::uuid::UUID &key,
    const RetrieveOptions &options
) const {
    return this->retrieve_one(
        {
            .where = Where{.key = x::filter::Set<Key>{.eq = key}},
            .include_program = options.include_program,
        },
        "key " + key.to_string()
    );
}

std::pair<std::vector<Arc>, x::errors::Error> Client::retrieve(
    const std::vector<std::string> &names,
    const RetrieveOptions &options
) const {
    return this->retrieve({
        .where = Where{.name = x::filter::String{.in = names}},
        .include_program = options.include_program,
    });
}

std::pair<std::vector<Arc>, x::errors::Error> Client::retrieve_by_keys(
    const std::vector<x::uuid::UUID> &keys,
    const RetrieveOptions &options
) const {
    return this->retrieve({
        .where = Where{.key = x::filter::Set<Key>{.in = keys}},
        .include_program = options.include_program,
    });
}

std::pair<std::vector<Arc>, x::errors::Error>
Client::retrieve(const RetrieveRequest &req) const {
    auto [pb_req, pb_err] = req.to_proto();
    if (pb_err) return {std::vector<Arc>(), pb_err};
    grpc::arc::RetrieveRequest grpc_req;
    *grpc_req.mutable_request() = pb_req;
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, grpc_req);
    if (err) return {std::vector<Arc>(), err};
    auto [decoded, proto_err] = RetrieveResponse::from_proto(res);
    if (proto_err) return {std::vector<Arc>(), proto_err};
    return {std::move(decoded.arcs), x::errors::NIL};
}

std::pair<Arc, x::errors::Error>
Client::retrieve_one(const RetrieveRequest &req, const std::string &query) const {
    auto [arcs, err] = this->retrieve(req);
    if (err) return {Arc{}, err};
    if (arcs.empty()) return {Arc{}, errors::not_found_error("Arc", query)};
    if (arcs.size() > 1) return {Arc{}, errors::multiple_found_error("arcs", query)};
    return {std::move(arcs[0]), x::errors::NIL};
}

x::errors::Error Client::del(const x::uuid::UUID &key) const {
    auto req = grpc::arc::DeleteRequest();
    req.add_keys(key.to_string());

    auto [_, err] = delete_client->send(DELETE_ENDPOINT, req);
    return err;
}

x::errors::Error Client::del(const std::vector<x::uuid::UUID> &keys) const {
    auto req = grpc::arc::DeleteRequest();
    for (const auto &key: keys)
        req.add_keys(key.to_string());
    auto [_, err] = delete_client->send(DELETE_ENDPOINT, req);
    return err;
}
}
