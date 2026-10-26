// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/device/device.h"
#include "client/cpp/device/proto.gen.h"
#include "client/cpp/device/types.gen.h"
#include "client/cpp/errors/errors.h"
#include "client/cpp/rack/rack.h"
#include "x/cpp/errors/errors.h"

namespace synnax::device {
Client::Client(
    std::unique_ptr<CreateClient> device_create_client,
    std::unique_ptr<RetrieveClient> device_retrieve_client,
    std::unique_ptr<DeleteClient> device_delete_client
):
    device_create_client(std::move(device_create_client)),
    device_retrieve_client(std::move(device_retrieve_client)),
    device_delete_client(std::move(device_delete_client)) {}

std::pair<Device, x::errors::Error> Client::retrieve(const std::string &key) const {
    return this->retrieve(key, RetrieveOptions{});
}

std::pair<Device, x::errors::Error>
Client::retrieve(const std::string &key, const RetrieveOptions &options) const {
    return this->retrieve_one(
        {
            .where = Where{.key = x::filter::Set<Key>{.eq = key}},
            .include_status = options.include_status,
            .include_parent = options.include_parent,
        },
        "key " + key
    );
}

std::pair<std::vector<Device>, x::errors::Error>
Client::retrieve(const std::vector<std::string> &keys) const {
    if (keys.empty()) return {std::vector<Device>(), x::errors::NIL};
    return this->retrieve(keys, RetrieveOptions{});
}

std::pair<std::vector<Device>, x::errors::Error> Client::retrieve(
    const std::vector<std::string> &keys,
    const RetrieveOptions &options
) const {
    return this->retrieve({
        .where = Where{.key = x::filter::Set<Key>{.in = keys}},
        .include_status = options.include_status,
        .include_parent = options.include_parent,
    });
}

std::pair<std::vector<Device>, x::errors::Error>
Client::retrieve(const RetrieveRequest &req) const {
    auto [pb_req, pb_err] = req.to_proto();
    if (pb_err) return {std::vector<Device>(), pb_err};
    grpc::device::RetrieveRequest grpc_req;
    *grpc_req.mutable_request() = pb_req;
    auto [res, err] = device_retrieve_client->send("/device/retrieve", grpc_req);
    if (err) return {std::vector<Device>(), err};
    auto [decoded, proto_err] = RetrieveResponse::from_proto(res);
    if (proto_err) return {std::vector<Device>(), proto_err};
    return {std::move(decoded.devices), x::errors::NIL};
}

std::pair<Device, x::errors::Error>
Client::retrieve_one(const RetrieveRequest &req, const std::string &query) const {
    auto [devices, err] = this->retrieve(req);
    if (err) return {Device{}, err};
    if (devices.empty()) return {Device{}, errors::not_found_error("device", query)};
    return {std::move(devices[0]), x::errors::NIL};
}

x::errors::Error Client::create(Device &device) const {
    auto req = grpc::device::CreateRequest();
    auto [pb, pb_err] = device.to_proto();
    if (pb_err) return pb_err;
    *req.add_devices() = pb;
    auto [res, err] = device_create_client->send("/device/create", req);
    if (err) return err;
    if (res.devices_size() == 0) return errors::unexpected_missing_error("device");
    device.key = res.devices().at(0).key();
    return err;
}

x::errors::Error Client::create(const std::vector<Device> &devs) const {
    auto req = grpc::device::CreateRequest();
    req.mutable_devices()->Reserve(static_cast<int>(devs.size()));
    for (const auto &device: devs) {
        auto [pb, pb_err] = device.to_proto();
        if (pb_err) return pb_err;
        *req.add_devices() = pb;
    }
    auto [res, err] = device_create_client->send("/device/create", req);
    return err;
}

x::errors::Error Client::del(const std::string &key) const {
    auto req = grpc::device::DeleteRequest();
    req.add_keys(key);
    auto [res, err] = device_delete_client->send("/device/delete", req);
    return err;
}

x::errors::Error Client::del(const std::vector<std::string> &keys) const {
    auto req = grpc::device::DeleteRequest();
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, err] = device_delete_client->send("/device/delete", req);
    return err;
}
}
