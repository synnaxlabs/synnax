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

// ========================================================================
// Arc Struct Implementation
// ========================================================================

Arc::Arc(std::string name) : name(std::move(name)) {}

Arc::Arc(const api::v1::Arc &pb) :
    key(pb.key()),
    name(pb.name()),
    graph(pb.graph()),
    text(pb.text()),
    deploy(pb.deploy()),
    version(pb.version()) {}

void Arc::to_proto(api::v1::Arc *pb) const {
    // Only set key if it's not empty (server generates UUID for new Arcs)
    if (!key.empty()) {
        pb->set_key(key);
    }
    pb->set_name(name);
    *pb->mutable_graph() = graph;
    *pb->mutable_text() = text;
    pb->set_deploy(deploy);
    pb->set_version(version);
}

// ========================================================================
// ArcClient Implementation
// ========================================================================

ArcClient::ArcClient(
    std::shared_ptr<ArcRetrieveClient> retrieve_client,
    std::shared_ptr<ArcCreateClient> create_client,
    std::shared_ptr<ArcDeleteClient> delete_client
) : retrieve_client(std::move(retrieve_client)),
    create_client(std::move(create_client)),
    delete_client(std::move(delete_client)) {}

// ========================================================================
// Create Operations
// ========================================================================

xerrors::Error ArcClient::create(Arc &arc) const {
    auto req = api::v1::ArcCreateRequest();
    arc.to_proto(req.add_arcs());
    auto [res, err] = create_client->send(ARC_CREATE_ENDPOINT, req);
    if (err) return err;
    if (res.arcs_size() == 0) return unexpected_missing("arc");

    const auto &first = res.arcs(0);
    arc.key = first.key();
    arc.name = first.name();
    arc.graph = first.graph();
    arc.text = first.text();
    arc.deploy = first.deploy();
    arc.version = first.version();

    return xerrors::NIL;
}

xerrors::Error ArcClient::create(std::vector<Arc> &arcs) const {
    auto req = api::v1::ArcCreateRequest();
    req.mutable_arcs()->Reserve(static_cast<int>(arcs.size()));
    for (const auto &arc : arcs)
        arc.to_proto(req.add_arcs());

    auto [res, err] = create_client->send(ARC_CREATE_ENDPOINT, req);
    if (err) return err;

    for (int i = 0; i < res.arcs_size(); i++) {
        const auto &pb = res.arcs(i);
        arcs[i].key = pb.key();
        arcs[i].name = pb.name();
        arcs[i].graph = pb.graph();
        arcs[i].text = pb.text();
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

// ========================================================================
// Retrieve Operations
// ========================================================================

std::pair<Arc, xerrors::Error> ArcClient::retrieve_by_name(const std::string &name) const {
    auto req = api::v1::ArcRetrieveRequest();
    req.add_names(name);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {Arc(), err};
    if (res.arcs_size() == 0) return {Arc(), unexpected_missing("arc")};
    if (res.arcs_size() > 1) return {Arc(), multiple_results("arc", name)};

    return {Arc(res.arcs(0)), xerrors::NIL};
}

std::pair<Arc, xerrors::Error> ArcClient::retrieve_by_key(const std::string &key) const {
    auto req = api::v1::ArcRetrieveRequest();
    req.add_keys(key);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {Arc(), err};
    if (res.arcs_size() == 0) return {Arc(), unexpected_missing("arc")};

    return {Arc(res.arcs(0)), xerrors::NIL};
}

std::pair<std::vector<Arc>, xerrors::Error> ArcClient::retrieve(
    const std::vector<std::string> &names
) const {
    auto req = api::v1::ArcRetrieveRequest();
    for (const auto &name : names)
        req.add_names(name);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {std::vector<Arc>(), err};

    std::vector<Arc> arcs;
    arcs.reserve(res.arcs_size());
    for (const auto &pb : res.arcs())
        arcs.emplace_back(pb);

    return {arcs, xerrors::NIL};
}

std::pair<std::vector<Arc>, xerrors::Error> ArcClient::retrieve_by_keys(
    const std::vector<std::string> &keys
) const {
    auto req = api::v1::ArcRetrieveRequest();
    for (const auto &key : keys)
        req.add_keys(key);

    auto [res, err] = retrieve_client->send(ARC_RETRIEVE_ENDPOINT, req);
    if (err) return {std::vector<Arc>(), err};

    std::vector<Arc> arcs;
    arcs.reserve(res.arcs_size());
    for (const auto &pb : res.arcs())
        arcs.emplace_back(pb);

    return {arcs, xerrors::NIL};
}

// ========================================================================
// Delete Operations
// ========================================================================

xerrors::Error ArcClient::delete_arc(const std::string &key) const {
    auto req = api::v1::ArcDeleteRequest();
    req.add_keys(key);

    auto [_, err] = delete_client->send(ARC_DELETE_ENDPOINT, req);
    return err;
}

xerrors::Error ArcClient::delete_arc(const std::vector<std::string> &keys) const {
    auto req = api::v1::ArcDeleteRequest();
    for (const auto &key : keys)
        req.add_keys(key);

    auto [_, err] = delete_client->send(ARC_DELETE_ENDPOINT, req);
    return err;
}

} // namespace synnax
