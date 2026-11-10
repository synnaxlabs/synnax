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
#include "client/cpp/icd/icd.h"
#include "client/cpp/icd/proto.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"

namespace synnax::icd {
Client::Client(
    std::unique_ptr<CreateClient> create_client,
    std::unique_ptr<RetrieveClient> retrieve_client,
    std::unique_ptr<DeleteClient> delete_client
):
    create_client(std::move(create_client)),
    retrieve_client(std::move(retrieve_client)),
    delete_client(std::move(delete_client)) {}

std::pair<ICD, x::errors::Error>
Client::retrieve(const Key &key, const RetrieveOptions &options) const {
    return this->retrieve(Where{.key = key}, x::filter::one, options);
}

std::pair<std::vector<ICD>, x::errors::Error>
Client::retrieve(const std::vector<Key> &keys, const RetrieveOptions &options) const {
    if (keys.empty()) return {std::vector<ICD>(), x::errors::NIL};
    return this->retrieve(Where{.key = keys}, options);
}

std::pair<std::vector<ICD>, x::errors::Error>
Client::retrieve(const Where &where, const RetrieveOptions &options) const {
    auto [pb_req, pb_err] = RetrieveRequest::from(where, options).to_proto();
    if (pb_err) return {std::vector<ICD>(), pb_err};
    grpc::icd::RetrieveRequest grpc_req;
    *grpc_req.mutable_request() = pb_req;
    auto [res, err] = this->retrieve_client->send("/icd/retrieve", grpc_req);
    if (err) return {std::vector<ICD>(), err};
    auto [decoded, proto_err] = RetrieveResponse::from_proto(res);
    if (proto_err) return {std::vector<ICD>(), proto_err};
    return {std::move(decoded.icds), x::errors::NIL};
}

std::pair<ICD, x::errors::Error> Client::retrieve(
    const Where &where,
    x::filter::One,
    const RetrieveOptions &options
) const {
    if (options.limit.has_value() || options.after.has_value())
        return {ICD{}, errors::paged_one_error()};
    auto [ics, err] = this->retrieve(where, options);
    if (err) return {ICD{}, err};
    if (const auto one_err = errors::check_one("ICD", where.to_json(), ics.size()))
        return {ICD{}, one_err};
    return {std::move(ics[0]), x::errors::NIL};
}

x::errors::Error Client::create(ICD &ic) const {
    auto req = grpc::icd::CreateRequest();
    auto [pb, pb_err] = ic.to_proto();
    if (pb_err) return pb_err;
    *req.add_icds() = pb;
    auto [res, err] = this->create_client->send("/icd/create", req);
    if (err) return err;
    if (res.icds_size() == 0) return errors::unexpected_missing_error("ICD");
    auto [created, proto_err] = ICD::from_proto(res.icds(0));
    if (proto_err) return proto_err;
    ic = std::move(created);
    return x::errors::NIL;
}

x::errors::Error Client::create(std::vector<ICD> &ics) const {
    auto req = grpc::icd::CreateRequest();
    req.mutable_icds()->Reserve(static_cast<int>(ics.size()));
    for (const auto &ic: ics) {
        auto [pb, pb_err] = ic.to_proto();
        if (pb_err) return pb_err;
        *req.add_icds() = pb;
    }
    auto [res, err] = this->create_client->send("/icd/create", req);
    if (err) return err;
    const auto n = std::min(res.icds_size(), static_cast<int>(ics.size()));
    for (int i = 0; i < n; i++) {
        auto [created, proto_err] = ICD::from_proto(res.icds(i));
        if (proto_err) return proto_err;
        ics[i] = std::move(created);
    }
    return x::errors::NIL;
}

x::errors::Error Client::del(const Key &key) const {
    auto req = grpc::icd::DeleteRequest();
    req.add_keys(key.to_string());
    auto [_, err] = this->delete_client->send("/icd/delete", req);
    return err;
}

x::errors::Error Client::del(const std::vector<Key> &keys) const {
    auto req = grpc::icd::DeleteRequest();
    for (const auto &k: keys)
        req.add_keys(k.to_string());
    auto [_, err] = this->delete_client->send("/icd/delete", req);
    return err;
}
}
