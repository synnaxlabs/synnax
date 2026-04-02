// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/errors/errors.h"
#include "client/cpp/view/proto.gen.h"
#include "client/cpp/view/types.gen.h"
#include "client/cpp/view/view.h"
#include "x/cpp/errors/errors.h"

namespace synnax::view {
Client::Client(
    std::unique_ptr<CreateClient> create_client,
    std::unique_ptr<RetrieveClient> retrieve_client,
    std::unique_ptr<DeleteClient> delete_client
):
    create_client(std::move(create_client)),
    retrieve_client(std::move(retrieve_client)),
    delete_client(std::move(delete_client)) {}

std::pair<View, x::errors::Error> Client::retrieve(const Key &key) const {
    auto req = grpc::view::RetrieveRequest();
    req.add_keys(key.to_string());
    auto [res, err] = this->retrieve_client->send("/view/retrieve", req);
    if (err) return {View(), err};
    if (res.views_size() == 0)
        return {View(), errors::not_found_error("view", "key " + key.to_string())};
    auto [v, proto_err] = View::from_proto(res.views(0));
    if (proto_err) return {View(), proto_err};
    return {std::move(v), x::errors::NIL};
}

std::pair<std::vector<View>, x::errors::Error>
Client::retrieve(const std::vector<Key> &keys) const {
    if (keys.empty()) return {std::vector<View>(), x::errors::NIL};
    auto req = grpc::view::RetrieveRequest();
    for (const auto &k: keys)
        req.add_keys(k.to_string());
    auto [res, err] = this->retrieve_client->send("/view/retrieve", req);
    if (err) return {std::vector<View>(), err};
    std::vector<View> views;
    views.reserve(res.views_size());
    for (const auto &v: res.views()) {
        auto [pld, proto_err] = View::from_proto(v);
        if (proto_err) return {std::vector<View>(), proto_err};
        views.push_back(std::move(pld));
    }
    return {views, x::errors::NIL};
}

x::errors::Error Client::create(View &view) const {
    auto req = grpc::view::CreateRequest();
    auto [pb, pb_err] = view.to_proto();
    if (pb_err) return pb_err;
    *req.add_views() = pb;
    auto [res, err] = this->create_client->send("/view/create", req);
    if (err) return err;
    if (res.views_size() == 0) return errors::unexpected_missing_error("view");
    auto [parsed_key, parse_err] = x::uuid::UUID::parse(res.views().at(0).key());
    if (parse_err) return parse_err;
    view.key = parsed_key;
    return x::errors::NIL;
}

x::errors::Error Client::create(std::vector<View> &views) const {
    auto req = grpc::view::CreateRequest();
    req.mutable_views()->Reserve(static_cast<int>(views.size()));
    for (const auto &view: views) {
        auto [pb, pb_err] = view.to_proto();
        if (pb_err) return pb_err;
        *req.add_views() = pb;
    }
    auto [res, err] = this->create_client->send("/view/create", req);
    if (err) return err;
    for (int i = 0; i < res.views_size() && i < static_cast<int>(views.size()); i++) {
        auto [parsed_key, parse_err] = x::uuid::UUID::parse(res.views(i).key());
        if (parse_err) return parse_err;
        views[i].key = parsed_key;
    }
    return x::errors::NIL;
}

x::errors::Error Client::del(const Key &key) const {
    auto req = grpc::view::DeleteRequest();
    req.add_keys(key.to_string());
    auto [_, err] = this->delete_client->send("/view/delete", req);
    return err;
}

x::errors::Error Client::del(const std::vector<Key> &keys) const {
    auto req = grpc::view::DeleteRequest();
    for (const auto &k: keys)
        req.add_keys(k.to_string());
    auto [_, err] = this->delete_client->send("/view/delete", req);
    return err;
}
}
