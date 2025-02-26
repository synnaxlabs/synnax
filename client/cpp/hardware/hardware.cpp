// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/hardware/hardware.h"

#include <utility>

#include "client/cpp/errors/errors.h"

using namespace synnax;

Rack::Rack(RackKey key, std::string name) : key(key),
                                            name(std::move(name)) {
}

Rack::Rack(std::string name) : name(std::move(name)) {
}

Rack::Rack(const api::v1::Rack &rack) : key(rack.key()),
                                        name(rack.name()) {
}

void Rack::to_proto(api::v1::Rack *rack) const {
    rack->set_key(key);
    rack->set_name(name);
}


const std::string RETRIEVE_RACK_ENDPOINT = "/hardware/rack/retrieve";
const std::string CREATE_RACK_ENDPOINT = "/hardware/rack/create";

std::pair<Rack, freighter::Error> HardwareClient::retrieveRack(
    const std::uint32_t key
) const {
    auto req = api::v1::HardwareRetrieveRackRequest();
    req.add_keys(key);
    auto [res, err] = rack_retrieve_client->send(RETRIEVE_RACK_ENDPOINT, req);
    if (err) return {Rack(), err};
    if (res.racks_size() == 0)
        return {
            Rack(),
            freighter::Error(synnax::NOT_FOUND,
                             "Rack matching" + std::to_string(key) + " not found")
        };
    auto rack = Rack(res.racks(0));
    rack.tasks = TaskClient(rack.key, task_create_client, task_retrieve_client,
                            task_delete_client);
    return {rack, err};
}

std::pair<Rack, freighter::Error> HardwareClient::retrieveRack(
    const std::string &name
) const {
    auto req = api::v1::HardwareRetrieveRackRequest();
    req.add_names(name);
    auto [res, err] = rack_retrieve_client->send(RETRIEVE_RACK_ENDPOINT, req);
    if (err) return {Rack(), err};
    if (res.racks_size() == 0)
        return {
            Rack(),
            freighter::Error(synnax::NOT_FOUND, "Rack matching" + name + " not found")
        };
    if (res.racks_size() > 1)
        return {
            Rack(),
            freighter::Error(synnax::MULTIPLE_RESULTS,
                             "Multiple racks matching" + name + " found")
        };
    auto rack = Rack(res.racks(0));
    rack.tasks = TaskClient(rack.key, task_create_client, task_retrieve_client,
                            task_delete_client);
    return {rack, err};
}


freighter::Error HardwareClient::createRack(Rack &rack) const {
    auto req = api::v1::HardwareCreateRackRequest();
    rack.to_proto(req.add_racks());
    auto [res, err] = rack_create_client->send(CREATE_RACK_ENDPOINT, req);
    if (err) return err;
    if (res.racks_size() == 0)
        return freighter::Error(
            synnax::UNEXPECTED_ERROR,
            "No racks returned from server on create. Please report this error to the Synnax team.");
    rack.key = res.racks().at(0).key();
    rack.tasks = TaskClient(rack.key, task_create_client, task_retrieve_client,
                            task_delete_client);
    return err;
}

std::pair<Rack, freighter::Error> HardwareClient::createRack(
    const std::string &name) const {
    auto rack = Rack(name);
    auto err = createRack(rack);
    return {rack, err};
}

freighter::Error HardwareClient::deleteRack(std::uint32_t key) const {
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
    bool internal
) : key(key),
    name(std::move(name)),
    type(std::move(type)),
    config(std::move(config)),
    internal(internal) {
}

Task::Task(
    std::string name,
    std::string type,
    std::string config,
    bool internal
) : key(createTaskKey(0, 0)),
    name(std::move(name)),
    type(std::move(type)),
    config(std::move(config)),
    internal(internal) {
}

Task::Task(
    RackKey rack,
    std::string name,
    std::string type,
    std::string config,
    bool internal
) : key(createTaskKey(rack, 0)),
    name(std::move(name)),
    type(std::move(type)),
    config(std::move(config)),
    internal(internal) {
}

Task::Task(const api::v1::Task &task) : key(task.key()),
                                        name(task.name()),
                                        type(task.type()),
                                        config(task.config()),
                                        internal(task.internal()) {
}

void Task::to_proto(api::v1::Task *task) const {
    task->set_key(key);
    task->set_name(name);
    task->set_type(type);
    task->set_config(config);
    task->set_internal(internal);
}

const std::string RETRIEVE_MODULE_ENDPOINT = "/hardware/task/retrieve";
const std::string CREATE_MODULE_ENDPOINT = "/hardware/task/create";
const std::string DELETE_MODULE_ENDPOINT = "/hardware/task/delete";

std::pair<Task, freighter::Error> TaskClient::retrieve(std::uint64_t key) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.add_keys(key);
    auto [res, err] = task_retrieve_client->send(RETRIEVE_MODULE_ENDPOINT, req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0)
        return {
            Task(),
            freighter::Error(synnax::NOT_FOUND,
                             "Task matching" + std::to_string(key) + " not found")
        };
    return {Task(res.tasks(0)), err};
}

std::pair<Task, freighter::Error> TaskClient::retrieve(const std::string &name) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.add_names(name);
    auto [res, err] = task_retrieve_client->send(RETRIEVE_MODULE_ENDPOINT, req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0)
        return {
            Task(),
            freighter::Error(synnax::NOT_FOUND, "Task matching" + name + " not found")
        };
    return {Task(res.tasks(0)), err};
}

std::pair<std::vector<Task>, freighter::Error> TaskClient::retrieve(
    const std::vector<std::string> &names
) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.mutable_names()->Add(names.begin(), names.end());
    auto [res, err] = task_retrieve_client->send(RETRIEVE_MODULE_ENDPOINT, req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks = {res.tasks().begin(), res.tasks().end()};
    return {tasks, err};
}


std::pair<Task, freighter::Error> TaskClient::retrieveByType(
    const std::string &type
) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.add_types(type);
    auto [res, err] = task_retrieve_client->send(RETRIEVE_MODULE_ENDPOINT, req);
    if (err) return {Task(), err};
    if (res.tasks_size() == 0)
        return {
            Task(),
            freighter::Error(synnax::NOT_FOUND, "Task matching" + type + " not found")
        };
    return {Task(res.tasks(0)), err};
}

std::pair<std::vector<Task>, freighter::Error> TaskClient::retrieveByType(
    const std::vector<std::string> &types
) const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    req.mutable_types()->Add(types.begin(), types.end());
    auto [res, err] = task_retrieve_client->send(RETRIEVE_MODULE_ENDPOINT, req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks = {res.tasks().begin(), res.tasks().end()};
    return {tasks, err};
}


freighter::Error TaskClient::create(Task &task) const {
    auto req = api::v1::HardwareCreateTaskRequest();
    task.to_proto(req.add_tasks());
    auto [res, err] = task_create_client->send(CREATE_MODULE_ENDPOINT, req);
    if (err) return err;
    if (res.tasks_size() == 0)
        return freighter::Error(
            synnax::UNEXPECTED_ERROR,
            "No tasks returned from server on create. Please report this error to the Synnax team."
        );
    task.key = res.tasks().at(0).key();
    return err;
}

freighter::Error TaskClient::del(std::uint64_t key) const {
    auto req = api::v1::HardwareDeleteTaskRequest();
    req.add_keys(key);
    auto [res, err] = task_delete_client->send(DELETE_MODULE_ENDPOINT, req);
    return err;
}

std::pair<std::vector<Task>, freighter::Error> TaskClient::list() const {
    auto req = api::v1::HardwareRetrieveTaskRequest();
    req.set_rack(rack);
    auto [res, err] = task_retrieve_client->send(RETRIEVE_MODULE_ENDPOINT, req);
    if (err) return {std::vector<Task>(), err};
    std::vector<Task> tasks = {res.tasks().begin(), res.tasks().end()};
    return {tasks, err};
}

std::pair<Device, freighter::Error> HardwareClient::retrieveDevice(
    const std::string &key) const {
    auto req = api::v1::HardwareRetrieveDeviceRequest();
    req.add_keys(key);
    auto [res, err] = device_retrieve_client->send(RETRIEVE_RACK_ENDPOINT, req);
    if (err) return {Device(), err};
    if (res.devices_size() == 0)
        return {
            Device(),
            freighter::Error(synnax::NOT_FOUND,
                             "Device matching" + key + " not found")
        };
    return {Device(res.devices(0)), err};
}

freighter::Error HardwareClient::createDevice(Device &device) const {
    auto req = api::v1::HardwareCreateDeviceRequest();
    device.to_proto(req.add_devices());
    auto [res, err] = device_create_client->send(CREATE_RACK_ENDPOINT, req);
    if (err) return err;
    if (res.devices_size() == 0)
        return freighter::Error(
            synnax::UNEXPECTED_ERROR,
            "No devices returned from server on create. Please report this error to the Synnax team.");
    device.key = res.devices().at(0).key();
    return err;
}

Device::Device(const api::v1::Device &device) : key(device.key()),
                                                name(device.name()),
                                                rack(device.rack()),
                                                location(device.location()),
                                                identifier(device.identifier()),
                                                make(device.make()),
                                                model(device.model()),
                                                properties(device.properties()) {
}

void Device::to_proto(api::v1::Device *device) const {
    device->set_key(key);
    device->set_name(name);
    device->set_rack(rack);
    device->set_location(location);
    device->set_identifier(identifier);
    device->set_make(make);
    device->set_model(model);
    device->set_properties(properties);
}
