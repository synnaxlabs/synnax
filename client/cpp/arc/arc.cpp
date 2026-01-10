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
const std::string ARC_CREATE_ENDPOINT = "/api/v1/arc/create";
const std::string ARC_RETRIEVE_ENDPOINT = "/api/v1/arc/retrieve";
const std::string ARC_DELETE_ENDPOINT = "/api/v1/arc/delete";

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
    *req.add_arcs() = arc.to_proto();
    auto [res, err] = create_client->send(ARC_CREATE_ENDPOINT, req);
    if (err) return err;
    if (res.arcs_size() == 0) return unexpected_missing_error("arc");

    auto [updated_arc, from_err] = Arc::from_proto(res.arcs(0));
    if (from_err) return from_err;
    arc = updated_arc;

    return x::errors::NIL;
}

x::errors::Error Client::create(std::vector<Arc> &arcs) const {
    auto req = grpc::arc::CreateRequest();
    req.mutable_arcs()->Reserve(static_cast<int>(arcs.size()));
    for (const auto &arc: arcs)
        *req.add_arcs() = arc.to_proto();

    auto [res, err] = create_client->send(ARC_CREATE_ENDPOINT, req);
    if (err) return err;

    for (int i = 0; i < res.arcs_size(); i++) {
        auto [updated_arc, from_err] = Arc::from_proto(res.arcs(i));
        if (from_err) return from_err;
        arcs[i] = updated_arc;
    }

    return x::errors::NIL;
}

std::pair<Arc, x::errors::Error> Client::create(const std::string &name) const {
    Arc arc{.name = name};
    auto err = create(arc);
    return {arc, err};
}

std::pair<Arc, x::errors::Error> Client::retrieve_by_name(
    const std::string &name,
    const RetrieveOptions &options
) const {
    auto req = grpc::arc::RetrieveRequest();
    req.add_names(name);
    options.apply(req);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {Arc{}, err};
    if (res.arcs_size() == 0) return {Arc{}, unexpected_missing_error("arc")};
    if (res.arcs_size() > 1) return {Arc{}, multiple_found_error("arc", name)};

    return Arc::from_proto(res.arcs(0));
}

std::pair<Arc, x::errors::Error>
Client::retrieve_by_key(const std::string &key, const RetrieveOptions &options) const {
    auto req = grpc::arc::RetrieveRequest();
    req.add_keys(key);
    options.apply(req);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {Arc{}, err};
    if (res.arcs_size() == 0) return {Arc{}, unexpected_missing_error("arc")};

    return Arc::from_proto(res.arcs(0));
}

std::pair<std::vector<Arc>, x::errors::Error> Client::retrieve(
    const std::vector<std::string> &names,
    const RetrieveOptions &options
) const {
    auto req = grpc::arc::RetrieveRequest();
    for (const auto &name: names)
        req.add_names(name);
    options.apply(req);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {std::vector<Arc>(), err};

    std::vector<Arc> arcs;
    arcs.reserve(res.arcs_size());
    for (const auto &pb: res.arcs()) {
        auto [arc, from_err] = Arc::from_proto(pb);
        if (from_err) return {std::vector<Arc>(), from_err};
        arcs.push_back(arc);
    }

    return {arcs, x::errors::NIL};
}

std::pair<std::vector<Arc>, x::errors::Error> Client::retrieve_by_keys(
    const std::vector<std::string> &keys,
    const RetrieveOptions &options
) const {
    auto req = grpc::arc::RetrieveRequest();
    for (const auto &key: keys)
        req.add_keys(key);
    options.apply(req);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {std::vector<Arc>(), err};

    std::vector<Arc> arcs;
    arcs.reserve(res.arcs_size());
    for (const auto &pb: res.arcs()) {
        auto [arc, from_err] = Arc::from_proto(pb);
        if (from_err) return {std::vector<Arc>(), from_err};
        arcs.push_back(arc);
    }

    return {arcs, x::errors::NIL};
}

x::errors::Error Client::delete_arc(const std::string &key) const {
    auto req = grpc::arc::DeleteRequest();
    req.add_keys(key);

    auto [_, err] = delete_client->send(ARC_DELETE_ENDPOINT, req);
    return err;
}

x::errors::Error Client::delete_arc(const std::vector<std::string> &keys) const {
    auto req = grpc::arc::DeleteRequest();
    for (const auto &key: keys)
        req.add_keys(key);

    auto [_, err] = delete_client->send(ARC_DELETE_ENDPOINT, req);
    return err;
}
}
