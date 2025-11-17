// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/status.pb.h"

#include "client/cpp/status/status.h"

namespace synnax {
const std::string SET_ENDPOINT = "/api/v1/status/set";
const std::string DELETE_ENDPOINT = "/api/v1/status/delete";

StatusClient::StatusClient(
    std::shared_ptr<StatusRetrieveClient> retrieve_client,
    std::shared_ptr<StatusSetClient> set_client,
    std::shared_ptr<StatusDeleteClient> delete_client
):
    retrieve_client(std::move(retrieve_client)),
    set_client(std::move(set_client)),
    delete_client(std::move(delete_client)) {}

xerrors::Error StatusClient::set(Status &status) const {
    api::v1::StatusSetRequest req;
    status.to_proto(req.add_statuses());
    auto [res, err] = this->set_client->send(SET_ENDPOINT, req);
    if (err) return err;
    if (res.statuses_size() > 0) {
        auto [decoded, decode_err] = Status::from_proto(res.statuses(0));
        if (decode_err) return decode_err;
        status = decoded;
    }
    return xerrors::NIL;
}

xerrors::Error StatusClient::set(std::vector<Status> &statuses) const {
    api::v1::StatusSetRequest req;
    req.mutable_statuses()->Reserve(static_cast<int>(statuses.size()));
    for (const auto &status: statuses)
        status.to_proto(req.add_statuses());
    auto [res, err] = this->set_client->send(SET_ENDPOINT, req);
    if (err) return err;
    for (int i = 0; i < res.statuses_size(); i++) {
        auto [decoded, decode_err] = Status::from_proto(res.statuses(i));
        if (decode_err) return decode_err;
        statuses[i] = decoded;
    }
    return xerrors::NIL;
}

std::pair<Status, xerrors::Error> StatusClient::retrieve(const std::string &key) const {
    auto [statuses, err] = this->retrieve(std::vector<std::string>{key});
    if (err) return {Status{}, err};
    if (statuses.empty()) {
        return {
            Status(),
            xerrors::Error(xerrors::NOT_FOUND, "no statuses found matching key" + key)
        };
    }
    return {statuses[0], xerrors::NIL};
}

std::pair<std::vector<Status>, xerrors::Error>
StatusClient::retrieve(const std::vector<std::string> &keys) const {
    api::v1::StatusRetrieveRequest req;
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, err] = this->retrieve_client->send("/status/retrieve", req);
    if (err) return {std::vector<Status>(), err};
    std::vector<Status> statuses;
    statuses.reserve(res.statuses_size());
    for (const auto &pb_status : res.statuses()) {
        auto [decoded, decode_err] = Status::from_proto(pb_status);
        if (decode_err) return {std::vector<Status>(), decode_err};
        statuses.push_back(decoded);
    }
    return {statuses, xerrors::NIL};
}

xerrors::Error StatusClient::del(const std::string &key) const {
    return this->del(std::vector{key});
}

xerrors::Error StatusClient::del(const std::vector<std::string> &keys) const {
    api::v1::StatusDeleteRequest req;
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, err] = this->delete_client->send(DELETE_ENDPOINT, req);
    return err;
}

} // namespace synnax
