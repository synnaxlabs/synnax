// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xerrors/errors.h"

const std::string ARC_CREATE_ENDPOINT = "/api/v1/arc/create";
const std::string ARC_RETRIEVE_ENDPOINT = "/api/v1/arc/retrieve";
const std::string ARC_DELETE_ENDPOINT = "/api/v1/arc/delete";

namespace synnax {

Arc::Arc(std::string name): name(std::move(name)) {}

Arc::Arc(const api::v1::Arc &pb):
    key(pb.key()),
    name(pb.name()),
    graph(pb.has_graph() ? arc::graph::Graph(pb.graph()) : arc::graph::Graph()),
    text(pb.has_text() ? arc::text::Text(pb.text()) : arc::text::Text()),
    module(pb.has_module() ? arc::module::Module(pb.module()) : arc::module::Module()),
    deploy(pb.deploy()),
    version(pb.version()) {}

void Arc::to_proto(api::v1::Arc *pb) const {
    // Only set key if it's not empty (server generates UUID for new Arcs)
    if (!key.empty()) pb->set_key(key);
    pb->set_name(name);
    graph.to_proto(pb->mutable_graph());
    text.to_proto(pb->mutable_text());
    module.to_proto(pb->mutable_module());
    pb->set_deploy(deploy);
    pb->set_version(version);
}

ArcClient::ArcClient(
    std::shared_ptr<ArcRetrieveClient> retrieve_client,
    std::shared_ptr<ArcCreateClient> create_client,
    std::shared_ptr<ArcDeleteClient> delete_client
):
    retrieve_client(std::move(retrieve_client)),
    create_client(std::move(create_client)),
    delete_client(std::move(delete_client)) {}

xerrors::Error ArcClient::create(Arc &arc) const {
    auto req = api::v1::ArcCreateRequest();
    arc.to_proto(req.add_arcs());
    auto [res, err] = create_client->send(ARC_CREATE_ENDPOINT, req);
    if (err) return err;
    if (res.arcs_size() == 0) return unexpected_missing_error("arc");

    const auto &first = res.arcs(0);
    arc.key = first.key();
    arc.name = first.name();
    if (first.has_graph()) arc.graph = arc::graph::Graph(first.graph());
    if (first.has_text()) arc.text = arc::text::Text(first.text());
    if (first.has_module()) arc.module = arc::module::Module(first.module());
    arc.deploy = first.deploy();
    arc.version = first.version();

    return xerrors::NIL;
}

xerrors::Error ArcClient::create(std::vector<Arc> &arcs) const {
    auto req = api::v1::ArcCreateRequest();
    req.mutable_arcs()->Reserve(static_cast<int>(arcs.size()));
    for (const auto &arc: arcs)
        arc.to_proto(req.add_arcs());

    auto [res, err] = create_client->send(ARC_CREATE_ENDPOINT, req);
    if (err) return err;

    for (int i = 0; i < res.arcs_size(); i++) {
        const auto &pb = res.arcs(i);
        arcs[i].key = pb.key();
        arcs[i].name = pb.name();
        if (pb.has_graph()) arcs[i].graph = arc::graph::Graph(pb.graph());
        if (pb.has_text()) arcs[i].text = arc::text::Text(pb.text());
        if (pb.has_module()) arcs[i].module = arc::module::Module(pb.module());
        arcs[i].deploy = pb.deploy();
        arcs[i].version = pb.version();
    }

    return xerrors::NIL;
}

std::pair<Arc, xerrors::Error> ArcClient::create(const std::string &name) const {
    auto arc = Arc(name);
    auto err = create(arc);
    return {arc, err};
}

std::pair<Arc, xerrors::Error> ArcClient::retrieve_by_name(
    const std::string &name,
    const RetrieveOptions &options
) const {
    auto req = api::v1::ArcRetrieveRequest();
    req.add_names(name);
    options.apply(req);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {Arc(), err};
    if (res.arcs_size() == 0) return {Arc(), unexpected_missing_error("arc")};
    if (res.arcs_size() > 1) return {Arc(), multiple_found_error("arc", name)};

    return {Arc(res.arcs(0)), xerrors::NIL};
}

std::pair<Arc, xerrors::Error> ArcClient::retrieve_by_key(
    const std::string &key,
    const RetrieveOptions &options
) const {
    auto req = api::v1::ArcRetrieveRequest();
    req.add_keys(key);
    options.apply(req);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {Arc(), err};
    if (res.arcs_size() == 0) return {Arc(), unexpected_missing_error("arc")};

    return {Arc(res.arcs(0)), xerrors::NIL};
}

std::pair<std::vector<Arc>, xerrors::Error> ArcClient::retrieve(
    const std::vector<std::string> &names,
    const RetrieveOptions &options
) const {
    auto req = api::v1::ArcRetrieveRequest();
    for (const auto &name: names)
        req.add_names(name);
    options.apply(req);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {std::vector<Arc>(), err};

    std::vector<Arc> arcs;
    arcs.reserve(res.arcs_size());
    for (const auto &pb: res.arcs())
        arcs.emplace_back(pb);

    return {arcs, xerrors::NIL};
}

std::pair<std::vector<Arc>, xerrors::Error> ArcClient::retrieve_by_keys(
    const std::vector<std::string> &keys,
    const RetrieveOptions &options
) const {
    auto req = api::v1::ArcRetrieveRequest();
    for (const auto &key: keys)
        req.add_keys(key);
    options.apply(req);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {std::vector<Arc>(), err};

    std::vector<Arc> arcs;
    arcs.reserve(res.arcs_size());
    for (const auto &pb: res.arcs())
        arcs.emplace_back(pb);

    return {arcs, xerrors::NIL};
}

xerrors::Error ArcClient::delete_arc(const std::string &key) const {
    auto req = api::v1::ArcDeleteRequest();
    req.add_keys(key);

    auto [_, err] = delete_client->send(ARC_DELETE_ENDPOINT, req);
    return err;
}

xerrors::Error ArcClient::delete_arc(const std::vector<std::string> &keys) const {
    auto req = api::v1::ArcDeleteRequest();
    for (const auto &key: keys)
        req.add_keys(key);

    auto [_, err] = delete_client->send(ARC_DELETE_ENDPOINT, req);
    return err;
}
}
