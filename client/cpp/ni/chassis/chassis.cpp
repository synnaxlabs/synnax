// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/errors/errors.h"
#include "client/cpp/ni/chassis/chassis.h"

namespace synnax::ni::chassis {
Client::Client(
    std::unique_ptr<CreateClient> create_client,
    std::unique_ptr<RetrieveClient> retrieve_client,
    std::unique_ptr<UpdateClient> update_client,
    std::unique_ptr<DeleteClient> delete_client
):
    create_client(std::move(create_client)),
    retrieve_client(std::move(retrieve_client)),
    update_client(std::move(update_client)),
    delete_client(std::move(delete_client)) {}

x::errors::Error Client::create(std::vector<Chassis> &chassis) const {
    grpc::ni_chassis::CreateRequest req;
    for (const auto &dev: chassis) {
        auto [pb, err] = dev.to_proto();
        if (err) return err;
        *req.add_chassis() = pb;
    }
    auto [res, err] = this->create_client->send("/ni/chassis/create", req);
    if (err) return err;
    if (res.chassis_size() != static_cast<int>(chassis.size()))
        return errors::unexpected_missing_error("chassis");
    for (std::size_t i = 0; i < chassis.size(); i++) {
        auto [key, key_err] = x::uuid::UUID::parse(res.chassis(i).key());
        if (key_err) return key_err;
        chassis[i].key = key;
    }
    return x::errors::NIL;
}

x::errors::Error Client::create(Chassis &chassis) const {
    std::vector list{chassis};
    if (const auto err = this->create(list)) return err;
    chassis.key = list[0].key;
    return x::errors::NIL;
}

std::pair<Chassis, x::errors::Error>
Client::retrieve(const Key &key, const RetrieveOptions &options) const {
    return this->retrieve(Where{.key = key}, x::filter::one, options);
}

std::pair<std::vector<Chassis>, x::errors::Error>
Client::retrieve(const Where &where, const RetrieveOptions &options) const {
    auto [req, req_err] = RetrieveRequest::from(where, options).to_proto();
    if (req_err) return {{}, req_err};
    auto [res, err] = this->retrieve_client->send("/ni/chassis/retrieve", req);
    if (err) return {{}, err};
    auto [decoded, decode_err] = RetrieveResponse::from_proto(res);
    if (decode_err) return {{}, decode_err};
    return {std::move(decoded.chassis), x::errors::NIL};
}

std::pair<Chassis, x::errors::Error> Client::retrieve(
    const Where &where,
    x::filter::One,
    const RetrieveOptions &options
) const {
    if (options.limit.has_value() || options.after.has_value())
        return {Chassis{}, errors::paged_one_error()};
    auto [chassis, err] = this->retrieve(where, options);
    if (err) return {Chassis{}, err};
    if (const auto
            one_err = errors::check_one("chassis", where.to_json(), chassis.size()))
        return {Chassis{}, one_err};
    return {std::move(chassis[0]), x::errors::NIL};
}

x::errors::Error Client::update(const Key &key, const Update &update) const {
    grpc::ni_chassis::UpdateRequest req;
    req.set_key(key.to_string());
    auto [pb, err] = to_proto(update);
    if (err) return err;
    *req.mutable_update() = pb;
    return this->update_client->send("/ni/chassis/update", req).second;
}

x::errors::Error Client::del(const std::vector<Key> &keys) const {
    grpc::ni_chassis::DeleteRequest req;
    for (const auto &key: keys)
        req.add_keys(key.to_string());
    return this->delete_client->send("/ni/chassis/delete", req).second;
}
}
