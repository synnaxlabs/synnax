// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/device/device.h"
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
    auto req = grpc::device::RetrieveRequest();
    req.add_keys(key);
    auto [res, err] = device_retrieve_client->send("/device/retrieve", req);
    if (err) return {Device(), err};
    if (res.devices_size() == 0)
        return {Device(), not_found_error("device", "key " + key)};
    auto [pld, proto_err] = Payload::from_proto(res.devices(0));
    if (proto_err) return {Device(), proto_err};
    return {Device(std::move(pld)), x::errors::NIL};
}

std::pair<Device, x::errors::Error>
Client::retrieve(const std::string &key, const RetrieveOptions &options) const {
    auto req = grpc::device::RetrieveRequest();
    req.add_keys(key);
    req.set_include_status(options.include_status);
    auto [res, err] = device_retrieve_client->send("/device/retrieve", req);
    if (err) return {Device(), err};
    if (res.devices_size() == 0)
        return {Device(), not_found_error("device", "key " + key)};
    auto [pld, proto_err] = Payload::from_proto(res.devices(0));
    if (proto_err) return {Device(), proto_err};
    return {Device(std::move(pld)), x::errors::NIL};
}

std::pair<std::vector<Device>, x::errors::Error>
Client::retrieve(const std::vector<std::string> &keys) const {
    if (keys.empty()) return {std::vector<Device>(), x::errors::NIL};
    RetrieveRequest req;
    req.keys = keys;
    return retrieve(req);
}

std::pair<std::vector<Device>, x::errors::Error> Client::retrieve(
    const std::vector<std::string> &keys,
    const RetrieveOptions &options
) const {
    RetrieveRequest req;
    req.keys = keys;
    req.include_status = options.include_status;
    return retrieve(req);
}

std::pair<std::vector<Device>, x::errors::Error>
Client::retrieve(RetrieveRequest &req) const {
    auto api_req = grpc::device::RetrieveRequest();
    req.to_proto(api_req);
    auto [res, err] = device_retrieve_client->send("/device/retrieve", api_req);
    if (err) return {std::vector<Device>(), err};
    std::vector<Device> devices;
    devices.reserve(res.devices_size());
    for (const auto &d: res.devices()) {
        auto [pld, proto_err] = Payload::from_proto(d);
        if (proto_err) return {std::vector<Device>(), proto_err};
        devices.push_back(Device(std::move(pld)));
    }
    return {devices, x::errors::NIL};
}

x::errors::Error Client::create(Device &device) const {
    auto req = grpc::device::CreateRequest();
    *req.add_devices() = device.to_proto();
    auto [res, err] = device_create_client->send("/device/create", req);
    if (err) return err;
    if (res.devices_size() == 0) return unexpected_missing_error("device");
    device.key = res.devices().at(0).key();
    return err;
}

x::errors::Error Client::create(const std::vector<Device> &devs) const {
    auto req = grpc::device::CreateRequest();
    req.mutable_devices()->Reserve(static_cast<int>(devs.size()));
    for (const auto &device: devs)
        *req.add_devices() = device.to_proto();
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

Device::Device(
    std::string key,
    std::string name,
    RackKey rack,
    std::string location,
    std::string make,
    std::string model,
    std::string properties
) {
    this->key = std::move(key);
    this->name = std::move(name);
    this->rack = rack;
    this->location = std::move(location);
    this->make = std::move(make);
    this->model = std::move(model);
    this->properties = std::move(properties);
}
}
