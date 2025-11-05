// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/errors/errors.h"
#include "client/cpp/hardware/hardware.h"
#include "x/cpp/xerrors/errors.h"

namespace synnax {
Rack::Rack(const RackKey key, std::string name): key(key), name(std::move(name)) {}

Rack::Rack(std::string name): name(std::move(name)) {}

Rack::Rack(const api::v1::Rack &rack): key(rack.key()), name(rack.name()) {}

void Rack::to_proto(api::v1::Rack *rack) const {
    rack->set_key(key);
    rack->set_name(name);
}

const std::string RETRIEVE_RACK_ENDPOINT = "/hardware/rack/retrieve";
const std::string CREATE_RACK_ENDPOINT = "/hardware/rack/create";

std::pair<Rack, xerrors::Error> HardwareClient::retrieve_rack(const RackKey key) const {
    auto req = api::v1::HardwareRetrieveRackRequest();
    req.add_keys(key);
    auto [res, err] = rack_retrieve_client->send(RETRIEVE_RACK_ENDPOINT, req);
    if (err) return {Rack(), err};
    if (res.racks_size() == 0)
        return {
            Rack(),
            xerrors::Error(
                xerrors::NOT_FOUND,
                "Rack matching" + std::to_string(key) + " not found"
            )
        };
    auto rack = Rack(res.racks(0));
    rack.tasks = TaskClient(
        rack.key,
        task_create_client,
        task_retrieve_client,
        task_delete_client
    );
    return {rack, err};
}

std::pair<Rack, xerrors::Error>
HardwareClient::retrieve_rack(const std::string &name) const {
    auto req = api::v1::HardwareRetrieveRackRequest();
    req.add_names(name);
    auto [res, err] = rack_retrieve_client->send(RETRIEVE_RACK_ENDPOINT, req);
    if (err) return {Rack(), err};
    if (res.racks_size() == 0)
        return {
            Rack(),
            xerrors::Error(xerrors::NOT_FOUND, "Rack matching" + name + " not found")
        };
    if (res.racks_size() > 1)
        return {
            Rack(),
            xerrors::Error(
                xerrors::MULTIPLE_RESULTS,
                "Multiple racks matching" + name + " found"
            )
        };
    auto rack = Rack(res.racks(0));
    rack.tasks = TaskClient(
        rack.key,
        task_create_client,
        task_retrieve_client,
        task_delete_client
    );
    return {rack, err};
}

xerrors::Error HardwareClient::create_rack(Rack &rack) const {
    auto req = api::v1::HardwareCreateRackRequest();
    rack.to_proto(req.add_racks());
    auto [res, err] = rack_create_client->send(CREATE_RACK_ENDPOINT, req);
    if (err) return err;
    if (res.racks_size() == 0) return unexpected_missing("rack");
    rack.key = res.racks().at(0).key();
    rack.tasks = TaskClient(
        rack.key,
        task_create_client,
        task_retrieve_client,
        task_delete_client
    );
    return err;
}

std::pair<Rack, xerrors::Error>
HardwareClient::create_rack(const std::string &name) const {
    auto rack = Rack(name);
    auto err = create_rack(rack);
    return {rack, err};
}

xerrors::Error HardwareClient::delete_rack(const RackKey key) const {
    auto req = api::v1::HardwareDeleteRackRequest();
    req.add_keys(key);
    auto [res, err] = rack_delete_client->send(CREATE_RACK_ENDPOINT, req);
    return err;
}

Task::Task(
    TaskKey key,
    std::string name,
    std::string type,
    std::string config,
    bool internal,
    bool snapshot
):
    key(key),
    name(std::move(name)),
    type(std::move(type)),
    config(std::move(config)),
    internal(internal),
    snapshot(snapshot) {}

Task::Task(
    std::string name,
    std::string type,
    std::string config,
    bool internal,
    bool snapshot
):
    key(create_task_key(0, 0)),
    name(std::move(name)),
    type(std::move(type)),
    config(std::move(config)),
    internal(internal),
    snapshot(snapshot) {}

Task::Task(
    RackKey rack,
    std::string name,
    std::string type,
    std::string config,
    bool internal,
    bool snapshot
):
    key(create_task_key(rack, 0)),
    name(std::move(name)),
    type(std::move(type)),
    config(std::move(config)),
    internal(internal),
    snapshot(snapshot) {}

Task::Task(const api::v1::Task &task):
    key(task.key()),
    name(task.name()),
    type(task.type()),
    config(task.config()),
    internal(task.internal()),
    snapshot(task.snapshot()) {}

void Task::to_proto(api::v1::Task *task) const {
    task->set_key(key);
    task->set_name(name);
    task->set_type(type);
    task->set_config(config);
    task->set_internal(internal);
    task->set_snapshot(snapshot);
}

const std::string RETRIEVE_TASK_ENDPOINT = "/hardware/task/retrieve";
const std::string CREATE_TASK_ENDPOINT = "/hardware/task/create";
const std::string DELETE_TASK_ENDPOINT = "/hardware/task/delete";

std::pair<Task, xerrors::Error> TaskClient::retrieve(const TaskKey key) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.add_keys(key);
    auto [res, err] = task_retrieve_client->send(RETRIEVE_TASK_ENDPOINT, req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0)
        return {
            Task(),
            xerrors::Error(
                xerrors::NOT_FOUND,
                "Task matching" + std::to_string(key) + " not found"
            )
        };
    return {Task(res.tasks(0)), err};
}

std::pair<Task, xerrors::Error> TaskClient::retrieve(const std::string &name) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.add_names(name);
    auto [res, err] = task_retrieve_client->send(RETRIEVE_TASK_ENDPOINT, req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0)
        return {
            Task(),
            xerrors::Error(xerrors::NOT_FOUND, "Task matching" + name + " not found")
        };
    return {Task(res.tasks(0)), err};
}

std::pair<std::vector<Task>, xerrors::Error>
TaskClient::retrieve(const std::vector<std::string> &names) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.mutable_names()->Add(names.begin(), names.end());
    auto [res, err] = task_retrieve_client->send(RETRIEVE_TASK_ENDPOINT, req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks = {res.tasks().begin(), res.tasks().end()};
    return {tasks, err};
}

std::pair<Task, xerrors::Error>
TaskClient::retrieve_by_type(const std::string &type) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.add_types(type);
    auto [res, err] = task_retrieve_client->send(RETRIEVE_TASK_ENDPOINT, req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0)
        return {
            Task(),
            xerrors::Error(xerrors::NOT_FOUND, "Task matching" + type + " not found")
        };
    return {Task(res.tasks(0)), err};
}

std::pair<std::vector<Task>, xerrors::Error>
TaskClient::retrieve_by_type(const std::vector<std::string> &types) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.mutable_types()->Add(types.begin(), types.end());
    auto [res, err] = task_retrieve_client->send(RETRIEVE_TASK_ENDPOINT, req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks = {res.tasks().begin(), res.tasks().end()};
    return {tasks, err};
}

xerrors::Error TaskClient::create(Task &task) const {
    auto req = api::v1::HardwareCreateTaskRequest();
    task.to_proto(req.add_tasks());
    auto [res, err] = task_create_client->send(CREATE_TASK_ENDPOINT, req);
    if (err) return err;
    if (res.tasks_size() == 0) return unexpected_missing("task");
    task.key = res.tasks().at(0).key();
    return err;
}

xerrors::Error TaskClient::del(const TaskKey key) const {
    auto req = api::v1::HardwareDeleteTaskRequest();
    req.add_keys(key);
    auto [res, err] = task_delete_client->send(DELETE_TASK_ENDPOINT, req);
    return err;
}

std::pair<std::vector<Task>, xerrors::Error> TaskClient::list() const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    auto [res, err] = task_retrieve_client->send(RETRIEVE_TASK_ENDPOINT, req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks = {res.tasks().begin(), res.tasks().end()};
    return {tasks, err};
}

const std::string RETRIEVE_DEVICE_ENDPOINT = "/hardware/device/retrieve";
const std::string CREATE_DEVICE_ENDPOINT = "/hardware/device/create";
const std::string DELETE_DEVICE_ENDPOINT = "/hardware/device/delete";

std::pair<Device, xerrors::Error>
HardwareClient::retrieve_device(const std::string &key) const {
    auto req = api::v1::HardwareRetrieveDeviceRequest();
    req.add_keys(key);
    auto [res, err] = device_retrieve_client->send(RETRIEVE_DEVICE_ENDPOINT, req);
    if (err) return {Device(), err};
    if (res.devices_size() == 0) {
        return {
            Device(),
            xerrors::Error(xerrors::NOT_FOUND, "Device matching" + key + " not found")
        };
    }
    return {Device(res.devices(0)), err};
}

std::pair<std::vector<Device>, xerrors::Error> HardwareClient::retrieve_devices(
    const std::vector<std::string> &keys,
    bool ignore_not_found
) const {
    auto req = api::v1::HardwareRetrieveDeviceRequest();
    req.mutable_keys()->Add(keys.begin(), keys.end());
    req.set_ignore_not_found(ignore_not_found);
    auto [res, err] = device_retrieve_client->send(RETRIEVE_DEVICE_ENDPOINT, req);
    std::vector<Device> devices = {res.devices().begin(), res.devices().end()};
    return {devices, err};
}

std::pair<std::vector<Device>, xerrors::Error>
HardwareClient::retrieve_devices(HardwareDeviceRetrieveRequest &req) const {
    auto api_req = api::v1::HardwareRetrieveDeviceRequest();
    req.to_proto(api_req);
    auto [res, err] = device_retrieve_client->send(RETRIEVE_DEVICE_ENDPOINT, api_req);
    if (err) return {std::vector<Device>(), err};
    std::vector<Device> devices = {res.devices().begin(), res.devices().end()};
    return {devices, err};
}

xerrors::Error HardwareClient::create_device(Device &device) const {
    auto req = api::v1::HardwareCreateDeviceRequest();
    device.to_proto(req.add_devices());
    auto [res, err] = device_create_client->send(CREATE_DEVICE_ENDPOINT, req);
    if (err) return err;
    if (res.devices_size() == 0) return unexpected_missing("device");
    device.key = res.devices().at(0).key();
    return err;
}

xerrors::Error HardwareClient::create_devices(const std::vector<Device> &devs) const {
    auto req = api::v1::HardwareCreateDeviceRequest();
    req.mutable_devices()->Reserve(static_cast<int>(devs.size()));
    for (auto &device: devs)
        device.to_proto(req.add_devices());
    auto [res, err] = device_create_client->send(CREATE_DEVICE_ENDPOINT, req);
    return err;
}

xerrors::Error HardwareClient::delete_device(const std::string &key) const {
    auto req = api::v1::HardwareDeleteDeviceRequest();
    req.add_keys(key);
    auto [res, err] = device_delete_client->send(DELETE_DEVICE_ENDPOINT, req);
    return err;
}

xerrors::Error
HardwareClient::delete_devices(const std::vector<std::string> &keys) const {
    auto req = api::v1::HardwareDeleteDeviceRequest();
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, err] = device_delete_client->send(DELETE_DEVICE_ENDPOINT, req);
    return err;
}

Device::Device(const api::v1::Device &device):
    key(device.key()),
    name(device.name()),
    rack(device.rack()),
    location(device.location()),
    make(device.make()),
    model(device.model()),
    properties(device.properties()),
    configured(device.configured()) {}

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
}
}
