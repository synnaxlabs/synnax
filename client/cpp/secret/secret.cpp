// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/errors/errors.h"
#include "client/cpp/secret/secret.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/uuid/uuid.h"

namespace synnax::secret {
namespace {
std::pair<Values, x::errors::Error>
to_values(const grpc::secret::ResolveResponse &res) {
    Values values;
    for (const auto &v: res.values()) {
        auto [key, err] = x::uuid::UUID::parse(v.key());
        if (err) return {Values{}, err};
        values[key] = v.value();
    }
    return {std::move(values), x::errors::NIL};
}
}

Client::Client(
    std::shared_ptr<RetrieveClient> retrieve_client,
    std::shared_ptr<ResolveClient> resolve_client
):
    retrieve_client(std::move(retrieve_client)),
    resolve_client(std::move(resolve_client)) {}

std::pair<Secret, x::errors::Error> Client::retrieve(const Key &key) const {
    auto [secrets, err] = this->retrieve(std::vector{key});
    if (err) return {Secret{}, err};
    if (const auto one_err = errors::check_one(
            "secret",
            Where{.key = key}.to_json(),
            "",
            secrets.size()
        ))
        return {Secret{}, one_err};
    return {std::move(secrets[0]), x::errors::NIL};
}

std::pair<std::vector<Secret>, x::errors::Error>
Client::retrieve(const std::vector<Key> &keys) const {
    auto [req, pb_err] = RetrieveRequest::from(Where{.key = keys}).to_proto();
    if (pb_err) return {{}, pb_err};
    auto [res, err] = this->retrieve_client->send("/secret/retrieve", req);
    if (err) return {{}, err};
    auto [decoded, decode_err] = RetrieveResponse::from_proto(res);
    if (decode_err) return {{}, decode_err};
    return {std::move(decoded.secrets), x::errors::NIL};
}

std::pair<Values, x::errors::Error>
Client::resolve(const std::vector<Key> &keys) const {
    grpc::secret::ResolveRequest req;
    for (const auto &key: keys)
        req.add_secrets(key.to_string());
    auto [res, err] = this->resolve_client->send("/secret/resolve", req);
    if (err) return {Values{}, err};
    return to_values(res);
}
}
