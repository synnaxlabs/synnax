// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/device/device.h"
#include "client/cpp/errors/errors.h"
#include "x/cpp/xerrors/errors.h"

namespace synnax {
DeviceClient::DeviceClient(
    std::unique_ptr<DeviceCreateClient> device_create_client,
    std::unique_ptr<DeviceRetrieveClient> device_retrieve_client,
    std::unique_ptr<DeviceDeleteClient> device_delete_client
):
    device_create_client(std::move(device_create_client)),
    device_retrieve_client(std::move(device_retrieve_client)),
    device_delete_client(std::move(device_delete_client)) {}

std::pair<Device, xerrors::Error>
DeviceClient::retrieve(const std::string &key, bool ignore_not_found) const {
    auto req = api::v1::DeviceRetrieveRequest();
    req.add_keys(key);
    req.set_ignore_not_found(ignore_not_found);
    auto [res, err] = device_retrieve_client->send("/device/retrieve", req);
    if (err) return {Device(), err};
    if (res.devices_size() == 0) {
        if (ignore_not_found) return {Device(), xerrors::NIL};
        return {Device(), not_found_error("device", "key " + key)};
    }
    return Device::from_proto(res.devices(0));
}

std::pair<Device, xerrors::Error> DeviceClient::retrieve(
    const std::string &key,
    const DeviceRetrieveOptions &options
) const {
    auto req = api::v1::DeviceRetrieveRequest();
    req.add_keys(key);
    req.set_include_status(options.include_status);
    auto [res, err] = device_retrieve_client->send("/device/retrieve", req);
    if (err) return {Device(), err};
    if (res.devices_size() == 0)
        return {Device(), not_found_error("device", "key " + key)};
    return Device::from_proto(res.devices(0));
}

std::pair<std::vector<Device>, xerrors::Error> DeviceClient::retrieve(
    const std::vector<std::string> &keys,
    bool ignore_not_found
) const {
    DeviceRetrieveRequest req;
    req.keys = keys;
    req.ignore_not_found = ignore_not_found;
    return retrieve(req);
}

std::pair<std::vector<Device>, xerrors::Error> DeviceClient::retrieve(
    const std::vector<std::string> &keys,
    const DeviceRetrieveOptions &options
) const {
    DeviceRetrieveRequest req;
    req.keys = keys;
    req.include_status = options.include_status;
    return retrieve(req);
}

std::pair<std::vector<Device>, xerrors::Error>
DeviceClient::retrieve(DeviceRetrieveRequest &req) const {
    auto api_req = api::v1::DeviceRetrieveRequest();
    req.to_proto(api_req);
    auto [res, err] = device_retrieve_client->send("/device/retrieve", api_req);
    if (err) return {std::vector<Device>(), err};
    std::vector<Device> devices;
    devices.reserve(res.devices_size());
    for (const auto &d: res.devices()) {
        auto [device, proto_err] = Device::from_proto(d);
        if (proto_err) return {std::vector<Device>(), proto_err};
        devices.push_back(std::move(device));
    }
    return {devices, xerrors::NIL};
}

xerrors::Error DeviceClient::create(Device &device) const {
    auto req = api::v1::DeviceCreateRequest();
    device.to_proto(req.add_devices());
    auto [res, err] = device_create_client->send("/device/create", req);
    if (err) return err;
    if (res.devices_size() == 0) return unexpected_missing_error("device");
    device.key = res.devices().at(0).key();
    return err;
}

xerrors::Error DeviceClient::create(const std::vector<Device> &devs) const {
    auto req = api::v1::DeviceCreateRequest();
    req.mutable_devices()->Reserve(static_cast<int>(devs.size()));
    for (auto &device: devs)
        device.to_proto(req.add_devices());
    auto [res, err] = device_create_client->send("/device/create", req);
    return err;
}

xerrors::Error DeviceClient::del(const std::string &key) const {
    auto req = api::v1::DeviceDeleteRequest();
    req.add_keys(key);
    auto [res, err] = device_delete_client->send("/device/delete", req);
    return err;
}

xerrors::Error DeviceClient::del(const std::vector<std::string> &keys) const {
    auto req = api::v1::DeviceDeleteRequest();
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, err] = device_delete_client->send("/device/delete", req);
    return err;
}

std::pair<Device, xerrors::Error> Device::from_proto(const api::v1::Device &device) {
    Device d;
    d.key = device.key();
    d.name = device.name();
    d.rack = device.rack();
    d.location = device.location();
    d.make = device.make();
    d.model = device.model();
    d.properties = device.properties();
    d.configured = device.configured();
    if (device.has_status()) {
        auto [s, err] = DeviceStatus::from_proto(device.status());
        if (err) return {d, err};
        d.status = s;
    }
    return {d, xerrors::NIL};
}

Device::Device(
    std::string key,
    std::string name,
    RackKey rack,
    std::string location,
    std::string make,
    std::string model,
    std::string properties
):
    key(std::move(key)),
    name(std::move(name)),
    rack(rack),
    location(std::move(location)),
    make(std::move(make)),
    model(std::move(model)),
    properties(std::move(properties)) {}

void Device::to_proto(api::v1::Device *device) const {
    device->set_key(key);
    device->set_name(name);
    device->set_rack(rack);
    device->set_location(location);
    device->set_make(make);
    device->set_model(model);
    device->set_properties(properties);
    device->set_configured(configured);
    if (!status.is_zero()) status.to_proto(device->mutable_status());
}
}
