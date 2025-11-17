// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/status/status.h"

namespace synnax {

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
    auto [res, err] = this->set_client->send("/status/set", req);
    if (err) return err;
    if (res.statuses_size() > 0) { status = Status(res.statuses(0)); }
    return xerrors::NIL;
}

xerrors::Error StatusClient::set(std::vector<Status> &statuses) const {
    api::v1::StatusSetRequest req;
    for (const auto &status: statuses) {
        status.to_proto(req.add_statuses());
    }
    auto [res, err] = this->set_client->send("/status/set", req);
    if (err) return err;
    for (int i = 0; i < res.statuses_size(); i++) {
        statuses[i] = Status(res.statuses(i));
    }
    return xerrors::NIL;
}

std::pair<Status, xerrors::Error> StatusClient::retrieve(const std::string &key) const {
    auto [statuses, err] = this->retrieve(std::vector<std::string>{key});
    if (err) return {Status{}, err};
    if (statuses.empty()) {
        return {Status{}, xerrors::Error("Status with key '" + key + "' not found")};
    }
    return {statuses[0], xerrors::NIL};
}

std::pair<std::vector<Status>, xerrors::Error>
StatusClient::retrieve(const std::vector<std::string> &keys) const {
    api::v1::StatusRetrieveRequest req;
    for (const auto &key: keys) {
        req.add_keys(key);
    }
    auto [res, err] = this->retrieve_client->send("/status/retrieve", req);
    if (err) return {std::vector<Status>(), err};
    std::vector<Status> statuses;
    statuses.reserve(res.statuses_size());
    for (const auto &st: res.statuses()) {
        statuses.push_back(Status(st));
    }
    return {statuses, xerrors::NIL};
}

xerrors::Error StatusClient::del(const std::string &key) const {
    return this->del(std::vector<std::string>{key});
}

xerrors::Error StatusClient::del(const std::vector<std::string> &keys) const {
    api::v1::StatusDeleteRequest req;
    for (const auto &key: keys) {
        req.add_keys(key);
    }
    auto [res, err] = this->delete_client->send("/status/delete", req);
    return err;
}

} // namespace synnax
