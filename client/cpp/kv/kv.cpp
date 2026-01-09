// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <utility>

#include "client/cpp/errors/errors.h"
#include "client/cpp/kv/kv.h"
#include "x/cpp/errors/errors.h"

namespace synnax::kv {
std::pair<std::string, x::errors::Error> Client::get(const std::string &key) const {
    auto req = grpc::kv::GetRequest();
    req.add_keys(key);
    req.set_range(this->range_key);
    auto [res, err] = this->kv_get_client->send("/range/kv/get", req);
    if (err) return {"", err};
    if (res.pairs_size() == 0)
        return {"", not_found_error("range key-value pair", "key " + key)};
    return {res.pairs().at(0).value(), err};
}

x::errors::Error Client::set(const std::string &key, const std::string &value) const {
    auto req = grpc::kv::SetRequest();
    req.set_range(this->range_key);
    const auto pair = req.add_pairs();
    pair->set_key(key);
    pair->set_value(value);
    auto [res, err] = this->kv_set_client->send("/range/kv/set", req);
    return err;
}

x::errors::Error Client::del(const std::string &key) const {
    auto req = grpc::kv::DeleteRequest();
    req.set_range(this->range_key);
    req.add_keys(key);
    auto [res, err] = this->kv_delete_client->send("/range/kv/delete", req);
    return err;
}
}
