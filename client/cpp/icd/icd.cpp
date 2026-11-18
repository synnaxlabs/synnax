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

#include "client/cpp/errors/errors.h"
#include "client/cpp/icd/icd.h"

namespace synnax::icd {
std::pair<ICD, x::errors::Error> Client::retrieve(const Key &key) const {
    const Where where{.key = key};
    auto [icds, err] = this->retrieve(where);
    if (err) return {ICD{}, err};
    if (const auto one_err = errors::check_one("ICD", where.to_json(), "", icds.size()))
        return {ICD{}, one_err};
    return {std::move(icds[0]), x::errors::NIL};
}

std::pair<std::vector<ICD>, x::errors::Error>
Client::retrieve(const std::vector<Key> &keys) const {
    if (keys.empty()) return {std::vector<ICD>(), x::errors::NIL};
    return this->retrieve(Where{.key = keys});
}

std::pair<std::vector<ICD>, x::errors::Error>
Client::retrieve(const Where &where) const {
    auto [req, req_err] = RetrieveRequest::from(where).to_proto();
    if (req_err) return {std::vector<ICD>(), req_err};
    auto [res, err] = this->retrieve_client->send("/icd/retrieve", req);
    if (err) return {std::vector<ICD>(), err};
    auto [decoded, decode_err] = RetrieveResponse::from_proto(res);
    if (decode_err) return {std::vector<ICD>(), decode_err};
    return {std::move(decoded.icds), x::errors::NIL};
}

x::errors::Error Client::create(ICD &ic) const {
    grpc::icd::CreateRequest req;
    auto [pb, pb_err] = ic.to_proto();
    if (pb_err) return pb_err;
    *req.add_icds() = pb;
    auto [res, err] = this->create_client->send("/icd/create", req);
    if (err) return err;
    if (res.icds_size() == 0) return errors::unexpected_missing_error("ICD");
    auto [created, created_err] = ICD::from_proto(res.icds(0));
    if (created_err) return created_err;
    ic = std::move(created);
    return x::errors::NIL;
}
}
