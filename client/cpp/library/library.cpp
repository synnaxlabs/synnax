// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "client/cpp/errors/errors.h"
#include "client/cpp/library/library.h"
#include "client/cpp/library/proto.gen.h"
#include "client/cpp/library/types.gen.h"
#include "x/cpp/errors/errors.h"

namespace synnax::library {
Client::Client(
    std::unique_ptr<CreateClient> create_client,
    std::unique_ptr<RetrieveClient> retrieve_client,
    std::unique_ptr<DeleteClient> delete_client
):
    create_client(std::move(create_client)),
    retrieve_client(std::move(retrieve_client)),
    delete_client(std::move(delete_client)) {}

std::pair<Library, x::errors::Error> Client::retrieve(const Key &key) const {
    auto req = grpc::library::RetrieveRequest();
    req.add_keys(key.to_string());
    auto [res, err] = this->retrieve_client->send("/library/retrieve", req);
    if (err) return {Library(), err};
    if (res.libraries_size() == 0)
        return {
            Library(),
            errors::not_found_error("library", "key " + key.to_string())
        };
    auto [lib, proto_err] = Library::from_proto(res.libraries(0));
    if (proto_err) return {Library(), proto_err};
    return {std::move(lib), x::errors::NIL};
}

std::pair<std::vector<Library>, x::errors::Error>
Client::retrieve(const std::vector<Key> &keys) const {
    if (keys.empty()) return {std::vector<Library>(), x::errors::NIL};
    auto req = grpc::library::RetrieveRequest();
    for (const auto &k: keys)
        req.add_keys(k.to_string());
    auto [res, err] = this->retrieve_client->send("/library/retrieve", req);
    if (err) return {std::vector<Library>(), err};
    std::vector<Library> libs;
    libs.reserve(res.libraries_size());
    for (const auto &pb: res.libraries()) {
        auto [lib, proto_err] = Library::from_proto(pb);
        if (proto_err) return {std::vector<Library>(), proto_err};
        libs.push_back(std::move(lib));
    }
    return {libs, x::errors::NIL};
}

x::errors::Error Client::create(Library &lib) const {
    auto req = grpc::library::CreateRequest();
    auto [pb, pb_err] = lib.to_proto();
    if (pb_err) return pb_err;
    *req.add_libraries() = pb;
    auto [res, err] = this->create_client->send("/library/create", req);
    if (err) return err;
    if (res.libraries_size() == 0) return errors::unexpected_missing_error("library");
    auto [created, proto_err] = Library::from_proto(res.libraries(0));
    if (proto_err) return proto_err;
    lib = std::move(created);
    return x::errors::NIL;
}

x::errors::Error Client::create(std::vector<Library> &libs) const {
    auto req = grpc::library::CreateRequest();
    req.mutable_libraries()->Reserve(static_cast<int>(libs.size()));
    for (const auto &lib: libs) {
        auto [pb, pb_err] = lib.to_proto();
        if (pb_err) return pb_err;
        *req.add_libraries() = pb;
    }
    auto [res, err] = this->create_client->send("/library/create", req);
    if (err) return err;
    const auto n = std::min(res.libraries_size(), static_cast<int>(libs.size()));
    for (int i = 0; i < n; i++) {
        auto [created, proto_err] = Library::from_proto(res.libraries(i));
        if (proto_err) return proto_err;
        libs[i] = std::move(created);
    }
    return x::errors::NIL;
}

x::errors::Error Client::del(const Key &key) const {
    auto req = grpc::library::DeleteRequest();
    req.add_keys(key.to_string());
    auto [_, err] = this->delete_client->send("/library/delete", req);
    return err;
}

x::errors::Error Client::del(const std::vector<Key> &keys) const {
    auto req = grpc::library::DeleteRequest();
    for (const auto &k: keys)
        req.add_keys(k.to_string());
    auto [_, err] = this->delete_client->send("/library/delete", req);
    return err;
}
}
