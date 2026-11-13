// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/errors/errors.h"
#include "client/cpp/ni/device/device.h"

namespace synnax::ni::device {
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

x::errors::Error Client::create(std::vector<Device> &devices) const {
    grpc::ni_device::CreateRequest req;
    for (const auto &dev: devices) {
        auto [pb, err] = dev.to_proto();
        if (err) return err;
        *req.add_devices() = pb;
    }
    auto [res, err] = this->create_client->send("/ni/device/create", req);
    if (err) return err;
    if (res.devices_size() != static_cast<int>(devices.size()))
        return errors::unexpected_missing_error("device");
    for (std::size_t i = 0; i < devices.size(); i++) {
        auto [key, key_err] = x::uuid::UUID::parse(res.devices(i).key());
        if (key_err) return key_err;
        devices[i].key = key;
    }
    return x::errors::NIL;
}

x::errors::Error Client::create(Device &device) const {
    std::vector devices{device};
    if (const auto err = this->create(devices)) return err;
    device.key = devices[0].key;
    return x::errors::NIL;
}

std::pair<Device, x::errors::Error>
Client::retrieve(const Key &key, const RetrieveOptions &options) const {
    return this->retrieve(Where{.key = key}, x::filter::one, options);
}

std::pair<std::vector<Device>, x::errors::Error>
Client::retrieve(const Where &where, const RetrieveOptions &options) const {
    auto [req, req_err] = RetrieveRequest::from(where, options).to_proto();
    if (req_err) return {{}, req_err};
    auto [res, err] = this->retrieve_client->send("/ni/device/retrieve", req);
    if (err) return {{}, err};
    auto [decoded, decode_err] = RetrieveResponse::from_proto(res);
    if (decode_err) return {{}, decode_err};
    return {std::move(decoded.devices), x::errors::NIL};
}

std::pair<Device, x::errors::Error> Client::retrieve(
    const Where &where,
    x::filter::One,
    const RetrieveOptions &options
) const {
    if (options.limit.has_value() || options.after.has_value())
        return {Device{}, errors::paged_one_error()};
    auto [devices, err] = this->retrieve(where, options);
    if (err) return {Device{}, err};
    if (const auto
            one_err = errors::check_one("device", where.to_json(), devices.size()))
        return {Device{}, one_err};
    return {std::move(devices[0]), x::errors::NIL};
}

x::errors::Error Client::update(const Key &key, const Update &update) const {
    grpc::ni_device::UpdateRequest req;
    req.set_key(key.to_string());
    auto [pb, err] = to_proto(update);
    if (err) return err;
    *req.mutable_update() = pb;
    return this->update_client->send("/ni/device/update", req).second;
}

x::errors::Error Client::del(const std::vector<Key> &keys) const {
    grpc::ni_device::DeleteRequest req;
    for (const auto &key: keys)
        req.add_keys(key.to_string());
    return this->delete_client->send("/ni/device/delete", req).second;
}
}
