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

std::pair<Arc, x::errors::Error> Arc::from_proto(const api::v1::Arc &pb) {
    auto [key, err] = x::uuid::UUID::parse(pb.key());
    if (err) return {{}, err};
    Arc arc;
    arc.key = key;
    arc.name = pb.name();
    if (pb.has_graph()) arc.graph = ::arc::graph::Graph(pb.graph());
    if (pb.has_text()) arc.text = ::arc::text::Text(pb.text());
    if (pb.has_module()) arc.module = ::arc::module::Module(pb.module());
    arc.deploy = pb.deploy();
    arc.version = pb.version();
    return {arc, x::errors::NIL};
}

void Arc::to_proto(api::v1::Arc *pb) const {
    if (!this->key.is_nil()) pb->set_key(this->key.to_string());
    pb->set_name(name);
    graph.to_proto(pb->mutable_graph());
    text.to_proto(pb->mutable_text());
    module.to_proto(pb->mutable_module());
    pb->set_deploy(deploy);
    pb->set_version(version);
}

Client::Client(
    std::shared_ptr<RetrieveClient> retrieve_client,
    std::shared_ptr<CreateClient> create_client,
    std::shared_ptr<DeleteClient> delete_client
):
    retrieve_client(std::move(retrieve_client)),
    create_client(std::move(create_client)),
    delete_client(std::move(delete_client)) {}

x::errors::Error Client::create(Arc &arc) const {
    auto req = api::v1::ArcCreateRequest();
    arc.to_proto(req.add_arcs());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (err) return err;
    if (res.arcs_size() == 0) return errors::unexpected_missing_error("arc");

    const auto &first = res.arcs(0);
    auto [key, parse_err] = x::uuid::UUID::parse(first.key());
    if (parse_err) return parse_err;
    arc.key = key;
    arc.name = first.name();
    if (first.has_graph()) arc.graph = ::arc::graph::Graph(first.graph());
    if (first.has_text()) arc.text = ::arc::text::Text(first.text());
    if (first.has_module()) arc.module = ::arc::module::Module(first.module());
    arc.deploy = first.deploy();
    arc.version = first.version();

    return x::errors::NIL;
}

x::errors::Error Client::create(std::vector<Arc> &arcs) const {
    auto req = api::v1::ArcCreateRequest();
    req.mutable_arcs()->Reserve(static_cast<int>(arcs.size()));
    for (const auto &arc: arcs)
        *req.add_arcs() = arc.to_proto();

    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (err) return err;

    for (int i = 0; i < res.arcs_size(); i++) {
        const auto &pb = res.arcs(i);
        auto [key, parse_err] = x::uuid::UUID::parse(pb.key());
        if (parse_err) return parse_err;
        arcs[i].key = key;
        arcs[i].name = pb.name();
        if (pb.has_graph()) arcs[i].graph = ::arc::graph::Graph(pb.graph());
        if (pb.has_text()) arcs[i].text = ::arc::text::Text(pb.text());
        if (pb.has_module()) arcs[i].module = ::arc::module::Module(pb.module());
        arcs[i].deploy = pb.deploy();
        arcs[i].version = pb.version();
    }

    return x::errors::NIL;
}

std::pair<Arc, x::errors::Error> Client::create(const std::string &name) const {
    auto arc = Arc{.name = name};
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

    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {Arc(), err};
    if (res.arcs_size() == 0) return {Arc(), errors::unexpected_missing_error("arc")};
    if (res.arcs_size() > 1) return {Arc(), errors::multiple_found_error("arc", name)};

    return Arc::from_proto(res.arcs(0));
}

std::pair<Arc, x::errors::Error> Client::retrieve_by_key(
    const x::uuid::UUID &key,
    const RetrieveOptions &options
) const {
    auto req = api::v1::ArcRetrieveRequest();
    req.add_keys(key.to_string());
    options.apply(req);

    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {Arc(), err};
    if (res.arcs_size() == 0) return {Arc(), errors::unexpected_missing_error("arc")};

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

    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {{}, err};

    std::vector<Arc> arcs;
    arcs.reserve(res.arcs_size());
    for (const auto &pb: res.arcs()) {
        auto [arc, parse_err] = Arc::from_proto(pb);
        if (parse_err) return {{}, parse_err};
        arcs.push_back(std::move(arc));
    }

    return {arcs, x::errors::NIL};
}

std::pair<std::vector<Arc>, x::errors::Error> Client::retrieve_by_keys(
    const std::vector<x::uuid::UUID> &keys,
    const RetrieveOptions &options
) const {
    auto req = grpc::arc::RetrieveRequest();
    for (const auto &key: keys)
        req.add_keys(key.to_string());
    options.apply(req);

    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {{}, err};

    std::vector<Arc> arcs;
    arcs.reserve(res.arcs_size());
    for (const auto &pb: res.arcs()) {
        auto [arc, parse_err] = Arc::from_proto(pb);
        if (parse_err) return {{}, parse_err};
        arcs.push_back(std::move(arc));
    }

    return {arcs, x::errors::NIL};
}

x::errors::Error Client::delete_arc(const x::uuid::UUID &key) const {
    auto req = api::v1::ArcDeleteRequest();
    req.add_keys(key.to_string());

    auto [_, err] = delete_client->send(DELETE_ENDPOINT, req);
    return err;
}

x::errors::Error Client::delete_arc(const std::vector<x::uuid::UUID> &keys) const {
    auto req = api::v1::ArcDeleteRequest();
    for (const auto &key: keys)
        req.add_keys(key.to_string());

    auto [_, err] = delete_client->send(DELETE_ENDPOINT, req);
    return err;
}
}
